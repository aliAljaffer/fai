import { SortableContext, useSortable, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { ArrowLeft, ArrowRight, GripVertical, MoreHorizontal, Pencil, Plus } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import type { BoardColumn as BoardColumnType, Task } from '../../api/types'
import { TaskCard } from './TaskCard'
import { TaskInlineEditor } from './TaskInlineEditor'

export function BoardColumn({
  column,
  columns,
  tasks,
  subtasksByParent,
  totalCount,
  hasFilters,
  doneColumnIds,
  highlighted,
  adding,
  dragging,
  runningTaskId,
  expandedTaskId,
  focusField,
  focusNonce,
  canMoveLeft,
  canMoveRight,
  onStartAdding,
  onCancelAdding,
  onTaskClick,
  onOpenDetails,
  onCloseExpanded,
  onAddTask,
  onRename,
  onMove,
  onToggleTimer,
  onToggleSubtask,
}: {
  column: BoardColumnType
  columns: BoardColumnType[]
  tasks: Task[]
  subtasksByParent: Map<number, Task[]>
  totalCount: number
  hasFilters: boolean
  doneColumnIds: Set<number>
  highlighted: boolean
  adding: boolean
  dragging: boolean
  runningTaskId: number | null
  expandedTaskId: number | null
  focusField?: 'due' | 'status' | 'labels'
  focusNonce?: number
  canMoveLeft: boolean
  canMoveRight: boolean
  onStartAdding: () => void
  onCancelAdding: () => void
  onTaskClick: (task: Task) => void
  onOpenDetails: (task: Task) => void
  onCloseExpanded: () => void
  onAddTask: (title: string) => Promise<unknown>
  onRename: (name: string) => Promise<unknown>
  onMove: (direction: 'left' | 'right') => void
  onToggleTimer: (task: Task) => void
  onToggleSubtask: (subtask: Task) => void
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
    isOver,
  } = useSortable({
    id: `column-${column.id}`,
    data: { type: 'column' },
  })
  const expandedTask = tasks.find((task) => task.id === expandedTaskId)
  const [title, setTitle] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [renaming, setRenaming] = useState(false)
  const [name, setName] = useState(column.name)
  const inputRef = useRef<HTMLInputElement>(null)
  const nameRef = useRef<HTMLInputElement>(null)
  const renameStarted = useRef(false)

  useEffect(() => {
    if (adding) {
      inputRef.current?.focus()
      inputRef.current?.scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: 'smooth' })
    } else {
      setTitle('')
    }
  }, [adding])

  useEffect(() => {
    if (renaming) nameRef.current?.select()
  }, [renaming])

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

  async function submitRename() {
    const value = name.trim()
    if (!renaming || renameStarted.current) return
    renameStarted.current = true
    setRenaming(false)
    if (!value || value === column.name) {
      setName(column.name)
      renameStarted.current = false
      return
    }
    try {
      await onRename(value)
    } catch (error) {
      setName(column.name)
      toast.error(error instanceof Error ? error.message : 'Could not rename column')
    } finally {
      renameStarted.current = false
    }
  }

  return (
    <section
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={cn('flex w-72 shrink-0 flex-col', isDragging && 'relative z-10')}
    >
      <div className="flex h-8 items-center justify-between gap-2 border-b pb-2">
        {renaming ? (
          <Input
            ref={nameRef}
            value={name}
            aria-label={`Rename ${column.name}`}
            className="h-6 min-w-0 px-1 text-sm"
            onChange={(event) => setName(event.target.value)}
            onBlur={() => void submitRename()}
            onKeyDown={(event) => {
              if (event.nativeEvent.isComposing) return
              if (event.key === 'Enter') {
                event.preventDefault()
                void submitRename()
              }
              if (event.key === 'Escape') {
                event.preventDefault()
                setName(column.name)
                setRenaming(false)
              }
            }}
          />
        ) : (
          <div
            {...attributes}
            {...listeners}
            className="flex min-w-0 flex-1 cursor-grab touch-none items-center gap-1 rounded-sm outline-none select-none focus-visible:ring-2 focus-visible:ring-ring/50 active:cursor-grabbing"
            aria-label={`Reorder ${column.name}`}
            title="Drag to reorder"
          >
            <GripVertical
              className="size-3.5 shrink-0 text-muted-foreground/60"
              aria-hidden
            />
            <h2 className="min-w-0 truncate text-sm font-medium">{column.name}</h2>
          </div>
        )}
        <div className="flex shrink-0 items-center gap-1">
          <span className="text-xs tabular-nums text-muted-foreground">
            {hasFilters && tasks.length !== totalCount ? `${tasks.length}/${totalCount}` : tasks.length}
          </span>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-xs"
                  className="text-muted-foreground"
                  aria-label={`Edit ${column.name}`}
                />
              }
            >
              <MoreHorizontal />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onClick={() => {
                  setName(column.name)
                  renameStarted.current = false
                  setRenaming(true)
                }}
              >
                <Pencil />
                Rename
              </DropdownMenuItem>
              <DropdownMenuItem disabled={!canMoveLeft} onClick={() => onMove('left')}>
                <ArrowLeft />
                Move left
              </DropdownMenuItem>
              <DropdownMenuItem disabled={!canMoveRight} onClick={() => onMove('right')}>
                <ArrowRight />
                Move right
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      {adding && (
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
      )}
      <div
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
              columns={columns}
              subtasks={subtasksByParent.get(task.id)}
              subtasksByParent={subtasksByParent}
              doneColumnIds={doneColumnIds}
              onToggleTimer={onToggleTimer}
              onToggleSubtask={onToggleSubtask}
              dragDisabled={expandedTaskId === task.id}
              expanded={expandedTaskId === task.id}
              onClick={() => onTaskClick(task)}
            />
          ))}
        </SortableContext>
        {expandedTask && (
          <div className="fai-expand">
            <div>
          <TaskInlineEditor
            task={expandedTask}
            columns={columns}
            onClose={onCloseExpanded}
            onOpenDetails={() => onOpenDetails(expandedTask)}
            focusField={focusField}
            focusNonce={focusNonce}
          />
            </div>
          </div>
        )}
        {!tasks.length && !adding && !dragging && (
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
      {!adding && (
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
