import { Clock, Square } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { useBoards, useRunningTime, useStartTime, useStopTime, useTasks } from '../api/hooks'
import { useUIStore } from '../store/ui'
import { useWorkspaceStore } from '../store/workspace'
import { formatDuration } from '../lib/format'

export function TimerWidget() {
  const { data: running } = useRunningTime()
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: boards } = useBoards(workspaceId)
  const { selectedBoardId } = useUIStore()
  const boardId =
    boards?.find((board) => board.id === selectedBoardId)?.id ?? boards?.[0]?.id
  const { data: tasks } = useTasks(boardId)
  const startTime = useStartTime()
  const stopTime = useStopTime()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [, tick] = useState(0)

  useEffect(() => {
    const id = setInterval(() => tick((n) => n + 1), 1000)
    return () => clearInterval(id)
  }, [])

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return (tasks ?? []).filter((task) => !needle || task.title.toLowerCase().includes(needle))
  }, [tasks, query])

  if (running) {
    const task = tasks?.find((t) => t.id === running.task_id)
    const elapsed = Math.floor((Date.now() - new Date(running.started_at).getTime()) / 1000)

    return (
      <div className="flex min-w-0 items-center gap-2 rounded-full border border-destructive/40 bg-destructive/10 px-3 py-1 text-sm">
        <span className="h-2 w-2 animate-pulse rounded-full bg-destructive" />
        <span className="max-w-40 truncate">{task?.title ?? `Task #${running.task_id}`}</span>
        <span className="font-mono text-xs text-muted-foreground">{formatDuration(elapsed)}</span>
        <Button
          size="icon-xs"
          variant="destructive"
          title="Stop timer"
          aria-label="Stop timer"
          onClick={() => stopTime.mutate(running.task_id)}
        >
          <Square className="fill-current" />
        </Button>
      </div>
    )
  }

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        className="text-muted-foreground"
        aria-label="Start a timer"
        onClick={() => setOpen(true)}
      >
        <Clock className="size-3.5" />
        <span className="hidden sm:inline">Timer</span>
      </Button>
      <Dialog
        open={open}
        onOpenChange={(next) => {
          setOpen(next)
          if (!next) setQuery('')
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Start a timer</DialogTitle>
          </DialogHeader>
          <Input
            autoFocus
            aria-label="Find a task"
            placeholder="Find a task…"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
          <div className="flex max-h-80 flex-col gap-1 overflow-y-auto">
            {filtered.map((task) => (
              <button
                key={task.id}
                className="rounded-md border px-3 py-2 text-left text-sm hover:bg-accent"
                onClick={() => {
                  startTime.mutate(task.id)
                  setOpen(false)
                  setQuery('')
                }}
              >
                {task.title}
              </button>
            ))}
            {!filtered.length && (
              <p className="py-4 text-center text-sm text-muted-foreground">No tasks found.</p>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
