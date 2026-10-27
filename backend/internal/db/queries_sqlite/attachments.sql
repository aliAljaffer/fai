-- name: CreateAttachment :one
INSERT INTO attachments (task_id, note_id, filename, content_type, size_bytes, storage_key)
VALUES (
  CAST(sqlc.narg('task_id') AS INTEGER),
  CAST(sqlc.narg('note_id') AS INTEGER),
  sqlc.arg('filename'),
  sqlc.arg('content_type'),
  sqlc.arg('size_bytes'),
  sqlc.arg('storage_key')
)
RETURNING *;

-- name: ListAttachmentsByTask :many
SELECT * FROM attachments WHERE task_id = CAST(?1 AS INTEGER) ORDER BY created_at;

-- name: ListAttachmentsByNote :many
SELECT * FROM attachments WHERE note_id = CAST(?1 AS INTEGER) ORDER BY created_at;

-- name: GetAttachment :one
SELECT * FROM attachments WHERE id = ?;

-- name: DeleteAttachment :exec
DELETE FROM attachments WHERE id = ?;
