export interface Workspace {
  id: number
  name: string
  tag: string
  created_at: string
}

export interface WorkspaceTaskTagExport {
  name: string
  color: string
}

export interface WorkspaceTaskExport {
  ref: string
  ref_num: number
  board: string
  column: string
  column_position: number
  column_is_done: boolean
  title: string
  description_md: string
  priority: string
  due_date: string | null
  kind: 'arc' | 'task' | 'step'
  parent_ref: string | null
  position: number
  completed_at: string | null
  archived: boolean
  tags: WorkspaceTaskTagExport[]
}

export interface WorkspaceTasksExport {
  version: number
  workspace: { name: string; tag: string }
  exported_at: string
  tasks: WorkspaceTaskExport[]
}

export interface WorkspaceTasksImportResult {
  imported: number
}

export interface WorkspaceResetResult {
  deleted: number
}

export interface BackupRestoreResult {
  restored: boolean
  counts: Record<string, number>
}

export interface Board {
  id: number
  name: string
  workspace_id: number
  created_at: string
}

export interface BoardColumn {
  id: number
  board_id: number
  name: string
  position: number
  is_done: boolean
  created_at: string
}

export interface Tag {
  id: number
  name: string
  color: string
}

export interface TaskComment {
  id: number
  task_id: number
  author: string
  body_md: string
  external_id: string | null
  created_at: string
}

export interface Note {
  id: number
  title: string
  body_md: string
  task_id: number | null
  workspace_id: number
  created_at: string
  updated_at: string
}

export interface Attachment {
  id: number
  task_id: number | null
  note_id: number | null
  filename: string
  content_type: string
  size_bytes: number
  created_at: string
}

export interface JiraLink {
  task_id: number
  jira_key: string
  jira_url: string
  jira_project_key: string
  link_origin: 'imported' | 'exported' | 'manual'
  last_imported_at: string | null
  last_exported_at: string | null
  raw_status: string | null
  created_at: string
}

export interface JiraSuggestion {
  jira_key: string
  summary: string
  status: string
  score: number
}

export interface JiraExportResult {
  task_id: number
  title: string
  status: 'created' | 'skipped' | 'failed'
  jira_key?: string
  issue_type?: string
  error?: string
}

export interface JiraExportResponse {
  created: number
  skipped: number
  failed: number
  results: JiraExportResult[]
}

export interface MeetingOutcome {
  id: number
  occurred_on: string
  title: string
  notes_md: string
  ends_at: string | null
  workspace_id: number
  outcome_prompt_sent: boolean
  created_at: string
}

export interface ExtractedTask {
  title: string
  suggested_done: boolean
  due_date: string | null
}

export interface Reminder {
  id: number
  title: string
  remind_at: string
  task_id: number | null
  workspace_id: number
  sent: boolean
  created_at: string
}

export interface TimeEntry {
  id: number
  task_id: number
  started_at: string
  ended_at: string | null
  duration_seconds: number | null
}

export interface Task {
  id: number
  board_id: number
  column_id: number
  workspace_id: number
  ref_num: number
  ref: string
  title: string
  description_md: string
  priority: string
  due_date: string | null
  position: number
  parent_task_id: number | null
  kind: 'arc' | 'task' | 'step'
  meeting_outcome_id: number | null
  completed_at: string | null
  archived: boolean
  tags: Tag[]
  created_at: string
  updated_at: string
}

export interface CalendarSource {
  id: number
  name: string
  url: string
  color: string
  workspace_id: number
  created_at: string
}

export interface ImportedEvent {
  id: string
  uid: string
  source_id: number
  title: string
  start: string
  end?: string
  all_day: boolean
  url?: string
  location?: string
}

export interface ImportedCalendar {
  source: CalendarSource
  events: ImportedEvent[]
  error?: string
  last_synced_at?: string
}

export interface TaskRef {
  kind: 'task'
  id: number
  title: string
  ref: string
  ref_num: number
  workspace_tag: string
  workspace_name: string
}

export interface TaskBacklink {
  id: number
  kind: 'task' | 'note' | 'reminder' | 'meeting' | 'chat'
  title: string
  workspace_tag: string
}

export interface LabelSuggestions {
  labels: string[]
}

export interface CustomPalette {
  id: string
  name: string
  colors: string[]
}

export interface AppearanceSettings {
  global_palette_id: string
  workspace_palettes: Record<string, string>
  custom_palettes: CustomPalette[]
  updated_at?: string
}

export interface AISettings {
  provider: 'anthropic' | 'openai' | 'openai-compat'
  base_url: string
  model: string
  has_api_key: boolean
  configured: boolean
  source: 'database' | 'environment'
  updated_at: string | null
}
