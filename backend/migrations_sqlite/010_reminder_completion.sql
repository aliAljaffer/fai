CREATE INDEX IF NOT EXISTS idx_reminders_completed_at ON reminders(completed_at) WHERE completed_at IS NOT NULL;
