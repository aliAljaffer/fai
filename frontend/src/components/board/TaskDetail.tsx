import MDEditor from '@uiw/react-md-editor'
import { Check, Play, Sparkles, Square, X } from 'lucide-react'
import { useTheme } from 'next-themes'
import { useMemo, useRef, useState } from 'react'
import { useHotkeys } from 'react-hotkeys-hook'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Sheet, SheetContent, SheetFooter, SheetHeader } from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { AttachmentsPanel } from '../AttachmentsPanel'
import { RefPicker } from '../refs/RefPicker'
import { useRefPicker } from '../refs/useRefPicker'
import { AIPanel } from './AIPanel'
import { CommentsPanel } from './CommentsPanel'
import { JiraPanel } from './JiraPanel'
import { LabelEditor } from './LabelEditor'
import { SubtasksPanel } from './SubtasksPanel'
import { TaskTypeIcon } from '../TaskTypeIcon'
import { TASK_KIND_LABELS, normalizeKind, taskStatus } from '../../lib/taskType'
import { MarkdownRenderer } from '../MarkdownRenderer'
import {
  useAddTaskTag,
  useAIStatus,
  useColumns,
  useCreateTag,
  useMoveTask,
  useRunningTime,
  useSetTaskType,
  useStartTime,
  useStopTime,
  useSuggestLabels,
  useTags,
  useTaskReferences,
  useTaskTimeEntries,
} from '../../api/hooks'
import { formatDuration } from '../../lib/format'
import type { Task, TaskBacklink } from '../../api/types'

export function TaskDetail({
  task,
  onClose,
  onSave,
  onDelete,
  embedded = false,
}: {
  task: Task
  onClose: () => void
  onSave: (fields: { title: string; description_md: string; priority: string }) => void
  onDelete: () => void
  embedded?: boolean
}) {
  const [title, setTitle] = useState(task.title)
  const [description, setDescription] = useState(task.description_md)
  const [priority, setPriority] = useState(task.priority)
  const titleRef = useRef<HTMLInputElement>(null)
  const { resolvedTheme } = useTheme()
  const refs = useRefPicker({ value: description, onChange: setDescription })

  useHotkeys('e', () => titleRef.current?.focus(), { preventDefault: true })

  const { data: running } = useRunningTime()
  const { data: entries } = useTaskTimeEntries(task.id)
  const { data: columns } = useColumns(task.board_id)
  const columnItems = useMemo(
    () => (columns ?? []).map((column) => ({ value: String(column.id), label: column.name })),
    [columns],
  )
  const kindItems = useMemo(
    () => (['arc', 'task', 'step'] as const).map((value) => ({ value, label: TASK_KIND_LABELS[value] })),
    [],
  )
  const setType = useSetTaskType(task.id, task.board_id)
  const moveTask = useMoveTask(task.board_id)
  const kind = normalizeKind(task.kind)
  const canBeStep = task.parent_task_id !== null || kind === 'step'
  const startTime = useStartTime()
  const stopTime = useStopTime()
  const isRunning = running?.task_id === task.id
  const totalSeconds = (entries ?? []).reduce((sum, e) => sum + (e.duration_seconds ?? 0), 0)
  const sessionSeconds = isRunning
    ? Math.max(0, Math.floor((Date.now() - new Date(running.started_at).getTime()) / 1000))
    : 0

  const content = (
    <>
        <SheetHeader className="px-0">
          <div className="flex items-center gap-2">
            <input
              ref={titleRef}
              aria-label="Task title"
              className="flex-1 border-b bg-transparent pb-2 text-lg font-medium outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
            <TaskTypeIcon kind={task.kind} status={taskStatus(task, columns)} size={20} />
            {task.ref && (
              <Badge variant="outline" className="shrink-0 font-mono text-[0.65rem] text-muted-foreground">
                {task.ref}
              </Badge>
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
          </div>
          <p className="text-xs text-muted-foreground">
            {formatDuration(totalSeconds)} tracked
            {isRunning && ` · ${formatDuration(sessionSeconds)} this session`}
          </p>
        </SheetHeader>

        <Tabs defaultValue="details">
          <TabsList>
            <TabsTrigger value="details">Details</TabsTrigger>
            <TabsTrigger value="comments">Comments</TabsTrigger>
            <TabsTrigger value="subtasks">{kind === 'arc' ? 'Tasks' : 'Steps'}</TabsTrigger>
            <TabsTrigger value="time">Time</TabsTrigger>
            <TabsTrigger value="files">Files</TabsTrigger>
            <TabsTrigger value="ai">AI</TabsTrigger>
          </TabsList>

          <TabsContent value="details" className="flex flex-col gap-4">
            <MDEditor
              value={description}
              onChange={(value, event) => {
                setDescription(value ?? '')
                refs.handleChange(value ?? '', event?.target ?? null)
              }}
              textareaProps={{
                'aria-label': 'Task description',
                onKeyDown: refs.handleKeyDown,
                onBlur: refs.handleBlur,
                onSelect: refs.sync,
              }}
              height={220}
              preview="edit"
              data-color-mode={resolvedTheme === "light" ? "light" : "dark"}
            />
            <RefPicker picker={refs} />
            {description && (
              <div className="rounded-lg border p-3">
                <MarkdownRenderer source={description} />
              </div>
            )}
            <div className="flex flex-col gap-1.5">
              <span className="text-xs font-medium text-muted-foreground">Labels</span>
              <LabelEditor task={task} />
              <LabelSuggestions task={task} />
            </div>
            <div className="flex items-center gap-2">
              <Select
                value={kind}
                items={kindItems}
                onValueChange={(value) => {
                  if (!value || value === kind) return
                  const next = value as 'arc' | 'task' | 'step'
                  setType.mutate(
                    {
                      kind: next,
                      parent_task_id: next === 'step' ? task.parent_task_id : null,
                    },
                    { onError: (error) => toast.error(error.message) },
                  )
                }}
              >
                <SelectTrigger className="w-32" aria-label="Task type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="arc">{TASK_KIND_LABELS.arc}</SelectItem>
                  <SelectItem value="task">{TASK_KIND_LABELS.task}</SelectItem>
                  <SelectItem value="step" disabled={!canBeStep}>
                    {TASK_KIND_LABELS.step}
                  </SelectItem>
                </SelectContent>
              </Select>
              <Select value={priority} onValueChange={(value) => value && setPriority(value)}>
                <SelectTrigger className="w-32" aria-label="Priority">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="low">low</SelectItem>
                  <SelectItem value="medium">medium</SelectItem>
                  <SelectItem value="high">high</SelectItem>
                </SelectContent>
              </Select>
              <Select
                value={String(task.column_id)}
                items={columnItems}
                onValueChange={(value) => {
                  if (!value || Number(value) === task.column_id) return
                  moveTask.mutate(
                    { id: task.id, column_id: Number(value) },
                    { onError: (error) => toast.error(error.message) },
                  )
                }}
                disabled={moveTask.isPending}
              >
                <SelectTrigger className="w-36" aria-label="Status column">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {columns?.map((column) => (
                    <SelectItem key={column.id} value={String(column.id)}>
                      {column.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <JiraPanel taskId={task.id} />
          </TabsContent>

          <TabsContent value="comments">
            <CommentsPanel taskId={task.id} />
          </TabsContent>

          <TabsContent value="subtasks">
            <SubtasksPanel task={task} />
          </TabsContent>

          <TabsContent value="time" className="flex items-center gap-3">
            <Button
              size="sm"
              variant={isRunning ? 'destructive' : 'outline'}
              onClick={() => (isRunning ? stopTime.mutate(task.id) : startTime.mutate(task.id))}
            >
              {isRunning ? 'Stop timer' : 'Start timer'}
            </Button>
            <span className="text-sm text-muted-foreground">{formatDuration(totalSeconds)} tracked</span>
          </TabsContent>

          <TabsContent value="files">
            <AttachmentsPanel taskId={task.id} />
          </TabsContent>

          <TabsContent value="ai">
            <AIPanel task={task} />
          </TabsContent>
        </Tabs>

        <TaskMentions taskId={task.id} />

        <SheetFooter className="mt-auto flex-row justify-start px-0">
          <Button
            onClick={() => {
              onSave({ title, description_md: description, priority })
              onClose()
            }}
          >
            Save
          </Button>
          <Button variant="destructive" onClick={onDelete}>
            Delete
          </Button>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
        </SheetFooter>
    </>
  )

  if (embedded) {
    return <div className="mx-auto flex w-full max-w-2xl flex-col gap-4 p-2">{content}</div>
  }

  return (
    <Sheet open modal={false} onOpenChange={(open) => !open && onClose()}>
      <SheetContent overlay={false} className="w-full gap-4 overflow-y-auto p-4 sm:max-w-xl">
        {content}
      </SheetContent>
    </Sheet>
  )
}

export function LabelSuggestions({ task }: { task: Task }) {
  const { data: status } = useAIStatus()
  const { data: allTags } = useTags()
  const suggest = useSuggestLabels(task.id)
  const createTag = useCreateTag()
  const addTag = useAddTaskTag(task.id, task.board_id)
  const [suggested, setSuggested] = useState<string[]>([])
  const [done, setDone] = useState<string[]>([])

  if (!status?.configured) return null

  const attached = new Set((task.tags ?? []).map((tag) => tag.name.toLowerCase()))
  const visible = suggested.filter((label) => !done.includes(label) && !attached.has(label))

  function dismiss(label: string) {
    setDone((current) => [...current, label])
  }

  async function accept(label: string) {
    dismiss(label)
    try {
      const existing = (allTags ?? []).find((tag) => tag.name.toLowerCase() === label)
      const tag = existing ?? (await createTag.mutateAsync({ name: label }))
      await addTag.mutateAsync(tag.id)
    } catch (error) {
      toast.error((error as Error).message)
    }
  }

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <Button
        size="xs"
        variant="outline"
        disabled={suggest.isPending}
        onClick={() =>
          suggest.mutate(undefined, {
            onSuccess: (result) => {
              setSuggested(result.labels)
              setDone([])
            },
            onError: (error) => toast.error(error.message),
          })
        }
      >
        <Sparkles />
        {suggest.isPending ? 'Thinking…' : 'Suggest labels'}
      </Button>
      {visible.map((label) => (
        <span
          key={label}
          className="inline-flex items-center gap-0.5 rounded-full border border-dashed py-0.5 pr-0.5 pl-2 text-xs text-muted-foreground"
        >
          {label}
          <button
            className="rounded-full p-0.5 hover:bg-muted hover:text-foreground"
            aria-label={`Accept ${label}`}
            title="Accept"
            onClick={() => void accept(label)}
          >
            <Check className="size-3" />
          </button>
          <button
            className="rounded-full p-0.5 hover:bg-muted hover:text-foreground"
            aria-label={`Reject ${label}`}
            title="Reject"
            onClick={() => dismiss(label)}
          >
            <X className="size-3" />
          </button>
        </span>
      ))}
    </div>
  )
}

const MENTION_KINDS = [
  { kind: 'task', label: 'Tasks' },
  { kind: 'note', label: 'Notes' },
  { kind: 'reminder', label: 'Reminders' },
  { kind: 'meeting', label: 'Meetings' },
  { kind: 'chat', label: 'Chat' },
] as const

function TaskMentions({ taskId }: { taskId: number }) {
  const { data } = useTaskReferences(taskId)
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const mentions = data ?? []

  if (!mentions.length) return null

  function open(item: TaskBacklink) {
    if (item.kind === 'task') {
      const next = new URLSearchParams(params)
      next.set('task', String(item.id))
      setParams(next, { replace: true })
      return
    }
    if (item.kind === 'note') {
      navigate(`/notes/${item.id}`)
      return
    }
    if (item.kind === 'meeting') {
      navigate('/calendar')
      return
    }
    if (item.kind === 'chat') {
      navigate('/chat')
      return
    }
    navigate('/board')
  }

  return (
    <div className="flex flex-col gap-2 border-t pt-3">
      <span className="text-xs font-medium text-muted-foreground">Mentioned in</span>
      {MENTION_KINDS.map((group) => {
        const items = mentions.filter((item) => item.kind === group.kind)
        if (!items.length) return null
        return (
          <div key={group.kind} className="flex flex-col gap-1">
            <span className="text-[0.65rem] tracking-wide text-muted-foreground uppercase">
              {group.label}
            </span>
            <ul className="flex flex-col gap-1">
              {items.map((item) => (
                <li key={`${item.kind}-${item.id}`}>
                  <button
                    type="button"
                    className="text-left text-sm text-primary underline"
                    onClick={() => open(item)}
                  >
                    {item.title}
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )
      })}
    </div>
  )
}
