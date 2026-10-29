import { addDays, format, parseISO } from 'date-fns'
import {
  ArrowRight,
  Bell,
  CalendarDays,
  CheckCircle2,
  ChevronRight,
  Clock,
  Plus,
  TriangleAlert,
  type LucideIcon,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { TagChip } from '../components/TagChip'
import { TaskTypeIcon } from '../components/TaskTypeIcon'
import {
  useBoards,
  useImportedCalendars,
  useReminders,
  useWorkspaceTasks,
} from '../api/hooks'
import { formatDueIn } from '../lib/format'
import { cn } from '@/lib/utils'
import { useUIStore } from '../store/ui'
import { useWorkspaceStore } from '../store/workspace'
import type { ImportedEvent, Reminder, Task } from '../api/types'

const priorityRank: Record<string, number> = { high: 0, medium: 1, low: 2 }

function sortTasks(tasks: Task[]) {
  return [...tasks].sort(
    (a, b) =>
      (priorityRank[a.priority] ?? 3) - (priorityRank[b.priority] ?? 3) ||
      (a.due_date ?? '').localeCompare(b.due_date ?? '') ||
      a.title.localeCompare(b.title),
  )
}

function greetingFor(hour: number) {
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
}

function StatTile({
  icon: Icon,
  label,
  value,
  accent,
  onClick,
}: {
  icon: LucideIcon
  label: string
  value: number
  accent: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex items-center gap-3 rounded-xl border bg-card p-4 text-left transition-colors hover:bg-muted/40 focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:outline-none"
    >
      <span
        className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted"
        style={{ color: accent }}
      >
        <Icon className="size-4" />
      </span>
      <span className="min-w-0">
        <span className="block text-2xl leading-none font-semibold tabular-nums">{value}</span>
        <span className="mt-1 block truncate text-xs text-muted-foreground">{label}</span>
      </span>
    </button>
  )
}

function TaskRow({ task, onOpen }: { task: Task; onOpen: (id: number) => void }) {
  return (
    <button
      type="button"
      onClick={() => onOpen(task.id)}
      className="group flex w-full items-center gap-3 rounded-lg px-2 py-2 text-left transition-colors hover:bg-muted/60 focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:outline-none"
    >
      <TaskTypeIcon
        kind={task.kind}
        size={16}
        label={task.kind === 'arc' ? 'Arc' : task.kind === 'step' ? 'Step' : 'Task'}
      />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium">{task.title}</span>
        <span className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
          {task.ref && <span className="font-mono text-[0.65rem]">{task.ref}</span>}
          {(task.tags ?? []).map((tag) => (
            <TagChip
              key={tag.id}
              tag={tag}
              className="rounded-full px-1.5 py-0.5 text-[0.65rem] text-muted-foreground"
            />
          ))}
        </span>
      </span>
      {task.priority === 'high' && (
        <span className="shrink-0 text-[0.65rem] font-medium tracking-wide text-destructive uppercase">
          High
        </span>
      )}
      <ChevronRight className="size-4 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
    </button>
  )
}

function EmptyLine({ children }: { children: React.ReactNode }) {
  return <p className="px-2 py-6 text-center text-sm text-muted-foreground">{children}</p>
}

function eventTime(event: ImportedEvent) {
  if (event.all_day) return 'All day'
  const start = parseISO(event.start)
  if (Number.isNaN(start.getTime())) return ''
  if (!event.end) return format(start, 'HH:mm')
  const end = parseISO(event.end)
  if (Number.isNaN(end.getTime())) return format(start, 'HH:mm')
  return `${format(start, 'HH:mm')} - ${format(end, 'HH:mm')}`
}

function ReminderRow({ reminder }: { reminder: Reminder }) {
  const dueIn = formatDueIn(reminder.remind_at)
  const overdue = dueIn.startsWith('overdue')
  return (
    <div className="flex items-start justify-between gap-3 py-2 first:pt-0 last:pb-0">
      <div className="min-w-0">
        <p className="truncate text-sm">{reminder.title}</p>
        <p className="text-xs text-muted-foreground">
          {format(parseISO(reminder.remind_at), 'd MMM, HH:mm')}
        </p>
      </div>
      <span
        className={cn(
          'shrink-0 rounded-full px-2 py-0.5 text-[0.65rem] font-medium',
          overdue ? 'bg-destructive/10 text-destructive' : 'bg-attention/15 text-attention',
        )}
      >
        {dueIn}
      </span>
    </div>
  )
}

export function Overview() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { selectedBoardId, requestNewTask } = useUIStore()
  const { data: boards } = useBoards(workspaceId)
  const board = boards?.find((item) => item.id === selectedBoardId) ?? boards?.[0]
  const { data: tasks, isLoading: tasksLoading } = useWorkspaceTasks(workspaceId)
  const { data: reminders } = useReminders(workspaceId)

  const [now] = useState(() => new Date())
  const today = format(now, 'yyyy-MM-dd')
  const tomorrow = format(addDays(now, 1), 'yyyy-MM-dd')
  const weekAhead = format(addDays(now, 7), 'yyyy-MM-dd')
  const { data: calendars } = useImportedCalendars(today, tomorrow, workspaceId)

  const openTasks = useMemo(
    () => (tasks ?? []).filter((task) => !task.completed_at && !task.parent_task_id),
    [tasks],
  )
  const overdue = useMemo(
    () => sortTasks(openTasks.filter((task) => task.due_date && task.due_date < today)),
    [openTasks, today],
  )
  const dueToday = useMemo(
    () => sortTasks(openTasks.filter((task) => task.due_date === today)),
    [openTasks, today],
  )
  const upcoming = useMemo(
    () =>
      sortTasks(
        openTasks.filter(
          (task) => task.due_date && task.due_date > today && task.due_date <= weekAhead,
        ),
      ),
    [openTasks, today, weekAhead],
  )
  const completedToday = useMemo(
    () =>
      (tasks ?? []).filter(
        (task) => task.completed_at && task.completed_at.slice(0, 10) === today,
      ),
    [tasks, today],
  )

  const events = useMemo(
    () =>
      (calendars ?? [])
        .flatMap((calendar) =>
          calendar.events.map((event) => ({
            event,
            sourceName: calendar.source.name,
            color: calendar.source.color,
          })),
        )
        .sort((a, b) => a.event.start.localeCompare(b.event.start)),
    [calendars],
  )

  const upcomingReminders = useMemo(
    () =>
      [...(reminders ?? [])]
        .filter((reminder) => !reminder.sent)
        .sort((a, b) => new Date(a.remind_at).getTime() - new Date(b.remind_at).getTime())
        .slice(0, 4),
    [reminders],
  )

  const upcomingGroups = useMemo(() => {
    const groups = new Map<string, Task[]>()
    for (const task of upcoming.slice(0, 8)) {
      const date = task.due_date!
      groups.set(date, [...(groups.get(date) ?? []), task])
    }
    return [...groups.entries()].sort(([a], [b]) => a.localeCompare(b))
  }, [upcoming])

  function openTask(id: number) {
    const next = new URLSearchParams(searchParams)
    next.set('task', String(id))
    setSearchParams(next)
  }

  function dayLabel(date: string) {
    if (date === tomorrow) return 'Tomorrow'
    return format(parseISO(date), 'EEEE d MMM')
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-col gap-1">
          <p className="text-xs font-medium tracking-[0.14em] text-muted-foreground uppercase">
            {format(now, 'EEEE d MMMM')}
          </p>
          <h1 className="text-2xl font-semibold tracking-tight">{greetingFor(now.getHours())}</h1>
          <p className="text-sm text-muted-foreground">
            Here is what the day looks like. Open a task to see the details.
          </p>
        </div>
        <Button
          size="sm"
          disabled={!board?.id}
          onClick={() => {
            navigate('/board')
            requestNewTask()
          }}
        >
          <Plus />
          New task
        </Button>
      </header>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatTile
          icon={TriangleAlert}
          label="Overdue"
          value={overdue.length}
          accent="var(--destructive)"
          onClick={() => navigate('/list')}
        />
        <StatTile
          icon={CalendarDays}
          label="Due today"
          value={dueToday.length}
          accent="var(--primary)"
          onClick={() => navigate('/board')}
        />
        <StatTile
          icon={Clock}
          label="Next 7 days"
          value={upcoming.length}
          accent="var(--status-doing)"
          onClick={() => navigate('/calendar')}
        />
        <StatTile
          icon={CheckCircle2}
          label="Completed today"
          value={completedToday.length}
          accent="var(--status-done)"
          onClick={() => navigate('/stats')}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between gap-2">
            <div>
              <CardTitle>Today</CardTitle>
              <CardDescription>Overdue and due today, across every board.</CardDescription>
            </div>
            <Button variant="ghost" size="xs" onClick={() => navigate('/board')}>
              Board
              <ArrowRight />
            </Button>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            {tasksLoading && (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-2/3" />
              </div>
            )}
            {!tasksLoading && overdue.length > 0 && (
              <section className="flex flex-col">
                <h2 className="flex items-center gap-1.5 px-2 text-xs font-medium text-destructive">
                  <TriangleAlert className="size-3.5" />
                  Overdue
                </h2>
                {overdue.map((task) => (
                  <TaskRow key={task.id} task={task} onOpen={openTask} />
                ))}
              </section>
            )}
            {!tasksLoading && (
              <section className="flex flex-col">
                <h2 className="px-2 text-xs font-medium text-muted-foreground">Due today</h2>
                {dueToday.length ? (
                  dueToday.map((task) => (
                    <TaskRow key={task.id} task={task} onOpen={openTask} />
                  ))
                ) : (
                  <EmptyLine>Nothing due today. Enjoy the clear runway.</EmptyLine>
                )}
              </section>
            )}
          </CardContent>
        </Card>

        <div className="flex flex-col gap-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between gap-2">
              <CardTitle>Schedule</CardTitle>
              <Button variant="ghost" size="xs" onClick={() => navigate('/calendar')}>
                Calendar
                <ArrowRight />
              </Button>
            </CardHeader>
            <CardContent>
              {events.length ? (
                <ul className="flex flex-col divide-y">
                  {events.map(({ event, sourceName, color }) => (
                    <li key={event.id}>
                      <button
                        type="button"
                        onClick={() => navigate('/calendar')}
                        className="flex w-full items-start gap-3 py-2 text-left transition-colors first:pt-0 last:pb-0 hover:text-primary focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:outline-none"
                      >
                        <span className="w-20 shrink-0 pt-0.5 text-xs tabular-nums text-muted-foreground">
                          {eventTime(event)}
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-sm">{event.title}</span>
                          <span className="mt-0.5 flex items-center gap-1.5 text-[0.65rem] text-muted-foreground">
                            <span
                              aria-hidden
                              className="size-1.5 shrink-0 rounded-full"
                              style={{ backgroundColor: color }}
                            />
                            {sourceName}
                            {event.location && <span className="truncate">· {event.location}</span>}
                          </span>
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
              ) : (
                <EmptyLine>No events on the calendar today.</EmptyLine>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between gap-2">
              <CardTitle className="flex items-center gap-1.5">
                <Bell className="size-3.5 text-attention" />
                Reminders
              </CardTitle>
              <Button variant="ghost" size="xs" onClick={() => navigate('/board')}>
                Board
                <ArrowRight />
              </Button>
            </CardHeader>
            <CardContent>
              {upcomingReminders.length ? (
                <div className="flex flex-col divide-y">
                  {upcomingReminders.map((reminder) => (
                    <ReminderRow key={reminder.id} reminder={reminder} />
                  ))}
                </div>
              ) : (
                <EmptyLine>No reminders coming up.</EmptyLine>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      {upcomingGroups.length > 0 && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between gap-2">
            <div>
              <CardTitle>Coming up</CardTitle>
              <CardDescription>The next seven days.</CardDescription>
            </div>
            <Button variant="ghost" size="xs" onClick={() => navigate('/calendar')}>
              Calendar
              <ArrowRight />
            </Button>
          </CardHeader>
          <CardContent className="grid gap-x-8 gap-y-4 md:grid-cols-2">
            {upcomingGroups.map(([date, dayTasks]) => (
              <section key={date}>
                <h2 className="mb-1 px-2 text-xs font-medium text-muted-foreground">
                  {dayLabel(date)}
                </h2>
                <div className="flex flex-col">
                  {dayTasks.map((task) => (
                    <TaskRow key={task.id} task={task} onOpen={openTask} />
                  ))}
                </div>
              </section>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
