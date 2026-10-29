import { NavLink, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { useEffect, useRef } from 'react'
import { useHotkeys } from 'react-hotkeys-hook'
import { Search } from 'lucide-react'
import { Toaster } from '@/components/ui/sonner'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { AuthCallback } from './pages/AuthCallback'
import { Board } from './pages/Board'
import { Calendar } from './pages/Calendar'
import { Chat } from './pages/Chat'
import { List } from './pages/List'
import { Notes } from './pages/Notes'
import { Overview } from './pages/Overview'
import { NoteEditor } from './pages/NoteEditor'
import { Settings } from './pages/Settings'
import { Stats } from './pages/Stats'
import { CommandPalette } from './components/CommandPalette'
import { AppIcon } from './components/Logo'
import { ThemeController } from './components/ThemeController'
import { ThemeToggle } from './components/ThemeToggle'
import { PendingOutcomesBanner } from './components/calendar/PendingOutcomesBanner'
import { TimerWidget } from './components/TimerWidget'
import { UpdateIndicator } from './components/UpdateIndicator'
import { TaskDetailSheet } from './components/board/TaskDetailSheet'
import { TaskDetailPage } from './pages/TaskDetailPage'
import { useUIStore } from './store/ui'
import { useWorkspaceStore } from './store/workspace'
import { useWorkspaces } from './api/hooks'
import { useLiveUpdates } from './lib/live'
import { undoLatest } from './lib/undo'

const NAV_ITEMS = [
  { to: '/', label: 'Overview', end: true },
  { to: '/board', label: 'Board' },
  { to: '/list', label: 'List' },
  { to: '/calendar', label: 'Calendar' },
  { to: '/notes', label: 'Notes' },
  { to: '/chat', label: 'Chat' },
  { to: '/stats', label: 'Stats' },
  { to: '/settings', label: 'Settings' },
]

const ROUTE_TITLES: Record<string, string> = {
  '/': 'Overview',
  '/board': 'Board',
  '/list': 'List',
  '/calendar': 'Calendar',
  '/notes': 'Notes',
  '/chat': 'Chat',
  '/stats': 'Stats',
  '/settings': 'Settings',
}

function titleForPath(pathname: string) {
  if (pathname.startsWith('/notes/')) return 'Note'
  if (pathname.startsWith('/tasks/')) return 'Task'
  return ROUTE_TITLES[pathname]
}

const isMac =
  typeof navigator !== 'undefined' && /Mac|iPhone|iPad|iPod/.test(navigator.userAgent)
const commandKey = isMac ? '⌘K' : 'Ctrl K'

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    'text-muted-foreground hover:text-foreground',
    isActive && 'font-semibold text-foreground',
  )

export function App() {
  const setCommandOpen = useUIStore((state) => state.setCommandOpen)
  const requestNewTask = useUIStore((state) => state.requestNewTask)
  const setSelectedBoardId = useUIStore((state) => state.setSelectedBoardId)
  const { data: workspaces } = useWorkspaces()
  useLiveUpdates()
  const { activeWorkspaceId, setActiveWorkspaceId } = useWorkspaceStore()
  const location = useLocation()
  const navigate = useNavigate()
  const announcementRef = useRef<HTMLDivElement>(null)
  const storedWorkspace = workspaces?.find((workspace) => workspace.id === activeWorkspaceId)
  const activeWorkspace = storedWorkspace ?? workspaces?.[0]

  useHotkeys('mod+z', (event) => {
    event.preventDefault()
    undoLatest()
  })

  useEffect(() => {
    const bridge = window.fai
    if (!bridge?.onMenuCommand) return
    return bridge.onMenuCommand((command) => {
      if (command === 'new-task') {
        navigate('/board')
        requestNewTask()
      } else if (command === 'command-palette') {
        setCommandOpen(true)
      } else if (command === 'settings') {
        navigate('/settings')
      } else if (command === 'undo') {
        undoLatest()
      }
    })
  }, [navigate, requestNewTask, setCommandOpen])

  useEffect(() => {
    if (!workspaces?.length || storedWorkspace) return
    setActiveWorkspaceId(workspaces[0].id)
  }, [workspaces, storedWorkspace, setActiveWorkspaceId])

  useEffect(() => {
    const title = titleForPath(location.pathname)
    document.title = title ? `${title} · fai` : 'fai'
    if (announcementRef.current) {
      announcementRef.current.textContent = title ? `${title} page` : ''
    }
  }, [location.pathname])

  return (
    <div className="flex min-h-screen flex-col">
      <ThemeController />
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-50 focus:rounded-md focus:bg-background focus:px-3 focus:py-2 focus:text-sm focus:font-medium focus:ring-3 focus:ring-ring"
      >
        Skip to content
      </a>
      <div ref={announcementRef} role="status" aria-live="polite" className="sr-only" />
      <nav
        aria-label="Primary"
        className="flex flex-wrap items-center gap-3 border-b px-4 py-3 sm:px-6"
      >
        <NavLink
          to="/"
          end
          aria-label="fai home"
          className="mr-1 flex items-center gap-2 rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
        >
          <AppIcon className="h-5 w-5" />
          <span className="font-bold">fai</span>
        </NavLink>
        {!!workspaces?.length && (
          <Select
            items={workspaces.map((workspace) => ({
              value: String(workspace.id),
              label: workspace.name,
            }))}
            value={activeWorkspace ? String(activeWorkspace.id) : undefined}
            onValueChange={(value) => {
              if (!value) return
              setActiveWorkspaceId(Number(value))
              setSelectedBoardId(undefined)
            }}
          >
            <SelectTrigger
              size="sm"
              aria-label="Switch workspace"
              className="mr-1 max-w-32 sm:max-w-44"
            >
              <SelectValue />
              <span className="hidden rounded border px-1 font-mono text-[0.65rem] text-muted-foreground sm:inline">
                {activeWorkspace?.tag}
              </span>
            </SelectTrigger>
            <SelectContent align="start">
              {workspaces.map((workspace) => (
                <SelectItem key={workspace.id} value={String(workspace.id)}>
                  {workspace.name}
                  <span className="font-mono text-[0.65rem] text-muted-foreground">
                    {workspace.tag}
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        <div className="hidden items-center gap-4 lg:flex">
          {NAV_ITEMS.map((item) => (
            <NavLink key={item.to} to={item.to} end={item.end} className={navLinkClass}>
              {item.label}
            </NavLink>
          ))}
        </div>
        <div className="ml-auto flex items-center gap-1.5 sm:gap-2">
          <TimerWidget />
          <ThemeToggle />
          <UpdateIndicator />
          <Button
            variant="ghost"
            size="sm"
            className="text-muted-foreground"
            aria-label="Open command palette"
            onClick={() => setCommandOpen(true)}
          >
            <Search className="size-4 sm:hidden" />
            <kbd className="hidden rounded border px-1 text-[0.65rem] sm:inline">
              {commandKey}
            </kbd>
          </Button>
        </div>
      </nav>
      <PendingOutcomesBanner />
      <nav
        aria-label="Mobile"
        className="fixed inset-x-0 bottom-0 z-40 flex overflow-x-auto border-t bg-background/95 backdrop-blur lg:hidden"
      >
        {NAV_ITEMS.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end}
            className={({ isActive }) =>
              cn(
                'flex min-w-16 flex-1 flex-col items-center justify-center px-2 py-2 text-[0.65rem] text-muted-foreground',
                isActive && 'font-semibold text-foreground',
              )
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>
      <main id="main-content" tabIndex={-1} className="flex flex-1 flex-col p-4 pb-20 outline-none sm:p-6 lg:pb-6">
        <Routes>
          <Route path="/" element={<Overview />} />
          <Route path="/board" element={<Board />} />
          <Route path="/list" element={<List />} />
          <Route path="/calendar" element={<Calendar />} />
          <Route path="/chat" element={<Chat />} />
          <Route path="/notes" element={<Notes />} />
          <Route path="/notes/:id" element={<NoteEditor />} />
          <Route path="/tasks/:id" element={<TaskDetailPage />} />
          <Route path="/stats" element={<Stats />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="/auth/callback" element={<AuthCallback />} />
        </Routes>
      </main>
      <TaskDetailSheet />
      <CommandPalette />
      <Toaster />
    </div>
  )
}
