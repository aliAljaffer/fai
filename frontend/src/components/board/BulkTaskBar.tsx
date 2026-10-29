import { useMemo, useState } from 'react'
import { Check, Sparkles, Trash2, Upload, X } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  useAIStatus,
  useBulkTaskActions,
  useCreateTag,
  useExportJiraTasks,
  useRestoreTasks,
  useSuggestLabelsBatch,
  type BulkTaskAction,
} from '../../api/hooks'
import { showUndo } from '../../lib/undo'
import type { BoardColumn, Tag, Task } from '../../api/types'

// Matches the backend cap on one label suggestion request.
const labelBatchSize = 20

export function BulkTaskBar({
  tasks,
  columns,
  tags,
  onClear,
}: {
  tasks: Task[]
  columns: BoardColumn[]
  tags: Tag[]
  onClear: () => void
}) {
  const bulk = useBulkTaskActions()
  const restore = useRestoreTasks(undefined)
  const { data: aiStatus } = useAIStatus()
  const createTag = useCreateTag()
  const suggestBatch = useSuggestLabelsBatch()
  const exportJira = useExportJiraTasks()
  const busy = bulk.isPending || restore.isPending
  const count = tasks.length
  const [suggestions, setSuggestions] = useState<Record<number, string[]>>({})
  const [suggesting, setSuggesting] = useState(false)

  // Label suggestions live per task, so a label only applies to the selected
  // tasks the AI proposed it for, not to every task in the selection.
  const suggestedLabels = useMemo(() => {
    const groups = new Map<string, number[]>()
    for (const task of tasks) {
      const attached = new Set((task.tags ?? []).map((tag) => tag.name.toLowerCase()))
      for (const label of suggestions[task.id] ?? []) {
        if (attached.has(label)) continue
        groups.set(label, [...(groups.get(label) ?? []), task.id])
      }
    }
    return [...groups].map(([label, taskIds]) => ({ label, taskIds }))
  }, [tasks, suggestions])

  const doneColumnIds = new Set(columns.filter((column) => column.is_done).map((column) => column.id))
  const doneColumn = columns.find((column) => column.is_done)
  const openColumn = columns.find((column) => !column.is_done)
  const anyDone = tasks.some(
    (task) => Boolean(task.completed_at) || doneColumnIds.has(task.column_id),
  )
  const allDone = tasks.every(
    (task) => Boolean(task.completed_at) || doneColumnIds.has(task.column_id),
  )

  const common = <T,>(values: T[]) => (new Set(values).size === 1 ? values[0] : undefined)
  const commonColumn = common(tasks.map((task) => String(task.column_id)))
  const commonPriority = common(tasks.map((task) => task.priority))
  const commonDue = common(tasks.map((task) => task.due_date ?? '')) ?? ''

  async function run(action: BulkTaskAction) {
    try {
      await bulk.mutateAsync({ tasks, action })
      return true
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Bulk action failed')
      return false
    }
  }

  async function suggestLabels() {
    setSuggesting(true)
    try {
      const ids = tasks.map((task) => task.id)
      const merged: Record<number, string[]> = {}
      for (let i = 0; i < ids.length; i += labelBatchSize) {
        const { results } = await suggestBatch.mutateAsync(ids.slice(i, i + labelBatchSize))
        for (const result of results) merged[result.task_id] = result.labels
      }
      setSuggestions(merged)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not suggest labels')
    } finally {
      setSuggesting(false)
    }
  }

  async function exportToJira() {
    try {
      const response = await exportJira.mutateAsync(tasks.map((task) => task.id))
      const counts = [
        `${response.created} created`,
        response.skipped ? `${response.skipped} skipped` : '',
        response.failed ? `${response.failed} failed` : '',
      ]
        .filter(Boolean)
        .join(', ')
      if (response.failed) toast.error(`Jira export: ${counts}`)
      else toast.success(`Jira export: ${counts}`)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Jira export failed')
    }
  }

  function dismissLabel(label: string) {
    setSuggestions((current) =>
      Object.fromEntries(
        Object.entries(current)
          .map(([id, labels]) => [id, labels.filter((item) => item !== label)] as const)
          .filter(([, labels]) => labels.length),
      ),
    )
  }

  async function acceptLabel(label: string, taskIds: number[]) {
    dismissLabel(label)
    try {
      const existing = tags.find((tag) => tag.name.toLowerCase() === label)
      const tag = existing ?? (await createTag.mutateAsync({ name: label }))
      await bulk.mutateAsync({
        tasks: tasks.filter((task) => taskIds.includes(task.id)),
        action: { type: 'addTag', tagId: tag.id },
      })
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not add label')
    }
  }

  async function remove() {
    const ids = tasks.map((task) => task.id)
    if (!(await run({ type: 'delete' }))) return
    onClear()
    showUndo(`Deleted ${count} ${count === 1 ? 'task' : 'tasks'}`, () =>
      restore.mutateAsync(ids).then(() => undefined),
    )
  }

  return (
    <>
      <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-muted/40 px-3 py-2">
        <span className="text-sm font-medium tabular-nums">{count} selected</span>
        <div className="flex flex-wrap items-center gap-2">
          <Select
            value={commonColumn ?? null}
            onValueChange={(value) => {
              if (value) void run({ type: 'move', columnId: Number(value) })
            }}
          >
            <SelectTrigger size="sm" className="w-36" aria-label="Move to column" disabled={busy}>
              <SelectValue placeholder="Move to…" />
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
            value={commonPriority ?? null}
            onValueChange={(value) => {
              if (value) void run({ type: 'priority', priority: value })
            }}
          >
            <SelectTrigger size="sm" className="w-32" aria-label="Set priority" disabled={busy}>
              <SelectValue placeholder="Priority" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="low">low</SelectItem>
              <SelectItem value="medium">medium</SelectItem>
              <SelectItem value="high">high</SelectItem>
            </SelectContent>
          </Select>

          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button size="sm" variant="outline" disabled={busy || !tags.length}>
                  Labels
                </Button>
              }
            />
            <DropdownMenuContent>
              {tags.length ? (
                tags.map((tag) => {
                  const attached = tasks.every((task) =>
                    (task.tags ?? []).some((item) => item.id === tag.id),
                  )
                  return (
                    <DropdownMenuCheckboxItem
                      key={tag.id}
                      checked={attached}
                      disabled={busy}
                      onCheckedChange={(checked) => {
                        void run(
                          checked
                            ? { type: 'addTag', tagId: tag.id }
                            : { type: 'removeTag', tagId: tag.id },
                        )
                      }}
                    >
                      <span
                        className="size-1.5 rounded-full"
                        style={{ backgroundColor: tag.color || undefined }}
                      />
                      {tag.name}
                    </DropdownMenuCheckboxItem>
                  )
                })
              ) : (
                <DropdownMenuGroup>
                  <DropdownMenuLabel>No labels yet</DropdownMenuLabel>
                </DropdownMenuGroup>
              )}
            </DropdownMenuContent>
          </DropdownMenu>

          {aiStatus?.configured && (
            <Button
              size="sm"
              variant="outline"
              disabled={busy || suggesting}
              onClick={() => void suggestLabels()}
            >
              <Sparkles />
              {suggesting ? 'Thinking…' : 'Suggest labels'}
            </Button>
          )}

          <Input
            type="date"
            aria-label="Set due date"
            className="h-7 w-36 text-xs"
            value={commonDue}
            disabled={busy}
            onChange={(event) => {
              const dueDate = event.target.value || null
              void run({ type: 'dueDate', dueDate })
            }}
          />
          {tasks.some((task) => task.due_date) && (
            <Button
              size="sm"
              variant="ghost"
              disabled={busy}
              onClick={() => void run({ type: 'dueDate', dueDate: null })}
            >
              Clear due
            </Button>
          )}

          {!allDone && (
            <Button
              size="sm"
              variant="outline"
              disabled={busy || !doneColumn}
              title={doneColumn ? undefined : 'Add a done column first'}
              onClick={() => {
                if (doneColumn) void run({ type: 'move', columnId: doneColumn.id })
              }}
            >
              Mark done
            </Button>
          )}
          {anyDone && (
            <Button
              size="sm"
              variant="outline"
              disabled={busy || !openColumn}
              title={openColumn ? undefined : 'Add an open column first'}
              onClick={() => {
                if (openColumn) void run({ type: 'move', columnId: openColumn.id })
              }}
            >
              Reopen
            </Button>
          )}

          <Button
            size="sm"
            variant="outline"
            disabled={busy || exportJira.isPending}
            onClick={() => void exportToJira()}
          >
            <Upload />
            {exportJira.isPending ? 'Exporting…' : 'Export to Jira'}
          </Button>

          <Button size="sm" variant="destructive" disabled={busy} onClick={() => void remove()}>
            <Trash2 />
            Delete
          </Button>
          <Button size="sm" variant="ghost" disabled={busy} onClick={onClear}>
            <X />
            Clear
          </Button>

          {suggestedLabels.map(({ label, taskIds }) => (
            <span
              key={label}
              className="inline-flex items-center gap-0.5 rounded-full border border-dashed py-0.5 pr-0.5 pl-2 text-xs text-muted-foreground"
            >
              {label}
              {taskIds.length < count && <span className="tabular-nums">×{taskIds.length}</span>}
              <button
                type="button"
                className="rounded-full p-0.5 hover:bg-muted hover:text-foreground"
                aria-label={`Accept ${label}`}
                title="Accept"
                onClick={() => void acceptLabel(label, taskIds)}
              >
                <Check className="size-3" />
              </button>
              <button
                type="button"
                className="rounded-full p-0.5 hover:bg-muted hover:text-foreground"
                aria-label={`Reject ${label}`}
                title="Reject"
                onClick={() => dismissLabel(label)}
              >
                <X className="size-3" />
              </button>
            </span>
          ))}
        </div>
      </div>

    </>
  )
}
