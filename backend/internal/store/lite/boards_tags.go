package lite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toBoard(b sqlitedb.Board) store.Board {
	t, _ := parseTime(b.CreatedAt)
	return store.Board{ID: b.ID, Name: b.Name, WorkspaceID: b.WorkspaceID, CreatedAt: t}
}

func (s *Store) ListBoards(ctx context.Context, workspaceID *int64) ([]store.Board, error) {
	rows, err := s.q.ListBoards(ctx, ptrToNullInt64(workspaceID))
	if err != nil {
		return nil, err
	}
	out := make([]store.Board, len(rows))
	for i, r := range rows {
		out[i] = toBoard(r)
	}
	return out, nil
}

func (s *Store) CreateBoard(ctx context.Context, name string, workspaceID int64) (store.Board, error) {
	b, err := s.q.CreateBoard(ctx, sqlitedb.CreateBoardParams{Name: name, WorkspaceID: workspaceID})
	return toBoard(b), err
}

func (s *Store) UpdateBoard(ctx context.Context, id int64, name string) (store.Board, error) {
	var b sqlitedb.Board
	err := s.db.QueryRowContext(ctx,
		`UPDATE boards SET name = ? WHERE id = ? RETURNING id, name, workspace_id, created_at`, name, id).
		Scan(&b.ID, &b.Name, &b.WorkspaceID, &b.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return store.Board{}, store.ErrNotFound
	}
	return toBoard(b), err
}

func (s *Store) DeleteBoard(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM boards WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func toColumn(c sqlitedb.BoardColumn) store.BoardColumn {
	t, _ := parseTime(c.CreatedAt)
	return store.BoardColumn{
		ID: c.ID, BoardID: c.BoardID, Name: c.Name,
		Position: int32(c.Position), IsDone: c.IsDone, CreatedAt: t,
	}
}

func (s *Store) ListColumnsByBoard(ctx context.Context, boardID int64) ([]store.BoardColumn, error) {
	rows, err := s.q.ListColumnsByBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}
	out := make([]store.BoardColumn, len(rows))
	for i, r := range rows {
		out[i] = toColumn(r)
	}
	return out, nil
}

func (s *Store) CreateColumn(ctx context.Context, boardID int64, name string, position int32, isDone bool) (store.BoardColumn, error) {
	c, err := s.q.CreateColumn(ctx, sqlitedb.CreateColumnParams{
		BoardID: boardID, Name: name, Position: int64(position), IsDone: isDone,
	})
	return toColumn(c), err
}

func (s *Store) UpdateColumn(ctx context.Context, boardID, id int64, name string, isDone bool) (store.BoardColumn, error) {
	var c sqlitedb.BoardColumn
	err := s.db.QueryRowContext(ctx,
		`UPDATE board_columns SET name = ?, is_done = ?
		 WHERE id = ? AND board_id = ?
		 RETURNING id, board_id, name, position, is_done, created_at`,
		name, isDone, id, boardID).
		Scan(&c.ID, &c.BoardID, &c.Name, &c.Position, &c.IsDone, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return store.BoardColumn{}, store.ErrNotFound
	}
	return toColumn(c), err
}

func (s *Store) ReorderColumns(ctx context.Context, boardID int64, ids []int64) ([]store.BoardColumn, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Positions are unique per board, so move them out of the way before assigning.
	if _, err := tx.ExecContext(ctx,
		`UPDATE board_columns SET position = -position - 1 WHERE board_id = ?`, boardID); err != nil {
		return nil, err
	}
	for i, id := range ids {
		res, err := tx.ExecContext(ctx,
			`UPDATE board_columns SET position = ? WHERE id = ? AND board_id = ?`, i, id, boardID)
		if err != nil {
			return nil, err
		}
		if n, err := res.RowsAffected(); err != nil {
			return nil, err
		} else if n != 1 {
			return nil, store.ErrNotFound
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.ListColumnsByBoard(ctx, boardID)
}

func toTag(t sqlitedb.Tag) store.Tag {
	return store.Tag{ID: t.ID, Name: t.Name, Color: t.Color}
}

func (s *Store) ListTags(ctx context.Context) ([]store.Tag, error) {
	rows, err := s.q.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.Tag, len(rows))
	for i, r := range rows {
		out[i] = toTag(r)
	}
	return out, nil
}

func (s *Store) CreateTag(ctx context.Context, name, color string) (store.Tag, error) {
	t, err := s.q.CreateTag(ctx, sqlitedb.CreateTagParams{Name: name, Color: color})
	return toTag(t), err
}

func (s *Store) AddTaskTag(ctx context.Context, taskID, tagID int64) error {
	return s.q.AddTaskTag(ctx, sqlitedb.AddTaskTagParams{TaskID: taskID, TagID: tagID})
}

func (s *Store) RemoveTaskTag(ctx context.Context, taskID, tagID int64) error {
	return s.q.RemoveTaskTag(ctx, sqlitedb.RemoveTaskTagParams{TaskID: taskID, TagID: tagID})
}

func (s *Store) ListTagsForTask(ctx context.Context, taskID int64) ([]store.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.color
		FROM tags t
		JOIN task_tags tt ON tt.tag_id = t.id
		WHERE tt.task_id = ?
		ORDER BY t.name`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []store.Tag{}
	for rows.Next() {
		var t store.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
