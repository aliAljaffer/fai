package pg

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toTaskComment(c sqlc.TaskComment) store.TaskComment {
	return store.TaskComment{
		ID: c.ID, TaskID: c.TaskID, Author: c.Author, BodyMd: c.BodyMd,
		ExternalID: textToPtr(c.ExternalID), CreatedAt: c.CreatedAt,
	}
}

func (s *Store) CreateTaskComment(ctx context.Context, taskID int64, author, bodyMd string, externalID *string) (store.TaskComment, error) {
	c, err := s.q.CreateTaskComment(ctx, sqlc.CreateTaskCommentParams{
		TaskID: taskID, Author: author, BodyMd: bodyMd, ExternalID: ptrToText(externalID),
	})
	return toTaskComment(c), err
}

func (s *Store) ListTaskComments(ctx context.Context, taskID int64) ([]store.TaskComment, error) {
	rows, err := s.q.ListTaskComments(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]store.TaskComment, len(rows))
	for i, c := range rows {
		out[i] = toTaskComment(c)
	}
	return out, nil
}

func (s *Store) DeleteTaskComment(ctx context.Context, taskID, commentID int64) error {
	return s.q.DeleteTaskComment(ctx, sqlc.DeleteTaskCommentParams{ID: commentID, TaskID: taskID})
}
