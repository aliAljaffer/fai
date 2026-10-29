import type { BoardColumn, Task } from '../api/types'

export type TaskKind = 'arc' | 'task' | 'step'
export type TaskStatus = 'todo' | 'doing' | 'done'

export const TASK_KIND_LABELS: Record<TaskKind, string> = {
  arc: 'Arc',
  task: 'Task',
  step: 'Step',
}

export const TASK_KIND_NAMES: Record<TaskKind, string> = {
  arc: 'Arcs',
  task: 'Tasks',
  step: 'Steps',
}

export function normalizeKind(kind?: string | null): TaskKind {
  return kind === 'arc' || kind === 'step' ? kind : 'task'
}

export function taskStatus(task: Task, columns?: BoardColumn[]): TaskStatus {
  const column = columns?.find((item) => item.id === task.column_id)
  if (task.completed_at || column?.is_done) return 'done'
  const open = (columns ?? []).filter((item) => !item.is_done)
  if (open.length > 1 && open.findIndex((item) => item.id === task.column_id) > 0) {
    return 'doing'
  }
  return 'todo'
}

export function statusColor(status: TaskStatus) {
  return `var(--status-${status})`
}

export const STATUS_TITLE: Record<TaskStatus, string> = {
  todo: 'To do',
  doing: 'In progress',
  done: 'Done',
}
