-- name: ListNotes :many
SELECT * FROM notes ORDER BY updated_at DESC;

-- name: GetNote :one
SELECT * FROM notes WHERE id = $1;

-- name: GetNoteByTitle :one
SELECT * FROM notes WHERE title = $1;

-- name: CreateNote :one
INSERT INTO notes (title, body_md, task_id) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateNote :one
UPDATE notes SET title = $2, body_md = $3, task_id = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteNote :exec
DELETE FROM notes WHERE id = $1;

-- name: ListBacklinks :many
SELECT * FROM notes WHERE id != $1 AND body_md ILIKE '%[[' || $2::text || ']]%';
