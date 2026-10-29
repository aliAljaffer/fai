package pg

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) GetAISettings(ctx context.Context) (store.AISettings, error) {
	var out store.AISettings
	err := s.pool.QueryRow(ctx,
		`SELECT provider, base_url, model, api_key, updated_at FROM ai_settings WHERE id = 1`).
		Scan(&out.Provider, &out.BaseURL, &out.Model, &out.APIKey, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.AISettings{}, store.ErrNotFound
	}
	return out, err
}

func (s *Store) SaveAISettings(ctx context.Context, in store.AISettings) (store.AISettings, error) {
	var out store.AISettings
	err := s.pool.QueryRow(ctx,
		`INSERT INTO ai_settings (id, provider, base_url, model, api_key, updated_at)
		 VALUES (1, $1, $2, $3, $4, now())
		 ON CONFLICT (id) DO UPDATE SET
		   provider = EXCLUDED.provider,
		   base_url = EXCLUDED.base_url,
		   model = EXCLUDED.model,
		   api_key = EXCLUDED.api_key,
		   updated_at = now()
		 RETURNING provider, base_url, model, api_key, updated_at`,
		in.Provider, in.BaseURL, in.Model, in.APIKey).
		Scan(&out.Provider, &out.BaseURL, &out.Model, &out.APIKey, &out.UpdatedAt)
	return out, err
}
