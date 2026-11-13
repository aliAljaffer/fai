-- +goose Up
ALTER TABLE tags ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id);

UPDATE tags SET workspace_id = COALESCE(
  (SELECT min(u.workspace_id) FROM (
  SELECT tt.tag_id, k.workspace_id FROM task_tags tt JOIN tasks k ON k.id = tt.task_id
  UNION
  SELECT nt.tag_id, n.workspace_id FROM note_tags nt JOIN notes n ON n.id = nt.note_id
) u WHERE u.tag_id = tags.id),
  (SELECT min(id) FROM workspaces));

ALTER TABLE tags ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE tags DROP CONSTRAINT tags_name_key;
ALTER TABLE tags ADD CONSTRAINT tags_workspace_name_key UNIQUE (workspace_id, name);

INSERT INTO tags (workspace_id, name, color)
SELECT u.workspace_id, t.name, t.color
FROM (
  SELECT tt.tag_id, k.workspace_id FROM task_tags tt JOIN tasks k ON k.id = tt.task_id
  UNION
  SELECT nt.tag_id, n.workspace_id FROM note_tags nt JOIN notes n ON n.id = nt.note_id
) u
JOIN tags t ON t.id = u.tag_id
WHERE u.workspace_id <> t.workspace_id;

UPDATE task_tags tt SET tag_id = scoped.id
FROM tasks k, tags shared, tags scoped
WHERE k.id = tt.task_id AND shared.id = tt.tag_id
  AND scoped.name = shared.name AND scoped.workspace_id = k.workspace_id
  AND scoped.id <> shared.id;

UPDATE note_tags nt SET tag_id = scoped.id
FROM notes n, tags shared, tags scoped
WHERE n.id = nt.note_id AND shared.id = nt.tag_id
  AND scoped.name = shared.name AND scoped.workspace_id = n.workspace_id
  AND scoped.id <> shared.id;

-- +goose Down
CREATE TEMP TABLE tag_merge ON COMMIT DROP AS
  SELECT id, min(id) OVER (PARTITION BY name) AS keep_id FROM tags;

UPDATE task_tags tt SET tag_id = m.keep_id
FROM tag_merge m WHERE m.id = tt.tag_id AND m.keep_id <> m.id;
UPDATE note_tags nt SET tag_id = m.keep_id
FROM tag_merge m WHERE m.id = nt.tag_id AND m.keep_id <> m.id;
DELETE FROM tags USING tag_merge m WHERE m.id = tags.id AND m.keep_id <> m.id;

ALTER TABLE tags DROP CONSTRAINT tags_workspace_name_key;
ALTER TABLE tags DROP COLUMN workspace_id;
ALTER TABLE tags ADD CONSTRAINT tags_name_key UNIQUE (name);
