import { Trash2, X } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import { useBulkTaskActions, type BulkTaskAction } from '../../api/hooks'
import type { BoardColumn, Tag, Task } from '../../api/types'

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
  const [confirmDelete, setConfirmDelete] = useState(false)
  const busy = bulk.isPending
  const count = tasks.length

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

  async function run(action: BulkTaskAction, message: string) {
    try {
      await bulk.mutateAsync({ tasks, action })
      toast.success(message)
      return true
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Bulk action failed')
      return false
    }
  }

  async function remove() {
    if (await run({ type: 'delete' }, `Deleted ${count} ${count === 1 ? 'task' : 'tasks'}`)) {
      setConfirmDelete(false)
      onClear()
    }
  }

  return (
    <>
      <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-muted/40 px-3 py-2">
        <span className="text-sm font-medium tabular-nums">{count} selected</span>
        <div className="flex flex-wrap items-center gap-2">
          <Select
            value={commonColumn ?? null}
            onValueChange={(value) => {
              if (value) void run({ type: 'move', columnId: Number(value) }, `Moved ${count}`)
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
              if (value) void run({ type: 'priority', priority: value }, `Set priority to ${value}`)
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
                          checked ? `Added label ${tag.name}` : `Removed label ${tag.name}`,
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

          <Input
            type="date"
            aria-label="Set due date"
            className="h-7 w-36 text-xs"
            value={commonDue}
            disabled={busy}
            onChange={(event) => {
              const dueDate = event.target.value || null
              void run({ type: 'dueDate', dueDate }, dueDate ? `Due date set to ${dueDate}` : 'Due date cleared')
            }}
          />
          {tasks.some((task) => task.due_date) && (
            <Button
              size="sm"
              variant="ghost"
              disabled={busy}
              onClick={() => void run({ type: 'dueDate', dueDate: null }, 'Due date cleared')}
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
                if (doneColumn) void run({ type: 'move', columnId: doneColumn.id }, `Marked ${count} done`)
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
                if (openColumn) void run({ type: 'move', columnId: openColumn.id }, `Reopened ${count}`)
              }}
            >
              Reopen
            </Button>
          )}

          <Button size="sm" variant="destructive" disabled={busy} onClick={() => setConfirmDelete(true)}>
            <Trash2 />
            Delete
          </Button>
          <Button size="sm" variant="ghost" disabled={busy} onClick={onClear}>
            <X />
            Clear
          </Button>
        </div>
      </div>

      <Dialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <DialogContent className="rounded-lg">
          <DialogHeader>
            <DialogTitle>Delete {count === 1 ? 'task' : `${count} tasks`}?</DialogTitle>
            <DialogDescription>
              Deleted tasks are removed from the board and list. This cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setConfirmDelete(false)} disabled={busy}>
              Cancel
            </Button>
            <Button variant="destructive" disabled={busy} onClick={() => void remove()}>
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
