package lite

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toNote(n sqlitedb.Note) store.Note {
	createdAt, _ := parseTime(n.CreatedAt)
	updatedAt, _ := parseTime(n.UpdatedAt)
	return store.Note{ID: n.ID, Title: n.Title, BodyMd: n.BodyMd, TaskID: nullInt64ToPtr(n.TaskID), WorkspaceID: n.WorkspaceID, CreatedAt: createdAt, UpdatedAt: updatedAt}
}

func (s *Store) ListNotes(ctx context.Context, workspaceID *int64) ([]store.Note, error) {
	rows, err := s.q.ListNotes(ctx, ptrToNullInt64(workspaceID))
	if err != nil {
		return nil, err
	}
	out := make([]store.Note, len(rows))
	for i, n := range rows {
		out[i] = toNote(n)
	}
	return out, nil
}

func (s *Store) GetNote(ctx context.Context, id int64) (store.Note, error) {
	n, err := s.q.GetNote(ctx, id)
	return toNote(n), err
}

func (s *Store) GetNoteByTitle(ctx context.Context, title string) (store.Note, error) {
	n, err := s.q.GetNoteByTitle(ctx, title)
	return toNote(n), err
}

func (s *Store) CreateNote(ctx context.Context, title, bodyMd string, taskID *int64, workspaceID int64) (store.Note, error) {
	n, err := s.q.CreateNote(ctx, sqlitedb.CreateNoteParams{Title: title, BodyMd: bodyMd, TaskID: ptrToNullInt64(taskID), WorkspaceID: workspaceID})
	return toNote(n), err
}

func (s *Store) UpdateNote(ctx context.Context, id int64, title, bodyMd string, taskID *int64) (store.Note, error) {
	n, err := s.q.UpdateNote(ctx, sqlitedb.UpdateNoteParams{ID: id, Title: title, BodyMd: bodyMd, TaskID: ptrToNullInt64(taskID)})
	return toNote(n), err
}

func (s *Store) DeleteNote(ctx context.Context, id int64) error {
	return s.q.DeleteNote(ctx, id)
}

func (s *Store) ListBacklinks(ctx context.Context, noteID int64, title string) ([]store.Note, error) {
	rows, err := s.q.ListBacklinks(ctx, sqlitedb.ListBacklinksParams{ID: noteID, Title: ptrToNullString(&title)})
	if err != nil {
		return nil, err
	}
	out := make([]store.Note, len(rows))
	for i, n := range rows {
		out[i] = toNote(n)
	}
	return out, nil
}
