package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
