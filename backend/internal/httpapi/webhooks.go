package httpapi

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/auth"
	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/webhook"
)

// notifyTask tells the configured agent endpoint about a task change. It does
// nothing when no endpoint is configured, so no extra reads happen by default.
func (s *Server) notifyTask(ctx context.Context, event string, task store.Task) {
	if s.Webhooks == nil {
		return
	}
	actor := ""
	if agent, ok := auth.AgentFromContext(ctx); ok {
		actor = "agent:" + agent.Name
	}
	payload := webhook.Task{
		ID: task.ID, Ref: task.Ref, Title: task.Title, Kind: task.Kind, Priority: task.Priority,
		DueDate: task.DueDate, Completed: task.CompletedAt != nil,
		Board:     s.boardName(ctx, task.BoardID),
		Column:    s.columnName(ctx, task),
		Workspace: s.workspaceTag(ctx, task.WorkspaceID),
	}
	s.Webhooks.Publish(webhook.Event{Event: event, Actor: actor, Task: payload})
}

func (s *Server) boardName(ctx context.Context, boardID int64) string {
	boards, err := s.Store.ListBoards(ctx, nil)
	if err != nil {
		return ""
	}
	for _, board := range boards {
		if board.ID == boardID {
			return board.Name
		}
	}
	return ""
}

func (s *Server) columnName(ctx context.Context, task store.Task) string {
	columns, err := s.Store.ListColumnsByBoard(ctx, task.BoardID)
	if err != nil {
		return ""
	}
	for _, column := range columns {
		if column.ID == task.ColumnID {
			return column.Name
		}
	}
	return ""
}

func (s *Server) workspaceTag(ctx context.Context, workspaceID int64) string {
	workspace, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return ""
	}
	return workspace.Tag
}
