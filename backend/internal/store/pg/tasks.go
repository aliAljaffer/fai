package pg

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) CreateTask(ctx context.Context, p store.CreateTaskParams) (store.Task, error) {
	dueDate, err := ptrToDate(p.DueDate)
	if err != nil {
		return store.Task{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return store.Task{}, err
	}
	defer tx.Rollback(ctx)

	workspaceID, refNum, err := s.allocateRef(ctx, tx, p.BoardID)
	if err != nil {
		return store.Task{}, err
	}

	t, err := s.q.WithTx(tx).CreateTask(ctx, sqlc.CreateTaskParams{
		BoardID: p.BoardID, ColumnID: p.ColumnID, WorkspaceID: workspaceID, RefNum: refNum,
		Title:         p.Title,
		DescriptionMd: p.DescriptionMd, Priority: p.Priority, DueDate: dueDate,
		Position: p.Position, ParentTaskID: ptrToInt8(p.ParentTaskID), MeetingOutcomeID: ptrToInt8(p.MeetingOutcomeID),
	})
	if err != nil {
		return store.Task{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return store.Task{}, err
	}

	if err := s.attachBodyLabels(ctx, t.ID, t.DescriptionMd); err != nil {
		return store.Task{}, err
	}
	tasks := []store.Task{{
		ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, WorkspaceID: t.WorkspaceID, RefNum: t.RefNum, Title: t.Title,
		DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
		Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
		CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, Tags: []store.Tag{}, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}}
	if err := s.attachTags(ctx, tasks); err != nil {
		return store.Task{}, err
	}
	return tasks[0], nil
}

// allocateRef bumps the workspace counter and inserts the task in one transaction,
// so concurrent creates cannot take the same ref_num.
func (s *Store) allocateRef(ctx context.Context, tx pgx.Tx, boardID int64) (int64, int64, error) {
	var workspaceID, refNum int64
	err := tx.QueryRow(ctx, `SELECT workspace_id FROM boards WHERE id = $1`, boardID).Scan(&workspaceID)
	if err != nil {
		return 0, 0, err
	}
	err = tx.QueryRow(ctx, `UPDATE workspaces SET ref_seq = ref_seq + 1 WHERE id = $1 RETURNING ref_seq`, workspaceID).
		Scan(&refNum)
	if err != nil {
		return 0, 0, err
	}
	return workspaceID, refNum, nil
}

func (s *Store) GetTask(ctx context.Context, id int64) (store.Task, error) {
	t, err := s.q.GetTask(ctx, id)
	if err != nil {
		return store.Task{}, err
	}
	tasks := []store.Task{{
		ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, WorkspaceID: t.WorkspaceID, RefNum: t.RefNum, Title: t.Title,
		DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
		Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
		CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, Tags: []store.Tag{}, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}}
	if err := s.attachTags(ctx, tasks); err != nil {
		return store.Task{}, err
	}
	return tasks[0], nil
}

func (s *Store) ListTasks(ctx context.Context, p store.ListTasksParams) ([]store.Task, error) {
	rows, err := s.q.ListTasks(ctx, sqlc.ListTasksParams{
		BoardID: ptrToInt8(p.BoardID), ColumnID: ptrToInt8(p.ColumnID), Tag: ptrToText(p.Tag), Q: ptrToText(p.Q),
	})
	if err != nil {
		return nil, err
	}
	out := make([]store.Task, len(rows))
	for i, t := range rows {
		out[i] = store.Task{
			ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, WorkspaceID: t.WorkspaceID, RefNum: t.RefNum, Title: t.Title,
			DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
			Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
			CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, Tags: []store.Tag{}, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
	}
	if err := s.attachTags(ctx, out); err != nil {
		return nil, err
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
			ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, WorkspaceID: t.WorkspaceID, RefNum: t.RefNum, Title: t.Title,
			DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
			Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
			CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, Tags: []store.Tag{}, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
	}
	if err := s.attachTags(ctx, out); err != nil {
		return nil, err
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
	if err != nil {
		return store.Task{}, err
	}
	if err := s.attachBodyLabels(ctx, id, descriptionMd); err != nil {
		return store.Task{}, err
	}
	tasks := []store.Task{{
		ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, WorkspaceID: t.WorkspaceID, RefNum: t.RefNum, Title: t.Title,
		DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
		Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
		CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, Tags: []store.Tag{}, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}}
	if err := s.attachTags(ctx, tasks); err != nil {
		return store.Task{}, err
	}
	return tasks[0], nil
}

func (s *Store) UpdateTaskTitle(ctx context.Context, id int64, title string) error {
	return s.q.UpdateTaskTitle(ctx, sqlc.UpdateTaskTitleParams{ID: id, Title: title})
}

func (s *Store) SetTaskParent(ctx context.Context, id int64, parentTaskID *int64) error {
	return s.q.SetTaskParent(ctx, sqlc.SetTaskParentParams{ID: id, ParentTaskID: ptrToInt8(parentTaskID)})
}

func (s *Store) SetTaskCompletedAt(ctx context.Context, id int64, at time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE tasks SET completed_at = $2, updated_at = now() WHERE id = $1`, id, at)
	return err
}

func (s *Store) MoveTask(ctx context.Context, id, columnID int64, position float64) (store.Task, error) {
	t, err := s.q.MoveTask(ctx, sqlc.MoveTaskParams{ID: id, ColumnID: columnID, Position: position})
	if err != nil {
		return store.Task{}, err
	}
	tasks := []store.Task{{
		ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, WorkspaceID: t.WorkspaceID, RefNum: t.RefNum, Title: t.Title,
		DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
		Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
		CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, Tags: []store.Tag{}, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}}
	if err := s.attachTags(ctx, tasks); err != nil {
		return store.Task{}, err
	}
	return tasks[0], nil
}

func (s *Store) ListTasksByColumnOrdered(ctx context.Context, columnID int64) ([]store.Task, error) {
	rows, err := s.q.ListTasksByColumnOrdered(ctx, columnID)
	if err != nil {
		return nil, err
	}
	out := make([]store.Task, len(rows))
	for i, t := range rows {
		out[i] = store.Task{
			ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, WorkspaceID: t.WorkspaceID, RefNum: t.RefNum, Title: t.Title,
			DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: dateToPtr(t.DueDate),
			Position: t.Position, ParentTaskID: int8ToPtr(t.ParentTaskID), MeetingOutcomeID: int8ToPtr(t.MeetingOutcomeID),
			CompletedAt: tsToPtr(t.CompletedAt), Archived: t.Archived, Tags: []store.Tag{}, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
	}
	if err := s.attachTags(ctx, out); err != nil {
		return nil, err
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

// attachBodyLabels adds tags for #label tokens. Labels already on the task are
// kept, including ones set through the tag API.
// shortcut: labels are only ever added, so removing "#label" from a body does
// not detach the tag; revisit if the UI starts offering label removal from text.
func (s *Store) attachBodyLabels(ctx context.Context, taskID int64, body string) error {
	for _, label := range store.ParseBodyLabels(body) {
		var tagID int64
		err := s.pool.QueryRow(ctx, `SELECT id FROM tags WHERE lower(name) = lower($1) ORDER BY id LIMIT 1`, label).Scan(&tagID)
		if errors.Is(err, pgx.ErrNoRows) {
			if _, err := s.pool.Exec(ctx, `INSERT INTO tags (name, color) VALUES ($1, '#888888') ON CONFLICT (name) DO NOTHING`, label); err != nil {
				return err
			}
			err = s.pool.QueryRow(ctx, `SELECT id FROM tags WHERE lower(name) = lower($1) ORDER BY id LIMIT 1`, label).Scan(&tagID)
		}
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, `INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, taskID, tagID); err != nil {
			return err
		}
	}
	return nil
}
