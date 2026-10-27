package httpapi

import (
	"net/http"

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
	Prompt string `json:"prompt"`
}

func (s *Server) aiChat(w http.ResponseWriter, r *http.Request) {
	if s.AI == nil {
		writeError(w, http.StatusServiceUnavailable, "AI is not configured")
		return
	}
	var body aiChatBody
	if err := decodeJSON(r, &body); err != nil || body.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}
	reply, err := s.AI.Chat(r.Context(), body.Prompt)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"reply": reply})
}
