import { useDroppable } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
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
    <div className="board-column">
      <div className="board-column-header">
        <h3>{column.name}</h3>
        <span className="task-count">{tasks.length}</span>
      </div>
      <div ref={setNodeRef} className="board-column-body">
        <SortableContext items={tasks.map((t) => t.id)} strategy={verticalListSortingStrategy}>
          {tasks.map((task) => (
            <TaskCard key={task.id} task={task} onClick={() => onTaskClick(task)} />
          ))}
        </SortableContext>
      </div>
      <button className="add-task-btn" onClick={() => onAddTask(column.id)}>
        + Add task
      </button>
    </div>
  )
}
