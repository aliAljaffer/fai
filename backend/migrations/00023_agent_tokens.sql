-- +goose Up
CREATE TABLE agent_tokens (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  prefix TEXT NOT NULL,
  workspace_id BIGINT REFERENCES workspaces(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_used_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE agent_tokens;
