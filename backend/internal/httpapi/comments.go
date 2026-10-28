package httpapi

import (
	"net/http"
	"strings"
)

func (s *Server) listTaskComments(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	comments, err := s.Store.ListTaskComments(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, comments)
}

type taskCommentBody struct {
	BodyMd string `json:"body_md"`
}

func (s *Server) createTaskComment(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var body taskCommentBody
	if err := decodeJSON(r, &body); err != nil || strings.TrimSpace(body.BodyMd) == "" {
		writeError(w, http.StatusBadRequest, "body_md is required")
		return
	}
	comment, err := s.Store.CreateTaskComment(r.Context(), taskID, "", body.BodyMd, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}

func (s *Server) deleteTaskComment(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	commentID, ok := parseID(r, "commentID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid comment id")
		return
	}
	if err := s.Store.DeleteTaskComment(r.Context(), taskID, commentID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
