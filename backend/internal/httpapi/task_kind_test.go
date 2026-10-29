package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func TestTaskKindHierarchy(t *testing.T) {
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	server := httptest.NewServer(NewRouter(&Server{Store: st}))
	defer server.Close()

	create := func(body map[string]any) (int, store.Task) {
		t.Helper()
		res := postJSON(t, server.URL+"/api/tasks", mustJSON(t, body))
		defer res.Body.Close()
		if res.StatusCode != http.StatusCreated {
			return res.StatusCode, store.Task{}
		}
		var task store.Task
		if err := json.NewDecoder(res.Body).Decode(&task); err != nil {
			t.Fatal(err)
		}
		return res.StatusCode, task
	}

	code, arc := create(map[string]any{"board_id": 1, "column_id": 1, "title": "Launch", "kind": "arc"})
	if code != http.StatusCreated || arc.Kind != "arc" {
		t.Fatalf("arc = %d %#v", code, arc)
	}

	if code, _ := create(map[string]any{"board_id": 1, "column_id": 1, "title": "Bad step", "kind": "step", "parent_task_id": arc.ID}); code != http.StatusBadRequest {
		t.Errorf("step under an arc = %d, want 400", code)
	}

	code, task := create(map[string]any{"board_id": 1, "column_id": 1, "title": "Pricing page", "kind": "task", "parent_task_id": arc.ID})
	if code != http.StatusCreated || task.Kind != "task" || task.ParentTaskID == nil || *task.ParentTaskID != arc.ID {
		t.Fatalf("task under arc = %d %#v", code, task)
	}

	code, step := create(map[string]any{"board_id": 1, "column_id": 1, "title": "Draft copy", "kind": "step", "parent_task_id": task.ID})
	if code != http.StatusCreated || step.Kind != "step" || step.ParentTaskID == nil || *step.ParentTaskID != task.ID {
		t.Fatalf("step under task = %d %#v", code, step)
	}

	res := patchJSON(t, server.URL+"/api/tasks/"+strconv.FormatInt(task.ID, 10)+"/type", `{"kind":"step","parent_task_id":null}`)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("converting a task with children to a step = %d, want 400", res.StatusCode)
	}

	res = patchJSON(t, server.URL+"/api/tasks/"+strconv.FormatInt(arc.ID, 10)+"/type", `{"kind":"task"}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("arc to task = %d", res.StatusCode)
	}
	var converted store.Task
	if err := json.NewDecoder(res.Body).Decode(&converted); err != nil {
		t.Fatal(err)
	}
	if converted.Kind != "task" {
		t.Errorf("converted kind = %q, want task", converted.Kind)
	}

	children, err := st.ListSubtasks(context.Background(), task.ID)
	if err != nil || len(children) != 1 {
		t.Fatalf("children = %#v, err = %v", children, err)
	}
}

func mustJSON(t *testing.T, body map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
