package httpapi

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/ai"
	"github.com/alialjaffer/fai/backend/internal/recurrence"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func chatQuestion(t *testing.T, server *Server, reply string) ChatOutcome {
	t.Helper()
	server.AI = &fakeAI{reply: reply}
	outcome, err := server.RunChat(context.Background(), []ai.ChatMessage{{Role: "user", Content: "do it"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Pending != nil {
		t.Fatalf("expected a question, got pending %q", outcome.Pending.Summary)
	}
	return outcome
}

func TestChatCreatesRepeatingReminder(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	server := &Server{Store: st}

	outcome, _ := runChatAction(t, server, `ACTION: {"type":"create_reminder","title":"Water plants","remind_at":"2026-11-12T09:00:00Z","repeat":{"freq":"weekly","interval":2}}`)
	if !strings.Contains(outcome.Reply, "repeats every 2 weeks") {
		t.Errorf("reply = %q", outcome.Reply)
	}
	reminders, _ := st.ListReminders(ctx, nil)
	if len(reminders) != 1 || reminders[0].Repeat == nil || reminders[0].Repeat.Interval != 2 {
		t.Fatalf("reminders = %#v", reminders)
	}
	if want := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC); !reminders[0].RepeatAnchor.Equal(want) {
		t.Errorf("anchor = %v, want %v", reminders[0].RepeatAnchor, want)
	}

	server.AI = &fakeAI{reply: `ACTION: {"type":"create_reminder","title":"Bad","remind_at":"2026-11-12T09:00:00Z","repeat":{"freq":"hourly","interval":1}}`}
	if _, err := server.RunChat(ctx, []ai.ChatMessage{{Role: "user", Content: "do it"}}, nil); err == nil {
		t.Error("an invalid repeat was accepted")
	}
}

func TestChatUpdateReminderRepeat(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	anchor := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	reminder, err := st.CreateReminder(ctx, "Water plants", anchor, nil, 0,
		&store.ReminderRepeat{Rule: recurrence.Rule{Freq: recurrence.Weekly, Interval: 1}, Anchor: anchor})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: st}
	id := strconv.FormatInt(reminder.ID, 10)

	runChatAction(t, server, `ACTION: {"type":"update_reminder","reminder_id":`+id+`,"remind_at":"2026-11-14T18:00:00Z"}`)
	stored, _ := st.GetReminder(ctx, reminder.ID)
	if want := time.Date(2026, 11, 14, 18, 0, 0, 0, time.UTC); stored.Repeat == nil || !stored.RepeatAnchor.Equal(want) {
		t.Errorf("a chat reschedule did not re-anchor: %#v anchor %v", stored.Repeat, stored.RepeatAnchor)
	}

	outcome, _ := runChatAction(t, server, `ACTION: {"type":"update_reminder","reminder_id":`+id+`,"repeat":{"freq":"daily","interval":3}}`)
	if !strings.Contains(outcome.Reply, "repeat every 3 days") {
		t.Errorf("reply = %q", outcome.Reply)
	}
	stored, _ = st.GetReminder(ctx, reminder.ID)
	if stored.Repeat.Freq != recurrence.Daily || stored.Repeat.Interval != 3 {
		t.Errorf("repeat = %#v", stored.Repeat)
	}

	outcome, _ = runChatAction(t, server, `ACTION: {"type":"update_reminder","reminder_id":`+id+`,"repeat":null}`)
	if !strings.Contains(outcome.Reply, "no repeat") {
		t.Errorf("reply = %q", outcome.Reply)
	}
	stored, _ = st.GetReminder(ctx, reminder.ID)
	if stored.Repeat != nil {
		t.Errorf("repeat survived null: %#v", stored.Repeat)
	}
}

func TestChatCompleteRepeatingReminderNamesNextTime(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	anchor := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Minute)
	reminder, err := st.CreateReminder(ctx, "Stand up", anchor, nil, 0,
		&store.ReminderRepeat{Rule: recurrence.Rule{Freq: recurrence.Daily, Interval: 1}, Anchor: anchor})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: st}

	outcome, confirmation := runChatAction(t, server, `ACTION: {"type":"complete_reminder","reminder_id":`+strconv.FormatInt(reminder.ID, 10)+`}`)
	stored, _ := st.GetReminder(ctx, reminder.ID)
	if stored.CompletedAt != nil || !stored.RemindAt.After(time.Now()) {
		t.Fatalf("stored = %#v", stored)
	}
	next := chatNextTime(stored.RemindAt)
	for name, text := range map[string]string{"summary": outcome.Reply, "confirmation": confirmation} {
		if !strings.Contains(text, "repeats") || !strings.Contains(text, next) {
			t.Errorf("%s = %q, want it to say it repeats and name %s", name, text, next)
		}
	}
}

func TestChatCreatesAndUpdatesRepeatingTasks(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	server := &Server{Store: st}

	outcome, _ := runChatAction(t, server, `ACTION: {"type":"create_task","title":"Take out bins","due_date":"2026-11-16","repeat":{"freq":"weekly","interval":1}}`)
	if !strings.Contains(outcome.Reply, "repeats every week") {
		t.Errorf("reply = %q", outcome.Reply)
	}
	tasks, _ := st.ListTasks(ctx, store.ListTasksParams{})
	if len(tasks) != 1 || tasks[0].Repeat == nil || *tasks[0].RepeatAnchor != "2026-11-16" {
		t.Fatalf("tasks = %#v", tasks)
	}
	ref := "@" + tasks[0].Ref

	question := chatQuestion(t, server, `ACTION: {"type":"create_task","title":"No date","repeat":{"freq":"daily","interval":1}}`)
	if !strings.Contains(question.Reply, "due date") {
		t.Errorf("reply = %q", question.Reply)
	}
	question = chatQuestion(t, server, `ACTION: {"type":"create_arc","title":"Arc","repeat":{"freq":"daily","interval":1},"tasks":[{"title":"One"}]}`)
	if !strings.Contains(question.Reply, "arc cannot repeat") {
		t.Errorf("reply = %q", question.Reply)
	}

	outcome, _ = runChatAction(t, server, `ACTION: {"type":"update_task","task":"`+ref+`","repeat":{"freq":"monthly","interval":2}}`)
	if !strings.Contains(outcome.Reply, "repeat every 2 months") {
		t.Errorf("reply = %q", outcome.Reply)
	}
	stored, _ := st.GetTask(ctx, tasks[0].ID)
	if stored.Repeat.Freq != recurrence.Monthly || *stored.RepeatAnchor != "2026-11-16" {
		t.Errorf("stored = %#v anchor %v", stored.Repeat, stored.RepeatAnchor)
	}

	runChatAction(t, server, `ACTION: {"type":"update_task","task":"`+ref+`","due_date":"2026-11-20"}`)
	stored, _ = st.GetTask(ctx, tasks[0].ID)
	if stored.Repeat == nil || *stored.RepeatAnchor != "2026-11-20" {
		t.Errorf("a due date change did not re-anchor: %#v anchor %v", stored.Repeat, stored.RepeatAnchor)
	}

	outcome, _ = runChatAction(t, server, `ACTION: {"type":"update_task","task":"`+ref+`","due_date":""}`)
	if !strings.Contains(outcome.Reply, "no repeat") {
		t.Errorf("reply = %q", outcome.Reply)
	}
	stored, _ = st.GetTask(ctx, tasks[0].ID)
	if stored.Repeat != nil || stored.DueDate != nil {
		t.Errorf("clearing the due date left %#v / %v", stored.Repeat, stored.DueDate)
	}

	question = chatQuestion(t, server, `ACTION: {"type":"update_task","task":"`+ref+`","repeat":{"freq":"daily","interval":1}}`)
	if !strings.Contains(question.Reply, "due date") {
		t.Errorf("reply = %q", question.Reply)
	}
}

func TestChatCompletingRepeatingTaskCreatesNext(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	due := dateString(today().AddDate(0, 0, 2))
	task, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: columnID(t, st, "To Do"), Title: "Pay rent", Priority: "medium", Position: 1024,
		DueDate: &due, Repeat: &store.TaskRepeat{Rule: recurrence.Rule{Freq: recurrence.Monthly, Interval: 1}, Anchor: due},
	})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: st}

	runChatAction(t, server, `ACTION: {"type":"complete_task","task":"@`+task.Ref+`"}`)
	tasks, _ := st.ListTasks(ctx, store.ListTasksParams{})
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(tasks))
	}
	for _, other := range tasks {
		switch other.ID {
		case task.ID:
			if other.CompletedAt == nil || other.Repeat != nil {
				t.Errorf("completed task = %#v", other)
			}
		default:
			if other.Repeat == nil || other.CompletedAt != nil || other.ColumnID != task.ColumnID {
				t.Errorf("next task = %#v", other)
			}
		}
	}
}

func TestFaiContextShowsRepeats(t *testing.T) {
	st := actionTestStore(t)
	ctx := context.Background()
	due := "2026-11-16"
	rule := recurrence.Rule{Freq: recurrence.Weekly, Interval: 2}
	if _, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: "Take out bins", Priority: "medium", Position: 1024,
		DueDate: &due, Repeat: &store.TaskRepeat{Rule: rule, Anchor: due},
	}); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	if _, err := st.CreateReminder(ctx, "Water plants", at, nil, 0, &store.ReminderRepeat{Rule: rule, Anchor: at}); err != nil {
		t.Fatal(err)
	}
	text := (&Server{Store: st}).faiContext(ctx)
	for _, want := range []string{"due 2026-11-16, repeats every 2 weeks", "Water plants at 2026-11-12T09:00:00Z, repeats every 2 weeks"} {
		if !strings.Contains(text, want) {
			t.Errorf("context missing %q:\n%s", want, text)
		}
	}
}
