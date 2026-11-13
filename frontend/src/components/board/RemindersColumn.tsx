import { Bell, ChevronLeft, Sparkles } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { RefPicker } from '../refs/RefPicker'
import { useRefPicker } from '../refs/useRefPicker'
import { useAIReminder, useAIStatus, useReminders } from '../../api/hooks'
import { useRemindersPanel } from '../../store/remindersPanel'
import { useWorkspaceStore } from '../../store/workspace'
import { ReminderCard } from './ReminderCard'

export function RemindersColumn({ canComplete = true }: { canComplete?: boolean }) {
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: reminders } = useReminders(workspaceId)
  const { data: aiStatus } = useAIStatus()
  const aiReminder = useAIReminder(workspaceId)
  const [text, setText] = useState('')
  const { open, setOpen } = useRemindersPanel()
  const refs = useRefPicker({ value: text, onChange: setText })

  const upcoming = [...(reminders ?? [])]
    .filter((reminder) => !reminder.sent && !reminder.completed_at)
    .sort((a, b) => new Date(a.remind_at).getTime() - new Date(b.remind_at).getTime())

  function submit() {
    const value = text.trim()
    if (!value || aiReminder.isPending) return
    aiReminder.mutate(value, {
      onSuccess: () => setText(''),
      onError: (error) => toast.error(error.message),
    })
  }

  if (!open) {
    return (
      <button
        type="button"
        aria-expanded={false}
        aria-label={`Show reminders (${upcoming.length})`}
        title="Show reminders"
        className={cn(
          'flex w-11 shrink-0 flex-col items-center gap-2.5 rounded-xl border py-2.5 text-muted-foreground outline-none transition-colors hover:bg-muted/50 hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50',
          upcoming.length > 0 && 'border-attention/40 text-attention hover:text-attention',
        )}
        onClick={() => setOpen(true)}
      >
        <Bell className="size-3.5" />
        <span
          className={cn(
            'min-w-5 rounded-full px-1 text-center font-mono text-[0.6875rem] tabular-nums',
            upcoming.length > 0 && 'bg-attention font-semibold text-attention-foreground',
          )}
        >
          {upcoming.length}
        </span>
        <span className="text-xs font-semibold [writing-mode:vertical-rl]">Reminders</span>
      </button>
    )
  }

  return (
    <section aria-label="Reminders" className="flex w-72 shrink-0 flex-col">
      <div className="flex h-8 items-center justify-between gap-2 border-b pb-2">
        <div className="flex min-w-0 items-center gap-2">
          <Bell aria-hidden className="size-3.5 shrink-0 text-attention" />
          <h2 className="min-w-0 truncate text-[0.8125rem] font-bold tracking-[0.01em]">Reminders</h2>
          <span className="font-mono text-[0.6875rem] tabular-nums text-muted-foreground">
            {upcoming.length}
          </span>
        </div>
        <Button
          variant="ghost"
          size="icon-xs"
          className="text-muted-foreground"
          aria-label="Hide reminders"
          title="Hide reminders"
          onClick={() => setOpen(false)}
        >
          <ChevronLeft />
        </Button>
      </div>

      <div className="flex max-h-[calc(100dvh-14rem)] flex-col gap-[11px] overflow-y-auto px-0.5 pt-[11px]">
        {upcoming.map((reminder) => (
          <ReminderCard key={reminder.id} reminder={reminder} canComplete={canComplete} />
        ))}

        {!upcoming.length && (
          <p className="px-1.5 py-4 text-xs text-muted-foreground">No reminders</p>
        )}
      </div>

      {aiStatus?.configured && (
        <div className="flex flex-col gap-1 px-0.5 pt-[11px]">
          <div className="flex items-center gap-1">
            <Input
              value={text}
              className="border-dashed"
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
              variant="ghost"
              size="icon-sm"
              className="shrink-0 text-attention hover:text-attention"
              aria-label="Create reminder with AI"
              disabled={!text.trim() || aiReminder.isPending}
              onClick={submit}
            >
              <Sparkles />
            </Button>
          </div>
          {aiReminder.isPending && (
            <span className="text-[0.6875rem] text-muted-foreground">Reading the time…</span>
          )}
        </div>
      )}
    </section>
  )
}
