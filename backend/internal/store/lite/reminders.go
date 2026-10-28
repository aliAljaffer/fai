package lite

import (
	"context"
	"time"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toReminder(r sqlitedb.Reminder) store.Reminder {
	remindAt, _ := parseTime(r.RemindAt)
	createdAt, _ := parseTime(r.CreatedAt)
	return store.Reminder{ID: r.ID, Title: r.Title, RemindAt: remindAt, TaskID: nullInt64ToPtr(r.TaskID), WorkspaceID: r.WorkspaceID, Sent: r.Sent, CreatedAt: createdAt}
}

func (s *Store) ListReminders(ctx context.Context, workspaceID *int64) ([]store.Reminder, error) {
	rows, err := s.q.ListReminders(ctx, ptrToNullInt64(workspaceID))
	if err != nil {
		return nil, err
	}
	out := make([]store.Reminder, len(rows))
	for i, r := range rows {
		out[i] = toReminder(r)
	}
	return out, nil
}

func (s *Store) CreateReminder(ctx context.Context, title string, remindAt time.Time, taskID *int64, workspaceID int64) (store.Reminder, error) {
	r, err := s.q.CreateReminder(ctx, sqlitedb.CreateReminderParams{Title: title, RemindAt: formatTime(remindAt), TaskID: ptrToNullInt64(taskID), WorkspaceID: workspaceID})
	return toReminder(r), err
}

func (s *Store) UpdateReminder(ctx context.Context, id int64, title string, remindAt time.Time, taskID *int64) (store.Reminder, error) {
	r, err := s.q.UpdateReminder(ctx, sqlitedb.UpdateReminderParams{ID: id, Title: title, RemindAt: formatTime(remindAt), TaskID: ptrToNullInt64(taskID)})
	return toReminder(r), err
}

func (s *Store) DeleteReminder(ctx context.Context, id int64) error {
	return s.q.DeleteReminder(ctx, id)
}

func (s *Store) ListDueUnsentReminders(ctx context.Context) ([]store.Reminder, error) {
	rows, err := s.q.ListDueUnsentReminders(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.Reminder, len(rows))
	for i, r := range rows {
		out[i] = toReminder(r)
	}
	return out, nil
}

func (s *Store) MarkReminderSent(ctx context.Context, id int64) error {
	return s.q.MarkReminderSent(ctx, id)
}
