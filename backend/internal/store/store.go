package store

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned by operations where "no such row" is a business-logic
// condition the caller must distinguish from a real failure (e.g. stopping a timer
// that isn't running), as opposed to the simple Get* lookups where httpapi already
// treats any error as 404 uniformly.
var ErrNotFound = errors.New("not found")

// ErrDuplicateTag marks a workspace create or rename whose tag is already taken.
var ErrDuplicateTag = errors.New("duplicate workspace tag")

type CreateTaskParams struct {
	BoardID          int64
	ColumnID         int64
	Title            string
	DescriptionMd    string
	Priority         string
	DueDate          *string
	Position         float64
	ParentTaskID     *int64
	Kind             string
	MeetingOutcomeID *int64
	CreatedByFai     bool
	// RefNum, when set, reuses an imported task number instead of allocating the
	// next one. The workspace counter is raised to at least this value.
	RefNum *int64
}

type ListTasksParams struct {
	BoardID  *int64
	ColumnID *int64
	Tag      *string
	Q        *string
}

// Store is the engine-agnostic persistence interface. internal/store/pg implements
// it against Postgres (the kubernetes deployment); internal/store/lite implements it
// against SQLite (the docker-compose deployment). httpapi depends only on this
// interface, never on a concrete driver, so the two deployment targets share one
// set of handlers.
type Store interface {
	// Workspaces
	ListWorkspaces(ctx context.Context) ([]Workspace, error)
	GetWorkspace(ctx context.Context, id int64) (Workspace, error)
	CreateWorkspace(ctx context.Context, name, tag string) (Workspace, error)
	UpdateWorkspace(ctx context.Context, id int64, name, tag *string) (Workspace, error)
	// ResetWorkspaceTasks deletes every task in the workspace (and everything
	// that cascades from tasks) and resets ref_seq. It also returns the storage
	// keys of task attachments so the caller can remove the files.
	ResetWorkspaceTasks(ctx context.Context, workspaceID int64) (int64, []string, error)

	// Boards
	ListBoards(ctx context.Context, workspaceID *int64) ([]Board, error)
	CreateBoard(ctx context.Context, name string, workspaceID int64) (Board, error)
	UpdateBoard(ctx context.Context, id int64, name string) (Board, error)
	DeleteBoard(ctx context.Context, id int64) error
	ListColumnsByBoard(ctx context.Context, boardID int64) ([]BoardColumn, error)
	CreateColumn(ctx context.Context, boardID int64, name string, position int32, isDone bool) (BoardColumn, error)
	UpdateColumn(ctx context.Context, boardID, id int64, name string, isDone bool) (BoardColumn, error)
	ReorderColumns(ctx context.Context, boardID int64, ids []int64) ([]BoardColumn, error)

	// Tags
	ListTags(ctx context.Context) ([]Tag, error)
	CreateTag(ctx context.Context, name, color string) (Tag, error)
	AddTaskTag(ctx context.Context, taskID, tagID int64) error
	RemoveTaskTag(ctx context.Context, taskID, tagID int64) error
	ListTagsForTask(ctx context.Context, taskID int64) ([]Tag, error)

	// Tasks
	CreateTask(ctx context.Context, p CreateTaskParams) (Task, error)
	GetTask(ctx context.Context, id int64) (Task, error)
	ListTasks(ctx context.Context, p ListTasksParams) ([]Task, error)
	ListSubtasks(ctx context.Context, parentTaskID int64) ([]Task, error)
	UpdateTask(ctx context.Context, id int64, title, descriptionMd, priority string, dueDate *string) (Task, error)
	PatchTask(ctx context.Context, id int64, patch TaskPatch) (Task, error)
	UpdateTaskTitle(ctx context.Context, id int64, title string) error
	SetTaskParent(ctx context.Context, id int64, parentTaskID *int64) error
	SetTaskKind(ctx context.Context, id int64, kind string) error
	SetTaskCompletedAt(ctx context.Context, id int64, at time.Time) error
	SetTaskTimestamps(ctx context.Context, id int64, createdAt, updatedAt time.Time) error
	MoveTask(ctx context.Context, id, columnID int64, position float64) (Task, error)
	ListTasksByColumnOrdered(ctx context.Context, columnID int64) ([]Task, error)
	SetTaskPosition(ctx context.Context, id int64, position float64) error
	ArchiveTask(ctx context.Context, id int64) error
	RestoreTask(ctx context.Context, id int64) error
	DeleteTask(ctx context.Context, id int64) error

	// Task comments
	CreateTaskComment(ctx context.Context, taskID int64, author, bodyMd string, externalID *string) (TaskComment, error)
	ListTaskComments(ctx context.Context, taskID int64) ([]TaskComment, error)
	DeleteTaskComment(ctx context.Context, taskID, commentID int64) error

	// Attachments (metadata only; bytes live on disk)
	CreateAttachment(ctx context.Context, p CreateAttachmentParams) (Attachment, error)
	ListAttachmentsByTask(ctx context.Context, taskID int64) ([]Attachment, error)
	ListAttachmentsByNote(ctx context.Context, noteID int64) ([]Attachment, error)
	GetAttachment(ctx context.Context, id int64) (Attachment, error)
	DeleteAttachment(ctx context.Context, id int64) error

	// Notes
	ListNotes(ctx context.Context, workspaceID *int64) ([]Note, error)
	GetNote(ctx context.Context, id int64) (Note, error)
	GetNoteByTitle(ctx context.Context, title string) (Note, error)
	CreateNote(ctx context.Context, title, bodyMd string, taskID *int64, workspaceID int64) (Note, error)
	UpdateNote(ctx context.Context, id int64, title, bodyMd string, taskID *int64) (Note, error)
	DeleteNote(ctx context.Context, id int64) error
	ListBacklinks(ctx context.Context, noteID int64, title string) ([]Note, error)

	// Chat
	CreateChatMessage(ctx context.Context, workspaceID int64, role, content string) (ChatMessage, error)
	ListChatMessages(ctx context.Context, workspaceID int64, limit int32) ([]ChatMessage, error)

	// Search
	SearchTasks(ctx context.Context, query string, workspaceID *int64) ([]Task, error)
	SearchNotes(ctx context.Context, query string, workspaceID *int64) ([]Note, error)

	// Stats
	CompletionsByDay(ctx context.Context, from, to string) ([]DayCount, error)
	TimeByDay(ctx context.Context, from, to string) ([]DaySeconds, error)

	// Time entries. StartTimer atomically closes whatever entry is currently
	// running (if any) and opens a new one for taskID, matching the "one globally
	// running timer" semantics from the plan (§3) — the stop-then-insert is the
	// actual mechanism, a unique-running-entry constraint is only the race guard.
	StartTimer(ctx context.Context, taskID int64) (TaskTimeEntry, error)
	StopEntryForTask(ctx context.Context, taskID int64) (TaskTimeEntry, error)
	GetRunningEntry(ctx context.Context) (*TaskTimeEntry, error)
	ListEntriesForTask(ctx context.Context, taskID int64) ([]TaskTimeEntry, error)

	// Reminders
	ListReminders(ctx context.Context, workspaceID *int64) ([]Reminder, error)
	CreateReminder(ctx context.Context, title string, remindAt time.Time, taskID *int64, workspaceID int64) (Reminder, error)
	UpdateReminder(ctx context.Context, id int64, title string, remindAt time.Time, taskID *int64) (Reminder, error)
	SetReminderCompleted(ctx context.Context, id int64, completed bool) (Reminder, error)
	DeleteReminder(ctx context.Context, id int64) error
	ListDueUnsentReminders(ctx context.Context) ([]Reminder, error)
	MarkReminderSent(ctx context.Context, id int64) error

	// Jira links
	GetJiraLinkByKey(ctx context.Context, jiraKey string) (*JiraLink, error)
	GetJiraLinkByTask(ctx context.Context, taskID int64) (*JiraLink, error)
	ListJiraLinksByWorkspace(ctx context.Context, workspaceID *int64) ([]JiraLink, error)
	// GetJiraTaskRef resolves the Fai task standing behind a Jira key, so imported
	// descriptions that mention the key can be rewritten into an @TAG-<num> ref.
	GetJiraTaskRef(ctx context.Context, jiraKey string) (*TaskRef, error)
	CreateJiraLinkImported(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey, rawStatus string) (JiraLink, error)
	CreateJiraLinkManual(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey, rawStatus string) (JiraLink, error)
	CreateJiraLinkExported(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey string) (JiraLink, error)
	UpdateJiraLinkImported(ctx context.Context, taskID int64, rawStatus string) error
	DeleteJiraLink(ctx context.Context, taskID int64) error

	// Calendar sources (external ICS subscriptions shown in the web calendar;
	// never included in the Fai ICS export)
	ListCalendarSources(ctx context.Context, workspaceID *int64) ([]CalendarSource, error)
	CreateCalendarSource(ctx context.Context, name, url, color string, workspaceID int64) (CalendarSource, error)
	UpdateCalendarSourceColor(ctx context.Context, id int64, color string) (CalendarSource, error)
	DeleteCalendarSource(ctx context.Context, id int64) error
	GetCalendarFeed(ctx context.Context, sourceID int64) ([]byte, time.Time, bool, error)
	SetCalendarFeed(ctx context.Context, sourceID int64, data []byte, fetchedAt time.Time) error

	// Meeting outcomes
	ListMeetingOutcomes(ctx context.Context, occurredOn *string, workspaceID *int64) ([]MeetingOutcome, error)
	ListPendingMeetingOutcomes(ctx context.Context) ([]MeetingOutcome, error)
	GetMeetingOutcome(ctx context.Context, id int64) (MeetingOutcome, error)
	GetMeetingOutcomeByEvent(ctx context.Context, sourceID int64, eventUID string) (MeetingOutcome, error)
	CreateMeetingOutcome(ctx context.Context, occurredOn, title, notesMd string, endsAt *time.Time, workspaceID int64, event MeetingEvent) (MeetingOutcome, error)
	UpdateMeetingOutcome(ctx context.Context, id int64, title, notesMd string, endsAt *time.Time) (MeetingOutcome, error)
	DeleteMeetingOutcome(ctx context.Context, id int64) error
	ListDueUnpromptedMeetingOutcomes(ctx context.Context) ([]MeetingOutcome, error)
	MarkMeetingOutcomePromptSent(ctx context.Context, id int64) error

	// Task references
	ResolveRefs(ctx context.Context, query string, workspaceID *int64) ([]TaskRef, error)
	ListTaskReferences(ctx context.Context, taskID int64) ([]Backlink, error)

	// AI settings. GetAISettings returns ErrNotFound until the owner saves a
	// provider, so the server can fall back to environment configuration.
	GetAISettings(ctx context.Context) (AISettings, error)
	SaveAISettings(ctx context.Context, settings AISettings) (AISettings, error)

	// Appearance is the shared palette selection for the hosted app.
	GetAppearance(ctx context.Context) (AppearanceSettings, error)
	SaveAppearance(ctx context.Context, settings AppearanceSettings) (AppearanceSettings, error)

	// Backup and restore operate on the whole instance. RestoreBackup replaces
	// every row; it never merges.
	DumpBackup(ctx context.Context) (BackupData, error)
	RestoreBackup(ctx context.Context, data BackupData) error

	Close()
}
