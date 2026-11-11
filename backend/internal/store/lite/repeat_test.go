package lite

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/recurrence"
	"github.com/alialjaffer/fai/backend/internal/store"
)

var weekly = recurrence.Rule{Freq: recurrence.Weekly, Interval: 2}

func newRepeatStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	return st
}

func TestReminderRepeatRoundTrip(t *testing.T) {
	st := newRepeatStore(t)
	ctx := t.Context()
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

	created, err := st.CreateReminder(ctx, "Water plants", at, nil, 0, &store.ReminderRepeat{Rule: weekly, Anchor: at})
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.GetReminder(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Repeat == nil || *got.Repeat != weekly || got.RepeatAnchor == nil || !got.RepeatAnchor.Equal(at) {
		t.Fatalf("reminder = %#v", got)
	}

	snoozed := at.Add(time.Hour)
	updated, err := st.UpdateReminder(ctx, created.ID, "Water plants", snoozed, nil, got.RepeatState())
	if err != nil {
		t.Fatal(err)
	}
	if !updated.RepeatAnchor.Equal(at) || !updated.RemindAt.Equal(snoozed) {
		t.Errorf("update moved the anchor: %#v", updated)
	}

	if _, err := st.SetReminderCompleted(ctx, created.ID, true); err != nil {
		t.Fatal(err)
	}
	next := at.AddDate(0, 0, 14)
	advanced, err := st.AdvanceReminder(ctx, created.ID, next)
	if err != nil {
		t.Fatal(err)
	}
	if !advanced.RemindAt.Equal(next) || advanced.Sent || advanced.CompletedAt != nil || advanced.Repeat == nil {
		t.Errorf("advanced = %#v", advanced)
	}

	cleared, err := st.UpdateReminder(ctx, created.ID, "Water plants", next, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Repeat != nil || cleared.RepeatAnchor != nil {
		t.Errorf("repeat survived a clear: %#v", cleared)
	}
}

func TestTaskRepeatRoundTripAndPatch(t *testing.T) {
	st := newRepeatStore(t)
	ctx := t.Context()
	due := "2026-03-01"

	task, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: "Take out bins", Priority: "medium", Position: 1,
		DueDate: &due, Repeat: &store.TaskRepeat{Rule: weekly, Anchor: due},
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Repeat == nil || *task.Repeat != weekly || task.RepeatAnchor == nil || *task.RepeatAnchor != due {
		t.Fatalf("created = %#v", task)
	}

	title := "Take out the bins"
	patched, err := st.PatchTask(ctx, task.ID, store.TaskPatch{Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	if patched.Repeat == nil || *patched.RepeatAnchor != due {
		t.Errorf("a patch without repeat changed it: %#v", patched)
	}

	newDue := "2026-03-09"
	patched, err = st.PatchTask(ctx, task.ID, store.TaskPatch{
		DueDate: &newDue, DueDateSet: true, RepeatSet: true, Repeat: &store.TaskRepeat{Rule: weekly, Anchor: newDue},
	})
	if err != nil {
		t.Fatal(err)
	}
	if *patched.RepeatAnchor != newDue {
		t.Errorf("anchor = %q, want %q", *patched.RepeatAnchor, newDue)
	}

	patched, err = st.PatchTask(ctx, task.ID, store.TaskPatch{RepeatSet: true})
	if err != nil {
		t.Fatal(err)
	}
	if patched.Repeat != nil || patched.RepeatAnchor != nil {
		t.Errorf("repeat survived a clear: %#v", patched)
	}

	found, err := st.SearchTasks(ctx, "bins", nil)
	if err != nil || len(found) != 1 {
		t.Fatalf("search = %v, %v", found, err)
	}
}

func TestRollRepeatingTaskCopiesTagsAndClearsTheRule(t *testing.T) {
	st := newRepeatStore(t)
	ctx := t.Context()
	due := "2026-03-01"
	task, err := st.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: "Pay rent", DescriptionMd: "bank app", Priority: "high", Position: 1,
		DueDate: &due, Repeat: &store.TaskRepeat{Rule: recurrence.Rule{Freq: recurrence.Monthly, Interval: 1}, Anchor: due},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"home", "money"} {
		tag, err := st.CreateTag(ctx, name, "#111111")
		if err != nil {
			t.Fatal(err)
		}
		if err := st.AddTaskTag(ctx, task.ID, tag.ID); err != nil {
			t.Fatal(err)
		}
	}

	next, err := st.RollRepeatingTask(ctx, store.RollTaskParams{
		TaskID: task.ID, BoardID: task.BoardID, ColumnID: 1, Position: 2048, DueDate: "2026-04-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if next.ID == task.ID || next.RefNum != task.RefNum+1 || next.Title != "Pay rent" || next.DescriptionMd != "bank app" || next.Priority != "high" {
		t.Errorf("next = %#v", next)
	}
	if next.DueDate == nil || *next.DueDate != "2026-04-01" || next.Position != 2048 || next.CompletedAt != nil {
		t.Errorf("next due/position = %#v", next)
	}
	if next.Repeat == nil || next.Repeat.Freq != recurrence.Monthly || *next.RepeatAnchor != due {
		t.Errorf("next repeat = %#v anchor %v", next.Repeat, next.RepeatAnchor)
	}
	if len(next.Tags) != 2 {
		t.Errorf("next tags = %#v", next.Tags)
	}

	original, err := st.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if original.Repeat != nil || original.RepeatAnchor != nil || len(original.Tags) != 2 {
		t.Errorf("original = %#v", original)
	}

	if _, err := st.RollRepeatingTask(ctx, store.RollTaskParams{
		TaskID: task.ID, BoardID: task.BoardID, ColumnID: 1, Position: 3072, DueDate: "2026-05-01",
	}); err != store.ErrNotFound {
		t.Errorf("second roll error = %v, want ErrNotFound", err)
	}
	tasks, err := st.ListTasks(ctx, store.ListTasksParams{})
	if err != nil || len(tasks) != 2 {
		t.Errorf("tasks = %d, %v", len(tasks), err)
	}
}

func TestBackupCarriesRepeatAndAcceptsOldBackups(t *testing.T) {
	source := newRepeatStore(t)
	ctx := t.Context()
	due := "2026-03-01"
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	if _, err := source.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: "Repeats", Priority: "medium", Position: 1,
		DueDate: &due, Repeat: &store.TaskRepeat{Rule: weekly, Anchor: due},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := source.CreateReminder(ctx, "Repeats", at, nil, 0, &store.ReminderRepeat{Rule: weekly, Anchor: at}); err != nil {
		t.Fatal(err)
	}
	dump, err := source.DumpBackup(ctx)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(dump)
	if err != nil {
		t.Fatal(err)
	}

	var current store.BackupData
	if err := json.Unmarshal(raw, &current); err != nil {
		t.Fatal(err)
	}
	target := newRepeatStore(t)
	if err := target.RestoreBackup(ctx, current); err != nil {
		t.Fatal(err)
	}
	tasks, _ := target.ListTasks(ctx, store.ListTasksParams{})
	reminders, _ := target.ListReminders(ctx, nil)
	if len(tasks) != 1 || tasks[0].Repeat == nil || *tasks[0].RepeatAnchor != due {
		t.Errorf("restored tasks = %#v", tasks)
	}
	if len(reminders) != 1 || reminders[0].Repeat == nil || !reminders[0].RepeatAnchor.Equal(at) {
		t.Errorf("restored reminders = %#v", reminders)
	}

	var old map[string]json.RawMessage
	if err := json.Unmarshal(raw, &old); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"tasks", "reminders"} {
		var items []map[string]any
		if err := json.Unmarshal(old[key], &items); err != nil {
			t.Fatal(err)
		}
		for _, item := range items {
			delete(item, "repeat")
			delete(item, "repeat_anchor")
		}
		if old[key], err = json.Marshal(items); err != nil {
			t.Fatal(err)
		}
	}
	oldRaw, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	var legacy store.BackupData
	if err := json.Unmarshal(oldRaw, &legacy); err != nil {
		t.Fatal(err)
	}
	if err := target.RestoreBackup(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	tasks, _ = target.ListTasks(ctx, store.ListTasksParams{})
	reminders, _ = target.ListReminders(ctx, nil)
	if len(tasks) != 1 || tasks[0].Repeat != nil || len(reminders) != 1 || reminders[0].Repeat != nil {
		t.Errorf("old backup restored with repeat: %#v %#v", tasks, reminders)
	}
}

func TestUpgradeAddsRepeatColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fai.db")
	st, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"tasks", "reminders"} {
		for _, column := range []string{"repeat_freq", "repeat_interval", "repeat_anchor"} {
			if _, err := db.Exec("ALTER TABLE " + table + " DROP COLUMN " + column); err != nil {
				t.Fatal(err)
			}
		}
	}
	db.Close()

	upgraded, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	reminder, err := upgraded.CreateReminder(t.Context(), "After upgrade", at, nil, 0, &store.ReminderRepeat{Rule: weekly, Anchor: at})
	if err != nil || reminder.Repeat == nil {
		t.Fatalf("reminder = %#v, err = %v", reminder, err)
	}
	if _, err := upgraded.CreateReminder(t.Context(), "Bad", at, nil, 0, &store.ReminderRepeat{Rule: recurrence.Rule{Freq: "hourly", Interval: 1}, Anchor: at}); err == nil {
		t.Error("the upgraded schema accepted an unknown frequency")
	}
}
