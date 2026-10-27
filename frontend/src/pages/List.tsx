import { useMemo, useState } from 'react'
import { useBoards, useColumns, useTasks } from '../api/hooks'
import { useUIStore } from '../store/ui'

export function List() {
  const { data: boards } = useBoards()
  const { selectedBoardId } = useUIStore()
  const boardId = selectedBoardId ?? boards?.[0]?.id

  const { data: columns } = useColumns(boardId)
  const { data: tasks } = useTasks(boardId)
  const [sortKey, setSortKey] = useState<'title' | 'priority' | 'due_date'>('title')

  const columnName = useMemo(() => {
    const map = new Map<number, string>()
    for (const c of columns ?? []) map.set(c.id, c.name)
    return map
  }, [columns])

  const sorted = useMemo(() => {
    const list = [...(tasks ?? [])]
    list.sort((a, b) => String(a[sortKey] ?? '').localeCompare(String(b[sortKey] ?? '')))
    return list
  }, [tasks, sortKey])

  return (
    <div className="list-page">
      <table>
        <thead>
          <tr>
            <th onClick={() => setSortKey('title')}>Title</th>
            <th>Column</th>
            <th onClick={() => setSortKey('priority')}>Priority</th>
            <th onClick={() => setSortKey('due_date')}>Due</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((t) => (
            <tr key={t.id}>
              <td>{t.title}</td>
              <td>{columnName.get(t.column_id)}</td>
              <td>{t.priority}</td>
              <td>{t.due_date ?? '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
