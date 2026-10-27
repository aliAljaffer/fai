-- name: StopRunningEntry :exec
UPDATE task_time_entries SET ended_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE ended_at IS NULL;

-- name: StartEntry :one
INSERT INTO task_time_entries (task_id) VALUES (?)
RETURNING id, task_id, started_at, ended_at, duration_seconds;

-- name: StopEntryForTask :one
UPDATE task_time_entries SET ended_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE task_id = sqlc.arg(task_id) AND ended_at IS NULL
RETURNING id, task_id, started_at, ended_at, duration_seconds;

-- name: GetRunningEntry :one
SELECT id, task_id, started_at, ended_at, duration_seconds
FROM task_time_entries WHERE ended_at IS NULL;

-- name: ListEntriesForTask :many
SELECT id, task_id, started_at, ended_at, duration_seconds
FROM task_time_entries WHERE task_id = ? ORDER BY started_at DESC;

-- name: TimeByDay :many
SELECT CAST(date(started_at) AS TEXT) AS day, CAST(sum(duration_seconds) AS INTEGER) AS seconds
FROM task_time_entries
WHERE ended_at IS NOT NULL
  AND date(started_at) BETWEEN CAST(sqlc.arg(from_date) AS TEXT) AND CAST(sqlc.arg(to_date) AS TEXT)
GROUP BY date(started_at)
ORDER BY date(started_at);
