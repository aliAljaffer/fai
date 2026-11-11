package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

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

type aiLabelsBatchBody struct {
	TaskIDs []int64 `json:"task_ids"`
}

type aiLabelsBatchResult struct {
	TaskID int64    `json:"task_id"`
	Labels []string `json:"labels"`
}

const maxLabelBatch = 20

// aiSuggestLabelsBatch labels many tasks with one model call. The selection is
// capped so a large list view selection cannot build a huge prompt.
func (s *Server) aiSuggestLabelsBatch(w http.ResponseWriter, r *http.Request) {
	provider := s.aiProvider(r.Context())
	if provider == nil {
		writeError(w, http.StatusServiceUnavailable, "AI is not configured")
		return
	}
	var body aiLabelsBatchBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(body.TaskIDs) == 0 {
		writeError(w, http.StatusBadRequest, "task_ids is required")
		return
	}
	if len(body.TaskIDs) > maxLabelBatch {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("at most %d tasks per request", maxLabelBatch))
		return
	}

	ctx := r.Context()
	tasks := make([]store.Task, len(body.TaskIDs))
	var tags []store.Tag
	errTaskNotFound := errors.New("task not found")
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(8)
	for i, id := range body.TaskIDs {
		i, id := i, id
		g.Go(func() error {
			task, err := s.Store.GetTask(ctx, id)
			if err != nil {
				return errTaskNotFound
			}
			tasks[i] = task
			return nil
		})
	}
	g.Go(func() error {
		var err error
		tags, err = s.Store.ListTags(ctx)
		return err
	})
	if err := g.Wait(); err != nil {
		if errors.Is(err, errTaskNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	existing := make([]string, 0, len(tags))
	for _, tag := range tags {
		existing = append(existing, tag.Name)
	}

	batch := make([]ai.LabelTask, 0, len(tasks))
	for _, task := range tasks {
		batch = append(batch, ai.LabelTask{Title: task.Title, Description: task.DescriptionMd})
	}
	text, err := provider.Chat(ctx, ai.FaiSystem, []ai.ChatMessage{
		{Role: "user", Content: ai.LabelsBatchPrompt(batch, existing)},
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	labels, err := ai.ParseLabelBatches(text, len(tasks))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	results := make([]aiLabelsBatchResult, 0, len(tasks))
	for i, task := range tasks {
		results = append(results, aiLabelsBatchResult{TaskID: task.ID, Labels: sanitiseLabels(labels[i])})
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
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

type ChatOutcome struct {
	Reply   string
	Created []ChatCreated
	Pending *PendingChatAction
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
	system := ai.StepSystem + "\n\nCurrent local time: " + now.Format("Monday, 2006-01-02 15:04 -07:00") + ".\n" + chatActionInstructions
	if context := s.faiContext(ctx); context != "" {
		system += "\n\nOpen tasks with their steps, board columns, open reminders, and note titles from Fai. Use them only when relevant, and do not invent anything that is not listed:\n" + context
	}

	reply, err := provider.Chat(ctx, system, messages)
	if err != nil {
		return ChatOutcome{}, err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return ChatOutcome{}, fmt.Errorf("the model did not answer")
	}
	return s.finishChat(ctx, reply, now)
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
	if len(outcome.Created) > 0 {
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

const (
	maxContextTaskLines = 200
	maxContextTaskDepth = 3
)

// faiContext gives the chat a small, bounded picture of the owner's work. Task
// bodies and note bodies stay out of the prompt.
func (s *Server) faiContext(ctx context.Context) string {
	var boards []store.Board
	var tasks []store.Task
	var notes []store.Note
	var reminders []store.Reminder
	var boardsErr, tasksErr, notesErr, remindersErr error
	var load errgroup.Group
	load.Go(func() error {
		boards, boardsErr = s.Store.ListBoards(ctx, nil)
		return nil
	})
	load.Go(func() error {
		tasks, tasksErr = s.Store.ListTasks(ctx, store.ListTasksParams{})
		return nil
	})
	load.Go(func() error {
		notes, notesErr = s.Store.ListNotes(ctx, nil)
		return nil
	})
	load.Go(func() error {
		reminders, remindersErr = s.Store.ListReminders(ctx, nil)
		return nil
	})
	_ = load.Wait()
	if boardsErr != nil || tasksErr != nil {
		return ""
	}

	boardName := make(map[int64]string, len(boards))
	for _, board := range boards {
		boardName[board.ID] = board.Name
	}

	open := func(task store.Task) bool { return !task.Archived && task.CompletedAt == nil }
	children := make(map[int64][]store.Task)
	for _, task := range tasks {
		if task.ParentTaskID != nil && open(task) {
			children[*task.ParentTaskID] = append(children[*task.ParentTaskID], task)
		}
	}

	var out strings.Builder
	out.WriteString("Tasks (children are indented under their parent):\n")
	lines, topLevel := 0, 0
	var writeTask func(task store.Task, depth int)
	writeTask = func(task store.Task, depth int) {
		if lines == maxContextTaskLines {
			return
		}
		out.WriteString(strings.Repeat("  ", depth))
		out.WriteString("- ")
		if task.Ref != "" {
			out.WriteString("@")
			out.WriteString(task.Ref)
			out.WriteString(" ")
		}
		out.WriteString(task.Title)
		if task.Kind == "arc" {
			out.WriteString(" (arc)")
		}
		if board := boardName[task.BoardID]; board != "" && depth == 0 {
			out.WriteString(" [")
			out.WriteString(board)
			out.WriteString("]")
		}
		if task.DueDate != nil {
			out.WriteString(" due ")
			out.WriteString(*task.DueDate)
		}
		if task.Repeat != nil {
			out.WriteString(", repeats ")
			out.WriteString(task.Repeat.Describe())
		}
		if task.Priority != "medium" {
			out.WriteString(", ")
			out.WriteString(task.Priority)
			out.WriteString(" priority")
		}
		out.WriteString("\n")
		lines++
		if depth < maxContextTaskDepth {
			for _, child := range children[task.ID] {
				writeTask(child, depth+1)
			}
		}
	}
	for _, task := range tasks {
		if task.ParentTaskID != nil || !open(task) {
			continue
		}
		if topLevel == 80 || lines == maxContextTaskLines {
			break
		}
		writeTask(task, 0)
		topLevel++
	}
	if lines == 0 {
		out.WriteString("No open tasks.\n")
	}

	out.WriteString("\nBoard columns:\n")
	columnLines := make([]string, len(boards))
	var loadColumns errgroup.Group
	loadColumns.SetLimit(4)
	for i, board := range boards {
		i, board := i, board
		loadColumns.Go(func() error {
			columns, err := s.Store.ListColumnsByBoard(ctx, board.ID)
			if err != nil {
				return nil
			}
			columnLines[i] = "- " + board.Name + ": " + columnNames(columns) + "\n"
			return nil
		})
	}
	_ = loadColumns.Wait()
	for _, line := range columnLines {
		out.WriteString(line)
	}

	if remindersErr == nil {
		out.WriteString("\nOpen reminders:\n")
		listed := 0
		for _, reminder := range reminders {
			if reminder.CompletedAt != nil {
				continue
			}
			if listed == 30 {
				break
			}
			out.WriteString(fmt.Sprintf("- reminder_id %d: %s at %s", reminder.ID, reminder.Title, reminder.RemindAt.Format(time.RFC3339)))
			if reminder.Repeat != nil {
				out.WriteString(", repeats " + reminder.Repeat.Describe())
			}
			out.WriteString("\n")
			listed++
		}
		if listed == 0 {
			out.WriteString("No open reminders.\n")
		}
	}

	if notesErr == nil {
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
