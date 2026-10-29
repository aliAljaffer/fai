-- +goose Up
ALTER TABLE reminders ADD COLUMN completed_at TIMESTAMPTZ;
CREATE INDEX idx_reminders_completed_at ON reminders(completed_at) WHERE completed_at IS NOT NULL;

-- +goose Down
DROP INDEX idx_reminders_completed_at;
ALTER TABLE reminders DROP COLUMN completed_at;
