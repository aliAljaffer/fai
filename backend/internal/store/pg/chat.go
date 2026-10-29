package pg

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) CreateChatMessage(ctx context.Context, workspaceID int64, role, content string) (store.ChatMessage, error) {
	m, err := s.q.CreateChatMessage(ctx, sqlc.CreateChatMessageParams{
		WorkspaceID: workspaceID, Role: role, Content: content,
	})
	if err != nil {
		return store.ChatMessage{}, err
	}
	return store.ChatMessage{
		ID: m.ID, WorkspaceID: m.WorkspaceID, Role: m.Role, Content: m.Content, CreatedAt: m.CreatedAt,
	}, nil
}

func (s *Store) ListChatMessages(ctx context.Context, workspaceID int64, limit int32) ([]store.ChatMessage, error) {
	rows, err := s.q.ListChatMessages(ctx, sqlc.ListChatMessagesParams{
		WorkspaceID: workspaceID, Limit: limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]store.ChatMessage, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		m := rows[i]
		out = append(out, store.ChatMessage{
			ID: m.ID, WorkspaceID: m.WorkspaceID, Role: m.Role, Content: m.Content, CreatedAt: m.CreatedAt,
		})
	}
	return out, nil
}
