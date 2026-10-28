// Package pg implements store.Store against Postgres, used by the kubernetes
// deployment. It's a thin adapter over the existing sqlc-generated queries,
// converting pgtype wrappers to/from the plain Go types store.Store exposes.
package pg

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/store"
)

type Store struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

var _ store.Store = (*Store)(nil)

func New(pool *pgxpool.Pool) *Store {
	return &Store{q: sqlc.New(pool), pool: pool}
}

func (s *Store) Close() {
	s.pool.Close()
}

func toBoard(b sqlc.Board) store.Board {
	return store.Board{ID: b.ID, Name: b.Name, WorkspaceID: b.WorkspaceID, CreatedAt: b.CreatedAt}
}

func (s *Store) ListBoards(ctx context.Context, workspaceID *int64) ([]store.Board, error) {
	rows, err := s.q.ListBoards(ctx, ptrToInt8(workspaceID))
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
	b, err := s.q.CreateBoard(ctx, sqlc.CreateBoardParams{Name: name, WorkspaceID: workspaceID})
	return toBoard(b), err
}

func (s *Store) UpdateBoard(ctx context.Context, id int64, name string) (store.Board, error) {
	var b sqlc.Board
	err := s.pool.QueryRow(ctx,
		`UPDATE boards SET name = $2 WHERE id = $1 RETURNING id, name, workspace_id, created_at`, id, name).
		Scan(&b.ID, &b.Name, &b.WorkspaceID, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Board{}, store.ErrNotFound
	}
	return toBoard(b), err
}

func toColumn(c sqlc.BoardColumn) store.BoardColumn {
	return store.BoardColumn{
		ID: c.ID, BoardID: c.BoardID, Name: c.Name,
		Position: c.Position, IsDone: c.IsDone, CreatedAt: c.CreatedAt,
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
	c, err := s.q.CreateColumn(ctx, sqlc.CreateColumnParams{
		BoardID: boardID, Name: name, Position: position, IsDone: isDone,
	})
	return toColumn(c), err
}

func (s *Store) UpdateColumn(ctx context.Context, boardID, id int64, name string, isDone bool) (store.BoardColumn, error) {
	var c sqlc.BoardColumn
	err := s.pool.QueryRow(ctx,
		`UPDATE board_columns SET name = $3, is_done = $4
		 WHERE id = $2 AND board_id = $1
		 RETURNING id, board_id, name, position, is_done, created_at`,
		boardID, id, name, isDone).
		Scan(&c.ID, &c.BoardID, &c.Name, &c.Position, &c.IsDone, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.BoardColumn{}, store.ErrNotFound
	}
	return toColumn(c), err
}

func (s *Store) ReorderColumns(ctx context.Context, boardID int64, ids []int64) ([]store.BoardColumn, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Positions are unique per board, so move them out of the way before assigning.
	if _, err := tx.Exec(ctx,
		`UPDATE board_columns SET position = -position - 1 WHERE board_id = $1`, boardID); err != nil {
		return nil, err
	}
	for i, id := range ids {
		tag, err := tx.Exec(ctx,
			`UPDATE board_columns SET position = $3 WHERE id = $2 AND board_id = $1`,
			boardID, id, int32(i))
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() != 1 {
			return nil, store.ErrNotFound
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ListColumnsByBoard(ctx, boardID)
}

func toTag(t sqlc.Tag) store.Tag {
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
	t, err := s.q.CreateTag(ctx, sqlc.CreateTagParams{Name: name, Color: color})
	return toTag(t), err
}

func (s *Store) AddTaskTag(ctx context.Context, taskID, tagID int64) error {
	return s.q.AddTaskTag(ctx, sqlc.AddTaskTagParams{TaskID: taskID, TagID: tagID})
}

func (s *Store) RemoveTaskTag(ctx context.Context, taskID, tagID int64) error {
	return s.q.RemoveTaskTag(ctx, sqlc.RemoveTaskTagParams{TaskID: taskID, TagID: tagID})
}
