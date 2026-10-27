package pg

import (
	"context"
	"time"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toReminder(r sqlc.Reminder) store.Reminder {
	return store.Reminder{ID: r.ID, Title: r.Title, RemindAt: r.RemindAt, TaskID: int8ToPtr(r.TaskID), Sent: r.Sent, CreatedAt: r.CreatedAt}
}

func (s *Store) ListReminders(ctx context.Context) ([]store.Reminder, error) {
	rows, err := s.q.ListReminders(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.Reminder, len(rows))
	for i, r := range rows {
		out[i] = toReminder(r)
	}
	return out, nil
}

func (s *Store) CreateReminder(ctx context.Context, title string, remindAt time.Time, taskID *int64) (store.Reminder, error) {
	r, err := s.q.CreateReminder(ctx, sqlc.CreateReminderParams{Title: title, RemindAt: remindAt, TaskID: ptrToInt8(taskID)})
	return toReminder(r), err
}

func (s *Store) UpdateReminder(ctx context.Context, id int64, title string, remindAt time.Time, taskID *int64) (store.Reminder, error) {
	r, err := s.q.UpdateReminder(ctx, sqlc.UpdateReminderParams{ID: id, Title: title, RemindAt: remindAt, TaskID: ptrToInt8(taskID)})
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
