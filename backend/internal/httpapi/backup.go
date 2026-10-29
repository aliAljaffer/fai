package httpapi

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

const (
	backupFormat  = "fai-backup"
	backupVersion = 1
	maxBackupSize = maxAttachmentsTotalSize + (1 << 30)
)

type backupManifest struct {
	Format           string                  `json:"format"`
	Version          int                     `json:"version"`
	CreatedAt        time.Time               `json:"created_at"`
	DataSHA256       string                  `json:"data_sha256"`
	Counts           map[string]int          `json:"counts"`
	Attachments      []backupAttachmentEntry `json:"attachments"`
	SecretsExcluded  bool                    `json:"secrets_excluded"`
	SecretsToReenter []string                `json:"secrets_to_reenter"`
}

type backupAttachmentEntry struct {
	StorageKey string `json:"storage_key"`
	SizeBytes  int64  `json:"size_bytes"`
	SHA256     string `json:"sha256"`
}

func backupCounts(data store.BackupData) map[string]int {
	return map[string]int{
		"workspaces":            len(data.Workspaces),
		"workspace_tag_aliases": len(data.WorkspaceTagAliases),
		"boards":                len(data.Boards),
		"columns":               len(data.Columns),
		"tasks":                 len(data.Tasks),
		"tags":                  len(data.Tags),
		"task_tags":             len(data.TaskTags),
		"notes":                 len(data.Notes),
		"note_tags":             len(data.NoteTags),
		"reminders":             len(data.Reminders),
		"time_entries":          len(data.TimeEntries),
		"jira_links":            len(data.JiraLinks),
		"attachments":           len(data.Attachments),
		"calendar_sources":      len(data.CalendarSources),
		"meeting_outcomes":      len(data.MeetingOutcomes),
		"task_comments":         len(data.TaskComments),
	}
}

func (s *Server) exportBackup(w http.ResponseWriter, r *http.Request) {
	data, err := s.Store.DumpBackup(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(data.Attachments) > 0 && s.AttachmentsDir == "" {
		writeError(w, http.StatusBadRequest, "attachments are not configured")
		return
	}

	tmp, err := os.CreateTemp("", "fai-backup-*.zip")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	zw := zip.NewWriter(tmp)
	dataBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	header := &zip.FileHeader{Name: "data.json", Method: zip.Deflate}
	header.SetModTime(time.Now())
	dataFile, err := zw.CreateHeader(header)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := dataFile.Write(dataBytes); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	seen := map[string]bool{}
	entries := make([]backupAttachmentEntry, 0, len(data.Attachments))
	for _, attachment := range data.Attachments {
		if attachment.StorageKey == "" || seen[attachment.StorageKey] {
			writeError(w, http.StatusInternalServerError, "duplicate or empty attachment storage key")
			return
		}
		seen[attachment.StorageKey] = true
		path, err := s.attachmentPath(attachment.StorageKey)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		source, err := os.Open(path)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("attachment %s: %v", attachment.StorageKey, err))
			return
		}
		header := &zip.FileHeader{Name: "attachments/" + attachment.StorageKey, Method: zip.Store}
		header.SetModTime(time.Now())
		target, err := zw.CreateHeader(header)
		if err != nil {
			source.Close()
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		hasher := sha256.New()
		written, err := io.Copy(io.MultiWriter(target, hasher), source)
		source.Close()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		entries = append(entries, backupAttachmentEntry{
			StorageKey: attachment.StorageKey,
			SizeBytes:  written,
			SHA256:     hex.EncodeToString(hasher.Sum(nil)),
		})
	}

	dataSum := sha256.Sum256(dataBytes)
	manifest := backupManifest{
		Format:          backupFormat,
		Version:         backupVersion,
		CreatedAt:       time.Now().UTC(),
		DataSHA256:      hex.EncodeToString(dataSum[:]),
		Counts:          backupCounts(data),
		Attachments:     entries,
		SecretsExcluded: true,
		SecretsToReenter: []string{
			"AI API key (Settings or AI_API_KEY)",
			"JIRA_EMAIL and JIRA_API_TOKEN",
			"SMTP_ADDR, SMTP_FROM, and SMTP_TO",
			"NTFY_URL and NTFY_TOPIC",
			"ICS_TOKEN",
		},
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	header = &zip.FileHeader{Name: "manifest.json", Method: zip.Deflate}
	header.SetModTime(time.Now())
	manifestFile, err := zw.CreateHeader(header)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := manifestFile.Write(manifestBytes); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := zw.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("fai-backup-%s.zip", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.ServeContent(w, r, filename, time.Now(), tmp)
}

func (s *Server) restoreBackup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBackupSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid backup upload")
		return
	}
	defer r.MultipartForm.RemoveAll()

	if strings.TrimSpace(r.FormValue("confirm")) != "I understand" {
		writeError(w, http.StatusBadRequest, `type "I understand" to confirm`)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "backup file is required")
		return
	}
	defer file.Close()

	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read backup file")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusBadRequest, "could not read backup file")
		return
	}
	zr, err := zip.NewReader(file, size)
	if err != nil {
		writeError(w, http.StatusBadRequest, "not a valid zip backup")
		return
	}

	files := map[string]*zip.File{}
	for _, entry := range zr.File {
		if _, exists := files[entry.Name]; exists {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("duplicate zip entry %s", entry.Name))
			return
		}
		files[entry.Name] = entry
	}

	manifestBytes, err := readZipEntry(files, "manifest.json", 16<<20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var manifest backupManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		writeError(w, http.StatusBadRequest, "invalid backup manifest")
		return
	}
	if manifest.Format != backupFormat {
		writeError(w, http.StatusBadRequest, "not a Fai backup")
		return
	}
	if manifest.Version > backupVersion {
		writeError(w, http.StatusBadRequest, "backup was made by a newer Fai version")
		return
	}

	dataBytes, err := readZipEntry(files, "data.json", 512<<20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sum := sha256.Sum256(dataBytes)
	if manifest.DataSHA256 != "" && manifest.DataSHA256 != hex.EncodeToString(sum[:]) {
		writeError(w, http.StatusBadRequest, "backup data checksum does not match")
		return
	}
	var data store.BackupData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		writeError(w, http.StatusBadRequest, "invalid backup data")
		return
	}
	if !reflectCountsMatch(manifest.Counts, data) {
		writeError(w, http.StatusBadRequest, "backup counts do not match its data")
		return
	}
	if len(data.Attachments) > 0 && s.AttachmentsDir == "" {
		writeError(w, http.StatusBadRequest, "attachments are not configured on this instance")
		return
	}

	stage, err := os.MkdirTemp("", "fai-backup-attachments-*")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.RemoveAll(stage)

	manifestEntries := map[string]backupAttachmentEntry{}
	for _, entry := range manifest.Attachments {
		manifestEntries[entry.StorageKey] = entry
	}
	seen := map[string]bool{}
	var stagedTotal int64
	for _, attachment := range data.Attachments {
		key := attachment.StorageKey
		if !validBackupStorageKey(key) || seen[key] {
			writeError(w, http.StatusBadRequest, "invalid attachment storage key in backup")
			return
		}
		seen[key] = true
		entry, ok := manifestEntries[key]
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("attachment %s is missing from the manifest", key))
			return
		}
		zipFile, ok := files["attachments/"+key]
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("attachment %s is missing from the backup", key))
			return
		}
		if entry.SizeBytes != attachment.SizeBytes || entry.SizeBytes > int64(maxAttachmentsTotalSize)-stagedTotal {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("attachment %s has an invalid size", key))
			return
		}
		target, err := os.OpenFile(filepath.Join(stage, key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		reader, err := zipFile.Open()
		if err != nil {
			target.Close()
			writeError(w, http.StatusBadRequest, fmt.Sprintf("attachment %s is unreadable", key))
			return
		}
		hasher := sha256.New()
		written, copyErr := io.Copy(io.MultiWriter(target, hasher), io.LimitReader(reader, entry.SizeBytes+1))
		reader.Close()
		closeErr := target.Close()
		if copyErr != nil || closeErr != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("attachment %s is unreadable", key))
			return
		}
		if written != entry.SizeBytes || hex.EncodeToString(hasher.Sum(nil)) != entry.SHA256 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("attachment %s checksum does not match", key))
			return
		}
		stagedTotal += written
	}

	if err := s.Store.RestoreBackup(r.Context(), data); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.replaceAttachments(stage); err != nil {
		writeError(w, http.StatusInternalServerError, "database restored, but attachments could not be replaced: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"restored": true,
		"counts":   manifest.Counts,
	})
}

func validBackupStorageKey(key string) bool {
	if key == "" || len(key) > 128 {
		return false
	}
	for _, r := range key {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func readZipEntry(files map[string]*zip.File, name string, limit int64) ([]byte, error) {
	entry, ok := files[name]
	if !ok {
		return nil, fmt.Errorf("backup is missing %s", name)
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, fmt.Errorf("could not read %s", name)
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("could not read %s", name)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s is too large", name)
	}
	return data, nil
}

func reflectCountsMatch(counts map[string]int, data store.BackupData) bool {
	if len(counts) == 0 {
		return true
	}
	for key, value := range backupCounts(data) {
		if counts[key] != value {
			return false
		}
	}
	return true
}

func (s *Server) replaceAttachments(stage string) error {
	if s.AttachmentsDir == "" {
		return nil
	}
	if err := os.MkdirAll(s.AttachmentsDir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(s.AttachmentsDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(s.AttachmentsDir, entry.Name())); err != nil {
			return err
		}
	}
	staged, err := os.ReadDir(stage)
	if err != nil {
		return err
	}
	for _, entry := range staged {
		if entry.IsDir() {
			continue
		}
		source, err := os.Open(filepath.Join(stage, entry.Name()))
		if err != nil {
			return err
		}
		target, err := os.OpenFile(filepath.Join(s.AttachmentsDir, entry.Name()), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			source.Close()
			return err
		}
		_, copyErr := io.Copy(target, source)
		source.Close()
		closeErr := target.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
