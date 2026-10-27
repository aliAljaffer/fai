import { useDroppable } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { Button } from '@/components/ui/button'
import type { BoardColumn as BoardColumnType, Task } from '../../api/types'
import { TaskCard } from './TaskCard'

export function BoardColumn({
  column,
  tasks,
  onTaskClick,
  onAddTask,
}: {
  column: BoardColumnType
  tasks: Task[]
  onTaskClick: (task: Task) => void
  onAddTask: (columnId: number) => void
}) {
  const { setNodeRef } = useDroppable({ id: `column-${column.id}` })

  return (
    <div className="w-70 shrink-0 rounded-lg border bg-card">
      <div className="flex items-center justify-between border-b px-3 py-2.5">
        <h3 className="text-sm font-medium">{column.name}</h3>
        <span className="text-xs text-muted-foreground">{tasks.length}</span>
      </div>
      <div ref={setNodeRef} className="flex min-h-15 flex-col gap-2 p-2">
        <SortableContext items={tasks.map((t) => t.id)} strategy={verticalListSortingStrategy}>
          {tasks.map((task) => (
            <TaskCard key={task.id} task={task} onClick={() => onTaskClick(task)} />
          ))}
        </SortableContext>
      </div>
      <Button
        variant="ghost"
        className="w-full justify-start text-muted-foreground"
        onClick={() => onAddTask(column.id)}
      >
        + Add task
      </Button>
    </div>
  )
}
