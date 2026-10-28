package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/alialjaffer/fai/backend/internal/store"
)

const (
	maxAttachmentBytes      = 25 << 20
	maxAttachmentsTotalSize = 10 << 30
)

func (s *Server) listTaskAttachments(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	attachments, err := s.Store.ListAttachmentsByTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, attachments)
}

func (s *Server) listNoteAttachments(w http.ResponseWriter, r *http.Request) {
	noteID, ok := parseID(r, "noteID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid note id")
		return
	}
	attachments, err := s.Store.ListAttachmentsByNote(r.Context(), noteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, attachments)
}

func (s *Server) uploadTaskAttachment(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseID(r, "taskID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	if _, err := s.Store.GetTask(r.Context(), taskID); err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	s.storeUpload(w, r, &taskID, nil)
}

func (s *Server) uploadNoteAttachment(w http.ResponseWriter, r *http.Request) {
	noteID, ok := parseID(r, "noteID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid note id")
		return
	}
	if _, err := s.Store.GetNote(r.Context(), noteID); err != nil {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}
	s.storeUpload(w, r, nil, &noteID)
}

func (s *Server) storeUpload(w http.ResponseWriter, r *http.Request, taskID, noteID *int64) {
	if s.AttachmentsDir == "" {
		writeError(w, http.StatusServiceUnavailable, "attachments are not configured")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentBytes+(1<<20))
	if err := r.ParseMultipartForm(maxAttachmentBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid upload (file may exceed 25 MiB)")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	if header.Size <= 0 {
		writeError(w, http.StatusBadRequest, "file is empty")
		return
	}
	if header.Size > maxAttachmentBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file is larger than 25 MiB")
		return
	}

	key, err := randomStorageKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.MkdirAll(s.AttachmentsDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	used, err := attachmentsDirSize(s.AttachmentsDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if used+header.Size > maxAttachmentsTotalSize {
		writeError(w, http.StatusInsufficientStorage, "attachment storage is full")
		return
	}
	path := filepath.Join(s.AttachmentsDir, key)
	dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	n, copyErr := io.Copy(dst, io.LimitReader(file, maxAttachmentBytes+1))
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil || n > maxAttachmentBytes {
		os.Remove(path)
		writeError(w, http.StatusBadRequest, "could not store upload")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	attachment, err := s.Store.CreateAttachment(r.Context(), store.CreateAttachmentParams{
		TaskID: taskID, NoteID: noteID,
		Filename: header.Filename, ContentType: contentType, SizeBytes: n, StorageKey: key,
	})
	if err != nil {
		os.Remove(path)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, attachment)
}

func (s *Server) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "attachmentID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid attachment id")
		return
	}
	attachment, err := s.Store.GetAttachment(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	path, err := s.attachmentPath(attachment.StorageKey)
	if err != nil {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "attachment file is missing")
		return
	}
	defer f.Close()

	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Filename})
	if disposition == "" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, attachment.Filename, attachment.CreatedAt, f)
}

func (s *Server) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "attachmentID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid attachment id")
		return
	}
	attachment, err := s.Store.GetAttachment(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	if err := s.Store.DeleteAttachment(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if path, err := s.attachmentPath(attachment.StorageKey); err == nil {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("attachments: removing %s: %v", attachment.StorageKey, err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) attachmentPath(key string) (string, error) {
	// Storage keys are generated hex names; refuse anything that could escape
	// the attachment directory if the database were tampered with.
	if key == "" || filepath.Base(key) != key {
		return "", errors.New("invalid storage key")
	}
	if s.AttachmentsDir == "" {
		return "", errors.New("attachments are not configured")
	}
	return filepath.Join(s.AttachmentsDir, key), nil
}

func randomStorageKey() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func attachmentsDirSize(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}
