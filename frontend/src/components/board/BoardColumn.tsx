import { SortableContext, useSortable, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { useColumnWidths } from '@/store/columnWidths'
import { ColumnResizeHandle } from './ColumnResizeHandle'
import { columnColorCss, useColumnColors } from '@/store/columnColors'
import { CSS } from '@dnd-kit/utilities'
import { ArrowLeft, ArrowRight, MoreHorizontal, Pencil, Plus } from 'lucide-react'
import { Fragment, useEffect, useRef, useState } from 'react'
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
import type { BoardColumn as BoardColumnType, Reminder, Task } from '../../api/types'
import { CompletedReminderCard } from './ReminderCard'
import { TaskCard } from './TaskCard'
import { TaskInlineEditor } from './TaskInlineEditor'

export function BoardColumn({
  column,
  columns,
  tasks,
  reminders = [],
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
  reminders?: Reminder[]
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

  const { colors } = useColumnColors()
  const storedWidth = useColumnWidths((state) => state.widths[String(column.id)])
  const [previewWidth, setPreviewWidth] = useState<number | null>(null)
  const sectionRef = useRef<HTMLElement | null>(null)
  const width = previewWidth ?? storedWidth

  return (
    <section
      ref={(element) => {
        setNodeRef(element)
        sectionRef.current = element
      }}
      style={{ transform: CSS.Transform.toString(transform), transition, width }}
      className={cn(
        'relative flex flex-col',
        width ? 'shrink-0' : 'min-w-64 flex-1 basis-0',
        isDragging && 'z-10',
      )}
    >
      <ColumnResizeHandle
        columnId={column.id}
        columnName={column.name}
        targetRef={sectionRef}
        onPreview={setPreviewWidth}
      />
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
            className="flex min-w-0 flex-1 cursor-grab touch-none items-center gap-2 rounded-sm outline-none select-none focus-visible:ring-2 focus-visible:ring-ring/50 active:cursor-grabbing"
            aria-label={`Reorder ${column.name}`}
            title="Drag to reorder"
          >
            <span
              aria-hidden
              className="size-2 shrink-0 rounded-full"
              style={{ backgroundColor: columnColorCss(column, colors) }}
            />
            <h2 className="min-w-0 truncate text-[0.8125rem] font-bold tracking-[0.01em]">
              {column.name}
            </h2>
            <span className="font-mono text-[0.6875rem] tabular-nums text-muted-foreground">
              {hasFilters && tasks.length !== totalCount
                ? `${tasks.length + reminders.length}/${totalCount + reminders.length}`
                : tasks.length + reminders.length}
            </span>
          </div>
        )}
        <div className="flex shrink-0 items-center gap-0.5">
          <Button
            variant="ghost"
            size="icon-xs"
            className="text-muted-foreground"
            aria-label={`Add task to ${column.name}`}
            title="Add task"
            onClick={onStartAdding}
          >
            <Plus />
          </Button>
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
          'flex max-h-[calc(100dvh-14rem)] min-h-24 flex-1 flex-col gap-[11px] overflow-y-auto rounded-md px-0.5 pt-[11px] pb-3 transition-[background-color,box-shadow] duration-150 motion-reduce:transition-none',
          (isOver || highlighted) && 'bg-muted/40 ring-1 ring-primary/25',
        )}
      >
        <SortableContext items={tasks.map((t) => t.id)} strategy={verticalListSortingStrategy}>
          {tasks.map((task) => (
            <Fragment key={task.id}>
            <TaskCard
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
            {expandedTask?.id === task.id && (
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
            </Fragment>
          ))}
        </SortableContext>
        {reminders.map((reminder) => (
          <CompletedReminderCard key={`reminder-${reminder.id}`} reminder={reminder} />
        ))}
        {!tasks.length && !reminders.length && !adding && !dragging && (
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
