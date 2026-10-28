-- name: ListBoards :many
SELECT * FROM boards
WHERE sqlc.narg('workspace_id')::bigint IS NULL OR workspace_id = sqlc.narg('workspace_id')
ORDER BY id;

-- name: GetBoard :one
SELECT * FROM boards WHERE id = $1;

-- name: CreateBoard :one
INSERT INTO boards (name, workspace_id)
VALUES ($1, COALESCE(NULLIF(sqlc.arg(workspace_id)::bigint, 0), (SELECT id FROM workspaces ORDER BY id LIMIT 1)))
RETURNING *;

-- name: ListColumnsByBoard :many
SELECT * FROM board_columns WHERE board_id = $1 ORDER BY position;

-- name: CreateColumn :one
INSERT INTO board_columns (board_id, name, position, is_done)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetColumn :one
SELECT * FROM board_columns WHERE id = $1;
