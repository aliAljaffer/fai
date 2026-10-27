package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) listBoards(w http.ResponseWriter, r *http.Request) {
	boards, err := s.Store.ListBoards(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, boards)
}

func (s *Server) createBoard(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	board, err := s.Store.CreateBoard(r.Context(), body.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, board)
}

func (s *Server) listColumns(w http.ResponseWriter, r *http.Request) {
	boardID, err := strconv.ParseInt(chi.URLParam(r, "boardID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}
	cols, err := s.Store.ListColumnsByBoard(r.Context(), boardID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cols)
}

func (s *Server) createColumn(w http.ResponseWriter, r *http.Request) {
	boardID, err := strconv.ParseInt(chi.URLParam(r, "boardID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}
	var body struct {
		Name     string `json:"name"`
		Position int32  `json:"position"`
		IsDone   bool   `json:"is_done"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	col, err := s.Store.CreateColumn(r.Context(), boardID, body.Name, body.Position, body.IsDone)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, col)
}
