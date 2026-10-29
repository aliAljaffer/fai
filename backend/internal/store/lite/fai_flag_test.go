package lite

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func TestCreateTaskCreatedByFai(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	manual, err := st.CreateTask(t.Context(), store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: "Manual", Priority: "medium", Position: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if manual.CreatedByFai {
		t.Fatal("manual task was marked as created by Fai")
	}

	created, err := st.CreateTask(t.Context(), store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: "From Fai", Priority: "medium", Position: 2, CreatedByFai: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.CreatedByFai {
		t.Fatal("Fai task lost its flag on create")
	}

	got, err := st.GetTask(t.Context(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CreatedByFai {
		t.Fatal("Fai task lost its flag on read")
	}
}

func TestCreatedByFaiUpgradeDefaultsFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fai.db")
	st, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	existing, err := st.CreateTask(t.Context(), store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: "Before the upgrade", Priority: "medium", Position: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	st.Close()

	// Recreate the pre-flag schema, as an install that predates the column.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`ALTER TABLE tasks DROP COLUMN created_by_fai`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	upgraded, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()

	got, err := upgraded.GetTask(t.Context(), existing.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CreatedByFai {
		t.Fatal("task from before the upgrade was marked as created by Fai")
	}
}
