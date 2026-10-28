-- name: ListReminders :many
SELECT id, title, remind_at, task_id, sent, created_at, workspace_id FROM reminders
WHERE sqlc.narg('workspace_id')::bigint IS NULL OR workspace_id = sqlc.narg('workspace_id')
ORDER BY remind_at;

-- name: GetReminder :one
SELECT id, title, remind_at, task_id, sent, created_at, workspace_id FROM reminders WHERE id = $1;

-- name: CreateReminder :one
INSERT INTO reminders (title, remind_at, task_id, workspace_id)
VALUES ($1, $2, $3, COALESCE(NULLIF(sqlc.arg(workspace_id)::bigint, 0), (SELECT id FROM workspaces ORDER BY id LIMIT 1)))
RETURNING id, title, remind_at, task_id, sent, created_at, workspace_id;

-- name: UpdateReminder :one
UPDATE reminders SET title = $2, remind_at = $3, task_id = $4, sent = false
WHERE id = $1
RETURNING id, title, remind_at, task_id, sent, created_at, workspace_id;

-- name: DeleteReminder :exec
DELETE FROM reminders WHERE id = $1;

-- name: ListDueUnsentReminders :many
SELECT id, title, remind_at, task_id, sent, created_at, workspace_id
FROM reminders WHERE sent = false AND remind_at <= now();

-- name: MarkReminderSent :exec
UPDATE reminders SET sent = true WHERE id = $1;
