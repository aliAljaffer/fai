import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useState } from 'react'
import { ChevronRight, GripVertical, Play, Square } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { cn } from '@/lib/utils'
import { formatDueDate, isPastDue } from '../../lib/format'
import { FaiTaskIcon } from '../FaiTaskIcon'
import { TagChip } from '../TagChip'
import { TaskTypeIcon } from '../TaskTypeIcon'
import { normalizeKind, taskStatus } from '../../lib/taskType'
import type { BoardColumn, Task } from '../../api/types'

interface TaskCardProps {
  task: Task
  columns?: BoardColumn[]
  onClick?: () => void
  runningTaskId?: number | null
  onToggleTimer?: (task: Task) => void
  isDone?: boolean
  overlay?: boolean
  subtasks?: Task[]
  subtasksByParent?: Map<number, Task[]>
  doneColumnIds?: Set<number>
  onToggleSubtask?: (subtask: Task) => void
  dragDisabled?: boolean
  expanded?: boolean
}

type SubtaskNode = { task: Task; depth: number }

function flattenSubtasks(
  map: Map<number, Task[]> | undefined,
  parentId: number,
  depth = 0,
): SubtaskNode[] {
  if (!map) return []
  return (map.get(parentId) ?? []).flatMap((task) => [
    { task, depth },
    ...flattenSubtasks(map, task.id, depth + 1),
  ])
}

function TaskCardBody({
  task,
  status,
  isDone,
  overlay,
}: {
  task: Task
  status?: 'todo' | 'doing' | 'done'
  isDone?: boolean
  overlay?: boolean
}) {
  const overdue = isPastDue(task.due_date, Boolean(task.completed_at) || isDone)
  const dueLabel = formatDueDate(task.due_date)
  const showMeta = task.priority !== 'medium' || Boolean(dueLabel)

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-start gap-1.5">
        <TaskTypeIcon kind={task.kind} status={status} size={14} className="mt-0.5" />
        {task.created_by_fai && <FaiTaskIcon className="mt-1" />}
        <p className={cn('line-clamp-3 text-sm leading-snug', !overlay && 'pr-13')}>{task.title}</p>
      </div>
      {!!task.tags?.length && (
        <div className="flex flex-wrap gap-1">
          {task.tags.map((tag) => (
            <TagChip
              key={tag.id}
              tag={tag}
              className="rounded-sm px-1 py-0.5 text-[0.65rem] text-muted-foreground"
            />
          ))}
        </div>
      )}
      {showMeta && (
        <div className="flex min-w-0 items-center gap-2 pr-13 text-xs text-muted-foreground">
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
    data: { type: 'task', columnId: props.task.column_id },
    disabled: props.dragDisabled,
  })
  const running = props.runningTaskId === props.task.id
  const [expanded, setExpanded] = useState(false)
  const isArc = props.task.kind === 'arc'
  const flattened = flattenSubtasks(props.subtasksByParent, props.task.id)
  const nodes: SubtaskNode[] = flattened.length
    ? flattened
    : (props.subtasks ?? []).map((task) => ({ task, depth: 0 }))
  const subtasks = nodes.map((node) => node.task)
  const counted = isArc ? (props.subtasks ?? []) : subtasks
  const status = taskStatus(props.task, props.columns)
  const doneCount = counted.filter(
    (subtask) => subtask.completed_at || props.doneColumnIds?.has(subtask.column_id),
  ).length

  return (
    <div
      ref={setNodeRef}
      id={`task-${props.task.id}`}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={cn(
        'group/task relative rounded-md border bg-card px-2.5 py-2 text-card-foreground transition-colors select-none hover:border-foreground/20',
        isDragging && 'opacity-40',
      )}
    >
      <button
        type="button"
        className="absolute inset-0 z-0 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
        aria-label={`Open task: ${props.task.title}`}
        aria-expanded={props.expanded}
        onClick={props.onClick}
      />
      <button
        type="button"
        {...attributes}
        {...listeners}
        aria-label={`Drag ${props.task.title}`}
        disabled={props.dragDisabled}
        className={cn(
          'absolute top-1 right-8 z-10 flex size-5 touch-none items-center justify-center rounded text-muted-foreground opacity-0 outline-none hover:bg-muted focus-visible:opacity-100 focus-visible:ring-2 focus-visible:ring-ring/50 group-hover/task:opacity-100 max-md:opacity-100',
          props.dragDisabled && 'hidden',
        )}
      >
        <GripVertical className="size-3.5" />
      </button>
      <div className="pointer-events-none relative z-0">
        <TaskCardBody task={props.task} status={status} isDone={props.isDone} />
      </div>
      {nodes.length > 0 && (
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
            {doneCount}/{counted.length} {isArc ? 'tasks' : 'steps'}
          </button>
          {expanded && (
            <ul className="flex flex-col gap-1 border-t pt-1.5">
              {nodes.map(({ task: subtask, depth }) => {
                const done = Boolean(subtask.completed_at) || props.doneColumnIds?.has(subtask.column_id)
                const isStep = normalizeKind(subtask.kind) === 'step'
                return (
                  <li
                    key={subtask.id}
                    className="flex items-center gap-1.5 text-xs"
                    style={depth ? { paddingLeft: depth * 12 } : undefined}
                  >
                    {isStep ? (
                      <Checkbox
                        checked={done}
                        className="pointer-events-auto"
                        aria-label={`Mark ${subtask.title} done`}
                        disabled={!props.onToggleSubtask}
                        onCheckedChange={() => props.onToggleSubtask?.(subtask)}
                        onClick={(event) => event.stopPropagation()}
                        onPointerDown={(event) => event.stopPropagation()}
                      />
                    ) : (
                      <TaskTypeIcon
                        kind={subtask.kind}
                        status={taskStatus(subtask, props.columns)}
                        size={12}
                      />
                    )}
                    {subtask.created_by_fai && <FaiTaskIcon size={11} />}
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
        <TaskCardBody task={props.task} status={taskStatus(props.task, props.columns)} isDone={props.isDone} overlay />
        {(props.subtasks?.length ?? 0) > 0 && (
          <p className="mt-2 text-xs text-muted-foreground">
            {props.subtasks!.length} {props.task.kind === 'arc' ? 'tasks' : 'steps'}
          </p>
        )}
      </div>
    )
  }
  return <SortableTaskCard {...props} />
}
