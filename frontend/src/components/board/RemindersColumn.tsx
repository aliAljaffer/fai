import { Bell, Sparkles, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { RefPicker } from '../refs/RefPicker'
import { useRefPicker } from '../refs/useRefPicker'
import {
  useAIReminder,
  useAIStatus,
  useDeleteReminder,
  useReminders,
  useUpdateReminder,
} from '../../api/hooks'
import type { Reminder } from '../../api/types'
import { formatDueIn } from '../../lib/format'
import { useWorkspaceStore } from '../../store/workspace'

const SNOOZE_OPTIONS = [
  { label: '+1 hour', ms: 60 * 60 * 1000 },
  { label: '+1 day', ms: 24 * 60 * 60 * 1000 },
  { label: '+1 week', ms: 7 * 24 * 60 * 60 * 1000 },
]

export function RemindersColumn() {
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: reminders } = useReminders(workspaceId)
  const { data: aiStatus } = useAIStatus()
  const updateReminder = useUpdateReminder()
  const deleteReminder = useDeleteReminder()
  const aiReminder = useAIReminder(workspaceId)
  const [text, setText] = useState('')
  const refs = useRefPicker({ value: text, onChange: setText })

  const upcoming = [...(reminders ?? [])]
    .filter((reminder) => !reminder.sent)
    .sort((a, b) => new Date(a.remind_at).getTime() - new Date(b.remind_at).getTime())

  function snooze(reminder: Reminder, ms: number) {
    const due = new Date(reminder.remind_at).getTime()
    const base = due < Date.now() ? Date.now() : due
    updateReminder.mutate({
      id: reminder.id,
      title: reminder.title,
      remind_at: new Date(base + ms).toISOString(),
      task_id: reminder.task_id ?? undefined,
    })
  }

  function submit() {
    const value = text.trim()
    if (!value || aiReminder.isPending) return
    aiReminder.mutate(value, {
      onSuccess: () => setText(''),
      onError: (error) => toast.error(error.message),
    })
  }

  return (
    <section className="flex w-72 shrink-0 flex-col rounded-md border border-attention/60 bg-attention/5 p-2 ring-1 ring-attention/25">
      <div className="flex h-8 items-center justify-between gap-2 border-b border-attention/30 pb-2">
        <h2 className="flex min-w-0 items-center gap-1.5 truncate text-sm font-medium text-attention">
          <Bell className="size-3.5" />
          Reminders
        </h2>
        <span className="shrink-0 text-xs tabular-nums text-attention">
          {upcoming.length}
        </span>
      </div>

      <div className="flex max-h-[calc(100dvh-14rem)] min-h-24 flex-1 flex-col gap-2 overflow-y-auto pt-2 pr-0.5">
        {upcoming.map((reminder) => {
          const dueIn = formatDueIn(reminder.remind_at)
          const overdue = dueIn.startsWith('overdue')
          return (
            <div
              key={reminder.id}
              className="group rounded-md border border-attention/40 bg-attention/10 p-2"
            >
              <div className="flex items-start justify-between gap-2">
                <p className="text-sm font-medium break-words">{reminder.title}</p>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  className="shrink-0 text-muted-foreground opacity-0 group-hover:opacity-100 focus-visible:opacity-100 max-md:opacity-100"
                  aria-label={`Delete ${reminder.title}`}
                  onClick={() => deleteReminder.mutate(reminder.id)}
                >
                  <Trash2 />
                </Button>
              </div>
              <div className="flex items-center justify-between gap-2 pt-1">
                <Badge
                  className={cn(
                    'border-transparent',
                    overdue
                      ? 'bg-destructive text-destructive-foreground'
                      : 'bg-attention text-attention-foreground',
                  )}
                >
                  {dueIn}
                </Badge>
                <div className="flex items-center gap-1">
                  <span className="text-[0.65rem] text-muted-foreground">
                    {new Date(reminder.remind_at).toLocaleString(undefined, {
                      month: 'short',
                      day: 'numeric',
                      hour: 'numeric',
                      minute: '2-digit',
                    })}
                  </span>
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <Button variant="ghost" size="xs" className="text-muted-foreground">
                          Snooze
                        </Button>
                      }
                    />
                    <DropdownMenuContent>
                      {SNOOZE_OPTIONS.map((option) => (
                        <DropdownMenuItem key={option.label} onClick={() => snooze(reminder, option.ms)}>
                          {option.label}
                        </DropdownMenuItem>
                      ))}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            </div>
          )
        })}

        {!upcoming.length && (
          <p className="flex h-16 items-center justify-center text-xs text-muted-foreground">
            No reminders yet.
          </p>
        )}
      </div>

      {aiStatus?.configured && (
        <div className="flex flex-col gap-1 pt-2">
          <div className="flex items-center gap-1">
            <Input
              value={text}
              placeholder="Remind me in 10 minutes to…"
              aria-label="New reminder in natural language"
              disabled={aiReminder.isPending}
              onChange={(event) => {
                setText(event.target.value)
                refs.handleChange(event.target.value, event.currentTarget)
              }}
              onKeyDown={(event) => {
                if (refs.handleKeyDown(event)) return
                if (event.nativeEvent.isComposing) return
                if (event.key === 'Enter') {
                  event.preventDefault()
                  submit()
                }
              }}
              onBlur={refs.handleBlur}
              onSelect={refs.sync}
            />
            <RefPicker picker={refs} />
            <Button
              size="icon-sm"
              className="shrink-0 bg-attention text-attention-foreground hover:bg-attention/80"
              aria-label="Create reminder with AI"
              disabled={!text.trim() || aiReminder.isPending}
              onClick={submit}
            >
              <Sparkles />
            </Button>
          </div>
          <span className="text-[0.65rem] text-muted-foreground">
            {aiReminder.isPending ? 'Reading the time…' : 'AI sets the time from your words'}
          </span>
        </div>
      )}
    </section>
  )
}
