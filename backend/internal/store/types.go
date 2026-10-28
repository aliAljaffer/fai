// Package store defines a database-engine-agnostic persistence interface so the
// backend can run against either Postgres (kubernetes deployment) or SQLite
// (docker-compose deployment) behind identical httpapi handlers. Domain types use
// plain Go types rather than pgtype/database-sql wrappers so they JSON-marshal
// identically regardless of which engine produced them (time.Time and *time.Time
// marshal via RFC3339Nano, matching pgtype.Timestamptz's own format; *string for a
// nullable date keeps the exact "YYYY-MM-DD" shape pgtype.Date already produced).
package store

import "time"

type Board struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type BoardColumn struct {
	ID        int64     `json:"id"`
	BoardID   int64     `json:"board_id"`
	Name      string    `json:"name"`
	Position  int32     `json:"position"`
	IsDone    bool      `json:"is_done"`
	CreatedAt time.Time `json:"created_at"`
}

type Tag struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Task struct {
	ID               int64      `json:"id"`
	BoardID          int64      `json:"board_id"`
	ColumnID         int64      `json:"column_id"`
	Title            string     `json:"title"`
	DescriptionMd    string     `json:"description_md"`
	Priority         string     `json:"priority"`
	DueDate          *string    `json:"due_date"`
	Position         float64    `json:"position"`
	ParentTaskID     *int64     `json:"parent_task_id"`
	MeetingOutcomeID *int64     `json:"meeting_outcome_id"`
	CompletedAt      *time.Time `json:"completed_at"`
	Archived         bool       `json:"archived"`
	Tags             []Tag      `json:"tags"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type Attachment struct {
	ID          int64     `json:"id"`
	TaskID      *int64    `json:"task_id"`
	NoteID      *int64    `json:"note_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	StorageKey  string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateAttachmentParams struct {
	TaskID      *int64
	NoteID      *int64
	Filename    string
	ContentType string
	SizeBytes   int64
	StorageKey  string
}

type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	BodyMd    string    `json:"body_md"`
	TaskID    *int64    `json:"task_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TaskTimeEntry struct {
	ID              int64      `json:"id"`
	TaskID          int64      `json:"task_id"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
	DurationSeconds *int32     `json:"duration_seconds"`
}

type JiraLink struct {
	TaskID         int64      `json:"task_id"`
	JiraKey        string     `json:"jira_key"`
	JiraURL        string     `json:"jira_url"`
	JiraProjectKey string     `json:"jira_project_key"`
	LinkOrigin     string     `json:"link_origin"`
	LastImportedAt *time.Time `json:"last_imported_at"`
	LastExportedAt *time.Time `json:"last_exported_at"`
	RawStatus      *string    `json:"raw_status"`
	CreatedAt      time.Time  `json:"created_at"`
}

type Reminder struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	RemindAt  time.Time `json:"remind_at"`
	TaskID    *int64    `json:"task_id"`
	Sent      bool      `json:"sent"`
	CreatedAt time.Time `json:"created_at"`
}

type MeetingOutcome struct {
	ID                int64      `json:"id"`
	OccurredOn        string     `json:"occurred_on"`
	Title             string     `json:"title"`
	NotesMd           string     `json:"notes_md"`
	EndsAt            *time.Time `json:"ends_at"`
	OutcomePromptSent bool       `json:"outcome_prompt_sent"`
	CreatedAt         time.Time  `json:"created_at"`
}

type CalendarSource struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

type DayCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type DaySeconds struct {
	Date    string `json:"date"`
	Seconds int64  `json:"seconds"`
}
