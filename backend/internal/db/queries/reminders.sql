-- name: ListReminders :many
SELECT id, title, remind_at, task_id, sent, created_at, workspace_id, completed_at, repeat_freq, repeat_interval, repeat_anchor FROM reminders
WHERE sqlc.narg('workspace_id')::bigint IS NULL OR workspace_id = sqlc.narg('workspace_id')
ORDER BY remind_at;

-- name: GetReminder :one
SELECT id, title, remind_at, task_id, sent, created_at, workspace_id, completed_at, repeat_freq, repeat_interval, repeat_anchor FROM reminders WHERE id = $1;

-- name: CreateReminder :one
INSERT INTO reminders (title, remind_at, task_id, workspace_id, repeat_freq, repeat_interval, repeat_anchor)
VALUES ($1, $2, $3, COALESCE(NULLIF(sqlc.arg(workspace_id)::bigint, 0), (SELECT id FROM workspaces ORDER BY id LIMIT 1)),
  sqlc.narg('repeat_freq'), sqlc.narg('repeat_interval'), sqlc.narg('repeat_anchor'))
RETURNING id, title, remind_at, task_id, sent, created_at, workspace_id, completed_at, repeat_freq, repeat_interval, repeat_anchor;

-- name: UpdateReminder :one
UPDATE reminders SET title = $2, remind_at = $3, task_id = $4, sent = false,
  repeat_freq = sqlc.narg('repeat_freq'), repeat_interval = sqlc.narg('repeat_interval'), repeat_anchor = sqlc.narg('repeat_anchor')
WHERE id = $1
RETURNING id, title, remind_at, task_id, sent, created_at, workspace_id, completed_at, repeat_freq, repeat_interval, repeat_anchor;

-- name: DeleteReminder :exec
DELETE FROM reminders WHERE id = $1;

-- name: ListDueUnsentReminders :many
SELECT id, title, remind_at, task_id, sent, created_at, workspace_id, completed_at, repeat_freq, repeat_interval, repeat_anchor
FROM reminders WHERE sent = false AND completed_at IS NULL AND remind_at <= now();

-- name: MarkReminderSent :exec
UPDATE reminders SET sent = true WHERE id = $1;

-- name: CompleteReminder :one
UPDATE reminders SET completed_at = now() WHERE id = $1
RETURNING id, title, remind_at, task_id, sent, created_at, workspace_id, completed_at, repeat_freq, repeat_interval, repeat_anchor;

-- name: ReopenReminder :one
UPDATE reminders SET completed_at = NULL WHERE id = $1
RETURNING id, title, remind_at, task_id, sent, created_at, workspace_id, completed_at, repeat_freq, repeat_interval, repeat_anchor;

-- name: AdvanceReminder :one
UPDATE reminders SET remind_at = $2, sent = false, completed_at = NULL WHERE id = $1
RETURNING id, title, remind_at, task_id, sent, created_at, workspace_id, completed_at, repeat_freq, repeat_interval, repeat_anchor;
