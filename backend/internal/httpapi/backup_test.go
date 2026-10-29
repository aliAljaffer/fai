package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func newBackupServer(t *testing.T) (*lite.Store, *httptest.Server, string) {
	t.Helper()
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	dir := t.TempDir()
	server := httptest.NewServer(NewRouter(&Server{Store: st, AttachmentsDir: dir}))
	t.Cleanup(server.Close)
	return st, server, dir
}

func TestFullBackupRestore(t *testing.T) {
	ctx := context.Background()
	source, sourceServer, sourceAttachments := newBackupServer(t)

	workspace, err := source.GetWorkspace(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	boards, err := source.ListBoards(ctx, &workspace.ID)
	if err != nil || len(boards) == 0 {
		t.Fatalf("boards = %#v, err = %v", boards, err)
	}
	columns, err := source.ListColumnsByBoard(ctx, boards[0].ID)
	if err != nil || len(columns) == 0 {
		t.Fatalf("columns = %#v, err = %v", columns, err)
	}
	todo := columns[0]
	done := columns[0]
	for _, column := range columns {
		if column.IsDone {
			done = column
		}
	}

	parent, err := source.CreateTask(ctx, store.CreateTaskParams{
		BoardID: boards[0].ID, ColumnID: todo.ID, Title: "Parent", DescriptionMd: "body",
		Priority: "high", Position: 1024, Kind: "task",
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := source.CreateTask(ctx, store.CreateTaskParams{
		BoardID: boards[0].ID, ColumnID: done.ID, Title: "Child", DescriptionMd: "child",
		Priority: "medium", Position: 1024, ParentTaskID: &parent.ID, Kind: "step",
	})
	if err != nil {
		t.Fatal(err)
	}
	tag, err := source.CreateTag(ctx, "alpha", "#123456")
	if err != nil {
		t.Fatal(err)
	}
	if err := source.AddTaskTag(ctx, parent.ID, tag.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.CreateNote(ctx, "Backup note", "mentions @FAI-1", &parent.ID, workspace.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.CreateReminder(ctx, "Backup reminder", time.Now().Add(time.Hour), &parent.ID, workspace.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.StartTimer(ctx, parent.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.StopEntryForTask(ctx, parent.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.CreateMeetingOutcome(ctx, "2026-10-01", "Backup meeting", "notes", nil, workspace.ID, store.MeetingEvent{}); err != nil {
		t.Fatal(err)
	}
	if _, err := source.CreateCalendarSource(ctx, "Backup calendar", "https://example.com/cal.ics", "#2563eb", workspace.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.CreateTaskComment(ctx, parent.ID, "owner", "backup comment", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := source.CreateJiraLinkManual(ctx, parent.ID, "ABC-1", "https://jira.example.com/ABC-1", "ABC", "In Progress"); err != nil {
		t.Fatal(err)
	}

	storageKey := "0123456789abcdef0123456789abcdef"
	attachmentBody := []byte("attachment contents")
	if err := os.WriteFile(filepath.Join(sourceAttachments, storageKey), attachmentBody, 0o600); err != nil {
		t.Fatal(err)
	}
	attachment, err := source.CreateAttachment(ctx, store.CreateAttachmentParams{
		TaskID: &parent.ID, Filename: "note.txt", ContentType: "text/plain",
		SizeBytes: int64(len(attachmentBody)), StorageKey: storageKey,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := source.SaveAISettings(ctx, store.AISettings{
		Provider: "anthropic", BaseURL: "https://api.anthropic.com", Model: "claude-test", APIKey: "super-secret-key",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := source.SaveAppearance(ctx, store.AppearanceSettings{
		GlobalPaletteID: "custom",
		CustomPalettes: []store.CustomPalette{{
			ID: "custom", Name: "Custom", Colors: []string{"#111111", "#222222", "#333333", "#444444", "#555555"},
		}},
		WorkspacePalettes: map[string]string{"1": "custom"},
	}); err != nil {
		t.Fatal(err)
	}

	res := httpGet(t, sourceServer.URL+"/api/backup")
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("backup status = %d: %s", res.StatusCode, body)
	}
	zipBytes, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string][]byte{}
	for _, entry := range zr.File {
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[entry.Name] = body
	}
	var manifest backupManifest
	if err := json.Unmarshal(entries["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Format != backupFormat || manifest.Version != backupVersion || !manifest.SecretsExcluded {
		t.Errorf("manifest = %#v", manifest)
	}
	if strings.Contains(string(entries["data.json"]), "super-secret-key") {
		t.Error("backup contains the AI API key")
	}
	if _, ok := entries["attachments/"+storageKey]; !ok {
		t.Error("backup has no attachment file")
	}

	target, targetServer, targetAttachments := newBackupServer(t)
	if err := os.WriteFile(filepath.Join(targetAttachments, "old-file"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := target.CreateTask(ctx, store.CreateTaskParams{
		BoardID: 1, ColumnID: 1, Title: "Target only", Priority: "medium", Position: 1024, Kind: "task",
	}); err != nil {
		t.Fatal(err)
	}

	rejected := postBackup(t, targetServer.URL+"/api/backup/restore", zipBytes, "no")
	if rejected.StatusCode != http.StatusBadRequest {
		t.Fatalf("rejected restore status = %d", rejected.StatusCode)
	}
	rejected.Body.Close()
	if tasks, err := target.ListTasks(ctx, store.ListTasksParams{}); err != nil || len(tasks) != 1 {
		t.Fatalf("target changed after rejected restore: %#v, %v", tasks, err)
	}

	restored := postBackup(t, targetServer.URL+"/api/backup/restore", zipBytes, "I understand")
	if restored.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(restored.Body)
		t.Fatalf("restore status = %d: %s", restored.StatusCode, body)
	}
	restored.Body.Close()

	tasks, err := target.ListTasks(ctx, store.ListTasksParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("restored tasks = %d", len(tasks))
	}
	byRef := map[string]store.Task{}
	for _, task := range tasks {
		byRef[task.Ref] = task
	}
	restoredParent := byRef["FAI-1"]
	restoredChild := byRef["FAI-2"]
	if restoredParent.ID != parent.ID || restoredChild.ID != child.ID {
		t.Errorf("task ids = %d, %d, want %d, %d", restoredParent.ID, restoredChild.ID, parent.ID, child.ID)
	}
	if restoredChild.ParentTaskID == nil || *restoredChild.ParentTaskID != restoredParent.ID {
		t.Errorf("restored parent = %v", restoredChild.ParentTaskID)
	}
	if len(restoredParent.Tags) != 1 || restoredParent.Tags[0].Name != "alpha" {
		t.Errorf("restored tags = %#v", restoredParent.Tags)
	}

	notes, err := target.ListNotes(ctx, nil)
	if err != nil || len(notes) != 1 || notes[0].Title != "Backup note" {
		t.Errorf("notes = %#v, err = %v", notes, err)
	}
	reminders, err := target.ListReminders(ctx, nil)
	if err != nil || len(reminders) != 1 || reminders[0].Title != "Backup reminder" {
		t.Errorf("reminders = %#v, err = %v", reminders, err)
	}
	timeEntries, err := target.ListEntriesForTask(ctx, parent.ID)
	if err != nil || len(timeEntries) != 1 {
		t.Errorf("time entries = %#v, err = %v", timeEntries, err)
	}
	outcomes, err := target.ListMeetingOutcomes(ctx, nil, nil)
	if err != nil || len(outcomes) != 1 || outcomes[0].Title != "Backup meeting" {
		t.Errorf("outcomes = %#v, err = %v", outcomes, err)
	}
	sources, err := target.ListCalendarSources(ctx, nil)
	if err != nil || len(sources) != 1 || sources[0].Name != "Backup calendar" {
		t.Errorf("calendar sources = %#v, err = %v", sources, err)
	}
	comments, err := target.ListTaskComments(ctx, parent.ID)
	if err != nil || len(comments) != 1 || comments[0].BodyMd != "backup comment" {
		t.Errorf("comments = %#v, err = %v", comments, err)
	}
	link, err := target.GetJiraLinkByTask(ctx, parent.ID)
	if err != nil || link == nil || link.JiraKey != "ABC-1" {
		t.Errorf("jira link = %#v, err = %v", link, err)
	}

	ai, err := target.GetAISettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ai.Provider != "anthropic" || ai.Model != "claude-test" || ai.APIKey != "" {
		t.Errorf("ai settings = %#v", ai)
	}
	appearance, err := target.GetAppearance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if appearance.GlobalPaletteID != "custom" || appearance.WorkspacePalettes["1"] != "custom" {
		t.Errorf("appearance = %#v", appearance)
	}

	restoredAttachment, err := target.GetAttachment(ctx, attachment.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(targetAttachments, restoredAttachment.StorageKey))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(attachmentBody) {
		t.Errorf("attachment body = %q", body)
	}
	if _, err := os.Stat(filepath.Join(targetAttachments, "old-file")); !os.IsNotExist(err) {
		t.Error("old attachment file survived restore")
	}

	next, err := target.CreateTask(ctx, store.CreateTaskParams{
		BoardID: boards[0].ID, ColumnID: todo.ID, Title: "After restore", Priority: "medium", Position: 1024, Kind: "task",
	})
	if err != nil {
		t.Fatal(err)
	}
	if next.Ref != "FAI-3" {
		t.Errorf("next ref after restore = %q", next.Ref)
	}
	if next.ID != 3 {
		t.Errorf("next task id after restore = %d", next.ID)
	}
}

func postBackup(t *testing.T, url string, data []byte, confirm string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	file, err := writer.CreateFormFile("file", "backup.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("confirm", confirm); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}
