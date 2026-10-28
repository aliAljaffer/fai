package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/ics"
	"github.com/alialjaffer/fai/backend/internal/store"
)

const maxImportedCalendarBytes = 8 << 20

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

var calendarHTTPClient = &http.Client{Timeout: 15 * time.Second}

type importedEvent struct {
	ID       string `json:"id"`
	UID      string `json:"uid"`
	SourceID int64  `json:"source_id"`
	Title    string `json:"title"`
	Start    string `json:"start"`
	End      string `json:"end,omitempty"`
	AllDay   bool   `json:"all_day"`
	URL      string `json:"url,omitempty"`
	Location string `json:"location,omitempty"`
}

type importedCalendar struct {
	Source store.CalendarSource `json:"source"`
	Events []importedEvent      `json:"events"`
	Error  string               `json:"error,omitempty"`
}

func (s *Server) listCalendarSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.Store.ListCalendarSources(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sources)
}

func (s *Server) createCalendarSource(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Color string `json:"color"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" || len(name) > 80 {
		writeError(w, http.StatusBadRequest, "name is required and must be under 80 characters")
		return
	}
	feedURL, err := normalizeCalendarURL(body.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	color := strings.TrimSpace(body.Color)
	if color == "" {
		color = "#2563eb"
	}
	if !colorPattern.MatchString(color) {
		writeError(w, http.StatusBadRequest, "color must be a hex value like #2563eb")
		return
	}

	source, err := s.Store.CreateCalendarSource(r.Context(), name, feedURL, color)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, source)
}

// webcal is the scheme Outlook and Apple publish; the feed itself is served over https.
func normalizeCalendarURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("url must be an http, https or webcal URL")
	}
	if parsed.Scheme == "webcal" {
		parsed.Scheme = "https"
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", errors.New("url must be an http, https or webcal URL")
	}
	return parsed.String(), nil
}

func (s *Server) updateCalendarSource(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "sourceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid source id")
		return
	}
	var body struct {
		Color string `json:"color"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	color := strings.TrimSpace(body.Color)
	if !colorPattern.MatchString(color) {
		writeError(w, http.StatusBadRequest, "color must be a hex value like #2563eb")
		return
	}
	source, err := s.Store.UpdateCalendarSourceColor(r.Context(), id, color)
	if err != nil {
		writeError(w, http.StatusNotFound, "calendar source not found")
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (s *Server) deleteCalendarSource(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "sourceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid source id")
		return
	}
	if err := s.Store.DeleteCalendarSource(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "calendar source not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listImportedCalendars(w http.ResponseWriter, r *http.Request) {
	from, to, err := importedRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sources, err := s.Store.ListCalendarSources(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]importedCalendar, 0, len(sources))
	for _, source := range sources {
		item := importedCalendar{Source: source, Events: []importedEvent{}}
		events, err := s.fetchImportedEvents(r, source, from, to)
		if err != nil {
			item.Error = err.Error()
		} else {
			item.Events = events
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Source.Name < out[j].Source.Name })
	writeJSON(w, http.StatusOK, out)
}

func importedRange(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	to := from.AddDate(0, 3, 0)

	if value := r.URL.Query().Get("from"); value != "" {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("from must be YYYY-MM-DD")
		}
		from = parsed
	}
	if value := r.URL.Query().Get("to"); value != "" {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("to must be YYYY-MM-DD")
		}
		to = parsed
	}
	if !from.Before(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("from must be before to")
	}
	if to.Sub(from) > 400*24*time.Hour {
		return time.Time{}, time.Time{}, fmt.Errorf("range must be 400 days or less")
	}
	return from, to, nil
}

func (s *Server) fetchImportedEvents(r *http.Request, source store.CalendarSource, from, to time.Time) ([]importedEvent, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("could not request calendar")
	}
	req.Header.Set("Accept", "text/calendar, text/plain;q=0.8, */*;q=0.5")
	req.Header.Set("User-Agent", "fai-calendar-import/1.0")

	res, err := calendarHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch calendar")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calendar returned HTTP %d", res.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(res.Body, maxImportedCalendarBytes+1))
	if err != nil {
		return nil, fmt.Errorf("could not read calendar")
	}
	if len(data) > maxImportedCalendarBytes {
		return nil, fmt.Errorf("calendar is larger than %d bytes", maxImportedCalendarBytes)
	}

	occurrences, err := ics.Parse(data, from, to)
	if err != nil {
		return nil, fmt.Errorf("could not parse calendar")
	}

	events := make([]importedEvent, 0, len(occurrences))
	for _, occurrence := range occurrences {
		event := importedEvent{
			ID:       fmt.Sprintf("%d-%s-%d", source.ID, occurrence.UID, occurrence.Start.UnixNano()),
			UID:      occurrence.UID,
			SourceID: source.ID,
			Title:    occurrence.Summary,
			Start:    formatEventTime(occurrence.Start, occurrence.AllDay),
			AllDay:   occurrence.AllDay,
			URL:      occurrence.URL,
			Location: occurrence.Location,
		}
		if occurrence.End.After(occurrence.Start) {
			event.End = formatEventTime(occurrence.End, occurrence.AllDay)
		}
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].Start == events[j].Start {
			return events[i].Title < events[j].Title
		}
		return events[i].Start < events[j].Start
	})
	return events, nil
}

func formatEventTime(t time.Time, allDay bool) string {
	if allDay {
		return t.Format("2006-01-02")
	}
	return t.Format(time.RFC3339)
}
