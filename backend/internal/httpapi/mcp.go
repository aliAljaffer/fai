package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/alialjaffer/fai/backend/internal/auth"
	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/tasks"
)

const faiInstructions = `Fai is a self-hosted task board, notes and reminders workspace.

Call get_context first in a session: it returns the workspaces, boards, columns, tags and running timer.
Refer to tasks by their ref (for example @WORK-12), never by guessing an id. Every task has a ref, and writing @WORK-12 in a note or comment links back to it.
Dates are YYYY-MM-DD. Timestamps are RFC3339 with an offset. Resolve relative dates against today's date from get_context.
"Complete" means move the task to its board's done column. Deleting archives a task; it is recoverable.
Prefer several small tasks with steps over one large task.`

const mcpDateLayout = "2006-01-02"

// taskService returns the shared task operations, building them on first use so
// a Server constructed without an explicit service still works.
func (s *Server) taskService() *tasks.Service {
	s.tasksOnce.Do(func() {
		if s.Tasks == nil {
			s.Tasks = tasks.New(s.Store)
		}
		s.tasksValue = s.Tasks
	})
	return s.tasksValue
}

// mcpServer builds the tool surface. One server serves every session; the SDK
// routes each request's session to it.
func (s *Server) mcpServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "fai", Version: "1.0.0"}, &mcp.ServerOptions{
		Instructions: faiInstructions,
	})
	s.registerBoardTools(server)
	s.registerTaskTools(server)
	s.registerItemTools(server)
	return server
}

func (s *Server) mcpHandler() http.Handler {
	server := s.mcpServer()
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute})
}

// --- views -------------------------------------------------------------------

type mcpWorkspace struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type mcpColumn struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	IsDone bool   `json:"is_done"`
}

type mcpBoard struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	Workspace string      `json:"workspace"`
	Columns   []mcpColumn `json:"columns"`
}

type mcpTask struct {
	ID          int64    `json:"id"`
	Ref         string   `json:"ref"`
	Title       string   `json:"title"`
	Kind        string   `json:"kind"`
	Priority    string   `json:"priority"`
	DueDate     *string  `json:"due_date,omitempty"`
	Board       string   `json:"board"`
	Column      string   `json:"column"`
	Workspace   string   `json:"workspace"`
	Tags        []string `json:"tags,omitempty"`
	Completed   bool     `json:"completed"`
	CreatedByAI bool     `json:"created_by_ai,omitempty"`
	Description string   `json:"description,omitempty"`
}

type mcpTaskRef struct {
	ID    int64  `json:"id"`
	Ref   string `json:"ref"`
	Title string `json:"title"`
}

type mcpNote struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	BodyMd    string `json:"body_md,omitempty"`
	TaskRef   string `json:"task_ref,omitempty"`
	Workspace string `json:"workspace"`
	UpdatedAt string `json:"updated_at"`
}

type mcpReminder struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	RemindAt  string `json:"remind_at"`
	TaskRef   string `json:"task_ref,omitempty"`
	Workspace string `json:"workspace"`
	Completed bool   `json:"completed"`
}

type mcpTag struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// index holds the board, column and workspace names used to label results, so a
// list of tasks costs one query per board instead of one per task.
type index struct {
	boards     map[int64]store.Board
	columns    map[int64]store.BoardColumn
	workspaces map[int64]store.Workspace
}

func (s *Server) loadIndex(ctx context.Context) (index, error) {
	idx := index{
		boards:     map[int64]store.Board{},
		columns:    map[int64]store.BoardColumn{},
		workspaces: map[int64]store.Workspace{},
	}
	workspaces, err := s.Store.ListWorkspaces(ctx)
	if err != nil {
		return idx, err
	}
	for _, workspace := range workspaces {
		idx.workspaces[workspace.ID] = workspace
	}
	boards, err := s.Store.ListBoards(ctx, nil)
	if err != nil {
		return idx, err
	}
	for _, board := range boards {
		idx.boards[board.ID] = board
		columns, err := s.Store.ListColumnsByBoard(ctx, board.ID)
		if err != nil {
			return idx, err
		}
		for _, column := range columns {
			idx.columns[column.ID] = column
		}
	}
	return idx, nil
}

func (idx index) workspaceTag(workspaceID int64) string {
	return idx.workspaces[workspaceID].Tag
}

func (idx index) boardName(boardID int64) string {
	return idx.boards[boardID].Name
}

func (idx index) columnName(columnID int64) string {
	return idx.columns[columnID].Name
}

func (idx index) task(task store.Task) mcpTask {
	view := mcpTask{
		ID:          task.ID,
		Ref:         task.Ref,
		Title:       task.Title,
		Kind:        task.Kind,
		Priority:    task.Priority,
		DueDate:     task.DueDate,
		Board:       idx.boardName(task.BoardID),
		Column:      idx.columnName(task.ColumnID),
		Workspace:   idx.workspaceTag(task.WorkspaceID),
		Completed:   task.CompletedAt != nil,
		CreatedByAI: task.CreatedByFai,
	}
	for _, tag := range task.Tags {
		view.Tags = append(view.Tags, tag.Name)
	}
	return view
}

// --- scoping and resolution --------------------------------------------------

// agentWorkspace returns the workspace an agent key is scoped to, or nil when
// the key may use every workspace.
func (s *Server) agentWorkspace(ctx context.Context) (*store.Workspace, error) {
	agent, ok := auth.AgentFromContext(ctx)
	if !ok || agent.WorkspaceTag == "" {
		return nil, nil
	}
	workspaces, err := s.Store.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	for _, workspace := range workspaces {
		if strings.EqualFold(workspace.Tag, agent.WorkspaceTag) {
			return &workspace, nil
		}
	}
	return nil, fmt.Errorf("this key is scoped to workspace %s, which no longer exists", agent.WorkspaceTag)
}

// resolveWorkspaceArg accepts a workspace tag or name. With no argument it
// returns the key's scope (or nil for every workspace).
func (s *Server) resolveWorkspaceArg(ctx context.Context, value string) (*store.Workspace, error) {
	value = strings.TrimSpace(value)
	scoped, err := s.agentWorkspace(ctx)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return scoped, nil
	}
	workspaces, err := s.Store.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	for _, workspace := range workspaces {
		if strings.EqualFold(workspace.Tag, value) || strings.EqualFold(workspace.Name, value) {
			if scoped != nil && scoped.ID != workspace.ID {
				return nil, fmt.Errorf("this key is scoped to workspace %s", scoped.Tag)
			}
			return &workspace, nil
		}
	}
	return nil, fmt.Errorf("no workspace named %q. Available tags: %s", value, workspaceTags(workspaces))
}

func workspaceTags(workspaces []store.Workspace) string {
	tags := make([]string, 0, len(workspaces))
	for _, workspace := range workspaces {
		tags = append(tags, workspace.Tag)
	}
	return strings.Join(tags, ", ")
}

// resolveBoardArg accepts a board name or id and enforces the key's workspace
// scope.
func (s *Server) resolveBoardArg(ctx context.Context, value string, workspace *store.Workspace) (store.Board, error) {
	var boards []store.Board
	var err error
	if workspace != nil {
		boards, err = s.Store.ListBoards(ctx, &workspace.ID)
	} else {
		boards, err = s.Store.ListBoards(ctx, nil)
	}
	if err != nil {
		return store.Board{}, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		if len(boards) == 0 {
			return store.Board{}, errors.New("no board exists yet. Create one with create_board.")
		}
		return boards[0], nil
	}
	if id, ok := parseInt64(value); ok {
		for _, board := range boards {
			if board.ID == id {
				return board, nil
			}
		}
	}
	for _, board := range boards {
		if strings.EqualFold(board.Name, value) {
			return board, nil
		}
	}
	names := make([]string, 0, len(boards))
	for _, board := range boards {
		names = append(names, board.Name)
	}
	return store.Board{}, fmt.Errorf("no board named %q. Available boards: %s", value, strings.Join(names, ", "))
}

func (s *Server) resolveTaskArg(ctx context.Context, value string) (store.Task, index, error) {
	workspace, err := s.agentWorkspace(ctx)
	if err != nil {
		return store.Task{}, index{}, err
	}
	var workspaceID *int64
	if workspace != nil {
		workspaceID = &workspace.ID
	}
	lookup, err := s.taskService().Resolve(ctx, value, workspaceID)
	if err != nil {
		return store.Task{}, index{}, err
	}
	if lookup.Task.ID == 0 {
		if len(lookup.Candidates) > 0 {
			refs := make([]string, 0, len(lookup.Candidates))
			for _, candidate := range lookup.Candidates {
				refs = append(refs, "@"+candidate.Ref+" ("+candidate.Title+")")
			}
			return store.Task{}, index{}, fmt.Errorf("%q matches more than one task: %s", value, strings.Join(refs, "; "))
		}
		return store.Task{}, index{}, fmt.Errorf("no task matches %q. Use list_tasks or search to find its ref.", value)
	}
	if workspace != nil && lookup.Task.WorkspaceID != workspace.ID {
		return store.Task{}, index{}, fmt.Errorf("this key is scoped to workspace %s", workspace.Tag)
	}
	idx, err := s.loadIndex(ctx)
	return lookup.Task, idx, err
}

func parseInt64(value string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return id, err == nil
}
