-- name: CreateChatMessage :one
INSERT INTO chat_messages (workspace_id, role, content)
VALUES (?, ?, ?)
RETURNING *;

-- name: ListChatMessages :many
SELECT * FROM chat_messages
WHERE workspace_id = ?1
ORDER BY created_at DESC, id DESC
LIMIT ?2;
