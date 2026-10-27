import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useCreateSubtask, useSubtasks } from '../../api/hooks'
import type { Task } from '../../api/types'

export function SubtasksPanel({ task }: { task: Task }) {
  const { data: subtasks } = useSubtasks(task.id)
  const createSubtask = useCreateSubtask(task.id)
  const [title, setTitle] = useState('')

  return (
    <div className="flex flex-col gap-2">
      {subtasks?.map((sub) => (
        <div key={sub.id} className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm">
          <span className={sub.completed_at ? 'text-muted-foreground line-through' : ''}>{sub.title}</span>
        </div>
      ))}
      {subtasks?.length === 0 && <p className="text-sm text-muted-foreground">No subtasks yet.</p>}
      <div className="flex gap-2">
        <Input
          placeholder="New subtask title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && title) {
              createSubtask.mutate({ board_id: task.board_id, column_id: task.column_id, title })
              setTitle('')
            }
          }}
        />
        <Button
          variant="outline"
          onClick={() => {
            if (!title) return
            createSubtask.mutate({ board_id: task.board_id, column_id: task.column_id, title })
            setTitle('')
          }}
        >
          Add
        </Button>
      </div>
    </div>
  )
}
