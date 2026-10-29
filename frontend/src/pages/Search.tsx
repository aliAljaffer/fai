import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
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
      <h1 className="text-lg font-semibold">Search</h1>
      <div className="flex flex-wrap items-center gap-3">
        <Input
          autoFocus
          aria-label="Search tasks and notes"
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
            <h2 className="mb-2 text-sm font-medium text-muted-foreground">Tasks</h2>
            <div className="flex flex-col gap-2">
              {data.tasks?.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  className="rounded-xl text-left outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
                  onClick={() => navigate({ search: `?task=${t.id}` })}
                >
                  <Card className="transition-colors hover:bg-muted/40">
                    <CardContent>{t.title}</CardContent>
                  </Card>
                </button>
              ))}
            </div>
          </div>
          <div>
            <h2 className="mb-2 text-sm font-medium text-muted-foreground">Notes</h2>
            <div className="flex flex-col gap-2">
              {data.notes?.map((n) => (
                <Link
                  key={n.id}
                  to={`/notes/${n.id}`}
                  className="rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
                >
                  <Card className="transition-colors hover:bg-muted/40">
                    <CardContent>{n.title}</CardContent>
                  </Card>
                </Link>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
