import { useDroppable } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import type { BoardColumn as BoardColumnType, Task } from '../../api/types'
import { TaskCard } from './TaskCard'

export function BoardColumn({
  column,
  tasks,
  adding,
  runningTaskId,
  onStartAdding,
  onCancelAdding,
  onTaskClick,
  onAddTask,
  onToggleTimer,
}: {
  column: BoardColumnType
  tasks: Task[]
  adding: boolean
  runningTaskId: number | null
  onStartAdding: () => void
  onCancelAdding: () => void
  onTaskClick: (task: Task) => void
  onAddTask: (title: string) => Promise<unknown>
  onToggleTimer: (task: Task) => void
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
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="flex w-70 shrink-0 flex-col rounded-lg border bg-card">
      <div className="flex items-center justify-between border-b px-3 py-2.5">
        <h3 className="text-sm font-medium">{column.name}</h3>
        <span className="text-xs text-muted-foreground">{tasks.length}</span>
      </div>
      <div
        ref={setNodeRef}
        className={cn(
          'flex min-h-15 flex-1 flex-col gap-2 rounded-b-lg p-2 transition-colors',
          isOver && 'bg-muted/60',
        )}
      >
        <SortableContext items={tasks.map((t) => t.id)} strategy={verticalListSortingStrategy}>
          {tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              runningTaskId={runningTaskId}
              onToggleTimer={onToggleTimer}
              onClick={() => onTaskClick(task)}
            />
          ))}
        </SortableContext>
        {!tasks.length && !adding && (
          <div className="flex h-14 items-center justify-center rounded-md border border-dashed text-xs text-muted-foreground">
            Drop tasks here
          </div>
        )}
      </div>
      {adding ? (
        <div className="flex flex-col gap-1.5 border-t p-2">
          <Input
            ref={inputRef}
            value={title}
            placeholder="What needs to be done?"
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
            <span>Enter to add, Esc to close</span>
            <Button size="xs" disabled={!title.trim() || submitting} onClick={() => void submit()}>
              Add
            </Button>
          </div>
        </div>
      ) : (
        <Button
          variant="ghost"
          className="w-full shrink-0 justify-start rounded-t-none text-muted-foreground"
          onClick={onStartAdding}
        >
          + Add task
        </Button>
      )}
    </div>
  )
}
