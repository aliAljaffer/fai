// Package tasks holds the task and board operations shared by the HTTP API,
// the in-app assistant and the MCP agent surface.
package tasks

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

// PositionGap is the spacing between adjacent tasks in a column.
const PositionGap = 1024.0

// ErrNoDoneColumn marks a board without a column marked done.
var ErrNoDoneColumn = errors.New("the board has no done column")

// ErrEmptyTitle rejects a create without a title.
var ErrEmptyTitle = errors.New("a task needs a title")

// Service runs task operations against the store.
type Service struct {
	Store store.Store
}

func New(st store.Store) *Service {
	return &Service{Store: st}
}

// Lookup is a task resolution result: either a task, or the candidates that
// matched a text query.
type Lookup struct {
	Task       store.Task
	Candidates []store.TaskRef
}

// ByID reads one task.
func (s *Service) ByID(ctx context.Context, id int64) (store.Task, error) {
	return s.Store.GetTask(ctx, id)
}

// Resolve finds a task from an id, an @TAG-12 reference, or a title query.
// A non-empty Candidates list means the query was ambiguous, and an empty
// Lookup means nothing matched.
func (s *Service) Resolve(ctx context.Context, query string, workspaceID *int64) (Lookup, error) {
	query = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(query), "@"))
	if query == "" {
		return Lookup{}, nil
	}
	if id, err := strconv.ParseInt(query, 10, 64); err == nil {
		if task, err := s.Store.GetTask(ctx, id); err == nil {
			return Lookup{Task: task}, nil
		}
	}
	refs, err := s.Store.ResolveRefs(ctx, query, workspaceID)
	if err != nil {
		return Lookup{}, err
	}
	if len(refs) > 1 {
		exact := make([]store.TaskRef, 0, len(refs))
		for _, ref := range refs {
			if strings.EqualFold(strings.TrimSpace(ref.Title), query) {
				exact = append(exact, ref)
			}
		}
		if len(exact) == 1 {
			refs = exact
		}
	}
	if len(refs) == 0 {
		return Lookup{}, nil
	}
	if len(refs) > 1 {
		return Lookup{Candidates: refs}, nil
	}
	task, err := s.Store.GetTask(ctx, refs[0].ID)
	if err != nil {
		return Lookup{}, err
	}
	return Lookup{Task: task}, nil
}

// DefaultColumn returns the first board's first column that is not done.
func (s *Service) DefaultColumn(ctx context.Context) (int64, int64, error) {
	boards, err := s.Store.ListBoards(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	if len(boards) == 0 {
		return 0, 0, errors.New("no board exists yet")
	}
	columns, err := s.Store.ListColumnsByBoard(ctx, boards[0].ID)
	if err != nil {
		return 0, 0, err
	}
	if len(columns) == 0 {
		return 0, 0, fmt.Errorf("board %q has no columns", boards[0].Name)
	}
	for _, column := range columns {
		if !column.IsDone {
			return boards[0].ID, column.ID, nil
		}
	}
	return boards[0].ID, columns[0].ID, nil
}

// Column resolves a column by name on a board. Available is filled in whenever
// the name did not match exactly one column.
func (s *Service) Column(ctx context.Context, boardID int64, name string) (store.BoardColumn, []store.BoardColumn, bool, error) {
	columns, err := s.Store.ListColumnsByBoard(ctx, boardID)
	if err != nil {
		return store.BoardColumn{}, nil, false, err
	}
	name = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(name), "#"))
	if name == "" {
		return store.BoardColumn{}, columns, false, nil
	}
	matches := make([]store.BoardColumn, 0, 1)
	for _, column := range columns {
		if strings.EqualFold(strings.TrimSpace(column.Name), name) {
			matches = append(matches, column)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil, true, nil
	}
	if len(matches) > 1 {
		return store.BoardColumn{}, matches, false, nil
	}
	return store.BoardColumn{}, columns, false, nil
}

// DoneColumn returns the column marked done on a board.
func (s *Service) DoneColumn(ctx context.Context, boardID int64) (store.BoardColumn, bool, error) {
	columns, err := s.Store.ListColumnsByBoard(ctx, boardID)
	if err != nil {
		return store.BoardColumn{}, false, err
	}
	for _, column := range columns {
		if column.IsDone {
			return column, true, nil
		}
	}
	return store.BoardColumn{}, false, nil
}

// AppendPosition returns the position for a new task at the end of columnID.
func (s *Service) AppendPosition(ctx context.Context, columnID int64) (float64, error) {
	existing, err := s.Store.ListTasksByColumnOrdered(ctx, columnID)
	if err != nil {
		return 0, err
	}
	if len(existing) == 0 {
		return PositionGap, nil
	}
	return existing[len(existing)-1].Position + PositionGap, nil
}

// CreateParams describes a new task. Steps become children of the new task.
type CreateParams struct {
	BoardID       int64
	ColumnID      int64
	Title         string
	DescriptionMd string
	Priority      string
	Due           *string
	ParentID      *int64
	Kind          string
	Repeat        *store.TaskRepeat
	CreatedByFai  bool
	Steps         []string
}

// Create inserts a task at the end of its column and then its children.
func (s *Service) Create(ctx context.Context, p CreateParams) (store.Task, error) {
	title := strings.TrimSpace(p.Title)
	if title == "" {
		return store.Task{}, ErrEmptyTitle
	}
	position, err := s.AppendPosition(ctx, p.ColumnID)
	if err != nil {
		return store.Task{}, err
	}
	priority := strings.TrimSpace(p.Priority)
	if priority == "" {
		priority = "medium"
	}
	task, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
		BoardID:       p.BoardID,
		ColumnID:      p.ColumnID,
		Title:         title,
		DescriptionMd: p.DescriptionMd,
		Priority:      priority,
		DueDate:       p.Due,
		Position:      position,
		ParentTaskID:  p.ParentID,
		Kind:          p.Kind,
		CreatedByFai:  p.CreatedByFai,
		Repeat:        p.Repeat,
	})
	if err != nil {
		return store.Task{}, err
	}
	if _, err := s.CreateChildren(ctx, task, p.Steps); err != nil {
		return store.Task{}, err
	}
	return task, nil
}

// CreateChildren adds steps to a task or tasks to an arc.
func (s *Service) CreateChildren(ctx context.Context, parent store.Task, titles []string) (int, error) {
	kind := "step"
	if parent.Kind == "arc" {
		kind = "task"
	}
	created := 0
	// Each insert reads the column's last position, so the loop stays sequential.
	for _, title := range CleanTitles(titles) {
		position, err := s.AppendPosition(ctx, parent.ColumnID)
		if err != nil {
			return created, err
		}
		if _, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
			BoardID: parent.BoardID, ColumnID: parent.ColumnID, Title: title, Priority: "medium",
			Position: position, ParentTaskID: &parent.ID, Kind: kind, CreatedByFai: true,
		}); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

// CleanTitles trims and drops empty titles.
func CleanTitles(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		if title := strings.TrimSpace(value); title != "" {
			out = append(out, title)
		}
	}
	return out
}

// Move places a task at the end of a column.
func (s *Service) Move(ctx context.Context, task store.Task, column store.BoardColumn, now time.Time) (store.Task, error) {
	position, err := s.AppendPosition(ctx, column.ID)
	if err != nil {
		return store.Task{}, err
	}
	return s.MoveTo(ctx, task, column.ID, position, now)
}

// MoveTo places a task at an explicit position. Moving a repeating task into
// the done column rolls it to its next due date.
func (s *Service) MoveTo(ctx context.Context, task store.Task, columnID int64, position float64, now time.Time) (store.Task, error) {
	moved, err := s.Store.MoveTask(ctx, task.ID, columnID, position)
	if err != nil {
		return store.Task{}, err
	}
	state := task.RepeatState()
	if state == nil || task.CompletedAt != nil || moved.CompletedAt == nil || task.DueDate == nil {
		return moved, nil
	}
	if err := s.rollRepeatingTask(ctx, task, *state, now); err != nil {
		return store.Task{}, err
	}
	moved.Repeat, moved.RepeatAnchor = nil, nil
	return moved, nil
}

// Complete moves a task into its board's done column.
func (s *Service) Complete(ctx context.Context, task store.Task, now time.Time) (store.Task, store.BoardColumn, error) {
	column, found, err := s.DoneColumn(ctx, task.BoardID)
	if err != nil {
		return store.Task{}, store.BoardColumn{}, err
	}
	if !found {
		return store.Task{}, store.BoardColumn{}, ErrNoDoneColumn
	}
	moved, err := s.Move(ctx, task, column, now)
	if err != nil {
		return store.Task{}, store.BoardColumn{}, err
	}
	return moved, column, nil
}

func (s *Service) rollRepeatingTask(ctx context.Context, before store.Task, state store.TaskRepeat, now time.Time) error {
	due, err := nextTaskDue(state, *before.DueDate, now)
	if err != nil {
		return err
	}
	position, err := s.AppendPosition(ctx, before.ColumnID)
	if err != nil {
		return err
	}
	_, err = s.Store.RollRepeatingTask(ctx, store.RollTaskParams{
		TaskID: before.ID, BoardID: before.BoardID, ColumnID: before.ColumnID, Position: position, DueDate: due,
	})
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	return err
}
