import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useBoards, useCalendarFeedURL, useColumns, useJiraImport } from '../api/hooks'
import { columnColor, useColumnColors } from '../store/columnColors'
import { useUIStore } from '../store/ui'

export function Settings() {
  const { data: boards } = useBoards()
  const jiraImport = useJiraImport()
  const { data: feed } = useCalendarFeedURL()
  const selectedBoardId = useUIStore((state) => state.selectedBoardId)
  const { data: columns } = useColumns(selectedBoardId ?? boards?.[0]?.id)
  const { colors, setColor } = useColumnColors()
  const [jql, setJql] = useState('')
  const [forceUpdate, setForceUpdate] = useState(false)

  const boardId = boards?.[0]?.id

  return (
    <div className="flex flex-col gap-6">
      <Card className="max-w-xl">
        <CardHeader>
          <CardTitle>Jira import</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <p className="text-sm text-muted-foreground">
            Imports issues from Jira as tasks on the first board. Leave the JQL field blank to use
            the server-configured default project. Already-imported issues are skipped unless
            "force update" is checked.
          </p>
          <Textarea
            placeholder='Optional JQL, e.g. project = "DEV" AND status != Done'
            value={jql}
            onChange={(e) => setJql(e.target.value)}
            rows={3}
          />
          <label className="flex items-center gap-2 text-sm">
            <Input
              type="checkbox"
              className="h-4 w-4"
              checked={forceUpdate}
              onChange={(e) => setForceUpdate(e.target.checked)}
            />
            Force update already-imported issues
          </label>
          <Button
            disabled={!boardId || jiraImport.isPending}
            onClick={() => {
              if (!boardId) return
              jiraImport.mutate(
                { board_id: boardId, jql: jql || undefined, force_update: forceUpdate },
                {
                  onSuccess: (result) =>
                    toast.success(
                      `Imported ${result.imported}, updated ${result.updated}, skipped ${result.skipped}`,
                    ),
                  onError: (err) => toast.error(err.message),
                },
              )
            }}
          >
            {jiraImport.isPending ? 'Importing…' : 'Import from Jira'}
          </Button>
        </CardContent>
      </Card>

      <Card className="max-w-xl">
        <CardHeader>
          <CardTitle>Status colors</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <p className="text-sm text-muted-foreground">
            Colors the status text in the List view. Saved in this browser only.
          </p>
          {columns?.map((column) => (
            <label
              key={column.id}
              className="flex items-center gap-2 rounded-md border px-2 py-1.5"
            >
              <input
                type="color"
                value={columnColor(column, colors)}
                aria-label={`Status color for ${column.name}`}
                className="h-6 w-6 shrink-0 cursor-pointer rounded border border-input bg-transparent p-0.5"
                onChange={(event) => setColor(column.id, event.target.value)}
              />
              <span className="text-sm">{column.name}</span>
            </label>
          ))}
          {!columns?.length && (
            <p className="text-sm text-muted-foreground">This board has no columns yet.</p>
          )}
        </CardContent>
      </Card>

      <Card className="max-w-xl">
        <CardHeader>
          <CardTitle>Calendar subscription</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <p className="text-sm text-muted-foreground">
            Subscribe from Apple Calendar (File → New Calendar Subscription, or Settings → Calendar →
            Accounts → Add Account → Other → Add Subscribed Calendar). Open tasks with a due date
            appear as all-day events and refresh about every 30 minutes. The feed contains Fai
            tasks only. Calendars imported in the app stay out of it.
          </p>
          <div className="flex gap-2">
            <Input
              readOnly
              value={feed?.url ?? ''}
              placeholder={feed ? '' : 'Calendar feed is not configured'}
              onFocus={(e) => e.currentTarget.select()}
            />
            <Button
              variant="outline"
              disabled={!feed}
              onClick={() => {
                if (!feed) return
                navigator.clipboard.writeText(feed.url).then(
                  () => toast.success('Subscription URL copied'),
                  () => toast.error('Could not copy the URL'),
                )
              }}
            >
              Copy
            </Button>
          </div>
          {feed && (
            <a className="text-sm text-primary underline" href={feed.url.replace(/^https:/, 'webcal:')}>
              Open in Calendar
            </a>
          )}
          <p className="text-xs text-muted-foreground">
            The URL contains a secret token — treat it like a password. Rotating ICS_TOKEN
            invalidates existing subscriptions.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
