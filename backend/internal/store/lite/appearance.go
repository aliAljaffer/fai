package lite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) GetAppearance(ctx context.Context) (store.AppearanceSettings, error) {
	out := store.DefaultAppearance()
	var customPalettes string
	var updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT global_palette_id, custom_palettes, updated_at FROM appearance_settings WHERE id = 1`).
		Scan(&out.GlobalPaletteID, &customPalettes, &updatedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	if err == nil {
		out.UpdatedAt, _ = parseTime(updatedAt)
		if customPalettes != "" && customPalettes != "null" {
			if err := json.Unmarshal([]byte(customPalettes), &out.CustomPalettes); err != nil {
				return out, err
			}
		}
	}
	if out.CustomPalettes == nil {
		out.CustomPalettes = []store.CustomPalette{}
	}

	rows, err := s.db.QueryContext(ctx, `SELECT workspace_id, palette_id FROM workspace_themes`)
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.AppearanceSettings{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO appearance_settings (id, global_palette_id, custom_palettes)
		 VALUES (1, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET
		   global_palette_id = excluded.global_palette_id,
		   custom_palettes = excluded.custom_palettes,
		   updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`,
		in.GlobalPaletteID, string(payload)); err != nil {
		return store.AppearanceSettings{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_themes`); err != nil {
		return store.AppearanceSettings{}, err
	}
	for workspace, paletteID := range in.WorkspacePalettes {
		workspaceID, err := strconv.ParseInt(workspace, 10, 64)
		if err != nil {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO workspace_themes (workspace_id, palette_id)
			 SELECT ?, ? WHERE EXISTS (SELECT 1 FROM workspaces WHERE id = ?)`,
			workspaceID, paletteID, workspaceID); err != nil {
			return store.AppearanceSettings{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return store.AppearanceSettings{}, err
	}
	return s.GetAppearance(ctx)
}
