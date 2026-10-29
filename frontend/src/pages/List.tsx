import { Fragment, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useHotkeys } from 'react-hotkeys-hook'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ChevronRight, SlidersHorizontal } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatTimestamp } from '../lib/format'
import { useBoards, useColumns, useDeleteTask, useJiraLinks, useMoveTask, useRestoreTasks, useRunningTime, useStartTime, useStopTime, useTags, useTasks, useUpdateTask } from '../api/hooks'
import { showUndo } from '../lib/undo'
import { BulkTaskBar } from '../components/board/BulkTaskBar'
import { TaskComposer } from '../components/board/TaskComposer'
import { TaskInlineEditor } from '../components/board/TaskInlineEditor'
import { TaskTypeIcon } from '../components/TaskTypeIcon'
import { taskStatus } from '../lib/taskType'
import { StatusPill } from '../components/StatusPill'
import { TagChip } from '../components/TagChip'
import { useDebounced } from '../lib/useDebounced'
import { columnColorCss, useColumnColors } from '../store/columnColors'
import { useUIStore } from '../store/ui'
import { useWorkspaceStore } from '../store/workspace'
import { cn } from '@/lib/utils'
import type { JiraLink, Task } from '../api/types'

type SortKey = 'title' | 'priority' | 'due_date' | 'created_at' | 'updated_at'
type ColumnKey = 'column' | 'tags' | Exclude<SortKey, 'title'>

const priorityRank: Record<string, number> = { high: 0, medium: 1, low: 2 }

function compare(a: Task, b: Task, key: SortKey) {
  if (key === 'priority') return (priorityRank[a.priority] ?? 3) - (priorityRank[b.priority] ?? 3)
  if (key === 'due_date') {
    if (!a.due_date) return 1
    if (!b.due_date) return -1
    return a.due_date.localeCompare(b.due_date)
  }
  if (key === 'created_at' || key === 'updated_at') {
    return a[key].localeCompare(b[key])
  }
  return a.title.localeCompare(b.title)
}

const optionalColumns: { key: ColumnKey; label: string }[] = [
  { key: 'column', label: 'Status' },
  { key: 'tags', label: 'Labels' },
  { key: 'priority', label: 'Priority' },
  { key: 'due_date', label: 'Due' },
  { key: 'created_at', label: 'Created' },
  { key: 'updated_at', label: 'Last updated' },
]

const byKey = new Map(optionalColumns.map((column) => [column.key, column]))

function isSortColumn(key: ColumnKey): key is Exclude<SortKey, 'title'> {
  return (
    key === 'priority' ||
    key === 'due_date' ||
    key === 'created_at' ||
    key === 'updated_at'
  )
}

const defaultVisible: ColumnKey[] = ['column', 'tags', 'priority', 'due_date', 'updated_at']

function loadVisibleColumns(): Set<ColumnKey> {
  try {
    const saved: unknown = JSON.parse(localStorage.getItem('list-columns') ?? 'null')
    if (Array.isArray(saved)) {
      const valid = saved.filter((key): key is ColumnKey => byKey.has(key as ColumnKey))
      return new Set(valid.length ? valid : [])
    }
  } catch {
    // fall through to defaults
  }
  return new Set(defaultVisible)
}

function JiraChip({ link }: { link?: JiraLink }) {
  if (!link) return null
  return (
    <a
      href={link.jira_url}
      target="_blank"
      rel="noreferrer"
      aria-label={`Open ${link.jira_key} in Jira`}
      className="shrink-0 rounded border px-1 font-mono text-[0.65rem] text-primary hover:underline"
      onClick={(event) => event.stopPropagation()}
    >
      {link.jira_key}
    </a>
  )
}

type Sort = { key: SortKey; dir: 1 | -1 }

const sortKeys = new Set<SortKey>(['title', 'priority', 'due_date', 'created_at', 'updated_at'])

function loadSort(): Sort {
  try {
    const saved: unknown = JSON.parse(localStorage.getItem('list-sort') ?? 'null')
    if (
      saved &&
      typeof saved === 'object' &&
      'key' in saved &&
      'dir' in saved &&
      sortKeys.has(saved.key as SortKey) &&
      (saved.dir === 1 || saved.dir === -1)
    ) {
      return { key: saved.key as SortKey, dir: saved.dir }
    }
  } catch {
    // fall through to default
  }
  return { key: 'title', dir: 1 }
}

export function List() {
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: boards } = useBoards(workspaceId)
  const { selectedBoardId } = useUIStore()
  const board = boards?.find((b) => b.id === selectedBoardId)
  const boardId = board?.id ?? boards?.[0]?.id
  const workspaceParam = board?.workspace_id ?? boards?.[0]?.workspace_id

  const { data: columns } = useColumns(boardId)
  const columnItems = useMemo(
    () => [
      { value: 'all', label: 'All columns' },
      ...(columns ?? []).map((column) => ({ value: String(column.id), label: column.name })),
    ],
    [columns],
  )
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

  const [searchParams, setSearchParams] = useSearchParams()
  const [sort, setSort] = useState<Sort>(loadSort)
  const [visible, setVisible] = useState<Set<ColumnKey>>(loadVisibleColumns)
  const [expanded, setExpanded] = useState<Set<number>>(() => new Set())
  const [selected, setSelected] = useState<Set<number>>(() => new Set())
  const [lastSelected, setLastSelected] = useState<number | null>(null)
  const [hideDone, setHideDone] = useState(false)
  const [inlineTaskId, setInlineTaskId] = useState<number | null>(null)
  const [focusTitle, setFocusTitle] = useState(false)
  const [fieldFocus, setFieldFocus] = useState<{
    field: 'due' | 'status' | 'labels'
    nonce: number
  } | null>(null)
  const composerRef = useRef<HTMLInputElement>(null)
  const filterRef = useRef<HTMLInputElement>(null)
  const moveTask = useMoveTask(boardId)
  const deleteTask = useDeleteTask(boardId)
  const restoreTasks = useRestoreTasks(boardId)
  const updateTask = useUpdateTask(boardId)
  const { data: running } = useRunningTime()
  const startTime = useStartTime()
  const stopTime = useStopTime()
  const navigate = useNavigate()
  const [showShortcuts, setShowShortcuts] = useState(false)

  useEffect(() => {
    const raw = searchParams.get('task')
    const id = raw ? Number(raw) : NaN
    if (!Number.isFinite(id)) return
    setInlineTaskId(id)
    const frame = requestAnimationFrame(() => {
      document.getElementById(`task-${id}`)?.scrollIntoView({ block: 'nearest' })
    })
    return () => cancelAnimationFrame(frame)
  }, [searchParams])

  useEffect(() => {
    localStorage.setItem('list-columns', JSON.stringify([...visible]))
  }, [visible])

  useEffect(() => {
    localStorage.setItem('list-sort', JSON.stringify(sort))
  }, [sort])

  const { data: jiraLinks } = useJiraLinks(workspaceParam)
  const jiraLinkById = useMemo(
    () => new Map((jiraLinks ?? []).map((jiraLink) => [jiraLink.task_id, jiraLink])),
    [jiraLinks],
  )

  const visibleList = useMemo(() => optionalColumns.filter((column) => visible.has(column.key)), [visible])

  const columnById = useMemo(
    () => new Map((columns ?? []).map((column) => [column.id, column])),
    [columns],
  )
  const { colors } = useColumnColors()

  const doneColumnIds = useMemo(
    () => new Set((columns ?? []).filter((column) => column.is_done).map((column) => column.id)),
    [columns],
  )

  const firstColumnId = useMemo(
    () => [...(columns ?? [])].sort((a, b) => a.position - b.position)[0]?.id,
    [columns],
  )

  useHotkeys('c', () => composerRef.current?.focus(), {
    enabled: Boolean(boardId && firstColumnId),
    preventDefault: true,
  })
  useHotkeys('/', () => filterRef.current?.focus(), { preventDefault: true })
  useHotkeys('j', () => moveFocus(1), { preventDefault: true })
  useHotkeys('k', () => moveFocus(-1), { preventDefault: true })
  useHotkeys(['enter', 'o'], () => toggleInline(inlineTaskId), {
    enabled: inlineTaskId !== null,
    preventDefault: true,
    enableOnFormTags: false,
  })
  useHotkeys('e', () => {
    const target = inlineTaskId ?? visibleRoots[0]?.id
    if (!target) return
    setInlineTaskId(target)
    setFocusTitle(true)
  }, { preventDefault: true })
  useHotkeys('x', () => toggleDone(inlineTaskId), {
    enabled: inlineTaskId !== null,
    preventDefault: true,
  })
  useHotkeys('backspace', () => removeTask(inlineTaskId), {
    enabled: inlineTaskId !== null,
    preventDefault: true,
  })
  useHotkeys('escape', () => setInlineTaskId(null), {
    enabled: inlineTaskId !== null,
    preventDefault: true,
  })
  useHotkeys(['1', '2', '3', '4'], (event) => {
    if (inlineTaskId === null) return
    const priority = event.key === '1' ? 'low' : event.key === '2' ? 'medium' : 'high'
    updateTask.mutate({ id: inlineTaskId, priority })
  }, { enabled: inlineTaskId !== null, preventDefault: true })
  useHotkeys('d', () => {
    if (inlineTaskId === null) return
    setInlineTaskId(inlineTaskId)
    setFieldFocus({ field: 'due', nonce: Date.now() })
  }, { enabled: inlineTaskId !== null, preventDefault: true })
  useHotkeys('m', () => {
    if (inlineTaskId === null) return
    setFieldFocus({ field: 'status', nonce: Date.now() })
  }, { enabled: inlineTaskId !== null, preventDefault: true })
  useHotkeys('l', () => {
    if (inlineTaskId === null) return
    setFieldFocus({ field: 'labels', nonce: Date.now() })
  }, { enabled: inlineTaskId !== null, preventDefault: true })
  useHotkeys('t', () => {
    if (inlineTaskId === null) return
    if (running?.task_id === inlineTaskId) stopTime.mutate(inlineTaskId)
    else startTime.mutate(inlineTaskId)
  }, { enabled: inlineTaskId !== null, preventDefault: true })
  useHotkeys('shift+enter', () => {
    if (inlineTaskId !== null) navigate(`/tasks/${inlineTaskId}`)
  }, { enabled: inlineTaskId !== null, preventDefault: true })
  useHotkeys('shift+slash', () => setShowShortcuts((current) => !current), { preventDefault: true })

  const taskIds = useMemo(() => new Set((tasks ?? []).map((task) => task.id)), [tasks])

  // Group tasks under their parent; a child whose parent was filtered out stays
  // visible as a top-level row instead of disappearing.
  const { roots, childrenByParent } = useMemo(() => {
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
    return { roots: topLevel, childrenByParent: byParent }
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
  const visibleTasks = useMemo(() => {
    const walk = (task: Task, depth: number): Task[] => {
      if (depth > 8) return [task]
      const children = expanded.has(task.id) ? (childrenByParent.get(task.id) ?? []) : []
      return [task, ...children.flatMap((child) => walk(child, depth + 1))]
    }
    return visibleRoots.flatMap((task) => walk(task, 0))
  }, [visibleRoots, expanded, childrenByParent])
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

  const cells: Record<ColumnKey, (task: Task) => ReactNode> = {
    column: (task) => {
      const column = columnById.get(task.column_id)
      return (
        <StatusPill
          label={column?.name ?? 'Unknown'}
          color={columnColorCss(column, colors)}
        />
      )
    },
    tags: (task) => (
      <div className="flex flex-wrap gap-1">
        {(task.tags ?? []).map((tag) => (
          <TagChip
            key={tag.id}
            tag={tag}
            className="rounded-full px-1.5 py-0.5 text-[0.65rem] text-muted-foreground"
          />
        ))}
      </div>
    ),
    priority: (task) => task.priority,
    due_date: (task) => task.due_date ?? '—',
    created_at: (task) => formatTimestamp(task.created_at),
    updated_at: (task) => formatTimestamp(task.updated_at),
  }

  function moveFocus(direction: 1 | -1) {
    if (!visibleRoots.length) return
    const index = visibleRoots.findIndex((task) => task.id === inlineTaskId)
    const next = index < 0
      ? (direction === 1 ? 0 : visibleRoots.length - 1)
      : Math.min(visibleRoots.length - 1, Math.max(0, index + direction))
    setInlineTaskId(visibleRoots[next].id)
    setFocusTitle(false)
  }

  function toggleInline(id: number | null) {
    if (id !== null) setInlineTaskId(id)
  }

  function toggleDone(id: number | null) {
    if (id === null) return
    const task = (tasks ?? []).find((item) => item.id === id)
    if (!task) return
    const done = Boolean(task.completed_at) || doneColumnIds.has(task.column_id)
    const target = done
      ? columns?.find((column) => !column.is_done)
      : columns?.find((column) => column.is_done)
    if (target) moveTask.mutate({ id, column_id: target.id })
  }

  function removeTask(id: number | null) {
    if (id === null) return
    const task = (tasks ?? []).find((item) => item.id === id)
    if (!task) return
    deleteTask.mutate(id, {
      onSuccess: () => {
        setInlineTaskId(null)
        showUndo(`Deleted "${task.title}"`, () =>
          restoreTasks.mutateAsync([task.id]).then(() => undefined),
        )
      },
    })
  }

  const renderRow = (task: Task, depth: number, leading?: ReactNode) => {
    const done = Boolean(task.completed_at) || doneColumnIds.has(task.column_id)
    return (
      <TableRow
        key={task.id}
        id={`task-${task.id}`}
        className="cursor-pointer hover:bg-muted/50"
        data-state={selected.has(task.id) ? 'selected' : undefined}
        onClick={() => setInlineTaskId((current) => (current === task.id ? null : task.id))}
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
          <div
            className="flex min-w-0 items-center gap-1"
            style={{ paddingLeft: `${depth * 1.25}rem` }}
          >
            {leading ?? <span className="inline-block size-3.5 shrink-0" />}
            <TaskTypeIcon
              kind={task.kind}
              status={taskStatus(task, columns)}
              size={14}
              label={task.kind === 'arc' ? 'Arc' : task.kind === 'step' ? 'Step' : 'Task'}
            />
            <button
              type="button"
              className={cn(
                'min-w-0 flex-1 truncate rounded-sm text-left outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring/50',
                done && 'text-muted-foreground line-through',
              )}
              aria-expanded={inlineTaskId === task.id}
              aria-controls={`task-expansion-${task.id}`}
              onClick={(event) => {
                event.stopPropagation()
                setInlineTaskId((current) => (current === task.id ? null : task.id))
              }}
            >
              {task.title}
            </button>
            <div className="ml-auto flex shrink-0 items-center gap-1">
              <JiraChip link={jiraLinkById.get(task.id)} />
              {task.ref && (
                <span className="rounded border px-1 font-mono text-[0.65rem] text-muted-foreground">
                  {task.ref}
                </span>
              )}
            </div>
          </div>
        </TableCell>
        {visibleList.map((column) => (
          <TableCell key={column.key}>{cells[column.key](task)}</TableCell>
        ))}
      </TableRow>
    )
  }

  const renderTree = (task: Task, depth: number): ReactNode => {
    const children = childrenByParent.get(task.id) ?? []
    const doneCount = children.filter(
      (child) => child.completed_at || doneColumnIds.has(child.column_id),
    ).length
    const isExpanded = expanded.has(task.id)
    return (
      <Fragment key={task.id}>
        {renderRow(
          task,
          depth,
          children.length ? (
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
                  className={cn('size-3.5 transition-transform', isExpanded && 'rotate-90')}
                />
              </button>
              <span className="shrink-0 text-xs text-muted-foreground">
                {doneCount}/{children.length}
              </span>
            </>
          ) : undefined,
        )}
        {inlineTaskId === task.id && (
          <TableRow>
            <TableCell colSpan={2 + visibleList.length} className="p-2">
              <div className="fai-expand">
                <div>
              <TaskInlineEditor
                task={task}
                columns={columns ?? []}
                autoFocusTitle={focusTitle}
                focusField={fieldFocus?.field}
                focusNonce={fieldFocus?.nonce}
                onClose={() => {
                  setInlineTaskId(null)
                  setFocusTitle(false)
                }}
                onOpenDetails={() => {
                  if (window.matchMedia('(max-width: 767px)').matches) {
                    navigate(`/tasks/${task.id}`)
                    return
                  }
                  const next = new URLSearchParams(searchParams)
                  next.set('task', String(task.id))
                  next.set('panel', '1')
                  setSearchParams(next)
                }}
              />
                </div>
              </div>
            </TableCell>
          </TableRow>
        )}
        {isExpanded && children.map((child) => renderTree(child, depth + 1))}
      </Fragment>
    )
  }

  return (
    <div className="flex flex-col gap-4 pb-24">
      <h1 className="text-lg font-semibold">List</h1>
      <div className="flex flex-wrap items-center gap-2">
        <Input
          ref={filterRef}
          aria-label="Filter tasks"
          placeholder="Filter tasks…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          className="max-w-xs"
        />
        <Select value={columnId} items={columnItems} onValueChange={(value) => value && setColumnId(value)}>
          <SelectTrigger className="w-40" aria-label="Filter by column">
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
          <SelectTrigger className="w-40" aria-label="Filter by tag">
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
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="outline" size="sm" className="ml-auto"><SlidersHorizontal /> Columns</Button>} />
          <DropdownMenuContent>
            <DropdownMenuGroup>
              <DropdownMenuLabel>Visible columns</DropdownMenuLabel>
              {optionalColumns.map(({ key, label }) => (
                <DropdownMenuCheckboxItem
                  key={key}
                  checked={visible.has(key)}
                  onCheckedChange={(checked) =>
                    setVisible((current) => {
                      const next = new Set(current)
                      if (checked) next.add(key)
                      else next.delete(key)
                      return next
                    })
                  }
                >
                  {label}
                </DropdownMenuCheckboxItem>
              ))}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <TaskComposer
        boardId={boardId}
        columnId={firstColumnId}
        inputRef={composerRef}
        placeholder="Add a task"
        className="max-w-xl"
      />

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
            <TableHead
              aria-sort={
                sort.key === 'title' ? (sort.dir === 1 ? 'ascending' : 'descending') : 'none'
              }
            >
              <button
                type="button"
                className="flex items-center gap-1 rounded-sm font-medium outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50"
                onClick={() => toggleSort('title')}
              >
                Title{arrow('title')}
              </button>
            </TableHead>
            {visibleList.map(({ key, label }) => (
              <TableHead
                key={key}
                aria-sort={
                  isSortColumn(key)
                    ? sort.key === key
                      ? sort.dir === 1
                        ? 'ascending'
                        : 'descending'
                      : 'none'
                    : undefined
                }
              >
                {isSortColumn(key) ? (
                  <button
                    type="button"
                    className="flex items-center gap-1 rounded-sm font-medium outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50"
                    onClick={() => toggleSort(key)}
                  >
                    {label}
                    {arrow(key)}
                  </button>
                ) : (
                  label
                )}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {visibleRoots.map((task) => renderTree(task, 0))}
          {!visibleRoots.length && (
            <TableRow>
              <TableCell colSpan={2 + visibleList.length} className="py-8 text-center text-muted-foreground">
                No tasks match these filters.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>

      {showShortcuts && (
        <div className="fixed right-4 bottom-20 z-40 w-64 rounded-lg border bg-popover p-3 text-xs shadow-lg lg:bottom-4">
          <div className="mb-2 flex items-center justify-between">
            <span className="font-medium">Shortcuts</span>
            <Button size="xs" variant="ghost" onClick={() => setShowShortcuts(false)}>
              Close
            </Button>
          </div>
          <dl className="grid grid-cols-2 gap-1">
            {[
              ['j / k', 'Next / previous'],
              ['Enter / o', 'Expand'],
              ['e', 'Edit title'],
              ['x', 'Toggle done'],
              ['1 to 4', 'Priority'],
              ['t', 'Timer'],
              ['Backspace', 'Delete'],
              ['c', 'Quick add'],
              ['/', 'Filter'],
              ['Shift+Enter', 'Details'],
              ['Escape', 'Collapse'],
            ].map(([key, label]) => (
              <Fragment key={key}>
                <dt className="font-mono text-muted-foreground">{key}</dt>
                <dd>{label}</dd>
              </Fragment>
            ))}
          </dl>
        </div>
      )}
    </div>
  )
}
