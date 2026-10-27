import { useSearchParams } from 'react-router-dom'
import { useDeleteTask, useTask, useUpdateTask } from '../../api/hooks'
import { TaskDetail } from './TaskDetail'

// One global sheet, driven by ?task=<id>, so any page (and the command palette)
// can open a task without duplicating the state.
export function TaskDetailSheet() {
  const [params, setParams] = useSearchParams()
  const raw = params.get('task')
  const id = raw && Number.isFinite(Number(raw)) ? Number(raw) : undefined
  const { data: task } = useTask(id)
  const updateTask = useUpdateTask(task?.board_id)
  const deleteTask = useDeleteTask(task?.board_id)

  function close() {
    const next = new URLSearchParams(params)
    next.delete('task')
    setParams(next, { replace: true })
  }

  if (!task) return null

  return (
    <TaskDetail
      key={task.id}
      task={task}
      onClose={close}
      onSave={(fields) => updateTask.mutate({ id: task.id, ...fields }, { onSuccess: close })}
      onDelete={() => deleteTask.mutate(task.id, { onSuccess: close })}
    />
  )
}
