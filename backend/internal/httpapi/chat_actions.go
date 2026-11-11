package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

const chatActionInstructions = `You can also create or change things for the owner. When you do, add one final line to your reply for each change:
ACTION: {"type":"create_task","title":"...","due_date":"YYYY-MM-DD","repeat":{"freq":"weekly","interval":1},"steps":["...","..."]}
ACTION: {"type":"create_arc","title":"...","tasks":[{"title":"...","due_date":"YYYY-MM-DD","steps":["...","..."]}]}
ACTION: {"type":"create_reminder","title":"...","remind_at":"RFC3339 with the local offset","repeat":{"freq":"daily","interval":1}}
ACTION: {"type":"find_task","task":"@TAG-123"}
ACTION: {"type":"move_task","task":"@TAG-123","column":"Column name"}
ACTION: {"type":"add_steps","task":"@TAG-123","steps":["...","..."]}
ACTION: {"type":"update_task","task":"@TAG-123","title":"...","description":"...","priority":"low|medium|high","due_date":"YYYY-MM-DD","repeat":{"freq":"weekly","interval":1}}
ACTION: {"type":"complete_task","task":"@TAG-123"}
ACTION: {"type":"delete_task","task":"@TAG-123"}
ACTION: {"type":"tag_task","task":"@TAG-123","add":["..."],"remove":["..."]}
ACTION: {"type":"comment_task","task":"@TAG-123","body":"..."}
ACTION: {"type":"log_time","task":"@TAG-123","minutes":90,"started_at":"RFC3339","ended_at":"RFC3339"}
ACTION: {"type":"start_timer","task":"@TAG-123"}
ACTION: {"type":"stop_timer","task":"@TAG-123"}
ACTION: {"type":"update_reminder","reminder_id":12,"title":"...","remind_at":"RFC3339 with the local offset","repeat":{"freq":"monthly","interval":1}}
ACTION: {"type":"complete_reminder","reminder_id":12}
ACTION: {"type":"delete_reminder","reminder_id":12}
ACTION: {"type":"create_note","title":"...","body":"Markdown"}
ACTION: {"type":"append_note","note":"Exact note title","body":"Markdown"}
Use create_task for work to do. When the owner asks for steps or subtasks under one task, put their titles in steps instead of separate create_task actions. Use create_arc when the owner asks for an arc or epic, or for a group of tasks that each have steps. Use create_reminder only when the owner asks to be reminded or names a time of day for a nudge.
Use find_task when the owner asks about an existing task. Use move_task to change the column of an existing task. Use add_steps to add steps or subtasks to an existing task, or tasks to an existing arc. Use complete_task when the owner says a task or step is done. Use delete_task only when the owner asks to delete or remove a task.
In update_task, include only the fields the owner wants to change. Set "due_date":"" to clear a due date. The description replaces the whole description.
In tag_task, list tag names without #. A new tag name creates the tag.
In log_time, send started_at and ended_at when the owner gives both times. When the owner gives only how long it took, send minutes, and add ended_at only when they say when it ended.
Use start_timer when the owner starts working on a task now. Use stop_timer when they stop; omit "task" to stop whichever timer is running. Only one timer runs at a time.
In update_reminder, include only the fields the owner wants to change. Copy reminder_id from the reminders list.
Add "repeat" to create_reminder, update_reminder, create_task, or update_task when the owner wants something to repeat. freq is daily, weekly, monthly, or yearly. interval is how many of that unit between repeats (every 2 weeks is weekly with interval 2). A repeating task needs a due date, and an arc cannot repeat. In update_reminder and update_task, omit "repeat" to keep the current rule and send "repeat":null to stop repeating. Completing a repeating reminder moves it to its next time. Completing a repeating task creates the next one.
append_note adds text to the end of an existing note. Copy the note title exactly from the notes list.
Always copy the @ref from the task context for task actions. Never guess a task, a column, a reminder, or a note.
Resolve relative dates (today, tomorrow, Monday, in 10 minutes) against the current local time given below. A weekday alone means the next such day, or today when it is that day. Set a due date when the owner gives one and omit due_date when they do not.
Write each ACTION line as strict JSON on its own line. Never claim you changed something unless you add the ACTION line.`

type chatAction struct {
	Type        string          `json:"type"`
	Title       string          `json:"title"`
	DueDate     *string         `json:"due_date"`
	RemindAt    string          `json:"remind_at"`
	Task        string          `json:"task"`
	TaskID      int64           `json:"task_id"`
	Column      string          `json:"column"`
	ColumnID    int64           `json:"column_id,omitempty"`
	TaskRef     string          `json:"task_ref,omitempty"`
	TaskTitle   string          `json:"task_title,omitempty"`
	Steps       []string        `json:"steps"`
	Tasks       []chatTaskSpec  `json:"tasks"`
	Description *string         `json:"description,omitempty"`
	Priority    *string         `json:"priority,omitempty"`
	Add         []string        `json:"add,omitempty"`
	Remove      []string        `json:"remove,omitempty"`
	Body        string          `json:"body,omitempty"`
	Minutes     int             `json:"minutes,omitempty"`
	StartedAt   string          `json:"started_at,omitempty"`
	EndedAt     string          `json:"ended_at,omitempty"`
	ReminderID  int64           `json:"reminder_id,omitempty"`
	Note        string          `json:"note,omitempty"`
	NoteID      int64           `json:"note_id,omitempty"`
	Repeat      json.RawMessage `json:"repeat,omitempty"`
}

type chatTaskSpec struct {
	Title   string   `json:"title"`
	DueDate *string  `json:"due_date"`
	Steps   []string `json:"steps"`
}

type ChatCreated struct {
	Type  string `json:"type"`
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type PendingChatAction struct {
	Type       string       `json:"type"`
	Summary    string       `json:"summary,omitempty"`
	Actions    []chatAction `json:"actions,omitempty"`
	TaskID     int64        `json:"task_id,omitempty"`
	TaskRef    string       `json:"task_ref,omitempty"`
	TaskTitle  string       `json:"task_title,omitempty"`
	ColumnID   int64        `json:"column_id,omitempty"`
	ColumnName string       `json:"column_name,omitempty"`
}

// prepare returns ok=false with a question and nothing pending. confirm gets the
// action back from the client, so it must validate again.
type chatActionHandler struct {
	answer  func(*Server, context.Context, chatAction) (string, error)
	prepare func(*Server, context.Context, chatAction, time.Time) (chatAction, string, bool, error)
	confirm func(*Server, context.Context, chatAction) (string, error)
}

var chatActionHandlers = map[string]chatActionHandler{
	"find_task":         {answer: (*Server).findChatTask},
	"create_task":       {prepare: (*Server).prepareChatCreate, confirm: (*Server).confirmChatCreate},
	"create_arc":        {prepare: (*Server).prepareChatCreate, confirm: (*Server).confirmChatCreate},
	"create_reminder":   {prepare: (*Server).prepareChatCreateReminder, confirm: (*Server).confirmChatCreateReminder},
	"move_task":         {prepare: (*Server).prepareChatMove, confirm: (*Server).confirmChatMove},
	"add_steps":         {prepare: (*Server).prepareChatAddSteps, confirm: (*Server).confirmChatAddSteps},
	"update_task":       {prepare: (*Server).prepareChatUpdateTask, confirm: (*Server).confirmChatUpdateTask},
	"complete_task":     {prepare: (*Server).prepareChatCompleteTask, confirm: (*Server).confirmChatCompleteTask},
	"delete_task":       {prepare: (*Server).prepareChatDeleteTask, confirm: (*Server).confirmChatDeleteTask},
	"tag_task":          {prepare: (*Server).prepareChatTagTask, confirm: (*Server).confirmChatTagTask},
	"comment_task":      {prepare: (*Server).prepareChatComment, confirm: (*Server).confirmChatComment},
	"log_time":          {prepare: (*Server).prepareChatLogTime, confirm: (*Server).confirmChatLogTime},
	"start_timer":       {prepare: (*Server).prepareChatStartTimer, confirm: (*Server).confirmChatStartTimer},
	"stop_timer":        {prepare: (*Server).prepareChatStopTimer, confirm: (*Server).confirmChatStopTimer},
	"update_reminder":   {prepare: (*Server).prepareChatUpdateReminder, confirm: (*Server).confirmChatUpdateReminder},
	"complete_reminder": {prepare: (*Server).prepareChatCompleteReminder, confirm: (*Server).confirmChatCompleteReminder},
	"delete_reminder":   {prepare: (*Server).prepareChatDeleteReminder, confirm: (*Server).confirmChatDeleteReminder},
	"create_note":       {prepare: (*Server).prepareChatCreateNote, confirm: (*Server).confirmChatCreateNote},
	"append_note":       {prepare: (*Server).prepareChatAppendNote, confirm: (*Server).confirmChatAppendNote},
}

// splitChatActions separates the model's prose from its action lines. A
// malformed action line stays in the reply so the owner sees what went wrong.
func splitChatActions(reply string) (string, []chatAction) {
	lines := strings.Split(reply, "\n")
	kept := make([]string, 0, len(lines))
	var actions []chatAction
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "ACTION:") {
			kept = append(kept, line)
			continue
		}
		var action chatAction
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(trimmed, "ACTION:"))), &action); err != nil {
			kept = append(kept, line)
			continue
		}
		actions = append(actions, action)
	}
	return strings.TrimSpace(strings.Join(kept, "\n")), actions
}

func (s *Server) finishChat(ctx context.Context, rawReply string, now time.Time) (ChatOutcome, error) {
	reply, actions := splitChatActions(rawReply)
	if reply == "" {
		reply = "Done."
	}
	outcome := ChatOutcome{Reply: reply}

	var pending []chatAction
	var summaries []string
	for _, action := range actions {
		handler, known := chatActionHandlers[action.Type]
		if !known {
			return ChatOutcome{}, fmt.Errorf("unknown action %q", action.Type)
		}
		if handler.answer != nil {
			text, err := handler.answer(s, ctx, action)
			if err != nil {
				return ChatOutcome{}, err
			}
			outcome.Reply = text
			continue
		}
		prepared, text, ok, err := handler.prepare(s, ctx, action, now)
		if err != nil {
			return ChatOutcome{}, err
		}
		if !ok {
			outcome.Reply = text
			return outcome, nil
		}
		pending = append(pending, prepared)
		summaries = append(summaries, text)
	}
	if len(pending) == 0 {
		return outcome, nil
	}

	outcome.Reply = confirmationText(reply, summaries)
	pendingAction := &PendingChatAction{
		Type:    "batch",
		Summary: outcome.Reply,
		Actions: pending,
	}
	if len(pending) == 1 && pending[0].Type == "move_task" {
		action := pending[0]
		pendingAction.Type = "move_task"
		pendingAction.TaskID = action.TaskID
		pendingAction.TaskRef = action.TaskRef
		pendingAction.TaskTitle = action.TaskTitle
		pendingAction.ColumnID = action.ColumnID
		pendingAction.ColumnName = action.Column
	}
	outcome.Pending = pendingAction
	return outcome, nil
}

func confirmationText(prose string, summaries []string) string {
	var out strings.Builder
	if strings.TrimSpace(prose) != "" {
		out.WriteString(strings.TrimSpace(prose))
	}
	if len(summaries) == 1 {
		if out.Len() > 0 {
			out.WriteString("\n\n")
		}
		out.WriteString(summaries[0])
		return out.String()
	}
	if out.Len() > 0 {
		out.WriteString("\n\n")
	}
	out.WriteString("Confirm these changes:")
	for _, summary := range summaries {
		out.WriteString("\n- ")
		out.WriteString(strings.TrimSuffix(strings.TrimSpace(summary), "?"))
	}
	return out.String()
}

func (s *Server) ConfirmChatAction(ctx context.Context, pending PendingChatAction) (string, error) {
	actions := pending.Actions
	if len(actions) == 0 && pending.Type == "move_task" {
		actions = []chatAction{{
			Type: "move_task", TaskID: pending.TaskID, TaskRef: pending.TaskRef,
			TaskTitle: pending.TaskTitle, ColumnID: pending.ColumnID, Column: pending.ColumnName,
		}}
	}
	if len(actions) == 0 {
		return "", fmt.Errorf("unknown pending action")
	}

	// Actions apply in the order the owner confirmed them.
	replies := make([]string, 0, len(actions))
	for _, action := range actions {
		handler, known := chatActionHandlers[action.Type]
		if !known || handler.confirm == nil {
			return "", fmt.Errorf("unknown pending action %q", action.Type)
		}
		reply, err := handler.confirm(s, ctx, action)
		if err != nil {
			return "", err
		}
		replies = append(replies, reply)
	}
	return strings.Join(replies, "\n"), nil
}

func (s *Server) resolveChatActionTask(ctx context.Context, action chatAction) (store.Task, []store.TaskRef, error) {
	if action.TaskID != 0 {
		task, err := s.Store.GetTask(ctx, action.TaskID)
		if err != nil {
			return store.Task{}, nil, nil
		}
		return task, nil, nil
	}
	query := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(action.Task), "@"))
	if query == "" {
		return store.Task{}, nil, nil
	}
	if id, err := strconv.ParseInt(query, 10, 64); err == nil {
		if task, err := s.Store.GetTask(ctx, id); err == nil {
			return task, nil, nil
		}
	}
	refs, err := s.Store.ResolveRefs(ctx, query, nil)
	if err != nil {
		return store.Task{}, nil, err
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
		return store.Task{}, nil, nil
	}
	if len(refs) > 1 {
		return store.Task{}, refs, nil
	}
	task, err := s.Store.GetTask(ctx, refs[0].ID)
	if err != nil {
		return store.Task{}, nil, err
	}
	return task, nil, nil
}

// A non-empty question means nothing was resolved.
func (s *Server) targetChatTask(ctx context.Context, action *chatAction, verb string) (store.Task, string, error) {
	task, refs, err := s.resolveChatActionTask(ctx, *action)
	if err != nil {
		return store.Task{}, "", err
	}
	if task.ID == 0 {
		return store.Task{}, chatTaskCandidates(verb, refs), nil
	}
	action.TaskID = task.ID
	action.TaskRef = task.Ref
	action.TaskTitle = task.Title
	return task, "", nil
}

func (s *Server) confirmedChatTask(ctx context.Context, action chatAction) (store.Task, error) {
	task, err := s.Store.GetTask(ctx, action.TaskID)
	if err != nil || action.TaskID == 0 {
		return store.Task{}, fmt.Errorf("the task no longer exists")
	}
	return task, nil
}

func chatTaskLabel(task store.Task) string {
	return "@" + task.Ref + " (" + task.Title + ")"
}

func chatTaskCandidates(action string, refs []store.TaskRef) string {
	var out strings.Builder
	if len(refs) == 0 {
		out.WriteString("I could not find that task. Use an @ref from the task list.")
		return out.String()
	}
	out.WriteString("Which task should I ")
	out.WriteString(action)
	out.WriteString("?")
	for i, ref := range refs {
		if i == 5 {
			break
		}
		out.WriteString("\n- @")
		out.WriteString(ref.Ref)
		out.WriteString(" ")
		out.WriteString(ref.Title)
	}
	return out.String()
}

func quoteSnippet(text string) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) > 80 {
		return "\"" + string(runes[:77]) + "...\""
	}
	return "\"" + string(runes) + "\""
}
