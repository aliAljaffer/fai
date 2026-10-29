import { cn } from '@/lib/utils'
import {
  STATUS_TITLE,
  TASK_KIND_LABELS,
  normalizeKind,
  statusColor,
  type TaskStatus,
} from '../lib/taskType'

const STROKE = 2

function Arc({ status }: { status: TaskStatus }) {
  return (
    <>
      <path
        d="M9.93 19.73A8 8 0 1 1 19.73 9.93"
        fill="none"
        stroke="currentColor"
        strokeWidth={STROKE}
        strokeLinecap="round"
      />
      {status === 'doing' && (
        <circle cx="17.66" cy="17.66" r="1.6" fill="none" stroke="currentColor" strokeWidth="1.7" />
      )}
      {status === 'done' && <circle cx="17.66" cy="17.66" r="2.2" fill="currentColor" />}
    </>
  )
}

// The crossbar echoes the f, and its stem-end marker gives the same
// hollow-to-filled status progression as the arc and step.
function TaskCrossbar({ status }: { status: TaskStatus }) {
  return (
    <>
      <path d="M4 7.5h16" fill="none" stroke="currentColor" strokeWidth={STROKE} strokeLinecap="round" />
      <path d="M12 7.5v6" fill="none" stroke="currentColor" strokeWidth={STROKE} strokeLinecap="round" />
      {status === 'doing' && (
        <circle cx="12" cy="16.7" r="1.6" fill="none" stroke="currentColor" strokeWidth="1.7" />
      )}
      {status === 'done' && <circle cx="12" cy="16.7" r="2.2" fill="currentColor" />}
    </>
  )
}

function Step({ status }: { status: TaskStatus }) {
  return (
    <>
      <rect x="3" y="10.9" width="7.5" height="2.2" rx="1.1" fill="currentColor" />
      {status === 'done' ? (
        <circle cx="15.5" cy="12" r="2.75" fill="currentColor" />
      ) : (
        <circle cx="15.5" cy="12" r="2.2" fill="none" stroke="currentColor" strokeWidth={STROKE} />
      )}
      {status === 'doing' && (
        <path d="M15.5 9.8a2.2 2.2 0 0 0 0 4.4z" fill="currentColor" />
      )}
    </>
  )
}

export function TaskTypeIcon({
  kind,
  status = 'todo',
  size = 16,
  className,
  label,
}: {
  kind?: string | null
  status?: TaskStatus
  size?: number
  className?: string
  label?: string
}) {
  const type = normalizeKind(kind)
  const accessible = label ? { role: 'img' as const, 'aria-label': label } : { 'aria-hidden': true as const }
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      className={cn('shrink-0', className)}
      style={{ color: statusColor(status) }}
      {...accessible}
    >
      {type === 'arc' && <Arc status={status} />}
      {type === 'task' && <TaskCrossbar status={status} />}
      {type === 'step' && <Step status={status} />}
      <title>{label ?? `${TASK_KIND_LABELS[type]} (${STATUS_TITLE[status]})`}</title>
    </svg>
  )
}
