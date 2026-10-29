package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/ai"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Server) aiStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"configured": s.aiProvider(r.Context()) != nil})
}

func (s *Server) aiBreakdown(w http.ResponseWriter, r *http.Request) {
	provider := s.aiProvider(r.Context())
	if provider == nil {
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

	titles, err := provider.GenerateSubtasks(ctx, task.Title, task.DescriptionMd)
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
			Priority: "medium", Position: position, ParentTaskID: &task.ID, CreatedByFai: true,
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
	provider := s.aiProvider(r.Context())
	if provider == nil {
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

	labels, err := provider.SuggestLabels(ctx, task.Title, task.DescriptionMd, existing)
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
	Messages    []ai.ChatMessage `json:"messages"`
	Prompt      string           `json:"prompt"`
	TZOffset    *int             `json:"tz_offset"`
	WorkspaceID int64            `json:"workspace_id"`
}

const chatActionInstructions = `You can also create or change things for the owner. When you do, add a final line to your reply:
ACTION: {"type":"create_task","title":"...","due_date":"YYYY-MM-DD"}
ACTION: {"type":"create_reminder","title":"...","remind_at":"RFC3339 with the local offset"}
ACTION: {"type":"find_task","task":"@TAG-123"}
ACTION: {"type":"move_task","task":"@TAG-123","column":"Column name"}
Use create_task for work to do. Use create_reminder only when the owner asks to be reminded or names a time of day for a nudge.
Use find_task when the owner asks about an existing task. Use move_task to change the column of an existing task. Always copy the @ref from the task context for task actions. Never guess a task or a column.
Resolve relative dates (today, tomorrow, Monday, in 10 minutes) against the current local time given below. A weekday alone means the next such day, or today when it is that day. Set a due date when the owner gives one and omit due_date when they do not.
Write at most one ACTION line, as strict JSON on one line. Never claim you changed something unless you add the ACTION line.`

type chatAction struct {
	Type     string  `json:"type"`
	Title    string  `json:"title"`
	DueDate  *string `json:"due_date"`
	RemindAt string  `json:"remind_at"`
	Task     string  `json:"task"`
	TaskID   int64   `json:"task_id"`
	Column   string  `json:"column"`
}

type ChatCreated struct {
	Type  string `json:"type"`
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type PendingChatAction struct {
	Type       string `json:"type"`
	TaskID     int64  `json:"task_id"`
	TaskRef    string `json:"task_ref"`
	TaskTitle  string `json:"task_title"`
	ColumnID   int64  `json:"column_id"`
	ColumnName string `json:"column_name"`
}

type ChatOutcome struct {
	Reply   string
	Created *ChatCreated
	Pending *PendingChatAction
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

func (s *Server) RunChat(ctx context.Context, messages []ai.ChatMessage, tzOffset *int) (ChatOutcome, error) {
	provider := s.aiProvider(ctx)
	if provider == nil {
		return ChatOutcome{}, fmt.Errorf("AI is not configured")
	}
	if len(messages) == 0 {
		return ChatOutcome{}, fmt.Errorf("messages are required")
	}
	if len(messages) > 10 {
		messages = messages[len(messages)-10:]
	}
	if messages[len(messages)-1].Role != "user" {
		return ChatOutcome{}, fmt.Errorf("the last message must come from the user")
	}

	now := localNow(tzOffset)
	system := ai.FaiSystem + "\n\nCurrent local time: " + now.Format("Monday, 2006-01-02 15:04 -07:00") + ".\n" + chatActionInstructions
	if context := s.faiContext(ctx); context != "" {
		system += "\n\nOpen task refs, task titles, board columns, and note titles from Fai. Use them only when relevant, and do not invent anything that is not listed:\n" + context
	}

	reply, err := provider.Chat(ctx, system, messages)
	if err != nil {
		return ChatOutcome{}, err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return ChatOutcome{}, fmt.Errorf("the model did not answer")
	}
	return s.finishChat(ctx, reply)
}

func (s *Server) finishChat(ctx context.Context, rawReply string) (ChatOutcome, error) {
	reply, action := splitChatAction(rawReply)
	if reply == "" {
		reply = "Done."
	}
	outcome := ChatOutcome{Reply: reply}
	if action == nil {
		return outcome, nil
	}

	switch action.Type {
	case "find_task":
		text, err := s.findChatTask(ctx, *action)
		if err != nil {
			return ChatOutcome{}, err
		}
		outcome.Reply = text
	case "move_task":
		pending, text, ok, err := s.prepareChatMove(ctx, *action)
		if err != nil {
			return ChatOutcome{}, err
		}
		if !ok {
			outcome.Reply = text
			return outcome, nil
		}
		outcome.Pending = &pending
		outcome.Reply = text
	default:
		created, err := s.applyChatAction(ctx, *action)
		if err != nil {
			return ChatOutcome{}, fmt.Errorf("could not create it: %w", err)
		}
		outcome.Created = &created
		if created.Type == "reminder" {
			s.broadcast("reminders")
		} else {
			s.broadcast("tasks")
		}
	}
	return outcome, nil
}

func (s *Server) applyChatAction(ctx context.Context, action chatAction) (ChatCreated, error) {
	title := strings.TrimSpace(action.Title)
	if title == "" {
		return ChatCreated{}, fmt.Errorf("the action has no title")
	}
	switch action.Type {
	case "create_task":
		var due *string
		if action.DueDate != nil && strings.TrimSpace(*action.DueDate) != "" {
			value := strings.TrimSpace(*action.DueDate)
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return ChatCreated{}, fmt.Errorf("invalid due date %q", value)
			}
			due = &value
		}
		boardID, columnID, err := s.defaultBoardColumn(ctx)
		if err != nil {
			return ChatCreated{}, err
		}
		position, err := s.appendPosition(ctx, columnID)
		if err != nil {
			return ChatCreated{}, err
		}
		task, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
			BoardID: boardID, ColumnID: columnID, Title: title,
			Priority: "medium", DueDate: due, Position: position, CreatedByFai: true,
		})
		if err != nil {
			return ChatCreated{}, err
		}
		return ChatCreated{Type: "task", ID: task.ID, Title: task.Title}, nil
	case "create_reminder":
		remindAt, err := time.Parse(time.RFC3339, action.RemindAt)
		if err != nil {
			return ChatCreated{}, fmt.Errorf("invalid remind_at")
		}
		reminder, err := s.Store.CreateReminder(ctx, title, remindAt, nil, 0)
		if err != nil {
			return ChatCreated{}, err
		}
		return ChatCreated{Type: "reminder", ID: reminder.ID, Title: reminder.Title}, nil
	default:
		return ChatCreated{}, fmt.Errorf("unknown action %q", action.Type)
	}
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

func (s *Server) findChatTask(ctx context.Context, action chatAction) (string, error) {
	task, refs, err := s.resolveChatActionTask(ctx, action)
	if err != nil {
		return "", err
	}
	if task.ID == 0 {
		return chatTaskCandidates("look up", refs), nil
	}
	var out strings.Builder
	out.WriteString("@" + task.Ref + " " + task.Title)
	columns, err := s.Store.ListColumnsByBoard(ctx, task.BoardID)
	if err == nil {
		for _, column := range columns {
			if column.ID == task.ColumnID {
				out.WriteString("\nColumn: " + column.Name)
				break
			}
		}
	}
	if task.DueDate != nil {
		out.WriteString("\nDue: " + *task.DueDate)
	}
	if task.Priority != "" && task.Priority != "medium" {
		out.WriteString("\nPriority: " + task.Priority)
	}
	return out.String(), nil
}

func (s *Server) prepareChatMove(ctx context.Context, action chatAction) (PendingChatAction, string, bool, error) {
	task, refs, err := s.resolveChatActionTask(ctx, action)
	if err != nil {
		return PendingChatAction{}, "", false, err
	}
	if task.ID == 0 {
		return PendingChatAction{}, chatTaskCandidates("move", refs), false, nil
	}

	column, available, found, err := s.resolveChatColumn(ctx, task.BoardID, action.Column)
	if err != nil {
		return PendingChatAction{}, "", false, err
	}
	if !found {
		name := strings.TrimSpace(action.Column)
		if name == "" {
			return PendingChatAction{}, "Which column should I move @" + task.Ref + " to? Available columns: " + columnNames(available) + ".", false, nil
		}
		return PendingChatAction{}, "I could not find a column named " + name + ". Available columns: " + columnNames(available) + ".", false, nil
	}

	pending := PendingChatAction{
		Type: "move_task", TaskID: task.ID, TaskRef: task.Ref, TaskTitle: task.Title,
		ColumnID: column.ID, ColumnName: column.Name,
	}
	return pending, "Move @" + task.Ref + " (" + task.Title + ") to " + column.Name + "?", true, nil
}

func (s *Server) resolveChatColumn(ctx context.Context, boardID int64, name string) (store.BoardColumn, []store.BoardColumn, bool, error) {
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

func columnNames(columns []store.BoardColumn) string {
	names := make([]string, 0, len(columns))
	for _, column := range columns {
		names = append(names, column.Name)
	}
	return strings.Join(names, ", ")
}

func (s *Server) ConfirmChatAction(ctx context.Context, pending PendingChatAction) (string, error) {
	if pending.Type != "move_task" {
		return "", fmt.Errorf("unknown pending action")
	}
	task, err := s.Store.GetTask(ctx, pending.TaskID)
	if err != nil {
		return "", fmt.Errorf("the task no longer exists")
	}
	columns, err := s.Store.ListColumnsByBoard(ctx, task.BoardID)
	if err != nil {
		return "", err
	}
	columnName := ""
	for _, column := range columns {
		if column.ID == pending.ColumnID {
			columnName = column.Name
			break
		}
	}
	if columnName == "" {
		return "", fmt.Errorf("the column no longer exists")
	}
	position, err := s.appendPosition(ctx, pending.ColumnID)
	if err != nil {
		return "", err
	}
	if _, err := s.Store.MoveTask(ctx, pending.TaskID, pending.ColumnID, position); err != nil {
		return "", err
	}
	s.broadcast("tasks")
	return "Moved @" + task.Ref + " to " + columnName + ".", nil
}

func (s *Server) Transcribe(ctx context.Context, audio []byte, format string) (string, error) {
	if s.Transcriber != nil {
		return s.Transcriber.Transcribe(ctx, audio, format)
	}
	provider := s.aiProvider(ctx)
	if provider == nil {
		return "", fmt.Errorf("AI is not configured")
	}
	return provider.Transcribe(ctx, audio, format)
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
	provider := s.aiProvider(r.Context())
	if provider == nil {
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

	outcome, err := s.RunChat(r.Context(), messages, body.TZOffset)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if body.WorkspaceID > 0 {
		_, _ = s.Store.CreateChatMessage(r.Context(), body.WorkspaceID, "user", messages[len(messages)-1].Content)
		_, _ = s.Store.CreateChatMessage(r.Context(), body.WorkspaceID, "assistant", outcome.Reply)
	}
	response := map[string]any{"reply": outcome.Reply}
	if outcome.Created != nil {
		response["created"] = outcome.Created
	}
	if outcome.Pending != nil {
		response["pending"] = outcome.Pending
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) listChatMessages(w http.ResponseWriter, r *http.Request) {
	workspaceID := optionalInt64Param(r, "workspace_id")
	if workspaceID == nil {
		writeError(w, http.StatusBadRequest, "workspace_id is required")
		return
	}
	limit := int32(50)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 200 {
			limit = int32(parsed)
		}
	}
	messages, err := s.Store.ListChatMessages(r.Context(), *workspaceID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

func (s *Server) confirmChatAction(w http.ResponseWriter, r *http.Request) {
	var pending PendingChatAction
	if err := decodeJSON(r, &pending); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	reply, err := s.ConfirmChatAction(r.Context(), pending)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"reply": reply})
}

// faiContext gives the chat a small, bounded picture of the owner's work. Task
// bodies and note bodies stay out of the prompt.
func (s *Server) faiContext(ctx context.Context) string {
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
		if task.Ref != "" {
			out.WriteString("@")
			out.WriteString(task.Ref)
			out.WriteString(" ")
		}
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

	out.WriteString("\nBoard columns:\n")
	for _, board := range boards {
		columns, err := s.Store.ListColumnsByBoard(ctx, board.ID)
		if err != nil {
			continue
		}
		out.WriteString("- ")
		out.WriteString(board.Name)
		out.WriteString(": ")
		out.WriteString(columnNames(columns))
		out.WriteString("\n")
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
