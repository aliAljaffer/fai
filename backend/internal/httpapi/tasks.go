package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
)

const positionGap = 1024.0
const positionEpsilon = 1e-6

func parseID(r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	return id, err == nil
}

func optionalInt64Param(r *http.Request, name string) pgtype.Int8 {
	v := r.URL.Query().Get(name)
	if v == "" {
		return pgtype.Int8{}
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: id, Valid: true}
}

func optionalTextParam(r *http.Request, name string) pgtype.Text {
	v := r.URL.Query().Get(name)
	if v == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.Q.ListTasks(r.Context(), sqlc.ListTasksParams{
		BoardID:  optionalInt64Param(r, "board_id"),
		ColumnID: optionalInt64Param(r, "column_id"),
		Tag:      optionalTextParam(r, "tag"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

type taskBody struct {
	BoardID       int64   `json:"board_id"`
	ColumnID      int64   `json:"column_id"`
	Title         string  `json:"title"`
	DescriptionMd string  `json:"description_md"`
	Priority      string  `json:"priority"`
	DueDate       *string `json:"due_date"`
}

func (b taskBody) dueDate() pgtype.Date {
	if b.DueDate == nil || *b.DueDate == "" {
		return pgtype.Date{}
	}
	d, err := time.Parse("2006-01-02", *b.DueDate)
	if err != nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: d, Valid: true}
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var body taskBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Priority == "" {
		body.Priority = "medium"
	}

	existing, err := s.Q.ListTasksByColumnOrdered(r.Context(), body.ColumnID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	position := positionGap
	if len(existing) > 0 {
		position = existing[len(existing)-1].Position + positionGap
	}

	task, err := s.Q.CreateTask(r.Context(), sqlc.CreateTaskParams{
		BoardID:       body.BoardID,
		ColumnID:      body.ColumnID,
		Title:         body.Title,
		DescriptionMd: body.DescriptionMd,
		Priority:      body.Priority,
		DueDate:       body.dueDate(),
		Position:      position,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	task, err := s.Q.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var body taskBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	task, err := s.Q.UpdateTask(r.Context(), sqlc.UpdateTaskParams{
		ID:            id,
		Title:         body.Title,
		DescriptionMd: body.DescriptionMd,
		Priority:      body.Priority,
		DueDate:       body.dueDate(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	if err := s.Q.ArchiveTask(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type moveBody struct {
	ColumnID     int64  `json:"column_id"`
	AfterTaskID  *int64 `json:"after_task_id"`
	BeforeTaskID *int64 `json:"before_task_id"`
}

func (s *Server) moveTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var body moveBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	ctx := r.Context()
	siblings, err := s.Q.ListTasksByColumnOrdered(ctx, body.ColumnID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Exclude the task being moved from its own neighbor calculation (same-column reorder).
	filtered := make([]idPos, 0, len(siblings))
	for _, t := range siblings {
		if t.ID != id {
			filtered = append(filtered, idPos{ID: t.ID, Position: t.Position})
		}
	}

	position, needsRenumber := computePosition(filtered, body.AfterTaskID, body.BeforeTaskID)
	if needsRenumber {
		for i, t := range filtered {
			newPos := float64(i+1) * positionGap
			if err := s.Q.SetTaskPosition(ctx, sqlc.SetTaskPositionParams{ID: t.ID, Position: newPos}); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		position, _ = computePosition(filtered, body.AfterTaskID, body.BeforeTaskID)
	}

	task, err := s.Q.MoveTask(ctx, sqlc.MoveTaskParams{ID: id, ColumnID: body.ColumnID, Position: position})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

type idPos struct {
	ID       int64
	Position float64
}

// computePosition finds the midpoint position for inserting between afterID and beforeID
// (both optional) among siblings already ordered by position. Returns needsRenumber=true
// if the gap between neighbors has collapsed below positionEpsilon.
func computePosition(siblings []idPos, afterID, beforeID *int64) (float64, bool) {
	var lo, hi *float64
	for _, t := range siblings {
		tp := t.Position
		if afterID != nil && t.ID == *afterID {
			lo = &tp
		}
		if beforeID != nil && t.ID == *beforeID {
			hi = &tp
		}
	}
	switch {
	case lo == nil && hi == nil:
		if len(siblings) == 0 {
			return positionGap, false
		}
		return siblings[len(siblings)-1].Position + positionGap, false
	case lo == nil:
		return *hi / 2, false
	case hi == nil:
		return *lo + positionGap, false
	default:
		gap := *hi - *lo
		if gap < positionEpsilon {
			return 0, true
		}
		return *lo + gap/2, false
	}
}
