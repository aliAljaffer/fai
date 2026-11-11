package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func TestAITaskCreatesTaskFromNaturalLanguage(t *testing.T) {
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	provider := &fakeAI{reply: `{"title":"Call the plumber","due_date":"2026-11-02","priority":"high"}`}
	server := httptest.NewServer(NewRouter(&Server{Store: st, AI: provider}))
	defer server.Close()

	res := postJSON(t, server.URL+"/api/tasks/ai", `{"text":"Call the plumber next Monday, high priority","board_id":1,"column_id":1}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var task store.Task
	if err := json.NewDecoder(res.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	if task.Title != "Call the plumber" || task.Priority != "high" {
		t.Fatalf("task = %#v", task)
	}
	if task.DueDate == nil || *task.DueDate != "2026-11-02" {
		t.Errorf("due date = %v", task.DueDate)
	}
	if !strings.Contains(provider.messages[0].Content, "Call the plumber next Monday") {
		t.Errorf("prompt = %q", provider.messages[0].Content)
	}
}

func TestAITaskFallsBackToMediumPriority(t *testing.T) {
	title, due, priority, repeat, err := parseAITask(`{"title":"Write the report","due_date":"","priority":"urgent"}`)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Write the report" || due != nil || priority != "medium" || repeat != nil {
		t.Fatalf("parsed = %q, %v, %q", title, due, priority)
	}
}

func TestParseAITaskRepeat(t *testing.T) {
	cases := []struct {
		name  string
		reply string
		want  bool
	}{
		{"valid with due date", `{"title":"Water plants","due_date":"2026-11-02","repeat":{"freq":"weekly","interval":1}}`, true},
		{"no due date", `{"title":"Water plants","due_date":"","repeat":{"freq":"weekly","interval":1}}`, false},
		{"invalid rule", `{"title":"Water plants","due_date":"2026-11-02","repeat":{"freq":"hourly","interval":1}}`, false},
		{"absent", `{"title":"Water plants","due_date":"2026-11-02"}`, false},
	}
	for _, tc := range cases {
		_, _, _, repeat, err := parseAITask(tc.reply)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if (repeat != nil) != tc.want {
			t.Errorf("%s: repeat = %#v", tc.name, repeat)
		}
		if repeat != nil && (repeat.Anchor != "2026-11-02" || repeat.Rule.Freq != "weekly") {
			t.Errorf("%s: repeat = %#v", tc.name, repeat)
		}
	}
}
