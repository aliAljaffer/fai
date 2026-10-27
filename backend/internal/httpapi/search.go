package httpapi

import "net/http"

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]any{"tasks": []any{}, "notes": []any{}})
		return
	}

	tasks, err := s.Q.SearchTasks(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	notes, err := s.Q.SearchNotes(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks, "notes": notes})
}
