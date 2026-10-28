package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Server) listBoards(w http.ResponseWriter, r *http.Request) {
	boards, err := s.Store.ListBoards(r.Context(), optionalInt64Param(r, "workspace_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, boards)
}

func (s *Server) createBoard(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		WorkspaceID int64  `json:"workspace_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	board, err := s.Store.CreateBoard(r.Context(), body.Name, body.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, board)
}

func (s *Server) updateBoard(w http.ResponseWriter, r *http.Request) {
	boardID, ok := parseID(r, "boardID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	board, err := s.Store.UpdateBoard(r.Context(), boardID, name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "board not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, board)
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

func (s *Server) updateColumn(w http.ResponseWriter, r *http.Request) {
	boardID, ok := parseID(r, "boardID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}
	columnID, ok := parseID(r, "columnID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid column id")
		return
	}
	var body struct {
		Name   string `json:"name"`
		IsDone bool   `json:"is_done"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	col, err := s.Store.UpdateColumn(r.Context(), boardID, columnID, name, body.IsDone)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "column not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, col)
}

func (s *Server) reorderColumns(w http.ResponseWriter, r *http.Request) {
	boardID, ok := parseID(r, "boardID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	existing, err := s.Store.ListColumnsByBoard(r.Context(), boardID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	known := make(map[int64]bool, len(existing))
	for _, column := range existing {
		known[column.ID] = true
	}
	if len(body.IDs) != len(known) {
		writeError(w, http.StatusBadRequest, "ids must list every column on the board")
		return
	}
	seen := make(map[int64]bool, len(body.IDs))
	for _, id := range body.IDs {
		if !known[id] || seen[id] {
			writeError(w, http.StatusBadRequest, "ids must list every column on the board")
			return
		}
		seen[id] = true
	}
	cols, err := s.Store.ReorderColumns(r.Context(), boardID, body.IDs)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "column not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cols)
}
