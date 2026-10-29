package lite

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) CreateChatMessage(ctx context.Context, workspaceID int64, role, content string) (store.ChatMessage, error) {
	m, err := s.q.CreateChatMessage(ctx, sqlitedb.CreateChatMessageParams{
		WorkspaceID: workspaceID, Role: role, Content: content,
	})
	if err != nil {
		return store.ChatMessage{}, err
	}
	created, _ := parseTime(m.CreatedAt)
	return store.ChatMessage{
		ID: m.ID, WorkspaceID: m.WorkspaceID, Role: m.Role, Content: m.Content, CreatedAt: created,
	}, nil
}

func (s *Store) ListChatMessages(ctx context.Context, workspaceID int64, limit int32) ([]store.ChatMessage, error) {
	rows, err := s.q.ListChatMessages(ctx, sqlitedb.ListChatMessagesParams{
		WorkspaceID: workspaceID, Limit: int64(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]store.ChatMessage, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		m := rows[i]
		created, _ := parseTime(m.CreatedAt)
		out = append(out, store.ChatMessage{
			ID: m.ID, WorkspaceID: m.WorkspaceID, Role: m.Role, Content: m.Content, CreatedAt: created,
		})
	}
	return out, nil
}
