import { describe, expect, it } from 'vitest'
import { planDrop, tasksInColumn } from './boardDrag'
import type { Task } from '../api/types'

function task(id: number, columnId: number, position: number): Task {
  return {
    id,
    board_id: 1,
    column_id: columnId,
    workspace_id: 1,
    ref_num: id,
    ref: `FAI-${id}`,
    title: `Task ${id}`,
    description_md: '',
    priority: 'medium',
    due_date: null,
    position,
    parent_task_id: null,
    kind: 'task',
    meeting_outcome_id: null,
    created_by_fai: false,
    completed_at: null,
    archived: false,
    tags: [],
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  }
}

const board = [
  task(1, 1, 1024), // Alpha, To Do
  task(2, 1, 2048), // Bravo, To Do
  task(3, 1, 3072), // Charlie, To Do
  task(4, 2, 1024), // Delta, In Progress
  task(5, 2, 2048), // Echo, In Progress
]

function order(tasks: Task[], columnId: number) {
  return tasksInColumn(tasks, columnId).map((item) => item.id)
}

describe('planDrop', () => {
  it('reorders within a column taking the hovered card slot', () => {
    const plan = planDrop(board, board, 1, 1, 3)
    expect(plan?.changed).toBe(true)
    expect(order(plan!.tasks, 1)).toEqual([2, 3, 1])
    expect(plan?.afterTaskId).toBe(3)
    expect(plan?.beforeTaskId).toBeUndefined()
  })

  it('reorders upward taking the hovered card slot', () => {
    const plan = planDrop(board, board, 3, 1, 1)
    expect(order(plan!.tasks, 1)).toEqual([3, 1, 2])
    expect(plan?.beforeTaskId).toBe(1)
    expect(plan?.afterTaskId).toBeUndefined()
  })

  it('inserts before a card in another column', () => {
    const plan = planDrop(board, board, 1, 2, 4, false)
    expect(order(plan!.tasks, 2)).toEqual([1, 4, 5])
    expect(plan).toMatchObject({ columnId: 2, beforeTaskId: 4, afterTaskId: undefined })
  })

  it('inserts after a card in another column', () => {
    const plan = planDrop(board, board, 1, 2, 4, true)
    expect(order(plan!.tasks, 2)).toEqual([4, 1, 5])
    expect(plan).toMatchObject({ columnId: 2, beforeTaskId: 5, afterTaskId: 4 })
  })

  // Regression: the drop used to be locked to the first card the pointer
  // entered the column over, so moving to another card before releasing still
  // dropped at the old spot.
  it('follows the pointer to a new card after entering the column', () => {
    const first = planDrop(board, board, 1, 2, 5, true)!
    expect(order(first.tasks, 2)).toEqual([4, 5, 1])

    const second = planDrop(first.tasks, board, 1, 2, 4, false)!
    expect(order(second.tasks, 2)).toEqual([1, 4, 5])
    expect(order(second.tasks, 1)).toEqual([2, 3])

    const drop = planDrop(board, board, 1, 2, 4, false)!
    expect(drop).toMatchObject({ columnId: 2, beforeTaskId: 4, changed: true })
  })

  it('inserts at the top when dropped on a column header', () => {
    const plan = planDrop(board, board, 1, 2, null, false)
    expect(order(plan!.tasks, 2)).toEqual([1, 4, 5])
    expect(plan?.beforeTaskId).toBe(4)
  })

  it('appends when dropped on empty space at the bottom', () => {
    const plan = planDrop(board, board, 1, 2, null, true)
    expect(order(plan!.tasks, 2)).toEqual([4, 5, 1])
    expect(plan?.afterTaskId).toBe(5)
  })

  it('reports no change when the card stays in place', () => {
    expect(planDrop(board, board, 1, 1, null, false)?.changed).toBe(false)
    expect(planDrop(board, board, 3, 1, null, true)?.changed).toBe(false)
    expect(planDrop(board, board, 1, 1, 1)).toBeNull()
  })

  it('returns null for a card that is not in the list', () => {
    expect(planDrop(board, board, 99, 1, 2)).toBeNull()
  })
})
