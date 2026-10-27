package httpapi

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
)

func parseDateParam(r *http.Request, name string, fallback time.Time) pgtype.Date {
	v := r.URL.Query().Get(name)
	if v == "" {
		return pgtype.Date{Time: fallback, Valid: true}
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return pgtype.Date{Time: fallback, Valid: true}
	}
	return pgtype.Date{Time: t, Valid: true}
}

func (s *Server) statsCompletions(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := parseDateParam(r, "from", now.AddDate(0, 0, -365))
	to := parseDateParam(r, "to", now)

	rows, err := s.Q.CompletionsByDay(r.Context(), sqlc.CompletionsByDayParams{FromDate: from, ToDate: to})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type point struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	out := make([]point, 0, len(rows))
	for _, row := range rows {
		out = append(out, point{Date: row.Day.Time.Format("2006-01-02"), Count: row.Count})
	}
	writeJSON(w, http.StatusOK, out)
}
