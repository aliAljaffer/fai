import { Bell, Check, MoreHorizontal, RotateCcw, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'
import { useCompleteReminder, useDeleteReminder, useUpdateReminder } from '../../api/hooks'
import type { Reminder } from '../../api/types'
import { formatDueIn, formatTimestamp } from '../../lib/format'
import { TaskRefText } from '../MarkdownRenderer'

const SNOOZE_OPTIONS = [
  { label: '+1 hour', ms: 60 * 60 * 1000 },
  { label: '+1 day', ms: 24 * 60 * 60 * 1000 },
  { label: '+1 week', ms: 7 * 24 * 60 * 60 * 1000 },
]

export function ReminderCard({
  reminder,
  canComplete = true,
}: {
  reminder: Reminder
  canComplete?: boolean
}) {
  const updateReminder = useUpdateReminder()
  const completeReminder = useCompleteReminder()
  const deleteReminder = useDeleteReminder()
  const dueIn = formatDueIn(reminder.remind_at)
  const overdue = dueIn.startsWith('overdue')

  function snooze(ms: number) {
    const due = new Date(reminder.remind_at).getTime()
    const base = due < Date.now() ? Date.now() : due
    updateReminder.mutate({
      id: reminder.id,
      title: reminder.title,
      remind_at: new Date(base + ms).toISOString(),
      task_id: reminder.task_id ?? undefined,
    })
  }

  function complete() {
    if (!canComplete) {
      toast.error('Add a Done column to complete reminders.')
      return
    }
    completeReminder.mutate({ id: reminder.id, completed: true })
  }

  return (
    <div className="group rounded-md border border-attention/40 bg-attention/10 p-2">
      <div className="flex items-start justify-between gap-2">
        <p className="text-sm font-medium break-words"><TaskRefText text={reminder.title} /></p>
        <div className="flex shrink-0 items-center">
          <Button
            variant="ghost"
            size="icon-xs"
            className="text-muted-foreground opacity-0 group-hover:opacity-100 focus-visible:opacity-100 max-md:opacity-100"
            aria-label={`Complete ${reminder.title}`}
            onClick={complete}
          >
            <Check />
          </Button>
          <Button
            variant="ghost"
            size="icon-xs"
            className="text-muted-foreground opacity-0 group-hover:opacity-100 focus-visible:opacity-100 max-md:opacity-100"
            aria-label={`Delete ${reminder.title}`}
            onClick={() => deleteReminder.mutate(reminder.id)}
          >
            <Trash2 />
          </Button>
        </div>
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
                <DropdownMenuItem key={option.label} onClick={() => snooze(option.ms)}>
                  {option.label}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </div>
  )
}

export function CompletedReminderCard({ reminder }: { reminder: Reminder }) {
  const completeReminder = useCompleteReminder()
  const deleteReminder = useDeleteReminder()

  return (
    <div className="group rounded-md border border-attention/40 bg-attention/10 p-2">
      <div className="flex items-start gap-2">
        <Check className="mt-0.5 size-3.5 shrink-0 text-attention" />
        <p className="min-w-0 flex-1 text-sm font-medium break-words text-attention line-through">
          <TaskRefText text={reminder.title} />
        </p>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="icon-xs"
                className="shrink-0 text-muted-foreground opacity-0 group-hover:opacity-100 focus-visible:opacity-100 max-md:opacity-100"
                aria-label={`Actions for ${reminder.title}`}
              />
            }
          >
            <MoreHorizontal />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              onClick={() => completeReminder.mutate({ id: reminder.id, completed: false })}
            >
              <RotateCcw />
              Reopen
            </DropdownMenuItem>
            <DropdownMenuItem variant="destructive" onClick={() => deleteReminder.mutate(reminder.id)}>
              <Trash2 />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 pt-1 text-[0.65rem] text-muted-foreground">
        <Badge className="border-transparent bg-attention text-attention-foreground">
          <Bell />
          Reminder
        </Badge>
        {reminder.completed_at && (
          <span>Done {formatTimestamp(reminder.completed_at)}</span>
        )}
        <span>Created {formatTimestamp(reminder.created_at)}</span>
      </div>
    </div>
  )
}
