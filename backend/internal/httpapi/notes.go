package httpapi

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
)

func (s *Server) listNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := s.Q.ListNotes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

type noteBody struct {
	Title  string `json:"title"`
	BodyMd string `json:"body_md"`
	TaskID *int64 `json:"task_id"`
}

func (b noteBody) taskID() pgtype.Int8 {
	if b.TaskID == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *b.TaskID, Valid: true}
}

func (s *Server) createNote(w http.ResponseWriter, r *http.Request) {
	var body noteBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	note, err := s.Q.CreateNote(r.Context(), sqlc.CreateNoteParams{
		Title:  body.Title,
		BodyMd: body.BodyMd,
		TaskID: body.taskID(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func (s *Server) getNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "noteID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid note id")
		return
	}
	note, err := s.Q.GetNote(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}
	backlinks, err := s.Q.ListBacklinks(r.Context(), sqlc.ListBacklinksParams{ID: id, Column2: note.Title})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"note": note, "backlinks": backlinks})
}

func (s *Server) getNoteByTitle(w http.ResponseWriter, r *http.Request) {
	title, err := url.PathUnescape(chi.URLParam(r, "title"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid title")
		return
	}
	note, err := s.Q.GetNoteByTitle(r.Context(), title)
	if err != nil {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (s *Server) updateNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "noteID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid note id")
		return
	}
	var body noteBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	note, err := s.Q.UpdateNote(r.Context(), sqlc.UpdateNoteParams{
		ID:     id,
		Title:  body.Title,
		BodyMd: body.BodyMd,
		TaskID: body.taskID(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (s *Server) deleteNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "noteID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid note id")
		return
	}
	if err := s.Q.DeleteNote(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
