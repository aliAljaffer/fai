package httpapi

import (
	"fmt"
	"net/http"
	"strings"

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

type aiChatBody struct {
	Messages []ai.ChatMessage `json:"messages"`
	Prompt   string           `json:"prompt"`
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

	system := ai.FaiSystem
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
	writeJSON(w, http.StatusOK, map[string]string{"reply": reply})
}

// faiContext gives the chat a small, bounded picture of the owner's work. Task
// bodies and note bodies stay out of the prompt.
func (s *Server) faiContext(r *http.Request) string {
	ctx := r.Context()
	boards, err := s.Store.ListBoards(ctx)
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

	notes, err := s.Store.ListNotes(ctx)
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
