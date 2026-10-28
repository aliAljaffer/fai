package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/jira"
	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

const jiraSearchBody = `{"issues":[
  {"key":"DEV-2","fields":{"summary":"Child","status":{"name":"To Do"},"parent":{"key":"DEV-1"}}},
  {"key":"DEV-1","fields":{"summary":"Parent","status":{"name":"To Do"}}},
  {"key":"DEV-3","fields":{"summary":"Orphan","status":{"name":"To Do"},"parent":{"key":"DEV-9"}}}
],"isLast":true}`

const jiraMovedBody = `{"issues":[
  {"key":"DEV-2","fields":{"summary":"Child","status":{"name":"To Do"},"parent":{"key":"DEV-4"}}},
  {"key":"DEV-1","fields":{"summary":"Parent","status":{"name":"To Do"}}},
  {"key":"DEV-4","fields":{"summary":"Other parent","status":{"name":"To Do"}}},
  {"key":"DEV-3","fields":{"summary":"Orphan","status":{"name":"To Do"},"parent":{"key":"DEV-9"}}}
],"isLast":true}`

func TestJiraImportNestsSubtasks(t *testing.T) {
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	searchBody := jiraSearchBody
	jiraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/comment") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"comments":[],"total":0}`))
			return
		}
		if r.URL.Path != "/rest/api/3/search/jql" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(searchBody))
	}))
	defer jiraServer.Close()

	server := httptest.NewServer(NewRouter(&Server{
		Store:              st,
		Jira:               jira.New(jiraServer.URL, "me@example.com", "token"),
		JiraDefaultProject: "DEV",
	}))
	defer server.Close()

	runImport := func(body string) map[string]int {
		t.Helper()
		res := postJSON(t, server.URL+"/api/jira/import", body)
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("import status = %d", res.StatusCode)
		}
		var counts map[string]int
		if err := json.NewDecoder(res.Body).Decode(&counts); err != nil {
			t.Fatal(err)
		}
		return counts
	}
	tasksByTitle := func() map[string]store.Task {
		t.Helper()
		tasks, err := st.ListTasks(context.Background(), store.ListTasksParams{})
		if err != nil {
			t.Fatal(err)
		}
		byTitle := map[string]store.Task{}
		for _, task := range tasks {
			byTitle[task.Title] = task
		}
		return byTitle
	}

	first := runImport(`{"board_id":1,"jql":"project = DEV"}`)
	if first["imported"] != 2 || first["skipped"] != 1 {
		t.Fatalf("first run = %#v", first)
	}

	tasks := tasksByTitle()
	child, ok := tasks["Child"]
	if !ok || child.ParentTaskID == nil || *child.ParentTaskID != tasks["Parent"].ID {
		t.Fatalf("child task = %#v", child)
	}
	if _, ok := tasks["Orphan"]; ok {
		t.Fatal("orphan subtask was imported")
	}

	second := runImport(`{"board_id":1,"jql":"project = DEV"}`)
	if second["imported"] != 0 || second["skipped"] != 3 {
		t.Fatalf("second run = %#v", second)
	}
	if tasks := tasksByTitle(); len(tasks) != 2 {
		t.Fatalf("second run changed task count: %d", len(tasks))
	}

	searchBody = jiraMovedBody
	moved := runImport(`{"board_id":1,"jql":"project = DEV","force_update":true}`)
	if moved["imported"] != 1 || moved["updated"] != 2 || moved["skipped"] != 1 {
		t.Fatalf("force run = %#v", moved)
	}
	tasks = tasksByTitle()
	child = tasks["Child"]
	if child.ParentTaskID == nil || *child.ParentTaskID != tasks["Other parent"].ID {
		t.Fatalf("moved child task = %#v", child)
	}
}

func TestJiraImportOnlyAssignedAndBackfillsCompletion(t *testing.T) {
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	var gotJQL string
	searchBody := `{"issues":[
	  {"key":"DEV-7","fields":{"summary":"Done issue","status":{"name":"Done"},
	    "resolutiondate":"2026-03-04T10:00:00.000+0000","updated":"2026-03-05T08:00:00.000+0000"}}
	],"isLast":true}`
	jiraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/comment") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"comments":[],"total":0}`))
			return
		}
		if r.URL.Path != "/rest/api/3/search/jql" {
			http.NotFound(w, r)
			return
		}
		gotJQL = r.URL.Query().Get("jql")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(searchBody))
	}))
	defer jiraServer.Close()

	server := httptest.NewServer(NewRouter(&Server{
		Store:               st,
		Jira:                jira.New(jiraServer.URL, "me@example.com", "token"),
		JiraDefaultProject:  "DEV",
		JiraStatusColumnMap: map[string]string{"Done": "Done"},
	}))
	defer server.Close()

	res := postJSON(t, server.URL+"/api/jira/import", `{"board_id":1}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status = %d", res.StatusCode)
	}

	if !strings.Contains(gotJQL, "assignee = currentUser()") {
		t.Errorf("import JQL = %q, want current-user filter", gotJQL)
	}

	tasks, err := st.ListTasks(context.Background(), store.ListTasksParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("imported %d tasks, want 1", len(tasks))
	}
	task := tasks[0]
	if task.ColumnID != 3 {
		t.Errorf("done issue column = %d, want 3", task.ColumnID)
	}
	if task.CompletedAt == nil || task.CompletedAt.Format("2006-01-02") != "2026-03-04" {
		t.Errorf("completed_at = %v, want 2026-03-04", task.CompletedAt)
	}
}

func TestJiraImportDescribesAndComments(t *testing.T) {
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	searchBody := `{"issues":[{"key":"DEV-7","fields":{
	  "summary":"Issue",
	  "description":{"type":"doc","version":1,"content":[
	    {"type":"paragraph","content":[{"type":"text","text":"Details","marks":[{"type":"strong"}]}]}
	  ]},
	  "assignee":{"accountId":"me","displayName":"Me"},
	  "status":{"name":"To Do"}
	}}],"isLast":true}`
	comment := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"%s"}]}]}`
	commentsBody := `{"comments":[
	  {"id":"1","author":{"accountId":"me","displayName":"Me"},"body":` + fmt.Sprintf(comment, "First") + `},
	  {"id":"2","author":{"accountId":"other","displayName":"Other"},"body":` + fmt.Sprintf(comment, "Other") + `},
	  {"id":"3","parentId":"2","author":{"accountId":"me","displayName":"Me"},"body":` + fmt.Sprintf(comment, "Reply to other") + `},
	  {"id":"4","parentId":"1","author":{"accountId":"me","displayName":"Me"},"body":` + fmt.Sprintf(comment, "Own reply") + `}
	],"total":4}`

	jiraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/rest/api/3/search/jql":
			w.Write([]byte(searchBody))
		case strings.HasSuffix(r.URL.Path, "/comment"):
			w.Write([]byte(commentsBody))
		default:
			http.NotFound(w, r)
		}
	}))
	defer jiraServer.Close()

	server := httptest.NewServer(NewRouter(&Server{
		Store:              st,
		Jira:               jira.New(jiraServer.URL, "me@example.com", "token"),
		JiraDefaultProject: "DEV",
	}))
	defer server.Close()

	runImport := func(body string) {
		t.Helper()
		res := postJSON(t, server.URL+"/api/jira/import", body)
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("import status = %d", res.StatusCode)
		}
	}

	runImport(`{"board_id":1}`)
	tasks, err := st.ListTasks(context.Background(), store.ListTasksParams{})
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %#v, err = %v", tasks, err)
	}
	task := tasks[0]
	if task.DescriptionMd != "**Details**" {
		t.Errorf("description = %q", task.DescriptionMd)
	}
	comments, err := st.ListTaskComments(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 2 || *comments[0].ExternalID != "1" || *comments[1].ExternalID != "4" {
		t.Fatalf("comments = %#v", comments)
	}
	if comments[0].Author != "Me" || comments[0].BodyMd != "First" {
		t.Errorf("comment = %#v", comments[0])
	}

	runImport(`{"board_id":1,"force_update":true}`)
	comments, err = st.ListTaskComments(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 2 {
		t.Fatalf("re-import duplicated comments: %d", len(comments))
	}
}

type jiraCreatedIssue struct {
	Key       string
	Summary   string
	IssueType string
	ParentKey string
}

type jiraFake struct {
	mu      sync.Mutex
	created []jiraCreatedIssue
	failFor string
}

func (f *jiraFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/rest/api/3/issue" {
		http.NotFound(w, r)
		return
	}
	var body struct {
		Fields struct {
			Summary   string `json:"summary"`
			IssueType struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Parent *struct {
				Key string `json:"key"`
			} `json:"parent"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	if body.Fields.Summary == f.failFor {
		f.mu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	issue := jiraCreatedIssue{Summary: body.Fields.Summary, IssueType: body.Fields.IssueType.Name}
	if body.Fields.Parent != nil {
		issue.ParentKey = body.Fields.Parent.Key
	}
	f.created = append(f.created, issue)
	key := fmt.Sprintf("DEV-%d", len(f.created))
	f.created[len(f.created)-1].Key = key
	f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"key": key})
}

func TestJiraExportTree(t *testing.T) {
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	fake := &jiraFake{failFor: "Broken parent"}
	jiraServer := httptest.NewServer(fake)
	defer jiraServer.Close()

	server := httptest.NewServer(NewRouter(&Server{
		Store:                st,
		Jira:                 jira.New(jiraServer.URL, "me@example.com", "token"),
		JiraDefaultProject:   "DEV",
		JiraEpicIssueType:    "Epic",
		JiraSubtaskIssueType: "Sub-task",
	}))
	defer server.Close()

	ctx := context.Background()
	newTask := func(title string, parent *int64) store.Task {
		t.Helper()
		task, err := st.CreateTask(ctx, store.CreateTaskParams{
			BoardID: 1, ColumnID: 1, Title: title, Priority: "medium", Position: 1, ParentTaskID: parent,
		})
		if err != nil {
			t.Fatal(err)
		}
		return task
	}

	big := newTask("Big parent", nil)
	for i := 1; i <= 4; i++ {
		newTask(fmt.Sprintf("Big child %d", i), &big.ID)
	}
	small := newTask("Small parent", nil)
	for i := 1; i <= 3; i++ {
		newTask(fmt.Sprintf("Small child %d", i), &small.ID)
	}
	solo := newTask("Solo", nil)
	broken := newTask("Broken parent", nil)
	newTask("Broken child 1", &broken.ID)
	newTask("Broken child 2", &broken.ID)

	runExport := func(body string) jiraExportResponse {
		t.Helper()
		res := postJSON(t, server.URL+"/api/jira/export", body)
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("export status = %d", res.StatusCode)
		}
		var out jiraExportResponse
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	createdBySummary := func(summary string) jiraCreatedIssue {
		t.Helper()
		fake.mu.Lock()
		defer fake.mu.Unlock()
		for _, issue := range fake.created {
			if issue.Summary == summary {
				return issue
			}
		}
		t.Fatalf("no jira issue created for %q", summary)
		return jiraCreatedIssue{}
	}
	resultFor := func(out jiraExportResponse, taskID int64) jiraExportResult {
		t.Helper()
		for _, res := range out.Results {
			if res.TaskID == taskID {
				return res
			}
		}
		t.Fatalf("no result for task %d", taskID)
		return jiraExportResult{}
	}

	ids := fmt.Sprintf(`{"task_ids":[%d,%d,%d,%d]}`, big.ID, small.ID, solo.ID, broken.ID)
	first := runExport(ids)
	if first.Created != 10 || first.Skipped != 0 || first.Failed != 3 {
		t.Fatalf("first export = %#v", first)
	}

	if got := createdBySummary("Big parent").IssueType; got != "Epic" {
		t.Errorf("parent with 4 subtasks type = %q, want Epic", got)
	}
	if got := createdBySummary("Small parent").IssueType; got != "Task" {
		t.Errorf("parent with 3 subtasks type = %q, want Task", got)
	}
	if got := createdBySummary("Solo").IssueType; got != "Task" {
		t.Errorf("task without subtasks type = %q, want Task", got)
	}
	bigChild := createdBySummary("Big child 1")
	if bigChild.IssueType != "Sub-task" || bigChild.ParentKey != "DEV-1" {
		t.Errorf("big child = %#v", bigChild)
	}
	if res := resultFor(first, broken.ID); res.Status != "failed" {
		t.Errorf("broken parent result = %#v", res)
	}

	fake.mu.Lock()
	before := len(fake.created)
	fake.mu.Unlock()
	second := runExport(ids)
	if second.Created != 0 || second.Skipped != 10 || second.Failed != 3 {
		t.Fatalf("second export = %#v", second)
	}
	fake.mu.Lock()
	after := len(fake.created)
	fake.mu.Unlock()
	if after != before {
		t.Fatalf("second export created %d new issues", after-before)
	}

	loneParent := newTask("Lone parent", nil)
	loneChild := newTask("Lone child", &loneParent.ID)
	third := runExport(fmt.Sprintf(`{"task_id":%d}`, loneChild.ID))
	if third.Created != 2 || third.Failed != 0 {
		t.Fatalf("child-only export = %#v", third)
	}
	parentIssue := createdBySummary("Lone parent")
	childIssue := createdBySummary("Lone child")
	if parentIssue.IssueType != "Task" || childIssue.IssueType != "Sub-task" || childIssue.ParentKey != parentIssue.Key {
		t.Errorf("child-only export parent = %#v child = %#v", parentIssue, childIssue)
	}
}
