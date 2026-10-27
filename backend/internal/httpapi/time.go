package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Server) startTime(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	entry, err := s.Store.StartTimer(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (s *Server) stopTime(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	entry, err := s.Store.StopEntryForTask(r.Context(), taskID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusConflict, "no running timer for this task")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) runningTime(w http.ResponseWriter, r *http.Request) {
	entry, err := s.Store.GetRunningEntry(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) listTimeForTask(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	entries, err := s.Store.ListEntriesForTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) statsTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := dateParam(r, "from", now.AddDate(0, 0, -30))
	to := dateParam(r, "to", now)

	rows, err := s.Store.TimeByDay(r.Context(), from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}
