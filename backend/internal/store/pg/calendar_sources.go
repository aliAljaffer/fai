package pg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) ListCalendarSources(ctx context.Context, workspaceID *int64) ([]store.CalendarSource, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, url, color, workspace_id, created_at
		FROM calendar_sources
		WHERE $1::bigint IS NULL OR workspace_id = $1
		ORDER BY name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.CalendarSource{}
	for rows.Next() {
		var c store.CalendarSource
		if err := rows.Scan(&c.ID, &c.Name, &c.URL, &c.Color, &c.WorkspaceID, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateCalendarSource(ctx context.Context, name, url, color string, workspaceID int64) (store.CalendarSource, error) {
	var c store.CalendarSource
	err := s.pool.QueryRow(ctx, `
		INSERT INTO calendar_sources (name, url, color, workspace_id)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4::bigint, 0), (SELECT id FROM workspaces ORDER BY id LIMIT 1)))
		RETURNING id, name, url, color, workspace_id, created_at`, name, url, color, workspaceID).
		Scan(&c.ID, &c.Name, &c.URL, &c.Color, &c.WorkspaceID, &c.CreatedAt)
	return c, err
}

func (s *Store) UpdateCalendarSourceColor(ctx context.Context, id int64, color string) (store.CalendarSource, error) {
	var c store.CalendarSource
	err := s.pool.QueryRow(ctx, `
		UPDATE calendar_sources SET color = $2 WHERE id = $1
		RETURNING id, name, url, color, workspace_id, created_at`, id, color).
		Scan(&c.ID, &c.Name, &c.URL, &c.Color, &c.WorkspaceID, &c.CreatedAt)
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

func (s *Store) GetCalendarFeed(ctx context.Context, sourceID int64) ([]byte, time.Time, bool, error) {
	var data string
	var fetchedAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT data, fetched_at FROM calendar_feed_cache WHERE source_id = $1`, sourceID).
		Scan(&data, &fetchedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, time.Time{}, false, nil
	}
	if err != nil {
		return nil, time.Time{}, false, err
	}
	return []byte(data), fetchedAt, true, nil
}

func (s *Store) SetCalendarFeed(ctx context.Context, sourceID int64, data []byte, fetchedAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO calendar_feed_cache (source_id, data, fetched_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (source_id) DO UPDATE SET data = EXCLUDED.data, fetched_at = EXCLUDED.fetched_at`,
		sourceID, string(data), fetchedAt)
	return err
}
