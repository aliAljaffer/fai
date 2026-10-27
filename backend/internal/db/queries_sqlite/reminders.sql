-- name: ListReminders :many
SELECT id, title, remind_at, task_id, sent, created_at FROM reminders ORDER BY remind_at;

-- name: CreateReminder :one
INSERT INTO reminders (title, remind_at, task_id) VALUES (sqlc.arg(title), sqlc.arg(remind_at), CAST(sqlc.narg('task_id') AS INTEGER))
RETURNING id, title, remind_at, task_id, sent, created_at;

-- name: UpdateReminder :one
UPDATE reminders SET title = sqlc.arg(title), remind_at = sqlc.arg(remind_at), task_id = CAST(sqlc.narg('task_id') AS INTEGER), sent = 0
WHERE id = sqlc.arg(id)
RETURNING id, title, remind_at, task_id, sent, created_at;

-- name: DeleteReminder :exec
DELETE FROM reminders WHERE id = ?;

-- name: ListDueUnsentReminders :many
SELECT id, title, remind_at, task_id, sent, created_at
FROM reminders WHERE sent = 0 AND remind_at <= strftime('%Y-%m-%dT%H:%M:%fZ', 'now');

-- name: MarkReminderSent :exec
UPDATE reminders SET sent = 1 WHERE id = ?;
