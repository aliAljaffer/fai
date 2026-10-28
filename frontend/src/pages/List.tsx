import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useBoards, useColumns, useTags, useTasks } from '../api/hooks'
import { useDebounced } from '../lib/useDebounced'
import { useUIStore } from '../store/ui'
import type { Task } from '../api/types'

type SortKey = 'title' | 'priority' | 'due_date'
const priorityRank: Record<string, number> = { high: 0, medium: 1, low: 2 }

function compare(a: Task, b: Task, key: SortKey) {
  if (key === 'priority') return (priorityRank[a.priority] ?? 3) - (priorityRank[b.priority] ?? 3)
  if (key === 'due_date') {
    if (!a.due_date) return 1
    if (!b.due_date) return -1
    return a.due_date.localeCompare(b.due_date)
  }
  return a.title.localeCompare(b.title)
}

export function List() {
  const { data: boards } = useBoards()
  const { selectedBoardId } = useUIStore()
  const boardId = selectedBoardId ?? boards?.[0]?.id

  const { data: columns } = useColumns(boardId)
  const { data: tags } = useTags()
  const [q, setQ] = useState('')
  const [columnId, setColumnId] = useState('all')
  const [tag, setTag] = useState('all')
  const debouncedQ = useDebounced(q)

  const { data: tasks } = useTasks(boardId, {
    q: debouncedQ || undefined,
    columnId: columnId === 'all' ? undefined : Number(columnId),
    tag: tag === 'all' ? undefined : tag,
  })

  const [, setSearchParams] = useSearchParams()
  const [sort, setSort] = useState<{ key: SortKey; dir: 1 | -1 }>({ key: 'title', dir: 1 })

  const columnName = useMemo(() => {
    const map = new Map<number, string>()
    for (const c of columns ?? []) map.set(c.id, c.name)
    return map
  }, [columns])

  const sorted = useMemo(() => {
    const list = (tasks ?? []).filter((task) => !task.parent_task_id)
    list.sort((a, b) => compare(a, b, sort.key) * sort.dir)
    return list
  }, [tasks, sort])

  function toggleSort(key: SortKey) {
    setSort((s) => (s.key === key ? { key, dir: s.dir === 1 ? -1 : 1 } : { key, dir: 1 }))
  }

  const arrow = (key: SortKey) => (sort.key === key ? (sort.dir === 1 ? ' ↑' : ' ↓') : '')

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          placeholder="Filter tasks…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          className="max-w-xs"
        />
        <Select value={columnId} onValueChange={(value) => value && setColumnId(value)}>
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All columns</SelectItem>
            {columns?.map((c) => (
              <SelectItem key={c.id} value={String(c.id)}>
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={tag} onValueChange={(value) => value && setTag(value)}>
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All tags</SelectItem>
            {tags?.map((t) => (
              <SelectItem key={t.id} value={t.name}>
                {t.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="cursor-pointer" onClick={() => toggleSort('title')}>
              Title{arrow('title')}
            </TableHead>
            <TableHead>Column</TableHead>
            <TableHead>Labels</TableHead>
            <TableHead className="cursor-pointer" onClick={() => toggleSort('priority')}>
              Priority{arrow('priority')}
            </TableHead>
            <TableHead className="cursor-pointer" onClick={() => toggleSort('due_date')}>
              Due{arrow('due_date')}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sorted.map((t) => (
            <TableRow
              key={t.id}
              className="cursor-pointer hover:bg-muted/50"
              onClick={() => setSearchParams({ task: String(t.id) })}
            >
              <TableCell>{t.title}</TableCell>
              <TableCell>{columnName.get(t.column_id)}</TableCell>
              <TableCell>
                <div className="flex flex-wrap gap-1">
                  {(t.tags ?? []).map((tag) => (
                    <span
                      key={tag.id}
                      className="inline-flex items-center gap-1 rounded-full border px-1.5 py-0.5 text-[0.65rem] text-muted-foreground"
                    >
                      <span
                        className="size-1.5 rounded-full"
                        style={{ backgroundColor: tag.color || undefined }}
                      />
                      {tag.name}
                    </span>
                  ))}
                </div>
              </TableCell>
              <TableCell>{t.priority}</TableCell>
              <TableCell>{t.due_date ?? '—'}</TableCell>
            </TableRow>
          ))}
          {!sorted.length && (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-muted-foreground">
                No tasks match these filters.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  )
}
