-- name: CompletionsByDay :many
SELECT completed_at::date AS day, count(*) AS count
FROM tasks
WHERE completed_at IS NOT NULL
  AND completed_at::date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
GROUP BY completed_at::date
ORDER BY completed_at::date;
