import {
  DndContext,
  DragOverlay,
  PointerSensor,
  closestCenter,
  pointerWithin,
  useSensor,
  useSensors,
  type CollisionDetection,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { Skeleton } from '@/components/ui/skeleton'
import {
  useBoards,
  useColumns,
  useCreateTask,
  useMoveTask,
  useRunningTime,
  useStartTime,
  useStopTime,
  useTasks,
} from '../api/hooks'
import { BoardColumn } from '../components/board/BoardColumn'
import { TaskCard } from '../components/board/TaskCard'
import { useUIStore } from '../store/ui'
import type { Task } from '../api/types'

const collisionDetection: CollisionDetection = (args) => {
  const hits = pointerWithin(args)
  const taskHits = hits.filter((hit) => hit.data?.type === 'task')
  const candidates = taskHits.length ? taskHits : hits
  if (!candidates.length) return closestCenter(args)
  const ids = new Set(candidates.map((hit) => hit.id))
  return closestCenter({
    ...args,
    droppableContainers: args.droppableContainers.filter((container) => ids.has(container.id)),
  })
}

export function Board() {
  const { data: boards } = useBoards()
  const { selectedBoardId, setSelectedBoardId, newTaskRequest } = useUIStore()
  const boardId = selectedBoardId ?? boards?.[0]?.id

  useEffect(() => {
    if (!selectedBoardId && boards?.[0]) setSelectedBoardId(boards[0].id)
  }, [boards, selectedBoardId, setSelectedBoardId])

  const { data: columns } = useColumns(boardId)
  const { data: tasks } = useTasks(boardId)
  const createTask = useCreateTask(boardId)
  const moveTask = useMoveTask(boardId)
  const { data: running } = useRunningTime()
  const startTime = useStartTime()
  const stopTime = useStopTime()
  const [, setSearchParams] = useSearchParams()

  const [activeTaskId, setActiveTaskId] = useState<number | null>(null)
  const [composerColumnId, setComposerColumnId] = useState<number | null>(null)
  const handledNewTaskRequest = useRef(0)

  useEffect(() => {
    setComposerColumnId(null)
  }, [boardId])

  useEffect(() => {
    if (newTaskRequest === handledNewTaskRequest.current) return
    const first = [...(columns ?? [])].sort((a, b) => a.position - b.position)[0]
    if (!first) return
    handledNewTaskRequest.current = newTaskRequest
    setComposerColumnId(first.id)
  }, [newTaskRequest, columns])

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }))

  const tasksByColumn = useMemo(() => {
    const map = new Map<number, Task[]>()
    for (const t of tasks ?? []) {
      const list = map.get(t.column_id) ?? []
      list.push(t)
      map.set(t.column_id, list)
    }
    for (const list of map.values()) list.sort((a, b) => a.position - b.position)
    return map
  }, [tasks])

  const activeTask = tasks?.find((t) => t.id === activeTaskId)

  function handleDragStart(event: DragStartEvent) {
    setActiveTaskId(Number(event.active.id))
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveTaskId(null)
    const { active, over } = event
    if (!over) return

    const taskId = Number(active.id)
    const overId = String(over.id)

    let targetColumnId: number
    let beforeTaskId: number | undefined

    if (overId.startsWith('column-')) {
      targetColumnId = Number(overId.replace('column-', ''))
    } else {
      const overTask = tasks?.find((t) => t.id === Number(overId))
      if (!overTask) return
      targetColumnId = overTask.column_id
      beforeTaskId = overTask.id
    }

    const moving = tasks?.find((t) => t.id === taskId)
    if (moving && moving.column_id === targetColumnId && moving.id === beforeTaskId) return

    moveTask.mutate(
      { id: taskId, column_id: targetColumnId, before_task_id: beforeTaskId },
      { onError: (error) => toast.error(error.message) },
    )
  }

  function toggleTimer(task: Task) {
    if (running?.task_id === task.id) stopTime.mutate(task.id)
    else startTime.mutate(task.id)
  }

  if (!boardId || !columns) {
    return (
      <div className="flex gap-4">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-80 w-70 rounded-lg" />
        ))}
      </div>
    )
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={collisionDetection}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      onDragCancel={() => setActiveTaskId(null)}
    >
      <div className="flex items-start gap-4">
        {columns.map((col) => (
          <BoardColumn
            key={col.id}
            column={col}
            tasks={tasksByColumn.get(col.id) ?? []}
            adding={composerColumnId === col.id}
            runningTaskId={running?.task_id ?? null}
            onStartAdding={() => setComposerColumnId(col.id)}
            onCancelAdding={() => setComposerColumnId(null)}
            onTaskClick={(task) => setSearchParams({ task: String(task.id) })}
            onAddTask={async (title) => {
              await createTask.mutateAsync({ board_id: boardId, column_id: col.id, title })
            }}
            onToggleTimer={toggleTimer}
          />
        ))}
      </div>
      <DragOverlay dropAnimation={null}>
        {activeTask ? (
          <TaskCard
            task={activeTask}
            runningTaskId={running?.task_id ?? null}
            onToggleTimer={toggleTimer}
            overlay
          />
        ) : null}
      </DragOverlay>
    </DndContext>
  )
}
