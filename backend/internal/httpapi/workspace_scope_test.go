package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func newScopeStore(t *testing.T) (*lite.Store, store.Workspace) {
	t.Helper()
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	workspace, err := st.CreateWorkspace(context.Background(), "Dev", "DEV")
	if err != nil {
		t.Fatal(err)
	}
	return st, workspace
}

func TestAIReminderUsesWorkspace(t *testing.T) {
	st, workspace := newScopeStore(t)

	provider := &fakeAI{reply: `{"title":"Reply to Mohammed","remind_at":"2099-01-01T10:00:00Z"}`}
	server := httptest.NewServer(NewRouter(&Server{Store: st, AI: provider}))
	defer server.Close()

	res := postJSON(t, server.URL+"/api/reminders/ai",
		`{"text":"Remind me to reply to Mohammed","workspace_id":`+strconv.FormatInt(workspace.ID, 10)+`}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("reminder status = %d", res.StatusCode)
	}
	created := decodeInto[store.Reminder](t, res)
	if created.WorkspaceID != workspace.ID {
		t.Errorf("reminder workspace = %d, want %d", created.WorkspaceID, workspace.ID)
	}

	scoped, err := st.ListReminders(context.Background(), &workspace.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(scoped) != 1 || scoped[0].ID != created.ID {
		t.Errorf("scoped reminders = %#v", scoped)
	}
}

func TestAIReminderRejectsUnknownWorkspace(t *testing.T) {
	st, _ := newScopeStore(t)

	provider := &fakeAI{reply: `{"title":"Reply to Mohammed","remind_at":"2099-01-01T10:00:00Z"}`}
	server := httptest.NewServer(NewRouter(&Server{Store: st, AI: provider}))
	defer server.Close()

	res := postJSON(t, server.URL+"/api/reminders/ai", `{"text":"Remind me to reply","workspace_id":9999}`)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown workspace status = %d", res.StatusCode)
	}
	if len(provider.messages) != 0 {
		t.Errorf("AI was called for an unknown workspace")
	}
}

func TestPendingMeetingOutcomesFilteredByWorkspace(t *testing.T) {
	st, workspace := newScopeStore(t)

	ctx := context.Background()
	past := time.Now().Add(-time.Hour)
	if _, err := st.CreateMeetingOutcome(ctx, "2026-10-20", "Personal sync", "", &past, 0); err != nil {
		t.Fatal(err)
	}
	dev, err := st.CreateMeetingOutcome(ctx, "2026-10-21", "Dev sync", "", &past, workspace.ID)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(NewRouter(&Server{Store: st}))
	defer server.Close()

	res := httpGet(t, server.URL+"/api/meeting-outcomes?pending=true")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("pending status = %d", res.StatusCode)
	}
	if all := decodeInto[[]store.MeetingOutcome](t, res); len(all) != 2 {
		t.Fatalf("pending outcomes = %#v", all)
	}

	res = httpGet(t, server.URL+"/api/meeting-outcomes?pending=true&workspace_id="+strconv.FormatInt(workspace.ID, 10))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("scoped pending status = %d", res.StatusCode)
	}
	scoped := decodeInto[[]store.MeetingOutcome](t, res)
	if len(scoped) != 1 || scoped[0].ID != dev.ID {
		t.Errorf("scoped pending outcomes = %#v", scoped)
	}
}
