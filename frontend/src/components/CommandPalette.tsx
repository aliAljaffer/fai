import { Command } from 'cmdk'
import {
  BarChart3,
  Calendar,
  LayoutDashboard,
  List,
  NotebookPen,
  Plus,
  Search,
  Settings,
  SquareCheckBig,
} from 'lucide-react'
import { useHotkeys } from 'react-hotkeys-hook'
import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { useAllTasks, useBoards, useColumns, useDeleteTask, useMoveTask, useRestoreTasks, useSearch, useTask, useTasks } from '../api/hooks'
import { showUndo } from '../lib/undo'
import { useDebounced } from '../lib/useDebounced'
import { useUIStore } from '../store/ui'
import { useWorkspaceStore } from '../store/workspace'

const pages = [
  { label: 'Overview', path: '/', icon: LayoutDashboard },
  { label: 'Board', path: '/board', icon: SquareCheckBig },
  { label: 'List', path: '/list', icon: List },
  { label: 'Calendar', path: '/calendar', icon: Calendar },
  { label: 'Notes', path: '/notes', icon: NotebookPen },
  { label: 'Stats', path: '/stats', icon: BarChart3 },
  { label: 'Settings', path: '/settings', icon: Settings },
]

export function CommandPalette() {
  const { commandOpen, setCommandOpen, selectedBoardId, requestNewTask } = useUIStore()
  const navigate = useNavigate()
  const location = useLocation()
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: boards } = useBoards(workspaceId)
  const boardId =
    boards?.find((board) => board.id === selectedBoardId)?.id ?? boards?.[0]?.id
  const [allWorkspaces, setAllWorkspaces] = useState(false)
  const [query, setQuery] = useState('')
  const { data: scopedTasks } = useTasks(boardId)
  const { data: allTasks } = useAllTasks(allWorkspaces)
  const tasks = allWorkspaces ? allTasks : scopedTasks
  const debouncedQuery = useDebounced(query.trim(), 200)
  const { data: search } = useSearch(
    debouncedQuery,
    allWorkspaces ? undefined : workspaceId,
  )
  const notes = search?.notes ?? []

  const focusedId = Number(new URLSearchParams(location.search).get('task'))
  const focusedValid = Number.isFinite(focusedId) && focusedId > 0
  const { data: focusedTask } = useTask(focusedValid ? focusedId : undefined)
  const { data: focusedColumns } = useColumns(focusedTask?.board_id)
  const moveTask = useMoveTask(focusedTask?.board_id)
  const deleteTask = useDeleteTask(focusedTask?.board_id)
  const restoreTasks = useRestoreTasks(focusedTask?.board_id)
  const taskOpen = new URLSearchParams(location.search).has('task')

  function close() {
    setCommandOpen(false)
    setQuery('')
  }

  function go(path: string) {
    close()
    navigate(path)
  }

  function openTask(id: number) {
    close()
    const params = new URLSearchParams(location.search)
    params.set('task', String(id))
    navigate({ pathname: location.pathname, search: params.toString() })
  }

  function newTask() {
    close()
    navigate('/board')
    requestNewTask()
  }

  useHotkeys('mod+k', () => setCommandOpen(!commandOpen), {
    enableOnFormTags: true,
    preventDefault: true,
  })
  useHotkeys('n', newTask, { preventDefault: true, enabled: !taskOpen && !commandOpen })

  return (
    <Dialog
      open={commandOpen}
      onOpenChange={(open) => {
        setCommandOpen(open)
        if (!open) setQuery('')
      }}
    >
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-hidden p-0 sm:max-w-lg" showCloseButton={false}>
        <DialogTitle className="sr-only">Command palette</DialogTitle>
        <Command
          label="Command palette"
          className="flex min-h-0 flex-col overflow-hidden"
          value={query}
          onValueChange={setQuery}
        >
          <div className="flex items-center gap-2 border-b px-3">
            <Search className="size-4 shrink-0 text-muted-foreground" />
            <Command.Input
              autoFocus
              placeholder="Jump to a page, task, note, or action…"
              className="h-10 w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
            />
          </div>
          <Command.List className="max-h-[calc(100dvh-8rem)] min-h-0 overflow-y-auto overscroll-contain p-2">
            <Command.Empty className="py-6 text-center text-sm text-muted-foreground">
              No results.
            </Command.Empty>

            {focusedTask && (
              <>
                <Command.Group
                  heading={`This task: ${focusedTask.title}`}
                  className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-muted-foreground"
                >
                  <Command.Item
                    value="Open task details"
                    onSelect={() => {
                      close()
                      const params = new URLSearchParams(location.search)
                      params.set('task', String(focusedTask.id))
                      params.set('panel', '1')
                      navigate({ pathname: location.pathname, search: params.toString() })
                    }}
                    className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm data-[selected=true]:bg-muted"
                  >
                    <SquareCheckBig className="size-4 text-muted-foreground" />
                    Open details
                  </Command.Item>
                  {(focusedColumns ?? []).map((column) => (
                    <Command.Item
                      key={column.id}
                      value={`Move to ${column.name}`}
                      onSelect={() => {
                        close()
                        moveTask.mutate({ id: focusedTask.id, column_id: column.id })
                      }}
                      className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm data-[selected=true]:bg-muted"
                    >
                      <SquareCheckBig className="size-4 text-muted-foreground" />
                      Move to {column.name}
                    </Command.Item>
                  ))}
                  <Command.Item
                    value="Delete task"
                    onSelect={() => {
                      close()
                      const title = focusedTask.title
                      deleteTask.mutate(focusedTask.id, {
                        onSuccess: () =>
                          showUndo(`Deleted "${title}"`, () =>
                            restoreTasks.mutateAsync([focusedTask.id]).then(() => undefined),
                          ),
                      })
                    }}
                    className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm data-[selected=true]:bg-muted"
                  >
                    <SquareCheckBig className="size-4 text-muted-foreground" />
                    Delete task
                  </Command.Item>
                </Command.Group>
                <Command.Separator className="my-1 h-px bg-border" />
              </>
            )}

            <Command.Group
              heading="Go to"
              className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-muted-foreground"
            >
              {pages.map(({ label, path, icon: Icon }) => (
                <Command.Item
                  key={path}
                  value={label}
                  onSelect={() => go(path)}
                  className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm data-[selected=true]:bg-muted"
                >
                  <Icon className="size-4 text-muted-foreground" />
                  {label}
                </Command.Item>
              ))}
            </Command.Group>

            <Command.Separator className="my-1 h-px bg-border" />

            <Command.Group
              heading="Actions"
              className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-muted-foreground"
            >
              <Command.Item
                value="Toggle all workspaces"
                onSelect={() => setAllWorkspaces((current) => !current)}
                className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm data-[selected=true]:bg-muted"
              >
                <SquareCheckBig className="size-4 text-muted-foreground" />
                {allWorkspaces ? 'Show active workspace only' : 'Show tasks from all workspaces'}
              </Command.Item>
              <Command.Item
                value="New task"
                onSelect={newTask}
                className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm data-[selected=true]:bg-muted"
              >
                <Plus className="size-4 text-muted-foreground" />
                New task
                <kbd className="ml-auto rounded border px-1.5 text-xs text-muted-foreground">n</kbd>
              </Command.Item>
            </Command.Group>

            {!!tasks?.length && (
              <>
                <Command.Separator className="my-1 h-px bg-border" />
                <Command.Group
                  heading="Tasks"
                  className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-muted-foreground"
                >
                  {tasks.map((task) => (
                    <Command.Item
                      key={task.id}
                      value={`${task.title} ${task.ref ?? ''} ${task.id}`}
                      onSelect={() => openTask(task.id)}
                      className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm data-[selected=true]:bg-muted"
                    >
                      <SquareCheckBig className="size-4 shrink-0 text-muted-foreground" />
                      <span className="truncate">{task.title}</span>
                    </Command.Item>
                  ))}
                </Command.Group>
              </>
            )}

            {!!notes.length && (
              <>
                <Command.Separator className="my-1 h-px bg-border" />
                <Command.Group
                  heading="Notes"
                  className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-muted-foreground"
                >
                  {notes.map((note) => (
                    <Command.Item
                      key={note.id}
                      value={`${note.title} ${note.body_md}`}
                      onSelect={() => go(`/notes/${note.id}`)}
                      className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm data-[selected=true]:bg-muted"
                    >
                      <NotebookPen className="size-4 shrink-0 text-muted-foreground" />
                      <span className="truncate">{note.title}</span>
                    </Command.Item>
                  ))}
                </Command.Group>
              </>
            )}
          </Command.List>
        </Command>
      </DialogContent>
    </Dialog>
  )
}
