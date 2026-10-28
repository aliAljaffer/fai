package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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

// CalendarSyncInterval is how often cached external feeds refresh in the
// background, and how stale a feed may be before opening the calendar refetches it.
const CalendarSyncInterval = 15 * time.Minute

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
	Source       store.CalendarSource `json:"source"`
	Events       []importedEvent      `json:"events"`
	Error        string               `json:"error,omitempty"`
	LastSyncedAt string               `json:"last_synced_at,omitempty"`
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
		data, fetchedAt, err := s.calendarFeedData(r.Context(), source, false)
		if !fetchedAt.IsZero() {
			item.LastSyncedAt = fetchedAt.UTC().Format(time.RFC3339)
		}
		if err == nil {
			item.Events, err = parseImportedEvents(source, data, from, to)
		}
		if err != nil {
			item.Error = err.Error()
		} else {
			item.Error = s.calendarSyncError(source.ID)
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

// calendarFeedData returns the stored feed, refetching it over https when it is
// missing, older than CalendarSyncInterval, or force is set. A failed refresh
// keeps serving the stored copy so a flaky feed does not blank the calendar.
func (s *Server) calendarFeedData(ctx context.Context, source store.CalendarSource, force bool) ([]byte, time.Time, error) {
	data, fetchedAt, ok, err := s.Store.GetCalendarFeed(ctx, source.ID)
	if err != nil {
		return nil, time.Time{}, err
	}
	if ok && !force && time.Since(fetchedAt) < CalendarSyncInterval {
		return data, fetchedAt, nil
	}
	fresh, err := fetchCalendarFeed(ctx, source)
	if err != nil {
		s.recordCalendarError(source.ID, err)
		if ok {
			return data, fetchedAt, nil
		}
		return nil, time.Time{}, err
	}
	s.recordCalendarError(source.ID, nil)
	fetchedAt = time.Now().UTC()
	if err := s.Store.SetCalendarFeed(ctx, source.ID, fresh, fetchedAt); err != nil {
		log.Printf("storing calendar feed %s: %v", source.Name, err)
	}
	return fresh, fetchedAt, nil
}

func (s *Server) recordCalendarError(sourceID int64, err error) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	if s.calendarErr == nil {
		s.calendarErr = map[int64]string{}
	}
	if err == nil {
		delete(s.calendarErr, sourceID)
		return
	}
	s.calendarErr[sourceID] = err.Error()
}

func (s *Server) calendarSyncError(sourceID int64) string {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	return s.calendarErr[sourceID]
}

// RunCalendarSync keeps stored Outlook and other iCalendar feeds current by
// refreshing them in the background until ctx is done.
func (s *Server) RunCalendarSync(ctx context.Context) {
	s.syncCalendarFeeds(ctx)
	ticker := time.NewTicker(CalendarSyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncCalendarFeeds(ctx)
		}
	}
}

func (s *Server) syncCalendarFeeds(ctx context.Context) {
	sources, err := s.Store.ListCalendarSources(ctx)
	if err != nil {
		log.Printf("listing calendar feeds for sync: %v", err)
		return
	}
	for _, source := range sources {
		if _, _, err := s.calendarFeedData(ctx, source, true); err != nil {
			log.Printf("syncing calendar feed %s: %v", source.Name, err)
		}
	}
}

func fetchCalendarFeed(ctx context.Context, source store.CalendarSource) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
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
	return data, nil
}

func parseImportedEvents(source store.CalendarSource, data []byte, from, to time.Time) ([]importedEvent, error) {
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
