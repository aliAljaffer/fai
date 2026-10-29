import { Bell, Sparkles } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { RefPicker } from '../refs/RefPicker'
import { useRefPicker } from '../refs/useRefPicker'
import { useAIReminder, useAIStatus, useReminders } from '../../api/hooks'
import { useWorkspaceStore } from '../../store/workspace'
import { ReminderCard } from './ReminderCard'

export function RemindersColumn({ canComplete = true }: { canComplete?: boolean }) {
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: reminders } = useReminders(workspaceId)
  const { data: aiStatus } = useAIStatus()
  const aiReminder = useAIReminder(workspaceId)
  const [text, setText] = useState('')
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
        {upcoming.map((reminder) => (
          <ReminderCard key={reminder.id} reminder={reminder} canComplete={canComplete} />
        ))}

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
