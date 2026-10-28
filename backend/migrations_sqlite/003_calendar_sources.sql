CREATE TABLE IF NOT EXISTS calendar_sources (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  url TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT '#2563eb',
  workspace_id INTEGER NOT NULL REFERENCES workspaces(id),
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
