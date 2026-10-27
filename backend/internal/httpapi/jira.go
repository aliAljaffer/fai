package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
)

func pgtypeText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func pgtypeTimestamptzNow() pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now(), Valid: true}
}

type jiraImportBody struct {
	BoardID     int64  `json:"board_id"`
	JQL         string `json:"jql"`
	ForceUpdate bool   `json:"force_update"`
}

func (s *Server) jiraImport(w http.ResponseWriter, r *http.Request) {
	if s.Jira == nil {
		writeError(w, http.StatusServiceUnavailable, "jira is not configured")
		return
	}

	var body jiraImportBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.BoardID == 0 {
		writeError(w, http.StatusBadRequest, "board_id is required")
		return
	}

	jql := body.JQL
	if jql == "" {
		if s.JiraDefaultProject == "" {
			writeError(w, http.StatusBadRequest, "jql is required (no JIRA_DEFAULT_PROJECT_KEY configured)")
			return
		}
		jql = fmt.Sprintf(`project = "%s" ORDER BY updated DESC`, s.JiraDefaultProject)
	}

	ctx := r.Context()
	issues, err := s.Jira.SearchIssues(jql, 50)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	columns, err := s.Q.ListColumnsByBoard(ctx, body.BoardID)
	if err != nil || len(columns) == 0 {
		writeError(w, http.StatusInternalServerError, "board has no columns")
		return
	}
	defaultColumnID := columns[0].ID
	for _, c := range columns {
		if c.Position < columns[0].Position {
			defaultColumnID = c.ID
		}
	}
	columnIDByName := map[string]int64{}
	for _, c := range columns {
		columnIDByName[c.Name] = c.ID
	}

	resolveColumn := func(status string) int64 {
		if name, ok := s.JiraStatusColumnMap[status]; ok {
			if id, ok := columnIDByName[name]; ok {
				return id
			}
		}
		return defaultColumnID
	}

	imported, updated, skipped := 0, 0, 0
	for _, issue := range issues {
		existing, err := s.Q.GetJiraLinkByKey(ctx, issue.Key)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		linked := err == nil

		if linked && !body.ForceUpdate {
			skipped++
			continue
		}

		columnID := resolveColumn(issue.Status)

		if linked {
			if err := s.Q.UpdateTaskTitle(ctx, sqlc.UpdateTaskTitleParams{
				ID:    existing.TaskID,
				Title: issue.Summary,
			}); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if err := s.Q.UpdateJiraLinkImported(ctx, sqlc.UpdateJiraLinkImportedParams{
				TaskID:    existing.TaskID,
				RawStatus: pgtypeText(issue.Status),
			}); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			updated++
			continue
		}

		position, err := s.appendPosition(ctx, columnID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		task, err := s.Q.CreateTask(ctx, sqlc.CreateTaskParams{
			BoardID:  body.BoardID,
			ColumnID: columnID,
			Title:    issue.Summary,
			Priority: "medium",
			Position: position,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := s.Q.CreateJiraLink(ctx, sqlc.CreateJiraLinkParams{
			TaskID:         task.ID,
			JiraKey:        issue.Key,
			JiraUrl:        s.Jira.IssueURL(issue.Key),
			JiraProjectKey: s.JiraDefaultProject,
			LinkOrigin:     "imported",
			LastImportedAt: pgtypeTimestamptzNow(),
			RawStatus:      pgtypeText(issue.Status),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		imported++
	}

	writeJSON(w, http.StatusOK, map[string]int{"imported": imported, "updated": updated, "skipped": skipped})
}
