package httpapi

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
)

func (s *Server) listReminders(w http.ResponseWriter, r *http.Request) {
	reminders, err := s.Q.ListReminders(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reminders)
}

type reminderBody struct {
	Title    string `json:"title"`
	RemindAt string `json:"remind_at"`
	TaskID   *int64 `json:"task_id"`
}

func (b reminderBody) taskID() pgtype.Int8 {
	if b.TaskID == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *b.TaskID, Valid: true}
}

func (b reminderBody) remindAt() (time.Time, error) {
	return time.Parse(time.RFC3339, b.RemindAt)
}

func (s *Server) createReminder(w http.ResponseWriter, r *http.Request) {
	var body reminderBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	remindAt, err := body.remindAt()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid remind_at, expected RFC3339")
		return
	}
	reminder, err := s.Q.CreateReminder(r.Context(), sqlc.CreateReminderParams{
		Title:    body.Title,
		RemindAt: remindAt,
		TaskID:   body.taskID(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, reminder)
}

func (s *Server) updateReminder(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "reminderID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid reminder id")
		return
	}
	var body reminderBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	remindAt, err := body.remindAt()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid remind_at, expected RFC3339")
		return
	}
	reminder, err := s.Q.UpdateReminder(r.Context(), sqlc.UpdateReminderParams{
		ID:       id,
		Title:    body.Title,
		RemindAt: remindAt,
		TaskID:   body.taskID(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reminder)
}

func (s *Server) deleteReminder(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "reminderID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid reminder id")
		return
	}
	if err := s.Q.DeleteReminder(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
