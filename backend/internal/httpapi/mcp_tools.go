package httpapi

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/alialjaffer/fai/backend/internal/auth"
	"github.com/alialjaffer/fai/backend/internal/recurrence"
	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/tasks"
)

// --- boards and workspaces ---------------------------------------------------

type getContextIn struct{}

type runningTimer struct {
	Task    string `json:"task"`
	Since   string `json:"since"`
	Seconds int64  `json:"seconds"`
}

type getContextOut struct {
	Agent      string         `json:"agent"`
	Today      string         `json:"today"`
	ServerTime string         `json:"server_time"`
	Workspaces []mcpWorkspace `json:"workspaces"`
	Boards     []mcpBoard     `json:"boards"`
	Tags       []mcpTag       `json:"tags"`
	Running    *runningTimer  `json:"running_timer,omitempty"`
}

type listBoardsIn struct {
	Workspace string `json:"workspace,omitempty" jsonschema:"workspace tag or name; defaults to this key's workspace"`
}

type boardListOut struct {
	Boards []mcpBoard `json:"boards"`
}

type createBoardIn struct {
	Name      string `json:"name"`
	Workspace string `json:"workspace,omitempty" jsonschema:"workspace tag or name; defaults to the first workspace"`
}

type createColumnIn struct {
	Board  string `json:"board" jsonschema:"board name or id"`
	Name   string `json:"name"`
	IsDone bool   `json:"is_done,omitempty" jsonschema:"true when this column means completed"`
}

type createWorkspaceIn struct {
	Name string `json:"name"`
	Tag  string `json:"tag" jsonschema:"3 or 4 letters, uppercase, unique; used in task refs like WORK-12"`
}

type createTagIn struct {
	Name      string `json:"name"`
	Color     string `json:"color,omitempty" jsonschema:"hex color such as #888888"`
	Workspace string `json:"workspace,omitempty"`
}

func (s *Server) registerBoardTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_context",
		Description: "Start here. Returns today's date, the agent name, workspaces, boards with their columns, tags, and any running timer.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ getContextIn) (*mcp.CallToolResult, getContextOut, error) {
		workspace, err := s.resolveWorkspaceArg(ctx, "")
		if err != nil {
			return nil, getContextOut{}, err
		}
		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, getContextOut{}, err
		}
		out := getContextOut{
			Today:      time.Now().Format(mcpDateLayout),
			ServerTime: time.Now().Format(time.RFC3339),
		}
		if agent, ok := auth.AgentFromContext(ctx); ok {
			out.Agent = agent.Name
		}
		for _, item := range idx.workspaces {
			if workspace != nil && item.ID != workspace.ID {
				continue
			}
			out.Workspaces = append(out.Workspaces, mcpWorkspace{ID: item.ID, Name: item.Name, Tag: item.Tag})
		}
		sort.Slice(out.Workspaces, func(i, j int) bool { return out.Workspaces[i].Tag < out.Workspaces[j].Tag })
		for _, board := range idx.boards {
			if workspace != nil && board.WorkspaceID != workspace.ID {
				continue
			}
			out.Boards = append(out.Boards, s.boardView(board, idx))
		}
		sort.Slice(out.Boards, func(i, j int) bool { return out.Boards[i].ID < out.Boards[j].ID })
		tags, err := s.Store.ListTags(ctx, workspaceIDOf(workspace))
		if err != nil {
			return nil, getContextOut{}, err
		}
		for _, tag := range tags {
			out.Tags = append(out.Tags, mcpTag{ID: tag.ID, Name: tag.Name, Color: tag.Color})
		}
		if entry, err := s.Store.GetRunningEntry(ctx); err == nil && entry != nil {
			task, err := s.Store.GetTask(ctx, entry.TaskID)
			if err == nil && (workspace == nil || task.WorkspaceID == workspace.ID) {
				out.Running = &runningTimer{
					Task: "@" + task.Ref + " " + task.Title, Since: entry.StartedAt.Format(time.RFC3339),
					Seconds: int64(time.Since(entry.StartedAt).Seconds()),
				}
			}
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_boards",
		Description: "List boards with their columns. Use a board and column name when creating or moving tasks.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listBoardsIn) (*mcp.CallToolResult, boardListOut, error) {
		workspace, err := s.resolveWorkspaceArg(ctx, in.Workspace)
		if err != nil {
			return nil, boardListOut{}, err
		}
		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, boardListOut{}, err
		}
		out := boardListOut{Boards: []mcpBoard{}}
		for _, board := range idx.boards {
			if workspace != nil && board.WorkspaceID != workspace.ID {
				continue
			}
			out.Boards = append(out.Boards, s.boardView(board, idx))
		}
		sort.Slice(out.Boards, func(i, j int) bool { return out.Boards[i].ID < out.Boards[j].ID })
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_board",
		Description: "Create a board in a workspace.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createBoardIn) (*mcp.CallToolResult, mcpBoard, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, mcpBoard{}, fmt.Errorf("name is required")
		}
		workspace, err := s.resolveWorkspaceArg(ctx, in.Workspace)
		if err != nil {
			return nil, mcpBoard{}, err
		}
		if workspace == nil {
			first, err := s.defaultWorkspace(ctx)
			if err != nil {
				return nil, mcpBoard{}, err
			}
			workspace = &first
		}
		board, err := s.Store.CreateBoard(ctx, name, workspace.ID)
		if err != nil {
			return nil, mcpBoard{}, err
		}
		s.broadcast("boards")
		return nil, mcpBoard{ID: board.ID, Name: board.Name, Workspace: workspace.Tag}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_column",
		Description: "Add a column to a board. Mark is_done on the column that means completed.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createColumnIn) (*mcp.CallToolResult, mcpColumn, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, mcpColumn{}, fmt.Errorf("name is required")
		}
		workspace, err := s.resolveWorkspaceArg(ctx, "")
		if err != nil {
			return nil, mcpColumn{}, err
		}
		board, err := s.resolveBoardArg(ctx, in.Board, workspace)
		if err != nil {
			return nil, mcpColumn{}, err
		}
		existing, err := s.Store.ListColumnsByBoard(ctx, board.ID)
		if err != nil {
			return nil, mcpColumn{}, err
		}
		position := int32(len(existing) + 1)
		column, err := s.Store.CreateColumn(ctx, board.ID, name, position, in.IsDone)
		if err != nil {
			return nil, mcpColumn{}, err
		}
		s.broadcast("boards")
		return nil, mcpColumn{ID: column.ID, Name: column.Name, IsDone: column.IsDone}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_workspace",
		Description: "Create a workspace. Its tag prefixes task refs, for example WORK-12.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createWorkspaceIn) (*mcp.CallToolResult, mcpWorkspace, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, mcpWorkspace{}, fmt.Errorf("name is required")
		}
		if workspace, err := s.agentWorkspace(ctx); err != nil {
			return nil, mcpWorkspace{}, err
		} else if workspace != nil {
			return nil, mcpWorkspace{}, fmt.Errorf("this key is scoped to workspace %s, so it cannot create another", workspace.Tag)
		}
		if !store.ValidWorkspaceTag(strings.TrimSpace(in.Tag)) {
			return nil, mcpWorkspace{}, fmt.Errorf("tag must be 3 or 4 letters")
		}
		workspace, err := s.Store.CreateWorkspace(ctx, name, in.Tag)
		if err != nil {
			return nil, mcpWorkspace{}, err
		}
		s.broadcast("workspaces")
		return nil, mcpWorkspace{ID: workspace.ID, Name: workspace.Name, Tag: workspace.Tag}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_tag",
		Description: "Create a tag in a workspace, so tasks can be grouped across boards.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createTagIn) (*mcp.CallToolResult, mcpTag, error) {
		names := sanitiseLabels([]string{in.Name})
		if len(names) == 0 {
			return nil, mcpTag{}, fmt.Errorf("name is required")
		}
		name := names[0]
		workspace, err := s.resolveWorkspaceArg(ctx, in.Workspace)
		if err != nil {
			return nil, mcpTag{}, err
		}
		if workspace == nil {
			first, err := s.defaultWorkspace(ctx)
			if err != nil {
				return nil, mcpTag{}, err
			}
			workspace = &first
		}
		color := strings.TrimSpace(in.Color)
		if color == "" {
			color = "#888888"
		}
		tag, err := s.Store.CreateTag(ctx, workspace.ID, name, color)
		if err != nil {
			return nil, mcpTag{}, err
		}
		s.broadcast("tags")
		return nil, mcpTag{ID: tag.ID, Name: tag.Name, Color: tag.Color}, nil
	})
}

func (s *Server) boardView(board store.Board, idx index) mcpBoard {
	view := mcpBoard{ID: board.ID, Name: board.Name, Workspace: idx.workspaceTag(board.WorkspaceID)}
	for _, column := range idx.columns {
		if column.BoardID != board.ID {
			continue
		}
		view.Columns = append(view.Columns, mcpColumn{ID: column.ID, Name: column.Name, IsDone: column.IsDone})
	}
	sort.Slice(view.Columns, func(i, j int) bool { return view.Columns[i].ID < view.Columns[j].ID })
	return view
}

func workspaceIDOf(workspace *store.Workspace) *int64 {
	if workspace == nil {
		return nil
	}
	return &workspace.ID
}

// defaultWorkspace returns the key's scope, or the first workspace.
func (s *Server) defaultWorkspace(ctx context.Context) (store.Workspace, error) {
	scoped, err := s.agentWorkspace(ctx)
	if err != nil {
		return store.Workspace{}, err
	}
	if scoped != nil {
		return *scoped, nil
	}
	workspaces, err := s.Store.ListWorkspaces(ctx)
	if err != nil {
		return store.Workspace{}, err
	}
	if len(workspaces) == 0 {
		return store.Workspace{}, fmt.Errorf("no workspace exists yet. Create one with create_workspace.")
	}
	return workspaces[0], nil
}

// --- tasks -------------------------------------------------------------------

type listTasksIn struct {
	Workspace        string `json:"workspace,omitempty" jsonschema:"workspace tag or name; defaults to this key's workspace"`
	Board            string `json:"board,omitempty" jsonschema:"board name or id"`
	Column           string `json:"column,omitempty" jsonschema:"column name; requires board"`
	Query            string `json:"query,omitempty" jsonschema:"text matched against title and description"`
	Tag              string `json:"tag,omitempty"`
	IncludeCompleted bool   `json:"include_completed,omitempty"`
	Limit            int    `json:"limit,omitempty" jsonschema:"maximum tasks to return, default 50, maximum 200"`
}

type taskListOut struct {
	Count int       `json:"count"`
	Tasks []mcpTask `json:"tasks"`
}

type getTaskIn struct {
	Task string `json:"task" jsonschema:"task ref such as @WORK-12, a plain ref, an id, or a title"`
}

type commentView struct {
	Author    string `json:"author,omitempty"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type referenceView struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Ref   string `json:"ref,omitempty"`
}

type taskDetail struct {
	Task        mcpTask         `json:"task"`
	Subtasks    []mcpTask       `json:"subtasks,omitempty"`
	Comments    []commentView   `json:"comments,omitempty"`
	MentionedIn []referenceView `json:"mentioned_in,omitempty"`
}

type mcpRepeat struct {
	Freq     string `json:"freq" jsonschema:"daily, weekly, monthly or yearly"`
	Interval int    `json:"interval,omitempty" jsonschema:"periods between repeats, 1 or more. Omit for 1"`
}

type createTaskIn struct {
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty" jsonschema:"Markdown description"`
	Workspace   string     `json:"workspace,omitempty"`
	Board       string     `json:"board,omitempty" jsonschema:"board name or id; defaults to the first board"`
	Column      string     `json:"column,omitempty" jsonschema:"column name; defaults to the board's first column that is not done"`
	DueDate     string     `json:"due_date,omitempty" jsonschema:"YYYY-MM-DD"`
	Priority    string     `json:"priority,omitempty" jsonschema:"low, medium or high"`
	Kind        string     `json:"kind,omitempty" jsonschema:"task or arc. An arc groups tasks; use steps for subtasks"`
	Steps       []string   `json:"steps,omitempty" jsonschema:"titles of steps (subtasks) to create under the new task"`
	Repeat      *mcpRepeat `json:"repeat,omitempty"`
	Parent      string     `json:"parent,omitempty" jsonschema:"parent task ref; creates a step under it and ignores board and column"`
}

type updateTaskIn struct {
	Task        string     `json:"task"`
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty" jsonschema:"replaces the whole description"`
	Priority    *string    `json:"priority,omitempty" jsonschema:"low, medium or high"`
	DueDate     *string    `json:"due_date,omitempty" jsonschema:"YYYY-MM-DD, or an empty string to clear"`
	Repeat      *mcpRepeat `json:"repeat,omitempty" jsonschema:"omit to keep the current rule"`
}

type moveTaskIn struct {
	Task   string `json:"task"`
	Column string `json:"column" jsonschema:"column name on the task's board"`
}

type completeTaskIn struct {
	Task string `json:"task"`
}

type deleteTaskIn struct {
	Task string `json:"task"`
}

type deleteTaskOut struct {
	Ref      string `json:"ref"`
	Archived bool   `json:"archived"`
}

type addStepsIn struct {
	Task  string   `json:"task"`
	Steps []string `json:"steps"`
}

type addStepsOut struct {
	Ref   string `json:"ref"`
	Added int    `json:"added"`
}

type addCommentIn struct {
	Task string `json:"task"`
	Body string `json:"body" jsonschema:"Markdown comment"`
}

type addCommentOut struct {
	Ref       string `json:"ref"`
	CommentID int64  `json:"comment_id"`
}

type tagTaskIn struct {
	Task   string   `json:"task"`
	Add    []string `json:"add,omitempty" jsonschema:"tag names without #; missing tags are created"`
	Remove []string `json:"remove,omitempty"`
}

type tagTaskOut struct {
	Ref  string   `json:"ref"`
	Tags []string `json:"tags"`
}

type searchIn struct {
	Query     string `json:"query"`
	Workspace string `json:"workspace,omitempty"`
}

type searchOut struct {
	Tasks []mcpTask `json:"tasks"`
	Notes []mcpNote `json:"notes"`
}

func (s *Server) registerTaskTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tasks",
		Description: "List tasks on a board or across a workspace. Completed tasks are omitted unless include_completed is set.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listTasksIn) (*mcp.CallToolResult, taskListOut, error) {
		workspace, err := s.resolveWorkspaceArg(ctx, in.Workspace)
		if err != nil {
			return nil, taskListOut{}, err
		}
		params := store.ListTasksParams{
			Tag: optionalText(in.Tag),
			Q:   optionalText(in.Query),
		}
		var boards []store.Board
		scoped := workspace != nil || strings.TrimSpace(in.Board) != ""
		if strings.TrimSpace(in.Board) != "" {
			board, err := s.resolveBoardArg(ctx, in.Board, workspace)
			if err != nil {
				return nil, taskListOut{}, err
			}
			boards = []store.Board{board}
			params.BoardID = &board.ID
		} else if workspace != nil {
			boards, err = s.Store.ListBoards(ctx, &workspace.ID)
			if err != nil {
				return nil, taskListOut{}, err
			}
		}
		if strings.TrimSpace(in.Column) != "" {
			if len(boards) != 1 {
				return nil, taskListOut{}, fmt.Errorf("name a board when filtering by column")
			}
			column, available, found, err := s.taskService().Column(ctx, boards[0].ID, in.Column)
			if err != nil {
				return nil, taskListOut{}, err
			}
			if !found {
				return nil, taskListOut{}, fmt.Errorf("no column named %q on %s. Columns: %s", in.Column, boards[0].Name, columnNames(available))
			}
			params.ColumnID = &column.ID
		}

		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, taskListOut{}, err
		}
		out := taskListOut{Tasks: []mcpTask{}}
		limit := in.Limit
		if limit <= 0 {
			limit = 50
		}
		if limit > 200 {
			limit = 200
		}
		collect := func(found []store.Task) {
			for _, task := range found {
				if !in.IncludeCompleted && task.CompletedAt != nil {
					continue
				}
				out.Tasks = append(out.Tasks, idx.task(task))
			}
		}
		// One query per board keeps a workspace-wide list bounded; boards are few.
		// A workspace with no boards returns nothing rather than everything.
		if scoped {
			for _, board := range boards {
				boardParams := params
				boardParams.BoardID = &board.ID
				found, err := s.Store.ListTasks(ctx, boardParams)
				if err != nil {
					return nil, taskListOut{}, err
				}
				collect(found)
			}
		} else {
			found, err := s.Store.ListTasks(ctx, params)
			if err != nil {
				return nil, taskListOut{}, err
			}
			collect(found)
		}
		sort.SliceStable(out.Tasks, func(i, j int) bool {
			if out.Tasks[i].DueDate == nil || out.Tasks[j].DueDate == nil {
				return out.Tasks[i].DueDate != nil
			}
			return *out.Tasks[i].DueDate < *out.Tasks[j].DueDate
		})
		if len(out.Tasks) > limit {
			out.Tasks = out.Tasks[:limit]
		}
		out.Count = len(out.Tasks)
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_task",
		Description: "Read one task with its description, steps, comments and the items that mention it.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in getTaskIn) (*mcp.CallToolResult, taskDetail, error) {
		task, idx, err := s.resolveTaskArg(ctx, in.Task)
		if err != nil {
			return nil, taskDetail{}, err
		}
		detail := taskDetail{Task: idx.task(task)}
		detail.Task.Description = task.DescriptionMd
		subtasks, err := s.Store.ListSubtasks(ctx, task.ID)
		if err != nil {
			return nil, taskDetail{}, err
		}
		for _, subtask := range subtasks {
			detail.Subtasks = append(detail.Subtasks, idx.task(subtask))
		}
		comments, err := s.Store.ListTaskComments(ctx, task.ID)
		if err != nil {
			return nil, taskDetail{}, err
		}
		for _, comment := range comments {
			detail.Comments = append(detail.Comments, commentView{
				Author: comment.Author, Body: comment.BodyMd, CreatedAt: comment.CreatedAt.Format(time.RFC3339),
			})
		}
		references, err := s.Store.ListTaskReferences(ctx, task.ID)
		if err != nil {
			return nil, taskDetail{}, err
		}
		for _, reference := range references {
			detail.MentionedIn = append(detail.MentionedIn, referenceView{Kind: reference.Kind, Title: reference.Title})
		}
		return nil, detail, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_task",
		Description: "Create a task, optionally with steps, a due date, a repeat rule, or under a parent.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createTaskIn) (*mcp.CallToolResult, mcpTask, error) {
		title := strings.TrimSpace(in.Title)
		if title == "" {
			return nil, mcpTask{}, fmt.Errorf("title is required")
		}
		due, err := mcpDueDate(in.DueDate)
		if err != nil {
			return nil, mcpTask{}, err
		}
		var parent *store.Task
		if strings.TrimSpace(in.Parent) != "" {
			found, _, err := s.resolveTaskArg(ctx, in.Parent)
			if err != nil {
				return nil, mcpTask{}, err
			}
			parent = &found
		}
		kind := strings.TrimSpace(in.Kind)
		if kind == "" {
			kind = "task"
		}
		if parent != nil {
			kind = "step"
		}
		if kind != "task" && kind != "arc" && kind != "step" {
			return nil, mcpTask{}, fmt.Errorf("kind must be task, arc or step")
		}
		var repeat *store.TaskRepeat
		if in.Repeat != nil {
			change, err := mcpRepeatChange(in.Repeat)
			if err != nil {
				return nil, mcpTask{}, err
			}
			if repeat, err = taskCreateRepeat(change, kind, due); err != nil {
				return nil, mcpTask{}, err
			}
		}

		params := tasks.CreateParams{
			Title: title, DescriptionMd: in.Description, Due: due, Kind: kind,
			Repeat: repeat, CreatedByFai: true, Steps: in.Steps,
		}
		if in.Priority != "" {
			priority := strings.ToLower(strings.TrimSpace(in.Priority))
			if priority != "low" && priority != "medium" && priority != "high" {
				return nil, mcpTask{}, fmt.Errorf("priority must be low, medium or high")
			}
			params.Priority = priority
		}
		if parent != nil {
			params.BoardID, params.ColumnID = parent.BoardID, parent.ColumnID
			params.ParentID = &parent.ID
		} else {
			workspace, err := s.resolveWorkspaceArg(ctx, in.Workspace)
			if err != nil {
				return nil, mcpTask{}, err
			}
			board, err := s.resolveBoardArg(ctx, in.Board, workspace)
			if err != nil {
				return nil, mcpTask{}, err
			}
			params.BoardID = board.ID
			column, err := s.columnForCreate(ctx, board, in.Column)
			if err != nil {
				return nil, mcpTask{}, err
			}
			params.ColumnID = column.ID
		}
		task, err := s.taskService().Create(ctx, params)
		if err != nil {
			return nil, mcpTask{}, err
		}
		s.broadcast("tasks")
		s.notifyTask(ctx, "task.created", task)
		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, mcpTask{}, err
		}
		return nil, idx.task(task), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_task",
		Description: "Change a task's title, description, priority, due date or repeat rule. Omitted fields stay as they are.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in updateTaskIn) (*mcp.CallToolResult, mcpTask, error) {
		task, idx, err := s.resolveTaskArg(ctx, in.Task)
		if err != nil {
			return nil, mcpTask{}, err
		}
		patch := store.TaskPatch{Title: trimmedPtr(in.Title), DescriptionMd: in.Description}
		if in.Priority != nil {
			priority := strings.ToLower(strings.TrimSpace(*in.Priority))
			if priority != "low" && priority != "medium" && priority != "high" {
				return nil, mcpTask{}, fmt.Errorf("priority must be low, medium or high")
			}
			patch.Priority = &priority
		}
		if in.DueDate != nil {
			due, err := mcpDueDate(*in.DueDate)
			if err != nil {
				return nil, mcpTask{}, err
			}
			patch.DueDate, patch.DueDateSet = due, true
		}
		change := repeatChange{}
		if in.Repeat != nil {
			if change, err = mcpRepeatChange(in.Repeat); err != nil {
				return nil, mcpTask{}, err
			}
		}
		if patch.Title == nil && patch.DescriptionMd == nil && patch.Priority == nil && !patch.DueDateSet && !change.set {
			return nil, mcpTask{}, fmt.Errorf("nothing to update")
		}
		if err := applyTaskRepeat(task, &patch, change); err != nil {
			return nil, mcpTask{}, err
		}
		updated, err := s.Store.PatchTask(ctx, task.ID, patch)
		if err != nil {
			return nil, mcpTask{}, err
		}
		s.broadcast("tasks")
		s.notifyTask(ctx, "task.updated", updated)
		return nil, idx.task(updated), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "move_task",
		Description: "Move a task to another column on its board. Moving into the done column completes it.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in moveTaskIn) (*mcp.CallToolResult, mcpTask, error) {
		task, idx, err := s.resolveTaskArg(ctx, in.Task)
		if err != nil {
			return nil, mcpTask{}, err
		}
		column, available, found, err := s.taskService().Column(ctx, task.BoardID, in.Column)
		if err != nil {
			return nil, mcpTask{}, err
		}
		if !found {
			return nil, mcpTask{}, fmt.Errorf("no column named %q. Columns: %s", in.Column, columnNames(available))
		}
		moved, err := s.taskService().Move(ctx, task, column, time.Now())
		if err != nil {
			return nil, mcpTask{}, err
		}
		s.broadcast("tasks")
		event := "task.moved"
		if column.IsDone {
			event = "task.completed"
		}
		s.notifyTask(ctx, event, moved)
		return nil, idx.task(moved), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "complete_task",
		Description: "Mark a task done by moving it to its board's done column. A repeating task rolls to its next due date.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in completeTaskIn) (*mcp.CallToolResult, mcpTask, error) {
		task, idx, err := s.resolveTaskArg(ctx, in.Task)
		if err != nil {
			return nil, mcpTask{}, err
		}
		if task.CompletedAt != nil {
			return nil, idx.task(task), nil
		}
		moved, _, err := s.taskService().Complete(ctx, task, time.Now())
		if err != nil {
			return nil, mcpTask{}, err
		}
		s.broadcast("tasks")
		s.notifyTask(ctx, "task.completed", moved)
		return nil, idx.task(moved), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_task",
		Description: "Archive a task. It leaves the board but can be restored from the UI.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in deleteTaskIn) (*mcp.CallToolResult, deleteTaskOut, error) {
		task, _, err := s.resolveTaskArg(ctx, in.Task)
		if err != nil {
			return nil, deleteTaskOut{}, err
		}
		if err := s.Store.ArchiveTask(ctx, task.ID); err != nil {
			return nil, deleteTaskOut{}, err
		}
		s.broadcast("tasks")
		return nil, deleteTaskOut{Ref: task.Ref, Archived: true}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_steps",
		Description: "Add steps (subtasks) to a task, or tasks to an arc.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in addStepsIn) (*mcp.CallToolResult, addStepsOut, error) {
		task, _, err := s.resolveTaskArg(ctx, in.Task)
		if err != nil {
			return nil, addStepsOut{}, err
		}
		added, err := s.taskService().CreateChildren(ctx, task, in.Steps)
		if added > 0 {
			s.broadcast("tasks")
			s.notifyTask(ctx, "task.updated", task)
		}
		if err != nil {
			return nil, addStepsOut{}, err
		}
		return nil, addStepsOut{Ref: task.Ref, Added: added}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_comment",
		Description: "Append a comment to a task. Use it for progress notes and handoffs.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in addCommentIn) (*mcp.CallToolResult, addCommentOut, error) {
		task, _, err := s.resolveTaskArg(ctx, in.Task)
		if err != nil {
			return nil, addCommentOut{}, err
		}
		body := strings.TrimSpace(in.Body)
		if body == "" {
			return nil, addCommentOut{}, fmt.Errorf("the comment is empty")
		}
		author := ""
		if agent, ok := auth.AgentFromContext(ctx); ok {
			author = "agent:" + agent.Name
		}
		comment, err := s.Store.CreateTaskComment(ctx, task.ID, author, body, nil)
		if err != nil {
			return nil, addCommentOut{}, err
		}
		s.broadcast("tasks")
		s.notifyTask(ctx, "task.commented", task)
		return nil, addCommentOut{Ref: task.Ref, CommentID: comment.ID}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tag_task",
		Description: "Add or remove tags on a task. Missing tags are created in the task's workspace.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in tagTaskIn) (*mcp.CallToolResult, tagTaskOut, error) {
		task, _, err := s.resolveTaskArg(ctx, in.Task)
		if err != nil {
			return nil, tagTaskOut{}, err
		}
		all, err := s.Store.ListTags(ctx, &task.WorkspaceID)
		if err != nil {
			return nil, tagTaskOut{}, err
		}
		for _, name := range sanitiseLabels(in.Add) {
			tag, exists := tagByName(all, name)
			if !exists {
				if tag, err = s.Store.CreateTag(ctx, task.WorkspaceID, name, "#888888"); err != nil {
					return nil, tagTaskOut{}, err
				}
			}
			if err := s.Store.AddTaskTag(ctx, task.ID, tag.ID); err != nil {
				return nil, tagTaskOut{}, err
			}
		}
		for _, name := range sanitiseLabels(in.Remove) {
			if tag, exists := tagByName(all, name); exists {
				if err := s.Store.RemoveTaskTag(ctx, task.ID, tag.ID); err != nil {
					return nil, tagTaskOut{}, err
				}
			}
		}
		s.broadcast("tasks")
		s.broadcast("tags")
		tags, err := s.Store.ListTagsForTask(ctx, task.ID)
		if err != nil {
			return nil, tagTaskOut{}, err
		}
		out := tagTaskOut{Ref: task.Ref, Tags: []string{}}
		for _, tag := range tags {
			out.Tags = append(out.Tags, tag.Name)
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search",
		Description: "Search tasks and notes by text in one workspace or everywhere this key can see.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in searchIn) (*mcp.CallToolResult, searchOut, error) {
		query := strings.TrimSpace(in.Query)
		if query == "" {
			return nil, searchOut{}, fmt.Errorf("query is required")
		}
		workspace, err := s.resolveWorkspaceArg(ctx, in.Workspace)
		if err != nil {
			return nil, searchOut{}, err
		}
		workspaceID := workspaceIDOf(workspace)
		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, searchOut{}, err
		}
		var foundTasks []store.Task
		var foundNotes []store.Note
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			var err error
			foundTasks, err = s.Store.SearchTasks(gctx, query, workspaceID)
			return err
		})
		g.Go(func() error {
			var err error
			foundNotes, err = s.Store.SearchNotes(gctx, query, workspaceID)
			return err
		})
		if err := g.Wait(); err != nil {
			return nil, searchOut{}, err
		}
		out := searchOut{Tasks: []mcpTask{}, Notes: []mcpNote{}}
		for i, task := range foundTasks {
			if i == 20 {
				break
			}
			out.Tasks = append(out.Tasks, idx.task(task))
		}
		for i, note := range foundNotes {
			if i == 20 {
				break
			}
			out.Notes = append(out.Notes, s.noteView(ctx, note, idx))
		}
		return nil, out, nil
	})
}

// columnForCreate picks the named column, or the first column that is not done.
func (s *Server) columnForCreate(ctx context.Context, board store.Board, name string) (store.BoardColumn, error) {
	if strings.TrimSpace(name) != "" {
		column, available, found, err := s.taskService().Column(ctx, board.ID, name)
		if err != nil {
			return store.BoardColumn{}, err
		}
		if !found {
			return store.BoardColumn{}, fmt.Errorf("no column named %q on %s. Columns: %s", name, board.Name, columnNames(available))
		}
		return column, nil
	}
	columns, err := s.Store.ListColumnsByBoard(ctx, board.ID)
	if err != nil {
		return store.BoardColumn{}, err
	}
	if len(columns) == 0 {
		return store.BoardColumn{}, fmt.Errorf("board %q has no columns. Add one with create_column.", board.Name)
	}
	for _, column := range columns {
		if !column.IsDone {
			return column, nil
		}
	}
	return columns[0], nil
}

func mcpDueDate(raw string) (*string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	if _, err := time.Parse(mcpDateLayout, value); err != nil {
		return nil, fmt.Errorf("due_date must be YYYY-MM-DD")
	}
	return &value, nil
}

func mcpRepeatChange(in *mcpRepeat) (repeatChange, error) {
	interval := in.Interval
	if interval == 0 {
		interval = 1
	}
	rule := recurrence.Rule{Freq: recurrence.Freq(in.Freq), Interval: interval}
	if err := rule.Validate(); err != nil {
		return repeatChange{}, err
	}
	return repeatChange{set: true, rule: &rule}, nil
}

func trimmedPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func optionalText(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
