// Package store defines a database-engine-agnostic persistence interface so the
// backend can run against either Postgres (kubernetes deployment) or SQLite
// (docker-compose deployment) behind identical httpapi handlers. Domain types use
// plain Go types rather than pgtype/database-sql wrappers so they JSON-marshal
// identically regardless of which engine produced them (time.Time and *time.Time
// marshal via RFC3339Nano, matching pgtype.Timestamptz's own format; *string for a
// nullable date keeps the exact "YYYY-MM-DD" shape pgtype.Date already produced).
package store

import "time"

type Workspace struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Tag       string    `json:"tag"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskRef is a task resolved from an @TAG-<ref_num> reference or a text query.
type TaskRef struct {
	Kind          string `json:"kind"`
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	Ref           string `json:"ref"`
	RefNum        int64  `json:"ref_num"`
	WorkspaceTag  string `json:"workspace_tag"`
	WorkspaceName string `json:"workspace_name"`
}

// Backlink is an item whose text mentions a task reference.
type Backlink struct {
	ID           int64  `json:"id"`
	Kind         string `json:"kind"`
	Title        string `json:"title"`
	WorkspaceTag string `json:"workspace_tag"`
}

type Board struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	WorkspaceID int64     `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`
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

type ChatMessage struct {
	ID          int64     `json:"id"`
	WorkspaceID int64     `json:"workspace_id"`
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
}

// TaskPatch updates only the fields it names. DueDateSet distinguishes an
// omitted due date from an explicit clear.
type TaskPatch struct {
	Title         *string
	DescriptionMd *string
	Priority      *string
	DueDate       *string
	DueDateSet    bool
}

type Task struct {
	ID               int64      `json:"id"`
	BoardID          int64      `json:"board_id"`
	ColumnID         int64      `json:"column_id"`
	WorkspaceID      int64      `json:"workspace_id"`
	RefNum           int64      `json:"ref_num"`
	Ref              string     `json:"ref"`
	Title            string     `json:"title"`
	DescriptionMd    string     `json:"description_md"`
	Priority         string     `json:"priority"`
	DueDate          *string    `json:"due_date"`
	Position         float64    `json:"position"`
	ParentTaskID     *int64     `json:"parent_task_id"`
	Kind             string     `json:"kind"`
	MeetingOutcomeID *int64     `json:"meeting_outcome_id"`
	CompletedAt      *time.Time `json:"completed_at"`
	Archived         bool       `json:"archived"`
	Tags             []Tag      `json:"tags"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type TaskComment struct {
	ID         int64     `json:"id"`
	TaskID     int64     `json:"task_id"`
	Author     string    `json:"author"`
	BodyMd     string    `json:"body_md"`
	ExternalID *string   `json:"external_id"`
	CreatedAt  time.Time `json:"created_at"`
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
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	BodyMd      string    `json:"body_md"`
	TaskID      *int64    `json:"task_id"`
	WorkspaceID int64     `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	RemindAt    time.Time `json:"remind_at"`
	TaskID      *int64    `json:"task_id"`
	WorkspaceID int64     `json:"workspace_id"`
	Sent        bool      `json:"sent"`
	CreatedAt   time.Time `json:"created_at"`
}

type MeetingOutcome struct {
	ID                int64      `json:"id"`
	OccurredOn        string     `json:"occurred_on"`
	Title             string     `json:"title"`
	NotesMd           string     `json:"notes_md"`
	EndsAt            *time.Time `json:"ends_at"`
	WorkspaceID       int64      `json:"workspace_id"`
	OutcomePromptSent bool       `json:"outcome_prompt_sent"`
	CreatedAt         time.Time  `json:"created_at"`
}

type CalendarSource struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Color       string    `json:"color"`
	WorkspaceID int64     `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type DayCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type DaySeconds struct {
	Date    string `json:"date"`
	Seconds int64  `json:"seconds"`
}

type CustomPalette struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Colors []string `json:"colors"`
}

type AppearanceSettings struct {
	GlobalPaletteID   string            `json:"global_palette_id"`
	WorkspacePalettes map[string]string `json:"workspace_palettes"`
	CustomPalettes    []CustomPalette   `json:"custom_palettes"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

func DefaultAppearance() AppearanceSettings {
	return AppearanceSettings{
		GlobalPaletteID:   "light-steel",
		WorkspacePalettes: map[string]string{},
		CustomPalettes:    []CustomPalette{},
	}
}

// AISettings is the single owner's stored model provider configuration. APIKey
// is never serialized to clients.
type AISettings struct {
	Provider  string    `json:"provider"`
	BaseURL   string    `json:"base_url"`
	Model     string    `json:"model"`
	APIKey    string    `json:"-"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NormalizeKind fills an omitted kind from the parent link, so callers that do
// not care about issue types still create valid rows.
func NormalizeKind(kind string, hasParent bool) string {
	switch kind {
	case "arc", "task", "step":
		return kind
	}
	if hasParent {
		return "step"
	}
	return "task"
}
