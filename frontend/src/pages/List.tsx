import { useMemo, useState } from 'react'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
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
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="cursor-pointer" onClick={() => setSortKey('title')}>
            Title
          </TableHead>
          <TableHead>Column</TableHead>
          <TableHead className="cursor-pointer" onClick={() => setSortKey('priority')}>
            Priority
          </TableHead>
          <TableHead className="cursor-pointer" onClick={() => setSortKey('due_date')}>
            Due
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {sorted.map((t) => (
          <TableRow key={t.id}>
            <TableCell>{t.title}</TableCell>
            <TableCell>{columnName.get(t.column_id)}</TableCell>
            <TableCell>{t.priority}</TableCell>
            <TableCell>{t.due_date ?? '—'}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
