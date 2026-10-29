import { Sparkles } from 'lucide-react'
import { useState, type Ref } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAITask, useAIStatus, useCreateTask } from '../../api/hooks'
import type { Task } from '../../api/types'

export function TaskComposer({
  boardId,
  columnId,
  dueDate,
  onCreated,
  inputRef,
  autoFocus,
  placeholder = 'Add a task',
  className,
}: {
  boardId?: number
  columnId?: number
  dueDate?: string
  onCreated?: (task: Task) => void
  inputRef?: Ref<HTMLInputElement>
  autoFocus?: boolean
  placeholder?: string
  className?: string
}) {
  const [title, setTitle] = useState('')
  const createTask = useCreateTask(boardId)
  const aiTask = useAITask(boardId)
  const { data: aiStatus } = useAIStatus()
  const aiReady = Boolean(aiStatus?.configured)
  const busy = createTask.isPending || aiTask.isPending
  const disabled = !boardId || !columnId || busy

  async function submit(withAI: boolean) {
    const value = title.trim()
    if (!value || disabled) return
    try {
      const task = withAI && aiReady
        ? await aiTask.mutateAsync({ text: value, board_id: boardId, column_id: columnId })
        : await createTask.mutateAsync({
            board_id: boardId,
            column_id: columnId,
            title: value,
            due_date: dueDate,
          })
      setTitle('')
      onCreated?.(task)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not create task')
    }
  }

  return (
    <div className={className}>
      <div className="flex items-center gap-2">
        <Input
          ref={inputRef}
          autoFocus={autoFocus}
          value={title}
          placeholder={placeholder}
          aria-label={placeholder}
          disabled={disabled}
          onChange={(event) => setTitle(event.target.value)}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing) return
            if (event.key === 'Enter') {
              event.preventDefault()
              void submit(event.metaKey || event.ctrlKey)
            }
          }}
        />
        {aiReady && (
          <Button
            size="icon"
            variant="outline"
            title="Create with AI"
            aria-label="Create with AI"
            disabled={!title.trim() || disabled}
            onClick={() => void submit(true)}
          >
            <Sparkles />
          </Button>
        )}
        <Button disabled={!title.trim() || disabled} onClick={() => void submit(false)}>
          Add
        </Button>
      </div>
      {aiReady && (
        <p className="mt-1 text-[0.65rem] text-muted-foreground">
          Enter adds the task, {'\u2318'}/Ctrl+Enter lets AI read dates and priority.
        </p>
      )}
    </div>
  )
}
