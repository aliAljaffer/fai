-- +goose Up
ALTER TABLE meeting_outcomes
  ADD COLUMN calendar_source_id BIGINT REFERENCES calendar_sources(id) ON DELETE SET NULL,
  ADD COLUMN event_uid TEXT NOT NULL DEFAULT '',
  ADD COLUMN event_title TEXT NOT NULL DEFAULT '',
  ADD COLUMN event_url TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_meeting_outcomes_event ON meeting_outcomes(calendar_source_id, event_uid)
  WHERE event_uid <> '';

-- +goose Down
DROP INDEX idx_meeting_outcomes_event;
ALTER TABLE meeting_outcomes
  DROP COLUMN event_url,
  DROP COLUMN event_title,
  DROP COLUMN event_uid,
  DROP COLUMN calendar_source_id;
