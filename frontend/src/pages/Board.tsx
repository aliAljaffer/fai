import {
  DndContext,
  DragOverlay,
  PointerSensor,
  closestCenter,
  pointerWithin,
  useSensor,
  useSensors,
  type CollisionDetection,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { Plus, Search } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
  useRunningTime,
  useStartTime,
  useStopTime,
  useTasks,
} from '../api/hooks'
import { BoardColumn } from '../components/board/BoardColumn'
import { TaskCard } from '../components/board/TaskCard'
import { isPastDue } from '../lib/format'
import { useUIStore } from '../store/ui'
import type { Board as BoardType, Task } from '../api/types'

const collisionDetection: CollisionDetection = (args) => {
  const hits = pointerWithin(args)
  const taskHits = hits.filter((hit) => hit.data?.type === 'task')
  const candidates = taskHits.length ? taskHits : hits
  if (!candidates.length) return closestCenter(args)
  const ids = new Set(candidates.map((hit) => hit.id))
  return closestCenter({
    ...args,
    droppableContainers: args.droppableContainers.filter((container) => ids.has(container.id)),
  })
}

function NewBoardDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (board: BoardType) => void
}) {
  const [name, setName] = useState('')
  const createBoard = useCreateBoard()

  async function submit() {
    const value = name.trim()
    if (!value || createBoard.isPending) return
    try {
      const board = await createBoard.mutateAsync(value)
      setName('')
      onCreated(board)
      onOpenChange(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not create board')
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next)
        if (!next) setName('')
      }}
    >
      <DialogContent className="rounded-lg">
        <DialogHeader>
          <DialogTitle>New board</DialogTitle>
          <DialogDescription>A board holds columns and the tasks inside them.</DialogDescription>
        </DialogHeader>
        <Input
          autoFocus
          value={name}
          placeholder="Board name"
          aria-label="Board name"
          disabled={createBoard.isPending}
          onChange={(event) => setName(event.target.value)}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing) return
            if (event.key === 'Enter') {
              event.preventDefault()
              void submit()
            }
          }}
        />
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={!name.trim() || createBoard.isPending} onClick={() => void submit()}>
            Create board
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
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
  const boardsQuery = useBoards()
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
  const moveTask = useMoveTask(boardId)
  const { data: running } = useRunningTime()
  const startTime = useStartTime()
  const stopTime = useStopTime()
  const [, setSearchParams] = useSearchParams()

  const [query, setQuery] = useState('')
  const [priority, setPriority] = useState('all')
  const [activeTaskId, setActiveTaskId] = useState<number | null>(null)
  const [composerColumnId, setComposerColumnId] = useState<number | null>(null)
  const [newBoardOpen, setNewBoardOpen] = useState(false)
  const [addingColumn, setAddingColumn] = useState(false)
  const [columnName, setColumnName] = useState('')
  const [columnIsDone, setColumnIsDone] = useState(false)
  const handledNewTaskRequest = useRef(0)

  const orderedColumns = useMemo(
    () => [...(columns ?? [])].sort((a, b) => a.position - b.position),
    [columns],
  )

  useEffect(() => {
    setComposerColumnId(null)
    setQuery('')
    setPriority('all')
    setAddingColumn(false)
  }, [boardId])

  useEffect(() => {
    if (newTaskRequest === handledNewTaskRequest.current) return
    if (!orderedColumns[0]) return
    handledNewTaskRequest.current = newTaskRequest
    setComposerColumnId(orderedColumns[0].id)
  }, [newTaskRequest, orderedColumns])

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }))

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
    for (const task of filteredTasks) {
      const list = map.get(task.column_id) ?? []
      list.push(task)
      map.set(task.column_id, list)
    }
    for (const list of map.values()) list.sort((a, b) => a.position - b.position)
    return map
  }, [filteredTasks])

  const totalByColumn = useMemo(() => {
    const map = new Map<number, number>()
    for (const task of tasks ?? []) {
      map.set(task.column_id, (map.get(task.column_id) ?? 0) + 1)
    }
    return map
  }, [tasks])

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

  function handleDragStart(event: DragStartEvent) {
    setActiveTaskId(Number(event.active.id))
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveTaskId(null)
    const { active, over } = event
    if (!over) return

    const taskId = Number(active.id)
    const overId = String(over.id)

    let targetColumnId: number
    let beforeTaskId: number | undefined

    if (overId.startsWith('column-')) {
      targetColumnId = Number(overId.replace('column-', ''))
    } else {
      const overTask = tasks?.find((task) => task.id === Number(overId))
      if (!overTask) return
      targetColumnId = overTask.column_id
      beforeTaskId = overTask.id
    }

    const moving = tasks?.find((task) => task.id === taskId)
    if (moving && moving.column_id === targetColumnId && moving.id === beforeTaskId) return

    moveTask.mutate(
      { id: taskId, column_id: targetColumnId, before_task_id: beforeTaskId },
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

  function startFirstTask() {
    if (orderedColumns[0]) setComposerColumnId(orderedColumns[0].id)
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
        <header className="flex flex-wrap items-center justify-between gap-3">
          <h1 className="text-lg font-semibold">Boards</h1>
          <Button size="sm" onClick={() => setNewBoardOpen(true)}>
            <Plus />
            New board
          </Button>
        </header>
        <div className="flex flex-col items-start gap-2 rounded-md border border-dashed p-6">
          <p className="text-sm font-medium">No boards yet</p>
          <p className="text-xs text-muted-foreground">
            A board holds your columns and tasks.
          </p>
          <Button size="sm" onClick={() => setNewBoardOpen(true)}>
            Create a board
          </Button>
        </div>
        <NewBoardDialog
          open={newBoardOpen}
          onOpenChange={setNewBoardOpen}
          onCreated={(created) => setSelectedBoardId(created.id)}
        />
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
      onDragEnd={handleDragEnd}
      onDragCancel={() => setActiveTaskId(null)}
    >
      <div className="flex flex-col gap-4">
        <header className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex min-w-0 flex-col gap-1">
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
            <Button size="sm" disabled={!orderedColumns.length} onClick={startFirstTask}>
              <Plus />
              New task
            </Button>
            <Button size="sm" variant="outline" onClick={() => setNewBoardOpen(true)}>
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
          <div className="flex items-start gap-4 overflow-x-auto pb-2">
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

            {orderedColumns.map((column) => (
              <BoardColumn
                key={column.id}
                column={column}
                tasks={tasksByColumn.get(column.id) ?? []}
                subtasksByParent={subtasksByParent}
                totalCount={totalByColumn.get(column.id) ?? 0}
                hasFilters={hasFilters}
                doneColumnIds={doneColumnIds}
                adding={composerColumnId === column.id}
                runningTaskId={running?.task_id ?? null}
                onStartAdding={() => setComposerColumnId(column.id)}
                onCancelAdding={() => setComposerColumnId(null)}
                onTaskClick={(task) => setSearchParams({ task: String(task.id) })}
                onAddTask={async (title) => {
                  await createTask.mutateAsync({ board_id: boardId, column_id: column.id, title })
                }}
                onToggleTimer={toggleTimer}
                onToggleSubtask={toggleSubtask}
              />
            ))}

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

      <DragOverlay dropAnimation={null}>
        {activeTask ? (
          <TaskCard
            task={activeTask}
            runningTaskId={running?.task_id ?? null}
            onToggleTimer={toggleTimer}
            subtasks={subtasksByParent.get(activeTask.id)}
            doneColumnIds={doneColumnIds}
            overlay
          />
        ) : null}
      </DragOverlay>

      <NewBoardDialog
        open={newBoardOpen}
        onOpenChange={setNewBoardOpen}
        onCreated={(created) => setSelectedBoardId(created.id)}
      />
    </DndContext>
  )
}
