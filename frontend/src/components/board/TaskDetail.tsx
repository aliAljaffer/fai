import { useState } from 'react'
import type { Task } from '../../api/types'

export function TaskDetail({
  task,
  onClose,
  onSave,
  onDelete,
}: {
  task: Task
  onClose: () => void
  onSave: (fields: { title: string; description_md: string; priority: string }) => void
  onDelete: () => void
}) {
  const [title, setTitle] = useState(task.title)
  const [description, setDescription] = useState(task.description_md)
  const [priority, setPriority] = useState(task.priority)

  return (
    <div className="overlay" onClick={onClose}>
      <div className="task-detail" onClick={(e) => e.stopPropagation()}>
        <input className="task-detail-title" value={title} onChange={(e) => setTitle(e.target.value)} />
        <textarea
          className="task-detail-description"
          placeholder="Description (markdown)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={8}
        />
        <select value={priority} onChange={(e) => setPriority(e.target.value)}>
          <option value="low">low</option>
          <option value="medium">medium</option>
          <option value="high">high</option>
        </select>
        <div className="task-detail-actions">
          <button
            onClick={() => {
              onSave({ title, description_md: description, priority })
              onClose()
            }}
          >
            Save
          </button>
          <button className="danger" onClick={onDelete}>
            Delete
          </button>
          <button onClick={onClose}>Cancel</button>
        </div>
      </div>
    </div>
  )
}
