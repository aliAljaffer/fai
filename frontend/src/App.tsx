import { NavLink, Route, Routes } from 'react-router-dom'
import { Board } from './pages/Board'
import { List } from './pages/List'

export function App() {
  return (
    <div className="app">
      <nav className="app-nav">
        <span className="app-title">fai</span>
        <NavLink to="/" end>
          Board
        </NavLink>
        <NavLink to="/list">List</NavLink>
      </nav>
      <main>
        <Routes>
          <Route path="/" element={<Board />} />
          <Route path="/list" element={<List />} />
        </Routes>
      </main>
    </div>
  )
}
