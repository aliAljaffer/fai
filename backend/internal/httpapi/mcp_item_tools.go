package httpapi

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/alialjaffer/fai/backend/internal/store"
)

// --- notes -------------------------------------------------------------------

type createNoteIn struct {
	Title     string `json:"title"`
	BodyMd    string `json:"body_md,omitempty" jsonschema:"Markdown body"`
	Workspace string `json:"workspace,omitempty"`
	Task      string `json:"task,omitempty" jsonschema:"optional task ref to attach the note to"`
}

type getNoteIn struct {
	Note string `json:"note" jsonschema:"note title, or an id"`
}

type appendNoteIn struct {
	Note   string `json:"note" jsonschema:"note title, or an id"`
	BodyMd string `json:"body_md" jsonschema:"Markdown appended to the end of the note"`
}

type noteOut struct {
	Note        mcpNote         `json:"note"`
	MentionedIn []referenceView `json:"mentioned_in,omitempty"`
}

func (s *Server) registerItemTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_note",
		Description: "Create a Markdown note, optionally attached to a task and inside a workspace.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createNoteIn) (*mcp.CallToolResult, noteOut, error) {
		title := strings.TrimSpace(in.Title)
		if title == "" {
			return nil, noteOut{}, fmt.Errorf("title is required")
		}
		var task *store.Task
		if strings.TrimSpace(in.Task) != "" {
			found, _, err := s.resolveTaskArg(ctx, in.Task)
			if err != nil {
				return nil, noteOut{}, err
			}
			task = &found
		}
		var workspaceID int64
		if task != nil {
			workspaceID = task.WorkspaceID
		} else {
			workspace, err := s.resolveWorkspaceArg(ctx, in.Workspace)
			if err != nil {
				return nil, noteOut{}, err
			}
			if workspace != nil {
				workspaceID = workspace.ID
			} else {
				first, err := s.defaultWorkspace(ctx)
				if err != nil {
					return nil, noteOut{}, err
				}
				workspaceID = first.ID
			}
		}
		var taskID *int64
		if task != nil {
			taskID = &task.ID
		}
		note, err := s.Store.CreateNote(ctx, title, in.BodyMd, taskID, workspaceID)
		if err != nil {
			return nil, noteOut{}, err
		}
		s.broadcast("notes")
		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, noteOut{}, err
		}
		return nil, noteOut{Note: s.noteView(ctx, note, idx)}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_note",
		Description: "Read a note by title or id, with the items that mention it.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in getNoteIn) (*mcp.CallToolResult, noteOut, error) {
		note, err := s.resolveNoteArg(ctx, in.Note)
		if err != nil {
			return nil, noteOut{}, err
		}
		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, noteOut{}, err
		}
		out := noteOut{Note: s.noteView(ctx, note, idx)}
		backlinks, err := s.Store.ListBacklinks(ctx, note.ID, note.Title)
		if err != nil {
			return nil, noteOut{}, err
		}
		for _, link := range backlinks {
			out.MentionedIn = append(out.MentionedIn, referenceView{Kind: "note", Title: link.Title})
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "append_note",
		Description: "Append Markdown to the end of an existing note. The rest of the note is kept.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in appendNoteIn) (*mcp.CallToolResult, noteOut, error) {
		note, err := s.resolveNoteArg(ctx, in.Note)
		if err != nil {
			return nil, noteOut{}, err
		}
		added := strings.TrimSpace(in.BodyMd)
		if added == "" {
			return nil, noteOut{}, fmt.Errorf("body_md is empty")
		}
		body := strings.TrimRight(note.BodyMd, "\n")
		if body != "" {
			body += "\n\n"
		}
		updated, err := s.Store.UpdateNote(ctx, note.ID, note.Title, body+added, note.TaskID)
		if err != nil {
			return nil, noteOut{}, err
		}
		s.broadcast("notes")
		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, noteOut{}, err
		}
		return nil, noteOut{Note: s.noteView(ctx, updated, idx)}, nil
	})

	// --- timers --------------------------------------------------------------

	mcp.AddTool(server, &mcp.Tool{
		Name:        "start_timer",
		Description: "Start the timer on a task. Starting one stops whichever timer is running.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in completeTaskIn) (*mcp.CallToolResult, runningTimer, error) {
		task, _, err := s.resolveTaskArg(ctx, in.Task)
		if err != nil {
			return nil, runningTimer{}, err
		}
		entry, err := s.Store.StartTimer(ctx, task.ID)
		if err != nil {
			return nil, runningTimer{}, err
		}
		s.broadcast("tasks")
		return nil, runningTimer{
			Task:    "@" + task.Ref + " " + task.Title,
			Since:   entry.StartedAt.Format(time.RFC3339),
			Seconds: 0,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop_timer",
		Description: "Stop the running timer, on the named task or on whichever task is running.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in stopTimerIn) (*mcp.CallToolResult, stopTimerOut, error) {
		var task store.Task
		if strings.TrimSpace(in.Task) != "" {
			found, _, err := s.resolveTaskArg(ctx, in.Task)
			if err != nil {
				return nil, stopTimerOut{}, err
			}
			task = found
		} else {
			entry, err := s.Store.GetRunningEntry(ctx)
			if err != nil {
				return nil, stopTimerOut{}, err
			}
			if entry == nil {
				return nil, stopTimerOut{}, fmt.Errorf("no timer is running")
			}
			if task, err = s.Store.GetTask(ctx, entry.TaskID); err != nil {
				return nil, stopTimerOut{}, err
			}
			if workspace, err := s.agentWorkspace(ctx); err != nil {
				return nil, stopTimerOut{}, err
			} else if workspace != nil && task.WorkspaceID != workspace.ID {
				return nil, stopTimerOut{}, fmt.Errorf("the running timer belongs to another workspace")
			}
		}
		entry, err := s.Store.StopEntryForTask(ctx, task.ID)
		if err != nil {
			return nil, stopTimerOut{}, fmt.Errorf("no timer is running on @%s", task.Ref)
		}
		s.broadcast("tasks")
		seconds := int64(0)
		if entry.DurationSeconds != nil {
			seconds = int64(*entry.DurationSeconds)
		}
		return nil, stopTimerOut{Ref: task.Ref, Seconds: seconds}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "log_time",
		Description: "Log time on a task. Give minutes, or started_at with an optional ended_at in RFC3339.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in logTimeIn) (*mcp.CallToolResult, stopTimerOut, error) {
		task, _, err := s.resolveTaskArg(ctx, in.Task)
		if err != nil {
			return nil, stopTimerOut{}, err
		}
		start, end, err := timeSpan(in.StartedAt, in.EndedAt, in.Minutes, time.Now())
		if err != nil {
			return nil, stopTimerOut{}, err
		}
		entry, err := s.Store.LogTimeEntry(ctx, task.ID, start, end)
		if err != nil {
			return nil, stopTimerOut{}, err
		}
		s.broadcast("tasks")
		seconds := int64(end.Sub(start).Seconds())
		if entry.DurationSeconds != nil {
			seconds = int64(*entry.DurationSeconds)
		}
		return nil, stopTimerOut{Ref: task.Ref, Seconds: seconds}, nil
	})

	// --- reminders -----------------------------------------------------------

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_reminder",
		Description: "Create a reminder at a specific time, optionally repeating and optionally linked to a task.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createReminderIn) (*mcp.CallToolResult, reminderOut, error) {
		title := strings.TrimSpace(in.Title)
		if title == "" {
			return nil, reminderOut{}, fmt.Errorf("title is required")
		}
		remindAt, err := time.Parse(time.RFC3339, strings.TrimSpace(in.RemindAt))
		if err != nil {
			return nil, reminderOut{}, fmt.Errorf("remind_at must be RFC3339 with an offset")
		}
		var task *store.Task
		if strings.TrimSpace(in.Task) != "" {
			found, _, err := s.resolveTaskArg(ctx, in.Task)
			if err != nil {
				return nil, reminderOut{}, err
			}
			task = &found
		}
		workspace, err := s.resolveWorkspaceArg(ctx, in.Workspace)
		if err != nil {
			return nil, reminderOut{}, err
		}
		var workspaceID int64
		switch {
		case task != nil:
			workspaceID = task.WorkspaceID
		case workspace != nil:
			workspaceID = workspace.ID
		default:
			first, err := s.defaultWorkspace(ctx)
			if err != nil {
				return nil, reminderOut{}, err
			}
			workspaceID = first.ID
		}
		var repeat *store.ReminderRepeat
		if in.Repeat != nil {
			change, err := mcpRepeatChange(in.Repeat)
			if err != nil {
				return nil, reminderOut{}, err
			}
			repeat = reminderRepeat(change, remindAt)
		}
		var taskID *int64
		if task != nil {
			taskID = &task.ID
		}
		reminder, err := s.Store.CreateReminder(ctx, title, remindAt, taskID, workspaceID, repeat)
		if err != nil {
			return nil, reminderOut{}, err
		}
		s.broadcast("reminders")
		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, reminderOut{}, err
		}
		return nil, reminderOut{Reminder: s.reminderView(ctx, reminder, idx)}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_reminders",
		Description: "List upcoming reminders, soonest first.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listRemindersIn) (*mcp.CallToolResult, reminderListOut, error) {
		workspace, err := s.resolveWorkspaceArg(ctx, in.Workspace)
		if err != nil {
			return nil, reminderListOut{}, err
		}
		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, reminderListOut{}, err
		}
		reminders, err := s.Store.ListReminders(ctx, workspaceIDOf(workspace))
		if err != nil {
			return nil, reminderListOut{}, err
		}
		out := reminderListOut{Reminders: []mcpReminder{}}
		for _, reminder := range reminders {
			if !in.IncludeCompleted && reminder.CompletedAt != nil {
				continue
			}
			out.Reminders = append(out.Reminders, s.reminderView(ctx, reminder, idx))
		}
		sort.SliceStable(out.Reminders, func(i, j int) bool {
			return out.Reminders[i].RemindAt < out.Reminders[j].RemindAt
		})
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "complete_reminder",
		Description: "Complete a reminder. A repeating reminder moves to its next time.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in completeReminderIn) (*mcp.CallToolResult, reminderOut, error) {
		reminder, err := s.Store.GetReminder(ctx, in.ReminderID)
		if err != nil {
			return nil, reminderOut{}, fmt.Errorf("no reminder with id %d", in.ReminderID)
		}
		if workspace, err := s.agentWorkspace(ctx); err != nil {
			return nil, reminderOut{}, err
		} else if workspace != nil && reminder.WorkspaceID != workspace.ID {
			return nil, reminderOut{}, fmt.Errorf("this key is scoped to workspace %s", workspace.Tag)
		}
		updated, err := s.completeReminderItem(ctx, reminder, time.Now())
		if err != nil {
			return nil, reminderOut{}, err
		}
		s.broadcast("reminders")
		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, reminderOut{}, err
		}
		return nil, reminderOut{Reminder: s.reminderView(ctx, updated, idx)}, nil
	})

	// --- agenda --------------------------------------------------------------

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_upcoming",
		Description: "Show what is due or scheduled in the next days: open tasks with due dates, and reminders.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listUpcomingIn) (*mcp.CallToolResult, upcomingOut, error) {
		days := in.Days
		if days <= 0 {
			days = 7
		}
		if days > 365 {
			days = 365
		}
		now := time.Now()
		horizon := now.AddDate(0, 0, days).Format(mcpDateLayout)
		workspace, err := s.resolveWorkspaceArg(ctx, in.Workspace)
		if err != nil {
			return nil, upcomingOut{}, err
		}
		idx, err := s.loadIndex(ctx)
		if err != nil {
			return nil, upcomingOut{}, err
		}
		out := upcomingOut{
			Today: now.Format(mcpDateLayout), Through: horizon,
			Tasks: []mcpTask{}, Reminders: []mcpReminder{},
		}
		var boards []store.Board
		if workspace != nil {
			boards, err = s.Store.ListBoards(ctx, &workspace.ID)
		} else {
			boards, err = s.Store.ListBoards(ctx, nil)
		}
		if err != nil {
			return nil, upcomingOut{}, err
		}
		for _, board := range boards {
			found, err := s.Store.ListTasks(ctx, store.ListTasksParams{BoardID: &board.ID})
			if err != nil {
				return nil, upcomingOut{}, err
			}
			for _, task := range found {
				if task.DueDate == nil || task.CompletedAt != nil || *task.DueDate > horizon {
					continue
				}
				out.Tasks = append(out.Tasks, idx.task(task))
			}
		}
		sort.SliceStable(out.Tasks, func(i, j int) bool {
			return *out.Tasks[i].DueDate < *out.Tasks[j].DueDate
		})
		reminders, err := s.Store.ListReminders(ctx, workspaceIDOf(workspace))
		if err != nil {
			return nil, upcomingOut{}, err
		}
		for _, reminder := range reminders {
			if reminder.CompletedAt != nil || reminder.RemindAt.After(now.AddDate(0, 0, days)) {
				continue
			}
			out.Reminders = append(out.Reminders, s.reminderView(ctx, reminder, idx))
		}
		sort.SliceStable(out.Reminders, func(i, j int) bool {
			return out.Reminders[i].RemindAt < out.Reminders[j].RemindAt
		})
		return nil, out, nil
	})
}

type stopTimerIn struct {
	Task string `json:"task,omitempty" jsonschema:"task ref; omit to stop whichever timer is running"`
}

type stopTimerOut struct {
	Ref     string `json:"ref"`
	Seconds int64  `json:"seconds"`
}

type logTimeIn struct {
	Task      string `json:"task"`
	Minutes   int    `json:"minutes,omitempty"`
	StartedAt string `json:"started_at,omitempty" jsonschema:"RFC3339 with an offset"`
	EndedAt   string `json:"ended_at,omitempty" jsonschema:"RFC3339 with an offset; defaults to now"`
}

type createReminderIn struct {
	Title     string     `json:"title"`
	RemindAt  string     `json:"remind_at" jsonschema:"RFC3339 with an offset"`
	Repeat    *mcpRepeat `json:"repeat,omitempty"`
	Task      string     `json:"task,omitempty"`
	Workspace string     `json:"workspace,omitempty"`
}

type reminderOut struct {
	Reminder mcpReminder `json:"reminder"`
}

type listRemindersIn struct {
	IncludeCompleted bool   `json:"include_completed,omitempty"`
	Workspace        string `json:"workspace,omitempty"`
}

type reminderListOut struct {
	Reminders []mcpReminder `json:"reminders"`
}

type completeReminderIn struct {
	ReminderID int64 `json:"reminder_id"`
}

type listUpcomingIn struct {
	Days      int    `json:"days,omitempty" jsonschema:"how many days ahead, default 7"`
	Workspace string `json:"workspace,omitempty"`
}

type upcomingOut struct {
	Today     string        `json:"today"`
	Through   string        `json:"through"`
	Tasks     []mcpTask     `json:"tasks"`
	Reminders []mcpReminder `json:"reminders"`
}

func (s *Server) noteView(ctx context.Context, note store.Note, idx index) mcpNote {
	view := mcpNote{
		ID: note.ID, Title: note.Title, BodyMd: note.BodyMd,
		Workspace: idx.workspaceTag(note.WorkspaceID), UpdatedAt: note.UpdatedAt.Format(time.RFC3339),
	}
	if note.TaskID != nil {
		if task, err := s.Store.GetTask(ctx, *note.TaskID); err == nil {
			view.TaskRef = task.Ref
		}
	}
	return view
}

func (s *Server) reminderView(ctx context.Context, reminder store.Reminder, idx index) mcpReminder {
	view := mcpReminder{
		ID: reminder.ID, Title: reminder.Title, RemindAt: reminder.RemindAt.Format(time.RFC3339),
		Workspace: idx.workspaceTag(reminder.WorkspaceID), Completed: reminder.CompletedAt != nil,
	}
	if reminder.TaskID != nil {
		if task, err := s.Store.GetTask(ctx, *reminder.TaskID); err == nil {
			view.TaskRef = task.Ref
		}
	}
	return view
}

func (s *Server) resolveNoteArg(ctx context.Context, value string) (store.Note, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return store.Note{}, fmt.Errorf("a note title or id is required")
	}
	var note store.Note
	var err error
	if id, ok := parseInt64(value); ok {
		if note, err = s.Store.GetNote(ctx, id); err == nil {
			return s.checkNoteScope(ctx, note)
		}
	}
	note, err = s.Store.GetNoteByTitle(ctx, value)
	if err != nil {
		return store.Note{}, fmt.Errorf("no note titled %q", value)
	}
	return s.checkNoteScope(ctx, note)
}

func (s *Server) checkNoteScope(ctx context.Context, note store.Note) (store.Note, error) {
	workspace, err := s.agentWorkspace(ctx)
	if err != nil {
		return store.Note{}, err
	}
	if workspace != nil && note.WorkspaceID != workspace.ID {
		return store.Note{}, fmt.Errorf("this key is scoped to workspace %s", workspace.Tag)
	}
	return note, nil
}
