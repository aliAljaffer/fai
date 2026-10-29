import { NavLink, Route, Routes, useLocation } from 'react-router-dom'
import { useEffect, useRef, useState } from 'react'
import { Menu, Search } from 'lucide-react'
import { Toaster } from '@/components/ui/sonner'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { cn } from '@/lib/utils'
import { AuthCallback } from './pages/AuthCallback'
import { Board } from './pages/Board'
import { Calendar } from './pages/Calendar'
import { Chat } from './pages/Chat'
import { List } from './pages/List'
import { Notes } from './pages/Notes'
import { NoteEditor } from './pages/NoteEditor'
import { Search as SearchPage } from './pages/Search'
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

const NAV_ITEMS = [
  { to: '/', label: 'Board', end: true },
  { to: '/list', label: 'List' },
  { to: '/calendar', label: 'Calendar' },
  { to: '/notes', label: 'Notes' },
  { to: '/chat', label: 'Chat' },
  { to: '/stats', label: 'Stats' },
  { to: '/search', label: 'Search' },
  { to: '/settings', label: 'Settings' },
]

const ROUTE_TITLES: Record<string, string> = {
  '/': 'Board',
  '/list': 'List',
  '/calendar': 'Calendar',
  '/notes': 'Notes',
  '/chat': 'Chat',
  '/stats': 'Stats',
  '/search': 'Search',
  '/settings': 'Settings',
}

function titleForPath(pathname: string) {
  if (pathname.startsWith('/notes/')) return 'Note'
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

const mobileNavLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    'rounded-md px-2 py-2 text-sm text-muted-foreground hover:bg-muted hover:text-foreground',
    isActive && 'bg-muted font-semibold text-foreground',
  )

export function App() {
  const setCommandOpen = useUIStore((state) => state.setCommandOpen)
  const setSelectedBoardId = useUIStore((state) => state.setSelectedBoardId)
  const { data: workspaces } = useWorkspaces()
  const { activeWorkspaceId, setActiveWorkspaceId } = useWorkspaceStore()
  const location = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)
  const announcementRef = useRef<HTMLDivElement>(null)
  const storedWorkspace = workspaces?.find((workspace) => workspace.id === activeWorkspaceId)
  const activeWorkspace = storedWorkspace ?? workspaces?.[0]

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
        <span className="mr-1 flex items-center gap-2">
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
          <Button
            variant="ghost"
            size="icon-sm"
            className="lg:hidden"
            aria-label="Open menu"
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen(true)}
          >
            <Menu />
          </Button>
        </div>
      </nav>
      <Sheet open={menuOpen} onOpenChange={setMenuOpen}>
        <SheetContent side="right" className="w-72">
          <SheetHeader>
            <SheetTitle>Menu</SheetTitle>
          </SheetHeader>
          <nav aria-label="Mobile" className="flex flex-col gap-1 px-4 pb-4">
            {NAV_ITEMS.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={mobileNavLinkClass}
                onClick={() => setMenuOpen(false)}
              >
                {item.label}
              </NavLink>
            ))}
          </nav>
        </SheetContent>
      </Sheet>
      <PendingOutcomesBanner />
      <main id="main-content" tabIndex={-1} className="flex flex-1 flex-col p-4 outline-none sm:p-6">
        <Routes>
          <Route path="/" element={<Board />} />
          <Route path="/list" element={<List />} />
          <Route path="/calendar" element={<Calendar />} />
          <Route path="/chat" element={<Chat />} />
          <Route path="/notes" element={<Notes />} />
          <Route path="/notes/:id" element={<NoteEditor />} />
          <Route path="/stats" element={<Stats />} />
          <Route path="/search" element={<SearchPage />} />
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
