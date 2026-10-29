import { useState } from 'react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useDebounced } from '../../lib/useDebounced'
import type { JiraExportResult } from '../../api/types'
import {
  useExportJira,
  useJiraLink,
  useJiraSuggestions,
  useLinkJira,
  useUnlinkJira,
} from '../../api/hooks'

function ExportResults({ results }: { results: JiraExportResult[] | null }) {
  if (!results?.length) return null
  return (
    <div className="flex flex-col gap-1 rounded-md border p-2 text-xs">
      {results.map((result) => (
        <div key={result.task_id} className="flex items-center gap-2">
          <span className="text-muted-foreground">{result.status}</span>
          {result.jira_key && <span className="font-medium">{result.jira_key}</span>}
          <span className="truncate">{result.title || result.error}</span>
        </div>
      ))}
    </div>
  )
}

export function JiraPanel({ taskId }: { taskId: number }) {
  const { data: link } = useJiraLink(taskId)
  const linkJira = useLinkJira(taskId)
  const unlinkJira = useUnlinkJira(taskId)
  const exportJira = useExportJira(taskId)

  const [open, setOpen] = useState(false)
  const [key, setKey] = useState('')
  const [results, setResults] = useState<JiraExportResult[] | null>(null)
  const [showSuggestions, setShowSuggestions] = useState(false)
  const debouncedKey = useDebounced(key.trim(), 300)
  const { data: suggestions, isFetching } = useJiraSuggestions(taskId, debouncedKey)
  const searching = debouncedKey.length >= 2

  if (link) {
    return (
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-2">
          <a href={link.jira_url} target="_blank" rel="noreferrer">
            <Badge variant="secondary">{link.jira_key}</Badge>
          </a>
          <Button size="sm" variant="ghost" onClick={() => unlinkJira.mutate()}>
            Unlink
          </Button>
        </div>
        <ExportResults results={results} />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex gap-2">
        <Button
          size="sm"
          variant="outline"
          disabled={exportJira.isPending}
          onClick={() =>
            exportJira.mutate(undefined, {
              onSuccess: (response) => setResults(response.results),
              onError: (err) => toast.error(err.message),
            })
          }
        >
          Export to Jira
        </Button>
      </div>
      {open && (
        <div className="flex flex-col gap-2 rounded-md border p-2">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium">Link to Jira issue</span>
            <Button size="xs" variant="ghost" onClick={() => setOpen(false)}>
              Close
            </Button>
          </div>
          <Input
            autoFocus
            aria-label="Search Jira issues"
            placeholder="Search by key or text (min 2 characters)"
            value={key}
            onChange={(event) => {
              setKey(event.target.value)
              setShowSuggestions(true)
            }}
            onFocus={() => setShowSuggestions(true)}
            onKeyDown={(event) => {
              if (event.key === 'Escape') setOpen(false)
              if (event.key === 'Enter' && key.trim() && !linkJira.isPending) {
                event.preventDefault()
                linkJira.mutate(key.trim(), {
                  onSuccess: () => setOpen(false),
                  onError: (err) => toast.error(err.message),
                })
              }
            }}
          />
          {showSuggestions && searching && (
            <div className="flex flex-col gap-1">
              {isFetching && <p className="px-1 text-xs text-muted-foreground">Searching…</p>}
              {!isFetching && !suggestions?.length && (
                <p className="px-1 text-xs text-muted-foreground">No matching issues.</p>
              )}
              {suggestions?.map((s) => (
                <button
                  key={s.jira_key}
                  className="rounded-md border px-3 py-2 text-left text-sm hover:bg-accent"
                  onClick={() => {
                    setKey(s.jira_key)
                    setShowSuggestions(false)
                  }}
                >
                  <span className="font-medium">{s.jira_key}</span> {s.summary}{' '}
                  <span className="text-muted-foreground">({s.status})</span>
                </button>
              ))}
            </div>
          )}
          <div className="flex justify-end">
            <Button
              size="sm"
              disabled={!key.trim() || linkJira.isPending}
              onClick={() =>
                linkJira.mutate(key.trim(), {
                  onSuccess: () => setOpen(false),
                  onError: (err) => toast.error(err.message),
                })
              }
            >
              Link
            </Button>
          </div>
        </div>
      )}
      <ExportResults results={results} />
    </div>
  )
}
