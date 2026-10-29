import { Check, Maximize2, MessageSquare, Paperclip, Play, Sparkles, Square, X } from 'lucide-react'
import { toast } from 'sonner'
import { addDays, format } from 'date-fns'
import { useEffect, useRef, useState } from 'react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useDebounced } from '@/lib/useDebounced'
import { cn } from '@/lib/utils'
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

const SECTION_HEADING = 'text-xs font-semibold tracking-wide text-muted-foreground uppercase'
const FIELD_LABEL = 'text-xs font-medium text-muted-foreground'

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
      className="@container/editor rounded-lg border bg-card text-sm whitespace-normal text-card-foreground"
      onKeyDown={(event) => {
        if (event.key === 'Escape') {
          event.stopPropagation()
          onClose()
        }
      }}
    >
      <div className="flex flex-col gap-2 p-3 @2xl/editor:px-4 @2xl/editor:pt-4">
        <div className="flex items-center gap-2">
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
            {task.ref && (
              <Badge variant="outline" className="font-mono text-muted-foreground">
                {task.ref}
              </Badge>
            )}
            {jira && (
              <Badge
                variant="outline"
                className="h-6 font-mono text-primary outline-none hover:underline"
                render={<a href={jira.jira_url} target="_blank" rel="noreferrer" />}
              >
                {jira.jira_key}
              </Badge>
            )}
            {!!comments?.length && (
              <span className="inline-flex items-center gap-1">
                <MessageSquare className="size-3.5" aria-hidden />
                {comments.length}
                <span className="sr-only">comments</span>
              </span>
            )}
            {!!attachments?.length && (
              <span className="inline-flex items-center gap-1">
                <Paperclip className="size-3.5" aria-hidden />
                {attachments.length}
                <span className="sr-only">files</span>
              </span>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-1">
            <Button
              size="icon-sm"
              variant={isRunning ? 'destructive' : 'outline'}
              title={isRunning ? 'Stop timer' : 'Start timer'}
              aria-label={isRunning ? 'Stop timer' : 'Start timer'}
              onClick={() => (isRunning ? stopTime.mutate(task.id) : startTime.mutate(task.id))}
            >
              {isRunning ? <Square className="fill-current" /> : <Play />}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              title="Open details"
              aria-label="Open details"
              onClick={onOpenDetails}
            >
              <Maximize2 />
              <span className="hidden @md/editor:inline">Open details</span>
            </Button>
            <Button size="icon-sm" variant="ghost" title="Collapse" aria-label="Collapse task" onClick={onClose}>
              <X />
            </Button>
          </div>
        </div>

        {editingTitle ? (
          <Input
            ref={titleRef}
            value={title}
            aria-label="Task title"
            className="h-auto py-1 text-base font-semibold md:text-base @2xl/editor:text-lg"
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
          <h2 className="text-base leading-snug font-semibold @2xl/editor:text-lg">
            <button
              type="button"
              title="Rename"
              className="-mx-1 w-full rounded-md px-1 py-0.5 text-left wrap-break-word outline-none transition-colors hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50 active:bg-muted"
              onClick={() => setEditingTitle(true)}
            >
              {task.title}
            </button>
          </h2>
        )}
      </div>

      <div className="grid gap-4 px-3 pb-3 @2xl/editor:grid-cols-[minmax(0,18rem)_minmax(0,1fr)] @2xl/editor:gap-6 @2xl/editor:px-4 @2xl/editor:pb-4">
        <div className="grid grid-cols-2 content-start gap-x-2 gap-y-3 self-start rounded-md bg-muted/50 p-3">
          <div className="flex min-w-0 flex-col gap-1">
            <label htmlFor={`inline-status-${task.id}`} className={FIELD_LABEL}>
              Status
            </label>
            <Select
              items={columns.map((column) => ({ value: String(column.id), label: column.name }))}
              value={String(task.column_id)}
              onValueChange={(value) =>
                value && move.mutate({ id: task.id, column_id: Number(value) })
              }
            >
              <SelectTrigger
                id={`inline-status-${task.id}`}
                aria-label="Status"
                className="w-full bg-background"
              >
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
          </div>

          <div className="flex min-w-0 flex-col gap-1">
            <label htmlFor={`inline-priority-${task.id}`} className={FIELD_LABEL}>
              Priority
            </label>
            <Select
              items={[
                { value: 'low', label: 'low' },
                { value: 'medium', label: 'medium' },
                { value: 'high', label: 'high' },
              ]}
              value={task.priority}
              onValueChange={(value) => value && update.mutate({ id: task.id, priority: value })}
            >
              <SelectTrigger
                id={`inline-priority-${task.id}`}
                aria-label="Priority"
                className="w-full bg-background capitalize"
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="low" className="capitalize">low</SelectItem>
                <SelectItem value="medium" className="capitalize">medium</SelectItem>
                <SelectItem value="high" className="capitalize">high</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="col-span-2 flex min-w-0 flex-col gap-1">
            <div className="flex items-baseline justify-between gap-2">
              <label htmlFor={`inline-due-${task.id}`} className={FIELD_LABEL}>
                Due date
              </label>
              {!task.due_date && <span className="text-xs text-muted-foreground">Not set</span>}
            </div>
            <Input
              id={`inline-due-${task.id}`}
              type="date"
              value={task.due_date ?? ''}
              aria-label="Due date"
              className="bg-background"
              onChange={(event) =>
                update.mutate({ id: task.id, due_date: event.target.value || null })
              }
            />
            <div className="flex flex-wrap gap-1 pt-1">
              {[
                ['Today', 0],
                ['Tomorrow', 1],
                ['Next week', 7],
              ].map(([label, offset]) => (
                <Button
                  key={label}
                  size="xs"
                  variant="outline"
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
            </div>
          </div>
        </div>

        <div className="flex min-w-0 flex-col gap-5">
          <section className="flex flex-col gap-2">
            <h3 className={SECTION_HEADING}>Description</h3>
            {editingDescription ? (
              <Textarea
                ref={descriptionRef}
                value={description}
                aria-label="Task description"
                rows={6}
                className="min-h-32 bg-background"
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
                className="-mx-2 max-h-40 cursor-text overflow-hidden rounded-md px-2 py-1 outline-none transition-colors hover:bg-muted/60 focus-visible:ring-3 focus-visible:ring-ring/50 active:bg-muted"
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
                className="w-full rounded-md border border-dashed px-3 py-2 text-left text-muted-foreground outline-none transition-colors hover:bg-muted/60 hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 active:bg-muted"
                onClick={() => setEditingDescription(true)}
              >
                Add a description…
              </button>
            )}
          </section>

          <section className="flex flex-col gap-2">
            <h3 className={SECTION_HEADING}>Labels</h3>
            <LabelEditor task={task} inputId={`inline-labels-${task.id}`} />
            <LabelSuggestions task={task} />
          </section>

          {(!!subtasks?.length || aiStatus?.configured) && (
            <section className="flex flex-col gap-2">
              <div className="flex min-h-6 items-center justify-between gap-2">
                <h3 className={SECTION_HEADING}>Subtasks</h3>
                {aiStatus?.configured && (
                  <Button
                    size="xs"
                    variant="outline"
                    disabled={breakdown.isPending}
                    onClick={() =>
                      breakdown.mutate(undefined, {
                        onError: (error) => toast.error(error.message),
                      })
                    }
                  >
                    <Sparkles />
                    {breakdown.isPending ? 'Generating…' : 'Generate steps'}
                  </Button>
                )}
              </div>
              <SubtaskTree
                parentId={task.id}
                boardId={task.board_id}
                columns={columns}
                onToggle={toggleSubtask}
              />
              <Input
                value={newSubtask}
                aria-label="Add a subtask"
                aria-describedby={`inline-subtask-hint-${task.id}`}
                placeholder={indentParent ? `Sub-step of ${indentParent.title}` : 'Add a subtask'}
                className={cn('bg-background', indentParent && 'ml-4 w-[calc(100%-1rem)]')}
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
              <p id={`inline-subtask-hint-${task.id}`} className="text-xs text-muted-foreground">
                {indentParent ? 'Enter saves, Shift+Tab outdents' : 'Enter saves, Tab indents'}
              </p>
            </section>
          )}
        </div>
      </div>

      <div
        aria-live="polite"
        className="flex min-h-9 items-center justify-end border-t px-3 py-2 text-xs text-muted-foreground @2xl/editor:px-4"
      >
        {failed ? (
          <span role="alert" className="text-destructive">
            Could not save.{' '}
            <button
              type="button"
              className="rounded-sm font-medium underline underline-offset-2 outline-none hover:no-underline focus-visible:ring-2 focus-visible:ring-ring/50"
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
            <Check className="size-3.5" aria-hidden /> Saved
          </span>
        )}
      </div>
    </div>
  )
}
