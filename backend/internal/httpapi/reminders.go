package httpapi

import (
	"net/http"
	"time"
)

func (s *Server) listReminders(w http.ResponseWriter, r *http.Request) {
	reminders, err := s.Store.ListReminders(r.Context())
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
	reminder, err := s.Store.CreateReminder(r.Context(), body.Title, remindAt, body.TaskID)
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
	reminder, err := s.Store.UpdateReminder(r.Context(), id, body.Title, remindAt, body.TaskID)
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
	if err := s.Store.DeleteReminder(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
