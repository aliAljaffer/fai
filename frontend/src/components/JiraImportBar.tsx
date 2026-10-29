import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { useJiraImportStatus } from '../api/hooks'

// Lives in the app shell so a running import stays visible on every page.
export function JiraImportBar() {
  const { data } = useJiraImportStatus()
  const [dismissed, setDismissed] = useState<string>()

  if (!data?.started_at) return null
  if (dismissed === data.finished_at) return null

  const handled = data.imported + data.updated + data.skipped
  const percent = data.total ? Math.min(100, Math.round((handled / data.total) * 100)) : 0

  return (
    <div className="px-6 pt-3">
      <div className="flex items-center gap-3 rounded-md border bg-accent px-3 py-2 text-sm">
        <div className="flex-1">
          {data.running ? (
            <>
              <span>
                Importing from Jira: {handled} of {data.total}
              </span>
              <div
                className="mt-1 h-1 overflow-hidden rounded bg-muted"
                role="progressbar"
                aria-label="Jira import progress"
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={percent}
              >
                <div
                  className="h-full bg-primary transition-[width] duration-300"
                  style={{ width: `${percent}%` }}
                />
              </div>
            </>
          ) : data.error ? (
            <span className="text-destructive">Jira import failed: {data.error}</span>
          ) : (
            <span>
              Jira import finished: imported {data.imported}, updated {data.updated}, skipped{' '}
              {data.skipped}
              {data.linked ? `, linked ${data.linked}` : ''}
            </span>
          )}
        </div>
        {!data.running && (
          <Button size="sm" variant="ghost" onClick={() => setDismissed(data.finished_at)}>
            Dismiss
          </Button>
        )}
      </div>
    </div>
  )
}
