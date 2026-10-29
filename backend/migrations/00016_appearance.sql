-- +goose Up
CREATE TABLE appearance_settings (
  id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  global_palette_id TEXT NOT NULL DEFAULT 'light-steel',
  custom_palettes TEXT NOT NULL DEFAULT '[]',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO appearance_settings (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE workspace_themes (
  workspace_id BIGINT PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
  palette_id TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE workspace_themes;
DROP TABLE appearance_settings;
