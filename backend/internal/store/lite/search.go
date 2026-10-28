package lite

import (
	"context"
	"database/sql"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

// SQLite FTS5 virtual tables aren't understood by sqlc's SQLite catalog analyzer
// (confirmed: it errors "column tasks_fts does not exist" on the `table MATCH`
// idiom even with the table declared), so these two queries are hand-written
// against *sql.DB directly rather than going through sqlc like everything else.

const searchTasksSQL = `
SELECT tasks.id, tasks.board_id, tasks.column_id, tasks.parent_task_id, tasks.meeting_outcome_id,
       tasks.workspace_id, tasks.ref_num,
       tasks.title, tasks.description_md, tasks.priority, tasks.due_date, tasks.position,
       tasks.completed_at, tasks.archived, tasks.created_at, tasks.updated_at
FROM tasks_fts
JOIN tasks ON tasks.id = tasks_fts.rowid
WHERE tasks_fts MATCH ? AND tasks.archived = 0
  AND (CAST(? AS INTEGER) IS NULL OR tasks.workspace_id = ?)
ORDER BY bm25(tasks_fts)
LIMIT 20`

func (s *Store) SearchTasks(ctx context.Context, query string, workspaceID *int64) ([]store.Task, error) {
	scope := ptrToNullInt64(workspaceID)
	rows, err := s.db.QueryContext(ctx, searchTasksSQL, query, scope, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []store.Task
	for rows.Next() {
		var t sqlitedb.Task
		if err := rows.Scan(
			&t.ID, &t.BoardID, &t.ColumnID, &t.ParentTaskID, &t.MeetingOutcomeID,
			&t.WorkspaceID, &t.RefNum,
			&t.Title, &t.DescriptionMd, &t.Priority, &t.DueDate, &t.Position,
			&t.CompletedAt, &t.Archived, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, toTask(t))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if err := s.decorateTasks(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

const searchNotesSQL = `
SELECT notes.id, notes.title, notes.body_md, notes.task_id, notes.workspace_id, notes.created_at, notes.updated_at
FROM notes_fts
JOIN notes ON notes.id = notes_fts.rowid
WHERE notes_fts MATCH ?
  AND (CAST(? AS INTEGER) IS NULL OR notes.workspace_id = ?)
ORDER BY bm25(notes_fts)
LIMIT 20`

func (s *Store) SearchNotes(ctx context.Context, query string, workspaceID *int64) ([]store.Note, error) {
	scope := ptrToNullInt64(workspaceID)
	rows, err := s.db.QueryContext(ctx, searchNotesSQL, query, scope, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []store.Note
	for rows.Next() {
		var n sqlitedb.Note
		var taskID sql.NullInt64
		if err := rows.Scan(&n.ID, &n.Title, &n.BodyMd, &taskID, &n.WorkspaceID, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		n.TaskID = taskID
		out = append(out, toNote(n))
	}
	return out, rows.Err()
}
