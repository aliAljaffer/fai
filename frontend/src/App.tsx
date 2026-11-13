import { Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { lazy, Suspense, useEffect, useRef } from 'react'
import { useHotkeys } from 'react-hotkeys-hook'
import { Toaster } from '@/components/ui/sonner'
import { Skeleton } from '@/components/ui/skeleton'
import { AuthCallback } from './pages/AuthCallback'
import { Overview } from './pages/Overview'
import { AppShell } from './components/layout/AppShell'
import { titleForPath } from './components/layout/nav'
import { useActiveWorkspace } from './components/layout/useActiveWorkspace'
import { useViewTransition } from './components/layout/useViewTransition'
import { CommandPalette } from './components/CommandPalette'
import { ThemeController } from './components/ThemeController'
import { TaskDetailSheet } from './components/board/TaskDetailSheet'
import { useUIStore } from './store/ui'
import { useWorkspaceStore } from './store/workspace'
import { useLiveUpdates } from './lib/live'
import { undoLatest } from './lib/undo'

const lazyPage = <K extends string>(load: () => Promise<Record<K, React.ComponentType>>, name: K) =>
  lazy(() => load().then((m) => ({ default: m[name] })))

const Board = lazyPage(() => import('./pages/Board'), 'Board')
const Calendar = lazyPage(() => import('./pages/Calendar'), 'Calendar')
const Chat = lazyPage(() => import('./pages/Chat'), 'Chat')
const List = lazyPage(() => import('./pages/List'), 'List')
const Notes = lazyPage(() => import('./pages/Notes'), 'Notes')
const NoteEditor = lazyPage(() => import('./pages/NoteEditor'), 'NoteEditor')
const Settings = lazyPage(() => import('./pages/Settings'), 'Settings')
const Stats = lazyPage(() => import('./pages/Stats'), 'Stats')
const TaskDetailPage = lazyPage(() => import('./pages/TaskDetailPage'), 'TaskDetailPage')

function RouteFallback() {
  return (
    <div className="flex flex-col gap-3" role="status" aria-label="Loading page">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-40 w-full" />
    </div>
  )
}

export function App() {
  const setCommandOpen = useUIStore((state) => state.setCommandOpen)
  const requestNewTask = useUIStore((state) => state.requestNewTask)
  useLiveUpdates()
  const setActiveWorkspaceId = useWorkspaceStore((state) => state.setActiveWorkspaceId)
  const { workspaces, stored: storedWorkspace } = useActiveWorkspace()
  const location = useLocation()
  const navigate = useNavigate()
  const viewTransition = useViewTransition()
  const announcementRef = useRef<HTMLDivElement>(null)

  useHotkeys('mod+z', (event) => {
    event.preventDefault()
    undoLatest()
  })

  useEffect(() => {
    const bridge = window.fai
    if (!bridge?.onMenuCommand) return
    return bridge.onMenuCommand((command) => {
      if (command === 'new-task') {
        navigate('/board', { viewTransition })
        requestNewTask()
      } else if (command === 'command-palette') {
        setCommandOpen(true)
      } else if (command === 'settings') {
        navigate('/settings', { viewTransition })
      } else if (command === 'undo') {
        undoLatest()
      }
    })
  }, [navigate, requestNewTask, setCommandOpen, viewTransition])

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
    <>
      <ThemeController />
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-50 focus:rounded-md focus:bg-background focus:px-3 focus:py-2 focus:text-sm focus:font-medium focus:ring-3 focus:ring-ring"
      >
        Skip to content
      </a>
      <div ref={announcementRef} role="status" aria-live="polite" className="sr-only" />
      <AppShell>
        <Suspense fallback={<RouteFallback />}>
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
        </Suspense>
      </AppShell>
      <TaskDetailSheet />
      <CommandPalette />
      <Toaster />
    </>
  )
}
