import { Fragment, useMemo, useState, type ReactNode } from 'react'
import { useSearchParams } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useBoards, useColumns, useJiraLink, useTags, useTasks } from '../api/hooks'
import { BulkTaskBar } from '../components/board/BulkTaskBar'
import { useDebounced } from '../lib/useDebounced'
import { labelChipStyle } from '../lib/palette'
import { columnColor, useColumnColors } from '../store/columnColors'
import { useUIStore } from '../store/ui'
import { useWorkspaceStore } from '../store/workspace'
import { cn } from '@/lib/utils'
import type { Task } from '../api/types'

type SortKey = 'title' | 'priority' | 'due_date'
const priorityRank: Record<string, number> = { high: 0, medium: 1, low: 2 }

function compare(a: Task, b: Task, key: SortKey) {
  if (key === 'priority') return (priorityRank[a.priority] ?? 3) - (priorityRank[b.priority] ?? 3)
  if (key === 'due_date') {
    if (!a.due_date) return 1
    if (!b.due_date) return -1
    return a.due_date.localeCompare(b.due_date)
  }
  return a.title.localeCompare(b.title)
}

function JiraKey({ taskId }: { taskId: number }) {
  const { data: link } = useJiraLink(taskId)
  if (!link) return null
  return (
    <a
      href={link.jira_url}
      target="_blank"
      rel="noreferrer"
      className="text-xs font-medium text-primary hover:underline"
      onClick={(event) => event.stopPropagation()}
    >
      {link.jira_key}
    </a>
  )
}

export function List() {
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: boards } = useBoards(workspaceId)
  const { selectedBoardId } = useUIStore()
  const boardId =
    boards?.find((board) => board.id === selectedBoardId)?.id ?? boards?.[0]?.id

  const { data: columns } = useColumns(boardId)
  const { data: tags } = useTags()
  const [q, setQ] = useState('')
  const [columnId, setColumnId] = useState('all')
  const [tag, setTag] = useState('all')
  const debouncedQ = useDebounced(q)

  const { data: tasks } = useTasks(boardId, {
    q: debouncedQ || undefined,
    columnId: columnId === 'all' ? undefined : Number(columnId),
    tag: tag === 'all' ? undefined : tag,
  })

  const [, setSearchParams] = useSearchParams()
  const [sort, setSort] = useState<{ key: SortKey; dir: 1 | -1 }>({ key: 'title', dir: 1 })
  const [expanded, setExpanded] = useState<Set<number>>(() => new Set())
  const [selected, setSelected] = useState<Set<number>>(() => new Set())
  const [lastSelected, setLastSelected] = useState<number | null>(null)
  const [hideDone, setHideDone] = useState(false)

  const columnById = useMemo(
    () => new Map((columns ?? []).map((column) => [column.id, column])),
    [columns],
  )
  const { colors } = useColumnColors()

  const doneColumnIds = useMemo(
    () => new Set((columns ?? []).filter((column) => column.is_done).map((column) => column.id)),
    [columns],
  )

  const taskIds = useMemo(() => new Set((tasks ?? []).map((task) => task.id)), [tasks])

  // Group subtasks under their parent; a child whose parent was filtered out stays visible
  // as a top-level row instead of disappearing.
  const { roots, subtasksByParent } = useMemo(() => {
    const byParent = new Map<number, Task[]>()
    for (const task of tasks ?? []) {
      if (!task.parent_task_id || !taskIds.has(task.parent_task_id)) continue
      const list = byParent.get(task.parent_task_id) ?? []
      list.push(task)
      byParent.set(task.parent_task_id, list)
    }
    for (const list of byParent.values()) {
      list.sort((a, b) => compare(a, b, sort.key) * sort.dir)
    }

    const topLevel = (tasks ?? []).filter(
      (task) => !task.parent_task_id || !taskIds.has(task.parent_task_id),
    )
    topLevel.sort((a, b) => compare(a, b, sort.key) * sort.dir)
    return { roots: topLevel, subtasksByParent: byParent }
  }, [tasks, taskIds, sort])

  // Rows currently on screen: roots plus expanded subtasks. Select-all and
  // shift-click ranges operate on these, not on filtered-but-collapsed rows.
  const visibleRoots = useMemo(
    () =>
      hideDone
        ? roots.filter((task) => !task.completed_at && !doneColumnIds.has(task.column_id))
        : roots,
    [roots, hideDone, doneColumnIds],
  )
  const visibleTasks = useMemo(
    () =>
      visibleRoots.flatMap((task) => [
        task,
        ...(expanded.has(task.id) ? (subtasksByParent.get(task.id) ?? []) : []),
      ]),
    [visibleRoots, expanded, subtasksByParent],
  )
  const visibleIds = useMemo(() => visibleTasks.map((task) => task.id), [visibleTasks])
  const allSelected = visibleIds.length > 0 && visibleIds.every((id) => selected.has(id))
  const someSelected = !allSelected && visibleIds.some((id) => selected.has(id))
  // Only act on selections that still exist in the current result set, so the
  // bulk bar never sends requests for deleted or filtered-out tasks.
  const selectedTasks = useMemo(
    () => (tasks ?? []).filter((task) => selected.has(task.id)),
    [tasks, selected],
  )

  function toggleSelected(taskId: number, shiftKey: boolean) {
    setSelected((current) => {
      const next = new Set(current)
      const from = shiftKey && lastSelected !== null ? visibleIds.indexOf(lastSelected) : -1
      const to = visibleIds.indexOf(taskId)
      if (from !== -1 && to !== -1) {
        const shouldSelect = !current.has(taskId)
        const [start, end] = from < to ? [from, to] : [to, from]
        for (let index = start; index <= end; index++) {
          if (shouldSelect) next.add(visibleIds[index])
          else next.delete(visibleIds[index])
        }
        return next
      }
      if (next.has(taskId)) next.delete(taskId)
      else next.add(taskId)
      return next
    })
    setLastSelected(taskId)
  }

  function toggleAllSelected() {
    if (allSelected) {
      setSelected((current) => {
        const next = new Set(current)
        for (const id of visibleIds) next.delete(id)
        return next
      })
    } else {
      setSelected((current) => new Set([...current, ...visibleIds]))
    }
  }

  function toggleExpanded(taskId: number) {
    setExpanded((current) => {
      const next = new Set(current)
      if (next.has(taskId)) next.delete(taskId)
      else next.add(taskId)
      return next
    })
  }

  function toggleSort(key: SortKey) {
    setSort((s) => (s.key === key ? { key, dir: s.dir === 1 ? -1 : 1 } : { key, dir: 1 }))
  }

  const arrow = (key: SortKey) => (sort.key === key ? (sort.dir === 1 ? ' ↑' : ' ↓') : '')

  const renderRow = (task: Task, depth: number, leading?: ReactNode) => {
    const done = Boolean(task.completed_at) || doneColumnIds.has(task.column_id)
    return (
      <TableRow
        key={task.id}
        className="cursor-pointer hover:bg-muted/50"
        data-state={selected.has(task.id) ? 'selected' : undefined}
        onClick={() => setSearchParams({ task: String(task.id) })}
      >
        <TableCell onClick={(event) => event.stopPropagation()}>
          <Checkbox
            checked={selected.has(task.id)}
            aria-label={`Select ${task.title}`}
            onClick={(event) => {
              event.stopPropagation()
              toggleSelected(task.id, event.shiftKey)
            }}
          />
        </TableCell>
        <TableCell>
          <div className="flex items-center gap-1" style={{ paddingLeft: `${depth * 1.25}rem` }}>
            {leading ?? <span className="inline-block size-3.5 shrink-0" />}
            <span className={cn('truncate', done && 'text-muted-foreground line-through')}>
              {task.title}
            </span>
          </div>
        </TableCell>
        <TableCell>
          <span className="inline-flex items-center gap-1.5">
            <span
              className="size-1.5 shrink-0 rounded-full"
              style={{ backgroundColor: columnColor(columnById.get(task.column_id), colors) }}
            />
            {columnById.get(task.column_id)?.name}
          </span>
        </TableCell>
        <TableCell>
          <div className="flex flex-wrap gap-1">
            {(task.tags ?? []).map((tag) => (
              <span
                key={tag.id}
                className="inline-flex items-center gap-1 rounded-full border px-1.5 py-0.5 text-[0.65rem] text-muted-foreground"
                style={labelChipStyle(tag.color)}
              >
                <span
                  className="size-1.5 rounded-full"
                  style={{ backgroundColor: tag.color || undefined }}
                />
                {tag.name}
              </span>
            ))}
          </div>
        </TableCell>
        <TableCell>{task.priority}</TableCell>
        <TableCell>{task.due_date ?? '—'}</TableCell>
        <TableCell>
          <div className="flex items-center gap-2">
            <JiraKey taskId={task.id} />
            {task.ref && (
              <span className="rounded border px-1 font-mono text-[0.65rem] text-muted-foreground">
                {task.ref}
              </span>
            )}
          </div>
        </TableCell>
      </TableRow>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          placeholder="Filter tasks…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          className="max-w-xs"
        />
        <Select value={columnId} onValueChange={(value) => value && setColumnId(value)}>
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All columns</SelectItem>
            {columns?.map((c) => (
              <SelectItem key={c.id} value={String(c.id)}>
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={tag} onValueChange={(value) => value && setTag(value)}>
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All tags</SelectItem>
            {tags?.map((t) => (
              <SelectItem key={t.id} value={t.name}>
                {t.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <label className="flex items-center gap-2 text-sm">
          <Checkbox
            checked={hideDone}
            aria-label="Hide done tasks"
            onCheckedChange={(checked) => setHideDone(Boolean(checked))}
          />
          Hide done
        </label>
      </div>

      {selectedTasks.length > 0 && (
        <BulkTaskBar
          tasks={selectedTasks}
          columns={columns ?? []}
          tags={tags ?? []}
          onClear={() => {
            setSelected(new Set())
            setLastSelected(null)
          }}
        />
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-8">
              <Checkbox
                checked={allSelected}
                indeterminate={someSelected}
                aria-label={allSelected ? 'Deselect all tasks' : 'Select all tasks'}
                onCheckedChange={() => toggleAllSelected()}
              />
            </TableHead>
            <TableHead className="cursor-pointer" onClick={() => toggleSort('title')}>
              Title{arrow('title')}
            </TableHead>
            <TableHead>Column</TableHead>
            <TableHead>Labels</TableHead>
            <TableHead className="cursor-pointer" onClick={() => toggleSort('priority')}>
              Priority{arrow('priority')}
            </TableHead>
            <TableHead className="cursor-pointer" onClick={() => toggleSort('due_date')}>
              Due{arrow('due_date')}
            </TableHead>
            <TableHead className="w-24">Jira</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {visibleRoots.map((task) => {
            const subtasks = subtasksByParent.get(task.id) ?? []
            const doneCount = subtasks.filter(
              (subtask) => subtask.completed_at || doneColumnIds.has(subtask.column_id),
            ).length
            const isExpanded = expanded.has(task.id)
            return (
              <Fragment key={task.id}>
                {renderRow(
                  task,
                  0,
                  subtasks.length ? (
                    <>
                      <button
                        type="button"
                        aria-expanded={isExpanded}
                        aria-label={`${isExpanded ? 'Collapse' : 'Expand'} ${task.title}`}
                        className="flex size-3.5 shrink-0 items-center justify-center rounded hover:bg-muted"
                        onClick={(event) => {
                          event.stopPropagation()
                          toggleExpanded(task.id)
                        }}
                      >
                        <ChevronRight
                          className={cn(
                            'size-3.5 transition-transform',
                            isExpanded && 'rotate-90',
                          )}
                        />
                      </button>
                      <span className="shrink-0 text-xs text-muted-foreground">
                        {doneCount}/{subtasks.length}
                      </span>
                    </>
                  ) : undefined,
                )}
                {isExpanded && subtasks.map((subtask) => renderRow(subtask, 1))}
              </Fragment>
            )
          })}
          {!visibleRoots.length && (
            <TableRow>
              <TableCell colSpan={7} className="py-8 text-center text-muted-foreground">
                No tasks match these filters.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  )
}
