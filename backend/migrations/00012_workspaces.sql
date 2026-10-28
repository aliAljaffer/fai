-- +goose Up
CREATE TABLE workspaces (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name TEXT NOT NULL,
  tag TEXT NOT NULL,
  ref_seq BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_workspaces_tag ON workspaces (upper(tag));

CREATE TABLE workspace_tag_aliases (
  workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  tag TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (workspace_id, tag)
);
CREATE UNIQUE INDEX idx_workspace_tag_aliases_tag ON workspace_tag_aliases (upper(tag));

ALTER TABLE boards ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE notes ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE reminders ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE calendar_sources ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE meeting_outcomes ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE tasks ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);
ALTER TABLE tasks ADD COLUMN ref_num BIGINT;

INSERT INTO workspaces (name, tag)
SELECT 'Personal', 'FAI'
WHERE NOT EXISTS (SELECT 1 FROM workspaces);

UPDATE boards SET workspace_id = (SELECT id FROM workspaces ORDER BY id LIMIT 1)
WHERE workspace_id IS NULL;
UPDATE notes SET workspace_id = (SELECT id FROM workspaces ORDER BY id LIMIT 1)
WHERE workspace_id IS NULL;
UPDATE reminders SET workspace_id = (SELECT id FROM workspaces ORDER BY id LIMIT 1)
WHERE workspace_id IS NULL;
UPDATE calendar_sources SET workspace_id = (SELECT id FROM workspaces ORDER BY id LIMIT 1)
WHERE workspace_id IS NULL;
UPDATE meeting_outcomes SET workspace_id = (SELECT id FROM workspaces ORDER BY id LIMIT 1)
WHERE workspace_id IS NULL;
UPDATE tasks SET workspace_id = boards.workspace_id
FROM boards
WHERE tasks.board_id = boards.id AND tasks.workspace_id IS NULL;

UPDATE tasks
SET ref_num = numbered.rn
FROM (
  SELECT id, row_number() OVER (PARTITION BY workspace_id ORDER BY id) AS rn
  FROM tasks
) AS numbered
WHERE tasks.id = numbered.id AND tasks.ref_num IS NULL;

UPDATE workspaces
SET ref_seq = COALESCE((SELECT max(ref_num) FROM tasks WHERE tasks.workspace_id = workspaces.id), 0);

ALTER TABLE boards ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE notes ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE reminders ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE calendar_sources ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE meeting_outcomes ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE tasks ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE tasks ALTER COLUMN ref_num SET NOT NULL;

CREATE UNIQUE INDEX idx_tasks_workspace_ref ON tasks (workspace_id, ref_num);

-- +goose Down
DROP INDEX idx_tasks_workspace_ref;
ALTER TABLE tasks DROP COLUMN ref_num;
ALTER TABLE tasks DROP COLUMN workspace_id;
ALTER TABLE meeting_outcomes DROP COLUMN workspace_id;
ALTER TABLE calendar_sources DROP COLUMN workspace_id;
ALTER TABLE reminders DROP COLUMN workspace_id;
ALTER TABLE notes DROP COLUMN workspace_id;
ALTER TABLE boards DROP COLUMN workspace_id;
DROP TABLE workspace_tag_aliases;
DROP TABLE workspaces;
