package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/alialjaffer/fai/backend/internal/store"
)

const positionGap = 1024.0
const positionEpsilon = 1e-6

func parseID(r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	return id, err == nil
}

func optionalInt64Param(r *http.Request, name string) *int64 {
	v := r.URL.Query().Get(name)
	if v == "" {
		return nil
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &id
}

func optionalTextParam(r *http.Request, name string) *string {
	v := r.URL.Query().Get(name)
	if v == "" {
		return nil
	}
	return &v
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.Store.ListTasks(r.Context(), store.ListTasksParams{
		BoardID:  optionalInt64Param(r, "board_id"),
		ColumnID: optionalInt64Param(r, "column_id"),
		Tag:      optionalTextParam(r, "tag"),
		Q:        optionalTextParam(r, "q"),
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
	ParentTaskID  *int64  `json:"parent_task_id"`
	Kind          string  `json:"kind"`
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
	body.Kind = store.NormalizeKind(body.Kind, body.ParentTaskID != nil)
	if err := s.validateTaskType(r.Context(), 0, body.Kind, body.ParentTaskID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	position, err := s.appendPosition(r.Context(), body.ColumnID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	task, err := s.Store.CreateTask(r.Context(), store.CreateTaskParams{
		BoardID:       body.BoardID,
		ColumnID:      body.ColumnID,
		Title:         body.Title,
		DescriptionMd: body.DescriptionMd,
		Priority:      body.Priority,
		DueDate:       body.DueDate,
		Position:      position,
		ParentTaskID:  body.ParentTaskID,
		Kind:          body.Kind,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

type taskTypeBody struct {
	Kind         string `json:"kind"`
	ParentTaskID *int64 `json:"parent_task_id"`
}

func (s *Server) setTaskType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var body taskTypeBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Kind != "arc" && body.Kind != "task" && body.Kind != "step" {
		writeError(w, http.StatusBadRequest, "kind must be arc, task or step")
		return
	}
	if err := s.validateTaskType(r.Context(), id, body.Kind, body.ParentTaskID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()
	if body.Kind == "step" || body.Kind == "task" {
		if err := s.Store.SetTaskParent(ctx, id, body.ParentTaskID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else if err := s.Store.SetTaskParent(ctx, id, nil); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.Store.SetTaskKind(ctx, id, body.Kind); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	task, err := s.Store.GetTask(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// validateTaskType keeps the three levels honest: an arc is always top level,
// a step always has a parent, and a step never becomes a container.
func (s *Server) validateTaskType(ctx context.Context, taskID int64, kind string, parentID *int64) error {
	if kind == "arc" && parentID != nil {
		return fmt.Errorf("an arc cannot have a parent")
	}
	if kind == "step" && parentID == nil {
		return fmt.Errorf("a step needs a parent")
	}
	if parentID != nil {
		if *parentID == taskID {
			return fmt.Errorf("a task cannot be its own parent")
		}
		parent, err := s.Store.GetTask(ctx, *parentID)
		if err != nil {
			return fmt.Errorf("parent task not found")
		}
		if parent.Kind == "step" {
			return fmt.Errorf("a step cannot contain tasks")
		}
		if kind == "task" && parent.Kind != "arc" {
			return fmt.Errorf("a task under another task must be a step")
		}
		if kind == "step" && parent.Kind == "arc" {
			return fmt.Errorf("a step belongs to a task, not an arc")
		}
	}
	if kind == "step" && taskID != 0 {
		children, err := s.Store.ListSubtasks(ctx, taskID)
		if err != nil {
			return err
		}
		if len(children) > 0 {
			return fmt.Errorf("a task with children cannot become a step")
		}
	}
	return nil
}

func (s *Server) listSubtasks(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	subtasks, err := s.Store.ListSubtasks(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, subtasks)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	task, err := s.Store.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

type taskPatchBody struct {
	Title         *string         `json:"title"`
	DescriptionMd *string         `json:"description_md"`
	Priority      *string         `json:"priority"`
	DueDate       json.RawMessage `json:"due_date"`
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var body taskPatchBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	patch := store.TaskPatch{
		Title:         body.Title,
		DescriptionMd: body.DescriptionMd,
		Priority:      body.Priority,
	}
	if body.DueDate != nil {
		patch.DueDateSet = true
		if string(body.DueDate) != "null" {
			var due string
			if err := json.Unmarshal(body.DueDate, &due); err != nil {
				writeError(w, http.StatusBadRequest, "due_date must be a date string or null")
				return
			}
			patch.DueDate = &due
		}
	}
	if patch.Title == nil && patch.DescriptionMd == nil && patch.Priority == nil && !patch.DueDateSet {
		writeError(w, http.StatusBadRequest, "no fields to update")
		return
	}
	task, err := s.Store.PatchTask(r.Context(), id, patch)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) restoreTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	if err := s.Store.RestoreTask(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	task, err := s.Store.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
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
	if err := s.Store.ArchiveTask(r.Context(), id); err != nil {
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
	siblings, err := s.Store.ListTasksByColumnOrdered(ctx, body.ColumnID)
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
			if err := s.Store.SetTaskPosition(ctx, t.ID, newPos); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		position, _ = computePosition(filtered, body.AfterTaskID, body.BeforeTaskID)
	}

	task, err := s.Store.MoveTask(ctx, id, body.ColumnID, position)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// appendPosition returns the position for a new task at the end of columnID.
func (s *Server) appendPosition(ctx context.Context, columnID int64) (float64, error) {
	existing, err := s.Store.ListTasksByColumnOrdered(ctx, columnID)
	if err != nil {
		return 0, err
	}
	if len(existing) == 0 {
		return positionGap, nil
	}
	return existing[len(existing)-1].Position + positionGap, nil
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
