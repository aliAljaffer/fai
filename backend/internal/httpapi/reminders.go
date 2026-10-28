package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/ai"
)

func (s *Server) listReminders(w http.ResponseWriter, r *http.Request) {
	reminders, err := s.Store.ListReminders(r.Context(), optionalInt64Param(r, "workspace_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reminders)
}

type reminderBody struct {
	Title       string `json:"title"`
	RemindAt    string `json:"remind_at"`
	TaskID      *int64 `json:"task_id"`
	WorkspaceID int64  `json:"workspace_id"`
}

func (b reminderBody) remindAt() (time.Time, error) {
	return time.Parse(time.RFC3339, b.RemindAt)
}

func (s *Server) createReminder(w http.ResponseWriter, r *http.Request) {
	var body reminderBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	remindAt, err := body.remindAt()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid remind_at, expected RFC3339")
		return
	}
	reminder, err := s.Store.CreateReminder(r.Context(), body.Title, remindAt, body.TaskID, body.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, reminder)
}

func (s *Server) updateReminder(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "reminderID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid reminder id")
		return
	}
	var body reminderBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	remindAt, err := body.remindAt()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid remind_at, expected RFC3339")
		return
	}
	reminder, err := s.Store.UpdateReminder(r.Context(), id, body.Title, remindAt, body.TaskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reminder)
}

func (s *Server) deleteReminder(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "reminderID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid reminder id")
		return
	}
	if err := s.Store.DeleteReminder(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func aiReminderPrompt(text string, now time.Time) string {
	return fmt.Sprintf(
		"Turn this message into one reminder. The current local time is %s.\n"+
			"Message: %q\n\n"+
			`Respond with ONLY JSON in this shape: {"title": string, "remind_at": string}. `+
			"remind_at must be RFC3339 with the same timezone offset as the current local time. "+
			"If the message gives a day but no time, use 09:00. If it gives no time at all, use one hour from now. "+
			"Keep the title short and imperative, without the reminder time.",
		now.Format(time.RFC3339), text,
	)
}

func parseAIReminder(text string) (string, time.Time, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end < start {
		return "", time.Time{}, fmt.Errorf("ai: no JSON object found in response")
	}
	var parsed struct {
		Title    string `json:"title"`
		RemindAt string `json:"remind_at"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &parsed); err != nil {
		return "", time.Time{}, fmt.Errorf("ai: invalid JSON object in response: %w", err)
	}
	title := strings.TrimSpace(parsed.Title)
	if title == "" {
		return "", time.Time{}, fmt.Errorf("ai: response has no reminder title")
	}
	remindAt, err := time.Parse(time.RFC3339, parsed.RemindAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("ai: response has an invalid remind_at")
	}
	return title, remindAt, nil
}

func localNow(offset *int) time.Time {
	minutes := 0
	if offset != nil && *offset >= -14*60 && *offset <= 14*60 {
		minutes = *offset
	}
	return time.Now().In(time.FixedZone("local", minutes*60))
}

func (s *Server) aiReminder(w http.ResponseWriter, r *http.Request) {
	if s.AI == nil {
		writeError(w, http.StatusServiceUnavailable, "AI is not configured")
		return
	}
	var body struct {
		Text     string `json:"text"`
		TZOffset *int   `json:"tz_offset"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	text := strings.TrimSpace(body.Text)
	if text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	if len([]rune(text)) > 500 {
		writeError(w, http.StatusBadRequest, "text is too long")
		return
	}

	now := localNow(body.TZOffset)

	reply, err := s.AI.Chat(r.Context(), ai.FaiSystem, []ai.ChatMessage{{
		Role:    "user",
		Content: aiReminderPrompt(text, now),
	}})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	title, remindAt, err := parseAIReminder(reply)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	reminder, err := s.Store.CreateReminder(r.Context(), title, remindAt, nil, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, reminder)
}
