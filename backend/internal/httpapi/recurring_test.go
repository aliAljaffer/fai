package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/recurrence"
	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func idURL(base string, id int64) string {
	return base + "/" + strconv.FormatInt(id, 10)
}

func expectStatus(t *testing.T, res *http.Response, want int) {
	t.Helper()
	if res.StatusCode != want {
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		t.Fatalf("status = %d, want %d: %s", res.StatusCode, want, body)
	}
}

func createReminderHTTP(t *testing.T, server string, body string) store.Reminder {
	t.Helper()
	res := postJSON(t, server+"/api/reminders", body)
	expectStatus(t, res, http.StatusCreated)
	return decodeInto[store.Reminder](t, res)
}

func createTaskHTTP(t *testing.T, server string, body string) store.Task {
	t.Helper()
	res := postJSON(t, server+"/api/tasks", body)
	expectStatus(t, res, http.StatusCreated)
	return decodeInto[store.Task](t, res)
}

func TestReminderRepeatIsValidatedAndAlwaysPresent(t *testing.T) {
	_, server := newWorkspaceServer(t)

	plain := postJSON(t, server.URL+"/api/reminders", `{"title":"Plain","remind_at":"2026-03-01T09:00:00Z"}`)
	expectStatus(t, plain, http.StatusCreated)
	raw, _ := io.ReadAll(plain.Body)
	plain.Body.Close()
	if !strings.Contains(string(raw), `"repeat":null`) {
		t.Errorf("a plain reminder must carry repeat:null, got %s", raw)
	}

	for _, repeat := range []string{
		`{"freq":"hourly","interval":1}`,
		`{"freq":"daily","interval":0}`,
		`{"freq":"daily","interval":366}`,
		`{"freq":"daily"}`,
		`{}`,
		`"daily"`,
		`{"freq":7,"interval":1}`,
	} {
		res := postJSON(t, server.URL+"/api/reminders", `{"title":"Bad","remind_at":"2026-03-01T09:00:00Z","repeat":`+repeat+`}`)
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("repeat %s: status = %d, want 400", repeat, res.StatusCode)
		}
		res.Body.Close()
	}
}

func TestReminderPatchSetsKeepsAndClearsRepeat(t *testing.T) {
	st, server := newWorkspaceServer(t)
	ctx := context.Background()
	anchor := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

	reminder := createReminderHTTP(t, server.URL, `{"title":"Water plants","remind_at":"2026-03-01T09:00:00Z","repeat":{"freq":"weekly","interval":2}}`)
	if reminder.Repeat == nil || *reminder.Repeat != (recurrence.Rule{Freq: recurrence.Weekly, Interval: 2}) {
		t.Fatalf("created repeat = %#v", reminder.Repeat)
	}
	url := idURL(server.URL+"/api/reminders", reminder.ID)

	res := patchJSON(t, url, `{"title":"Water plants","remind_at":"2026-03-01T10:00:00Z"}`)
	expectStatus(t, res, http.StatusOK)
	snoozed := decodeInto[store.Reminder](t, res)
	if snoozed.Repeat == nil || !snoozed.RemindAt.Equal(anchor.Add(time.Hour)) {
		t.Fatalf("snooze lost the rule or the time: %#v", snoozed)
	}
	stored, err := st.GetReminder(ctx, reminder.ID)
	if err != nil || !stored.RepeatAnchor.Equal(anchor) {
		t.Errorf("snooze moved the anchor: %v, %v", stored.RepeatAnchor, err)
	}

	res = patchJSON(t, url, `{"title":"Water plants","remind_at":"2026-03-05T08:00:00Z","repeat":{"freq":"daily","interval":1}}`)
	expectStatus(t, res, http.StatusOK)
	res.Body.Close()
	stored, _ = st.GetReminder(ctx, reminder.ID)
	if stored.Repeat.Freq != recurrence.Daily || !stored.RepeatAnchor.Equal(time.Date(2026, 3, 5, 8, 0, 0, 0, time.UTC)) {
		t.Errorf("setting a rule did not re-anchor: %#v anchor %v", stored.Repeat, stored.RepeatAnchor)
	}

	res = patchJSON(t, url, `{"title":"Water plants","remind_at":"2026-03-05T08:00:00Z","repeat":{"freq":"daily","interval":0}}`)
	expectStatus(t, res, http.StatusBadRequest)
	res.Body.Close()

	res = patchJSON(t, url, `{"title":"Water plants","remind_at":"2026-03-05T08:00:00Z","repeat":null}`)
	expectStatus(t, res, http.StatusOK)
	cleared := decodeInto[store.Reminder](t, res)
	if cleared.Repeat != nil {
		t.Errorf("null did not clear the rule: %#v", cleared.Repeat)
	}
}

func TestCompletingRepeatingReminderMovesItForward(t *testing.T) {
	_, server := newWorkspaceServer(t)
	past := time.Now().UTC().Add(-3 * 24 * time.Hour).Truncate(time.Second)

	repeating := createReminderHTTP(t, server.URL, `{"title":"Stand up","remind_at":"`+past.Format(time.RFC3339)+`","repeat":{"freq":"daily","interval":1}}`)
	res := patchJSON(t, idURL(server.URL+"/api/reminders", repeating.ID)+"/completion", `{"completed":true}`)
	expectStatus(t, res, http.StatusOK)
	moved := decodeInto[store.Reminder](t, res)
	if moved.ID != repeating.ID || moved.CompletedAt != nil || moved.Sent || moved.Repeat == nil {
		t.Fatalf("moved = %#v", moved)
	}
	if !moved.RemindAt.After(time.Now()) || moved.RemindAt.Sub(time.Now()) > 24*time.Hour {
		t.Errorf("remind_at = %s, want within the next day", moved.RemindAt)
	}
	if moved.RemindAt.Sub(past)%(24*time.Hour) != 0 {
		t.Errorf("remind_at %s is off the daily series from %s", moved.RemindAt, past)
	}

	plain := createReminderHTTP(t, server.URL, `{"title":"Once","remind_at":"`+past.Format(time.RFC3339)+`"}`)
	res = patchJSON(t, idURL(server.URL+"/api/reminders", plain.ID)+"/completion", `{"completed":true}`)
	expectStatus(t, res, http.StatusOK)
	if done := decodeInto[store.Reminder](t, res); done.CompletedAt == nil {
		t.Error("a plain reminder was not completed")
	}
}

func TestReminderRepeatReachesFutureOccurrenceForEarlyCompletion(t *testing.T) {
	_, server := newWorkspaceServer(t)
	future := time.Now().UTC().Add(5 * time.Hour).Truncate(time.Second)

	reminder := createReminderHTTP(t, server.URL, `{"title":"Early","remind_at":"`+future.Format(time.RFC3339)+`","repeat":{"freq":"daily","interval":1}}`)
	res := patchJSON(t, idURL(server.URL+"/api/reminders", reminder.ID)+"/completion", `{"completed":true}`)
	expectStatus(t, res, http.StatusOK)
	moved := decodeInto[store.Reminder](t, res)
	if want := future.Add(24 * time.Hour); !moved.RemindAt.Equal(want) {
		t.Errorf("remind_at = %s, want %s", moved.RemindAt, want)
	}
}

func today() time.Time {
	return time.Now().UTC().Truncate(24 * time.Hour)
}

func dateString(t time.Time) string {
	return t.Format(recurrence.DateLayout)
}

func TestTaskRepeatRules(t *testing.T) {
	st, server := newWorkspaceServer(t)
	ctx := context.Background()
	todo := strconv.FormatInt(columnID(t, st, "To Do"), 10)
	base := `{"board_id":1,"column_id":` + todo + `,`

	for name, body := range map[string]string{
		"no due date":  base + `"title":"A","repeat":{"freq":"daily","interval":1}}`,
		"an arc":       base + `"title":"A","kind":"arc","due_date":"2026-03-01","repeat":{"freq":"daily","interval":1}}`,
		"bad freq":     base + `"title":"A","due_date":"2026-03-01","repeat":{"freq":"hourly","interval":1}}`,
		"bad interval": base + `"title":"A","due_date":"2026-03-01","repeat":{"freq":"daily","interval":500}}`,
	} {
		res := postJSON(t, server.URL+"/api/tasks", body)
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, res.StatusCode)
		}
		res.Body.Close()
	}

	task := createTaskHTTP(t, server.URL, base+`"title":"Bins","due_date":"2026-03-01","repeat":{"freq":"weekly","interval":1}}`)
	if task.Repeat == nil || task.Repeat.Freq != recurrence.Weekly {
		t.Fatalf("created = %#v", task.Repeat)
	}
	url := idURL(server.URL+"/api/tasks", task.ID)

	res := patchJSON(t, url, `{"title":"Bins out"}`)
	expectStatus(t, res, http.StatusOK)
	if kept := decodeInto[store.Task](t, res); kept.Repeat == nil {
		t.Error("a patch without repeat cleared the rule")
	}

	res = patchJSON(t, url, `{"due_date":"2026-03-10"}`)
	expectStatus(t, res, http.StatusOK)
	res.Body.Close()
	stored, _ := st.GetTask(ctx, task.ID)
	if stored.Repeat == nil || *stored.RepeatAnchor != "2026-03-10" {
		t.Errorf("a due date change did not re-anchor: %#v anchor %v", stored.Repeat, stored.RepeatAnchor)
	}

	res = patchJSON(t, url, `{"repeat":{"freq":"monthly","interval":3}}`)
	expectStatus(t, res, http.StatusOK)
	res.Body.Close()
	stored, _ = st.GetTask(ctx, task.ID)
	if stored.Repeat.Freq != recurrence.Monthly || stored.Repeat.Interval != 3 || *stored.RepeatAnchor != "2026-03-10" {
		t.Errorf("set = %#v anchor %v", stored.Repeat, stored.RepeatAnchor)
	}

	res = patchJSON(t, url+"/type", `{"kind":"arc"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("arc conversion of a repeating task = %d, want 400", res.StatusCode)
	}
	res.Body.Close()

	res = patchJSON(t, url, `{"due_date":null}`)
	expectStatus(t, res, http.StatusOK)
	noDue := decodeInto[store.Task](t, res)
	if noDue.DueDate != nil || noDue.Repeat != nil {
		t.Errorf("clearing the due date left %v / %#v", noDue.DueDate, noDue.Repeat)
	}

	res = patchJSON(t, url, `{"repeat":{"freq":"daily","interval":1}}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("repeat on a task without a due date = %d, want 400", res.StatusCode)
	}
	res.Body.Close()

	res = patchJSON(t, url, `{"due_date":"2026-04-01","repeat":{"freq":"daily","interval":1}}`)
	expectStatus(t, res, http.StatusOK)
	set := decodeInto[store.Task](t, res)
	if set.Repeat == nil {
		t.Error("due date and repeat together did not set the rule")
	}

	res = patchJSON(t, url, `{"repeat":null}`)
	expectStatus(t, res, http.StatusOK)
	if cleared := decodeInto[store.Task](t, res); cleared.Repeat != nil || cleared.DueDate == nil {
		t.Errorf("null repeat: %#v due %v", cleared.Repeat, cleared.DueDate)
	}
}

func moveTaskHTTP(t *testing.T, server string, id, columnID int64) store.Task {
	t.Helper()
	res := postJSON(t, idURL(server+"/api/tasks", id)+"/move", `{"column_id":`+strconv.FormatInt(columnID, 10)+`}`)
	expectStatus(t, res, http.StatusOK)
	return decodeInto[store.Task](t, res)
}

func listTasksHTTP(t *testing.T, server string) []store.Task {
	t.Helper()
	return decodeInto[[]store.Task](t, httpGet(t, server+"/api/tasks?board_id=1"))
}

func tagTask(t *testing.T, st *lite.Store, taskID int64, names ...string) {
	t.Helper()
	for _, name := range names {
		tag, err := st.CreateTag(context.Background(), 1, name, "#111111")
		if err != nil {
			t.Fatal(err)
		}
		if err := st.AddTaskTag(context.Background(), taskID, tag.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMovingRepeatingTaskToDoneCreatesExactlyOneNext(t *testing.T) {
	st, server := newWorkspaceServer(t)
	todoID, doneID := columnID(t, st, "To Do"), columnID(t, st, "Done")
	todo := strconv.FormatInt(todoID, 10)

	overdue := createTaskHTTP(t, server.URL, `{"board_id":1,"column_id":`+todo+`,"title":"Overdue Monday","description_md":"notes","priority":"high","due_date":"2020-01-06","repeat":{"freq":"weekly","interval":1}}`)
	tagTask(t, st, overdue.ID, "a", "b", "c", "d", "e")

	done := moveTaskHTTP(t, server.URL, overdue.ID, doneID)
	if done.CompletedAt == nil || done.Repeat != nil {
		t.Fatalf("completed task = completed %v, repeat %#v", done.CompletedAt, done.Repeat)
	}

	tasks := listTasksHTTP(t, server.URL)
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(tasks))
	}
	var next store.Task
	for _, task := range tasks {
		if task.ID != overdue.ID {
			next = task
		}
	}
	if next.Title != "Overdue Monday" || next.DescriptionMd != "notes" || next.Priority != "high" || next.ColumnID != todoID || next.CompletedAt != nil {
		t.Errorf("next = %#v", next)
	}
	if len(next.Tags) != 5 {
		t.Errorf("next tags = %d, want 5", len(next.Tags))
	}
	if next.Repeat == nil || next.Repeat.Freq != recurrence.Weekly {
		t.Errorf("next repeat = %#v", next.Repeat)
	}
	due, err := time.Parse(recurrence.DateLayout, *next.DueDate)
	if err != nil {
		t.Fatal(err)
	}
	if due.Before(today()) || due.Sub(today()) >= 7*24*time.Hour || due.Weekday() != time.Monday {
		t.Errorf("next due = %s, want the first Monday on or after today", *next.DueDate)
	}

	moveTaskHTTP(t, server.URL, overdue.ID, todoID)
	moveTaskHTTP(t, server.URL, overdue.ID, doneID)
	if tasks := listTasksHTTP(t, server.URL); len(tasks) != 2 {
		t.Errorf("moving the finished task again created more tasks: %d", len(tasks))
	}
}

func TestNextTaskFollowsTheDueDateWhenItIsInTheFuture(t *testing.T) {
	st, server := newWorkspaceServer(t)
	todo := strconv.FormatInt(columnID(t, st, "To Do"), 10)
	due := dateString(today().AddDate(0, 0, 3))

	task := createTaskHTTP(t, server.URL, `{"board_id":1,"column_id":`+todo+`,"title":"Early","due_date":"`+due+`","repeat":{"freq":"weekly","interval":1}}`)
	moveTaskHTTP(t, server.URL, task.ID, columnID(t, st, "Done"))

	want := dateString(today().AddDate(0, 0, 10))
	for _, other := range listTasksHTTP(t, server.URL) {
		if other.ID != task.ID && (other.DueDate == nil || *other.DueDate != want) {
			t.Errorf("next due = %v, want %s", other.DueDate, want)
		}
	}
}

func TestMovingOtherTasksCreatesNothing(t *testing.T) {
	st, server := newWorkspaceServer(t)
	ctx := context.Background()
	todoID, doneID := columnID(t, st, "To Do"), columnID(t, st, "Done")
	due := "2026-03-01"

	plain := createTaskHTTP(t, server.URL, `{"board_id":1,"column_id":`+strconv.FormatInt(todoID, 10)+`,"title":"Plain","due_date":"2026-03-01"}`)
	moveTaskHTTP(t, server.URL, plain.ID, doneID)

	if _, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: doneID, Title: "Born done", Priority: "medium", Position: 1,
		DueDate: &due, Repeat: &store.TaskRepeat{Rule: recurrence.Rule{Freq: recurrence.Daily, Interval: 1}, Anchor: due},
	}); err != nil {
		t.Fatal(err)
	}
	if tasks := listTasksHTTP(t, server.URL); len(tasks) != 2 {
		t.Errorf("tasks = %d, want 2", len(tasks))
	}
}

func TestWorkspaceExportImportKeepsRepeat(t *testing.T) {
	st, server := newWorkspaceServer(t)
	todo := strconv.FormatInt(columnID(t, st, "To Do"), 10)
	createTaskHTTP(t, server.URL, `{"board_id":1,"column_id":`+todo+`,"title":"Bins","due_date":"2026-03-10","repeat":{"freq":"weekly","interval":2}}`)

	res := httpGet(t, server.URL+"/api/workspaces/1/tasks/export")
	expectStatus(t, res, http.StatusOK)
	var exported workspaceTasksExport
	if err := json.NewDecoder(res.Body).Decode(&exported); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if len(exported.Tasks) != 1 || exported.Tasks[0].Repeat == nil || exported.Tasks[0].Repeat.Interval != 2 {
		t.Fatalf("exported = %#v", exported.Tasks)
	}

	res = postJSON(t, server.URL+"/api/workspaces/1/reset", `{"confirm":"I understand"}`)
	expectStatus(t, res, http.StatusOK)
	res.Body.Close()
	body, _ := json.Marshal(exported)
	res = postJSON(t, server.URL+"/api/workspaces/1/tasks/import", string(body))
	expectStatus(t, res, http.StatusOK)
	res.Body.Close()

	tasks := listTasksHTTP(t, server.URL)
	if len(tasks) != 1 || tasks[0].Repeat == nil || *tasks[0].Repeat != (recurrence.Rule{Freq: recurrence.Weekly, Interval: 2}) {
		t.Fatalf("imported = %#v", tasks)
	}
	stored, err := st.GetTask(context.Background(), tasks[0].ID)
	if err != nil || stored.RepeatAnchor == nil || *stored.RepeatAnchor != "2026-03-10" {
		t.Errorf("imported anchor = %v, %v", stored.RepeatAnchor, err)
	}

	bad := `{"version":1,"tasks":[{"title":"No date","kind":"task","repeat":{"freq":"daily","interval":1}}]}`
	res = postJSON(t, server.URL+"/api/workspaces/1/tasks/import", bad)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("import of a repeat without a due date = %d, want 400", res.StatusCode)
	}
	res.Body.Close()
}
