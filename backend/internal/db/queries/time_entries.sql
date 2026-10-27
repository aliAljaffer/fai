-- name: StopRunningEntry :exec
UPDATE task_time_entries SET ended_at = now() WHERE ended_at IS NULL;

-- name: StartEntry :one
INSERT INTO task_time_entries (task_id) VALUES ($1)
RETURNING id, task_id, started_at, ended_at, duration_seconds;

-- name: StopEntryForTask :one
UPDATE task_time_entries SET ended_at = now()
WHERE task_id = $1 AND ended_at IS NULL
RETURNING id, task_id, started_at, ended_at, duration_seconds;

-- name: GetRunningEntry :one
SELECT id, task_id, started_at, ended_at, duration_seconds
FROM task_time_entries WHERE ended_at IS NULL;

-- name: ListEntriesForTask :many
SELECT id, task_id, started_at, ended_at, duration_seconds
FROM task_time_entries WHERE task_id = $1 ORDER BY started_at DESC;

-- name: TimeByDay :many
SELECT started_at::date AS day, sum(duration_seconds)::bigint AS seconds
FROM task_time_entries
WHERE ended_at IS NOT NULL
  AND started_at::date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
GROUP BY started_at::date
ORDER BY started_at::date;
