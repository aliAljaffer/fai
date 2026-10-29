package httpapi

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/ics"
	"github.com/alialjaffer/fai/backend/internal/store"
)

// calendarFeed is reachable without OIDC because calendar clients cannot complete
// a browser login; the unguessable token in the URL is the credential instead.
func (s *Server) calendarFeed(w http.ResponseWriter, r *http.Request) {
	if s.ICSToken == "" {
		writeError(w, http.StatusServiceUnavailable, "calendar feed is not configured")
		return
	}
	token := r.URL.Query().Get("token")
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.ICSToken)) != 1 {
		writeError(w, http.StatusUnauthorized, "invalid calendar token")
		return
	}

	tasks, err := s.Store.ListTasks(r.Context(), store.ListTasksParams{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var workspaceID int64
	if raw := r.URL.Query().Get("workspace_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid workspace id")
			return
		}
		workspaceID = parsed
	}

	base := s.feedBaseURL(r)
	events := make([]ics.Event, 0, len(tasks))
	for _, t := range tasks {
		if workspaceID != 0 && t.WorkspaceID != workspaceID {
			continue
		}
		if t.DueDate == nil || t.CompletedAt != nil {
			continue
		}
		due, err := time.Parse("2006-01-02", *t.DueDate)
		if err != nil {
			continue
		}
		events = append(events, ics.Event{
			UID:         fmt.Sprintf("task-%d@fai", t.ID),
			Date:        due,
			Summary:     t.Title,
			Description: "Priority: " + t.Priority,
			URL:         fmt.Sprintf("%s/?task=%d", base, t.ID),
			Priority:    icsPriority(t.Priority),
			Updated:     t.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="fai.ics"`)
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Write([]byte(ics.Calendar("fai tasks", events, time.Now())))
}

// calendarFeedInfo hands the authenticated web UI the subscription URL to show.
func (s *Server) calendarFeedInfo(w http.ResponseWriter, r *http.Request) {
	if s.ICSToken == "" {
		writeError(w, http.StatusServiceUnavailable, "calendar feed is not configured")
		return
	}
	feed := s.feedBaseURL(r) + "/api/calendar.ics?token=" + url.QueryEscape(s.ICSToken)
	if raw := r.URL.Query().Get("workspace_id"); raw != "" {
		if _, err := strconv.ParseInt(raw, 10, 64); err != nil {
			writeError(w, http.StatusBadRequest, "invalid workspace id")
			return
		}
		feed += "&workspace_id=" + url.QueryEscape(raw)
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": feed})
}

func (s *Server) feedBaseURL(r *http.Request) string {
	if s.PublicURL != "" {
		return strings.TrimSuffix(s.PublicURL, "/")
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func icsPriority(priority string) int {
	switch priority {
	case "high":
		return 1
	case "medium":
		return 5
	case "low":
		return 9
	default:
		return 0
	}
}
