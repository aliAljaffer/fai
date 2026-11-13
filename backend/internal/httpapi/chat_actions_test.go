package httpapi

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/ai"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func chatTask(t *testing.T, st store.Store, title string, kind string, parent *int64) store.Task {
	t.Helper()
	task, err := st.CreateTask(context.Background(), store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: title, Priority: "medium", Position: 1024, Kind: kind, ParentTaskID: parent,
	})
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func runChatAction(t *testing.T, server *Server, reply string) (ChatOutcome, string) {
	t.Helper()
	server.AI = &fakeAI{reply: reply}
	outcome, err := server.RunChat(context.Background(), []ai.ChatMessage{{Role: "user", Content: "do it"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending == nil {
		t.Fatalf("pending = nil, reply = %q", outcome.Reply)
	}
	confirmation, err := server.ConfirmChatAction(context.Background(), *outcome.Pending)
	if err != nil {
		t.Fatal(err)
	}
	return outcome, confirmation
}

func TestFaiContextShowsChildrenAndReminders(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	arc := chatTask(t, st, "Homelab", "arc", nil)
	child := chatTask(t, st, "Update Cilium", "task", &arc.ID)
	step := chatTask(t, st, "Read the changelog", "", &child.ID)
	if _, err := st.CreateReminder(ctx, "Call the dentist", time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC), nil, 0, nil); err != nil {
		t.Fatal(err)
	}
	text := (&Server{Store: st}).faiContext(ctx)
	for _, want := range []string{
		"- @" + arc.Ref + " Homelab (arc)",
		"\n  - @" + child.Ref + " Update Cilium",
		"\n    - @" + step.Ref + " Read the changelog",
		"Call the dentist at 2026-11-12T09:00:00Z",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("context missing %q:\n%s", want, text)
		}
	}
}

func TestChatUpdateTaskPatchesOnlyNamedFields(t *testing.T) {
	st := actionTestStore(t)
	task := chatTask(t, st, "Renew the domain", "", nil)
	server := &Server{Store: st}
	outcome, _ := runChatAction(t, server, `ACTION: {"type":"update_task","task":"@`+task.Ref+`","priority":"High","due_date":"2026-11-20"}`)
	if !strings.Contains(outcome.Reply, "priority to high") || !strings.Contains(outcome.Reply, "due date to 2026-11-20") {
		t.Errorf("reply = %q", outcome.Reply)
	}
	updated, _ := st.GetTask(context.Background(), task.ID)
	if updated.Priority != "high" || updated.DueDate == nil || *updated.DueDate != "2026-11-20" || updated.Title != "Renew the domain" {
		t.Fatalf("task = %#v", updated)
	}

	runChatAction(t, server, `ACTION: {"type":"update_task","task":"@`+task.Ref+`","due_date":""}`)
	cleared, _ := st.GetTask(context.Background(), task.ID)
	if cleared.DueDate != nil {
		t.Fatalf("due date = %v, want nil", *cleared.DueDate)
	}
}

func TestChatConfirmRevalidatesClientActions(t *testing.T) {
	st := actionTestStore(t)
	task := chatTask(t, st, "Renew the domain", "", nil)
	priority := "critical"
	_, err := (&Server{Store: st}).ConfirmChatAction(context.Background(), PendingChatAction{
		Type: "batch", Actions: []chatAction{{Type: "update_task", TaskID: task.ID, Priority: &priority}},
	})
	if err == nil {
		t.Fatal("tampered priority was accepted")
	}
}

func TestChatCompleteAndDeleteTask(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	task := chatTask(t, st, "Renew the domain", "", nil)
	step := chatTask(t, st, "Pay the invoice", "", &task.ID)
	server := &Server{Store: st}

	runChatAction(t, server, `ACTION: {"type":"complete_task","task":"@`+step.Ref+`"}`)
	done, _ := st.GetTask(ctx, step.ID)
	if done.ColumnID != columnID(t, st, "Done") || done.CompletedAt == nil {
		t.Fatalf("step = %#v", done)
	}

	runChatAction(t, server, `ACTION: {"type":"delete_task","task":"@`+task.Ref+`"}`)
	if _, err := st.GetTask(ctx, task.ID); err == nil {
		t.Fatal("task was not archived")
	}
}

func TestChatTagTaskCreatesAndRemovesTags(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	task := chatTask(t, st, "Renew the domain", "", nil)
	old, err := st.CreateTag(ctx, 1, "later", "#888888")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddTaskTag(ctx, task.ID, old.ID); err != nil {
		t.Fatal(err)
	}
	runChatAction(t, (&Server{Store: st}), `ACTION: {"type":"tag_task","task":"@`+task.Ref+`","add":["#Home Lab"],"remove":["later"]}`)
	tags, err := st.ListTagsForTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0].Name != "home-lab" {
		t.Fatalf("tags = %#v", tags)
	}
}

func TestChatCommentTask(t *testing.T) {
	st := actionTestStore(t)
	task := chatTask(t, st, "Renew the domain", "", nil)
	runChatAction(t, (&Server{Store: st}), `ACTION: {"type":"comment_task","task":"@`+task.Ref+`","body":"Registrar raised the price."}`)
	comments, err := st.ListTaskComments(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 || comments[0].BodyMd != "Registrar raised the price." {
		t.Fatalf("comments = %#v", comments)
	}
}

func TestChatLogTime(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	task := chatTask(t, st, "Renew the domain", "", nil)
	server := &Server{Store: st}

	outcome, _ := runChatAction(t, server, `ACTION: {"type":"log_time","task":"@`+task.Ref+`","minutes":90}`)
	if !strings.Contains(outcome.Reply, "Log 1h 30m") {
		t.Errorf("reply = %q", outcome.Reply)
	}
	runChatAction(t, server, `ACTION: {"type":"log_time","task":"@`+task.Ref+`","started_at":"2026-11-10T09:00:00+03:00","ended_at":"2026-11-10T09:45:00+03:00"}`)

	entries, err := st.ListEntriesForTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	seconds := map[int32]bool{}
	for _, entry := range entries {
		if entry.DurationSeconds == nil {
			t.Fatalf("entry %d is still running", entry.ID)
		}
		seconds[*entry.DurationSeconds] = true
	}
	if len(entries) != 2 || !seconds[90*60] || !seconds[45*60] {
		t.Fatalf("entries = %#v", entries)
	}

	server.AI = &fakeAI{reply: `ACTION: {"type":"log_time","task":"@` + task.Ref + `","minutes":30,"ended_at":"` + time.Now().Add(2*time.Hour).Format(time.RFC3339) + `"}`}
	if _, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "log it"}}, nil); err == nil {
		t.Fatal("a future time entry was accepted")
	}
}

func TestChatReminderActions(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	reminder, err := st.CreateReminder(ctx, "Call the dentist", time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC), nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: st}
	id := strconv.FormatInt(reminder.ID, 10)

	runChatAction(t, server, `ACTION: {"type":"update_reminder","reminder_id":`+id+`,"remind_at":"2026-11-13T10:00:00Z"}`)
	reminders, _ := st.ListReminders(ctx, nil)
	if len(reminders) != 1 || !reminders[0].RemindAt.Equal(time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)) || reminders[0].Title != "Call the dentist" {
		t.Fatalf("reminders = %#v", reminders)
	}

	runChatAction(t, server, `ACTION: {"type":"complete_reminder","reminder_id":`+id+`}`)
	reminders, _ = st.ListReminders(ctx, nil)
	if reminders[0].CompletedAt == nil {
		t.Fatal("reminder was not completed")
	}

	runChatAction(t, server, `ACTION: {"type":"delete_reminder","reminder_id":`+id+`}`)
	reminders, _ = st.ListReminders(ctx, nil)
	if len(reminders) != 0 {
		t.Fatalf("reminders = %#v", reminders)
	}
}

func TestChatNoteActions(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	server := &Server{Store: st}

	runChatAction(t, server, `ACTION: {"type":"create_note","title":"Homelab","body":"Nodes: 5"}`)
	runChatAction(t, server, `ACTION: {"type":"append_note","note":"homelab","body":"Cilium 1.16"}`)
	note, err := st.GetNoteByTitle(ctx, "Homelab")
	if err != nil {
		t.Fatal(err)
	}
	if note.BodyMd != "Nodes: 5\n\nCilium 1.16" {
		t.Fatalf("body = %q", note.BodyMd)
	}

	server.AI = &fakeAI{reply: `ACTION: {"type":"create_note","title":"Homelab","body":"again"}`}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "new note"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending != nil || !strings.Contains(outcome.Reply, "already exists") {
		t.Fatalf("outcome = %#v", outcome)
	}
}

func TestArcSummaryListsEveryTask(t *testing.T) {
	got := arcSummary("Arc", []chatTaskSpec{{Title: "A", Steps: []string{"x", " ", "y"}}, {Title: "B"}})
	want := "Create arc \"Arc\" with these 2 tasks?\n1. A (2 steps)\n2. B"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestChatTimerActions(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	first := chatTask(t, st, "Renew the domain", "", nil)
	second := chatTask(t, st, "Patch the router", "", nil)
	server := &Server{Store: st}
	running := func() *store.TaskTimeEntry {
		t.Helper()
		entry, err := st.GetRunningEntry(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return entry
	}

	outcome, _ := runChatAction(t, server, `ACTION: {"type":"start_timer","task":"@`+first.Ref+`"}`)
	if !strings.Contains(outcome.Reply, "Start the timer on @"+first.Ref) || strings.Contains(outcome.Reply, "stops") {
		t.Errorf("reply = %q", outcome.Reply)
	}
	if entry := running(); entry == nil || entry.TaskID != first.ID {
		t.Fatalf("running = %#v", entry)
	}

	outcome, _ = runChatAction(t, server, `ACTION: {"type":"start_timer","task":"@`+second.Ref+`"}`)
	if !strings.Contains(outcome.Reply, "stops the timer on @"+first.Ref+" (Renew the domain)") {
		t.Errorf("reply = %q", outcome.Reply)
	}
	if entry := running(); entry == nil || entry.TaskID != second.ID {
		t.Fatalf("running = %#v", entry)
	}

	server.AI = &fakeAI{reply: `ACTION: {"type":"start_timer","task":"@` + second.Ref + `"}`}
	outcome, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "start"}}, nil)
	if err != nil || outcome.Pending != nil || !strings.Contains(outcome.Reply, "already running") {
		t.Fatalf("outcome = %#v, err = %v", outcome, err)
	}

	server.AI = &fakeAI{reply: `ACTION: {"type":"stop_timer","task":"@` + first.Ref + `"}`}
	outcome, err = server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "stop"}}, nil)
	if err != nil || outcome.Pending != nil || !strings.Contains(outcome.Reply, "No timer is running on @"+first.Ref) {
		t.Fatalf("outcome = %#v, err = %v", outcome, err)
	}

	outcome, confirmation := runChatAction(t, server, `ACTION: {"type":"stop_timer","task":"@`+second.Ref+`"}`)
	if !strings.Contains(outcome.Reply, "Stop the timer on @"+second.Ref) || !strings.Contains(confirmation, "Stopped the timer on @"+second.Ref) {
		t.Errorf("reply = %q, confirmation = %q", outcome.Reply, confirmation)
	}
	if running() != nil {
		t.Fatal("timer still running")
	}

	runChatAction(t, server, `ACTION: {"type":"start_timer","task":"@`+first.Ref+`"}`)
	outcome, _ = runChatAction(t, server, `ACTION: {"type":"stop_timer"}`)
	if !strings.Contains(outcome.Reply, "@"+first.Ref) || running() != nil {
		t.Errorf("reply = %q", outcome.Reply)
	}

	server.AI = &fakeAI{reply: `ACTION: {"type":"stop_timer"}`}
	outcome, err = server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "stop"}}, nil)
	if err != nil || outcome.Pending != nil || !strings.Contains(outcome.Reply, "No timer is running.") {
		t.Fatalf("outcome = %#v, err = %v", outcome, err)
	}
}
