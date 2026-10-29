import { Sparkles } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { useAITask, useAIStatus, useCreateTask } from '../../api/hooks'

export function NewTaskDialog({
  open,
  onOpenChange,
  boardId,
  columnId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  boardId?: number
  columnId?: number
}) {
  const [title, setTitle] = useState('')
  const createTask = useCreateTask(boardId)
  const { data: aiStatus } = useAIStatus()
  const aiTask = useAITask(boardId)
  const aiReady = Boolean(aiStatus?.configured)
  const busy = createTask.isPending || aiTask.isPending

  async function submit() {
    const value = title.trim()
    if (!value || !boardId || !columnId || busy) return
    try {
      await createTask.mutateAsync({ board_id: boardId, column_id: columnId, title: value })
      setTitle('')
      onOpenChange(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not create task')
    }
  }

  async function submitWithAI() {
    const value = title.trim()
    if (!value || !boardId || !columnId || busy) return
    try {
      await aiTask.mutateAsync({ text: value, board_id: boardId, column_id: columnId })
      setTitle('')
      onOpenChange(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not create task')
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next)
        if (!next) setTitle('')
      }}
    >
      <DialogContent className="rounded-lg">
        <DialogHeader>
          <DialogTitle>New task</DialogTitle>
          <DialogDescription>
            {aiReady
              ? 'Describe it naturally. AI sets the title, due date, and priority.'
              : 'It lands in the first column of this board.'}
          </DialogDescription>
        </DialogHeader>
        <Input
          autoFocus
          value={title}
          placeholder={aiReady ? 'Call the plumber tomorrow at 9am, high priority' : 'Task title'}
          aria-label={aiReady ? 'New task in natural language' : 'Task title'}
          disabled={busy}
          onChange={(event) => setTitle(event.target.value)}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing) return
            if (event.key === 'Enter') {
              event.preventDefault()
              void (aiReady ? submitWithAI() : submit())
            }
          }}
        />
        {aiReady && (
          <span className="-mt-1 text-[0.65rem] text-muted-foreground">
            {aiTask.isPending ? 'Reading the details…' : 'AI reads dates and priority from your words'}
          </span>
        )}
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          {aiReady && (
            <Button
              variant="outline"
              disabled={!title.trim() || !columnId || busy}
              onClick={() => void submit()}
            >
              Create task
            </Button>
          )}
          <Button
            disabled={!title.trim() || !columnId || busy}
            onClick={() => void (aiReady ? submitWithAI() : submit())}
          >
            {aiReady && <Sparkles />}
            {aiReady ? 'Add with AI' : 'Create task'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
