-- +goose Up
CREATE TABLE reminders (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  title TEXT NOT NULL,
  remind_at TIMESTAMPTZ NOT NULL,
  task_id BIGINT REFERENCES tasks(id) ON DELETE SET NULL,
  sent BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_reminders_due_unsent ON reminders(remind_at) WHERE sent = false;

-- +goose Down
DROP TABLE reminders;
