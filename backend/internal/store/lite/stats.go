package lite

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) CompletionsByDay(ctx context.Context, from, to string) ([]store.DayCount, error) {
	rows, err := s.q.CompletionsByDay(ctx, sqlitedb.CompletionsByDayParams{FromDate: from, ToDate: to})
	if err != nil {
		return nil, err
	}
	out := make([]store.DayCount, len(rows))
	for i, r := range rows {
		out[i] = store.DayCount{Date: r.Day, Count: r.Count}
	}
	return out, nil
}

func (s *Store) TimeByDay(ctx context.Context, from, to string) ([]store.DaySeconds, error) {
	rows, err := s.q.TimeByDay(ctx, sqlitedb.TimeByDayParams{FromDate: from, ToDate: to})
	if err != nil {
		return nil, err
	}
	out := make([]store.DaySeconds, len(rows))
	for i, r := range rows {
		out[i] = store.DaySeconds{Date: r.Day, Seconds: r.Seconds}
	}
	return out, nil
}
