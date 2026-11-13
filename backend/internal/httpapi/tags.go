package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Server) listTags(w http.ResponseWriter, r *http.Request) {
	tags, err := s.Store.ListTags(r.Context(), optionalInt64Param(r, "workspace_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

func (s *Server) createTag(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WorkspaceID int64  `json:"workspace_id"`
		Name        string `json:"name"`
		Color       string `json:"color"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.WorkspaceID == 0 {
		writeError(w, http.StatusBadRequest, "workspace_id is required")
		return
	}
	if body.Color == "" {
		body.Color = "#888888"
	}
	tag, err := s.Store.CreateTag(r.Context(), body.WorkspaceID, body.Name, body.Color)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tag)
}

func (s *Server) addTaskTag(w http.ResponseWriter, r *http.Request) {
	taskID, err1 := strconv.ParseInt(chi.URLParam(r, "taskID"), 10, 64)
	tagID, err2 := strconv.ParseInt(chi.URLParam(r, "tagID"), 10, 64)
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.Store.AddTaskTag(r.Context(), taskID, tagID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task or tag not found in the same workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) removeTaskTag(w http.ResponseWriter, r *http.Request) {
	taskID, err1 := strconv.ParseInt(chi.URLParam(r, "taskID"), 10, 64)
	tagID, err2 := strconv.ParseInt(chi.URLParam(r, "tagID"), 10, 64)
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.Store.RemoveTaskTag(r.Context(), taskID, tagID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
