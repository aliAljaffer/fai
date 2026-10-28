-- name: ListBoards :many
SELECT * FROM boards
WHERE CAST(sqlc.narg('workspace_id') AS INTEGER) IS NULL OR workspace_id = sqlc.narg('workspace_id')
ORDER BY id;

-- name: CreateBoard :one
INSERT INTO boards (name, workspace_id)
VALUES (sqlc.arg(name), COALESCE(NULLIF(CAST(sqlc.arg(workspace_id) AS INTEGER), 0), (SELECT id FROM workspaces ORDER BY id LIMIT 1)))
RETURNING *;

-- name: ListColumnsByBoard :many
SELECT * FROM board_columns WHERE board_id = ? ORDER BY position;

-- name: CreateColumn :one
INSERT INTO board_columns (board_id, name, position, is_done)
VALUES (?, ?, ?, ?) RETURNING *;
