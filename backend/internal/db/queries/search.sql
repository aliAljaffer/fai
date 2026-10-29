-- name: SearchTasks :many
SELECT id, board_id, column_id, title, description_md, priority, due_date, position, parent_task_id, kind, meeting_outcome_id, completed_at, archived, created_at, updated_at, workspace_id, ref_num
FROM tasks
WHERE archived = false AND search_vector @@ websearch_to_tsquery('english', $1)
  AND (sqlc.narg('workspace_id')::bigint IS NULL OR workspace_id = sqlc.narg('workspace_id'))
ORDER BY ts_rank(search_vector, websearch_to_tsquery('english', $1)) DESC
LIMIT 20;

-- name: SearchNotes :many
SELECT id, title, body_md, task_id, created_at, updated_at, workspace_id
FROM notes
WHERE search_vector @@ websearch_to_tsquery('english', $1)
  AND (sqlc.narg('workspace_id')::bigint IS NULL OR workspace_id = sqlc.narg('workspace_id'))
ORDER BY ts_rank(search_vector, websearch_to_tsquery('english', $1)) DESC
LIMIT 20;
