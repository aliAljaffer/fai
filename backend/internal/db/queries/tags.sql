-- name: ListTags :many
SELECT * FROM tags
WHERE sqlc.narg('workspace_id')::bigint IS NULL OR workspace_id = sqlc.narg('workspace_id')
ORDER BY name;

-- name: CreateTag :one
INSERT INTO tags (workspace_id, name, color) VALUES ($1, $2, $3)
RETURNING *;

-- name: AddTaskTag :execrows
INSERT INTO task_tags (task_id, tag_id)
SELECT k.id, t.id FROM tasks k JOIN tags t ON t.workspace_id = k.workspace_id
WHERE k.id = sqlc.arg(task_id) AND t.id = sqlc.arg(tag_id)
ON CONFLICT DO NOTHING;

-- name: TaskHasTag :one
SELECT EXISTS (SELECT 1 FROM task_tags WHERE task_id = $1 AND tag_id = $2);

-- name: RemoveTaskTag :exec
DELETE FROM task_tags WHERE task_id = $1 AND tag_id = $2;
