import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import {
  useBoards,
  useCalendarFeedURL,
  useColumns,
  useCreateWorkspace,
  useJiraImport,
  useUpdateWorkspace,
  useWorkspaces,
} from '../api/hooks'
import { columnColor, useColumnColors } from '../store/columnColors'
import { useUIStore } from '../store/ui'
import { useWorkspaceStore } from '../store/workspace'
import type { Workspace } from '../api/types'

const TAG_PATTERN = /^[A-Za-z]{3,4}$/

function serverError(error: unknown) {
  const message = error instanceof Error ? error.message : 'Request failed'
  const body = message.match(/\{.*\}$/)?.[0]
  if (!body) return message
  try {
    const parsed = JSON.parse(body) as { error?: string }
    return parsed.error ?? message
  } catch {
    return message
  }
}

function WorkspaceRow({ workspace }: { workspace: Workspace }) {
  const updateWorkspace = useUpdateWorkspace()
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(workspace.name)
  const [tag, setTag] = useState(workspace.tag)
  const [error, setError] = useState('')

  function startEditing() {
    setName(workspace.name)
    setTag(workspace.tag)
    setError('')
    setEditing(true)
  }

  function save() {
    const trimmedName = name.trim()
    const trimmedTag = tag.trim()
    if (!trimmedName) {
      setError('Name is required.')
      return
    }
    if (!TAG_PATTERN.test(trimmedTag)) {
      setError('Tag must be 3 or 4 letters.')
      return
    }
    updateWorkspace.mutate(
      { id: workspace.id, name: trimmedName, tag: trimmedTag },
      {
        onSuccess: () => setEditing(false),
        onError: (err) => setError(serverError(err)),
      },
    )
  }

  if (!editing) {
    return (
      <div className="flex items-center justify-between gap-2 rounded-md border px-2 py-1.5">
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate text-sm">{workspace.name}</span>
          <span className="rounded border px-1 font-mono text-[0.65rem] text-muted-foreground">
            {workspace.tag}
          </span>
        </div>
        <Button size="xs" variant="ghost" onClick={startEditing}>
          Rename
        </Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2 rounded-md border px-2 py-1.5">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          autoFocus
          value={name}
          aria-label={`Name for ${workspace.name}`}
          placeholder="Workspace name"
          className="h-7 max-w-48"
          onChange={(event) => setName(event.target.value)}
        />
        <Input
          value={tag}
          aria-label={`Tag for ${workspace.name}`}
          placeholder="TAG"
          className="h-7 w-20 font-mono uppercase"
          onChange={(event) => setTag(event.target.value)}
        />
        <Button size="xs" disabled={updateWorkspace.isPending} onClick={save}>
          Save
        </Button>
        <Button size="xs" variant="ghost" onClick={() => setEditing(false)}>
          Cancel
        </Button>
      </div>
      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  )
}

function WorkspacesCard() {
  const { data: workspaces } = useWorkspaces()
  const createWorkspace = useCreateWorkspace()
  const [name, setName] = useState('')
  const [tag, setTag] = useState('')
  const [error, setError] = useState('')

  function submit() {
    const trimmedName = name.trim()
    const trimmedTag = tag.trim()
    if (!trimmedName) {
      setError('Name is required.')
      return
    }
    if (!TAG_PATTERN.test(trimmedTag)) {
      setError('Tag must be 3 or 4 letters.')
      return
    }
    createWorkspace.mutate(
      { name: trimmedName, tag: trimmedTag },
      {
        onSuccess: () => {
          setName('')
          setTag('')
          setError('')
        },
        onError: (err) => setError(serverError(err)),
      },
    )
  }

  return (
    <Card className="max-w-xl">
      <CardHeader>
        <CardTitle>Workspaces</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm text-muted-foreground">
          Workspaces separate boards, notes, reminders, and calendars. Tasks are numbered per
          workspace and keep the tag they were created with after a rename.
        </p>
        <div className="flex flex-col gap-1">
          {workspaces?.map((workspace) => (
            <WorkspaceRow key={workspace.id} workspace={workspace} />
          ))}
          {!workspaces?.length && (
            <p className="text-sm text-muted-foreground">No workspaces yet.</p>
          )}
        </div>
        <div className="flex flex-col gap-2 border-t pt-3">
          <div className="flex flex-wrap items-center gap-2">
            <Input
              value={name}
              aria-label="New workspace name"
              placeholder="Workspace name"
              className="max-w-48"
              onChange={(event) => setName(event.target.value)}
            />
            <Input
              value={tag}
              aria-label="New workspace tag"
              placeholder="TAG"
              className="w-20 font-mono uppercase"
              onChange={(event) => setTag(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') submit()
              }}
            />
            <Button size="sm" disabled={createWorkspace.isPending} onClick={submit}>
              Create workspace
            </Button>
          </div>
          {error && <p className="text-xs text-destructive">{error}</p>}
        </div>
      </CardContent>
    </Card>
  )
}

export function Settings() {
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: boards } = useBoards(workspaceId)
  const { data: workspaces } = useWorkspaces()
  const jiraImport = useJiraImport()
  const { data: feed } = useCalendarFeedURL()
  const selectedBoardId = useUIStore((state) => state.selectedBoardId)
  const board =
    boards?.find((b) => b.id === selectedBoardId) ?? boards?.[0]
  const boardId = board?.id
  const { data: columns } = useColumns(boardId)
  const { colors, setColor } = useColumnColors()
  const { data: allBoards } = useBoards()
  const [targetBoardId, setTargetBoardId] = useState<number>()
  const importBoard =
    allBoards?.find((b) => b.id === (targetBoardId ?? boardId)) ?? allBoards?.[0]
  const workspaceName = (id: number) =>
    workspaces?.find((w) => w.id === id)?.name ?? 'Unknown workspace'
  const [jql, setJql] = useState('')
  const [forceUpdate, setForceUpdate] = useState(false)

  return (
    <div className="flex flex-col gap-6">
      <WorkspacesCard />

      <Card className="max-w-xl">
        <CardHeader>
          <CardTitle>Jira import</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <p className="text-sm text-muted-foreground">
            Imports only Jira issues assigned to you. Epics become arcs, sub-tasks become steps,
            and every other issue type becomes a task. Items go to the selected board and its
            workspace. Leave the JQL field blank to use the server-configured default project.
            Already-imported issues are skipped unless "force update" is checked. Completion dates
            are taken from Jira.
          </p>
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium" htmlFor="jira-import-board">
              Target board
            </label>
            <Select
              items={(allBoards ?? []).map((b) => ({
                value: String(b.id),
                label: `${workspaceName(b.workspace_id)} / ${b.name}`,
              }))}
              value={importBoard ? String(importBoard.id) : undefined}
              onValueChange={(value) => value && setTargetBoardId(Number(value))}
            >
              <SelectTrigger id="jira-import-board" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {allBoards?.map((b) => (
                  <SelectItem key={b.id} value={String(b.id)}>
                    {workspaceName(b.workspace_id)}
                    <span className="text-muted-foreground">/ {b.name}</span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
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
            disabled={!importBoard || jiraImport.isPending}
            onClick={() => {
              if (!importBoard) return
              jiraImport.mutate(
                { board_id: importBoard.id, jql: jql || undefined, force_update: forceUpdate },
                {
                  onSuccess: (result) =>
                    toast.success(
                      `Imported ${result.imported}, updated ${result.updated}, skipped ${result.skipped}${
                        result.linked ? `, linked ${result.linked}` : ''
                      }`,
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
