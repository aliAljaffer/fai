CREATE TABLE IF NOT EXISTS calendar_feed_cache (
  source_id INTEGER PRIMARY KEY REFERENCES calendar_sources(id) ON DELETE CASCADE,
  data TEXT NOT NULL,
  fetched_at TEXT NOT NULL
);
