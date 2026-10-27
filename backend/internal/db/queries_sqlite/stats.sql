-- name: CompletionsByDay :many
SELECT CAST(date(completed_at) AS TEXT) AS day, count(*) AS count
FROM tasks
WHERE completed_at IS NOT NULL
  AND date(completed_at) BETWEEN CAST(sqlc.arg(from_date) AS TEXT) AND CAST(sqlc.arg(to_date) AS TEXT)
GROUP BY date(completed_at)
ORDER BY date(completed_at);
