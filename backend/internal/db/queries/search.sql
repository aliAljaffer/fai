-- name: SearchTasks :many
SELECT id, board_id, column_id, title, description_md, priority, due_date, position, completed_at, archived, created_at, updated_at
FROM tasks
WHERE archived = false AND search_vector @@ websearch_to_tsquery('english', $1)
ORDER BY ts_rank(search_vector, websearch_to_tsquery('english', $1)) DESC
LIMIT 20;

-- name: SearchNotes :many
SELECT id, title, body_md, task_id, created_at, updated_at
FROM notes
WHERE search_vector @@ websearch_to_tsquery('english', $1)
ORDER BY ts_rank(search_vector, websearch_to_tsquery('english', $1)) DESC
LIMIT 20;
