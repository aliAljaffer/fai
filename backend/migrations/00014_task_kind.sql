-- +goose Up
ALTER TABLE tasks ADD COLUMN kind TEXT NOT NULL DEFAULT 'task'
  CHECK (kind IN ('arc', 'task', 'step'));

UPDATE tasks SET kind = 'step' WHERE parent_task_id IS NOT NULL;

ALTER TABLE tasks ADD CONSTRAINT tasks_arc_has_no_parent
  CHECK (kind <> 'arc' OR parent_task_id IS NULL);

-- +goose Down
ALTER TABLE tasks DROP CONSTRAINT tasks_arc_has_no_parent;
ALTER TABLE tasks DROP COLUMN kind;
