package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := s.Store.ListWorkspaces(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, workspaces)
}

type workspaceBody struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	var body workspaceBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Tag = strings.TrimSpace(body.Tag)
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !store.ValidWorkspaceTag(body.Tag) {
		writeError(w, http.StatusBadRequest, "tag must be 3 or 4 letters")
		return
	}
	workspace, err := s.Store.CreateWorkspace(r.Context(), body.Name, body.Tag)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateTag) {
			writeError(w, http.StatusBadRequest, "tag is already in use")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, workspace)
}

func (s *Server) updateWorkspace(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "workspaceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	var body workspaceBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var name *string
	if strings.TrimSpace(body.Name) != "" {
		trimmed := strings.TrimSpace(body.Name)
		name = &trimmed
	}
	var tag *string
	if strings.TrimSpace(body.Tag) != "" {
		if !store.ValidWorkspaceTag(strings.TrimSpace(body.Tag)) {
			writeError(w, http.StatusBadRequest, "tag must be 3 or 4 letters")
			return
		}
		trimmed := strings.TrimSpace(body.Tag)
		tag = &trimmed
	}
	workspace, err := s.Store.UpdateWorkspace(r.Context(), id, name, tag)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "workspace not found")
		case errors.Is(err, store.ErrDuplicateTag):
			writeError(w, http.StatusBadRequest, "tag is already in use")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}
