package lite

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toBoard(b sqlitedb.Board) store.Board {
	t, _ := parseTime(b.CreatedAt)
	return store.Board{ID: b.ID, Name: b.Name, CreatedAt: t}
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
