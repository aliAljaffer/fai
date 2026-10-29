import { Check, Play, Square, X } from 'lucide-react'
import { toast } from 'sonner'
import { addDays, format } from 'date-fns'
import { useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useDebounced } from '@/lib/useDebounced'
import { cn } from '@/lib/utils'
import { formatDueDate } from '@/lib/format'
import { normalizeKind } from '@/lib/taskType'
import { MarkdownRenderer } from '../MarkdownRenderer'
import { LabelEditor } from './LabelEditor'
import { SubtaskTree } from './SubtaskTree'
import { LabelSuggestions } from './TaskDetail'
import {
  useAIBreakdown,
  useAIStatus,
  useJiraLink,
  useMoveTask,
  useRunningTime,
  useStartTime,
  useStopTime,
  useSubtasks,
  useTaskAttachments,
  useTaskComments,
  useUpdateTask,
  useCreateSubtask,
} from '../../api/hooks'
import type { BoardColumn, Task } from '../../api/types'

export function TaskInlineEditor({
  task,
  columns,
  onClose,
  onOpenDetails,
  autoFocusTitle,
  focusField,
  focusNonce,
}: {
  task: Task
  columns: BoardColumn[]
  onClose: () => void
  onOpenDetails: () => void
  autoFocusTitle?: boolean
  focusField?: 'title' | 'due' | 'status' | 'labels'
  focusNonce?: number
}) {
  const update = useUpdateTask(task.board_id)
  const move = useMoveTask(task.board_id)
  const { data: subtasks } = useSubtasks(task.id)
  const { data: comments } = useTaskComments(task.id)
  const { data: attachments } = useTaskAttachments(task.id)
  const { data: jira } = useJiraLink(task.id)
  const { data: aiStatus } = useAIStatus()
  const breakdown = useAIBreakdown(task.id)
  const { data: running } = useRunningTime()
  const startTime = useStartTime()
  const stopTime = useStopTime()
  const [title, setTitle] = useState(task.title)
  const [editingTitle, setEditingTitle] = useState(Boolean(autoFocusTitle))
  const [description, setDescription] = useState(task.description_md)
  const [editingDescription, setEditingDescription] = useState(false)
  const [newSubtask, setNewSubtask] = useState('')
  const [indentParentId, setIndentParentId] = useState<number | null>(null)
  const createSubtask = useCreateSubtask(task.id)
  const debouncedDescription = useDebounced(description, 800)
  const lastSaved = useRef(task.description_md)
  const titleRef = useRef<HTMLInputElement>(null)
  const descriptionRef = useRef<HTMLTextAreaElement>(null)

  const doneColumn = columns.find((column) => column.is_done)
  const openColumn = columns.find((column) => !column.is_done)
  const isRunning = running?.task_id === task.id
  const childKind = normalizeKind(task.kind) === 'arc' ? 'task' : 'step'
  const lastSubtask = subtasks?.[subtasks.length - 1]
  const indentParent = subtasks?.find((subtask) => subtask.id === indentParentId)

  useEffect(() => {
    if (!editingTitle) setTitle(task.title)
  }, [task.title, editingTitle])

  useEffect(() => {
    if (!editingDescription) setDescription(task.description_md)
  }, [task.description_md, editingDescription])

  useEffect(() => {
    if (debouncedDescription === lastSaved.current) return
    lastSaved.current = debouncedDescription
    update.mutate({ id: task.id, description_md: debouncedDescription })
  }, [debouncedDescription, task.id, update])

  useEffect(() => {
    if (editingTitle) titleRef.current?.focus()
  }, [editingTitle])

  useEffect(() => {
    if (editingDescription) descriptionRef.current?.focus()
  }, [editingDescription])

  useEffect(() => {
    if (!focusField || focusNonce === undefined) return
    if (focusField === 'title') {
      setEditingTitle(true)
      return
    }
    const id = `inline-${focusField}-${task.id}`
    document.getElementById(id)?.focus()
  }, [focusField, focusNonce, task.id])

  function commitTitle() {
    setEditingTitle(false)
    const value = title.trim()
    if (!value) {
      setTitle(task.title)
      return
    }
    if (value !== task.title) update.mutate({ id: task.id, title: value })
  }

  function toggleSubtask(subtask: Task, checked: boolean) {
    const target = checked ? doneColumn : openColumn
    if (!target) return
    move.mutate({ id: subtask.id, column_id: target.id })
  }

  const saving = update.isPending || move.isPending
  const failed = update.isError || move.isError

  return (
    <div
      id={`task-expansion-${task.id}`}
      className="flex flex-col gap-3 rounded-md border bg-muted/20 p-3"
      onKeyDown={(event) => {
        if (event.key === 'Escape') {
          event.stopPropagation()
          onClose()
        }
      }}
    >
      <div className="flex flex-wrap items-center gap-2">
        {editingTitle ? (
          <Input
            ref={titleRef}
            value={title}
            aria-label="Task title"
            className="h-8 flex-1"
            onChange={(event) => setTitle(event.target.value)}
            onBlur={commitTitle}
            onKeyDown={(event) => {
              if (event.nativeEvent.isComposing) return
              if (event.key === 'Enter') {
                event.preventDefault()
                commitTitle()
              }
              if (event.key === 'Escape') {
                event.preventDefault()
                setTitle(task.title)
                setEditingTitle(false)
              }
            }}
          />
        ) : (
          <button
            type="button"
            className="min-w-0 flex-1 rounded-sm text-left text-sm font-medium outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring/50"
            onClick={() => setEditingTitle(true)}
          >
            {task.title}
          </button>
        )}
        {task.ref && (
          <span className="rounded border px-1 font-mono text-[0.65rem] text-muted-foreground">
            {task.ref}
          </span>
        )}
        {jira && (
          <a
            href={jira.jira_url}
            target="_blank"
            rel="noreferrer"
            className="rounded border px-1 font-mono text-[0.65rem] text-primary hover:underline"
          >
            {jira.jira_key}
          </a>
        )}
        {!!comments?.length && (
          <span className="text-xs text-muted-foreground">{comments.length} comments</span>
        )}
        {!!attachments?.length && (
          <span className="text-xs text-muted-foreground">{attachments.length} files</span>
        )}
        <Button
          size="icon-sm"
          variant={isRunning ? 'destructive' : 'outline'}
          title={isRunning ? 'Stop timer' : 'Start timer'}
          aria-label={isRunning ? 'Stop timer' : 'Start timer'}
          onClick={() => (isRunning ? stopTime.mutate(task.id) : startTime.mutate(task.id))}
        >
          {isRunning ? <Square className="fill-current" /> : <Play />}
        </Button>
        {aiStatus?.configured && (
          <Button
            size="sm"
            variant="outline"
            disabled={breakdown.isPending}
            onClick={() =>
              breakdown.mutate(undefined, {
                onError: (error) => toast.error(error.message),
              })
            }
          >
            {breakdown.isPending ? 'Generating…' : 'Generate steps'}
          </Button>
        )}
        <Button size="sm" variant="outline" onClick={onOpenDetails}>
          Open details
        </Button>
        <Button size="icon-sm" variant="ghost" aria-label="Collapse task" onClick={onClose}>
          <X />
        </Button>
      </div>

      <div className="flex flex-wrap items-center gap-2 text-xs">
        <Select
          items={columns.map((column) => ({ value: String(column.id), label: column.name }))}
          value={String(task.column_id)}
          onValueChange={(value) =>
            value && move.mutate({ id: task.id, column_id: Number(value) })
          }
        >
          <SelectTrigger id={`inline-status-${task.id}`} size="sm" aria-label="Status">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {columns.map((column) => (
              <SelectItem key={column.id} value={String(column.id)}>
                {column.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          items={[
            { value: 'low', label: 'low' },
            { value: 'medium', label: 'medium' },
            { value: 'high', label: 'high' },
          ]}
          value={task.priority}
          onValueChange={(value) => value && update.mutate({ id: task.id, priority: value })}
        >
          <SelectTrigger size="sm" aria-label="Priority">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="low">low</SelectItem>
            <SelectItem value="medium">medium</SelectItem>
            <SelectItem value="high">high</SelectItem>
          </SelectContent>
        </Select>

        <label className="flex items-center gap-1.5 text-muted-foreground">
          Due
          <Input
            id={`inline-due-${task.id}`}
            type="date"
            value={task.due_date ?? ''}
            aria-label="Due date"
            className="h-7 w-36 text-xs"
            onChange={(event) =>
              update.mutate({ id: task.id, due_date: event.target.value || null })
            }
          />
        </label>
        {[
          ['Today', 0],
          ['Tomorrow', 1],
          ['Next week', 7],
        ].map(([label, offset]) => (
          <Button
            key={label}
            size="xs"
            variant="ghost"
            onClick={() =>
              update.mutate({
                id: task.id,
                due_date: format(addDays(new Date(), Number(offset)), 'yyyy-MM-dd'),
              })
            }
          >
            {label}
          </Button>
        ))}
        {!task.due_date && <span className="text-muted-foreground">{formatDueDate(null)}</span>}
      </div>

      <div>
        <span className="mb-1 block text-xs font-medium text-muted-foreground">Description</span>
        {editingDescription ? (
          <textarea
            ref={descriptionRef}
            value={description}
            aria-label="Task description"
            rows={6}
            className="w-full rounded-md border bg-background p-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
            onChange={(event) => setDescription(event.target.value)}
            onBlur={() => {
              if (description !== lastSaved.current) {
                lastSaved.current = description
                update.mutate({ id: task.id, description_md: description })
              }
              setEditingDescription(false)
            }}
          />
        ) : description.trim() ? (
          <div
            role="button"
            tabIndex={0}
            aria-label="Edit description"
            className="max-h-40 cursor-text overflow-hidden rounded-md p-1 -m-1 outline-none hover:bg-muted/40 focus-visible:ring-2 focus-visible:ring-ring/50"
            onClick={(event) => {
              if ((event.target as HTMLElement).closest('a')) return
              setEditingDescription(true)
            }}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                setEditingDescription(true)
              }
            }}
          >
            <MarkdownRenderer source={description} />
          </div>
        ) : (
          <button
            type="button"
            className="w-full rounded-md p-1 -m-1 text-left text-xs text-muted-foreground outline-none hover:bg-muted/40 focus-visible:ring-2 focus-visible:ring-ring/50"
            onClick={() => setEditingDescription(true)}
          >
            Click to add a description…
          </button>
        )}
      </div>

      <LabelEditor task={task} inputId={`inline-labels-${task.id}`} />
      <LabelSuggestions task={task} />

      {(!!subtasks?.length || aiStatus?.configured) && (
        <div className="flex flex-col gap-1">
          <span className="text-xs font-medium text-muted-foreground">Subtasks</span>
          <SubtaskTree
            parentId={task.id}
            boardId={task.board_id}
            columns={columns}
            onToggle={toggleSubtask}
          />
          <Input
            value={newSubtask}
            aria-label="Add a subtask"
            placeholder={
              indentParent
                ? `Sub-step of ${indentParent.title}, Enter saves`
                : 'Add a subtask, Enter saves, Tab indents'
            }
            className={cn('h-7 text-xs', indentParent && 'ml-4 w-[calc(100%-1rem)]')}
            disabled={createSubtask.isPending}
            onChange={(event) => setNewSubtask(event.target.value)}
            onKeyDown={(event) => {
              if (event.nativeEvent.isComposing) return
              if (event.key === 'Tab') {
                if (event.shiftKey) {
                  if (indentParentId !== null) {
                    event.preventDefault()
                    setIndentParentId(null)
                  }
                  return
                }
                if (!lastSubtask) return
                event.preventDefault()
                setIndentParentId((current) => (current === null ? lastSubtask.id : null))
                return
              }
              if (event.key !== 'Enter') return
              event.preventDefault()
              const value = newSubtask.trim()
              if (!value) return
              createSubtask.mutate(
                {
                  board_id: task.board_id,
                  column_id: task.column_id,
                  title: value,
                  kind: indentParentId === null ? childKind : 'step',
                  parent_task_id: indentParentId ?? undefined,
                },
                {
                  onSuccess: () => {
                    setNewSubtask('')
                    setIndentParentId(null)
                  },
                  onError: (error) => toast.error(error.message),
                },
              )
            }}
          />
        </div>
      )}

      <div aria-live="polite" className="min-h-4 text-xs text-muted-foreground">
        {failed ? (
          <span role="alert" className="text-destructive">
            Could not save.{' '}
            <button
              type="button"
              className="underline"
              onClick={() =>
                update.mutate(update.variables ?? { id: task.id, title: task.title })
              }
            >
              Retry
            </button>
          </span>
        ) : saving ? (
          'Saving…'
        ) : (
          <span className="inline-flex items-center gap-1">
            <Check className="size-3" /> Saved
          </span>
        )}
      </div>
    </div>
  )
}
