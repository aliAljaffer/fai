-- name: ListReminders :many
SELECT id, title, remind_at, task_id, workspace_id, sent, created_at, completed_at, repeat_freq, repeat_interval, repeat_anchor FROM reminders
WHERE CAST(sqlc.narg('workspace_id') AS INTEGER) IS NULL OR workspace_id = sqlc.narg('workspace_id')
ORDER BY remind_at;

-- name: GetReminder :one
SELECT id, title, remind_at, task_id, workspace_id, sent, created_at, completed_at, repeat_freq, repeat_interval, repeat_anchor FROM reminders WHERE id = ?;

-- name: CreateReminder :one
INSERT INTO reminders (title, remind_at, task_id, workspace_id, repeat_freq, repeat_interval, repeat_anchor)
VALUES (sqlc.arg(title), sqlc.arg(remind_at), CAST(sqlc.narg('task_id') AS INTEGER),
  COALESCE(NULLIF(CAST(sqlc.arg(workspace_id) AS INTEGER), 0), (SELECT id FROM workspaces ORDER BY id LIMIT 1)),
  CAST(sqlc.narg('repeat_freq') AS TEXT), CAST(sqlc.narg('repeat_interval') AS INTEGER), CAST(sqlc.narg('repeat_anchor') AS TEXT))
RETURNING id, title, remind_at, task_id, workspace_id, sent, created_at, completed_at, repeat_freq, repeat_interval, repeat_anchor;

-- name: UpdateReminder :one
UPDATE reminders SET title = sqlc.arg(title), remind_at = sqlc.arg(remind_at), task_id = CAST(sqlc.narg('task_id') AS INTEGER), sent = 0,
  repeat_freq = CAST(sqlc.narg('repeat_freq') AS TEXT), repeat_interval = CAST(sqlc.narg('repeat_interval') AS INTEGER), repeat_anchor = CAST(sqlc.narg('repeat_anchor') AS TEXT)
WHERE id = sqlc.arg(id)
RETURNING id, title, remind_at, task_id, workspace_id, sent, created_at, completed_at, repeat_freq, repeat_interval, repeat_anchor;

-- name: DeleteReminder :exec
DELETE FROM reminders WHERE id = ?;

-- name: ListDueUnsentReminders :many
SELECT id, title, remind_at, task_id, workspace_id, sent, created_at, completed_at, repeat_freq, repeat_interval, repeat_anchor
FROM reminders WHERE sent = 0 AND completed_at IS NULL AND remind_at <= strftime('%Y-%m-%dT%H:%M:%fZ', 'now');

-- name: MarkReminderSent :exec
UPDATE reminders SET sent = 1 WHERE id = ?;

-- name: CompleteReminder :one
UPDATE reminders SET completed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?
RETURNING id, title, remind_at, task_id, workspace_id, sent, created_at, completed_at, repeat_freq, repeat_interval, repeat_anchor;

-- name: ReopenReminder :one
UPDATE reminders SET completed_at = NULL WHERE id = ?
RETURNING id, title, remind_at, task_id, workspace_id, sent, created_at, completed_at, repeat_freq, repeat_interval, repeat_anchor;

-- name: AdvanceReminder :one
UPDATE reminders SET remind_at = sqlc.arg(remind_at), sent = 0, completed_at = NULL WHERE id = sqlc.arg(id)
RETURNING id, title, remind_at, task_id, workspace_id, sent, created_at, completed_at, repeat_freq, repeat_interval, repeat_anchor;
