-- name: ListTags :many
SELECT * FROM tags ORDER BY name;

-- name: CreateTag :one
INSERT INTO tags (name, color) VALUES ($1, $2) RETURNING *;

-- name: AddTaskTag :exec
INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveTaskTag :exec
DELETE FROM task_tags WHERE task_id = $1 AND tag_id = $2;

-- name: ListTagsForTask :many
SELECT t.* FROM tags t
JOIN task_tags tt ON tt.tag_id = t.id
WHERE tt.task_id = $1
ORDER BY t.name;
