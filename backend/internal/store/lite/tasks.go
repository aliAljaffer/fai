package lite

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toTask(t sqlitedb.Task) store.Task {
	createdAt, _ := parseTime(t.CreatedAt)
	updatedAt, _ := parseTime(t.UpdatedAt)
	return store.Task{
		ID: t.ID, BoardID: t.BoardID, ColumnID: t.ColumnID, WorkspaceID: t.WorkspaceID, RefNum: t.RefNum,
		Title:         t.Title,
		DescriptionMd: t.DescriptionMd, Priority: t.Priority, DueDate: nullStringToPtr(t.DueDate),
		Position: t.Position, ParentTaskID: nullInt64ToPtr(t.ParentTaskID), Kind: t.Kind, MeetingOutcomeID: nullInt64ToPtr(t.MeetingOutcomeID),
		CompletedAt: nullTimeToPtr(t.CompletedAt), Archived: t.Archived, Tags: []store.Tag{}, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func (s *Store) decorateTasks(ctx context.Context, tasks []store.Task) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, tag FROM workspaces`)
	if err != nil {
		return err
	}
	tagsByWorkspace := map[int64]string{}
	for rows.Next() {
		var id int64
		var tag string
		if err := rows.Scan(&id, &tag); err != nil {
			rows.Close()
			return err
		}
		tagsByWorkspace[id] = tag
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for i := range tasks {
		tags, err := s.ListTagsForTask(ctx, tasks[i].ID)
		if err != nil {
			return err
		}
		tasks[i].Tags = tags
		tasks[i].Ref = tagsByWorkspace[tasks[i].WorkspaceID] + "-" + strconv.FormatInt(tasks[i].RefNum, 10)
	}
	return nil
}

func (s *Store) CreateTask(ctx context.Context, p store.CreateTaskParams) (store.Task, error) {
	p.Kind = store.NormalizeKind(p.Kind, p.ParentTaskID != nil)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Task{}, err
	}
	defer tx.Rollback()

	workspaceID, refNum, err := allocateRef(ctx, tx, p.BoardID, p.RefNum)
	if err != nil {
		return store.Task{}, err
	}

	t, err := s.q.WithTx(tx).CreateTask(ctx, sqlitedb.CreateTaskParams{
		BoardID: p.BoardID, ColumnID: p.ColumnID, WorkspaceID: workspaceID, RefNum: refNum,
		Title:         p.Title,
		DescriptionMd: p.DescriptionMd, Priority: p.Priority, DueDate: ptrToNullString(p.DueDate),
		Position: p.Position, ParentTaskID: ptrToNullInt64(p.ParentTaskID), Kind: p.Kind, MeetingOutcomeID: ptrToNullInt64(p.MeetingOutcomeID),
	})
	if err != nil {
		return store.Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.Task{}, err
	}

	if err := s.attachBodyLabels(ctx, t.ID, t.DescriptionMd); err != nil {
		return store.Task{}, err
	}
	tasks := []store.Task{toTask(t)}
	if err := s.decorateTasks(ctx, tasks); err != nil {
		return store.Task{}, err
	}
	return tasks[0], nil
}

// allocateRef bumps the workspace counter and inserts the task in one transaction,
// so concurrent creates cannot take the same ref_num. An explicit refNum keeps a
// number from an import and only raises the counter.
func allocateRef(ctx context.Context, tx *sql.Tx, boardID int64, requested *int64) (int64, int64, error) {
	var workspaceID int64
	err := tx.QueryRowContext(ctx, `SELECT workspace_id FROM boards WHERE id = ?`, boardID).Scan(&workspaceID)
	if err != nil {
		return 0, 0, err
	}
	if requested != nil && *requested > 0 {
		refNum := *requested
		_, err := tx.ExecContext(ctx, `UPDATE workspaces SET ref_seq = CASE WHEN ref_seq < ? THEN ? ELSE ref_seq END WHERE id = ?`,
			refNum, refNum, workspaceID)
		if err != nil {
			return 0, 0, err
		}
		return workspaceID, refNum, nil
	}
	var refNum int64
	err = tx.QueryRowContext(ctx, `UPDATE workspaces SET ref_seq = ref_seq + 1 WHERE id = ? RETURNING ref_seq`, workspaceID).
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
	tasks := []store.Task{toTask(t)}
	if err := s.decorateTasks(ctx, tasks); err != nil {
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
	if err := s.decorateTasks(ctx, out); err != nil {
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
	if err := s.decorateTasks(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) UpdateTask(ctx context.Context, id int64, title, descriptionMd, priority string, dueDate *string) (store.Task, error) {
	t, err := s.q.UpdateTask(ctx, sqlitedb.UpdateTaskParams{
		ID: id, Title: title, DescriptionMd: descriptionMd, Priority: priority, DueDate: ptrToNullString(dueDate),
	})
	if err != nil {
		return store.Task{}, err
	}
	if err := s.attachBodyLabels(ctx, id, descriptionMd); err != nil {
		return store.Task{}, err
	}
	tasks := []store.Task{toTask(t)}
	if err := s.decorateTasks(ctx, tasks); err != nil {
		return store.Task{}, err
	}
	return tasks[0], nil
}

func (s *Store) PatchTask(ctx context.Context, id int64, patch store.TaskPatch) (store.Task, error) {
	dueDate := ptrToNullString(patch.DueDate)
	t, err := s.q.PatchTask(ctx, sqlitedb.PatchTaskParams{
		ID:            id,
		Title:         ptrToNullString(patch.Title),
		DescriptionMd: ptrToNullString(patch.DescriptionMd),
		Priority:      ptrToNullString(patch.Priority),
		DueDateSet:    patch.DueDateSet,
		DueDate:       dueDate,
	})
	if err != nil {
		return store.Task{}, err
	}
	if patch.DescriptionMd != nil {
		if err := s.attachBodyLabels(ctx, id, t.DescriptionMd); err != nil {
			return store.Task{}, err
		}
	}
	tasks := []store.Task{toTask(t)}
	if err := s.decorateTasks(ctx, tasks); err != nil {
		return store.Task{}, err
	}
	return tasks[0], nil
}

func (s *Store) RestoreTask(ctx context.Context, id int64) error {
	return s.q.RestoreTask(ctx, id)
}

func (s *Store) UpdateTaskTitle(ctx context.Context, id int64, title string) error {
	return s.q.UpdateTaskTitle(ctx, sqlitedb.UpdateTaskTitleParams{ID: id, Title: title})
}

func (s *Store) SetTaskParent(ctx context.Context, id int64, parentTaskID *int64) error {
	return s.q.SetTaskParent(ctx, sqlitedb.SetTaskParentParams{ID: id, ParentTaskID: ptrToNullInt64(parentTaskID)})
}

func (s *Store) SetTaskKind(ctx context.Context, id int64, kind string) error {
	return s.q.SetTaskKind(ctx, sqlitedb.SetTaskKindParams{ID: id, Kind: kind})
}

func (s *Store) SetTaskCompletedAt(ctx context.Context, id int64, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET completed_at = ?, updated_at = ? WHERE id = ?`,
		formatTime(at), formatTime(time.Now()), id)
	return err
}

func (s *Store) SetTaskTimestamps(ctx context.Context, id int64, createdAt, updatedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET created_at = ?, updated_at = ? WHERE id = ?`,
		formatTime(createdAt), formatTime(updatedAt), id)
	return err
}

func (s *Store) MoveTask(ctx context.Context, id, columnID int64, position float64) (store.Task, error) {
	t, err := s.q.MoveTask(ctx, sqlitedb.MoveTaskParams{ID: id, ColumnID: columnID, Position: position})
	if err != nil {
		return store.Task{}, err
	}
	tasks := []store.Task{toTask(t)}
	if err := s.decorateTasks(ctx, tasks); err != nil {
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

// attachBodyLabels adds tags for #label tokens. Labels already on the task are
// kept, including ones set through the tag API.
// shortcut: labels are only ever added, so removing "#label" from a body does
// not detach the tag; revisit if the UI starts offering label removal from text.
func (s *Store) attachBodyLabels(ctx context.Context, taskID int64, body string) error {
	for _, label := range store.ParseBodyLabels(body) {
		tagID, err := s.tagIDByName(ctx, label)
		if err != nil {
			return err
		}
		_, err = s.db.ExecContext(ctx, `INSERT INTO task_tags (task_id, tag_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, taskID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) tagIDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM tags WHERE lower(name) = lower(?) ORDER BY id LIMIT 1`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO tags (name, color) VALUES (?, '#888888') ON CONFLICT DO NOTHING`, name)
	if err != nil {
		return 0, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT id FROM tags WHERE lower(name) = lower(?) ORDER BY id LIMIT 1`, name).Scan(&id)
	return id, err
}
