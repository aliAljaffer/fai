import { DndContext, PointerSensor, closestCorners, useSensor, useSensors, type DragEndEvent } from '@dnd-kit/core'
import { useEffect, useMemo, useState } from 'react'
import { Skeleton } from '@/components/ui/skeleton'
import { useBoards, useColumns, useCreateTask, useDeleteTask, useMoveTask, useTasks, useUpdateTask } from '../api/hooks'
import { BoardColumn } from '../components/board/BoardColumn'
import { TaskDetail } from '../components/board/TaskDetail'
import { useUIStore } from '../store/ui'
import type { Task } from '../api/types'

export function Board() {
  const { data: boards } = useBoards()
  const { selectedBoardId, setSelectedBoardId } = useUIStore()
  const boardId = selectedBoardId ?? boards?.[0]?.id

  useEffect(() => {
    if (!selectedBoardId && boards?.[0]) setSelectedBoardId(boards[0].id)
  }, [boards, selectedBoardId, setSelectedBoardId])

  const { data: columns } = useColumns(boardId)
  const { data: tasks } = useTasks(boardId)
  const createTask = useCreateTask(boardId)
  const updateTask = useUpdateTask(boardId)
  const moveTask = useMoveTask(boardId)
  const deleteTask = useDeleteTask(boardId)

  const [activeTask, setActiveTask] = useState<Task | null>(null)

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }))

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

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over) return

    const taskId = Number(active.id)
    const overId = String(over.id)

    let targetColumnId: number
    let beforeTaskId: number | undefined

    if (overId.startsWith('column-')) {
      targetColumnId = Number(overId.replace('column-', ''))
    } else {
      const overTaskId = Number(overId)
      const overTask = tasks?.find((t) => t.id === overTaskId)
      if (!overTask) return
      targetColumnId = overTask.column_id
      beforeTaskId = overTaskId
    }

    moveTask.mutate({ id: taskId, column_id: targetColumnId, before_task_id: beforeTaskId })
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
    <div>
      <DndContext sensors={sensors} collisionDetection={closestCorners} onDragEnd={handleDragEnd}>
        <div className="flex items-start gap-4">
          {columns.map((col) => (
            <BoardColumn
              key={col.id}
              column={col}
              tasks={tasksByColumn.get(col.id) ?? []}
              onTaskClick={setActiveTask}
              onAddTask={(columnId) => {
                const title = prompt('Task title?')
                if (title) createTask.mutate({ board_id: boardId, column_id: columnId, title })
              }}
            />
          ))}
        </div>
      </DndContext>

      {activeTask && (
        <TaskDetail
          task={activeTask}
          onClose={() => setActiveTask(null)}
          onSave={(fields) => updateTask.mutate({ id: activeTask.id, ...fields })}
          onDelete={() => {
            deleteTask.mutate(activeTask.id)
            setActiveTask(null)
          }}
        />
      )}
    </div>
  )
}
