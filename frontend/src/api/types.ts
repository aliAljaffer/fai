export interface Board {
  id: number
  name: string
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

export interface Note {
  id: number
  title: string
  body_md: string
  task_id: number | null
  created_at: string
  updated_at: string
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

export interface Reminder {
  id: number
  title: string
  remind_at: string
  task_id: number | null
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
  title: string
  description_md: string
  priority: string
  due_date: string | null
  position: number
  completed_at: string | null
  archived: boolean
  created_at: string
  updated_at: string
}
