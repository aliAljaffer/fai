import { useDroppable } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { Plus } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import type { BoardColumn as BoardColumnType, Task } from '../../api/types'
import { TaskCard } from './TaskCard'

export function BoardColumn({
  column,
  tasks,
  subtasksByParent,
  totalCount,
  hasFilters,
  doneColumnIds,
  highlighted,
  adding,
  runningTaskId,
  onStartAdding,
  onCancelAdding,
  onTaskClick,
  onAddTask,
  onToggleTimer,
  onToggleSubtask,
}: {
  column: BoardColumnType
  tasks: Task[]
  subtasksByParent: Map<number, Task[]>
  totalCount: number
  hasFilters: boolean
  doneColumnIds: Set<number>
  highlighted: boolean
  adding: boolean
  runningTaskId: number | null
  onStartAdding: () => void
  onCancelAdding: () => void
  onTaskClick: (task: Task) => void
  onAddTask: (title: string) => Promise<unknown>
  onToggleTimer: (task: Task) => void
  onToggleSubtask: (subtask: Task) => void
}) {
  const { setNodeRef, isOver } = useDroppable({
    id: `column-${column.id}`,
    data: { type: 'column' },
  })
  const [title, setTitle] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (adding) inputRef.current?.focus()
    else setTitle('')
  }, [adding])

  async function submit() {
    const value = title.trim()
    if (!value || submitting) return
    setSubmitting(true)
    try {
      await onAddTask(value)
      setTitle('')
      inputRef.current?.focus()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not add task')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section className="flex w-72 shrink-0 flex-col">
      <div className="flex h-8 items-center justify-between gap-2 border-b pb-2">
        <h2 className="truncate text-sm font-medium">{column.name}</h2>
        <span className="shrink-0 text-xs tabular-nums text-muted-foreground">
          {hasFilters && tasks.length !== totalCount ? `${tasks.length}/${totalCount}` : tasks.length}
        </span>
      </div>
      <div
        ref={setNodeRef}
        className={cn(
          'flex max-h-[calc(100dvh-14rem)] min-h-24 flex-1 flex-col gap-2 overflow-y-auto rounded-md pt-2 pr-0.5 transition-[background-color,box-shadow] duration-150 motion-reduce:transition-none',
          (isOver || highlighted) && 'bg-muted/40 ring-1 ring-primary/25',
        )}
      >
        <SortableContext items={tasks.map((t) => t.id)} strategy={verticalListSortingStrategy}>
          {tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              runningTaskId={runningTaskId}
              isDone={column.is_done}
              subtasks={subtasksByParent.get(task.id)}
              doneColumnIds={doneColumnIds}
              onToggleTimer={onToggleTimer}
              onToggleSubtask={onToggleSubtask}
              onClick={() => onTaskClick(task)}
            />
          ))}
        </SortableContext>
        {!tasks.length && !adding && (
          hasFilters ? (
            <p className="flex h-16 items-center justify-center text-xs text-muted-foreground">
              No matching tasks.
            </p>
          ) : (
            <button
              type="button"
              className="flex h-16 flex-col items-center justify-center gap-0.5 rounded-md border border-dashed text-xs text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
              onClick={onStartAdding}
            >
              <span>No tasks yet</span>
              <span className="font-medium text-foreground">Add the first task</span>
            </button>
          )
        )}
      </div>
      {adding ? (
        <div className="flex flex-col gap-1.5 pt-2">
          <Input
            ref={inputRef}
            value={title}
            placeholder="Task title"
            aria-label={`New task in ${column.name}`}
            disabled={submitting}
            onChange={(event) => setTitle(event.target.value)}
            onKeyDown={(event) => {
              if (event.nativeEvent.isComposing) return
              if (event.key === 'Enter') {
                event.preventDefault()
                void submit()
              }
              if (event.key === 'Escape') {
                event.preventDefault()
                onCancelAdding()
              }
            }}
            onBlur={() => {
              if (!title.trim()) onCancelAdding()
            }}
          />
          <div className="flex items-center justify-between text-[0.65rem] text-muted-foreground">
            <span>Enter to add, Esc to cancel</span>
            <div className="flex items-center gap-1">
              <Button size="xs" variant="ghost" onClick={onCancelAdding}>
                Cancel
              </Button>
              <Button size="xs" disabled={!title.trim() || submitting} onClick={() => void submit()}>
                Add
              </Button>
            </div>
          </div>
        </div>
      ) : (
        <Button
          variant="ghost"
          size="sm"
          className="mt-1 w-full justify-start text-muted-foreground"
          onClick={onStartAdding}
        >
          <Plus />
          Add task
        </Button>
      )}
    </section>
  )
}
