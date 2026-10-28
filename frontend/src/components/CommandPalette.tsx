import { Command } from 'cmdk'
import {
  BarChart3,
  Bell,
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
import { useLocation, useNavigate } from 'react-router-dom'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { useBoards, useTasks } from '../api/hooks'
import { useUIStore } from '../store/ui'

const pages = [
  { label: 'Board', path: '/', icon: LayoutDashboard },
  { label: 'List', path: '/list', icon: List },
  { label: 'Calendar', path: '/calendar', icon: Calendar },
  { label: 'Notes', path: '/notes', icon: NotebookPen },
  { label: 'Stats', path: '/stats', icon: BarChart3 },
  { label: 'Reminders', path: '/reminders', icon: Bell },
  { label: 'Search', path: '/search', icon: Search },
  { label: 'Settings', path: '/settings', icon: Settings },
]

export function CommandPalette() {
  const { commandOpen, setCommandOpen, selectedBoardId, requestNewTask } = useUIStore()
  const navigate = useNavigate()
  const location = useLocation()
  const { data: boards } = useBoards()
  const boardId = selectedBoardId ?? boards?.[0]?.id
  const { data: tasks } = useTasks(boardId)

  const taskOpen = new URLSearchParams(location.search).has('task')

  function close() {
    setCommandOpen(false)
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
    navigate('/')
    requestNewTask()
  }

  useHotkeys('mod+k', () => setCommandOpen(!commandOpen), {
    enableOnFormTags: true,
    preventDefault: true,
  })
  useHotkeys('n', newTask, { preventDefault: true, enabled: !taskOpen && !commandOpen })

  return (
    <Dialog open={commandOpen} onOpenChange={setCommandOpen}>
      <DialogContent className="overflow-hidden p-0 sm:max-w-lg" showCloseButton={false}>
        <DialogTitle className="sr-only">Command palette</DialogTitle>
        <Command label="Command palette" className="flex flex-col">
          <div className="flex items-center gap-2 border-b px-3">
            <Search className="size-4 shrink-0 text-muted-foreground" />
            <Command.Input
              autoFocus
              placeholder="Jump to a page, task, or action…"
              className="h-10 w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
            />
          </div>
          <Command.List className="max-h-80 overflow-y-auto p-2">
            <Command.Empty className="py-6 text-center text-sm text-muted-foreground">
              No results.
            </Command.Empty>

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
                      value={`${task.title} ${task.id}`}
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
          </Command.List>
        </Command>
      </DialogContent>
    </Dialog>
  )
}
