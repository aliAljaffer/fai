import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Checkbox } from '@/components/ui/checkbox'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { useSearch } from '../api/hooks'
import { useWorkspaceStore } from '../store/workspace'

export function Search() {
  const [q, setQ] = useState('')
  const [allWorkspaces, setAllWorkspaces] = useState(false)
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const navigate = useNavigate()
  const { data } = useSearch(q, allWorkspaces ? undefined : activeWorkspaceId)

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <Input
          autoFocus
          placeholder="Search tasks and notes…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          className="max-w-md"
        />
        <label className="flex items-center gap-2 text-sm">
          <Checkbox
            checked={allWorkspaces}
            aria-label="Search all workspaces"
            onCheckedChange={(checked) => setAllWorkspaces(Boolean(checked))}
          />
          All workspaces
        </label>
      </div>

      {data && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <h3 className="mb-2 text-sm font-medium text-muted-foreground">Tasks</h3>
            <div className="flex flex-col gap-2">
              {data.tasks?.map((t) => (
                <Card
                  key={t.id}
                  className="cursor-pointer"
                  onClick={() => navigate({ search: `?task=${t.id}` })}
                >
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
