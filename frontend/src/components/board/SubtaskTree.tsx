import { Fragment } from 'react'
import { Checkbox } from '@/components/ui/checkbox'
import { useTasks } from '../../api/hooks'
import { cn } from '@/lib/utils'
import { TaskTypeIcon } from '../TaskTypeIcon'
import { normalizeKind, taskStatus } from '../../lib/taskType'
import type { BoardColumn, Task } from '../../api/types'

export function SubtaskTree({
  parentId,
  boardId,
  columns,
  onToggle,
}: {
  parentId: number
  boardId: number
  columns?: BoardColumn[]
  onToggle?: (subtask: Task, done: boolean) => void
}) {
  const { data: tasks } = useTasks(boardId)
  return (
    <SubtaskRows
      tasks={tasks ?? []}
      parentId={parentId}
      columns={columns}
      onToggle={onToggle}
      doneColumnId={columns?.find((column) => column.is_done)?.id}
      depth={0}
    />
  )
}

function SubtaskRows({
  tasks,
  parentId,
  columns,
  onToggle,
  doneColumnId,
  depth,
}: {
  tasks: Task[]
  parentId: number
  columns?: BoardColumn[]
  onToggle?: (subtask: Task, done: boolean) => void
  doneColumnId?: number
  depth: number
}) {
  const children = tasks
    .filter((task) => task.parent_task_id === parentId)
    .sort((a, b) => a.created_at.localeCompare(b.created_at))

  return (
    <>
      {children.map((subtask) => {
        const done = Boolean(subtask.completed_at) || subtask.column_id === doneColumnId
        const isStep = normalizeKind(subtask.kind) === 'step'
        return (
          <Fragment key={subtask.id}>
            <div
              className="flex items-center gap-2 text-sm"
              style={depth ? { paddingLeft: depth * 16 } : undefined}
            >
              {isStep ? (
                <Checkbox
                  checked={done}
                  aria-label={`Mark ${subtask.title} done`}
                  disabled={!onToggle}
                  onCheckedChange={(checked) => onToggle?.(subtask, Boolean(checked))}
                />
              ) : (
                <span className="inline-block size-4 shrink-0" />
              )}
              <TaskTypeIcon kind={subtask.kind} status={taskStatus(subtask, columns)} size={14} />
              <span className={cn(done && 'text-muted-foreground line-through')}>
                {subtask.title}
              </span>
            </div>
            <SubtaskRows
              tasks={tasks}
              parentId={subtask.id}
              columns={columns}
              onToggle={onToggle}
              doneColumnId={doneColumnId}
              depth={depth + 1}
            />
          </Fragment>
        )
      })}
    </>
  )
}
