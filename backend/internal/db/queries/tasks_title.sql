-- name: UpdateTaskTitle :exec
UPDATE tasks SET title = $2, updated_at = now() WHERE id = $1;
