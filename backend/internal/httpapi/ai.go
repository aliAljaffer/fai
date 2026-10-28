package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/ai"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Server) aiStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"configured": s.AI != nil})
}

func (s *Server) aiBreakdown(w http.ResponseWriter, r *http.Request) {
	if s.AI == nil {
		writeError(w, http.StatusServiceUnavailable, "AI is not configured")
		return
	}
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	ctx := r.Context()
	task, err := s.Store.GetTask(ctx, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	titles, err := s.AI.GenerateSubtasks(ctx, task.Title, task.DescriptionMd)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	type createdSubtask struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	created := make([]createdSubtask, 0, len(titles))
	for _, title := range titles {
		position, err := s.appendPosition(ctx, task.ColumnID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		sub, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
			BoardID: task.BoardID, ColumnID: task.ColumnID, Title: title,
			Priority: "medium", Position: position, ParentTaskID: &task.ID,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		created = append(created, createdSubtask{ID: sub.ID, Title: sub.Title})
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) aiSuggestLabels(w http.ResponseWriter, r *http.Request) {
	if s.AI == nil {
		writeError(w, http.StatusServiceUnavailable, "AI is not configured")
		return
	}
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	ctx := r.Context()
	task, err := s.Store.GetTask(ctx, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	tags, err := s.Store.ListTags(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	existing := make([]string, 0, len(tags))
	for _, tag := range tags {
		existing = append(existing, tag.Name)
	}

	labels, err := s.AI.SuggestLabels(ctx, task.Title, task.DescriptionMd, existing)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string][]string{"labels": sanitiseLabels(labels)})
}

func sanitiseLabels(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		label := strings.ToLower(strings.TrimSpace(value))
		label = strings.ReplaceAll(label, " ", "-")
		label = strings.TrimPrefix(label, "#")
		if label == "" {
			continue
		}
		if _, duplicate := seen[label]; duplicate {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
		if len(out) == 6 {
			break
		}
	}
	return out
}

type aiChatBody struct {
	Messages []ai.ChatMessage `json:"messages"`
	Prompt   string           `json:"prompt"`
	TZOffset *int             `json:"tz_offset"`
}

const chatActionInstructions = `You can also create things for the owner. When they ask for a task or a reminder, add a final line to your reply:
ACTION: {"type":"create_task","title":"...","due_date":"YYYY-MM-DD"}
ACTION: {"type":"create_reminder","title":"...","remind_at":"RFC3339 with the local offset"}
Use create_task for work to do. Use create_reminder only when the owner asks to be reminded or names a time of day for a nudge.
Resolve relative dates (today, tomorrow, Monday, in 10 minutes) against the current local time given below. A weekday alone means the next such day, or today when it is that day. Set a due date when the owner gives one and omit due_date when they do not.
Write at most one ACTION line, as strict JSON on one line. Never claim you created something unless you add the ACTION line.`

type chatAction struct {
	Type     string  `json:"type"`
	Title    string  `json:"title"`
	DueDate  *string `json:"due_date"`
	RemindAt string  `json:"remind_at"`
}

type chatCreated struct {
	Type  string `json:"type"`
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// splitChatAction separates the model's prose from the optional action line. A
// malformed action line stays in the reply so the owner sees what went wrong.
func splitChatAction(reply string) (string, *chatAction) {
	idx := strings.LastIndex(reply, "ACTION:")
	if idx == -1 {
		return reply, nil
	}
	line := reply[idx+len("ACTION:"):]
	if nl := strings.IndexByte(line, '\n'); nl >= 0 {
		line = line[:nl]
	}
	var action chatAction
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &action); err != nil {
		return reply, nil
	}
	return strings.TrimSpace(reply[:idx]), &action
}

func (s *Server) applyChatAction(ctx context.Context, action chatAction) (chatCreated, error) {
	title := strings.TrimSpace(action.Title)
	if title == "" {
		return chatCreated{}, fmt.Errorf("the action has no title")
	}
	switch action.Type {
	case "create_task":
		var due *string
		if action.DueDate != nil && strings.TrimSpace(*action.DueDate) != "" {
			value := strings.TrimSpace(*action.DueDate)
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return chatCreated{}, fmt.Errorf("invalid due date %q", value)
			}
			due = &value
		}
		boardID, columnID, err := s.defaultBoardColumn(ctx)
		if err != nil {
			return chatCreated{}, err
		}
		position, err := s.appendPosition(ctx, columnID)
		if err != nil {
			return chatCreated{}, err
		}
		task, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
			BoardID: boardID, ColumnID: columnID, Title: title,
			Priority: "medium", DueDate: due, Position: position,
		})
		if err != nil {
			return chatCreated{}, err
		}
		return chatCreated{Type: "task", ID: task.ID, Title: task.Title}, nil
	case "create_reminder":
		remindAt, err := time.Parse(time.RFC3339, action.RemindAt)
		if err != nil {
			return chatCreated{}, fmt.Errorf("invalid remind_at")
		}
		reminder, err := s.Store.CreateReminder(ctx, title, remindAt, nil, 0)
		if err != nil {
			return chatCreated{}, err
		}
		return chatCreated{Type: "reminder", ID: reminder.ID, Title: reminder.Title}, nil
	default:
		return chatCreated{}, fmt.Errorf("unknown action %q", action.Type)
	}
}

func (s *Server) defaultBoardColumn(ctx context.Context) (int64, int64, error) {
	boards, err := s.Store.ListBoards(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	if len(boards) == 0 {
		return 0, 0, fmt.Errorf("no board exists yet")
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

func (s *Server) aiChat(w http.ResponseWriter, r *http.Request) {
	if s.AI == nil {
		writeError(w, http.StatusServiceUnavailable, "AI is not configured")
		return
	}
	var body aiChatBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	messages := body.Messages
	if len(messages) == 0 && strings.TrimSpace(body.Prompt) != "" {
		messages = []ai.ChatMessage{{Role: "user", Content: strings.TrimSpace(body.Prompt)}}
	}
	if len(messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages are required")
		return
	}
	if len(messages) > 10 {
		messages = messages[len(messages)-10:]
	}
	for i, message := range messages {
		if message.Role != "user" && message.Role != "assistant" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("message %d has an unsupported role", i+1))
			return
		}
		if strings.TrimSpace(message.Content) == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("message %d is empty", i+1))
			return
		}
		if len([]rune(message.Content)) > 4000 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("message %d is too long", i+1))
			return
		}
	}
	if messages[len(messages)-1].Role != "user" {
		writeError(w, http.StatusBadRequest, "the last message must come from the user")
		return
	}

	now := localNow(body.TZOffset)
	system := ai.FaiSystem + "\n\nCurrent local time: " + now.Format("Monday, 2006-01-02 15:04 -07:00") + ".\n" + chatActionInstructions
	if context := s.faiContext(r); context != "" {
		system += "\n\nOpen task titles and note titles from Fai. Use them only when relevant, and do not invent anything that is not listed:\n" + context
	}

	reply, err := s.AI.Chat(r.Context(), system, messages)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		writeError(w, http.StatusBadGateway, "the model did not answer")
		return
	}

	reply, action := splitChatAction(reply)
	if reply == "" {
		reply = "Done."
	}
	response := map[string]any{"reply": reply}
	if action != nil {
		created, err := s.applyChatAction(r.Context(), *action)
		if err != nil {
			writeError(w, http.StatusBadGateway, "could not create it: "+err.Error())
			return
		}
		response["created"] = created
	}
	writeJSON(w, http.StatusOK, response)
}

// faiContext gives the chat a small, bounded picture of the owner's work. Task
// bodies and note bodies stay out of the prompt.
func (s *Server) faiContext(r *http.Request) string {
	ctx := r.Context()
	boards, err := s.Store.ListBoards(ctx, nil)
	if err != nil {
		return ""
	}
	boardName := make(map[int64]string, len(boards))
	for _, board := range boards {
		boardName[board.ID] = board.Name
	}

	tasks, err := s.Store.ListTasks(ctx, store.ListTasksParams{})
	if err != nil {
		return ""
	}
	var out strings.Builder
	out.WriteString("Tasks:\n")
	count := 0
	for _, task := range tasks {
		if task.Archived || task.CompletedAt != nil || task.ParentTaskID != nil {
			continue
		}
		if count == 80 {
			break
		}
		out.WriteString("- ")
		out.WriteString(task.Title)
		if board := boardName[task.BoardID]; board != "" {
			out.WriteString(" [")
			out.WriteString(board)
			out.WriteString("]")
		}
		if task.DueDate != nil {
			out.WriteString(" due ")
			out.WriteString(*task.DueDate)
		}
		if task.Priority != "medium" {
			out.WriteString(", ")
			out.WriteString(task.Priority)
			out.WriteString(" priority")
		}
		out.WriteString("\n")
		count++
	}
	if count == 0 {
		out.WriteString("No open tasks.\n")
	}

	notes, err := s.Store.ListNotes(ctx, nil)
	if err == nil {
		out.WriteString("\nNotes:\n")
		if len(notes) == 0 {
			out.WriteString("No notes.\n")
		}
		for i, note := range notes {
			if i == 40 {
				break
			}
			out.WriteString("- ")
			out.WriteString(note.Title)
			out.WriteString("\n")
		}
	}
	return strings.TrimSpace(out.String())
}
