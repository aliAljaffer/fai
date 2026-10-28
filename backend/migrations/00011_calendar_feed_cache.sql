-- +goose Up
CREATE TABLE calendar_feed_cache (
  source_id BIGINT PRIMARY KEY REFERENCES calendar_sources(id) ON DELETE CASCADE,
  data TEXT NOT NULL,
  fetched_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE calendar_feed_cache;
