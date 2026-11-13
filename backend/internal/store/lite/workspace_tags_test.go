package lite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func TestSharedTagsMoveIntoWorkspaces(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "fai.db")
	st, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := st.CreateWorkspace(ctx, "Work", "WRK")
	if err != nil {
		t.Fatal(err)
	}
	board, err := st.CreateBoard(ctx, "Work board", second.ID)
	if err != nil {
		t.Fatal(err)
	}
	column, err := st.CreateColumn(ctx, board.ID, "To Do", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	first, err := st.CreateTask(ctx, store.CreateTaskParams{BoardID: 1, ColumnID: 1, Title: "Home", Priority: "medium", Position: 1})
	if err != nil {
		t.Fatal(err)
	}
	work, err := st.CreateTask(ctx, store.CreateTaskParams{BoardID: board.ID, ColumnID: column.ID, Title: "Work", Priority: "medium", Position: 1})
	if err != nil {
		t.Fatal(err)
	}
	note, err := st.CreateNote(ctx, "Work note", "", nil, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		DROP TABLE tags;
		CREATE TABLE tags (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, color TEXT NOT NULL DEFAULT '#888888');
		INSERT INTO tags (id, name, color) VALUES (1, 'shared', '#111111'), (2, 'unused', '#222222'), (3, 'work-only', '#333333'), (4, 'note-only', '#444444');
		INSERT INTO task_tags (task_id, tag_id) VALUES (?1, 1), (?2, 1), (?2, 3);
		INSERT INTO note_tags (note_id, tag_id) VALUES (?3, 4);`, first.ID, work.ID, note.ID)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	st, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	firstID := int64(1)
	home, err := st.ListTags(ctx, &firstID)
	if err != nil {
		t.Fatal(err)
	}
	if len(home) != 2 || home[0] != (store.Tag{ID: 1, WorkspaceID: 1, Name: "shared", Color: "#111111"}) || home[1].Name != "unused" {
		t.Fatalf("first workspace tags = %#v", home)
	}
	away, err := st.ListTags(ctx, &second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(away) != 3 || away[0].Name != "note-only" || away[0].ID != 4 || away[1].Name != "shared" || away[1].ID <= 4 || away[1].Color != "#111111" || away[2].ID != 3 {
		t.Fatalf("second workspace tags = %#v", away)
	}

	workTags, err := st.ListTagsForTask(ctx, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(workTags) != 2 || workTags[0].ID != away[1].ID || workTags[1].ID != 3 {
		t.Fatalf("work task tags = %#v", workTags)
	}
	homeTags, err := st.ListTagsForTask(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(homeTags) != 1 || homeTags[0].ID != 1 {
		t.Fatalf("home task tags = %#v", homeTags)
	}

	var noteTag int64
	if err := st.db.QueryRowContext(ctx, `SELECT tag_id FROM note_tags WHERE note_id = ?`, note.ID).Scan(&noteTag); err != nil || noteTag != 4 {
		t.Fatalf("note tag = %d, err %v", noteTag, err)
	}
	rows, err := st.db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	if rows.Next() {
		t.Fatal("foreign key check reports a broken reference")
	}
	rows.Close()
	var enforced int
	if err := st.db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&enforced); err != nil || enforced != 1 {
		t.Fatalf("foreign keys = %d, err %v", enforced, err)
	}
}

func TestTagsBelongToOneWorkspace(t *testing.T) {
	ctx := context.Background()
	st, err := New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	second, err := st.CreateWorkspace(ctx, "Work", "WRK")
	if err != nil {
		t.Fatal(err)
	}
	homeTag, err := st.CreateTag(ctx, 1, "bug", "#111111")
	if err != nil {
		t.Fatal(err)
	}
	workTag, err := st.CreateTag(ctx, second.ID, "bug", "#222222")
	if err != nil {
		t.Fatalf("same name in another workspace: %v", err)
	}
	if _, err := st.CreateTag(ctx, 1, "bug", "#333333"); err == nil {
		t.Fatal("duplicate name in one workspace was accepted")
	}

	task, err := st.CreateTask(ctx, store.CreateTaskParams{BoardID: 1, ColumnID: 1, Title: "Fix #bug and #new", Priority: "medium", Position: 1, DescriptionMd: "#bug #fresh"})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddTaskTag(ctx, task.ID, workTag.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("tag from another workspace: err = %v", err)
	}
	for range 2 {
		if err := st.AddTaskTag(ctx, task.ID, homeTag.ID); err != nil {
			t.Fatal(err)
		}
	}
	tags, err := st.ListTagsForTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 || tags[0].ID != homeTag.ID || tags[1].Name != "fresh" || tags[1].WorkspaceID != 1 {
		t.Fatalf("task tags = %#v", tags)
	}
	away, err := st.ListTags(ctx, &second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(away) != 1 || away[0].ID != workTag.ID {
		t.Fatalf("second workspace tags = %#v", away)
	}
}
