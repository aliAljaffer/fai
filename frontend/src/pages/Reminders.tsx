import { useState } from 'react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { useCreateReminder, useDeleteReminder, useReminders, useUpdateReminder } from '../api/hooks'
import type { Reminder } from '../api/types'

const SNOOZE_OPTIONS = [
  { label: '+1 hour', ms: 60 * 60 * 1000 },
  { label: '+1 day', ms: 24 * 60 * 60 * 1000 },
  { label: '+1 week', ms: 7 * 24 * 60 * 60 * 1000 },
]

export function Reminders() {
  const { data: reminders } = useReminders()
  const createReminder = useCreateReminder()
  const updateReminder = useUpdateReminder()
  const deleteReminder = useDeleteReminder()

  const [open, setOpen] = useState(false)
  const [title, setTitle] = useState('')
  const [remindAt, setRemindAt] = useState('')

  const sorted = [...(reminders ?? [])].sort(
    (a, b) => new Date(a.remind_at).getTime() - new Date(b.remind_at).getTime(),
  )

  function snooze(reminder: Reminder, ms: number) {
    const base = new Date(reminder.remind_at).getTime() < Date.now() ? Date.now() : new Date(reminder.remind_at).getTime()
    updateReminder.mutate({
      id: reminder.id,
      title: reminder.title,
      remind_at: new Date(base + ms).toISOString(),
      task_id: reminder.task_id ?? undefined,
    })
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex justify-end">
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger render={<Button>+ New reminder</Button>} />
          <DialogContent>
            <DialogHeader>
              <DialogTitle>New reminder</DialogTitle>
            </DialogHeader>
            <div className="flex flex-col gap-3">
              <Input placeholder="Title" value={title} onChange={(e) => setTitle(e.target.value)} />
              <Input type="datetime-local" value={remindAt} onChange={(e) => setRemindAt(e.target.value)} />
            </div>
            <DialogFooter>
              <Button
                onClick={() => {
                  if (!title || !remindAt) return
                  createReminder.mutate({ title, remind_at: new Date(remindAt).toISOString() })
                  setTitle('')
                  setRemindAt('')
                  setOpen(false)
                }}
              >
                Create
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      <div className="flex flex-col gap-2">
        {sorted.map((reminder) => (
          <Card key={reminder.id}>
            <CardContent className="flex items-center justify-between gap-3">
              <div>
                <div className="font-medium">{reminder.title}</div>
                <div className="text-sm text-muted-foreground">
                  {new Date(reminder.remind_at).toLocaleString()}
                </div>
              </div>
              <div className="flex items-center gap-2">
                {reminder.sent && <Badge variant="secondary">sent</Badge>}
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <Button size="sm" variant="outline">
                        Snooze
                      </Button>
                    }
                  />
                  <DropdownMenuContent>
                    {SNOOZE_OPTIONS.map((opt) => (
                      <DropdownMenuItem key={opt.label} onClick={() => snooze(reminder, opt.ms)}>
                        {opt.label}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
                <Button size="sm" variant="ghost" onClick={() => deleteReminder.mutate(reminder.id)}>
                  Delete
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
