-- +goose Up
CREATE TABLE task_time_entries (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ended_at TIMESTAMPTZ,
  duration_seconds INTEGER GENERATED ALWAYS AS (
    CASE WHEN ended_at IS NOT NULL THEN EXTRACT(EPOCH FROM (ended_at - started_at))::INTEGER END) STORED
);
CREATE INDEX idx_time_entries_task_id ON task_time_entries(task_id);
CREATE UNIQUE INDEX one_running_timer_idx ON task_time_entries ((true)) WHERE ended_at IS NULL;

-- +goose Down
DROP TABLE task_time_entries;
