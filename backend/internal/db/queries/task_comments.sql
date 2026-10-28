-- name: CreateTaskComment :one
INSERT INTO task_comments (task_id, author, body_md, external_id)
VALUES ($1, $2, $3, sqlc.narg('external_id'))
RETURNING *;

-- name: ListTaskComments :many
SELECT * FROM task_comments WHERE task_id = $1 ORDER BY created_at, id;

-- name: DeleteTaskComment :exec
DELETE FROM task_comments WHERE id = $1 AND task_id = $2;
