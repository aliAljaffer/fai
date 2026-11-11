-- name: CreateTask :one
INSERT INTO tasks (board_id, column_id, workspace_id, ref_num, title, description_md, priority, due_date, position, parent_task_id, kind, meeting_outcome_id, completed_at, created_by_fai, repeat_freq, repeat_interval, repeat_anchor)
SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, sqlc.narg('parent_task_id'), sqlc.arg('kind'), sqlc.narg('meeting_outcome_id'),
  CASE WHEN col.is_done THEN now() ELSE NULL END, sqlc.arg('created_by_fai'),
  sqlc.narg('repeat_freq'), sqlc.narg('repeat_interval'), sqlc.narg('repeat_anchor')
FROM (SELECT is_done FROM board_columns WHERE id = $2) col
RETURNING id, board_id, column_id, title, description_md, priority, due_date, position, completed_at, archived, created_at, updated_at, parent_task_id, kind, meeting_outcome_id, workspace_id, ref_num, created_by_fai, repeat_freq, repeat_interval, repeat_anchor;

-- name: GetTask :one
SELECT id, board_id, column_id, title, description_md, priority, due_date, position, completed_at, archived, created_at, updated_at, parent_task_id, kind, meeting_outcome_id, workspace_id, ref_num, created_by_fai, repeat_freq, repeat_interval, repeat_anchor
FROM tasks WHERE id = $1 AND archived = false;

-- name: ListTasks :many
SELECT DISTINCT t.id, t.board_id, t.column_id, t.title, t.description_md, t.priority, t.due_date, t.position, t.completed_at, t.archived, t.created_at, t.updated_at, t.parent_task_id, t.kind, t.meeting_outcome_id, t.workspace_id, t.ref_num, t.created_by_fai, t.repeat_freq, t.repeat_interval, t.repeat_anchor
FROM tasks t
LEFT JOIN task_tags tt ON tt.task_id = t.id
LEFT JOIN tags tg ON tg.id = tt.tag_id
WHERE t.archived = false
  AND (sqlc.narg('board_id')::bigint IS NULL OR t.board_id = sqlc.narg('board_id'))
  AND (sqlc.narg('column_id')::bigint IS NULL OR t.column_id = sqlc.narg('column_id'))
  AND (sqlc.narg('tag')::text IS NULL OR tg.name = sqlc.narg('tag'))
  AND (sqlc.narg('q')::text IS NULL
       OR t.title ILIKE '%' || sqlc.narg('q') || '%'
       OR t.description_md ILIKE '%' || sqlc.narg('q') || '%')
ORDER BY t.column_id, t.position;

-- name: ListSubtasks :many
SELECT id, board_id, column_id, title, description_md, priority, due_date, position, completed_at, archived, created_at, updated_at, parent_task_id, kind, meeting_outcome_id, workspace_id, ref_num, created_by_fai, repeat_freq, repeat_interval, repeat_anchor
FROM tasks WHERE parent_task_id = $1 AND archived = false ORDER BY created_at;


-- name: PatchTask :one
UPDATE tasks SET
  title = COALESCE(sqlc.narg('title'), title),
  description_md = COALESCE(sqlc.narg('description_md'), description_md),
  priority = COALESCE(sqlc.narg('priority'), priority),
  due_date = CASE WHEN sqlc.arg('due_date_set')::boolean THEN sqlc.narg('due_date') ELSE due_date END,
  repeat_freq = CASE WHEN sqlc.arg('repeat_set')::boolean THEN sqlc.narg('repeat_freq') ELSE repeat_freq END,
  repeat_interval = CASE WHEN sqlc.arg('repeat_set')::boolean THEN sqlc.narg('repeat_interval') ELSE repeat_interval END,
  repeat_anchor = CASE WHEN sqlc.arg('repeat_set')::boolean THEN sqlc.narg('repeat_anchor') ELSE repeat_anchor END,
  updated_at = now()
WHERE id = sqlc.arg('id') AND archived = false
RETURNING *;

-- name: RestoreTask :exec
UPDATE tasks SET archived = false, updated_at = now() WHERE id = $1;

-- name: UpdateTask :one
UPDATE tasks SET
  title = $2,
  description_md = $3,
  priority = $4,
  due_date = $5,
  updated_at = now()
WHERE id = $1 AND archived = false
RETURNING id, board_id, column_id, title, description_md, priority, due_date, position, completed_at, archived, created_at, updated_at, parent_task_id, kind, meeting_outcome_id, workspace_id, ref_num, created_by_fai, repeat_freq, repeat_interval, repeat_anchor;

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
RETURNING id, board_id, column_id, title, description_md, priority, due_date, position, completed_at, archived, created_at, updated_at, parent_task_id, kind, meeting_outcome_id, workspace_id, ref_num, created_by_fai, repeat_freq, repeat_interval, repeat_anchor;

-- name: ListTasksByColumnOrdered :many
SELECT id, board_id, column_id, title, description_md, priority, due_date, position, completed_at, archived, created_at, updated_at, parent_task_id, kind, meeting_outcome_id, workspace_id, ref_num, created_by_fai, repeat_freq, repeat_interval, repeat_anchor
FROM tasks WHERE column_id = $1 AND archived = false ORDER BY position;

-- name: SetTaskPosition :exec
UPDATE tasks SET position = $2, updated_at = now() WHERE id = $1;

-- name: ArchiveTask :exec
UPDATE tasks SET archived = true, updated_at = now() WHERE id = $1;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1;

-- name: SetTaskParent :exec
UPDATE tasks SET parent_task_id = sqlc.narg('parent_task_id'), updated_at = now() WHERE id = sqlc.arg('id');

-- name: SetTaskKind :exec
UPDATE tasks SET kind = sqlc.arg(kind), updated_at = now() WHERE id = sqlc.arg(id);

-- name: CreateNextRepeatTask :one
INSERT INTO tasks (board_id, column_id, workspace_id, ref_num, title, description_md, priority, due_date, position, parent_task_id, kind, repeat_freq, repeat_interval, repeat_anchor)
SELECT src.board_id, sqlc.arg(column_id), src.workspace_id, sqlc.arg(ref_num), src.title, src.description_md, src.priority, sqlc.arg(due_date)::date, sqlc.arg(position), src.parent_task_id, src.kind, src.repeat_freq, src.repeat_interval, src.repeat_anchor
FROM tasks src WHERE src.id = sqlc.arg(source_id) AND src.repeat_freq IS NOT NULL
RETURNING id, board_id, column_id, title, description_md, priority, due_date, position, completed_at, archived, created_at, updated_at, parent_task_id, kind, meeting_outcome_id, workspace_id, ref_num, created_by_fai, repeat_freq, repeat_interval, repeat_anchor;

-- name: CopyTaskTags :exec
INSERT INTO task_tags (task_id, tag_id)
SELECT sqlc.arg(new_task_id), tt.tag_id FROM task_tags tt WHERE tt.task_id = sqlc.arg(source_task_id);

-- name: ClearTaskRepeat :exec
UPDATE tasks SET repeat_freq = NULL, repeat_interval = NULL, repeat_anchor = NULL, updated_at = now() WHERE id = $1;
