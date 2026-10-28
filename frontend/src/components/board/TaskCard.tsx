import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useState } from 'react'
import { ChevronRight, Play, Square } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { cn } from '@/lib/utils'
import { formatDueDate, isPastDue } from '../../lib/format'
import { labelChipStyle } from '../../lib/palette'
import type { Task } from '../../api/types'

interface TaskCardProps {
  task: Task
  onClick?: () => void
  runningTaskId?: number | null
  onToggleTimer?: (task: Task) => void
  isDone?: boolean
  overlay?: boolean
  subtasks?: Task[]
  doneColumnIds?: Set<number>
  onToggleSubtask?: (subtask: Task) => void
}

function TaskCardBody({
  task,
  isDone,
  overlay,
}: {
  task: Task
  isDone?: boolean
  overlay?: boolean
}) {
  const overdue = isPastDue(task.due_date, Boolean(task.completed_at) || isDone)
  const dueLabel = formatDueDate(task.due_date)
  const showMeta = task.priority !== 'medium' || Boolean(dueLabel)

  return (
    <div className="flex flex-col gap-2">
      <p className={cn('line-clamp-3 text-sm leading-snug', !overlay && 'pr-6')}>{task.title}</p>
      {!!task.tags?.length && (
        <div className="flex flex-wrap gap-1">
          {task.tags.map((tag) => (
            <span
              key={tag.id}
              className="inline-flex items-center gap-1 rounded-sm border px-1 py-0.5 text-[0.65rem] text-muted-foreground"
              style={labelChipStyle(tag.color)}
            >
              <span
                className="size-1.5 rounded-full"
                style={{ backgroundColor: tag.color || undefined }}
              />
              {tag.name}
            </span>
          ))}
        </div>
      )}
      {showMeta && (
        <div className="flex min-w-0 items-center gap-2 pr-6 text-xs text-muted-foreground">
          {task.priority !== 'medium' && (
            <span className={cn('capitalize', task.priority === 'high' && 'text-destructive')}>
              {task.priority}
            </span>
          )}
          {dueLabel && (
            <span className={cn('truncate', overdue && 'font-medium text-destructive')}>
              {overdue ? 'Overdue' : 'Due'} {dueLabel}
            </span>
          )}
        </div>
      )}
    </div>
  )
}

function SortableTaskCard(props: TaskCardProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: props.task.id,
    data: { type: 'task' },
  })
  const running = props.runningTaskId === props.task.id
  const [expanded, setExpanded] = useState(false)
  const subtasks = props.subtasks ?? []
  const doneCount = subtasks.filter(
    (subtask) => subtask.completed_at || props.doneColumnIds?.has(subtask.column_id),
  ).length

  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      {...attributes}
      {...listeners}
      role={undefined}
      tabIndex={undefined}
      aria-roledescription={undefined}
      className={cn(
        'group/task relative cursor-grab touch-none rounded-md border bg-card px-2.5 py-2 text-card-foreground transition-colors select-none hover:border-foreground/20 active:cursor-grabbing',
        isDragging && 'opacity-40',
      )}
    >
      <button
        type="button"
        className="absolute inset-0 z-0 cursor-grab rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring/50 active:cursor-grabbing"
        aria-label={`Open task: ${props.task.title}`}
        onClick={props.onClick}
      />
      <div className="pointer-events-none relative z-0">
        <TaskCardBody task={props.task} isDone={props.isDone} />
      </div>
      {subtasks.length > 0 && (
        <div className="pointer-events-none relative z-10 mt-2 flex flex-col gap-1">
          <button
            type="button"
            aria-expanded={expanded}
            className="pointer-events-auto flex w-fit items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
            onClick={(event) => {
              event.stopPropagation()
              setExpanded((current) => !current)
            }}
            onPointerDown={(event) => event.stopPropagation()}
          >
            <ChevronRight
              className={cn('size-3 transition-transform', expanded && 'rotate-90')}
            />
            {doneCount}/{subtasks.length} subtasks
          </button>
          {expanded && (
            <ul className="flex flex-col gap-1 border-t pt-1.5">
              {subtasks.map((subtask) => {
                const done = Boolean(subtask.completed_at) || props.doneColumnIds?.has(subtask.column_id)
                return (
                  <li key={subtask.id} className="flex items-center gap-1.5 text-xs">
                    <Checkbox
                      checked={done}
                      className="pointer-events-auto"
                      aria-label={`Mark ${subtask.title} done`}
                      disabled={!props.onToggleSubtask}
                      onCheckedChange={() => props.onToggleSubtask?.(subtask)}
                      onClick={(event) => event.stopPropagation()}
                      onPointerDown={(event) => event.stopPropagation()}
                    />
                    <span className={cn(done && 'text-muted-foreground line-through')}>
                      {subtask.title}
                    </span>
                  </li>
                )
              })}
            </ul>
          )}
        </div>
      )}
      {props.onToggleTimer && (
        <Button
          size="icon-xs"
          variant="ghost"
          title={running ? 'Stop timer' : 'Start timer'}
          aria-label={`${running ? 'Stop' : 'Start'} timer: ${props.task.title}`}
          className={cn(
            'absolute top-1 right-1 z-10 text-muted-foreground transition-opacity',
            running
              ? 'text-destructive opacity-100'
              : 'opacity-0 group-hover/task:opacity-100 group-focus-within/task:opacity-100 max-md:opacity-100',
          )}
          onClick={(event) => {
            event.stopPropagation()
            props.onToggleTimer?.(props.task)
          }}
          onPointerDown={(event) => event.stopPropagation()}
        >
          {running ? <Square className="fill-current" /> : <Play />}
        </Button>
      )}
    </div>
  )
}

export function TaskCard(props: TaskCardProps) {
  if (props.overlay) {
    return (
      <div className="w-72 rounded-md border bg-card px-2.5 py-2 text-card-foreground shadow-md">
        <TaskCardBody task={props.task} isDone={props.isDone} overlay />
        {(props.subtasks?.length ?? 0) > 0 && (
          <p className="mt-2 text-xs text-muted-foreground">
            {props.subtasks!.length} subtasks
          </p>
        )}
      </div>
    )
  }
  return <SortableTaskCard {...props} />
}
