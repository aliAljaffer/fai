package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
)

func (s *Server) listMeetingOutcomes(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("pending") == "true" {
		outcomes, err := s.Q.ListPendingMeetingOutcomes(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, outcomes)
		return
	}

	var occurredOn pgtype.Date
	if v := r.URL.Query().Get("date"); v != "" {
		if d, err := time.Parse("2006-01-02", v); err == nil {
			occurredOn = pgtype.Date{Time: d, Valid: true}
		}
	}
	outcomes, err := s.Q.ListMeetingOutcomes(r.Context(), occurredOn)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, outcomes)
}

type meetingOutcomeBody struct {
	OccurredOn string  `json:"occurred_on"`
	Title      string  `json:"title"`
	NotesMd    string  `json:"notes_md"`
	EndsAt     *string `json:"ends_at"`
}

func (b meetingOutcomeBody) endsAt() pgtype.Timestamptz {
	if b.EndsAt == nil || *b.EndsAt == "" {
		return pgtype.Timestamptz{}
	}
	t, err := time.Parse(time.RFC3339, *b.EndsAt)
	if err != nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func (s *Server) createMeetingOutcome(w http.ResponseWriter, r *http.Request) {
	var body meetingOutcomeBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	occurredOn, err := time.Parse("2006-01-02", body.OccurredOn)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid occurred_on, expected YYYY-MM-DD")
		return
	}
	outcome, err := s.Q.CreateMeetingOutcome(r.Context(), sqlc.CreateMeetingOutcomeParams{
		OccurredOn: pgtype.Date{Time: occurredOn, Valid: true},
		Title:      body.Title,
		NotesMd:    body.NotesMd,
		EndsAt:     body.endsAt(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, outcome)
}

func (s *Server) getMeetingOutcome(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "outcomeID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	outcome, err := s.Q.GetMeetingOutcome(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "meeting outcome not found")
		return
	}
	writeJSON(w, http.StatusOK, outcome)
}

func (s *Server) updateMeetingOutcome(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "outcomeID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body meetingOutcomeBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	outcome, err := s.Q.UpdateMeetingOutcome(r.Context(), sqlc.UpdateMeetingOutcomeParams{
		ID:      id,
		Title:   body.Title,
		NotesMd: body.NotesMd,
		EndsAt:  body.endsAt(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, outcome)
}

func (s *Server) deleteMeetingOutcome(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "outcomeID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.Q.DeleteMeetingOutcome(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type createTaskFromOutcomeBody struct {
	Title    string `json:"title"`
	MarkDone bool   `json:"mark_done"`
	BoardID  int64  `json:"board_id"`
}

func (s *Server) createTaskFromOutcome(w http.ResponseWriter, r *http.Request) {
	outcomeID, ok := parseID(r, "outcomeID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body createTaskFromOutcomeBody
	if err := decodeJSON(r, &body); err != nil || body.Title == "" || body.BoardID == 0 {
		writeError(w, http.StatusBadRequest, "title and board_id are required")
		return
	}

	ctx := r.Context()
	columns, err := s.Q.ListColumnsByBoard(ctx, body.BoardID)
	if err != nil || len(columns) == 0 {
		writeError(w, http.StatusInternalServerError, "board has no columns")
		return
	}
	columnID := columns[0].ID
	if body.MarkDone {
		for _, c := range columns {
			if c.IsDone {
				columnID = c.ID
				break
			}
		}
	}

	position, err := s.appendPosition(ctx, columnID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	task, err := s.Q.CreateTask(ctx, sqlc.CreateTaskParams{
		BoardID:          body.BoardID,
		ColumnID:         columnID,
		Title:            body.Title,
		Priority:         "medium",
		Position:         position,
		MeetingOutcomeID: pgtypeInt8(outcomeID),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

type extractedTask struct {
	Title         string `json:"title"`
	SuggestedDone bool   `json:"suggested_done"`
}

func (s *Server) extractTasksFromOutcome(w http.ResponseWriter, r *http.Request) {
	if s.AI == nil {
		writeError(w, http.StatusServiceUnavailable, "AI is not configured")
		return
	}
	id, ok := parseID(r, "outcomeID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	outcome, err := s.Q.GetMeetingOutcome(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "meeting outcome not found")
		return
	}

	prompt := "Extract a short list of action-item tasks from these meeting notes. " +
		"For each, decide if it sounds already completed during the meeting (suggested_done) " +
		"or is still open. Notes:\n\n" + outcome.NotesMd + "\n\n" +
		`Respond with ONLY a JSON array of {"title": string, "suggested_done": boolean}, nothing else.`

	reply, err := s.AI.Chat(r.Context(), prompt)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	tasks, err := parseExtractedTasks(reply)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

// parseExtractedTasks tolerates surrounding prose or a markdown code fence around the
// JSON array, since models don't always follow "respond with only JSON" exactly.
func parseExtractedTasks(text string) ([]extractedTask, error) {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("ai: no JSON array found in response")
	}
	var out []extractedTask
	if err := json.Unmarshal([]byte(text[start:end+1]), &out); err != nil {
		return nil, fmt.Errorf("ai: invalid JSON array in response: %w", err)
	}
	return out, nil
}
