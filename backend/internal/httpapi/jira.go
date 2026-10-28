package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sahilm/fuzzy"

	"github.com/alialjaffer/fai/backend/internal/jira"
	"github.com/alialjaffer/fai/backend/internal/store"
)

type jiraImportBody struct {
	BoardID     int64  `json:"board_id"`
	JQL         string `json:"jql"`
	ForceUpdate bool   `json:"force_update"`
}

// onlyAssigned keeps every import scoped to the configured Jira account, so a
// project-wide or custom JQL cannot pull in someone else's issues. An ORDER BY
// has to stay outside the grouped expression to remain valid JQL.
func onlyAssigned(jql string) string {
	const filter = "assignee = currentUser()"
	if i := strings.Index(strings.ToLower(jql), "order by"); i >= 0 {
		return "(" + strings.TrimSpace(jql[:i]) + ") AND " + filter + " " + jql[i:]
	}
	return "(" + jql + ") AND " + filter
}

// completionTime returns Jira's resolution time, or the last update when the
// workflow leaves the resolution empty, so stats reflect history rather than
// the moment the import ran.
func completionTime(issue jira.Issue) *time.Time {
	raw := issue.ResolutionDate
	if raw == "" {
		raw = issue.Updated
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.000-0700"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return &t
		}
	}
	return nil
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
		jql = fmt.Sprintf(`project = "%s"`, s.JiraDefaultProject)
	}
	jql = onlyAssigned(jql)
	if !strings.Contains(strings.ToLower(jql), "order by") {
		jql += " ORDER BY updated DESC"
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
	columnIsDone := map[int64]bool{}
	for _, c := range columns {
		columnIDByName[c.Name] = c.ID
		columnIsDone[c.ID] = c.IsDone
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

	inRun := make(map[string]bool, len(issues))
	for _, issue := range issues {
		inRun[issue.Key] = true
	}
	created := map[string]int64{}

	// The list view nests one level, so a parent that is itself a subtask gets
	// flattened to its topmost ancestor.
	rootTask := func(taskID int64) (int64, bool) {
		seen := map[int64]bool{}
		for !seen[taskID] {
			seen[taskID] = true
			task, err := s.Store.GetTask(ctx, taskID)
			if err != nil {
				return 0, false
			}
			if task.ParentTaskID == nil {
				return taskID, true
			}
			taskID = *task.ParentTaskID
		}
		return 0, false
	}

	resolveParent := func(key string) (int64, bool, error) {
		if id, ok := created[key]; ok {
			return id, true, nil
		}
		link, err := s.Store.GetJiraLinkByKey(ctx, key)
		if err != nil || link == nil {
			return 0, false, err
		}
		return link.TaskID, true, nil
	}

	importComments := func(taskID int64, issue jira.Issue, comments []jira.Comment) error {
		if issue.AssigneeID == "" {
			return nil
		}
		existing, err := s.Store.ListTaskComments(ctx, taskID)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, comment := range existing {
			if comment.ExternalID != nil {
				seen[*comment.ExternalID] = true
			}
		}

		byID := map[string]jira.Comment{}
		for _, comment := range comments {
			byID[comment.ID] = comment
		}
		for _, comment := range comments {
			// Only the assignee's own comments count, and a reply is skipped when
			// it answers someone else.
			if comment.AuthorID != issue.AssigneeID || seen[comment.ID] {
				continue
			}
			if comment.ParentID != "" {
				parent, ok := byID[comment.ParentID]
				if !ok || parent.AuthorID != issue.AssigneeID {
					continue
				}
			}
			externalID := comment.ID
			if _, err := s.Store.CreateTaskComment(ctx, taskID, comment.AuthorName, comment.Body, &externalID); err != nil {
				return err
			}
		}
		return nil
	}

	handle := func(issue jira.Issue, parentTaskID *int64) error {
		existing, err := s.Store.GetJiraLinkByKey(ctx, issue.Key)
		if err != nil {
			return err
		}

		if existing != nil && !body.ForceUpdate {
			skipped++
			return nil
		}

		comments, err := s.Jira.GetComments(issue.Key)
		if err != nil {
			return err
		}

		columnID := resolveColumn(issue.Status)
		resolvedAt := completionTime(issue)

		if existing != nil {
			task, err := s.Store.GetTask(ctx, existing.TaskID)
			if err != nil {
				return err
			}
			if task.ColumnID != columnID {
				position, err := s.appendPosition(ctx, columnID)
				if err != nil {
					return err
				}
				if _, err := s.Store.MoveTask(ctx, existing.TaskID, columnID, position); err != nil {
					return err
				}
			}
			if resolvedAt != nil && columnIsDone[columnID] {
				if err := s.Store.SetTaskCompletedAt(ctx, existing.TaskID, *resolvedAt); err != nil {
					return err
				}
			}
			description := task.DescriptionMd
			if issue.Description != "" {
				description = issue.Description
			}
			if _, err := s.Store.UpdateTask(ctx, existing.TaskID, issue.Summary, description, task.Priority, task.DueDate); err != nil {
				return err
			}
			if err := s.Store.UpdateJiraLinkImported(ctx, existing.TaskID, issue.Status); err != nil {
				return err
			}
			if err := s.Store.SetTaskParent(ctx, existing.TaskID, parentTaskID); err != nil {
				return err
			}
			if err := importComments(existing.TaskID, issue, comments); err != nil {
				return err
			}
			updated++
			return nil
		}

		position, err := s.appendPosition(ctx, columnID)
		if err != nil {
			return err
		}
		task, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
			BoardID: body.BoardID, ColumnID: columnID, Title: issue.Summary,
			DescriptionMd: issue.Description,
			Priority:      "medium", Position: position, ParentTaskID: parentTaskID,
		})
		if err != nil {
			return err
		}
		if resolvedAt != nil && columnIsDone[columnID] {
			if err := s.Store.SetTaskCompletedAt(ctx, task.ID, *resolvedAt); err != nil {
				return err
			}
		}
		if _, err := s.Store.CreateJiraLinkImported(ctx, task.ID, issue.Key, s.Jira.IssueURL(issue.Key), s.JiraDefaultProject, issue.Status); err != nil {
			return err
		}
		if err := importComments(task.ID, issue, comments); err != nil {
			return err
		}
		created[issue.Key] = task.ID
		imported++
		return nil
	}

	pending := issues
	for len(pending) > 0 {
		var deferred []jira.Issue
		progressed := false
		for _, issue := range pending {
			var parentTaskID *int64
			if issue.ParentKey != "" {
				parentID, ok, err := resolveParent(issue.ParentKey)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				if !ok {
					if inRun[issue.ParentKey] {
						deferred = append(deferred, issue)
					} else {
						skipped++
					}
					continue
				}
				rootID, ok := rootTask(parentID)
				if !ok {
					skipped++
					continue
				}
				parentTaskID = &rootID
			}

			if err := handle(issue, parentTaskID); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			progressed = true
		}
		if !progressed {
			// Leftovers form a cycle, or their parent was skipped as an orphan.
			skipped += len(deferred)
			break
		}
		pending = deferred
	}

	writeJSON(w, http.StatusOK, map[string]int{"imported": imported, "updated": updated, "skipped": skipped})
}

type jiraExportBody struct {
	TaskID          int64   `json:"task_id"`
	TaskIDs         []int64 `json:"task_ids"`
	ProjectKey      string  `json:"project_key"`
	IncludeSubtasks *bool   `json:"include_subtasks"`
}

type jiraExportResult struct {
	TaskID    int64  `json:"task_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	JiraKey   string `json:"jira_key,omitempty"`
	IssueType string `json:"issue_type,omitempty"`
	Error     string `json:"error,omitempty"`
}

type jiraExportResponse struct {
	Created int                `json:"created"`
	Skipped int                `json:"skipped"`
	Failed  int                `json:"failed"`
	Results []jiraExportResult `json:"results"`
}

type jiraExporter struct {
	ctx         context.Context
	s           *Server
	projectKey  string
	epicType    string
	subtaskType string
	includeSubs bool
	resp        *jiraExportResponse
	keys        map[int64]string
}

func (e *jiraExporter) record(res jiraExportResult) {
	e.resp.Results = append(e.resp.Results, res)
	switch res.Status {
	case "created":
		e.resp.Created++
	case "skipped":
		e.resp.Skipped++
	default:
		e.resp.Failed++
	}
	if res.JiraKey != "" {
		e.keys[res.TaskID] = res.JiraKey
	}
}

// issueType picks the Jira type from the subtask count; a task that is itself a
// child always exports as the instance's subtask type.
func (e *jiraExporter) issueType(task store.Task, subtaskCount int) string {
	if task.ParentTaskID != nil {
		return e.subtaskType
	}
	if subtaskCount > 3 {
		return e.epicType
	}
	return "Task"
}

// create links the task to a new Jira issue, or skips it when already linked.
func (e *jiraExporter) create(task store.Task, issueType, parentKey string) (string, error) {
	existing, err := e.s.Store.GetJiraLinkByTask(e.ctx, task.ID)
	if err != nil {
		return "", err
	}
	if existing != nil {
		e.record(jiraExportResult{TaskID: task.ID, Title: task.Title, Status: "skipped", JiraKey: existing.JiraKey})
		return existing.JiraKey, nil
	}

	key, err := e.s.Jira.CreateIssue(e.projectKey, task.Title, issueType, parentKey)
	if err != nil {
		e.record(jiraExportResult{TaskID: task.ID, Title: task.Title, Status: "failed", IssueType: issueType, Error: err.Error()})
		return "", err
	}
	if _, err := e.s.Store.CreateJiraLinkExported(e.ctx, task.ID, key, e.s.Jira.IssueURL(key), e.projectKey); err != nil {
		e.record(jiraExportResult{TaskID: task.ID, Title: task.Title, Status: "failed", JiraKey: key, IssueType: issueType, Error: err.Error()})
		return "", err
	}
	e.record(jiraExportResult{TaskID: task.ID, Title: task.Title, Status: "created", JiraKey: key, IssueType: issueType})
	return key, nil
}

// ensureParent exports a missing parent before its child and returns its Jira key.
func (e *jiraExporter) ensureParent(task store.Task) (string, bool) {
	if task.ParentTaskID == nil {
		return "", true
	}
	if key, ok := e.keys[*task.ParentTaskID]; ok {
		return key, true
	}
	parent, err := e.s.Store.GetTask(e.ctx, *task.ParentTaskID)
	if err != nil {
		return "", false
	}
	parentKey, ok := e.ensureParent(parent)
	if !ok {
		return "", false
	}
	subtasks, err := e.s.Store.ListSubtasks(e.ctx, parent.ID)
	if err != nil {
		return "", false
	}
	key, err := e.create(parent, e.issueType(parent, len(subtasks)), parentKey)
	if err != nil {
		return "", false
	}
	return key, true
}

func (e *jiraExporter) exportTree(taskID int64) {
	if _, done := e.keys[taskID]; done {
		return
	}
	task, err := e.s.Store.GetTask(e.ctx, taskID)
	if err != nil {
		e.record(jiraExportResult{TaskID: taskID, Status: "failed", Error: "task not found"})
		return
	}
	parentKey, ok := e.ensureParent(task)
	if !ok {
		e.record(jiraExportResult{TaskID: task.ID, Title: task.Title, Status: "failed", Error: "parent export failed"})
		return
	}
	subtasks, err := e.s.Store.ListSubtasks(e.ctx, task.ID)
	if err != nil {
		e.record(jiraExportResult{TaskID: task.ID, Title: task.Title, Status: "failed", Error: err.Error()})
		return
	}

	key, err := e.create(task, e.issueType(task, len(subtasks)), parentKey)
	if err != nil {
		if e.includeSubs {
			for _, child := range subtasks {
				if _, done := e.keys[child.ID]; done {
					continue
				}
				e.record(jiraExportResult{TaskID: child.ID, Title: child.Title, Status: "failed", Error: "parent export failed"})
			}
		}
		return
	}
	if !e.includeSubs {
		return
	}
	for _, child := range subtasks {
		if _, done := e.keys[child.ID]; done {
			continue
		}
		if _, err := e.create(child, e.subtaskType, key); err != nil {
			continue
		}
	}
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

	var taskIDs []int64
	seen := map[int64]bool{}
	for _, id := range append([]int64{body.TaskID}, body.TaskIDs...) {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		taskIDs = append(taskIDs, id)
	}
	if len(taskIDs) == 0 {
		writeError(w, http.StatusBadRequest, "task_id or task_ids is required")
		return
	}

	epicType := s.JiraEpicIssueType
	if epicType == "" {
		epicType = "Epic"
	}
	subtaskType := s.JiraSubtaskIssueType
	if subtaskType == "" {
		subtaskType = "Sub-task"
	}
	resp := &jiraExportResponse{Results: []jiraExportResult{}}
	exporter := &jiraExporter{
		ctx: r.Context(), s: s, projectKey: projectKey,
		epicType: epicType, subtaskType: subtaskType,
		includeSubs: body.IncludeSubtasks == nil || *body.IncludeSubtasks,
		resp:        resp, keys: map[int64]string{},
	}
	for _, id := range taskIDs {
		exporter.exportTree(id)
	}
	writeJSON(w, http.StatusOK, resp)
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
