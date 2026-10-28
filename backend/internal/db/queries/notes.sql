-- name: ListNotes :many
SELECT id, title, body_md, task_id, created_at, updated_at, workspace_id FROM notes
WHERE sqlc.narg('workspace_id')::bigint IS NULL OR workspace_id = sqlc.narg('workspace_id')
ORDER BY updated_at DESC;

-- name: GetNote :one
SELECT id, title, body_md, task_id, created_at, updated_at, workspace_id FROM notes WHERE id = $1;

-- name: GetNoteByTitle :one
SELECT id, title, body_md, task_id, created_at, updated_at, workspace_id FROM notes WHERE title = $1;

-- name: CreateNote :one
INSERT INTO notes (title, body_md, task_id, workspace_id)
VALUES ($1, $2, $3, COALESCE(NULLIF(sqlc.arg(workspace_id)::bigint, 0), (SELECT id FROM workspaces ORDER BY id LIMIT 1)))
RETURNING id, title, body_md, task_id, created_at, updated_at, workspace_id;

-- name: UpdateNote :one
UPDATE notes SET title = $2, body_md = $3, task_id = $4, updated_at = now()
WHERE id = $1
RETURNING id, title, body_md, task_id, created_at, updated_at, workspace_id;

-- name: DeleteNote :exec
DELETE FROM notes WHERE id = $1;

-- name: ListBacklinks :many
SELECT id, title, body_md, task_id, created_at, updated_at, workspace_id
FROM notes WHERE id != $1 AND body_md ILIKE '%[[' || $2::text || ']]%';
