package pg

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) ListNotes(ctx context.Context, workspaceID *int64) ([]store.Note, error) {
	rows, err := s.q.ListNotes(ctx, ptrToInt8(workspaceID))
	if err != nil {
		return nil, err
	}
	out := make([]store.Note, len(rows))
	for i, n := range rows {
		out[i] = store.Note{ID: n.ID, Title: n.Title, BodyMd: n.BodyMd, TaskID: int8ToPtr(n.TaskID), WorkspaceID: n.WorkspaceID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt}
	}
	return out, nil
}

func (s *Store) GetNote(ctx context.Context, id int64) (store.Note, error) {
	n, err := s.q.GetNote(ctx, id)
	return store.Note{ID: n.ID, Title: n.Title, BodyMd: n.BodyMd, TaskID: int8ToPtr(n.TaskID), WorkspaceID: n.WorkspaceID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt}, err
}

func (s *Store) GetNoteByTitle(ctx context.Context, title string) (store.Note, error) {
	n, err := s.q.GetNoteByTitle(ctx, title)
	return store.Note{ID: n.ID, Title: n.Title, BodyMd: n.BodyMd, TaskID: int8ToPtr(n.TaskID), WorkspaceID: n.WorkspaceID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt}, err
}

func (s *Store) CreateNote(ctx context.Context, title, bodyMd string, taskID *int64, workspaceID int64) (store.Note, error) {
	n, err := s.q.CreateNote(ctx, sqlc.CreateNoteParams{Title: title, BodyMd: bodyMd, TaskID: ptrToInt8(taskID), WorkspaceID: workspaceID})
	return store.Note{ID: n.ID, Title: n.Title, BodyMd: n.BodyMd, TaskID: int8ToPtr(n.TaskID), WorkspaceID: n.WorkspaceID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt}, err
}

func (s *Store) UpdateNote(ctx context.Context, id int64, title, bodyMd string, taskID *int64) (store.Note, error) {
	n, err := s.q.UpdateNote(ctx, sqlc.UpdateNoteParams{ID: id, Title: title, BodyMd: bodyMd, TaskID: ptrToInt8(taskID)})
	return store.Note{ID: n.ID, Title: n.Title, BodyMd: n.BodyMd, TaskID: int8ToPtr(n.TaskID), WorkspaceID: n.WorkspaceID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt}, err
}

func (s *Store) DeleteNote(ctx context.Context, id int64) error {
	return s.q.DeleteNote(ctx, id)
}

func (s *Store) ListBacklinks(ctx context.Context, noteID int64, title string) ([]store.Note, error) {
	rows, err := s.q.ListBacklinks(ctx, sqlc.ListBacklinksParams{ID: noteID, Column2: title})
	if err != nil {
		return nil, err
	}
	out := make([]store.Note, len(rows))
	for i, n := range rows {
		out[i] = store.Note{ID: n.ID, Title: n.Title, BodyMd: n.BodyMd, TaskID: int8ToPtr(n.TaskID), WorkspaceID: n.WorkspaceID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt}
	}
	return out, nil
}
