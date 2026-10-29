package pg

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) SearchTasks(ctx context.Context, query string, workspaceID *int64) ([]store.Task, error) {
	rows, err := s.q.SearchTasks(ctx, sqlc.SearchTasksParams{WebsearchToTsquery: query, WorkspaceID: ptrToInt8(workspaceID)})
	if err != nil {
		return nil, err
	}
	out := make([]store.Task, len(rows))
	for i, t := range rows {
		out[i] = store.Task{
			ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, WorkspaceID: t.WorkspaceID, RefNum: t.RefNum, Title: t.Title,
			DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
			Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), Kind: t.Kind, MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
			CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, Tags: []store.Tag{}, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
	}
	if err := s.attachTags(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) SearchNotes(ctx context.Context, query string, workspaceID *int64) ([]store.Note, error) {
	rows, err := s.q.SearchNotes(ctx, sqlc.SearchNotesParams{WebsearchToTsquery: query, WorkspaceID: ptrToInt8(workspaceID)})
	if err != nil {
		return nil, err
	}
	out := make([]store.Note, len(rows))
	for i, n := range rows {
		out[i] = store.Note{ID: n.ID, Title: n.Title, BodyMd: n.BodyMd, TaskID: int8ToPtr(n.TaskID), WorkspaceID: n.WorkspaceID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt}
	}
	return out, nil
}

func (s *Store) CompletionsByDay(ctx context.Context, from, to string) ([]store.DayCount, error) {
	fd, err := dateFromRequired(from)
	if err != nil {
		return nil, err
	}
	td, err := dateFromRequired(to)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.CompletionsByDay(ctx, sqlc.CompletionsByDayParams{FromDate: fd, ToDate: td})
	if err != nil {
		return nil, err
	}
	out := make([]store.DayCount, len(rows))
	for i, r := range rows {
		out[i] = store.DayCount{Date: dateRequired(r.Day), Count: r.Count}
	}
	return out, nil
}

func (s *Store) TimeByDay(ctx context.Context, from, to string) ([]store.DaySeconds, error) {
	fd, err := dateFromRequired(from)
	if err != nil {
		return nil, err
	}
	td, err := dateFromRequired(to)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.TimeByDay(ctx, sqlc.TimeByDayParams{FromDate: fd, ToDate: td})
	if err != nil {
		return nil, err
	}
	out := make([]store.DaySeconds, len(rows))
	for i, r := range rows {
		out[i] = store.DaySeconds{Date: dateRequired(r.Day), Seconds: r.Seconds}
	}
	return out, nil
}
