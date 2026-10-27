import CalendarHeatmap from 'react-calendar-heatmap'
import { Bar, BarChart, CartesianGrid, XAxis } from 'recharts'
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from '@/components/ui/chart'
import { useCompletions, useTimeStats } from '../api/hooks'

function isoDate(d: Date) {
  return d.toISOString().slice(0, 10)
}

const today = new Date()
const yearAgo = new Date(today)
yearAgo.setDate(yearAgo.getDate() - 365)
const thirtyDaysAgo = new Date(today)
thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)

const completionsConfig = {
  count: { label: 'Completed', color: 'var(--chart-1)' },
} satisfies ChartConfig

const timeConfig = {
  hours: { label: 'Hours tracked', color: 'var(--chart-2)' },
} satisfies ChartConfig

export function Stats() {
  const { data } = useCompletions(isoDate(yearAgo), isoDate(today))
  const { data: timeData } = useTimeStats(isoDate(thirtyDaysAgo), isoDate(today))
  const byDate = new Map((data ?? []).map((p) => [p.date, p.count]))
  const secondsByDate = new Map((timeData ?? []).map((p) => [p.date, p.seconds]))

  const last30 = Array.from({ length: 30 }, (_, i) => {
    const d = new Date(thirtyDaysAgo)
    d.setDate(d.getDate() + i)
    const date = isoDate(d)
    return {
      date: date.slice(5),
      count: byDate.get(date) ?? 0,
      hours: Math.round(((secondsByDate.get(date) ?? 0) / 3600) * 10) / 10,
    }
  })

  return (
    <div className="flex flex-col gap-8">
      <div>
        <h3 className="mb-3 text-sm font-medium text-muted-foreground">Completions, past year</h3>
        <div className="heatmap-wrap rounded-lg border p-4">
          <CalendarHeatmap
            startDate={yearAgo}
            endDate={today}
            values={data ?? []}
            classForValue={(value) => {
              const count = (value as { count?: number } | undefined)?.count ?? 0
              if (count === 0) return 'fill-none'
              if (count < 2) return 'fill-low'
              if (count < 4) return 'fill-mid'
              return 'fill-high'
            }}
          />
        </div>
      </div>

      <div>
        <h3 className="mb-3 text-sm font-medium text-muted-foreground">Completions, last 30 days</h3>
        <ChartContainer config={completionsConfig} className="h-64 w-full">
          <BarChart data={last30}>
            <CartesianGrid vertical={false} />
            <XAxis dataKey="date" tickLine={false} axisLine={false} interval={4} />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Bar dataKey="count" fill="var(--color-count)" radius={4} />
          </BarChart>
        </ChartContainer>
      </div>

      <div>
        <h3 className="mb-3 text-sm font-medium text-muted-foreground">Time tracked, last 30 days</h3>
        <ChartContainer config={timeConfig} className="h-64 w-full">
          <BarChart data={last30}>
            <CartesianGrid vertical={false} />
            <XAxis dataKey="date" tickLine={false} axisLine={false} interval={4} />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Bar dataKey="hours" fill="var(--color-hours)" radius={4} />
          </BarChart>
        </ChartContainer>
      </div>
    </div>
  )
}
