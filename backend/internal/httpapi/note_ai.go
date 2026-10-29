package httpapi

import (
	"net/http"
	"strings"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Server) planNoteTask(w http.ResponseWriter, r *http.Request) {
	provider := s.aiProvider(r.Context())
	if provider == nil {
		writeError(w, http.StatusServiceUnavailable, "AI is not configured")
		return
	}
	id, ok := parseID(r, "noteID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid note id")
		return
	}
	note, err := s.Store.GetNote(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}
	var body struct {
		Title  string `json:"title"`
		BodyMd string `json:"body_md"`
	}
	_ = decodeJSON(r, &body)
	if strings.TrimSpace(body.Title) != "" || strings.TrimSpace(body.BodyMd) != "" {
		note.Title = body.Title
		note.BodyMd = body.BodyMd
	}
	if strings.TrimSpace(note.Title) == "" && strings.TrimSpace(note.BodyMd) == "" {
		writeError(w, http.StatusBadRequest, "note has no content")
		return
	}

	plan, err := provider.PlanTask(r.Context(), note.Title, note.BodyMd)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

type noteTaskBody struct {
	BoardID  int64    `json:"board_id"`
	Title    string   `json:"title"`
	Subtasks []string `json:"subtasks"`
	NoteBody string   `json:"note_body"`
}

func (s *Server) createTaskFromNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "noteID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid note id")
		return
	}
	note, err := s.Store.GetNote(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}

	var body noteTaskBody
	if err := decodeJSON(r, &body); err != nil || body.BoardID == 0 || strings.TrimSpace(body.Title) == "" {
		writeError(w, http.StatusBadRequest, "title and board_id are required")
		return
	}

	ctx := r.Context()
	columns, err := s.Store.ListColumnsByBoard(ctx, body.BoardID)
	if err != nil || len(columns) == 0 {
		writeError(w, http.StatusBadRequest, "board has no columns")
		return
	}
	columnID := columns[0].ID
	bestPosition := int32(1<<31 - 1)
	for _, column := range columns {
		if !column.IsDone && column.Position < bestPosition {
			columnID = column.ID
			bestPosition = column.Position
		}
	}

	position, err := s.appendPosition(ctx, columnID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	title := strings.TrimSpace(body.Title)
	description := note.BodyMd
	if strings.TrimSpace(body.NoteBody) != "" {
		description = body.NoteBody
	}
	parent, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
		BoardID:       body.BoardID,
		ColumnID:      columnID,
		Title:         title,
		DescriptionMd: description,
		Priority:      "medium",
		Position:      position,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	subtasks := make([]store.Task, 0, len(body.Subtasks))
	seen := make(map[string]struct{}, len(body.Subtasks))
	for _, raw := range body.Subtasks {
		subtaskTitle := strings.TrimSpace(raw)
		if subtaskTitle == "" {
			continue
		}
		key := strings.ToLower(subtaskTitle)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		if len(subtasks) == 30 {
			break
		}
		position, err := s.appendPosition(ctx, columnID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		subtask, err := s.Store.CreateTask(ctx, store.CreateTaskParams{
			BoardID:      body.BoardID,
			ColumnID:     columnID,
			Title:        subtaskTitle,
			Priority:     "medium",
			Position:     position,
			ParentTaskID: &parent.ID,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		subtasks = append(subtasks, subtask)
	}

	linked := parent.ID
	if _, err := s.Store.UpdateNote(ctx, note.ID, note.Title, note.BodyMd, &linked); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"task": parent, "subtasks": subtasks})
}
