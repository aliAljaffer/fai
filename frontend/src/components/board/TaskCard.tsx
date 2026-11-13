import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import {
  Fragment,
  useState,
  type CSSProperties,
  type KeyboardEventHandler,
  type ReactNode,
} from 'react'
import { Check, ChevronRight, GripVertical, Play, Square } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { cn } from '@/lib/utils'
import { formatDueDate, isPastDue } from '../../lib/format'
import { FaiTaskIcon } from '../FaiTaskIcon'
import { TagChip } from '../TagChip'
import { RepeatIndicator } from './RepeatPicker'
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
  accent?: string
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

interface StepProgress {
  done: number
  total: number
  unit: 'tasks' | 'steps'
  expanded: boolean
  onToggle: () => void
}

function TaskMeta({
  task,
  done,
  steps,
}: {
  task: Task
  done: boolean
  steps?: StepProgress
}) {
  const overdue = isPastDue(task.due_date, done)
  const dueLabel = formatDueDate(task.due_date)
  const completedLabel = formatDueDate(task.completed_at)
  const items: ReactNode[] = []

  if (done) {
    items.push(
      <span key="done" className="inline-flex items-center gap-1 font-semibold text-(--status-done)">
        <Check aria-hidden className="size-3" />
        Done{completedLabel ? ` ${completedLabel}` : ''}
      </span>,
    )
  } else {
    if (task.priority === 'high' || task.priority === 'low') {
      items.push(
        <span key="priority" className="inline-flex items-center gap-[5px] capitalize">
          <span
            aria-hidden
            className={cn(
              'size-[7px] rounded-full',
              task.priority === 'high' ? 'bg-destructive' : 'border-[1.5px] border-muted-foreground',
            )}
          />
          {task.priority}
        </span>,
      )
    }
    if (dueLabel) {
      items.push(
        <span key="due" className="inline-flex min-w-0 items-center gap-1">
          <span
            className={cn(
              'truncate',
              overdue && 'rounded-full bg-destructive/12 px-[7px] py-0.5 font-semibold text-destructive',
            )}
          >
            {overdue ? 'Overdue' : 'Due'} {dueLabel}
          </span>
          {task.repeat && <RepeatIndicator repeat={task.repeat} />}
        </span>,
      )
    }
  }
  if (steps) {
    items.push(
      <button
        key="steps"
        type="button"
        aria-expanded={steps.expanded}
        className="pointer-events-auto inline-flex items-center gap-1 rounded-sm outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50"
        onClick={(event) => {
          event.stopPropagation()
          steps.onToggle()
        }}
        onPointerDown={(event) => event.stopPropagation()}
      >
        <ChevronRight className={cn('size-3 transition-transform', steps.expanded && 'rotate-90')} />
        {steps.done}/{steps.total} {steps.unit}
      </button>,
    )
  }
  if (!items.length) return null

  return (
    <div className="mt-2.5 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 font-mono text-[0.6875rem] text-foreground/70">
      {items.map((item, index) => (
        <Fragment key={index}>
          {index > 0 && <span aria-hidden>·</span>}
          {item}
        </Fragment>
      ))}
    </div>
  )
}

function TaskCardBody({
  task,
  status,
  isDone,
  overlay,
  steps,
}: {
  task: Task
  status?: 'todo' | 'doing' | 'done'
  isDone?: boolean
  overlay?: boolean
  steps?: StepProgress
}) {
  const done = Boolean(task.completed_at) || Boolean(isDone)

  return (
    <div className="flex flex-col">
      <div
        className={cn(
          'mb-1.5 flex min-h-[22px] flex-wrap items-center gap-x-2 gap-y-1',
          !overlay && 'pr-13',
          done && 'opacity-70',
        )}
      >
        <TaskTypeIcon kind={task.kind} status={status} size={13} />
        {task.created_by_fai && <FaiTaskIcon />}
        <span className="font-mono text-[0.625rem] tracking-[0.06em] text-muted-foreground uppercase">
          {task.ref}
        </span>
        {task.tags?.map((tag) => (
          <TagChip
            key={tag.id}
            tag={tag}
            className="h-[22px] rounded-full px-[9px] text-xs leading-none font-semibold"
          />
        ))}
      </div>
      <p
        className={cn(
          'line-clamp-3 text-sm leading-[1.35]',
          done ? 'font-medium text-muted-foreground' : 'font-semibold',
        )}
      >
        {task.title}
      </p>
      {task.description_md && (
        <p className="mt-1 line-clamp-2 text-[0.8125rem] leading-[1.45] text-muted-foreground">
          {task.description_md}
        </p>
      )}
      {steps && !done && (
        <progress
          className="mt-2.5 -mb-1 block h-[3px] w-full appearance-none overflow-clip rounded-full border-0 bg-foreground/10 [&::-moz-progress-bar]:bg-(--task-accent) [&::-webkit-progress-bar]:bg-transparent [&::-webkit-progress-value]:rounded-full [&::-webkit-progress-value]:bg-(--task-accent)"
          aria-label={`${steps.done} of ${steps.total} ${steps.unit} done`}
          max={steps.total}
          value={steps.done}
        />
      )}
      <TaskMeta task={task} done={done} steps={steps} />
    </div>
  )
}

function SortableTaskCard(props: TaskCardProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: props.task.id,
    data: { type: 'task', columnId: props.task.column_id },
    disabled: props.dragDisabled,
  })
  // Pointer drags start anywhere on the card. Keyboard drags stay on the grip
  // so Space and arrows on the card's buttons do not start a drag.
  const { onKeyDown, ...cardListeners } = listeners ?? {}
  const running = props.runningTaskId === props.task.id
  const [stepsExpanded, setStepsExpanded] = useState(false)
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
  const cardDone = Boolean(props.task.completed_at) || Boolean(props.isDone)
  const overdue = isPastDue(props.task.due_date, cardDone)
  const steps: StepProgress | undefined = nodes.length
    ? {
        done: doneCount,
        total: counted.length,
        unit: isArc ? 'tasks' : 'steps',
        expanded: stepsExpanded,
        onToggle: () => setStepsExpanded((current) => !current),
      }
    : undefined

  return (
    <div
      ref={setNodeRef}
      id={`task-${props.task.id}`}
      style={
        {
          transform: CSS.Transform.toString(transform),
          transition,
          '--task-accent': props.accent ?? 'var(--primary)',
        } as CSSProperties
      }
      {...cardListeners}
      className={cn(
        'group/task relative shrink-0 cursor-grab overflow-hidden rounded-xl border bg-card px-[13px] py-3 text-card-foreground transition-colors select-none hover:border-foreground/20 active:cursor-grabbing',
        overdue && 'border-destructive/40 hover:border-destructive/60',
        isDragging && 'opacity-40',
      )}
    >
      <button
        type="button"
        className="absolute inset-0 z-0 rounded-xl outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
        aria-label={`Open task: ${props.task.title}`}
        aria-expanded={props.expanded}
        onClick={props.onClick}
      />
      <button
        type="button"
        {...attributes}
        onKeyDown={onKeyDown as KeyboardEventHandler<HTMLButtonElement>}
        aria-label={`Drag ${props.task.title}`}
        disabled={props.dragDisabled}
        className={cn(
          'absolute top-2.5 right-8 z-10 flex size-5 touch-none items-center justify-center rounded text-muted-foreground opacity-0 outline-none hover:bg-muted focus-visible:opacity-100 focus-visible:ring-2 focus-visible:ring-ring/50 group-hover/task:opacity-100 max-md:opacity-100',
          props.dragDisabled && 'hidden',
        )}
      >
        <GripVertical className="size-3.5" />
      </button>
      <div className="pointer-events-none relative z-0">
        <TaskCardBody task={props.task} status={status} isDone={props.isDone} steps={steps} />
      </div>
      {stepsExpanded && nodes.length > 0 && (
        <ul className="pointer-events-none relative z-10 mt-2 flex flex-col gap-1 border-t pt-1.5">
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
      {props.onToggleTimer && (
        <Button
          size="icon-xs"
          variant="ghost"
          title={running ? 'Stop timer' : 'Start timer'}
          aria-label={`${running ? 'Stop' : 'Start'} timer: ${props.task.title}`}
          className={cn(
            'absolute top-2.5 right-2 z-10 text-muted-foreground transition-opacity',
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
      <div className="w-72 rounded-xl border bg-card px-[13px] py-3 text-card-foreground shadow-md">
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
