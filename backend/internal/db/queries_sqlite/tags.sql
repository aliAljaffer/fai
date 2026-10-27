-- name: ListTags :many
SELECT * FROM tags ORDER BY name;

-- name: CreateTag :one
INSERT INTO tags (name, color) VALUES (?, ?) RETURNING *;

-- name: AddTaskTag :exec
INSERT INTO task_tags (task_id, tag_id) VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: RemoveTaskTag :exec
DELETE FROM task_tags WHERE task_id = ? AND tag_id = ?;
