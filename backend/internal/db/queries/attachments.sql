-- name: CreateAttachment :one
INSERT INTO attachments (task_id, note_id, filename, content_type, size_bytes, storage_key)
VALUES (
  sqlc.narg('task_id')::bigint,
  sqlc.narg('note_id')::bigint,
  sqlc.arg('filename'),
  sqlc.arg('content_type'),
  sqlc.arg('size_bytes'),
  sqlc.arg('storage_key')
)
RETURNING id, task_id, note_id, filename, content_type, size_bytes, storage_key, created_at;

-- name: ListAttachmentsByTask :many
SELECT id, task_id, note_id, filename, content_type, size_bytes, storage_key, created_at
FROM attachments WHERE task_id = $1::bigint ORDER BY created_at;

-- name: ListAttachmentsByNote :many
SELECT id, task_id, note_id, filename, content_type, size_bytes, storage_key, created_at
FROM attachments WHERE note_id = $1::bigint ORDER BY created_at;

-- name: GetAttachment :one
SELECT id, task_id, note_id, filename, content_type, size_bytes, storage_key, created_at
FROM attachments WHERE id = $1;

-- name: DeleteAttachment :exec
DELETE FROM attachments WHERE id = $1;
