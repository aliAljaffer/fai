package lite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) ListCalendarSources(ctx context.Context, workspaceID *int64) ([]store.CalendarSource, error) {
	scope := ptrToNullInt64(workspaceID)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, url, color, workspace_id, created_at
		FROM calendar_sources
		WHERE CAST(? AS INTEGER) IS NULL OR workspace_id = ?
		ORDER BY name`, scope, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.CalendarSource{}
	for rows.Next() {
		var c store.CalendarSource
		var createdAt string
		if err := rows.Scan(&c.ID, &c.Name, &c.URL, &c.Color, &c.WorkspaceID, &createdAt); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = parseTime(createdAt)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateCalendarSource(ctx context.Context, name, url, color string, workspaceID int64) (store.CalendarSource, error) {
	var c store.CalendarSource
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO calendar_sources (name, url, color, workspace_id)
		VALUES (?, ?, ?, COALESCE(NULLIF(CAST(? AS INTEGER), 0), (SELECT id FROM workspaces ORDER BY id LIMIT 1)))
		RETURNING id, name, url, color, workspace_id, created_at`, name, url, color, workspaceID).
		Scan(&c.ID, &c.Name, &c.URL, &c.Color, &c.WorkspaceID, &createdAt)
	if err != nil {
		return store.CalendarSource{}, err
	}
	c.CreatedAt, _ = parseTime(createdAt)
	return c, nil
}

func (s *Store) UpdateCalendarSourceColor(ctx context.Context, id int64, color string) (store.CalendarSource, error) {
	var c store.CalendarSource
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
		UPDATE calendar_sources SET color = ? WHERE id = ?
		RETURNING id, name, url, color, workspace_id, created_at`, color, id).
		Scan(&c.ID, &c.Name, &c.URL, &c.Color, &c.WorkspaceID, &createdAt)
	if err != nil {
		return store.CalendarSource{}, err
	}
	c.CreatedAt, _ = parseTime(createdAt)
	return c, nil
}

func (s *Store) DeleteCalendarSource(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM calendar_sources WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: calendar source %d", store.ErrNotFound, id)
	}
	return nil
}

func (s *Store) GetCalendarFeed(ctx context.Context, sourceID int64) ([]byte, time.Time, bool, error) {
	var data, fetchedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT data, fetched_at FROM calendar_feed_cache WHERE source_id = ?`, sourceID).
		Scan(&data, &fetchedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, time.Time{}, false, nil
	}
	if err != nil {
		return nil, time.Time{}, false, err
	}
	at, err := parseTime(fetchedAt)
	if err != nil {
		return nil, time.Time{}, false, err
	}
	return []byte(data), at, true, nil
}

func (s *Store) SetCalendarFeed(ctx context.Context, sourceID int64, data []byte, fetchedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO calendar_feed_cache (source_id, data, fetched_at)
		VALUES (?, ?, ?)
		ON CONFLICT (source_id) DO UPDATE SET data = excluded.data, fetched_at = excluded.fetched_at`,
		sourceID, string(data), formatTime(fetchedAt))
	return err
}
