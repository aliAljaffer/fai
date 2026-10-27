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
