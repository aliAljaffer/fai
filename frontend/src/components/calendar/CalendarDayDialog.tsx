import { MessageSquarePlus, Plus } from 'lucide-react'
import { useState } from 'react'
import { format, parseISO } from 'date-fns'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { useColumns, useCreateTask } from '../../api/hooks'

export function CalendarDayDialog({
  open,
  onOpenChange,
  date,
  boardId,
  allowSummary,
  onMeetingSummary,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  date: string
  boardId: number
  allowSummary: boolean
  onMeetingSummary: () => void
}) {
  const { data: columns } = useColumns(boardId)
  const createTask = useCreateTask(boardId)
  const [title, setTitle] = useState('')

  const column = [...(columns ?? [])]
    .sort((a, b) => a.position - b.position)
    .find((item) => !item.is_done)

  async function addTask() {
    if (!title.trim() || !column || createTask.isPending) return
    try {
      await createTask.mutateAsync({
        board_id: boardId,
        column_id: column.id,
        title: title.trim(),
        due_date: date,
      })
      toast.success('Task added')
      onOpenChange(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not add task')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md rounded-lg">
        <DialogHeader>
          <DialogTitle>{format(parseISO(date), 'd MMM yyyy')}</DialogTitle>
          <DialogDescription>
            {allowSummary
              ? 'Add a task for this day, or log a meeting that already happened.'
              : 'Add a task for this day.'}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          <Input
            autoFocus
            value={title}
            placeholder="Task title"
            aria-label="Task title"
            disabled={createTask.isPending}
            onChange={(event) => setTitle(event.target.value)}
            onKeyDown={(event) => {
              if (event.nativeEvent.isComposing) return
              if (event.key === 'Enter') {
                event.preventDefault()
                void addTask()
              }
            }}
          />
          <Button
            disabled={!title.trim() || !column || createTask.isPending}
            onClick={() => void addTask()}
          >
            <Plus />
            New task
          </Button>
        </div>
        {allowSummary && (
          <DialogFooter className="sm:justify-start">
            <Button variant="outline" onClick={onMeetingSummary}>
              <MessageSquarePlus />
              Meeting summary
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  )
}
