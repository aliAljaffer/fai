package telegram

import (
	"path/filepath"
	"testing"
)

func TestFileOffsetStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "offset")
	store := FileOffsetStore{Path: path}

	value, err := store.Load()
	if err != nil || value != 0 {
		t.Fatalf("initial load = %d, %v", value, err)
	}
	if err := store.Save(42); err != nil {
		t.Fatal(err)
	}
	value, err = store.Load()
	if err != nil || value != 42 {
		t.Fatalf("load = %d, %v", value, err)
	}
	if err := store.Save(43); err != nil {
		t.Fatal(err)
	}
	value, err = store.Load()
	if err != nil || value != 43 {
		t.Fatalf("reload = %d, %v", value, err)
	}
}
