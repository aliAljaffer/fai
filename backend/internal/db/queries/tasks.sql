-- name: CreateTask :one
WITH col AS (SELECT is_done FROM board_columns WHERE id = $2)
INSERT INTO tasks (board_id, column_id, title, description_md, priority, due_date, position, parent_task_id, completed_at)
SELECT $1, $2, $3, $4, $5, $6, $7, sqlc.narg('parent_task_id'), CASE WHEN col.is_done THEN now() ELSE NULL END
FROM col
RETURNING id, board_id, column_id, title, description_md, priority, due_date, position, parent_task_id, completed_at, archived, created_at, updated_at;

-- name: GetTask :one
SELECT id, board_id, column_id, title, description_md, priority, due_date, position, parent_task_id, completed_at, archived, created_at, updated_at
FROM tasks WHERE id = $1 AND archived = false;

-- name: ListTasks :many
SELECT DISTINCT t.id, t.board_id, t.column_id, t.title, t.description_md, t.priority, t.due_date, t.position, t.parent_task_id, t.completed_at, t.archived, t.created_at, t.updated_at
FROM tasks t
LEFT JOIN task_tags tt ON tt.task_id = t.id
LEFT JOIN tags tg ON tg.id = tt.tag_id
WHERE t.archived = false
  AND (sqlc.narg('board_id')::bigint IS NULL OR t.board_id = sqlc.narg('board_id'))
  AND (sqlc.narg('column_id')::bigint IS NULL OR t.column_id = sqlc.narg('column_id'))
  AND (sqlc.narg('tag')::text IS NULL OR tg.name = sqlc.narg('tag'))
ORDER BY t.column_id, t.position;

-- name: ListSubtasks :many
SELECT id, board_id, column_id, title, description_md, priority, due_date, position, parent_task_id, completed_at, archived, created_at, updated_at
FROM tasks WHERE parent_task_id = $1 AND archived = false ORDER BY created_at;

-- name: UpdateTask :one
UPDATE tasks SET
  title = $2,
  description_md = $3,
  priority = $4,
  due_date = $5,
  updated_at = now()
WHERE id = $1 AND archived = false
RETURNING id, board_id, column_id, title, description_md, priority, due_date, position, parent_task_id, completed_at, archived, created_at, updated_at;

-- name: MoveTask :one
UPDATE tasks t SET
  column_id = $2,
  position = $3,
  completed_at = CASE
    WHEN (SELECT is_done FROM board_columns WHERE id = $2) THEN COALESCE(t.completed_at, now())
    ELSE NULL
  END,
  updated_at = now()
WHERE t.id = $1
RETURNING id, board_id, column_id, title, description_md, priority, due_date, position, parent_task_id, completed_at, archived, created_at, updated_at;

-- name: ListTasksByColumnOrdered :many
SELECT id, board_id, column_id, title, description_md, priority, due_date, position, parent_task_id, completed_at, archived, created_at, updated_at
FROM tasks WHERE column_id = $1 AND archived = false ORDER BY position;

-- name: SetTaskPosition :exec
UPDATE tasks SET position = $2, updated_at = now() WHERE id = $1;

-- name: ArchiveTask :exec
UPDATE tasks SET archived = true, updated_at = now() WHERE id = $1;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1;
