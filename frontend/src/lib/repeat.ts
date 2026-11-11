import type { Repeat } from '../api/types'

export const REPEAT_OPTIONS: { value: Repeat['freq']; label: string; unit: string }[] = [
  { value: 'daily', label: 'Daily', unit: 'day' },
  { value: 'weekly', label: 'Weekly', unit: 'week' },
  { value: 'monthly', label: 'Monthly', unit: 'month' },
  { value: 'yearly', label: 'Yearly', unit: 'year' },
]

export function formatRepeat(repeat: Repeat): string {
  const option = REPEAT_OPTIONS.find((item) => item.value === repeat.freq)
  if (!option) return ''
  if (repeat.interval <= 1) return option.label
  return `Every ${repeat.interval} ${option.unit}s`
}

export function repeatBlocker(task: { kind: string; due_date: string | null }): string | null {
  if (task.kind === 'arc') return 'Arcs cannot repeat'
  if (!task.due_date) return 'Set a due date first'
  return null
}
