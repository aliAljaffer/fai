CREATE TABLE IF NOT EXISTS workspaces (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  tag TEXT NOT NULL,
  ref_seq INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_workspaces_tag ON workspaces (upper(tag));

CREATE TABLE IF NOT EXISTS workspace_tag_aliases (
  workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  tag TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  PRIMARY KEY (workspace_id, tag)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_workspace_tag_aliases_tag ON workspace_tag_aliases (upper(tag));

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
UPDATE tasks SET workspace_id = (SELECT workspace_id FROM boards WHERE boards.id = tasks.board_id)
WHERE workspace_id IS NULL;

UPDATE tasks
SET ref_num = (
  SELECT count(*) FROM tasks AS earlier
  WHERE earlier.workspace_id = tasks.workspace_id AND earlier.id <= tasks.id
)
WHERE ref_num IS NULL;

UPDATE workspaces
SET ref_seq = COALESCE((SELECT max(ref_num) FROM tasks WHERE tasks.workspace_id = workspaces.id), 0);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_workspace_ref ON tasks (workspace_id, ref_num);
