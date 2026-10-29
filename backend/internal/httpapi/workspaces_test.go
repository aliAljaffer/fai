package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func newWorkspaceServer(t *testing.T) (*lite.Store, *httptest.Server) {
	t.Helper()
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	server := httptest.NewServer(NewRouter(&Server{Store: st}))
	t.Cleanup(server.Close)
	return st, server
}

func patchJSON(t *testing.T, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func decodeInto[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	defer res.Body.Close()
	var out T
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestWorkspaceCRUD(t *testing.T) {
	_, server := newWorkspaceServer(t)

	res := postJSON(t, server.URL+"/api/workspaces", `{"name":"Dev","tag":"dev"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", res.StatusCode)
	}
	created := decodeInto[store.Workspace](t, res)
	if created.Tag != "DEV" || created.Name != "Dev" {
		t.Fatalf("created = %#v", created)
	}

	res = postJSON(t, server.URL+"/api/workspaces", `{"name":"Too short","tag":"AB"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("short tag status = %d", res.StatusCode)
	}
	res.Body.Close()

	res = postJSON(t, server.URL+"/api/workspaces", `{"name":"Digits","tag":"AB1"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("digit tag status = %d", res.StatusCode)
	}
	res.Body.Close()

	res = postJSON(t, server.URL+"/api/workspaces", `{"name":"Dup","tag":"fai"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("duplicate tag status = %d", res.StatusCode)
	}
	res.Body.Close()

	res = httpGet(t, server.URL+"/api/workspaces")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", res.StatusCode)
	}
	list := decodeInto[[]store.Workspace](t, res)
	if len(list) != 2 {
		t.Errorf("workspaces = %#v", list)
	}

	res = patchJSON(t, server.URL+"/api/workspaces/"+strconv.FormatInt(created.ID, 10), `{"name":"Work","tag":"wrk"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d", res.StatusCode)
	}
	renamed := decodeInto[store.Workspace](t, res)
	if renamed.Name != "Work" || renamed.Tag != "WRK" {
		t.Errorf("renamed = %#v", renamed)
	}

	res = patchJSON(t, server.URL+"/api/workspaces/"+strconv.FormatInt(created.ID, 10), `{"tag":"FAI"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("duplicate rename status = %d", res.StatusCode)
	}
	res.Body.Close()

	res = patchJSON(t, server.URL+"/api/workspaces/9999", `{"tag":"ZZZ"}`)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("missing workspace status = %d", res.StatusCode)
	}
	res.Body.Close()
}

func httpGet(t *testing.T, url string) *http.Response {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestRenameKeepsOldReferenceResolving(t *testing.T) {
	_, server := newWorkspaceServer(t)

	res := postJSON(t, server.URL+"/api/workspaces", `{"name":"Dev","tag":"DEV"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("workspace status = %d", res.StatusCode)
	}
	workspace := decodeInto[store.Workspace](t, res)

	res = postJSON(t, server.URL+"/api/boards", `{"name":"Dev board","workspace_id":`+strconv.FormatInt(workspace.ID, 10)+`}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("board status = %d", res.StatusCode)
	}
	board := decodeInto[store.Board](t, res)

	res = postJSON(t, server.URL+"/api/boards/"+strconv.FormatInt(board.ID, 10)+"/columns", `{"name":"To Do","position":0}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("column status = %d", res.StatusCode)
	}
	column := decodeInto[store.BoardColumn](t, res)

	res = postJSON(t, server.URL+"/api/tasks", `{"board_id":`+strconv.FormatInt(board.ID, 10)+
		`,"column_id":`+strconv.FormatInt(column.ID, 10)+`,"title":"Ship the release"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("task status = %d", res.StatusCode)
	}
	task := decodeInto[store.Task](t, res)
	if task.Ref != "DEV-1" || task.RefNum != 1 || task.WorkspaceID != workspace.ID {
		t.Fatalf("task = %#v", task)
	}

	res = patchJSON(t, server.URL+"/api/workspaces/"+strconv.FormatInt(workspace.ID, 10), `{"tag":"WRK"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("rename status = %d", res.StatusCode)
	}
	res.Body.Close()

	for _, query := range []string{"DEV-1", "wrk-1"} {
		res = httpGet(t, server.URL+"/api/refs/resolve?q="+query)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("resolve status = %d", res.StatusCode)
		}
		refs := decodeInto[[]store.TaskRef](t, res)
		if len(refs) != 1 || refs[0].ID != task.ID {
			t.Fatalf("resolve %q = %#v", query, refs)
		}
		if refs[0].Ref != "WRK-1" || refs[0].WorkspaceName != "Dev" || refs[0].Kind != "task" {
			t.Errorf("resolve %q = %#v", query, refs[0])
		}
	}

	res = httpGet(t, server.URL+"/api/refs/resolve?q=SHIP")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("text resolve status = %d", res.StatusCode)
	}
	matches := decodeInto[[]store.TaskRef](t, res)
	if len(matches) != 1 || matches[0].ID != task.ID {
		t.Errorf("text matches = %#v", matches)
	}

	res = httpGet(t, server.URL+"/api/refs/resolve?q=")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("empty resolve status = %d", res.StatusCode)
	}
	empty := decodeInto[[]store.TaskRef](t, res)
	if len(empty) != 0 {
		t.Errorf("empty query = %#v", empty)
	}

	res = httpGet(t, server.URL+"/api/refs/resolve?q=release&workspace_id=1")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("scoped resolve status = %d", res.StatusCode)
	}
	if scoped := decodeInto[[]store.TaskRef](t, res); len(scoped) != 1 || scoped[0].ID != task.ID {
		t.Errorf("scoped resolve = %#v", scoped)
	}
}

func TestTaskJSONAndReferencesEndpoint(t *testing.T) {
	st, server := newWorkspaceServer(t)
	ctx := context.Background()

	res := httpGet(t, server.URL+"/api/boards/1/columns")
	columns := decodeInto[[]store.BoardColumn](t, res)
	if len(columns) == 0 {
		t.Fatal("seed board has no columns")
	}

	res = postJSON(t, server.URL+"/api/tasks", `{"board_id":1,"column_id":`+strconv.FormatInt(columns[0].ID, 10)+
		`,"title":"Seed task"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("task status = %d", res.StatusCode)
	}
	task := decodeInto[store.Task](t, res)
	if task.Ref != "FAI-1" || task.RefNum != 1 {
		t.Fatalf("task ref = %#v", task)
	}

	res = postJSON(t, server.URL+"/api/notes", `{"title":"Referrer","body_md":"blocks @FAI-1"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("note status = %d", res.StatusCode)
	}
	res.Body.Close()

	res = httpGet(t, server.URL+"/api/tasks/"+strconv.FormatInt(task.ID, 10))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get task status = %d", res.StatusCode)
	}
	fetched := decodeInto[store.Task](t, res)
	if fetched.Ref != "FAI-1" || fetched.WorkspaceID != task.WorkspaceID {
		t.Errorf("fetched = %#v", fetched)
	}

	res = httpGet(t, server.URL+"/api/tasks/"+strconv.FormatInt(task.ID, 10)+"/references")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("references status = %d", res.StatusCode)
	}
	backlinks := decodeInto[[]store.Backlink](t, res)
	if len(backlinks) != 1 || backlinks[0].Kind != "note" || backlinks[0].Title != "Referrer" {
		t.Errorf("backlinks = %#v", backlinks)
	}
	if backlinks[0].WorkspaceTag != "FAI" {
		t.Errorf("backlink tag = %q", backlinks[0].WorkspaceTag)
	}

	res = httpGet(t, server.URL+"/api/tasks/9999/references")
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("missing task references status = %d", res.StatusCode)
	}
	res.Body.Close()

	notes, err := st.ListNotes(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].WorkspaceID == 0 {
		t.Errorf("notes = %#v", notes)
	}
}

func TestExistingSQLiteDatabaseUpgrades(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	oldSchema := `
CREATE TABLE boards (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')));
CREATE TABLE board_columns (id INTEGER PRIMARY KEY AUTOINCREMENT, board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
  name TEXT NOT NULL, position INTEGER NOT NULL, is_done BOOLEAN NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')), UNIQUE (board_id, position));
CREATE TABLE tasks (id INTEGER PRIMARY KEY AUTOINCREMENT, board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
  column_id INTEGER NOT NULL REFERENCES board_columns(id) ON DELETE RESTRICT, title TEXT NOT NULL,
  parent_task_id INTEGER REFERENCES tasks(id) ON DELETE SET NULL,
  meeting_outcome_id INTEGER REFERENCES meeting_outcomes(id) ON DELETE SET NULL,
  description_md TEXT NOT NULL DEFAULT '', priority TEXT NOT NULL DEFAULT 'medium', due_date TEXT, position REAL NOT NULL,
  completed_at TEXT, archived BOOLEAN NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')));
CREATE TABLE tags (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, color TEXT NOT NULL DEFAULT '#888888');
CREATE TABLE task_tags (task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE, PRIMARY KEY (task_id, tag_id));
CREATE TABLE notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL UNIQUE, body_md TEXT NOT NULL DEFAULT '',
  task_id INTEGER REFERENCES tasks(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')));
CREATE TABLE reminders (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, remind_at TEXT NOT NULL,
  task_id INTEGER REFERENCES tasks(id) ON DELETE SET NULL, sent BOOLEAN NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')));
CREATE TABLE meeting_outcomes (id INTEGER PRIMARY KEY AUTOINCREMENT, occurred_on TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '', notes_md TEXT NOT NULL DEFAULT '', ends_at TEXT,
  outcome_prompt_sent BOOLEAN NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')));
INSERT INTO boards (name) VALUES ('Personal');
INSERT INTO board_columns (board_id, name, position, is_done) VALUES (1, 'To Do', 0, 0);
INSERT INTO tasks (board_id, column_id, title, position) VALUES (1, 1, 'Old task', 1024);
INSERT INTO notes (title, body_md) VALUES ('Old note', 'body');
INSERT INTO reminders (title, remind_at) VALUES ('Old reminder', '2099-01-01T10:00:00Z');
`
	if _, err := db.Exec(oldSchema); err != nil {
		t.Fatal(err)
	}
	db.Close()

	st, err := lite.New(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	workspaces, err := st.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 || workspaces[0].Tag != "FAI" {
		t.Fatalf("workspaces = %#v", workspaces)
	}
	task, err := st.GetTask(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if task.Ref != "FAI-1" || task.WorkspaceID != workspaces[0].ID {
		t.Errorf("upgraded task = %#v", task)
	}
	notes, err := st.ListNotes(ctx, &workspaces[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Errorf("upgraded notes = %#v", notes)
	}
	reminders, err := st.ListReminders(ctx, &workspaces[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reminders) != 1 {
		t.Errorf("upgraded reminders = %#v", reminders)
	}
	st.Close()

	st, err = lite.New(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st.Close()
	task, err = st.GetTask(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if task.Ref != "FAI-1" {
		t.Errorf("ref after reopen = %q", task.Ref)
	}
}
