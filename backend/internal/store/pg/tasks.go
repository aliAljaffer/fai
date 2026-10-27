package pg

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) CreateTask(ctx context.Context, p store.CreateTaskParams) (store.Task, error) {
	dueDate, err := ptrToDate(p.DueDate)
	if err != nil {
		return store.Task{}, err
	}
	t, err := s.q.CreateTask(ctx, sqlc.CreateTaskParams{
		BoardID: p.BoardID, ColumnID: p.ColumnID, Title: p.Title,
		DescriptionMd: p.DescriptionMd, Priority: p.Priority, DueDate: dueDate,
		Position: p.Position, ParentTaskID: ptrToInt8(p.ParentTaskID), MeetingOutcomeID: ptrToInt8(p.MeetingOutcomeID),
	})
	return store.Task{
		ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, Title: t.Title,
		DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
		Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
		CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}, err
}

func (s *Store) GetTask(ctx context.Context, id int64) (store.Task, error) {
	t, err := s.q.GetTask(ctx, id)
	return store.Task{
		ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, Title: t.Title,
		DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
		Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
		CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}, err
}

func (s *Store) ListTasks(ctx context.Context, p store.ListTasksParams) ([]store.Task, error) {
	rows, err := s.q.ListTasks(ctx, sqlc.ListTasksParams{
		BoardID: ptrToInt8(p.BoardID), ColumnID: ptrToInt8(p.ColumnID), Tag: ptrToText(p.Tag),
	})
	if err != nil {
		return nil, err
	}
	out := make([]store.Task, len(rows))
	for i, t := range rows {
		out[i] = store.Task{
			ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, Title: t.Title,
			DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
			Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
			CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
	}
	return out, nil
}

func (s *Store) ListSubtasks(ctx context.Context, parentTaskID int64) ([]store.Task, error) {
	rows, err := s.q.ListSubtasks(ctx, ptrToInt8(&parentTaskID))
	if err != nil {
		return nil, err
	}
	out := make([]store.Task, len(rows))
	for i, t := range rows {
		out[i] = store.Task{
			ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, Title: t.Title,
			DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
			Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
			CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
	}
	return out, nil
}

func (s *Store) UpdateTask(ctx context.Context, id int64, title, descriptionMd, priority string, dueDate *string) (store.Task, error) {
	dd, err := ptrToDate(dueDate)
	if err != nil {
		return store.Task{}, err
	}
	t, err := s.q.UpdateTask(ctx, sqlc.UpdateTaskParams{
		ID: id, Title: title, DescriptionMd: descriptionMd, Priority: priority, DueDate: dd,
	})
	return store.Task{
		ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, Title: t.Title,
		DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
		Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
		CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}, err
}

func (s *Store) UpdateTaskTitle(ctx context.Context, id int64, title string) error {
	return s.q.UpdateTaskTitle(ctx, sqlc.UpdateTaskTitleParams{ID: id, Title: title})
}

func (s *Store) MoveTask(ctx context.Context, id, columnID int64, position float64) (store.Task, error) {
	t, err := s.q.MoveTask(ctx, sqlc.MoveTaskParams{ID: id, ColumnID: columnID, Position: position})
	return store.Task{
		ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, Title: t.Title,
		DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
		Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
		CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}, err
}

func (s *Store) ListTasksByColumnOrdered(ctx context.Context, columnID int64) ([]store.Task, error) {
	rows, err := s.q.ListTasksByColumnOrdered(ctx, columnID)
	if err != nil {
		return nil, err
	}
	out := make([]store.Task, len(rows))
	for i, t := range rows {
		out[i] = store.Task{
			ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, Title: t.Title,
			DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
			Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
			CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
	}
	return out, nil
}

func (s *Store) SetTaskPosition(ctx context.Context, id int64, position float64) error {
	return s.q.SetTaskPosition(ctx, sqlc.SetTaskPositionParams{ID: id, Position: position})
}

func (s *Store) ArchiveTask(ctx context.Context, id int64) error {
	return s.q.ArchiveTask(ctx, id)
}

func (s *Store) DeleteTask(ctx context.Context, id int64) error {
	return s.q.DeleteTask(ctx, id)
}
