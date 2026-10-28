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

func (s *Server) listMeetingOutcomes(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("pending") == "true" {
		outcomes, err := s.Store.ListPendingMeetingOutcomes(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if workspaceID := optionalInt64Param(r, "workspace_id"); workspaceID != nil {
			scoped := outcomes[:0]
			for _, outcome := range outcomes {
				if outcome.WorkspaceID == *workspaceID {
					scoped = append(scoped, outcome)
				}
			}
			outcomes = scoped
		}
		writeJSON(w, http.StatusOK, outcomes)
		return
	}

	var occurredOn *string
	if v := r.URL.Query().Get("date"); v != "" {
		if _, err := time.Parse("2006-01-02", v); err == nil {
			occurredOn = &v
		}
	}
	outcomes, err := s.Store.ListMeetingOutcomes(r.Context(), occurredOn, optionalInt64Param(r, "workspace_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, outcomes)
}

type meetingOutcomeBody struct {
	OccurredOn  string  `json:"occurred_on"`
	Title       string  `json:"title"`
	NotesMd     string  `json:"notes_md"`
	EndsAt      *string `json:"ends_at"`
	WorkspaceID int64   `json:"workspace_id"`
}

func (b meetingOutcomeBody) endsAt() *time.Time {
	if b.EndsAt == nil || *b.EndsAt == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *b.EndsAt)
	if err != nil {
		return nil
	}
	return &t
}

func (s *Server) createMeetingOutcome(w http.ResponseWriter, r *http.Request) {
	var body meetingOutcomeBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if _, err := time.Parse("2006-01-02", body.OccurredOn); err != nil {
		writeError(w, http.StatusBadRequest, "invalid occurred_on, expected YYYY-MM-DD")
		return
	}
	outcome, err := s.Store.CreateMeetingOutcome(r.Context(), body.OccurredOn, body.Title, body.NotesMd, body.endsAt(), body.WorkspaceID)
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
	outcome, err := s.Store.GetMeetingOutcome(r.Context(), id)
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
	outcome, err := s.Store.UpdateMeetingOutcome(r.Context(), id, body.Title, body.NotesMd, body.endsAt())
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
	if err := s.Store.DeleteMeetingOutcome(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type createTaskFromOutcomeBody struct {
	Title    string  `json:"title"`
	MarkDone bool    `json:"mark_done"`
	BoardID  int64   `json:"board_id"`
	DueDate  *string `json:"due_date"`
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
	if body.DueDate != nil {
		if *body.DueDate == "" {
			body.DueDate = nil
		} else if _, err := time.Parse("2006-01-02", *body.DueDate); err != nil {
			writeError(w, http.StatusBadRequest, "invalid due_date, expected YYYY-MM-DD")
			return
		}
	}

	ctx := r.Context()
	columns, err := s.Store.ListColumnsByBoard(ctx, body.BoardID)
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
	task, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
		BoardID: body.BoardID, ColumnID: columnID, Title: body.Title,
		Priority: "medium", Position: position, MeetingOutcomeID: &outcomeID,
		DueDate: body.DueDate,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

type extractedTask struct {
	Title         string  `json:"title"`
	SuggestedDone bool    `json:"suggested_done"`
	DueDate       *string `json:"due_date"`
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
	outcome, err := s.Store.GetMeetingOutcome(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "meeting outcome not found")
		return
	}

	meetingDate := outcome.OccurredOn
	if outcome.EndsAt != nil {
		meetingDate += ", ending " + outcome.EndsAt.Format("2006-01-02 15:04")
	}

	prompt := "Extract a short list of action-item tasks from these meeting notes. " +
		"The meeting took place on " + meetingDate + ". " +
		"For each, decide if it sounds already completed during the meeting (suggested_done) " +
		"or is still open. Resolve any relative date in the notes against the meeting date " +
		"and set due_date to that calendar day as yyyy-MM-dd, or null when the notes give no date. " +
		"Notes:\n\n" + outcome.NotesMd + "\n\n" +
		`Respond with ONLY a JSON array of {"title": string, "suggested_done": boolean, "due_date": string or null}, nothing else.`

	reply, err := s.AI.Chat(r.Context(), ai.FaiSystem, []ai.ChatMessage{{Role: "user", Content: prompt}})
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
	for i := range out {
		if out[i].DueDate != nil {
			if _, err := time.Parse("2006-01-02", *out[i].DueDate); err != nil {
				out[i].DueDate = nil
			}
		}
	}
	return out, nil
}
