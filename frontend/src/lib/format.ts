import { format, parseISO } from 'date-fns'

export function formatDuration(totalSeconds: number): string {
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  const s = Math.floor(totalSeconds % 60)
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

export function formatDueDate(value: string | null): string | null {
  if (!value) return null
  const date = parseISO(value)
  return Number.isNaN(date.getTime()) ? value : format(date, 'MMM d')
}

export function isPastDue(value: string | null, completed = false): boolean {
  if (!value || completed) return false
  const date = parseISO(value)
  if (Number.isNaN(date.getTime())) return false
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return date < today
}

export function formatDueIn(value: string, now = new Date()): string {
  const due = parseISO(value)
  if (Number.isNaN(due.getTime())) return ''
  const minutes = Math.round((due.getTime() - now.getTime()) / 60000)
  if (minutes <= -1) {
    const past = -minutes
    if (past < 60) return `overdue ${past}m`
    if (past < 1440) return `overdue ${Math.floor(past / 60)}h`
    return `overdue ${Math.floor(past / 1440)}d`
  }
  if (minutes === 0) return 'due now'
  if (minutes < 60) return `due in ${minutes}m`
  if (minutes < 1440) return `due in ${Math.floor(minutes / 60)}h`
  return `due in ${Math.floor(minutes / 1440)}d`
}

export function formatTimestamp(value: string): string {
  const date = parseISO(value)
  return Number.isNaN(date.getTime()) ? value : format(date, 'MMM d, HH:mm')
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB']
  let value = bytes / 1024
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit++
  }
  return `${value.toFixed(value < 10 ? 1 : 0)} ${units[unit]}`
}
