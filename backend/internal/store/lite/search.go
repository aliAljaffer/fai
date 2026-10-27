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
       tasks.title, tasks.description_md, tasks.priority, tasks.due_date, tasks.position,
       tasks.completed_at, tasks.archived, tasks.created_at, tasks.updated_at
FROM tasks_fts
JOIN tasks ON tasks.id = tasks_fts.rowid
WHERE tasks_fts MATCH ? AND tasks.archived = 0
ORDER BY bm25(tasks_fts)
LIMIT 20`

func (s *Store) SearchTasks(ctx context.Context, query string) ([]store.Task, error) {
	rows, err := s.db.QueryContext(ctx, searchTasksSQL, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []store.Task
	for rows.Next() {
		var t sqlitedb.Task
		if err := rows.Scan(
			&t.ID, &t.BoardID, &t.ColumnID, &t.ParentTaskID, &t.MeetingOutcomeID,
			&t.Title, &t.DescriptionMd, &t.Priority, &t.DueDate, &t.Position,
			&t.CompletedAt, &t.Archived, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, toTask(t))
	}
	return out, rows.Err()
}

const searchNotesSQL = `
SELECT notes.id, notes.title, notes.body_md, notes.task_id, notes.created_at, notes.updated_at
FROM notes_fts
JOIN notes ON notes.id = notes_fts.rowid
WHERE notes_fts MATCH ?
ORDER BY bm25(notes_fts)
LIMIT 20`

func (s *Store) SearchNotes(ctx context.Context, query string) ([]store.Note, error) {
	rows, err := s.db.QueryContext(ctx, searchNotesSQL, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []store.Note
	for rows.Next() {
		var n sqlitedb.Note
		var taskID sql.NullInt64
		if err := rows.Scan(&n.ID, &n.Title, &n.BodyMd, &taskID, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		n.TaskID = taskID
		out = append(out, toNote(n))
	}
	return out, rows.Err()
}
