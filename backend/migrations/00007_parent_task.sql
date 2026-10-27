-- +goose Up
ALTER TABLE tasks ADD COLUMN parent_task_id BIGINT REFERENCES tasks(id) ON DELETE SET NULL;
CREATE INDEX idx_tasks_parent ON tasks(parent_task_id) WHERE parent_task_id IS NOT NULL;

-- +goose Down
DROP INDEX idx_tasks_parent;
ALTER TABLE tasks DROP COLUMN parent_task_id;
