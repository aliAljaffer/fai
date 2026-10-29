-- +goose Up
CREATE TABLE ai_settings (
  id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  provider TEXT NOT NULL DEFAULT 'anthropic'
    CHECK (provider IN ('anthropic', 'openai', 'openai-compat')),
  base_url TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  api_key TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE ai_settings;
