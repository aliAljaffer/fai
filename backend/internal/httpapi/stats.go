package httpapi

import (
	"net/http"
	"time"
)

func dateParam(r *http.Request, name string, fallback time.Time) string {
	v := r.URL.Query().Get(name)
	if v == "" {
		return fallback.Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", v); err != nil {
		return fallback.Format("2006-01-02")
	}
	return v
}

func (s *Server) statsCompletions(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := dateParam(r, "from", now.AddDate(0, 0, -365))
	to := dateParam(r, "to", now)

	rows, err := s.Store.CompletionsByDay(r.Context(), from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}
