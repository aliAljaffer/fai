import { describe, expect, it } from 'vitest'
import { formatRepeat, repeatBlocker } from './repeat'

describe('formatRepeat', () => {
  it('names single intervals', () => {
    expect(formatRepeat({ freq: 'daily', interval: 1 })).toBe('Daily')
    expect(formatRepeat({ freq: 'weekly', interval: 1 })).toBe('Weekly')
    expect(formatRepeat({ freq: 'monthly', interval: 1 })).toBe('Monthly')
    expect(formatRepeat({ freq: 'yearly', interval: 1 })).toBe('Yearly')
  })

  it('names longer intervals', () => {
    expect(formatRepeat({ freq: 'daily', interval: 2 })).toBe('Every 2 days')
    expect(formatRepeat({ freq: 'weekly', interval: 2 })).toBe('Every 2 weeks')
    expect(formatRepeat({ freq: 'monthly', interval: 3 })).toBe('Every 3 months')
    expect(formatRepeat({ freq: 'yearly', interval: 5 })).toBe('Every 5 years')
  })
})

describe('repeatBlocker', () => {
  it('blocks arcs even with a due date', () => {
    expect(repeatBlocker({ kind: 'arc', due_date: '2026-01-01' })).toBe('Arcs cannot repeat')
  })

  it('blocks tasks without a due date', () => {
    expect(repeatBlocker({ kind: 'task', due_date: null })).toBe('Set a due date first')
    expect(repeatBlocker({ kind: 'arc', due_date: null })).toBe('Arcs cannot repeat')
  })

  it('allows tasks with a due date', () => {
    expect(repeatBlocker({ kind: 'task', due_date: '2026-01-01' })).toBeNull()
    expect(repeatBlocker({ kind: 'step', due_date: '2026-01-01' })).toBeNull()
  })
})
