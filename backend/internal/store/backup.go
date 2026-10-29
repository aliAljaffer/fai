package store

import "time"

// BackupData is the portable, engine-agnostic contents of one Fai instance.
// Attachment bytes are not part of it; the storage keys point at files the
// caller packs separately. Secrets are deliberately absent.
type BackupData struct {
	Workspaces          []BackupWorkspace         `json:"workspaces"`
	WorkspaceTagAliases []BackupWorkspaceTagAlias `json:"workspace_tag_aliases"`
	Boards              []BackupBoard             `json:"boards"`
	Columns             []BackupColumn            `json:"columns"`
	Tags                []BackupTag               `json:"tags"`
	Tasks               []BackupTask              `json:"tasks"`
	TaskTags            []BackupTaskTag           `json:"task_tags"`
	Notes               []BackupNote              `json:"notes"`
	NoteTags            []BackupNoteTag           `json:"note_tags"`
	Reminders           []BackupReminder          `json:"reminders"`
	TimeEntries         []BackupTimeEntry         `json:"time_entries"`
	JiraLinks           []BackupJiraLink          `json:"jira_links"`
	Attachments         []BackupAttachment        `json:"attachments"`
	CalendarSources     []BackupCalendarSource    `json:"calendar_sources"`
	MeetingOutcomes     []BackupMeetingOutcome    `json:"meeting_outcomes"`
	TaskComments        []BackupTaskComment       `json:"task_comments"`
	AISettings          *BackupAISettings         `json:"ai_settings"`
	Appearance          *AppearanceSettings       `json:"appearance"`
}

type BackupWorkspace struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Tag       string    `json:"tag"`
	RefSeq    int64     `json:"ref_seq"`
	CreatedAt time.Time `json:"created_at"`
}

type BackupWorkspaceTagAlias struct {
	WorkspaceID int64     `json:"workspace_id"`
	Tag         string    `json:"tag"`
	CreatedAt   time.Time `json:"created_at"`
}

type BackupBoard struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	WorkspaceID int64     `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type BackupColumn struct {
	ID        int64     `json:"id"`
	BoardID   int64     `json:"board_id"`
	Name      string    `json:"name"`
	Position  int32     `json:"position"`
	IsDone    bool      `json:"is_done"`
	CreatedAt time.Time `json:"created_at"`
}

type BackupTag struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type BackupTask struct {
	ID               int64      `json:"id"`
	BoardID          int64      `json:"board_id"`
	ColumnID         int64      `json:"column_id"`
	WorkspaceID      int64      `json:"workspace_id"`
	RefNum           int64      `json:"ref_num"`
	Title            string     `json:"title"`
	DescriptionMd    string     `json:"description_md"`
	Priority         string     `json:"priority"`
	DueDate          *string    `json:"due_date"`
	Position         float64    `json:"position"`
	ParentTaskID     *int64     `json:"parent_task_id"`
	Kind             string     `json:"kind"`
	MeetingOutcomeID *int64     `json:"meeting_outcome_id"`
	CreatedByFai     bool       `json:"created_by_fai"`
	CompletedAt      *time.Time `json:"completed_at"`
	Archived         bool       `json:"archived"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type BackupTaskTag struct {
	TaskID int64 `json:"task_id"`
	TagID  int64 `json:"tag_id"`
}

type BackupNote struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	BodyMd      string    `json:"body_md"`
	TaskID      *int64    `json:"task_id"`
	WorkspaceID int64     `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BackupNoteTag struct {
	NoteID int64 `json:"note_id"`
	TagID  int64 `json:"tag_id"`
}

type BackupReminder struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	RemindAt    time.Time  `json:"remind_at"`
	TaskID      *int64     `json:"task_id"`
	WorkspaceID int64      `json:"workspace_id"`
	Sent        bool       `json:"sent"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type BackupTimeEntry struct {
	ID        int64      `json:"id"`
	TaskID    int64      `json:"task_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
}

type BackupJiraLink struct {
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

type BackupAttachment struct {
	ID          int64     `json:"id"`
	TaskID      *int64    `json:"task_id"`
	NoteID      *int64    `json:"note_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	StorageKey  string    `json:"storage_key"`
	CreatedAt   time.Time `json:"created_at"`
}

type BackupCalendarSource struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Color       string    `json:"color"`
	WorkspaceID int64     `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type BackupMeetingOutcome struct {
	ID                int64      `json:"id"`
	OccurredOn        string     `json:"occurred_on"`
	Title             string     `json:"title"`
	NotesMd           string     `json:"notes_md"`
	EndsAt            *time.Time `json:"ends_at"`
	WorkspaceID       int64      `json:"workspace_id"`
	OutcomePromptSent bool       `json:"outcome_prompt_sent"`
	CreatedAt         time.Time  `json:"created_at"`
}

type BackupTaskComment struct {
	ID         int64     `json:"id"`
	TaskID     int64     `json:"task_id"`
	Author     string    `json:"author"`
	BodyMd     string    `json:"body_md"`
	ExternalID *string   `json:"external_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type BackupAISettings struct {
	Provider  string    `json:"provider"`
	BaseURL   string    `json:"base_url"`
	Model     string    `json:"model"`
	UpdatedAt time.Time `json:"updated_at"`
}
