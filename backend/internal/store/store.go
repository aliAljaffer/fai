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

type CreateTaskParams struct {
	BoardID          int64
	ColumnID         int64
	Title            string
	DescriptionMd    string
	Priority         string
	DueDate          *string
	Position         float64
	ParentTaskID     *int64
	MeetingOutcomeID *int64
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
	// Boards
	ListBoards(ctx context.Context) ([]Board, error)
	CreateBoard(ctx context.Context, name string) (Board, error)
	ListColumnsByBoard(ctx context.Context, boardID int64) ([]BoardColumn, error)
	CreateColumn(ctx context.Context, boardID int64, name string, position int32, isDone bool) (BoardColumn, error)

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
	UpdateTaskTitle(ctx context.Context, id int64, title string) error
	SetTaskParent(ctx context.Context, id int64, parentTaskID *int64) error
	MoveTask(ctx context.Context, id, columnID int64, position float64) (Task, error)
	ListTasksByColumnOrdered(ctx context.Context, columnID int64) ([]Task, error)
	SetTaskPosition(ctx context.Context, id int64, position float64) error
	ArchiveTask(ctx context.Context, id int64) error
	DeleteTask(ctx context.Context, id int64) error

	// Attachments (metadata only; bytes live on disk)
	CreateAttachment(ctx context.Context, p CreateAttachmentParams) (Attachment, error)
	ListAttachmentsByTask(ctx context.Context, taskID int64) ([]Attachment, error)
	ListAttachmentsByNote(ctx context.Context, noteID int64) ([]Attachment, error)
	GetAttachment(ctx context.Context, id int64) (Attachment, error)
	DeleteAttachment(ctx context.Context, id int64) error

	// Notes
	ListNotes(ctx context.Context) ([]Note, error)
	GetNote(ctx context.Context, id int64) (Note, error)
	GetNoteByTitle(ctx context.Context, title string) (Note, error)
	CreateNote(ctx context.Context, title, bodyMd string, taskID *int64) (Note, error)
	UpdateNote(ctx context.Context, id int64, title, bodyMd string, taskID *int64) (Note, error)
	DeleteNote(ctx context.Context, id int64) error
	ListBacklinks(ctx context.Context, noteID int64, title string) ([]Note, error)

	// Search
	SearchTasks(ctx context.Context, query string) ([]Task, error)
	SearchNotes(ctx context.Context, query string) ([]Note, error)

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
	ListReminders(ctx context.Context) ([]Reminder, error)
	CreateReminder(ctx context.Context, title string, remindAt time.Time, taskID *int64) (Reminder, error)
	UpdateReminder(ctx context.Context, id int64, title string, remindAt time.Time, taskID *int64) (Reminder, error)
	DeleteReminder(ctx context.Context, id int64) error
	ListDueUnsentReminders(ctx context.Context) ([]Reminder, error)
	MarkReminderSent(ctx context.Context, id int64) error

	// Jira links
	GetJiraLinkByKey(ctx context.Context, jiraKey string) (*JiraLink, error)
	GetJiraLinkByTask(ctx context.Context, taskID int64) (*JiraLink, error)
	CreateJiraLinkImported(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey, rawStatus string) (JiraLink, error)
	CreateJiraLinkManual(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey, rawStatus string) (JiraLink, error)
	CreateJiraLinkExported(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey string) (JiraLink, error)
	UpdateJiraLinkImported(ctx context.Context, taskID int64, rawStatus string) error
	DeleteJiraLink(ctx context.Context, taskID int64) error

	// Calendar sources (external ICS subscriptions shown in the web calendar;
	// never included in the Fai ICS export)
	ListCalendarSources(ctx context.Context) ([]CalendarSource, error)
	CreateCalendarSource(ctx context.Context, name, url, color string) (CalendarSource, error)
	UpdateCalendarSourceColor(ctx context.Context, id int64, color string) (CalendarSource, error)
	DeleteCalendarSource(ctx context.Context, id int64) error
	GetCalendarFeed(ctx context.Context, sourceID int64) ([]byte, time.Time, bool, error)
	SetCalendarFeed(ctx context.Context, sourceID int64, data []byte, fetchedAt time.Time) error

	// Meeting outcomes
	ListMeetingOutcomes(ctx context.Context, occurredOn *string) ([]MeetingOutcome, error)
	ListPendingMeetingOutcomes(ctx context.Context) ([]MeetingOutcome, error)
	GetMeetingOutcome(ctx context.Context, id int64) (MeetingOutcome, error)
	CreateMeetingOutcome(ctx context.Context, occurredOn, title, notesMd string, endsAt *time.Time) (MeetingOutcome, error)
	UpdateMeetingOutcome(ctx context.Context, id int64, title, notesMd string, endsAt *time.Time) (MeetingOutcome, error)
	DeleteMeetingOutcome(ctx context.Context, id int64) error
	ListDueUnpromptedMeetingOutcomes(ctx context.Context) ([]MeetingOutcome, error)
	MarkMeetingOutcomePromptSent(ctx context.Context, id int64) error

	Close()
}
