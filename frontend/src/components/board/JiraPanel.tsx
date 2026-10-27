import { useState } from 'react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  useExportJira,
  useJiraLink,
  useJiraSuggestions,
  useLinkJira,
  useUnlinkJira,
} from '../../api/hooks'
import type { JiraSuggestion } from '../../api/types'

export function JiraPanel({ taskId }: { taskId: number }) {
  const { data: link } = useJiraLink(taskId)
  const linkJira = useLinkJira(taskId)
  const unlinkJira = useUnlinkJira(taskId)
  const exportJira = useExportJira(taskId)
  const suggestJira = useJiraSuggestions(taskId)

  const [open, setOpen] = useState(false)
  const [key, setKey] = useState('')
  const [suggestions, setSuggestions] = useState<JiraSuggestion[] | null>(null)

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
      <Dialog open={open} onOpenChange={setOpen}>
        <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
          Link existing issue
        </Button>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Link to Jira issue</DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-3">
            <Input placeholder="e.g. DEV-123" value={key} onChange={(e) => setKey(e.target.value)} />
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                suggestJira.mutate(undefined, {
                  onSuccess: setSuggestions,
                  onError: (err) => toast.error(err.message),
                })
              }
            >
              {suggestJira.isPending ? 'Searching…' : 'Suggest matches'}
            </Button>
            {suggestions?.map((s) => (
              <button
                key={s.jira_key}
                className="rounded-md border px-3 py-2 text-left text-sm hover:bg-accent"
                onClick={() => setKey(s.jira_key)}
              >
                <span className="font-medium">{s.jira_key}</span> {s.summary}{' '}
                <span className="text-muted-foreground">({s.status})</span>
              </button>
            ))}
          </div>
          <DialogFooter>
            <Button
              onClick={() =>
                linkJira.mutate(key, {
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
