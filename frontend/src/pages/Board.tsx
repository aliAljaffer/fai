import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  TouchSensor,
  closestCenter,
  pointerWithin,
  useSensor,
  useSensors,
  type CollisionDetection,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  horizontalListSortingStrategy,
  sortableKeyboardCoordinates,
} from '@dnd-kit/sortable'
import { Pencil, Plus, Search } from 'lucide-react'
import { Fragment, useEffect, useMemo, useRef, useState } from 'react'
import { useHotkeys } from 'react-hotkeys-hook'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  useBoards,
  useColumns,
  useCreateBoard,
  useCreateColumn,
  useCreateTask,
  useMoveTask,
  useReorderColumns,
  useDeleteTask,
  useRestoreTasks,
  useRunningTime,
  useStartTime,
  useStopTime,
  useTasks,
  useUpdateBoard,
  useUpdateColumn,
  useUpdateTask,
  useWorkspaces,
} from '../api/hooks'
import { BoardColumn } from '../components/board/BoardColumn'
import { RemindersColumn } from '../components/board/RemindersColumn'
import { TaskCard } from '../components/board/TaskCard'
import { isPastDue } from '../lib/format'
import { showUndo } from '../lib/undo'
import { useUIStore } from '../store/ui'
import { useWorkspaceStore } from '../store/workspace'
import type { Task } from '../api/types'

const collisionDetection: CollisionDetection = (args) => {
  const hits = pointerWithin(args)
  if (args.active.data.current?.type === 'column') {
    const columnHits = hits.filter((hit) => hit.data?.type === 'column')
    const ids = new Set(
      (columnHits.length
        ? columnHits
        : args.droppableContainers.filter(
            (container) => container.data.current?.type === 'column',
          )
      ).map((hit) => hit.id),
    )
    if (!ids.size) return []
    return closestCenter({
      ...args,
      droppableContainers: args.droppableContainers.filter((container) => ids.has(container.id)),
    })
  }
  const taskHits = hits.filter((hit) => hit.data?.type === 'task')
  const candidates = taskHits.length ? taskHits : hits
  if (!candidates.length) return closestCenter(args)
  const ids = new Set(candidates.map((hit) => hit.id))
  return closestCenter({
    ...args,
    droppableContainers: args.droppableContainers.filter((container) => ids.has(container.id)),
  })
}

function tasksInColumn(tasks: Task[], columnId: number) {
  return tasks
    .filter((task) => task.column_id === columnId)
    .sort((a, b) => a.position - b.position)
}

function withPositions(tasks: Task[], ordered: Task[]) {
  const positions = new Map(ordered.map((task, index) => [task.id, (index + 1) * 1024]))
  return tasks.map((task) =>
    positions.has(task.id) ? { ...task, position: positions.get(task.id)! } : task,
  )
}

function previewTaskAtEnd(tasks: Task[], activeId: number, columnId: number) {
  const active = tasks.find((task) => task.id === activeId)
  if (!active) return tasks
  const full = tasksInColumn(tasks, columnId)
  if (full[full.length - 1]?.id === activeId) return tasks

  const rest = tasks.filter((task) => task.id !== activeId)
  const target = tasksInColumn(rest, columnId)
  const moving = { ...active, column_id: columnId }
  const ordered = [...target, moving]
  return withPositions([...rest, moving], ordered)
}

function previewTaskMove(
  tasks: Task[],
  activeId: number,
  targetColumnId: number,
  overTaskId: number | null,
  below: boolean,
) {
  const active = tasks.find((task) => task.id === activeId)
  if (!active || active.column_id === targetColumnId) return tasks

  const rest = tasks.filter((task) => task.id !== activeId)
  const target = tasksInColumn(rest, targetColumnId)
  let index = target.length
  if (overTaskId !== null) {
    const overIndex = target.findIndex((task) => task.id === overTaskId)
    if (overIndex >= 0) index = overIndex + (below ? 1 : 0)
  }

  const moving = { ...active, column_id: targetColumnId }
  const ordered = [...target.slice(0, index), moving, ...target.slice(index)]
  return withPositions([...rest, moving], ordered)
}

function computeTaskDrop(tasks: Task[], activeId: number, overId: string) {
  const active = tasks.find((task) => task.id === activeId)
  if (!active) return null

  let targetColumnId: number
  let overTaskId: number | null = null
  if (overId.startsWith('column-')) {
    targetColumnId = Number(overId.slice('column-'.length))
  } else {
    const overTask = tasks.find((task) => task.id === Number(overId))
    if (!overTask) return null
    targetColumnId = overTask.column_id
    overTaskId = overTask.id
  }

  const withActive = tasksInColumn(tasks, targetColumnId)
  const activeIndex = withActive.findIndex((task) => task.id === activeId)
  const withoutActive = withActive.filter((task) => task.id !== activeId)

  let insertAt: number
  if (overTaskId === null) {
    insertAt = withoutActive.length
  } else {
    if (overTaskId === activeId) return null
    insertAt = withActive.findIndex((task) => task.id === overTaskId)
    if (insertAt < 0) return null
  }

  return {
    id: activeId,
    columnId: targetColumnId,
    beforeTaskId: withoutActive[insertAt]?.id,
    afterTaskId: withoutActive[insertAt - 1]?.id,
    changed: active.column_id !== targetColumnId || activeIndex !== insertAt,
  }
}

function LoadError({
  title,
  detail,
  onRetry,
}: {
  title: string
  detail: string
  onRetry: () => void
}) {
  return (
    <div className="flex max-w-md flex-col items-start gap-2 rounded-md border border-destructive/40 p-4">
      <p className="text-sm font-medium">{title}</p>
      <p className="text-xs text-muted-foreground">{detail}</p>
      <Button size="sm" variant="outline" onClick={onRetry}>
        Retry
      </Button>
    </div>
  )
}

function BoardSkeleton({ columns = 4 }: { columns?: number }) {
  return (
    <div className="flex flex-col gap-4" aria-busy="true" aria-label="Loading board">
      <div className="flex flex-col gap-2">
        <Skeleton className="h-6 w-40" />
        <Skeleton className="h-4 w-56" />
      </div>
      <div className="flex gap-4">
        {Array.from({ length: columns }).map((_, index) => (
          <div key={index} className="flex w-72 shrink-0 flex-col gap-2">
            <Skeleton className="h-6 w-full" />
            <Skeleton className="h-14 w-full rounded-md" />
            <Skeleton className="h-14 w-full rounded-md" />
            <Skeleton className="h-14 w-full rounded-md" />
          </div>
        ))}
      </div>
    </div>
  )
}

export function Board() {
  const { activeWorkspaceId, setActiveWorkspaceId } = useWorkspaceStore()
  const workspaceId = activeWorkspaceId
  const { data: workspaces } = useWorkspaces()
  const boardsQuery = useBoards(workspaceId)
  const boards = boardsQuery.data
  const { selectedBoardId, setSelectedBoardId, newTaskRequest } = useUIStore()
  const board = boards?.find((item) => item.id === selectedBoardId) ?? boards?.[0]
  const boardId = board?.id

  useEffect(() => {
    if (boards?.[0] && !boards.some((item) => item.id === selectedBoardId)) {
      setSelectedBoardId(boards[0].id)
    }
  }, [boards, selectedBoardId, setSelectedBoardId])

  const columnsQuery = useColumns(boardId)
  const tasksQuery = useTasks(boardId)
  const columns = columnsQuery.data
  const tasks = tasksQuery.data
  const createTask = useCreateTask(boardId)
  const createColumn = useCreateColumn(boardId)
  const updateColumn = useUpdateColumn(boardId)
  const reorderColumns = useReorderColumns(boardId)
  const moveTask = useMoveTask(boardId)
  const updateTask = useUpdateTask(boardId)
  const deleteTask = useDeleteTask(boardId)
  const restoreTasks = useRestoreTasks(boardId)
  const { data: running } = useRunningTime()
  const startTime = useStartTime()
  const stopTime = useStopTime()
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()

  const [query, setQuery] = useState('')
  const [priority, setPriority] = useState('all')
  const [activeTaskId, setActiveTaskId] = useState<number | null>(null)
  const [overColumnId, setOverColumnId] = useState<number | null>(null)
  const [dragTasks, setDragTasks] = useState<Task[] | null>(null)
  const [composerColumnId, setComposerColumnId] = useState<number | null>(null)
  const [expandedTaskId, setExpandedTaskId] = useState<number | null>(null)
  const [boardEditor, setBoardEditor] = useState<'new' | 'rename' | null>(null)
  const [showShortcuts, setShowShortcuts] = useState(false)
  const [fieldFocus, setFieldFocus] = useState<{
    field: 'due' | 'status' | 'labels'
    nonce: number
  } | null>(null)
  const [boardName, setBoardName] = useState('')
  const [addingColumn, setAddingColumn] = useState(false)
  const [columnName, setColumnName] = useState('')
  const [columnIsDone, setColumnIsDone] = useState(false)
  const handledNewTaskRequest = useRef(0)
  const createBoard = useCreateBoard(workspaceId)
  const updateBoard = useUpdateBoard()
  const boardSaving = createBoard.isPending || updateBoard.isPending

  const orderedColumns = useMemo(
    () => [...(columns ?? [])].sort((a, b) => a.position - b.position),
    [columns],
  )

  useEffect(() => {
    const raw = searchParams.get('task')
    const id = raw ? Number(raw) : NaN
    if (!Number.isFinite(id)) return
    setExpandedTaskId(id)
    const frame = requestAnimationFrame(() => {
      document.getElementById(`task-${id}`)?.scrollIntoView({ block: 'nearest' })
    })
    return () => cancelAnimationFrame(frame)
  }, [searchParams])

  useEffect(() => {
    setComposerColumnId(null)
    setQuery('')
    setPriority('all')
    setAddingColumn(false)
    setExpandedTaskId(null)
  }, [boardId])

  useEffect(() => {
    if (newTaskRequest === handledNewTaskRequest.current) return
    if (!orderedColumns[0]) return
    handledNewTaskRequest.current = newTaskRequest
    setComposerColumnId(orderedColumns[0].id)
  }, [newTaskRequest, orderedColumns])

  useHotkeys('c', () => {
    if (orderedColumns[0]) setComposerColumnId(orderedColumns[0].id)
  }, { enabled: Boolean(boardId && orderedColumns.length), preventDefault: true })
  useHotkeys('j', () => moveBoardFocus(1), { preventDefault: true })
  useHotkeys('k', () => moveBoardFocus(-1), { preventDefault: true })
  useHotkeys(['enter', 'o'], () => {
    if (expandedTaskId !== null) setExpandedTaskId(null)
    else if (filteredTasks[0]) setExpandedTaskId(filteredTasks[0].id)
  }, { preventDefault: true })
  useHotkeys('x', () => {
    if (expandedTaskId === null) return
    const task = (tasks ?? []).find((item) => item.id === expandedTaskId)
    if (!task) return
    const done = Boolean(task.completed_at) || doneColumnIds.has(task.column_id)
    const target = done
      ? orderedColumns.find((column) => !column.is_done)
      : orderedColumns.find((column) => column.is_done)
    if (target) moveTask.mutate({ id: task.id, column_id: target.id })
  }, { enabled: expandedTaskId !== null, preventDefault: true })
  useHotkeys(['1', '2', '3', '4'], (event) => {
    if (expandedTaskId === null) return
    const priority = event.key === '1' ? 'low' : event.key === '2' ? 'medium' : 'high'
    updateTask.mutate({ id: expandedTaskId, priority })
  }, { enabled: expandedTaskId !== null, preventDefault: true })
  useHotkeys('d', () => {
    const id = expandedTaskId ?? filteredTasks[0]?.id
    if (id === undefined) return
    setExpandedTaskId(id)
    setFieldFocus({ field: 'due', nonce: Date.now() })
  }, { preventDefault: true })
  useHotkeys('m', () => {
    const id = expandedTaskId ?? filteredTasks[0]?.id
    if (id === undefined) return
    setExpandedTaskId(id)
    setFieldFocus({ field: 'status', nonce: Date.now() })
  }, { preventDefault: true })
  useHotkeys('l', () => {
    const id = expandedTaskId ?? filteredTasks[0]?.id
    if (id === undefined) return
    setExpandedTaskId(id)
    setFieldFocus({ field: 'labels', nonce: Date.now() })
  }, { preventDefault: true })
  useHotkeys('t', () => {
    if (expandedTaskId === null) return
    if (running?.task_id === expandedTaskId) stopTime.mutate(expandedTaskId)
    else startTime.mutate(expandedTaskId)
  }, { enabled: expandedTaskId !== null, preventDefault: true })
  useHotkeys('backspace', () => {
    if (expandedTaskId === null) return
    const task = (tasks ?? []).find((item) => item.id === expandedTaskId)
    if (!task) return
    deleteTask.mutate(task.id, {
      onSuccess: () => {
        setExpandedTaskId(null)
        showUndo(`Deleted "${task.title}"`, () =>
          restoreTasks.mutateAsync([task.id]).then(() => undefined),
        )
      },
    })
  }, { enabled: expandedTaskId !== null, preventDefault: true })
  useHotkeys('escape', () => setExpandedTaskId(null), {
    enabled: expandedTaskId !== null,
    preventDefault: true,
  })
  useHotkeys('shift+enter', () => {
    if (expandedTaskId !== null) setSearchParams({ task: String(expandedTaskId), panel: '1' })
  }, { enabled: expandedTaskId !== null, preventDefault: true })
  useHotkeys('shift+slash', () => setShowShortcuts((current) => !current), { preventDefault: true })

  async function submitBoard() {
    const value = boardName.trim()
    if (!value || boardSaving) return
    try {
      if (boardEditor === 'rename' && board) {
        await updateBoard.mutateAsync({ id: board.id, name: value })
      } else {
        const created = await createBoard.mutateAsync(value)
        setSelectedBoardId(created.id)
      }
      setBoardName('')
      setBoardEditor(null)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not save board')
    }
  }

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 250, tolerance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const doneColumnIds = useMemo(
    () => new Set(orderedColumns.filter((column) => column.is_done).map((column) => column.id)),
    [orderedColumns],
  )

  const hasFilters = query.trim().length > 0 || priority !== 'all'

  const topLevelTasks = useMemo(
    () => (tasks ?? []).filter((task) => !task.parent_task_id),
    [tasks],
  )

  const subtasksByParent = useMemo(() => {
    const map = new Map<number, Task[]>()
    for (const task of tasks ?? []) {
      if (!task.parent_task_id) continue
      const list = map.get(task.parent_task_id) ?? []
      list.push(task)
      map.set(task.parent_task_id, list)
    }
    for (const list of map.values()) list.sort((a, b) => a.position - b.position)
    return map
  }, [tasks])

  const filteredTasks = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return topLevelTasks.filter((task) => {
      if (priority !== 'all' && task.priority !== priority) return false
      if (!needle) return true
      return (
        task.title.toLowerCase().includes(needle) ||
        task.tags.some((tag) => tag.name.toLowerCase().includes(needle))
      )
    })
  }, [topLevelTasks, query, priority])

  const tasksByColumn = useMemo(() => {
    const map = new Map<number, Task[]>()
    for (const task of dragTasks ?? filteredTasks) {
      const list = map.get(task.column_id) ?? []
      list.push(task)
      map.set(task.column_id, list)
    }
    for (const list of map.values()) list.sort((a, b) => a.position - b.position)
    return map
  }, [dragTasks, filteredTasks])

  const totalByColumn = useMemo(() => {
    const map = new Map<number, number>()
    const previewActive = dragTasks?.find((task) => task.id === activeTaskId)
    for (const task of tasks ?? []) {
      const columnId =
        previewActive && task.id === previewActive.id ? previewActive.column_id : task.column_id
      map.set(columnId, (map.get(columnId) ?? 0) + 1)
    }
    return map
  }, [tasks, dragTasks, activeTaskId])

  const openCount = topLevelTasks.filter(
    (task) => !task.completed_at && !doneColumnIds.has(task.column_id),
  ).length
  const doneCount = topLevelTasks.length - openCount
  const overdueCount = topLevelTasks.filter(
    (task) =>
      !doneColumnIds.has(task.column_id) &&
      isPastDue(task.due_date, Boolean(task.completed_at)),
  ).length

  const activeTask = tasks?.find((task) => task.id === activeTaskId)

  function moveBoardFocus(direction: 1 | -1) {
    if (!filteredTasks.length) return
    const index = filteredTasks.findIndex((task) => task.id === expandedTaskId)
    const next = index < 0
      ? (direction === 1 ? 0 : filteredTasks.length - 1)
      : Math.min(filteredTasks.length - 1, Math.max(0, index + direction))
    setExpandedTaskId(filteredTasks[next].id)
  }

  function handleDragStart(event: DragStartEvent) {
    setOverColumnId(null)
    setExpandedTaskId(null)
    if (event.active.data.current?.type === 'column') {
      setActiveTaskId(null)
      setDragTasks(null)
      return
    }
    setActiveTaskId(Number(event.active.id))
    setDragTasks(filteredTasks.map((task) => ({ ...task })))
  }

  function handleDragOver(event: DragOverEvent) {
    if (event.active.data.current?.type === 'column') return
    const overId = event.over ? String(event.over.id) : ''
    const overTask = overId.startsWith('column-')
      ? undefined
      : dragTasks?.find((task) => task.id === Number(overId))
    const targetColumnId = overId.startsWith('column-')
      ? Number(overId.slice('column-'.length))
      : overTask?.column_id ?? null
    setOverColumnId(targetColumnId)
    if (!dragTasks || !event.over) return

    const activeId = Number(event.active.id)
    const activeTask = dragTasks.find((task) => task.id === activeId)
    if (!activeTask || targetColumnId === null) return
    if (activeTask.column_id === targetColumnId) {
      if (!overId.startsWith('column-')) return
      setDragTasks((current) =>
        current ? previewTaskAtEnd(current, activeId, targetColumnId) : current,
      )
      return
    }

    const translated = event.active.rect.current.translated
    const below = Boolean(
      translated && translated.top > event.over.rect.top + event.over.rect.height,
    )
    setDragTasks((current) =>
      current
        ? previewTaskMove(current, activeId, targetColumnId, overTask?.id ?? null, below)
        : current,
    )
  }

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    setActiveTaskId(null)
    setOverColumnId(null)
    if (!over) {
      setDragTasks(null)
      return
    }
    const overId = String(over.id)

    if (active.data.current?.type === 'column') {
      setDragTasks(null)
      if (!overId.startsWith('column-')) return
      const activeColumnId = Number(String(active.id).slice('column-'.length))
      const overColumnId = Number(overId.slice('column-'.length))
      if (activeColumnId === overColumnId) return
      const ids = orderedColumns.map((column) => column.id)
      const from = ids.indexOf(activeColumnId)
      const to = ids.indexOf(overColumnId)
      if (from < 0 || to < 0) return
      const next = [...ids]
      next.splice(from, 1)
      next.splice(to, 0, activeColumnId)
      reorderColumns.mutate(next, { onError: (error) => toast.error(error.message) })
      return
    }

    const taskId = Number(active.id)
    let result: ReturnType<typeof computeTaskDrop> = null
    if (overId === String(active.id)) {
      const original = tasks?.find((task) => task.id === taskId)
      const preview = dragTasks?.find((task) => task.id === taskId)
      if (dragTasks && original && preview && original.column_id !== preview.column_id) {
        const columnTasks = tasksInColumn(dragTasks, preview.column_id)
        const index = columnTasks.findIndex((task) => task.id === taskId)
        const withoutActive = columnTasks.filter((task) => task.id !== taskId)
        result = {
          id: taskId,
          columnId: preview.column_id,
          beforeTaskId: withoutActive[index]?.id,
          afterTaskId: withoutActive[index - 1]?.id,
          changed: true,
        }
      }
    } else {
      result = dragTasks ? computeTaskDrop(dragTasks, taskId, overId) : null
    }
    setDragTasks(null)
    if (!result || !result.changed) return
    moveTask.mutate(
      {
        id: result.id,
        column_id: result.columnId,
        before_task_id: result.beforeTaskId,
        after_task_id: result.afterTaskId,
      },
      { onError: (error) => toast.error(error.message) },
    )
  }

  function toggleTimer(task: Task) {
    if (running?.task_id === task.id) stopTime.mutate(task.id)
    else startTime.mutate(task.id)
  }

  function toggleSubtask(subtask: Task) {
    const done = Boolean(subtask.completed_at) || doneColumnIds.has(subtask.column_id)
    const target = orderedColumns.find((column) => (done ? !column.is_done : column.is_done))
    if (!target) {
      toast.error(done ? 'Add an open column to reopen subtasks.' : 'Add a done column to complete subtasks.')
      return
    }
    moveTask.mutate(
      { id: subtask.id, column_id: target.id },
      { onError: (error) => toast.error(error.message) },
    )
  }

  async function submitColumn() {
    const value = columnName.trim()
    if (!value || createColumn.isPending) return
    const nextPosition = orderedColumns.length
      ? Math.max(...orderedColumns.map((column) => column.position)) + 1
      : 1
    try {
      await createColumn.mutateAsync({ name: value, position: nextPosition, is_done: columnIsDone })
      setColumnName('')
      setColumnIsDone(false)
      setAddingColumn(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not create column')
    }
  }

  async function renameColumn(columnId: number, name: string) {
    const column = orderedColumns.find((item) => item.id === columnId)
    await updateColumn.mutateAsync({ id: columnId, name, is_done: column?.is_done ?? false })
  }

  function moveColumn(columnId: number, direction: 'left' | 'right') {
    const index = orderedColumns.findIndex((column) => column.id === columnId)
    const target = direction === 'left' ? index - 1 : index + 1
    if (index < 0 || target < 0 || target >= orderedColumns.length) return
    const ids = orderedColumns.map((column) => column.id)
    ;[ids[index], ids[target]] = [ids[target], ids[index]]
    reorderColumns.mutate(ids, { onError: (error) => toast.error(error.message) })
  }

  if (boardsQuery.isError) {
    return (
      <div className="flex flex-col gap-4">
        <header className="flex items-center justify-between gap-3">
          <h1 className="text-lg font-semibold">Boards</h1>
        </header>
        <LoadError
          title="Could not load boards"
          detail="Check that the API is reachable, then try again."
          onRetry={() => boardsQuery.refetch()}
        />
      </div>
    )
  }

  if (!boards) return <BoardSkeleton />

  if (!boards.length) {
    return (
      <div className="flex flex-col gap-4">
        <h1 className="text-lg font-semibold">Boards</h1>
        <div className="flex max-w-md flex-col items-start gap-2 rounded-md border border-dashed p-6">
          <p className="text-sm font-medium">No boards yet</p>
          <p className="text-xs text-muted-foreground">A board holds your columns and tasks.</p>
          <div className="flex w-full items-center gap-2">
            <Input
              autoFocus
              value={boardName}
              placeholder="Board name"
              aria-label="Board name"
              disabled={boardSaving}
              onChange={(event) => setBoardName(event.target.value)}
              onKeyDown={(event) => {
                if (event.nativeEvent.isComposing) return
                if (event.key === 'Enter') {
                  event.preventDefault()
                  void submitBoard()
                }
              }}
            />
            <Button size="sm" disabled={!boardName.trim() || boardSaving} onClick={() => void submitBoard()}>
              Create board
            </Button>
          </div>
        </div>
      </div>
    )
  }

  if (!boardId) return <BoardSkeleton />

  const summary = [
    `${openCount} open`,
    `${doneCount} done`,
    ...(overdueCount ? [`${overdueCount} overdue`] : []),
    ...(hasFilters ? [`${filteredTasks.length} shown`] : []),
  ].join(' · ')

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={collisionDetection}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
      onDragCancel={() => {
        setActiveTaskId(null)
        setOverColumnId(null)
        setDragTasks(null)
      }}
    >
      <div className="flex min-w-0 flex-col gap-4 pb-24">
        <h1 className="sr-only">{board?.name ?? 'Board'}</h1>
        <header className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex min-w-0 flex-col gap-1">
            <div className="flex min-w-0 items-center gap-1">
              {boardEditor ? (
                <>
                  <Input
                    autoFocus
                    value={boardName}
                    placeholder="Board name"
                    aria-label="Board name"
                    className="h-8 w-56"
                    disabled={boardSaving}
                    onChange={(event) => setBoardName(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.nativeEvent.isComposing) return
                      if (event.key === 'Enter') {
                        event.preventDefault()
                        void submitBoard()
                      }
                      if (event.key === 'Escape') {
                        event.preventDefault()
                        setBoardEditor(null)
                        setBoardName('')
                      }
                    }}
                  />
                  <Button size="xs" disabled={!boardName.trim() || boardSaving} onClick={() => void submitBoard()}>
                    Save
                  </Button>
                  <Button
                    size="xs"
                    variant="ghost"
                    onClick={() => {
                      setBoardEditor(null)
                      setBoardName('')
                    }}
                  >
                    Cancel
                  </Button>
                </>
              ) : (
                <>
                  <Select
                    items={(workspaces ?? []).map((item) => ({
                      value: String(item.id),
                      label: item.name,
                    }))}
                    value={activeWorkspaceId ? String(activeWorkspaceId) : undefined}
                    onValueChange={(value) => {
                      if (!value) return
                      setActiveWorkspaceId(Number(value))
                      setSelectedBoardId(undefined)
                    }}
                  >
                    <SelectTrigger size="sm" aria-label="Switch workspace" className="max-w-32">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent align="start">
                      {(workspaces ?? []).map((item) => (
                        <SelectItem key={item.id} value={String(item.id)}>
                          {item.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <Select
                    items={boards.map((item) => ({ value: String(item.id), label: item.name }))}
                    value={boardId ? String(boardId) : undefined}
                    onValueChange={(value) => value && setSelectedBoardId(Number(value))}
                  >
                    <SelectTrigger
                      aria-label="Switch board"
                      className="h-auto w-fit max-w-full border-0 bg-transparent py-0 pr-1 pl-0 text-lg font-semibold hover:bg-transparent focus-visible:ring-0 dark:bg-transparent dark:hover:bg-transparent [&_svg]:size-4"
                    >
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent align="start">
                      {boards.map((item) => (
                        <SelectItem key={item.id} value={String(item.id)}>
                          {item.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    className="text-muted-foreground"
                    aria-label="Rename board"
                    onClick={() => {
                      setBoardName(board?.name ?? '')
                      setBoardEditor('rename')
                    }}
                  >
                    <Pencil />
                  </Button>
                </>
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              {tasks ? summary : 'Loading tasks…'}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <div className="relative">
              <Search
                aria-hidden
                className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground"
              />
              <Input
                value={query}
                placeholder="Filter tasks"
                aria-label="Filter tasks"
                className="w-52 pl-7"
                onChange={(event) => setQuery(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Escape') setQuery('')
                }}
              />
            </div>
            <Select
              items={{
                all: 'All priorities',
                high: 'High',
                medium: 'Medium',
                low: 'Low',
              }}
              value={priority}
              onValueChange={(value) => value && setPriority(value)}
            >
              <SelectTrigger className="w-36" aria-label="Filter by priority">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All priorities</SelectItem>
                <SelectItem value="high">High</SelectItem>
                <SelectItem value="medium">Medium</SelectItem>
                <SelectItem value="low">Low</SelectItem>
              </SelectContent>
            </Select>
            <Button
              size="sm"
              disabled={!orderedColumns.length}
              onClick={() => orderedColumns[0] && setComposerColumnId(orderedColumns[0].id)}
            >
              <Plus />
              New task
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                setBoardName('')
                setBoardEditor('new')
              }}
            >
              <Plus />
              New board
            </Button>
          </div>
        </header>

        {columnsQuery.isError || tasksQuery.isError ? (
          <LoadError
            title="Could not load this board"
            detail="Check that the API is reachable, then try again."
            onRetry={() => {
              columnsQuery.refetch()
              tasksQuery.refetch()
            }}
          />
        ) : !columns || !tasks ? (
          <BoardSkeleton columns={columns?.length ?? 4} />
        ) : (
          <div className="flex w-full min-w-0 items-start gap-4 overflow-x-auto pb-2">
            <RemindersColumn />

            {!orderedColumns.length && !addingColumn && (
              <div className="flex w-full max-w-xl flex-col items-start gap-2 rounded-md border border-dashed p-6">
                <p className="text-sm font-medium">No columns yet</p>
                <p className="text-xs text-muted-foreground">
                  Add a column to start adding tasks to this board.
                </p>
                <Button size="sm" onClick={() => setAddingColumn(true)}>
                  <Plus />
                  Add column
                </Button>
              </div>
            )}

            <SortableContext
              items={orderedColumns.map((column) => `column-${column.id}`)}
              strategy={horizontalListSortingStrategy}
            >
              {orderedColumns.map((column, index) => (
                <BoardColumn
                  key={column.id}
                  column={column}
                  columns={orderedColumns}
                  tasks={tasksByColumn.get(column.id) ?? []}
                  subtasksByParent={subtasksByParent}
                  totalCount={totalByColumn.get(column.id) ?? 0}
                  hasFilters={hasFilters}
                  doneColumnIds={doneColumnIds}
                  highlighted={overColumnId === column.id}
                  adding={composerColumnId === column.id}
                  dragging={dragTasks !== null}
                  runningTaskId={running?.task_id ?? null}
                  canMoveLeft={index > 0}
                  canMoveRight={index < orderedColumns.length - 1}
                  onStartAdding={() => setComposerColumnId(column.id)}
                  onCancelAdding={() => setComposerColumnId(null)}
                  expandedTaskId={expandedTaskId}
                  focusField={fieldFocus?.field}
                  focusNonce={fieldFocus?.nonce}
                  onTaskClick={(task) =>
                    setExpandedTaskId((current) => (current === task.id ? null : task.id))
                  }
                  onOpenDetails={(task) => {
                    if (window.matchMedia('(max-width: 767px)').matches) {
                      navigate(`/tasks/${task.id}`)
                      return
                    }
                    setSearchParams({ task: String(task.id), panel: '1' })
                  }}
                  onCloseExpanded={() => setExpandedTaskId(null)}
                  onAddTask={async (title) => {
                    await createTask.mutateAsync({ board_id: boardId, column_id: column.id, title })
                  }}
                  onRename={(name) => renameColumn(column.id, name)}
                  onMove={(direction) => moveColumn(column.id, direction)}
                  onToggleTimer={toggleTimer}
                  onToggleSubtask={toggleSubtask}
                />
              ))}
            </SortableContext>

            {addingColumn ? (
              <section className="flex w-72 shrink-0 flex-col gap-2 rounded-md border border-dashed p-2">
                <Input
                  autoFocus
                  value={columnName}
                  placeholder="Column name"
                  aria-label="Column name"
                  disabled={createColumn.isPending}
                  onChange={(event) => setColumnName(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.nativeEvent.isComposing) return
                    if (event.key === 'Enter') {
                      event.preventDefault()
                      void submitColumn()
                    }
                    if (event.key === 'Escape') {
                      event.preventDefault()
                      setAddingColumn(false)
                      setColumnName('')
                      setColumnIsDone(false)
                    }
                  }}
                />
                <label className="flex items-center gap-2 text-xs text-muted-foreground">
                  <input
                    type="checkbox"
                    className="size-3.5 accent-primary"
                    checked={columnIsDone}
                    onChange={(event) => setColumnIsDone(event.target.checked)}
                  />
                  Marks tasks as done
                </label>
                <div className="flex justify-end gap-1">
                  <Button
                    size="xs"
                    variant="ghost"
                    onClick={() => {
                      setAddingColumn(false)
                      setColumnName('')
                      setColumnIsDone(false)
                    }}
                  >
                    Cancel
                  </Button>
                  <Button
                    size="xs"
                    disabled={!columnName.trim() || createColumn.isPending}
                    onClick={() => void submitColumn()}
                  >
                    Add column
                  </Button>
                </div>
              </section>
            ) : (
              orderedColumns.length > 0 && (
                <Button
                  size="sm"
                  variant="outline"
                  className="w-72 shrink-0 justify-start border-dashed text-muted-foreground"
                  onClick={() => setAddingColumn(true)}
                >
                  <Plus />
                  Add column
                </Button>
              )
            )}
          </div>
        )}
      </div>

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
              ['Enter / o', 'Expand or collapse'],
              ['x', 'Toggle done'],
              ['1 to 4', 'Priority'],
              ['t', 'Timer'],
              ['Backspace', 'Delete'],
              ['c', 'Quick add'],
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

      <DragOverlay dropAnimation={null}>
        {activeTask ? (
          <TaskCard
            task={activeTask}
            columns={orderedColumns}
            runningTaskId={running?.task_id ?? null}
            onToggleTimer={toggleTimer}
            subtasks={subtasksByParent.get(activeTask.id)}
            doneColumnIds={doneColumnIds}
            overlay
          />
        ) : null}
      </DragOverlay>
    </DndContext>
  )
}
