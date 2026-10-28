import { useState } from 'react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { useDebounced } from '../../lib/useDebounced'
import {
  useExportJira,
  useJiraLink,
  useJiraSuggestions,
  useLinkJira,
  useUnlinkJira,
} from '../../api/hooks'

export function JiraPanel({ taskId }: { taskId: number }) {
  const { data: link } = useJiraLink(taskId)
  const linkJira = useLinkJira(taskId)
  const unlinkJira = useUnlinkJira(taskId)
  const exportJira = useExportJira(taskId)

  const [open, setOpen] = useState(false)
  const [key, setKey] = useState('')
  const [showSuggestions, setShowSuggestions] = useState(false)
  const debouncedKey = useDebounced(key.trim(), 300)
  const { data: suggestions, isFetching } = useJiraSuggestions(taskId, debouncedKey)
  const searching = debouncedKey.length >= 2

  if (link) {
    return (
      <div className="flex items-center gap-2">
        <a href={link.jira_url} target="_blank" rel="noreferrer">
          <Badge variant="secondary">{link.jira_key}</Badge>
        </a>
        <Button size="sm" variant="ghost" onClick={() => unlinkJira.mutate()}>
          Unlink
        </Button>
      </div>
    )
  }

  return (
    <div className="flex gap-2">
      <Dialog
        open={open}
        onOpenChange={(next) => {
          setOpen(next)
          if (!next) {
            setKey('')
            setShowSuggestions(false)
          }
        }}
      >
        <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
          Link existing issue
        </Button>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Link to Jira issue</DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-2">
            <Input
              placeholder="Search by key or text (min 2 characters)"
              value={key}
              onChange={(event) => {
                setKey(event.target.value)
                setShowSuggestions(true)
              }}
              onFocus={() => setShowSuggestions(true)}
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
          </div>
          <DialogFooter>
            <Button
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
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Button
        size="sm"
        variant="outline"
        onClick={() => exportJira.mutate(undefined, { onError: (err) => toast.error(err.message) })}
      >
        Export as new issue
      </Button>
    </div>
  )
}
