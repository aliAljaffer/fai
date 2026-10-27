import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { api } from '../api/client'
import type { Note, Task } from '../api/types'

export function Search() {
  const [q, setQ] = useState('')
  const navigate = useNavigate()
  const { data } = useQuery({
    queryKey: ['search', q],
    queryFn: () => api.get<{ tasks: Task[]; notes: Note[] }>(`/search?q=${encodeURIComponent(q)}`),
    enabled: q.length > 0,
  })

  return (
    <div className="flex flex-col gap-4">
      <Input
        autoFocus
        placeholder="Search tasks and notes…"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        className="max-w-md"
      />

      {data && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <h3 className="mb-2 text-sm font-medium text-muted-foreground">Tasks</h3>
            <div className="flex flex-col gap-2">
              {data.tasks?.map((t) => (
                <Card key={t.id} className="cursor-pointer" onClick={() => navigate('/list')}>
                  <CardContent>{t.title}</CardContent>
                </Card>
              ))}
            </div>
          </div>
          <div>
            <h3 className="mb-2 text-sm font-medium text-muted-foreground">Notes</h3>
            <div className="flex flex-col gap-2">
              {data.notes?.map((n) => (
                <Card key={n.id} className="cursor-pointer" onClick={() => navigate(`/notes/${n.id}`)}>
                  <CardContent>{n.title}</CardContent>
                </Card>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
