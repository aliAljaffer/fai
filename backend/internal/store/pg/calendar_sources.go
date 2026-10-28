package pg

import (
	"context"
	"fmt"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) ListCalendarSources(ctx context.Context) ([]store.CalendarSource, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, url, color, created_at
		FROM calendar_sources
		ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.CalendarSource{}
	for rows.Next() {
		var c store.CalendarSource
		if err := rows.Scan(&c.ID, &c.Name, &c.URL, &c.Color, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateCalendarSource(ctx context.Context, name, url, color string) (store.CalendarSource, error) {
	var c store.CalendarSource
	err := s.pool.QueryRow(ctx, `
		INSERT INTO calendar_sources (name, url, color)
		VALUES ($1, $2, $3)
		RETURNING id, name, url, color, created_at`, name, url, color).
		Scan(&c.ID, &c.Name, &c.URL, &c.Color, &c.CreatedAt)
	return c, err
}

func (s *Store) UpdateCalendarSourceColor(ctx context.Context, id int64, color string) (store.CalendarSource, error) {
	var c store.CalendarSource
	err := s.pool.QueryRow(ctx, `
		UPDATE calendar_sources SET color = $2 WHERE id = $1
		RETURNING id, name, url, color, created_at`, id, color).
		Scan(&c.ID, &c.Name, &c.URL, &c.Color, &c.CreatedAt)
	return c, err
}

func (s *Store) DeleteCalendarSource(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM calendar_sources WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: calendar source %d", store.ErrNotFound, id)
	}
	return nil
}
