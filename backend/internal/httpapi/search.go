package httpapi

import "net/http"

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]any{"tasks": []any{}, "notes": []any{}})
		return
	}

	tasks, err := s.Store.SearchTasks(r.Context(), q, optionalInt64Param(r, "workspace_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	notes, err := s.Store.SearchNotes(r.Context(), q, optionalInt64Param(r, "workspace_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks, "notes": notes})
}
