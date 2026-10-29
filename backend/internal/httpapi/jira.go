package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sahilm/fuzzy"

	"github.com/alialjaffer/fai/backend/internal/jira"
	"github.com/alialjaffer/fai/backend/internal/store"
)

// jiraKeyMention matches Jira keys such as ABC-1 inside imported description text.
var jiraKeyMention = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-[0-9]{1,18}\b`)

// jiraKind maps a Jira issue type onto the Fai type. Epics become arcs,
// sub-tasks become steps, and every other issue type is a task.
func jiraKind(issueType string) string {
	switch strings.ToLower(strings.TrimSpace(issueType)) {
	case "epic":
		return "arc"
	case "sub-task", "subtask", "sub task":
		return "step"
	default:
		return "task"
	}
}

type jiraImportBody struct {
	BoardID     int64  `json:"board_id"`
	JQL         string `json:"jql"`
	ForceUpdate bool   `json:"force_update"`
}

// jiraImportState is the single import run this server tracks, so the UI can
// show progress while the owner moves around the app.
type jiraImportState struct {
	Running    bool   `json:"running"`
	Total      int    `json:"total"`
	Imported   int    `json:"imported"`
	Updated    int    `json:"updated"`
	Skipped    int    `json:"skipped"`
	Linked     int    `json:"linked"`
	Error      string `json:"error,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
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

// jiraTime parses a Jira timestamp, which uses ISO 8601 with a numeric offset.
func jiraTime(raw string) *time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.000-0700"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return &t
		}
	}
	return nil
}

// completionTime returns Jira's resolution time, or the last update when the
// workflow leaves the resolution empty, so stats reflect history rather than
// the moment the import ran.
func completionTime(issue jira.Issue) *time.Time {
	raw := issue.ResolutionDate
	if raw == "" {
		raw = issue.Updated
	}
	return jiraTime(raw)
}

// issueTimestamps returns the task's own creation and last update times so an
// import backfills history instead of stamping every row with the import time.
func issueTimestamps(issue jira.Issue) (time.Time, time.Time, bool) {
	created := jiraTime(issue.Created)
	updated := jiraTime(issue.Updated)
	if created == nil {
		created = updated
	}
	if updated == nil {
		updated = created
	}
	if created == nil {
		return time.Time{}, time.Time{}, false
	}
	return *created, *updated, true
}

// Jira descriptions mention sibling issues by key (ABC-123). Rewriting those
// keys into @TAG-<num> mentions turns them into the app's task refs, so the
// referenced task gets a backlink and the text stays clickable.
func rewriteJiraMentions(description string, resolve func(key string) (string, bool)) string {
	matches := jiraKeyMention.FindAllStringSubmatchIndex(description, -1)
	if len(matches) == 0 {
		return description
	}
	var out strings.Builder
	last := 0
	for _, m := range matches {
		start, end := m[0], m[1]
		// Leave URLs (/browse/ABC-123), existing @refs, and spans like
		// ABC-123-2 alone; replacing them would corrupt the markup.
		if start > 0 && (description[start-1] == '@' || description[start-1] == '/') {
			continue
		}
		if end+1 < len(description) && description[end] == '-' && description[end+1] >= '0' && description[end+1] <= '9' {
			continue
		}
		ref, ok := resolve(description[start:end])
		if !ok {
			continue
		}
		out.WriteString(description[last:start])
		out.WriteByte('@')
		out.WriteString(ref)
		last = end
	}
	if last == 0 {
		return description
	}
	out.WriteString(description[last:])
	return out.String()
}

type importedDescription struct {
	taskID      int64
	title       string
	description string
	priority    string
	dueDate     *string
	createdAt   time.Time
	updatedAt   time.Time
	hasTimes    bool
}

func (s *Server) jiraLinks(w http.ResponseWriter, r *http.Request) {
	links, err := s.Store.ListJiraLinksByWorkspace(r.Context(), optionalInt64Param(r, "workspace_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, links)
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

	s.jiraMu.Lock()
	if s.jiraRun != nil && s.jiraRun.Running {
		s.jiraMu.Unlock()
		writeError(w, http.StatusConflict, "an import is already running")
		return
	}
	run := &jiraImportState{Running: true, StartedAt: time.Now().UTC().Format(time.RFC3339)}
	s.jiraRun = run
	s.jiraMu.Unlock()

	// The run outlives the request so a proxy timeout cannot abort it, but a
	// hung Jira call must not block every later import.
	go func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		s.finishJiraImport(run, s.runJiraImport(ctx, body, run))
	}(context.WithoutCancel(r.Context()))

	writeJSON(w, http.StatusAccepted, jiraImportState{Running: true})
}

func (s *Server) jiraImportStatus(w http.ResponseWriter, r *http.Request) {
	s.jiraMu.Lock()
	defer s.jiraMu.Unlock()
	if s.jiraRun == nil {
		writeJSON(w, http.StatusOK, jiraImportState{})
		return
	}
	writeJSON(w, http.StatusOK, *s.jiraRun)
}

func (s *Server) finishJiraImport(run *jiraImportState, err error) {
	s.jiraMu.Lock()
	run.Running = false
	run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		run.Error = err.Error()
	}
	s.jiraMu.Unlock()
	s.broadcast("jira")
}

func (s *Server) reportJiraProgress(run *jiraImportState, imported, updated, skipped, total int) {
	s.jiraMu.Lock()
	run.Imported, run.Updated, run.Skipped, run.Total = imported, updated, skipped, total
	s.jiraMu.Unlock()
}

func (s *Server) runJiraImport(ctx context.Context, body jiraImportBody, run *jiraImportState) error {
	jql := body.JQL
	if jql == "" {
		if s.JiraDefaultProject == "" {
			return errors.New("jql is required (no JIRA_DEFAULT_PROJECT_KEY configured)")
		}
		jql = fmt.Sprintf(`project = "%s"`, s.JiraDefaultProject)
	}
	jql = onlyAssigned(jql)
	if !strings.Contains(strings.ToLower(jql), "order by") {
		jql += " ORDER BY updated DESC"
	}

	issues, err := s.Jira.SearchIssues(jql, 100)
	if err != nil {
		return err
	}
	s.reportJiraProgress(run, 0, 0, 0, len(issues))

	columns, err := s.Store.ListColumnsByBoard(ctx, body.BoardID)
	if err != nil || len(columns) == 0 {
		return errors.New("board has no columns")
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
	var changed []importedDescription

	inRun := make(map[string]bool, len(issues))
	for _, issue := range issues {
		inRun[issue.Key] = true
	}
	commentsByKey, commentErrs := s.prefetchComments(ctx, issues, body.ForceUpdate)
	created := map[string]int64{}

	lookupTask := func(key string) (store.Task, bool) {
		if id, ok := created[key]; ok {
			task, err := s.Store.GetTask(ctx, id)
			return task, err == nil
		}
		link, err := s.Store.GetJiraLinkByKey(ctx, key)
		if err != nil || link == nil {
			return store.Task{}, false
		}
		task, err := s.Store.GetTask(ctx, link.TaskID)
		return task, err == nil
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

	handle := func(issue jira.Issue, parentTaskID *int64, kind string) error {
		existing, err := s.Store.GetJiraLinkByKey(ctx, issue.Key)
		if err != nil {
			return err
		}

		if existing != nil && !body.ForceUpdate {
			skipped++
			return nil
		}

		comments := commentsByKey[issue.Key]
		if err := commentErrs[issue.Key]; err != nil {
			return err
		}

		columnID := resolveColumn(issue.Status)
		resolvedAt := completionTime(issue)
		createdAt, updatedAt, hasTimes := issueTimestamps(issue)

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
			if issue.Description != "" {
				changed = append(changed, importedDescription{taskID: existing.TaskID, title: issue.Summary, description: issue.Description, priority: task.Priority, dueDate: task.DueDate, createdAt: createdAt, updatedAt: updatedAt, hasTimes: hasTimes})
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
			if err := s.Store.SetTaskKind(ctx, existing.TaskID, kind); err != nil {
				return err
			}
			if err := importComments(existing.TaskID, issue, comments); err != nil {
				return err
			}
			if hasTimes {
				if err := s.Store.SetTaskTimestamps(ctx, existing.TaskID, createdAt, updatedAt); err != nil {
					return err
				}
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
			Priority:      "medium", Position: position, ParentTaskID: parentTaskID, Kind: kind,
		})
		if err != nil {
			return err
		}
		if resolvedAt != nil && columnIsDone[columnID] {
			if err := s.Store.SetTaskCompletedAt(ctx, task.ID, *resolvedAt); err != nil {
				return err
			}
		}
		if issue.Description != "" {
			changed = append(changed, importedDescription{taskID: task.ID, title: issue.Summary, description: issue.Description, priority: "medium", createdAt: createdAt, updatedAt: updatedAt, hasTimes: hasTimes})
		}
		if _, err := s.Store.CreateJiraLinkImported(ctx, task.ID, issue.Key, s.Jira.IssueURL(issue.Key), s.JiraDefaultProject, issue.Status); err != nil {
			return err
		}
		if err := importComments(task.ID, issue, comments); err != nil {
			return err
		}
		if hasTimes {
			if err := s.Store.SetTaskTimestamps(ctx, task.ID, createdAt, updatedAt); err != nil {
				return err
			}
		}
		created[issue.Key] = task.ID
		imported++
		return nil
	}

	// resolveParent returns the Fai parent for an issue. A step needs its parent,
	// so an unresolvable parent skips it; a task under an arc is deferred until
	// the arc exists and is imported at top level when the arc is not in the run.
	resolveParent := func(issue jira.Issue, kind string) (parentID *int64, deferIssue, skipIssue bool, err error) {
		if kind == "arc" || issue.ParentKey == "" {
			return nil, false, false, nil
		}
		parent, ok := lookupTask(issue.ParentKey)
		if !ok {
			if inRun[issue.ParentKey] {
				return nil, true, false, nil
			}
			if kind == "step" {
				return nil, false, true, nil
			}
			return nil, false, false, nil
		}
		if kind == "task" {
			if parent.Kind == "arc" {
				return &parent.ID, false, false, nil
			}
			return nil, false, false, nil
		}
		// A step attaches to a task or an arc; a nested step is not valid in Jira,
		// but climb out of one rather than dropping the issue.
		for parent.Kind == "step" && parent.ParentTaskID != nil {
			next, err := s.Store.GetTask(ctx, *parent.ParentTaskID)
			if err != nil {
				return nil, false, true, nil
			}
			parent = next
		}
		return &parent.ID, false, false, nil
	}

	pending := issues
	for len(pending) > 0 {
		var deferred []jira.Issue
		progressed := false
		for _, issue := range pending {
			kind := jiraKind(issue.IssueType)
			parentID, deferIssue, skipIssue, err := resolveParent(issue, kind)
			if err != nil {
				return err
			}
			if deferIssue {
				deferred = append(deferred, issue)
				continue
			}
			if skipIssue {
				skipped++
				s.reportJiraProgress(run, imported, updated, skipped, len(issues))
				continue
			}
			if err := handle(issue, parentID, kind); err != nil {
				return err
			}
			progressed = true
			s.reportJiraProgress(run, imported, updated, skipped, len(issues))
		}
		if !progressed {
			// Leftovers form a cycle, or their parent was skipped as an orphan.
			skipped += len(deferred)
			break
		}
		pending = deferred
	}

	// All links exist by now, so descriptions can reference issues imported
	// in the same run.
	var selfID int64
	refByKey := map[string]string{}
	resolveMention := func(key string) (string, bool) {
		if ref, seen := refByKey[key]; seen {
			return ref, true
		}
		target, err := s.Store.GetJiraTaskRef(ctx, key)
		if err != nil || target == nil || target.ID == selfID {
			return "", false
		}
		refByKey[key] = target.Ref
		return target.Ref, true
	}
	linked := 0
	for _, item := range changed {
		selfID = item.taskID
		description := rewriteJiraMentions(item.description, resolveMention)
		if description == item.description {
			continue
		}
		if _, err := s.Store.UpdateTask(ctx, item.taskID, item.title, description, item.priority, item.dueDate); err != nil {
			return err
		}
		if item.hasTimes {
			if err := s.Store.SetTaskTimestamps(ctx, item.taskID, item.createdAt, item.updatedAt); err != nil {
				return err
			}
		}
		linked++
	}

	s.jiraMu.Lock()
	run.Linked = linked
	s.jiraMu.Unlock()
	return nil
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
	ctx           context.Context
	s             *Server
	projectKey    string
	epicType      string
	taskType      string
	subtaskType   string
	epicLinkField string
	includeSubs   bool
	resp          *jiraExportResponse
	keys          map[int64]string
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

// issueType is the inverse of jiraKind: arcs export as Epics, steps as
// sub-tasks, and tasks as the configured task type.
func (e *jiraExporter) issueType(task store.Task) string {
	switch task.Kind {
	case "arc":
		return e.epicType
	case "step":
		return e.subtaskType
	default:
		return e.taskType
	}
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

	parent := ""
	epic := ""
	if task.Kind == "step" {
		parent = parentKey
	} else if task.Kind == "task" && parentKey != "" {
		epic = parentKey
	}
	key, err := e.s.Jira.CreateIssue(e.projectKey, task.Title, issueType, parent, epic, e.epicLinkField)
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

// ensureParent returns the parent's Jira key. A child is never exported alone:
// when the parent has no link the child is skipped.
func (e *jiraExporter) ensureParent(task store.Task) (string, bool) {
	if task.ParentTaskID == nil {
		return "", true
	}
	if key, ok := e.keys[*task.ParentTaskID]; ok {
		return key, true
	}
	link, err := e.s.Store.GetJiraLinkByTask(e.ctx, *task.ParentTaskID)
	if err != nil || link == nil {
		return "", false
	}
	e.keys[*task.ParentTaskID] = link.JiraKey
	return link.JiraKey, true
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
		e.record(jiraExportResult{TaskID: task.ID, Title: task.Title, Status: "skipped", Error: "parent is not linked to Jira"})
		return
	}
	children, err := e.s.Store.ListSubtasks(e.ctx, task.ID)
	if err != nil {
		e.record(jiraExportResult{TaskID: task.ID, Title: task.Title, Status: "failed", Error: err.Error()})
		return
	}

	if _, err := e.create(task, e.issueType(task), parentKey); err != nil {
		if e.includeSubs {
			for _, child := range children {
				if _, done := e.keys[child.ID]; done {
					continue
				}
				e.record(jiraExportResult{TaskID: child.ID, Title: child.Title, Status: "skipped", Error: "parent export failed"})
			}
		}
		return
	}
	if !e.includeSubs {
		return
	}
	for _, child := range children {
		if _, done := e.keys[child.ID]; done {
			continue
		}
		e.exportTree(child.ID)
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
	taskType := s.JiraTaskIssueType
	if taskType == "" {
		taskType = "Task"
	}
	epicLinkField := s.JiraEpicLinkField
	if epicLinkField == "" {
		epicLinkField = "parent"
	}
	resp := &jiraExportResponse{Results: []jiraExportResult{}}
	exporter := &jiraExporter{
		ctx: r.Context(), s: s, projectKey: projectKey,
		epicType: epicType, taskType: taskType, subtaskType: subtaskType, epicLinkField: epicLinkField,
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

// prefetchComments fetches comments concurrently. Sequentially this single step
// exceeded the 100s proxy timeout on a 100 issue import.
func (s *Server) prefetchComments(ctx context.Context, issues []jira.Issue, force bool) (map[string][]jira.Comment, map[string]error) {
	const workers = 8
	comments := make(map[string][]jira.Comment, len(issues))
	failures := map[string]error{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	keys := make(chan string)

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range keys {
				got, err := s.Jira.GetComments(key)
				mu.Lock()
				if err != nil {
					failures[key] = err
				} else {
					comments[key] = got
				}
				mu.Unlock()
			}
		}()
	}

	for _, issue := range issues {
		if !force {
			link, err := s.Store.GetJiraLinkByKey(ctx, issue.Key)
			if err != nil {
				failures[issue.Key] = err
				continue
			}
			if link != nil {
				continue
			}
		}
		keys <- issue.Key
	}
	close(keys)
	wg.Wait()
	return comments, failures
}
