package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func feedRequest(server *Server, query string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	server.calendarFeed(rec, httptest.NewRequest(http.MethodGet, "/api/calendar.ics"+query, nil))
	return rec
}

func TestCalendarFeedToken(t *testing.T) {
	server := &Server{Store: actionTestStore(t), ICSToken: "secret"}
	for query, want := range map[string]int{
		"":                             http.StatusUnauthorized,
		"?token=wrong":                 http.StatusUnauthorized,
		"?token=secret":                http.StatusOK,
		"?token=secret&workspace_id=x": http.StatusBadRequest,
	} {
		if got := feedRequest(server, query).Code; got != want {
			t.Errorf("%q: status = %d, want %d", query, got, want)
		}
	}
	if got := feedRequest(&Server{Store: server.Store}, "?token=").Code; got != http.StatusServiceUnavailable {
		t.Errorf("unconfigured status = %d", got)
	}
}

func TestCalendarFeedContents(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	dev, err := st.CreateWorkspace(ctx, "Dev", "DEV")
	if err != nil {
		t.Fatal(err)
	}
	board, err := st.CreateBoard(ctx, "Dev", dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	column, err := st.CreateColumn(ctx, board.ID, "To Do", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	newTask := func(boardID, columnID int64, title, due string) store.Task {
		t.Helper()
		params := store.CreateTaskParams{BoardID: boardID, ColumnID: columnID, Title: title, Priority: "medium", Position: 1024}
		if due != "" {
			params.DueDate = &due
		}
		task, err := st.CreateTask(ctx, params)
		if err != nil {
			t.Fatal(err)
		}
		return task
	}
	newTask(1, 1, "Personal open", "2026-12-01")
	newTask(1, 1, "Personal undated", "")
	done := newTask(1, 1, "Personal done", "2026-12-02")
	if err := st.SetTaskCompletedAt(ctx, done.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	newTask(board.ID, column.ID, "Dev open", "2026-12-03")

	remindAt := time.Date(2026, 12, 1, 9, 30, 0, 0, time.FixedZone("AST", 3*3600))
	if _, err := st.CreateReminder(ctx, "Personal reminder", remindAt, nil, 0, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateReminder(ctx, "Dev reminder", remindAt, nil, dev.ID, nil); err != nil {
		t.Fatal(err)
	}
	finished, err := st.CreateReminder(ctx, "Finished reminder", remindAt, nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetReminderCompleted(ctx, finished.ID, true); err != nil {
		t.Fatal(err)
	}

	server := &Server{Store: st, ICSToken: "secret", PublicURL: "https://fai.example"}
	all := feedRequest(server, "?token=secret").Body.String()
	for _, want := range []string{"SUMMARY:Personal open", "SUMMARY:Dev open", "SUMMARY:Personal reminder", "SUMMARY:Dev reminder", "DTSTART:20261201T063000Z\r\n", "UID:reminder-1@fai"} {
		if !strings.Contains(all, want) {
			t.Errorf("feed is missing %q:\n%s", want, all)
		}
	}
	for _, unwanted := range []string{"Personal undated", "Personal done", "Finished reminder"} {
		if strings.Contains(all, unwanted) {
			t.Errorf("feed contains %q", unwanted)
		}
	}

	scoped := feedRequest(server, "?token=secret&workspace_id="+strconv.FormatInt(dev.ID, 10)).Body.String()
	for _, want := range []string{"SUMMARY:Dev open", "SUMMARY:Dev reminder"} {
		if !strings.Contains(scoped, want) {
			t.Errorf("scoped feed is missing %q", want)
		}
	}
	for _, unwanted := range []string{"Personal open", "Personal reminder"} {
		if strings.Contains(scoped, unwanted) {
			t.Errorf("scoped feed contains %q", unwanted)
		}
	}
}
