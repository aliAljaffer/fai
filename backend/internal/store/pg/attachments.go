package pg

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toAttachment(a sqlc.Attachment) store.Attachment {
	return store.Attachment{
		ID: a.ID, TaskID: int8ToPtr(a.TaskID), NoteID: int8ToPtr(a.NoteID),
		Filename: a.Filename, ContentType: a.ContentType, SizeBytes: a.SizeBytes,
		StorageKey: a.StorageKey, CreatedAt: a.CreatedAt,
	}
}

func (s *Store) CreateAttachment(ctx context.Context, p store.CreateAttachmentParams) (store.Attachment, error) {
	a, err := s.q.CreateAttachment(ctx, sqlc.CreateAttachmentParams{
		TaskID: ptrToInt8(p.TaskID), NoteID: ptrToInt8(p.NoteID),
		Filename: p.Filename, ContentType: p.ContentType, SizeBytes: p.SizeBytes, StorageKey: p.StorageKey,
	})
	return toAttachment(a), err
}

func (s *Store) ListAttachmentsByTask(ctx context.Context, taskID int64) ([]store.Attachment, error) {
	rows, err := s.q.ListAttachmentsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]store.Attachment, len(rows))
	for i, a := range rows {
		out[i] = toAttachment(a)
	}
	return out, nil
}

func (s *Store) ListAttachmentsByNote(ctx context.Context, noteID int64) ([]store.Attachment, error) {
	rows, err := s.q.ListAttachmentsByNote(ctx, noteID)
	if err != nil {
		return nil, err
	}
	out := make([]store.Attachment, len(rows))
	for i, a := range rows {
		out[i] = toAttachment(a)
	}
	return out, nil
}

func (s *Store) GetAttachment(ctx context.Context, id int64) (store.Attachment, error) {
	a, err := s.q.GetAttachment(ctx, id)
	return toAttachment(a), err
}

func (s *Store) DeleteAttachment(ctx context.Context, id int64) error {
	return s.q.DeleteAttachment(ctx, id)
}
