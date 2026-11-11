-- +goose Up
ALTER TABLE reminders
  ADD COLUMN repeat_freq TEXT CHECK (repeat_freq IN ('daily', 'weekly', 'monthly', 'yearly')),
  ADD COLUMN repeat_interval INTEGER CHECK (repeat_interval BETWEEN 1 AND 365),
  ADD COLUMN repeat_anchor TIMESTAMPTZ,
  ADD CONSTRAINT reminders_repeat_complete CHECK (
    (repeat_freq IS NULL) = (repeat_interval IS NULL) AND (repeat_freq IS NULL) = (repeat_anchor IS NULL));

ALTER TABLE tasks
  ADD COLUMN repeat_freq TEXT CHECK (repeat_freq IN ('daily', 'weekly', 'monthly', 'yearly')),
  ADD COLUMN repeat_interval INTEGER CHECK (repeat_interval BETWEEN 1 AND 365),
  ADD COLUMN repeat_anchor DATE,
  ADD CONSTRAINT tasks_repeat_complete CHECK (
    (repeat_freq IS NULL) = (repeat_interval IS NULL) AND (repeat_freq IS NULL) = (repeat_anchor IS NULL));

-- +goose Down
ALTER TABLE tasks
  DROP CONSTRAINT tasks_repeat_complete,
  DROP COLUMN repeat_anchor,
  DROP COLUMN repeat_interval,
  DROP COLUMN repeat_freq;

ALTER TABLE reminders
  DROP CONSTRAINT reminders_repeat_complete,
  DROP COLUMN repeat_anchor,
  DROP COLUMN repeat_interval,
  DROP COLUMN repeat_freq;
