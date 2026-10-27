-- name: ListBoards :many
SELECT * FROM boards ORDER BY id;

-- name: CreateBoard :one
INSERT INTO boards (name) VALUES (?) RETURNING *;

-- name: ListColumnsByBoard :many
SELECT * FROM board_columns WHERE board_id = ? ORDER BY position;

-- name: CreateColumn :one
INSERT INTO board_columns (board_id, name, position, is_done)
VALUES (?, ?, ?, ?) RETURNING *;
