package lite

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toTaskComment(c sqlitedb.TaskComment) store.TaskComment {
	createdAt, _ := parseTime(c.CreatedAt)
	return store.TaskComment{
		ID: c.ID, TaskID: c.TaskID, Author: c.Author, BodyMd: c.BodyMd,
		ExternalID: nullStringToPtr(c.ExternalID), CreatedAt: createdAt,
	}
}

func (s *Store) CreateTaskComment(ctx context.Context, taskID int64, author, bodyMd string, externalID *string) (store.TaskComment, error) {
	c, err := s.q.CreateTaskComment(ctx, sqlitedb.CreateTaskCommentParams{
		TaskID: taskID, Author: author, BodyMd: bodyMd, ExternalID: ptrToNullString(externalID),
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
	return s.q.DeleteTaskComment(ctx, sqlitedb.DeleteTaskCommentParams{ID: commentID, TaskID: taskID})
}
