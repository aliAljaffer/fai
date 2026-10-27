package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
)

func (s *Server) startTime(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback(ctx)

	q := s.Q.WithTx(tx)
	if err := q.StopRunningEntry(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	entry, err := q.StartEntry(ctx, taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(ctx); err != nil {
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
	entry, err := s.Q.StopEntryForTask(r.Context(), taskID)
	if errors.Is(err, pgx.ErrNoRows) {
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
	entry, err := s.Q.GetRunningEntry(r.Context())
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusOK, nil)
		return
	}
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
	entries, err := s.Q.ListEntriesForTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) statsTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := parseDateParam(r, "from", now.AddDate(0, 0, -30))
	to := parseDateParam(r, "to", now)

	rows, err := s.Q.TimeByDay(r.Context(), sqlc.TimeByDayParams{FromDate: from, ToDate: to})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type point struct {
		Date    string `json:"date"`
		Seconds int64  `json:"seconds"`
	}
	out := make([]point, 0, len(rows))
	for _, row := range rows {
		out = append(out, point{Date: row.Day.Time.Format("2006-01-02"), Seconds: row.Seconds})
	}
	writeJSON(w, http.StatusOK, out)
}
