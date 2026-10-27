import { NavLink, Route, Routes } from 'react-router-dom'
import { Toaster } from '@/components/ui/sonner'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { AuthCallback } from './pages/AuthCallback'
import { Board } from './pages/Board'
import { Calendar } from './pages/Calendar'
import { List } from './pages/List'
import { Notes } from './pages/Notes'
import { NoteEditor } from './pages/NoteEditor'
import { Reminders } from './pages/Reminders'
import { Search } from './pages/Search'
import { Settings } from './pages/Settings'
import { Stats } from './pages/Stats'
import { CommandPalette } from './components/CommandPalette'
import { TimerWidget } from './components/TimerWidget'
import { TaskDetailSheet } from './components/board/TaskDetailSheet'
import { useUIStore } from './store/ui'

export function App() {
  const setCommandOpen = useUIStore((state) => state.setCommandOpen)

  return (
    <div className="min-h-screen">
      <nav className="flex items-center gap-5 border-b px-6 py-3">
        <span className="mr-2 font-bold">fai</span>
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
          to="/stats"
          className={({ isActive }) =>
            cn('text-muted-foreground hover:text-foreground', isActive && 'font-semibold text-foreground')
          }
        >
          Stats
        </NavLink>
        <NavLink
          to="/reminders"
          className={({ isActive }) =>
            cn('text-muted-foreground hover:text-foreground', isActive && 'font-semibold text-foreground')
          }
        >
          Reminders
        </NavLink>
        <TimerWidget />
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
      <main className="p-6">
        <Routes>
          <Route path="/" element={<Board />} />
          <Route path="/list" element={<List />} />
          <Route path="/calendar" element={<Calendar />} />
          <Route path="/notes" element={<Notes />} />
          <Route path="/notes/:id" element={<NoteEditor />} />
          <Route path="/stats" element={<Stats />} />
          <Route path="/reminders" element={<Reminders />} />
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
