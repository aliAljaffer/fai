package lite

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func TestTaskKindUpgradeBackfillsSteps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fai.db")
	st, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	parent, err := st.CreateTask(t.Context(), store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: "Parent", Priority: "medium", Position: 1, Kind: "task",
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := st.CreateTask(t.Context(), store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: "Child", Priority: "medium", Position: 2,
		ParentTaskID: &parent.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	st.Close()

	// Recreate the pre-kind schema, as an install that predates the column.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`ALTER TABLE tasks DROP COLUMN kind`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	upgraded, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()

	parent, err = upgraded.GetTask(t.Context(), parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	child, err = upgraded.GetTask(t.Context(), child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if parent.Kind != "task" || child.Kind != "step" {
		t.Errorf("kinds after upgrade = %q / %q, want task / step", parent.Kind, child.Kind)
	}
}
