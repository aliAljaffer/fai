import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { useRunningTime, useStopTime, useTasks } from '../api/hooks'
import { useUIStore } from '../store/ui'
import { formatDuration } from '../lib/format'

export function TimerWidget() {
  const { data: running } = useRunningTime()
  const stopTime = useStopTime()
  const { selectedBoardId } = useUIStore()
  const { data: tasks } = useTasks(selectedBoardId)
  const [, tick] = useState(0)

  useEffect(() => {
    const id = setInterval(() => tick((n) => n + 1), 1000)
    return () => clearInterval(id)
  }, [])

  if (!running) return null

  const task = tasks?.find((t) => t.id === running.task_id)
  const elapsed = Math.floor((Date.now() - new Date(running.started_at).getTime()) / 1000)

  return (
    <div className="ml-4 flex items-center gap-2 rounded-full border px-3 py-1 text-sm">
      <span className="h-2 w-2 animate-pulse rounded-full bg-destructive" />
      <span>{task?.title ?? `Task #${running.task_id}`}</span>
      <span className="text-muted-foreground">{formatDuration(elapsed)}</span>
      <Button size="sm" variant="ghost" onClick={() => stopTime.mutate(running.task_id)}>
        Stop
      </Button>
    </div>
  )
}
