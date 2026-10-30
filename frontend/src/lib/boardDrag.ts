import type { Task } from '../api/types'

const positionGap = 1024

export type DropPlan = {
  id: number
  columnId: number
  beforeTaskId?: number
  afterTaskId?: number
  changed: boolean
  tasks: Task[]
}

export function tasksInColumn(tasks: Task[], columnId: number) {
  return tasks
    .filter((task) => task.column_id === columnId)
    .sort((a, b) => a.position - b.position)
}

function withPositions(tasks: Task[], ordered: Task[]) {
  const positions = new Map(ordered.map((task, index) => [task.id, (index + 1) * positionGap]))
  return tasks.map((task) =>
    positions.has(task.id) ? { ...task, position: positions.get(task.id)! } : task,
  )
}

// below is the pointer position relative to the hovered card. Leave it unset
// for a same-column reorder so the card takes the hovered card's slot, the
// order dnd-kit's sortable preview shows.
export function planDrop(
  tasks: Task[],
  original: Task[],
  activeId: number,
  targetColumnId: number,
  overTaskId: number | null,
  below?: boolean,
): DropPlan | null {
  const active = tasks.find((task) => task.id === activeId)
  if (!active) return null

  const column = tasksInColumn(tasks, targetColumnId)
  const activeIndex = column.findIndex((task) => task.id === activeId)
  const rest = column.filter((task) => task.id !== activeId)

  let insertAt: number
  if (overTaskId === null) {
    insertAt = below ? rest.length : 0
  } else {
    if (overTaskId === activeId) return null
    const overIndex = rest.findIndex((task) => task.id === overTaskId)
    if (overIndex < 0) return null
    const overIndexBefore = column.findIndex((task) => task.id === overTaskId)
    insertAt =
      below === undefined
        ? overIndex + (activeIndex >= 0 && activeIndex < overIndexBefore ? 1 : 0)
        : overIndex + (below ? 1 : 0)
  }

  const moving = { ...active, column_id: targetColumnId }
  const ordered = [...rest.slice(0, insertAt), moving, ...rest.slice(insertAt)]
  const beforeTaskId = ordered[insertAt + 1]?.id
  const afterTaskId = ordered[insertAt - 1]?.id
  const sameOrder =
    column.length === ordered.length &&
    column.every((task, index) => task.id === ordered[index].id)

  const current = original.find((task) => task.id === activeId)
  const currentColumn = current ? tasksInColumn(original, current.column_id) : []
  const currentIndex = currentColumn.findIndex((task) => task.id === activeId)
  const changed =
    !current ||
    current.column_id !== targetColumnId ||
    currentColumn[currentIndex + 1]?.id !== beforeTaskId ||
    currentColumn[currentIndex - 1]?.id !== afterTaskId

  return {
    id: activeId,
    columnId: targetColumnId,
    beforeTaskId,
    afterTaskId,
    changed,
    tasks: sameOrder
      ? tasks
      : withPositions([...tasks.filter((task) => task.id !== activeId), moving], ordered),
  }
}
