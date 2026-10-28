package httpapi

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/sahilm/fuzzy"

	"github.com/alialjaffer/fai/backend/internal/store"
)

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
	issues, err := s.Jira.SearchIssues(jql, 100)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	columns, err := s.Store.ListColumnsByBoard(ctx, body.BoardID)
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
		existing, err := s.Store.GetJiraLinkByKey(ctx, issue.Key)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if existing != nil && !body.ForceUpdate {
			skipped++
			continue
		}

		columnID := resolveColumn(issue.Status)

		if existing != nil {
			if err := s.Store.UpdateTaskTitle(ctx, existing.TaskID, issue.Summary); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if err := s.Store.UpdateJiraLinkImported(ctx, existing.TaskID, issue.Status); err != nil {
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
		task, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
			BoardID: body.BoardID, ColumnID: columnID, Title: issue.Summary,
			Priority: "medium", Position: position,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := s.Store.CreateJiraLinkImported(ctx, task.ID, issue.Key, s.Jira.IssueURL(issue.Key), s.JiraDefaultProject, issue.Status); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		imported++
	}

	writeJSON(w, http.StatusOK, map[string]int{"imported": imported, "updated": updated, "skipped": skipped})
}

type jiraExportBody struct {
	TaskID     int64  `json:"task_id"`
	ProjectKey string `json:"project_key"`
}

func (s *Server) jiraExport(w http.ResponseWriter, r *http.Request) {
	if s.Jira == nil {
		writeError(w, http.StatusServiceUnavailable, "jira is not configured")
		return
	}
	var body jiraExportBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	projectKey := body.ProjectKey
	if projectKey == "" {
		projectKey = s.JiraDefaultProject
	}
	if projectKey == "" {
		writeError(w, http.StatusBadRequest, "project_key is required (no JIRA_DEFAULT_PROJECT_KEY configured)")
		return
	}

	ctx := r.Context()
	task, err := s.Store.GetTask(ctx, body.TaskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if existing, err := s.Store.GetJiraLinkByTask(ctx, task.ID); err == nil && existing != nil {
		writeError(w, http.StatusConflict, "task is already linked to a jira issue")
		return
	}

	key, err := s.Jira.CreateIssue(projectKey, task.Title)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	link, err := s.Store.CreateJiraLinkExported(ctx, task.ID, key, s.Jira.IssueURL(key), projectKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

type jiraLinkBody struct {
	JiraKey string `json:"jira_key"`
}

func (s *Server) jiraLink(w http.ResponseWriter, r *http.Request) {
	if s.Jira == nil {
		writeError(w, http.StatusServiceUnavailable, "jira is not configured")
		return
	}
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var body jiraLinkBody
	if err := decodeJSON(r, &body); err != nil || body.JiraKey == "" {
		writeError(w, http.StatusBadRequest, "jira_key is required")
		return
	}

	issue, err := s.Jira.GetIssue(body.JiraKey)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if issue == nil {
		writeError(w, http.StatusNotFound, "jira issue not found")
		return
	}

	projectKey := s.JiraDefaultProject
	if parts := strings.SplitN(issue.Key, "-", 2); len(parts) == 2 {
		projectKey = parts[0]
	}

	ctx := r.Context()
	if existing, err := s.Store.GetJiraLinkByKey(ctx, issue.Key); err == nil && existing != nil {
		writeError(w, http.StatusConflict, "jira issue is already linked to another task")
		return
	}
	if existing, err := s.Store.GetJiraLinkByTask(ctx, taskID); err == nil && existing != nil {
		writeError(w, http.StatusConflict, "task is already linked to a jira issue")
		return
	}

	link, err := s.Store.CreateJiraLinkManual(ctx, taskID, issue.Key, s.Jira.IssueURL(issue.Key), projectKey, issue.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

func (s *Server) jiraGetLink(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	link, err := s.Store.GetJiraLinkByTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, link)
}

func (s *Server) jiraUnlink(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	if err := s.Store.DeleteJiraLink(r.Context(), taskID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type jiraSuggestion struct {
	JiraKey string `json:"jira_key"`
	Summary string `json:"summary"`
	Status  string `json:"status"`
	Score   int    `json:"score"`
}

func (s *Server) jiraSuggestions(w http.ResponseWriter, r *http.Request) {
	if s.Jira == nil {
		writeError(w, http.StatusServiceUnavailable, "jira is not configured")
		return
	}
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	limit := 5
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		ctx := r.Context()
		task, err := s.Store.GetTask(ctx, taskID)
		if err != nil {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		query = task.Title
	} else if len([]rune(query)) < 2 {
		writeJSON(w, http.StatusOK, []jiraSuggestion{})
		return
	}

	issues, err := s.Jira.SearchByText(query, s.JiraDefaultProject, 25)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	summaries := make([]string, len(issues))
	for i, issue := range issues {
		summaries[i] = issue.Key + " " + issue.Summary
	}
	matches := fuzzy.Find(query, summaries)
	sort.Sort(matches)

	if limit > len(matches) {
		limit = len(matches)
	}
	suggestions := make([]jiraSuggestion, 0, limit)
	for _, m := range matches[:limit] {
		issue := issues[m.Index]
		suggestions = append(suggestions, jiraSuggestion{
			JiraKey: issue.Key,
			Summary: issue.Summary,
			Status:  issue.Status,
			Score:   m.Score,
		})
	}
	writeJSON(w, http.StatusOK, suggestions)
}
