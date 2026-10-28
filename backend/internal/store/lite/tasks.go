package lite

import (
	"context"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toTask(t sqlitedb.Task) store.Task {
	createdAt, _ := parseTime(t.CreatedAt)
	updatedAt, _ := parseTime(t.UpdatedAt)
	return store.Task{
		ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, Title: t.Title,
		DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: nullStringToPtr(t.DueDate),
		Position: t.Position, ParentTaskID: nullInt64ToPtr(t.ParentTaskID), MeetingOutcomeID: nullInt64ToPtr(t.MeetingOutcomeID),
		CompletedAt: nullTimeToPtr(t.CompletedAt), Archived: t.Archived, Tags: []store.Tag{}, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func (s *Store) attachTags(ctx context.Context, tasks []store.Task) error {
	for i := range tasks {
		tags, err := s.ListTagsForTask(ctx, tasks[i].ID)
		if err != nil {
			return err
		}
		tasks[i].Tags = tags
	}
	return nil
}

func (s *Store) CreateTask(ctx context.Context, p store.CreateTaskParams) (store.Task, error) {
	t, err := s.q.CreateTask(ctx, sqlitedb.CreateTaskParams{
		BoardID: p.BoardID, ColumnID: p.ColumnID, Title: p.Title,
		DescriptionMd: p.DescriptionMd, Priority: p.Priority, DueDate: ptrToNullString(p.DueDate),
		Position: p.Position, ParentTaskID: ptrToNullInt64(p.ParentTaskID), MeetingOutcomeID: ptrToNullInt64(p.MeetingOutcomeID),
	})
	return toTask(t), err
}

func (s *Store) GetTask(ctx context.Context, id int64) (store.Task, error) {
	t, err := s.q.GetTask(ctx, id)
	if err != nil {
		return store.Task{}, err
	}
	task := toTask(t)
	tasks := []store.Task{task}
	if err := s.attachTags(ctx, tasks); err != nil {
		return store.Task{}, err
	}
	return tasks[0], nil
}

func (s *Store) ListTasks(ctx context.Context, p store.ListTasksParams) ([]store.Task, error) {
	rows, err := s.q.ListTasks(ctx, sqlitedb.ListTasksParams{
		BoardID: ptrToNullInt64(p.BoardID), ColumnID: ptrToNullInt64(p.ColumnID), Tag: ptrToNullString(p.Tag), Q: ptrToNullString(p.Q),
	})
	if err != nil {
		return nil, err
	}
	out := make([]store.Task, len(rows))
	for i, t := range rows {
		out[i] = toTask(t)
	}
	if err := s.attachTags(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListSubtasks(ctx context.Context, parentTaskID int64) ([]store.Task, error) {
	rows, err := s.q.ListSubtasks(ctx, ptrToNullInt64(&parentTaskID))
	if err != nil {
		return nil, err
	}
	out := make([]store.Task, len(rows))
	for i, t := range rows {
		out[i] = toTask(t)
	}
	if err := s.attachTags(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) UpdateTask(ctx context.Context, id int64, title, descriptionMd, priority string, dueDate *string) (store.Task, error) {
	t, err := s.q.UpdateTask(ctx, sqlitedb.UpdateTaskParams{
		ID: id, Title: title, DescriptionMd: descriptionMd, Priority: priority, DueDate: ptrToNullString(dueDate),
	})
	return toTask(t), err
}

func (s *Store) UpdateTaskTitle(ctx context.Context, id int64, title string) error {
	return s.q.UpdateTaskTitle(ctx, sqlitedb.UpdateTaskTitleParams{ID: id, Title: title})
}

func (s *Store) MoveTask(ctx context.Context, id, columnID int64, position float64) (store.Task, error) {
	t, err := s.q.MoveTask(ctx, sqlitedb.MoveTaskParams{ID: id, ColumnID: columnID, Position: position})
	return toTask(t), err
}

func (s *Store) ListTasksByColumnOrdered(ctx context.Context, columnID int64) ([]store.Task, error) {
	rows, err := s.q.ListTasksByColumnOrdered(ctx, columnID)
	if err != nil {
		return nil, err
	}
	out := make([]store.Task, len(rows))
	for i, t := range rows {
		out[i] = toTask(t)
	}
	return out, nil
}

func (s *Store) SetTaskPosition(ctx context.Context, id int64, position float64) error {
	return s.q.SetTaskPosition(ctx, sqlitedb.SetTaskPositionParams{ID: id, Position: position})
}

func (s *Store) ArchiveTask(ctx context.Context, id int64) error {
	return s.q.ArchiveTask(ctx, id)
}

func (s *Store) DeleteTask(ctx context.Context, id int64) error {
	return s.q.DeleteTask(ctx, id)
}
