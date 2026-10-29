package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/ai"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func aiTaskPrompt(text string, now time.Time) string {
	return fmt.Sprintf(
		"Turn this message into one task. The current local time is %s.\n"+
			"Message: %q\n\n"+
			`Respond with ONLY JSON in this shape: {"title": string, "due_date": string, "priority": string}. `+
			"due_date must be YYYY-MM-DD in the same local timezone, or an empty string when the message gives no date. "+
			"Resolve relative dates (today, tomorrow, Monday, next week) against the current local time. "+
			"A weekday alone means the next such day, or today when it is that day. "+
			"priority must be high, medium, or low. Use medium when the message does not say. "+
			"Keep the title short and imperative, without date or priority words.",
		now.Format(time.RFC3339), text,
	)
}

func parseAITask(text string) (string, *string, string, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end < start {
		return "", nil, "", fmt.Errorf("ai: no JSON object found in response")
	}
	var parsed struct {
		Title    string `json:"title"`
		DueDate  string `json:"due_date"`
		Priority string `json:"priority"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &parsed); err != nil {
		return "", nil, "", fmt.Errorf("ai: invalid JSON object in response: %w", err)
	}
	title := strings.TrimSpace(parsed.Title)
	if title == "" {
		return "", nil, "", fmt.Errorf("ai: response has no task title")
	}
	var due *string
	if value := strings.TrimSpace(parsed.DueDate); value != "" {
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return "", nil, "", fmt.Errorf("ai: response has an invalid due_date")
		}
		due = &value
	}
	priority := strings.ToLower(strings.TrimSpace(parsed.Priority))
	switch priority {
	case "high", "medium", "low":
	default:
		priority = "medium"
	}
	return title, due, priority, nil
}

func (s *Server) aiTask(w http.ResponseWriter, r *http.Request) {
	provider := s.aiProvider(r.Context())
	if provider == nil {
		writeError(w, http.StatusServiceUnavailable, "AI is not configured")
		return
	}
	var body struct {
		Text        string `json:"text"`
		TZOffset    *int   `json:"tz_offset"`
		BoardID     int64  `json:"board_id"`
		ColumnID    int64  `json:"column_id"`
		WorkspaceID int64  `json:"workspace_id"`
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
	if body.WorkspaceID != 0 {
		if _, err := s.Store.GetWorkspace(r.Context(), body.WorkspaceID); err != nil {
			writeError(w, http.StatusBadRequest, "unknown workspace_id")
			return
		}
	}

	now := localNow(body.TZOffset)
	reply, err := provider.Chat(r.Context(), ai.FaiSystem, []ai.ChatMessage{{
		Role:    "user",
		Content: aiTaskPrompt(text, now),
	}})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	title, due, priority, err := parseAITask(reply)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	boardID, columnID := body.BoardID, body.ColumnID
	if boardID == 0 && columnID == 0 {
		boardID, columnID, err = s.defaultBoardColumn(r.Context())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	} else if boardID == 0 || columnID == 0 {
		writeError(w, http.StatusBadRequest, "board_id and column_id must both be set")
		return
	}
	position, err := s.appendPosition(r.Context(), columnID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	task, err := s.Store.CreateTask(r.Context(), store.CreateTaskParams{
		BoardID:      boardID,
		ColumnID:     columnID,
		Title:        title,
		Priority:     priority,
		DueDate:      due,
		Position:     position,
		CreatedByFai: true,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, task)
}
