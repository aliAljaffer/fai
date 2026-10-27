package httpapi

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

func (s *Server) listNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := s.Store.ListNotes(r.Context())
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

func (s *Server) createNote(w http.ResponseWriter, r *http.Request) {
	var body noteBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	note, err := s.Store.CreateNote(r.Context(), body.Title, body.BodyMd, body.TaskID)
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
	note, err := s.Store.GetNote(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}
	backlinks, err := s.Store.ListBacklinks(r.Context(), id, note.Title)
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
	note, err := s.Store.GetNoteByTitle(r.Context(), title)
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
	note, err := s.Store.UpdateNote(r.Context(), id, body.Title, body.BodyMd, body.TaskID)
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
	if err := s.Store.DeleteNote(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
