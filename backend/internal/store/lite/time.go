package lite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toEntry(e sqlitedb.TaskTimeEntry) store.TaskTimeEntry {
	startedAt, _ := parseTime(e.StartedAt)
	var duration *int32
	if e.DurationSeconds.Valid {
		d := int32(e.DurationSeconds.Int64)
		duration = &d
	}
	return store.TaskTimeEntry{
		ID: e.ID, TaskID: e.TaskID, StartedAt: startedAt,
		EndedAt: nullTimeToPtr(e.EndedAt), DurationSeconds: duration,
	}
}

func (s *Store) StartTimer(ctx context.Context, taskID int64) (store.TaskTimeEntry, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.TaskTimeEntry{}, err
	}
	defer tx.Rollback()

	q := s.q.WithTx(tx)
	if err := q.StopRunningEntry(ctx); err != nil {
		return store.TaskTimeEntry{}, err
	}
	entry, err := q.StartEntry(ctx, taskID)
	if err != nil {
		return store.TaskTimeEntry{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.TaskTimeEntry{}, err
	}
	return toEntry(entry), nil
}

func (s *Store) StopEntryForTask(ctx context.Context, taskID int64) (store.TaskTimeEntry, error) {
	e, err := s.q.StopEntryForTask(ctx, taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return store.TaskTimeEntry{}, store.ErrNotFound
	}
	return toEntry(e), err
}

func (s *Store) GetRunningEntry(ctx context.Context) (*store.TaskTimeEntry, error) {
	e, err := s.q.GetRunningEntry(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	entry := toEntry(e)
	return &entry, nil
}

func (s *Store) ListEntriesForTask(ctx context.Context, taskID int64) ([]store.TaskTimeEntry, error) {
	rows, err := s.q.ListEntriesForTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]store.TaskTimeEntry, len(rows))
	for i, e := range rows {
		out[i] = toEntry(e)
	}
	return out, nil
}
