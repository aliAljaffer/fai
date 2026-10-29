CREATE TABLE IF NOT EXISTS appearance_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  global_palette_id TEXT NOT NULL DEFAULT 'light-steel',
  custom_palettes TEXT NOT NULL DEFAULT '[]',
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

INSERT OR IGNORE INTO appearance_settings (id) VALUES (1);

CREATE TABLE IF NOT EXISTS workspace_themes (
  workspace_id INTEGER PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
  palette_id TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
