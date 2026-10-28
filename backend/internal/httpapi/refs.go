package httpapi

import (
	"errors"
	"net/http"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Server) resolveRefs(w http.ResponseWriter, r *http.Request) {
	refs, err := s.Store.ResolveRefs(r.Context(), r.URL.Query().Get("q"), optionalInt64Param(r, "workspace_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, refs)
}

func (s *Server) listTaskReferences(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	refs, err := s.Store.ListTaskReferences(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, refs)
}
