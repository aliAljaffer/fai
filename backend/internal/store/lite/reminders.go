package lite

import (
	"context"
	"database/sql"
	"time"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toReminder(r sqlitedb.Reminder) store.Reminder {
	remindAt, _ := parseTime(r.RemindAt)
	createdAt, _ := parseTime(r.CreatedAt)
	return store.Reminder{ID: r.ID, Title: r.Title, RemindAt: remindAt, TaskID: nullInt64ToPtr(r.TaskID), WorkspaceID: r.WorkspaceID, Sent: r.Sent, CreatedAt: createdAt, CompletedAt: nullTimeToPtr(r.CompletedAt),
		Repeat: repeatRule(r.RepeatFreq, r.RepeatInterval), RepeatAnchor: nullTimeToPtr(r.RepeatAnchor)}
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

func (s *Store) GetReminder(ctx context.Context, id int64) (store.Reminder, error) {
	r, err := s.q.GetReminder(ctx, id)
	return toReminder(r), err
}

func reminderRepeatColumns(repeat *store.ReminderRepeat) (sql.NullString, sql.NullInt64, sql.NullString) {
	if repeat == nil {
		return sql.NullString{}, sql.NullInt64{}, sql.NullString{}
	}
	freq, interval := repeatColumns(&repeat.Rule)
	return freq, interval, ptrTimeToNull(&repeat.Anchor)
}

func (s *Store) CreateReminder(ctx context.Context, title string, remindAt time.Time, taskID *int64, workspaceID int64, repeat *store.ReminderRepeat) (store.Reminder, error) {
	freq, interval, anchor := reminderRepeatColumns(repeat)
	r, err := s.q.CreateReminder(ctx, sqlitedb.CreateReminderParams{
		Title: title, RemindAt: formatTime(remindAt), TaskID: ptrToNullInt64(taskID), WorkspaceID: workspaceID,
		RepeatFreq: freq, RepeatInterval: interval, RepeatAnchor: anchor,
	})
	return toReminder(r), err
}

func (s *Store) UpdateReminder(ctx context.Context, id int64, title string, remindAt time.Time, taskID *int64, repeat *store.ReminderRepeat) (store.Reminder, error) {
	freq, interval, anchor := reminderRepeatColumns(repeat)
	r, err := s.q.UpdateReminder(ctx, sqlitedb.UpdateReminderParams{
		ID: id, Title: title, RemindAt: formatTime(remindAt), TaskID: ptrToNullInt64(taskID),
		RepeatFreq: freq, RepeatInterval: interval, RepeatAnchor: anchor,
	})
	return toReminder(r), err
}

func (s *Store) AdvanceReminder(ctx context.Context, id int64, remindAt time.Time) (store.Reminder, error) {
	r, err := s.q.AdvanceReminder(ctx, sqlitedb.AdvanceReminderParams{ID: id, RemindAt: formatTime(remindAt)})
	return toReminder(r), err
}

func (s *Store) SetReminderCompleted(ctx context.Context, id int64, completed bool) (store.Reminder, error) {
	if completed {
		r, err := s.q.CompleteReminder(ctx, id)
		return toReminder(r), err
	}
	r, err := s.q.ReopenReminder(ctx, id)
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
