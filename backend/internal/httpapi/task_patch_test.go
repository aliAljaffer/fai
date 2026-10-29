package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func TestTaskPartialPatchAndRestore(t *testing.T) {
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	server := httptest.NewServer(NewRouter(&Server{Store: st}))
	defer server.Close()

	res := postJSON(t, server.URL+"/api/tasks", `{"board_id":1,"column_id":1,"title":"Original","description_md":"keep me","priority":"high","due_date":"2026-01-02","kind":"task"}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", res.StatusCode)
	}
	var task store.Task
	if err := json.NewDecoder(res.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	base := server.URL + "/api/tasks/" + strconv.FormatInt(task.ID, 10)

	res = patchJSON(t, base, `{"title":"Renamed"}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("patch title = %d", res.StatusCode)
	}
	var patched store.Task
	if err := json.NewDecoder(res.Body).Decode(&patched); err != nil {
		t.Fatal(err)
	}
	if patched.Title != "Renamed" || patched.DescriptionMd != "keep me" || patched.Priority != "high" {
		t.Errorf("partial patch clobbered fields: %#v", patched)
	}
	if patched.DueDate == nil || *patched.DueDate != "2026-01-02" {
		t.Errorf("due date = %v, want 2026-01-02", patched.DueDate)
	}

	res = patchJSON(t, base, `{"due_date":null}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("clear due date = %d", res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(&patched); err != nil {
		t.Fatal(err)
	}
	if patched.DueDate != nil {
		t.Errorf("due date = %q, want nil", *patched.DueDate)
	}

	res = patchJSON(t, base, `{}`)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("empty patch = %d, want 400", res.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodDelete, base, nil)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", res.StatusCode)
	}

	res, err = http.Post(base+"/restore", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("restore = %d", res.StatusCode)
	}
	var restored store.Task
	if err := json.NewDecoder(res.Body).Decode(&restored); err != nil {
		t.Fatal(err)
	}
	if restored.Archived || restored.ID != task.ID {
		t.Errorf("restored = %#v", restored)
	}
	if _, err := st.GetTask(t.Context(), task.ID); err != nil {
		t.Errorf("restored task not readable: %v", err)
	}
}
