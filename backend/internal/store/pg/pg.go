// Package pg implements store.Store against Postgres, used by the kubernetes
// deployment. It's a thin adapter over the existing sqlc-generated queries,
// converting pgtype wrappers to/from the plain Go types store.Store exposes.
package pg

import (
	"context"

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
	return store.Board{ID: b.ID, Name: b.Name, CreatedAt: b.CreatedAt}
}

func (s *Store) ListBoards(ctx context.Context) ([]store.Board, error) {
	rows, err := s.q.ListBoards(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.Board, len(rows))
	for i, r := range rows {
		out[i] = toBoard(r)
	}
	return out, nil
}

func (s *Store) CreateBoard(ctx context.Context, name string) (store.Board, error) {
	b, err := s.q.CreateBoard(ctx, name)
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
