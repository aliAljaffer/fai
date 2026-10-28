-- name: CreateTask :one
INSERT INTO tasks (board_id, column_id, workspace_id, ref_num, title, description_md, priority, due_date, position, parent_task_id, meeting_outcome_id, completed_at)
VALUES (
  sqlc.arg(board_id), sqlc.arg(column_id), sqlc.arg(workspace_id), sqlc.arg(ref_num),
  sqlc.arg(title), sqlc.arg(description_md), sqlc.arg(priority),
  CAST(sqlc.narg('due_date') AS TEXT), sqlc.arg(position),
  CAST(sqlc.narg('parent_task_id') AS INTEGER), CAST(sqlc.narg('meeting_outcome_id') AS INTEGER),
  CASE WHEN (SELECT board_columns.is_done FROM board_columns WHERE board_columns.id = sqlc.arg(column_id))
    THEN strftime('%Y-%m-%dT%H:%M:%fZ', 'now') ELSE NULL END
)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ? AND archived = 0;

-- name: ListTasks :many
SELECT DISTINCT t.* FROM tasks t
LEFT JOIN task_tags tt ON tt.task_id = t.id
LEFT JOIN tags tg ON tg.id = tt.tag_id
WHERE t.archived = 0
  AND (CAST(sqlc.narg('board_id') AS INTEGER) IS NULL OR t.board_id = sqlc.narg('board_id'))
  AND (CAST(sqlc.narg('column_id') AS INTEGER) IS NULL OR t.column_id = sqlc.narg('column_id'))
  AND (CAST(sqlc.narg('tag') AS TEXT) IS NULL OR tg.name = sqlc.narg('tag'))
  AND (CAST(sqlc.narg('q') AS TEXT) IS NULL
       OR t.title LIKE '%' || sqlc.narg('q') || '%'
       OR t.description_md LIKE '%' || sqlc.narg('q') || '%')
ORDER BY t.column_id, t.position;

-- name: ListSubtasks :many
SELECT * FROM tasks WHERE parent_task_id = ? AND archived = 0 ORDER BY created_at;

-- name: UpdateTask :one
UPDATE tasks SET
  title = sqlc.arg(title),
  description_md = sqlc.arg(description_md),
  priority = sqlc.arg(priority),
  due_date = CAST(sqlc.narg('due_date') AS TEXT),
  updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE tasks.id = sqlc.arg(id) AND archived = 0
RETURNING *;

-- name: MoveTask :one
UPDATE tasks SET
  column_id = sqlc.arg(column_id),
  position = sqlc.arg(position),
  completed_at = CASE
    WHEN (SELECT board_columns.is_done FROM board_columns WHERE board_columns.id = sqlc.arg(column_id))
      THEN COALESCE(completed_at, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
    ELSE NULL
  END,
  updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE tasks.id = sqlc.arg(id)
RETURNING *;

-- name: ListTasksByColumnOrdered :many
SELECT * FROM tasks WHERE column_id = ? AND archived = 0 ORDER BY position;

-- name: SetTaskPosition :exec
UPDATE tasks SET position = sqlc.arg(position), updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = sqlc.arg(id);

-- name: ArchiveTask :exec
UPDATE tasks SET archived = 1, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = ?;

-- name: UpdateTaskTitle :exec
UPDATE tasks SET title = sqlc.arg(title), updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = sqlc.arg(id);

-- name: SetTaskParent :exec
UPDATE tasks SET parent_task_id = CAST(sqlc.narg('parent_task_id') AS INTEGER), updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = sqlc.arg(id);
