import { ArrowLeft } from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { TaskDetail } from '../components/board/TaskDetail'
import { useDeleteTask, useTask, useUpdateTask } from '../api/hooks'

export function TaskDetailPage() {
  const { id } = useParams()
  const taskId = Number(id)
  const navigate = useNavigate()
  const { data: task } = useTask(Number.isFinite(taskId) ? taskId : undefined)
  const updateTask = useUpdateTask(task?.board_id)
  const deleteTask = useDeleteTask(task?.board_id)

  if (!task) return null

  return (
    <div className="flex flex-col gap-3">
      <Button variant="ghost" size="sm" className="w-fit" onClick={() => navigate(-1)}>
        <ArrowLeft />
        Back
      </Button>
      <TaskDetail
        embedded
        task={task}
        onClose={() => navigate(-1)}
        onSave={(fields) => updateTask.mutate({ id: task.id, ...fields })}
        onDelete={() =>
          deleteTask.mutate(task.id, { onSuccess: () => navigate('/board') })
        }
      />
    </div>
  )
}
