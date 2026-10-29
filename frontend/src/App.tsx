import { NavLink, Route, Routes } from 'react-router-dom'
import { useEffect } from 'react'
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
import { NoteEditor } from './pages/NoteEditor'
import { Search } from './pages/Search'
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
import { useUIStore } from './store/ui'
import { useWorkspaceStore } from './store/workspace'
import { useWorkspaces } from './api/hooks'

export function App() {
  const setCommandOpen = useUIStore((state) => state.setCommandOpen)
  const setSelectedBoardId = useUIStore((state) => state.setSelectedBoardId)
  const { data: workspaces } = useWorkspaces()
  const { activeWorkspaceId, setActiveWorkspaceId } = useWorkspaceStore()
  const storedWorkspace = workspaces?.find((workspace) => workspace.id === activeWorkspaceId)
  const activeWorkspace = storedWorkspace ?? workspaces?.[0]

  useEffect(() => {
    if (!workspaces?.length || storedWorkspace) return
    setActiveWorkspaceId(workspaces[0].id)
  }, [workspaces, storedWorkspace, setActiveWorkspaceId])

  return (
    <div className="flex min-h-screen flex-col">
      <ThemeController />
      <nav className="flex items-center gap-5 border-b px-6 py-3">
        <span className="mr-2 flex items-center gap-2">
          <AppIcon className="h-5 w-5" />
          <span className="font-bold">fai</span>
        </span>
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
            <SelectTrigger size="sm" aria-label="Switch workspace" className="mr-1">
              <SelectValue />
              <span className="rounded border px-1 font-mono text-[0.65rem] text-muted-foreground">
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
        <NavLink
          to="/"
          end
          className={({ isActive }) =>
            cn('text-muted-foreground hover:text-foreground', isActive && 'font-semibold text-foreground')
          }
        >
          Board
        </NavLink>
        <NavLink
          to="/list"
          className={({ isActive }) =>
            cn('text-muted-foreground hover:text-foreground', isActive && 'font-semibold text-foreground')
          }
        >
          List
        </NavLink>
        <NavLink
          to="/calendar"
          className={({ isActive }) =>
            cn('text-muted-foreground hover:text-foreground', isActive && 'font-semibold text-foreground')
          }
        >
          Calendar
        </NavLink>
        <NavLink
          to="/notes"
          className={({ isActive }) =>
            cn('text-muted-foreground hover:text-foreground', isActive && 'font-semibold text-foreground')
          }
        >
          Notes
        </NavLink>
        <NavLink
          to="/chat"
          className={({ isActive }) =>
            cn('text-muted-foreground hover:text-foreground', isActive && 'font-semibold text-foreground')
          }
        >
          Chat
        </NavLink>
        <NavLink
          to="/stats"
          className={({ isActive }) =>
            cn('text-muted-foreground hover:text-foreground', isActive && 'font-semibold text-foreground')
          }
        >
          Stats
        </NavLink>
        <TimerWidget />
        <ThemeToggle />
        <UpdateIndicator />
        <Button
          variant="ghost"
          size="sm"
          className="ml-auto text-muted-foreground"
          aria-label="Open command palette"
          onClick={() => setCommandOpen(true)}
        >
          <kbd className="rounded border px-1 text-[0.65rem]">⌘K</kbd>
        </Button>
        <NavLink
          to="/search"
          className={({ isActive }) =>
            cn('text-muted-foreground hover:text-foreground', isActive && 'font-semibold text-foreground')
          }
        >
          Search
        </NavLink>
        <NavLink
          to="/settings"
          className={({ isActive }) =>
            cn('text-muted-foreground hover:text-foreground', isActive && 'font-semibold text-foreground')
          }
        >
          Settings
        </NavLink>
      </nav>
      <PendingOutcomesBanner />
      <main className="flex flex-1 flex-col p-6">
        <Routes>
          <Route path="/" element={<Board />} />
          <Route path="/list" element={<List />} />
          <Route path="/calendar" element={<Calendar />} />
          <Route path="/chat" element={<Chat />} />
          <Route path="/notes" element={<Notes />} />
          <Route path="/notes/:id" element={<NoteEditor />} />
          <Route path="/stats" element={<Stats />} />
          <Route path="/search" element={<Search />} />
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
