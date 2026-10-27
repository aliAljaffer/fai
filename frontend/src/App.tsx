import { NavLink, Route, Routes } from 'react-router-dom'
import { Toaster } from '@/components/ui/sonner'
import { cn } from '@/lib/utils'
import { Board } from './pages/Board'
import { List } from './pages/List'
import { Notes } from './pages/Notes'
import { NoteEditor } from './pages/NoteEditor'

export function App() {
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
          to="/notes"
          className={({ isActive }) =>
            cn('text-muted-foreground hover:text-foreground', isActive && 'font-semibold text-foreground')
          }
        >
          Notes
        </NavLink>
      </nav>
      <main className="p-6">
        <Routes>
          <Route path="/" element={<Board />} />
          <Route path="/list" element={<List />} />
          <Route path="/notes" element={<Notes />} />
          <Route path="/notes/:id" element={<NoteEditor />} />
        </Routes>
      </main>
      <Toaster />
    </div>
  )
}
