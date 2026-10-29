package lite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) GetAISettings(ctx context.Context) (store.AISettings, error) {
	var out store.AISettings
	var updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT provider, base_url, model, api_key, updated_at FROM ai_settings WHERE id = 1`).
		Scan(&out.Provider, &out.BaseURL, &out.Model, &out.APIKey, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return store.AISettings{}, store.ErrNotFound
	}
	if err != nil {
		return store.AISettings{}, err
	}
	out.UpdatedAt, _ = parseTime(updatedAt)
	return out, nil
}

func (s *Store) SaveAISettings(ctx context.Context, in store.AISettings) (store.AISettings, error) {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ai_settings (id, provider, base_url, model, api_key)
		 VALUES (1, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET
		   provider = excluded.provider,
		   base_url = excluded.base_url,
		   model = excluded.model,
		   api_key = excluded.api_key,
		   updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`,
		in.Provider, in.BaseURL, in.Model, in.APIKey)
	if err != nil {
		return store.AISettings{}, err
	}
	return s.GetAISettings(ctx)
}
