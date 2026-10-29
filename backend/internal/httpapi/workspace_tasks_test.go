package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func TestWorkspaceTasksResetKeepsColumnsAndCounter(t *testing.T) {
	_, server := newWorkspaceServer(t)

	res := httpGet(t, server.URL+"/api/boards/1/columns")
	columns := decodeInto[[]store.BoardColumn](t, res)
	if len(columns) == 0 {
		t.Fatal("seed board has no columns")
	}

	res = postJSON(t, server.URL+"/api/tasks", `{"board_id":1,"column_id":`+strconv.FormatInt(columns[0].ID, 10)+`,"title":"First"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("first task status = %d", res.StatusCode)
	}
	res.Body.Close()
	res = postJSON(t, server.URL+"/api/tasks", `{"board_id":1,"column_id":`+strconv.FormatInt(columns[0].ID, 10)+`,"title":"Second"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("second task status = %d", res.StatusCode)
	}
	res.Body.Close()

	res = postJSON(t, server.URL+"/api/workspaces/1/reset", `{"confirm":"nope"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong confirmation status = %d", res.StatusCode)
	}
	res.Body.Close()

	res = httpGet(t, server.URL+"/api/tasks?board_id=1")
	if tasks := decodeInto[[]store.Task](t, res); len(tasks) != 2 {
		t.Fatalf("tasks after rejected reset = %d", len(tasks))
	}

	res = postJSON(t, server.URL+"/api/workspaces/1/reset", `{"confirm":"I understand"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("reset status = %d", res.StatusCode)
	}
	reset := decodeInto[struct {
		Deleted int64 `json:"deleted"`
	}](t, res)
	if reset.Deleted != 2 {
		t.Errorf("deleted = %d", reset.Deleted)
	}

	res = httpGet(t, server.URL+"/api/tasks?board_id=1")
	if tasks := decodeInto[[]store.Task](t, res); len(tasks) != 0 {
		t.Errorf("tasks after reset = %d", len(tasks))
	}
	res = httpGet(t, server.URL+"/api/boards/1/columns")
	if kept := decodeInto[[]store.BoardColumn](t, res); len(kept) != len(columns) {
		t.Errorf("columns after reset = %d, want %d", len(kept), len(columns))
	}

	res = postJSON(t, server.URL+"/api/tasks", `{"board_id":1,"column_id":`+strconv.FormatInt(columns[0].ID, 10)+`,"title":"After reset"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("task after reset status = %d", res.StatusCode)
	}
	task := decodeInto[store.Task](t, res)
	if task.Ref != "FAI-1" || task.RefNum != 1 {
		t.Errorf("ref after reset = %q (%d)", task.Ref, task.RefNum)
	}
}

func TestWorkspaceTasksExportImport(t *testing.T) {
	_, server := newWorkspaceServer(t)

	res := httpGet(t, server.URL+"/api/boards/1/columns")
	columns := decodeInto[[]store.BoardColumn](t, res)
	if len(columns) < 2 {
		t.Fatal("seed board needs at least two columns")
	}
	todo, done := columns[0], columns[1]
	for _, column := range columns {
		if column.IsDone {
			done = column
		}
	}

	res = postJSON(t, server.URL+"/api/tasks", `{"board_id":1,"column_id":`+strconv.FormatInt(todo.ID, 10)+`,"title":"Parent","description_md":"has #alpha tag","kind":"task"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("parent status = %d", res.StatusCode)
	}
	parent := decodeInto[store.Task](t, res)

	res = postJSON(t, server.URL+"/api/tasks", `{"board_id":1,"column_id":`+strconv.FormatInt(done.ID, 10)+`,"title":"Child","description_md":"child #alpha","kind":"step","parent_task_id":`+strconv.FormatInt(parent.ID, 10)+`}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("child status = %d", res.StatusCode)
	}
	child := decodeInto[store.Task](t, res)

	res = httpGet(t, server.URL+"/api/workspaces/1/tasks/export")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("export status = %d", res.StatusCode)
	}
	var exported workspaceTasksExport
	if err := json.NewDecoder(res.Body).Decode(&exported); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if len(exported.Tasks) != 2 {
		t.Fatalf("exported tasks = %d", len(exported.Tasks))
	}
	if exported.Tasks[0].Ref != parent.Ref || exported.Tasks[1].Ref != child.Ref {
		t.Errorf("export order = %q, %q", exported.Tasks[0].Ref, exported.Tasks[1].Ref)
	}
	if exported.Tasks[1].ParentRef == nil || *exported.Tasks[1].ParentRef != parent.Ref {
		t.Errorf("parent ref = %v", exported.Tasks[1].ParentRef)
	}
	if len(exported.Tasks[0].Tags) != 1 || exported.Tasks[0].Tags[0].Name != "alpha" {
		t.Errorf("tags = %#v", exported.Tasks[0].Tags)
	}
	if exported.Tasks[1].CompletedAt == nil {
		t.Error("done-column task has no completed_at")
	}

	res = postJSON(t, server.URL+"/api/workspaces/1/reset", `{"confirm":"I understand"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("reset status = %d", res.StatusCode)
	}
	res.Body.Close()

	body, err := json.Marshal(exported)
	if err != nil {
		t.Fatal(err)
	}
	res = postJSON(t, server.URL+"/api/workspaces/1/tasks/import", string(body))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status = %d", res.StatusCode)
	}
	imported := decodeInto[struct {
		Imported int `json:"imported"`
	}](t, res)
	if imported.Imported != 2 {
		t.Errorf("imported = %d", imported.Imported)
	}

	res = httpGet(t, server.URL+"/api/tasks?board_id=1")
	tasks := decodeInto[[]store.Task](t, res)
	if len(tasks) != 2 {
		t.Fatalf("tasks after import = %d", len(tasks))
	}
	byRef := map[string]store.Task{}
	for _, task := range tasks {
		byRef[task.Ref] = task
	}
	restoredParent, ok := byRef["FAI-1"]
	if !ok {
		t.Fatalf("restored refs = %#v", byRef)
	}
	restoredChild, ok := byRef["FAI-2"]
	if !ok {
		t.Fatalf("restored refs = %#v", byRef)
	}
	if restoredChild.ParentTaskID == nil || *restoredChild.ParentTaskID != restoredParent.ID {
		t.Errorf("restored parent id = %v, want %d", restoredChild.ParentTaskID, restoredParent.ID)
	}
	if restoredChild.Kind != "step" {
		t.Errorf("restored child kind = %q", restoredChild.Kind)
	}
	if restoredChild.CompletedAt == nil {
		t.Error("restored child lost completed_at")
	}
	if len(restoredParent.Tags) != 1 || restoredParent.Tags[0].Name != "alpha" {
		t.Errorf("restored tags = %#v", restoredParent.Tags)
	}
	if restoredChild.CompletedAt != nil && !restoredChild.CompletedAt.Truncate(time.Second).Equal(child.CompletedAt.Truncate(time.Second)) {
		t.Errorf("restored completed_at = %v, want %v", restoredChild.CompletedAt, child.CompletedAt)
	}
}

func TestWorkspaceTasksImportCreatesMissingBoardAndColumn(t *testing.T) {
	_, server := newWorkspaceServer(t)

	body := `{
		"version": 1,
		"workspace": {"name": "Personal", "tag": "FAI"},
		"tasks": [
			{"ref": "FAI-5", "ref_num": 5, "board": "New board", "column": "Doing", "title": "New task", "kind": "task", "position": 1024}
		]
	}`
	res := postJSON(t, server.URL+"/api/workspaces/1/tasks/import", body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status = %d", res.StatusCode)
	}
	res.Body.Close()

	res = httpGet(t, server.URL+"/api/boards?workspace_id=1")
	boards := decodeInto[[]store.Board](t, res)
	var newBoard *store.Board
	for i := range boards {
		if boards[i].Name == "New board" {
			newBoard = &boards[i]
		}
	}
	if newBoard == nil {
		t.Fatalf("boards = %#v", boards)
	}
	res = httpGet(t, server.URL+"/api/boards/"+strconv.FormatInt(newBoard.ID, 10)+"/columns")
	columns := decodeInto[[]store.BoardColumn](t, res)
	if len(columns) != 1 || columns[0].Name != "Doing" {
		t.Errorf("columns = %#v", columns)
	}

	res = postJSON(t, server.URL+"/api/tasks", `{"board_id":`+strconv.FormatInt(newBoard.ID, 10)+`,"column_id":`+strconv.FormatInt(columns[0].ID, 10)+`,"title":"Next"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("task after import status = %d", res.StatusCode)
	}
	task := decodeInto[store.Task](t, res)
	if task.Ref != "FAI-6" {
		t.Errorf("ref after imported ref 5 = %q", task.Ref)
	}
}
