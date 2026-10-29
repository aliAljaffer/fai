CREATE TABLE IF NOT EXISTS ai_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  provider TEXT NOT NULL DEFAULT 'anthropic'
    CHECK (provider IN ('anthropic', 'openai', 'openai-compat')),
  base_url TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  api_key TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
