package lite

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/recurrence"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func benchStore(b *testing.B) *Store {
	b.Helper()
	st, err := New(filepath.Join(b.TempDir(), "fai.db"))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(st.Close)
	return st
}

func BenchmarkListTasks(b *testing.B) {
	st := benchStore(b)
	for i := range 2000 {
		if _, err := st.CreateTask(b.Context(), store.CreateTaskParams{
			BoardID: 1, ColumnID: 1, Title: "Task", Priority: "medium", Position: float64(i + 1),
		}); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for range b.N {
		tasks, err := st.ListTasks(b.Context(), store.ListTasksParams{})
		if err != nil || len(tasks) != 2000 {
			b.Fatalf("got %d tasks, err %v", len(tasks), err)
		}
	}
}

func BenchmarkListReminders(b *testing.B) {
	st := benchStore(b)
	now := time.Now()
	for i := range 300 {
		if _, err := st.CreateReminder(b.Context(), "Reminder", now.Add(time.Duration(i)*time.Hour), nil, 0, nil); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for range b.N {
		reminders, err := st.ListReminders(b.Context(), nil)
		if err != nil || len(reminders) != 300 {
			b.Fatalf("got %d reminders, err %v", len(reminders), err)
		}
	}
}

func BenchmarkRollRepeatingTask(b *testing.B) {
	st := benchStore(b)
	ctx := b.Context()
	due := "2026-03-01"
	rule := store.TaskRepeat{Rule: recurrence.Rule{Freq: recurrence.Weekly, Interval: 1}, Anchor: due}
	var tagIDs []int64
	for i := range 5 {
		tag, err := st.CreateTag(ctx, "tag"+string(rune('a'+i)), "#111111")
		if err != nil {
			b.Fatal(err)
		}
		tagIDs = append(tagIDs, tag.ID)
	}
	ids := make([]int64, b.N)
	for i := range ids {
		task, err := st.CreateTask(ctx, store.CreateTaskParams{
			BoardID: 1, ColumnID: 1, Title: "Task", Priority: "medium", Position: float64(i + 1), DueDate: &due, Repeat: &rule,
		})
		if err != nil {
			b.Fatal(err)
		}
		for _, tagID := range tagIDs {
			if err := st.AddTaskTag(ctx, task.ID, tagID); err != nil {
				b.Fatal(err)
			}
		}
		ids[i] = task.ID
	}
	b.ResetTimer()
	for i := range b.N {
		if _, err := st.RollRepeatingTask(ctx, store.RollTaskParams{
			TaskID: ids[i], BoardID: 1, ColumnID: 1, Position: 1, DueDate: "2026-03-08",
		}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMoveTask(b *testing.B) {
	st := benchStore(b)
	ctx := b.Context()
	task, err := st.CreateTask(ctx, store.CreateTaskParams{BoardID: 1, ColumnID: 1, Title: "Task", Priority: "medium", Position: 1})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		if _, err := st.MoveTask(ctx, task.ID, 1, 2); err != nil {
			b.Fatal(err)
		}
	}
}
