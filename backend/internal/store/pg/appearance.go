package pg

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) GetAppearance(ctx context.Context) (store.AppearanceSettings, error) {
	out := store.DefaultAppearance()
	var customPalettes string
	err := s.pool.QueryRow(ctx,
		`SELECT global_palette_id, custom_palettes, updated_at FROM appearance_settings WHERE id = 1`).
		Scan(&out.GlobalPaletteID, &customPalettes, &out.UpdatedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if err == nil && customPalettes != "" && customPalettes != "null" {
		if err := json.Unmarshal([]byte(customPalettes), &out.CustomPalettes); err != nil {
			return out, err
		}
	}
	if out.CustomPalettes == nil {
		out.CustomPalettes = []store.CustomPalette{}
	}

	rows, err := s.pool.Query(ctx, `SELECT workspace_id, palette_id FROM workspace_themes`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var workspaceID int64
		var paletteID string
		if err := rows.Scan(&workspaceID, &paletteID); err != nil {
			return out, err
		}
		out.WorkspacePalettes[strconv.FormatInt(workspaceID, 10)] = paletteID
	}
	return out, rows.Err()
}

func (s *Store) SaveAppearance(ctx context.Context, in store.AppearanceSettings) (store.AppearanceSettings, error) {
	payload, err := json.Marshal(in.CustomPalettes)
	if err != nil {
		return store.AppearanceSettings{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return store.AppearanceSettings{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO appearance_settings (id, global_palette_id, custom_palettes, updated_at)
		 VALUES (1, $1, $2, now())
		 ON CONFLICT (id) DO UPDATE SET
		   global_palette_id = EXCLUDED.global_palette_id,
		   custom_palettes = EXCLUDED.custom_palettes,
		   updated_at = now()`,
		in.GlobalPaletteID, string(payload)); err != nil {
		return store.AppearanceSettings{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM workspace_themes`); err != nil {
		return store.AppearanceSettings{}, err
	}
	for workspace, paletteID := range in.WorkspacePalettes {
		workspaceID, err := strconv.ParseInt(workspace, 10, 64)
		if err != nil {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO workspace_themes (workspace_id, palette_id)
			 SELECT $1, $2 WHERE EXISTS (SELECT 1 FROM workspaces WHERE id = $1)`,
			workspaceID, paletteID); err != nil {
			return store.AppearanceSettings{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return store.AppearanceSettings{}, err
	}
	return s.GetAppearance(ctx)
}
