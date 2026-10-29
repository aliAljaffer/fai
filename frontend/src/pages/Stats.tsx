import {
  eachDayOfInterval,
  differenceInCalendarDays,
  format,
  parseISO,
  startOfDay,
  startOfWeek,
  subDays,
  subWeeks,
  addDays,
} from 'date-fns'
import {
  CheckCircle2,
  Clock,
  Flame,
  Gauge,
  ListTodo,
  Timer,
} from 'lucide-react'
import { useMemo, type ComponentType, type ReactNode } from 'react'
import CalendarHeatmap from 'react-calendar-heatmap'
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ComposedChart,
  Line,
  Pie,
  PieChart,
  XAxis,
  YAxis,
} from 'recharts'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { FaCat } from 'react-icons/fa'
import { Skeleton } from '@/components/ui/skeleton'
import {
  useAllColumns,
  useBoards,
  useCompletions,
  useTimeStats,
  useWorkspaceTasks,
  useWorkspaces,
} from '../api/hooks'
import type { BoardColumn } from '../api/types'
import { taskStatus } from '../lib/taskType'

const TODAY = startOfDay(new Date())
const YEAR_AGO = subDays(TODAY, 365)
const THIRTY_START = subDays(TODAY, 29)

function dayKey(date: Date) {
  return format(date, 'yyyy-MM-dd')
}

function round1(value: number) {
  return Math.round(value * 10) / 10
}

const flowConfig = {
  created: { label: 'Created', color: 'var(--chart-3)' },
  completed: { label: 'Completed', color: 'var(--chart-1)' },
} satisfies ChartConfig

const timeConfig = {
  hours: { label: 'Hours tracked', color: 'var(--chart-2)' },
  average: { label: '7-day average', color: 'var(--chart-4)' },
} satisfies ChartConfig

const statusConfig = {
  todo: { label: 'To do', color: 'var(--status-todo)' },
  doing: { label: 'In progress', color: 'var(--status-doing)' },
  done: { label: 'Done', color: 'var(--status-done)' },
} satisfies ChartConfig

const priorityConfig = {
  high: { label: 'High', color: 'var(--destructive)' },
  medium: { label: 'Medium', color: 'var(--attention)' },
  low: { label: 'Low', color: 'var(--muted-foreground)' },
} satisfies ChartConfig

const tagsConfig = {
  value: { label: 'Open tasks', color: 'var(--chart-1)' },
} satisfies ChartConfig

const loadConfig = {
  value: { label: 'Open tasks', color: 'var(--chart-2)' },
} satisfies ChartConfig

const HEAT_LEGEND = [
  'var(--muted)',
  'color-mix(in oklch, var(--primary) 35%, var(--muted))',
  'color-mix(in oklch, var(--primary) 65%, var(--muted))',
  'var(--primary)',
]

function StatCard({
  label,
  value,
  hint,
  icon: Icon,
}: {
  label: string
  value: string
  hint: string
  icon: ComponentType<{ className?: string }>
}) {
  return (
    <Card size="sm">
      <CardContent className="flex items-start justify-between gap-3">
        <div className="grid gap-1">
          <span className="text-xs text-muted-foreground">{label}</span>
          <span className="text-2xl font-semibold tabular-nums">{value}</span>
          <span className="text-xs text-muted-foreground">{hint}</span>
        </div>
        <span className="rounded-md bg-muted p-1.5 text-muted-foreground">
          <Icon className="size-4" />
        </span>
      </CardContent>
    </Card>
  )
}

function ChartCard({
  title,
  description,
  className,
  children,
}: {
  title: string
  description: string
  className?: string
  children: ReactNode
}) {
  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  )
}

function EmptyChart({ message }: { message: string }) {
  return (
    <div className="flex h-64 items-center justify-center text-sm text-muted-foreground">
      {message}
    </div>
  )
}

function DonutLegend({
  items,
}: {
  items: { name: string; value: number; color: string }[]
}) {
  return (
    <ul className="grid w-full gap-2">
      {items.map((item) => (
        <li key={item.name} className="flex items-center justify-between gap-3 text-sm">
          <span className="flex min-w-0 items-center gap-2 text-muted-foreground">
            <span
              className="size-2.5 shrink-0 rounded-[2px]"
              style={{ backgroundColor: item.color }}
            />
            <span className="truncate">{item.name}</span>
          </span>
          <span className="font-medium tabular-nums">{item.value}</span>
        </li>
      ))}
    </ul>
  )
}

export function Stats() {
  const completionsQuery = useCompletions(dayKey(YEAR_AGO), dayKey(TODAY))
  const timeQuery = useTimeStats(dayKey(THIRTY_START), dayKey(TODAY))
  const { data: tasks, isLoading: tasksLoading } = useWorkspaceTasks()
  const { data: boards } = useBoards()
  const columnQueries = useAllColumns(boards)
  const { data: workspaces } = useWorkspaces()

  const completions = completionsQuery.data
  const timeData = timeQuery.data

  const completionByDay = useMemo(
    () => new Map((completions ?? []).map((point) => [point.date, point.count])),
    [completions],
  )

  const createdByDay = useMemo(() => {
    const map = new Map<string, number>()
    for (const task of tasks) {
      const key = dayKey(new Date(task.created_at))
      map.set(key, (map.get(key) ?? 0) + 1)
    }
    return map
  }, [tasks])

  const columnsByBoard = useMemo(() => {
    const map = new Map<number, BoardColumn[]>()
    boards?.forEach((board, index) => {
      map.set(board.id, columnQueries[index]?.data ?? [])
    })
    return map
  }, [boards, columnQueries])

  const openTasks = useMemo(() => tasks.filter((task) => !task.completed_at), [tasks])
  const completedTasks = useMemo(() => tasks.filter((task) => task.completed_at), [tasks])
  const faiTasks = useMemo(() => tasks.filter((task) => task.created_by_fai), [tasks])
  const faiCreatedByDay = useMemo(() => {
    const map = new Map<string, number>()
    for (const task of faiTasks) {
      const key = dayKey(new Date(task.created_at))
      map.set(key, (map.get(key) ?? 0) + 1)
    }
    return map
  }, [faiTasks])

  const last30 = useMemo(() => {
    const secondsByDate = new Map((timeData ?? []).map((point) => [point.date, point.seconds]))
    const days = eachDayOfInterval({ start: THIRTY_START, end: TODAY }).map((day) => {
      const key = dayKey(day)
      return {
        date: key,
        label: format(day, 'MMM d'),
        created: createdByDay.get(key) ?? 0,
        faiCreated: faiCreatedByDay.get(key) ?? 0,
        completed: completionByDay.get(key) ?? 0,
        hours: round1((secondsByDate.get(key) ?? 0) / 3600),
      }
    })
    return days.map((day, index) => {
      const window = days.slice(Math.max(0, index - 6), index + 1)
      const average = window.reduce((sum, item) => sum + item.hours, 0) / window.length
      return { ...day, average: round1(average) }
    })
  }, [completionByDay, createdByDay, faiCreatedByDay, timeData])

  const weekly = useMemo(() => {
    const base = startOfWeek(subWeeks(TODAY, 11), { weekStartsOn: 1 })
    return Array.from({ length: 12 }, (_, week) => {
      const start = addDays(base, week * 7)
      let created = 0
      let completed = 0
      for (let offset = 0; offset < 7; offset++) {
        const day = addDays(start, offset)
        if (day > TODAY) break
        const key = dayKey(day)
        created += createdByDay.get(key) ?? 0
        completed += completionByDay.get(key) ?? 0
      }
      return { label: format(start, 'MMM d'), created, completed }
    })
  }, [completionByDay, createdByDay])

  const weekday = useMemo(() => {
    const labels = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
    const counts = [0, 0, 0, 0, 0, 0, 0]
    const base = startOfWeek(subWeeks(TODAY, 11), { weekStartsOn: 1 })
    for (const day of eachDayOfInterval({ start: base, end: TODAY })) {
      const index = (day.getDay() + 6) % 7
      counts[index] += completionByDay.get(dayKey(day)) ?? 0
    }
    return labels.map((day, index) => ({ day, completed: counts[index] }))
  }, [completionByDay])

  const currentStreak = useMemo(() => {
    let cursor = TODAY
    if ((completionByDay.get(dayKey(cursor)) ?? 0) === 0) cursor = subDays(cursor, 1)
    let streak = 0
    while ((completionByDay.get(dayKey(cursor)) ?? 0) > 0) {
      streak += 1
      cursor = subDays(cursor, 1)
    }
    return streak
  }, [completionByDay])

  const longestStreak = useMemo(() => {
    let best = 0
    let run = 0
    for (const day of eachDayOfInterval({ start: YEAR_AGO, end: TODAY })) {
      if ((completionByDay.get(dayKey(day)) ?? 0) > 0) {
        run += 1
        best = Math.max(best, run)
      } else {
        run = 0
      }
    }
    return best
  }, [completionByDay])

  const statusData = useMemo(() => {
    const counts = { todo: 0, doing: 0, done: 0 }
    for (const task of tasks) {
      counts[taskStatus(task, columnsByBoard.get(task.board_id))] += 1
    }
    return [
      {
        key: 'todo',
        name: 'To do',
        value: counts.todo,
        color: 'var(--status-todo)',
        fill: 'var(--status-todo)',
      },
      {
        key: 'doing',
        name: 'In progress',
        value: counts.doing,
        color: 'var(--status-doing)',
        fill: 'var(--status-doing)',
      },
      {
        key: 'done',
        name: 'Done',
        value: counts.done,
        color: 'var(--status-done)',
        fill: 'var(--status-done)',
      },
    ]
  }, [tasks, columnsByBoard])

  const priorityData = useMemo(() => {
    const counts = new Map<string, number>()
    for (const task of openTasks) {
      const priority = task.priority === 'high' || task.priority === 'low' ? task.priority : 'medium'
      counts.set(priority, (counts.get(priority) ?? 0) + 1)
    }
    return [
      {
        key: 'high',
        name: 'High',
        value: counts.get('high') ?? 0,
        color: 'var(--destructive)',
        fill: 'var(--destructive)',
      },
      {
        key: 'medium',
        name: 'Medium',
        value: counts.get('medium') ?? 0,
        color: 'var(--attention)',
        fill: 'var(--attention)',
      },
      {
        key: 'low',
        name: 'Low',
        value: counts.get('low') ?? 0,
        color: 'var(--muted-foreground)',
        fill: 'var(--muted-foreground)',
      },
    ]
  }, [openTasks])

  const topTags = useMemo(() => {
    const counts = new Map<string, { name: string; value: number; color: string; fill: string }>()
    for (const task of openTasks) {
      for (const tag of task.tags ?? []) {
        const color = tag.color || 'var(--chart-1)'
        const entry = counts.get(tag.name) ?? { name: tag.name, value: 0, color, fill: color }
        entry.value += 1
        counts.set(tag.name, entry)
      }
    }
    return [...counts.values()].sort((a, b) => b.value - a.value).slice(0, 8)
  }, [openTasks])

  const dueData = useMemo(() => {
    const buckets = [
      { name: 'Overdue', value: 0, color: 'var(--destructive)', fill: 'var(--destructive)' },
      { name: 'Today', value: 0, color: 'var(--attention)', fill: 'var(--attention)' },
      { name: 'Next 7 days', value: 0, color: 'var(--chart-3)', fill: 'var(--chart-3)' },
      { name: 'Next 30 days', value: 0, color: 'var(--chart-1)', fill: 'var(--chart-1)' },
      { name: 'Later', value: 0, color: 'var(--chart-2)', fill: 'var(--chart-2)' },
      {
        name: 'No due date',
        value: 0,
        color: 'var(--muted-foreground)',
        fill: 'var(--muted-foreground)',
      },
    ]
    for (const task of openTasks) {
      if (!task.due_date) {
        buckets[5].value += 1
        continue
      }
      const diff = differenceInCalendarDays(parseISO(task.due_date), TODAY)
      if (diff < 0) buckets[0].value += 1
      else if (diff === 0) buckets[1].value += 1
      else if (diff <= 7) buckets[2].value += 1
      else if (diff <= 30) buckets[3].value += 1
      else buckets[4].value += 1
    }
    return buckets.filter((bucket) => bucket.value > 0)
  }, [openTasks])

  const workspaceData = useMemo(() => {
    const names = new Map((workspaces ?? []).map((workspace) => [workspace.id, workspace.name]))
    const counts = new Map<number, number>()
    for (const task of openTasks) {
      counts.set(task.workspace_id, (counts.get(task.workspace_id) ?? 0) + 1)
    }
    return [...counts.entries()]
      .map(([id, value]) => ({ name: names.get(id) ?? `Workspace ${id}`, value }))
      .sort((a, b) => b.value - a.value)
      .slice(0, 8)
  }, [openTasks, workspaces])

  const totalCompleted = (completions ?? []).reduce((sum, point) => sum + point.count, 0)
  const last30Completed = last30.reduce((sum, day) => sum + day.completed, 0)
  const trackedSeconds = (timeData ?? []).reduce((sum, point) => sum + point.seconds, 0)
  const trackedHours = round1(trackedSeconds / 3600)
  const averageHours = round1(trackedSeconds / 3600 / 30)
  const overdueCount = openTasks.filter(
    (task) => task.due_date && differenceInCalendarDays(parseISO(task.due_date), TODAY) < 0,
  ).length
  const completionRate = tasks.length
    ? Math.round((completedTasks.length / tasks.length) * 100)
    : 0

  const cycleDays = useMemo(() => {
    const values: number[] = []
    for (const task of completedTasks) {
      if (!task.completed_at) continue
      const completed = new Date(task.completed_at)
      if (completed < THIRTY_START) continue
      values.push(Math.max(0, differenceInCalendarDays(completed, new Date(task.created_at))))
    }
    if (!values.length) return 0
    return round1(values.reduce((sum, value) => sum + value, 0) / values.length)
  }, [completedTasks])

  const faiCompleted = faiTasks.filter((task) => task.completed_at).length
  const faiOpen = faiTasks.length - faiCompleted
  const faiCompletionRate = faiTasks.length
    ? Math.round((faiCompleted / faiTasks.length) * 100)
    : 0
  const faiShare = tasks.length ? Math.round((faiTasks.length / tasks.length) * 100) : 0
  const flowTotal = last30.reduce((sum, day) => sum + day.created + day.completed, 0)
  const weeklyTotal = weekly.reduce((sum, week) => sum + week.created + week.completed, 0)
  const weekdayTotal = weekday.reduce((sum, day) => sum + day.completed, 0)
  const statusTotal = statusData.reduce((sum, item) => sum + item.value, 0)
  const priorityTotal = priorityData.reduce((sum, item) => sum + item.value, 0)

  if (tasksLoading || completionsQuery.isLoading) {
    return (
      <div className="flex flex-col gap-6">
        <h1 className="text-lg font-semibold">Stats</h1>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
          {Array.from({ length: 6 }, (_, index) => (
            <Skeleton key={index} className="h-24 w-full" />
          ))}
        </div>
        <Skeleton className="h-44 w-full" />
        <div className="grid gap-4 lg:grid-cols-2">
          <Skeleton className="h-80 w-full" />
          <Skeleton className="h-80 w-full" />
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-lg font-semibold">Stats</h1>
        <p className="text-sm text-muted-foreground">
          A year of activity across all workspaces.
        </p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
        <StatCard
          label="Completed"
          value={totalCompleted.toLocaleString()}
          hint={`${last30Completed} in the last 30 days`}
          icon={CheckCircle2}
        />
        <StatCard
          label="Current streak"
          value={`${currentStreak}d`}
          hint={`Longest ${longestStreak} days`}
          icon={Flame}
        />
        <StatCard
          label="Open tasks"
          value={openTasks.length.toLocaleString()}
          hint={`${overdueCount} overdue`}
          icon={ListTodo}
        />
        <StatCard
          label="Completion rate"
          value={`${completionRate}%`}
          hint={`${completedTasks.length} of ${tasks.length} tasks`}
          icon={Gauge}
        />
        <StatCard
          label="Tracked, 30 days"
          value={`${trackedHours}h`}
          hint={`${averageHours}h per day average`}
          icon={Clock}
        />
        <StatCard
          label="Avg. cycle time"
          value={cycleDays ? `${cycleDays}d` : 'n/a'}
          hint="For tasks completed in 30 days"
          icon={Timer}
        />
      </div>

      <div>
        <h2 className="mb-3 flex items-center gap-1.5 text-sm font-medium text-muted-foreground">
          <FaCat className="size-3.5 text-primary" aria-hidden />
          Fai
        </h2>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            label="Added by Fai"
            value={faiTasks.length.toLocaleString()}
            hint={`${faiShare}% of all tasks`}
            icon={FaCat}
          />
          <StatCard
            label="Fai completion"
            value={`${faiCompletionRate}%`}
            hint={`${faiCompleted} of ${faiTasks.length} completed`}
            icon={Gauge}
          />
          <StatCard
            label="Fai still open"
            value={faiOpen.toLocaleString()}
            hint="Created by Fai, not done"
            icon={ListTodo}
          />
          <StatCard
            label="Fai this month"
            value={last30.reduce((sum, day) => sum + day.faiCreated, 0).toLocaleString()}
            hint="Tasks added in the last 30 days"
            icon={FaCat}
          />
        </div>
      </div>

      <div>
        <h2 className="mb-3 text-sm font-medium text-muted-foreground">Completions, past year</h2>
        <div
          role="img"
          aria-label={`Completion heatmap for the past year. ${totalCompleted} tasks completed.`}
          className="heatmap-wrap rounded-lg border p-4"
        >
          <CalendarHeatmap
            startDate={YEAR_AGO}
            endDate={TODAY}
            values={completions ?? []}
            classForValue={(value) => {
              const count = (value as { count?: number } | undefined)?.count ?? 0
              if (count === 0) return 'fill-none'
              if (count < 2) return 'fill-low'
              if (count < 4) return 'fill-mid'
              return 'fill-high'
            }}
          />
        </div>
        <div
          aria-hidden="true"
          className="mt-2 flex items-center justify-end gap-1.5 text-xs text-muted-foreground"
        >
          <span>Less</span>
          {HEAT_LEGEND.map((color) => (
            <span key={color} className="size-3 rounded-sm" style={{ backgroundColor: color }} />
          ))}
          <span>More</span>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <ChartCard
          title="Created vs completed"
          description="Daily flow over the last 30 days"
        >
          {flowTotal === 0 ? (
            <EmptyChart message="No activity in the last 30 days" />
          ) : (
            <ChartContainer config={flowConfig} className="h-64 w-full">
              <AreaChart data={last30} margin={{ left: 0, right: 8, top: 8 }}>
                <defs>
                  <linearGradient id="fill-created" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--color-created)" stopOpacity={0.5} />
                    <stop offset="95%" stopColor="var(--color-created)" stopOpacity={0.05} />
                  </linearGradient>
                  <linearGradient id="fill-completed" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--color-completed)" stopOpacity={0.5} />
                    <stop offset="95%" stopColor="var(--color-completed)" stopOpacity={0.05} />
                  </linearGradient>
                </defs>
                <CartesianGrid vertical={false} />
                <XAxis dataKey="label" tickLine={false} axisLine={false} interval={4} tickMargin={8} />
                <YAxis tickLine={false} axisLine={false} width={28} allowDecimals={false} />
                <ChartTooltip content={<ChartTooltipContent indicator="line" />} />
                <Area
                  type="monotone"
                  dataKey="created"
                  stroke="var(--color-created)"
                  fill="url(#fill-created)"
                  strokeWidth={2}
                />
                <Area
                  type="monotone"
                  dataKey="completed"
                  stroke="var(--color-completed)"
                  fill="url(#fill-completed)"
                  strokeWidth={2}
                />
                <ChartLegend content={<ChartLegendContent />} />
              </AreaChart>
            </ChartContainer>
          )}
        </ChartCard>

        <ChartCard title="Time tracked" description="Hours per day over the last 30 days">
          {trackedSeconds === 0 ? (
            <EmptyChart message="No time tracked in the last 30 days" />
          ) : (
            <ChartContainer config={timeConfig} className="h-64 w-full">
              <ComposedChart data={last30} margin={{ left: 0, right: 8, top: 8 }}>
                <defs>
                  <linearGradient id="fill-hours" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--color-hours)" stopOpacity={0.5} />
                    <stop offset="95%" stopColor="var(--color-hours)" stopOpacity={0.05} />
                  </linearGradient>
                </defs>
                <CartesianGrid vertical={false} />
                <XAxis dataKey="label" tickLine={false} axisLine={false} interval={4} tickMargin={8} />
                <YAxis
                  tickLine={false}
                  axisLine={false}
                  width={34}
                  tickFormatter={(value: number) => `${value}h`}
                />
                <ChartTooltip content={<ChartTooltipContent indicator="line" />} />
                <Area
                  type="monotone"
                  dataKey="hours"
                  stroke="var(--color-hours)"
                  fill="url(#fill-hours)"
                  strokeWidth={2}
                />
                <Line
                  type="monotone"
                  dataKey="average"
                  stroke="var(--color-average)"
                  strokeWidth={2}
                  strokeDasharray="4 4"
                  dot={false}
                />
                <ChartLegend content={<ChartLegendContent />} />
              </ComposedChart>
            </ChartContainer>
          )}
        </ChartCard>

        <ChartCard title="Weekly throughput" description="Created and completed per week, last 12 weeks">
          {weeklyTotal === 0 ? (
            <EmptyChart message="No activity in the last 12 weeks" />
          ) : (
            <ChartContainer config={flowConfig} className="h-64 w-full">
              <BarChart data={weekly} margin={{ left: 0, right: 8, top: 8 }}>
                <CartesianGrid vertical={false} />
                <XAxis dataKey="label" tickLine={false} axisLine={false} interval={1} tickMargin={8} />
                <YAxis tickLine={false} axisLine={false} width={28} allowDecimals={false} />
                <ChartTooltip content={<ChartTooltipContent indicator="line" />} />
                <Bar dataKey="created" fill="var(--color-created)" radius={4} />
                <Bar dataKey="completed" fill="var(--color-completed)" radius={4} />
                <ChartLegend content={<ChartLegendContent />} />
              </BarChart>
            </ChartContainer>
          )}
        </ChartCard>

        <ChartCard title="Completions by weekday" description="When work actually gets finished">
          {weekdayTotal === 0 ? (
            <EmptyChart message="No completions in the last 12 weeks" />
          ) : (
            <ChartContainer config={flowConfig} className="h-64 w-full">
              <BarChart data={weekday} margin={{ left: 0, right: 8, top: 8 }}>
                <CartesianGrid vertical={false} />
                <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} />
                <YAxis tickLine={false} axisLine={false} width={28} allowDecimals={false} />
                <ChartTooltip content={<ChartTooltipContent hideLabel />} />
                <Bar dataKey="completed" fill="var(--color-completed)" radius={4} />
              </BarChart>
            </ChartContainer>
          )}
        </ChartCard>

        <ChartCard title="Task status" description="Every task across all boards">
          {statusTotal === 0 ? (
            <EmptyChart message="No tasks yet" />
          ) : (
            <div className="flex flex-col items-center gap-6 sm:flex-row">
              <div className="relative h-48 w-48 shrink-0">
                <ChartContainer config={statusConfig} className="aspect-square h-48 w-48">
                  <PieChart>
                    <Pie
                      data={statusData}
                      dataKey="value"
                      nameKey="name"
                      innerRadius={52}
                      outerRadius={78}
                      strokeWidth={2}
                      stroke="var(--background)"
                    >
                      {statusData.map((entry) => (
                        <Cell key={entry.key} fill={entry.color} />
                      ))}
                    </Pie>
                    <ChartTooltip
                      content={<ChartTooltipContent nameKey="name" hideLabel />}
                    />
                  </PieChart>
                </ChartContainer>
                <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
                  <span className="text-2xl font-semibold tabular-nums">{statusTotal}</span>
                  <span className="text-xs text-muted-foreground">tasks</span>
                </div>
              </div>
              <DonutLegend items={statusData} />
            </div>
          )}
        </ChartCard>

        <ChartCard title="Open tasks by priority" description="Where attention is needed">
          {priorityTotal === 0 ? (
            <EmptyChart message="No open tasks" />
          ) : (
            <div className="flex flex-col items-center gap-6 sm:flex-row">
              <div className="relative h-48 w-48 shrink-0">
                <ChartContainer config={priorityConfig} className="aspect-square h-48 w-48">
                  <PieChart>
                    <Pie
                      data={priorityData}
                      dataKey="value"
                      nameKey="name"
                      innerRadius={52}
                      outerRadius={78}
                      strokeWidth={2}
                      stroke="var(--background)"
                    >
                      {priorityData.map((entry) => (
                        <Cell key={entry.key} fill={entry.color} />
                      ))}
                    </Pie>
                    <ChartTooltip
                      content={<ChartTooltipContent nameKey="name" hideLabel />}
                    />
                  </PieChart>
                </ChartContainer>
                <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
                  <span className="text-2xl font-semibold tabular-nums">{priorityTotal}</span>
                  <span className="text-xs text-muted-foreground">open</span>
                </div>
              </div>
              <DonutLegend items={priorityData} />
            </div>
          )}
        </ChartCard>

        <ChartCard title="Busiest tags" description="Open tasks per tag">
          {topTags.length === 0 ? (
            <EmptyChart message="No tagged open tasks" />
          ) : (
            <ChartContainer config={tagsConfig} className="h-64 w-full">
              <BarChart data={topTags} layout="vertical" margin={{ left: 8, right: 12 }}>
                <CartesianGrid horizontal={false} />
                <XAxis type="number" hide allowDecimals={false} />
                <YAxis
                  dataKey="name"
                  type="category"
                  tickLine={false}
                  axisLine={false}
                  width={96}
                />
                <ChartTooltip content={<ChartTooltipContent hideLabel />} />
                <Bar dataKey="value" radius={[0, 4, 4, 0]}>
                  {topTags.map((tag) => (
                    <Cell key={tag.name} fill={tag.color} />
                  ))}
                </Bar>
              </BarChart>
            </ChartContainer>
          )}
        </ChartCard>

        <ChartCard title="Due horizon" description="When open tasks are due">
          {dueData.length === 0 ? (
            <EmptyChart message="No open tasks" />
          ) : (
            <ChartContainer config={loadConfig} className="h-64 w-full">
              <BarChart data={dueData} layout="vertical" margin={{ left: 8, right: 12 }}>
                <CartesianGrid horizontal={false} />
                <XAxis type="number" hide allowDecimals={false} />
                <YAxis
                  dataKey="name"
                  type="category"
                  tickLine={false}
                  axisLine={false}
                  width={96}
                />
                <ChartTooltip content={<ChartTooltipContent hideLabel />} />
                <Bar dataKey="value" radius={[0, 4, 4, 0]}>
                  {dueData.map((bucket) => (
                    <Cell key={bucket.name} fill={bucket.color} />
                  ))}
                </Bar>
              </BarChart>
            </ChartContainer>
          )}
        </ChartCard>

        <ChartCard
          title="Workspace load"
          description="Open tasks per workspace"
          className="lg:col-span-2"
        >
          {workspaceData.length === 0 ? (
            <EmptyChart message="No open tasks" />
          ) : (
            <ChartContainer config={loadConfig} className="h-56 w-full">
              <BarChart data={workspaceData} layout="vertical" margin={{ left: 8, right: 12 }}>
                <CartesianGrid horizontal={false} />
                <XAxis type="number" hide allowDecimals={false} />
                <YAxis
                  dataKey="name"
                  type="category"
                  tickLine={false}
                  axisLine={false}
                  width={120}
                />
                <ChartTooltip content={<ChartTooltipContent hideLabel />} />
                <Bar dataKey="value" fill="var(--color-value)" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ChartContainer>
          )}
        </ChartCard>
      </div>
    </div>
  )
}
