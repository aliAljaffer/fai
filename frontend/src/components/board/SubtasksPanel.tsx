import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { useColumns, useCreateSubtask, useMoveTask, useSubtasks } from '../../api/hooks'
import { cn } from '@/lib/utils'
import type { Task } from '../../api/types'

export function SubtasksPanel({ task }: { task: Task }) {
  const { data: subtasks } = useSubtasks(task.id)
  const { data: columns } = useColumns(task.board_id)
  const createSubtask = useCreateSubtask(task.id)
  const moveTask = useMoveTask(task.board_id)
  const [title, setTitle] = useState('')

  const doneColumn = columns?.find((column) => column.is_done)
  const openColumn = columns?.find((column) => !column.is_done)

  function addSubtask() {
    const value = title.trim()
    if (!value) return
    createSubtask.mutate(
      { board_id: task.board_id, column_id: task.column_id, title: value },
      { onError: (error) => toast.error(error.message) },
    )
    setTitle('')
  }

  function toggle(subtask: Task, done: boolean) {
    const target = done ? doneColumn : openColumn
    if (!target) {
      toast.error(done ? 'This board has no done column.' : 'This board has no open column.')
      return
    }
    moveTask.mutate(
      { id: subtask.id, column_id: target.id },
      { onError: (error) => toast.error(error.message) },
    )
  }

  return (
    <div className="flex flex-col gap-2">
      {subtasks?.map((subtask) => {
        const done = Boolean(subtask.completed_at) || Boolean(columns?.some((column) => column.is_done && column.id === subtask.column_id))
        return (
          <div key={subtask.id} className="flex items-center gap-2 text-sm">
            <Checkbox
              checked={done}
              aria-label={`Mark ${subtask.title} done`}
              onCheckedChange={(checked) => toggle(subtask, Boolean(checked))}
            />
            <span className={cn(done && 'text-muted-foreground line-through')}>{subtask.title}</span>
          </div>
        )
      })}
      {subtasks?.length === 0 && <p className="text-sm text-muted-foreground">No subtasks yet.</p>}
      <div className="flex gap-2">
        <Input
          placeholder="New subtask title"
          aria-label="New subtask title"
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing) return
            if (event.key === 'Enter') {
              event.preventDefault()
              addSubtask()
            }
          }}
        />
        <Button variant="outline" disabled={!title.trim() || createSubtask.isPending} onClick={addSubtask}>
          Add
        </Button>
      </div>
    </div>
  )
}
