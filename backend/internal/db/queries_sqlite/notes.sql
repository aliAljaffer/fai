-- name: ListNotes :many
SELECT id, title, body_md, task_id, created_at, updated_at FROM notes ORDER BY updated_at DESC;

-- name: GetNote :one
SELECT id, title, body_md, task_id, created_at, updated_at FROM notes WHERE id = ?;

-- name: GetNoteByTitle :one
SELECT id, title, body_md, task_id, created_at, updated_at FROM notes WHERE title = ?;

-- name: CreateNote :one
INSERT INTO notes (title, body_md, task_id) VALUES (sqlc.arg(title), sqlc.arg(body_md), CAST(sqlc.narg('task_id') AS INTEGER))
RETURNING id, title, body_md, task_id, created_at, updated_at;

-- name: UpdateNote :one
UPDATE notes SET title = sqlc.arg(title), body_md = sqlc.arg(body_md), task_id = CAST(sqlc.narg('task_id') AS INTEGER), updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = sqlc.arg(id)
RETURNING id, title, body_md, task_id, created_at, updated_at;

-- name: DeleteNote :exec
DELETE FROM notes WHERE id = ?;

-- name: ListBacklinks :many
SELECT id, title, body_md, task_id, created_at, updated_at
FROM notes WHERE id != sqlc.arg(id) AND body_md LIKE '%[[' || sqlc.arg(title) || ']]%';
