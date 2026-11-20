import { useState } from 'react'
import { Check, Copy, KeyRound, Trash2 } from 'lucide-react'
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
import { useAgentTokens, useCreateAgentToken, useDeleteAgentToken, useWorkspaces } from '../../api/hooks'
import type { AgentToken, CreatedAgentToken } from '../../api/types'

const ALL_WORKSPACES = 'all'

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

function formatMoment(value: string | null) {
  if (!value) return 'Never used'
  return new Date(value).toLocaleString()
}

function CopyField({ value, label }: { value: string; label: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <div className="flex gap-2">
      <Input readOnly aria-label={label} value={value} onFocus={(event) => event.currentTarget.select()} />
      <Button
        variant="outline"
        aria-label={`Copy ${label}`}
        onClick={() => {
          navigator.clipboard.writeText(value).then(
            () => {
              setCopied(true)
              toast.success('Copied')
              window.setTimeout(() => setCopied(false), 2000)
            },
            () => toast.error('Could not copy'),
          )
        }}
      >
        {copied ? <Check /> : <Copy />}
      </Button>
    </div>
  )
}

export function AgentsCard() {
  const { data: tokens } = useAgentTokens()
  const { data: workspaces } = useWorkspaces()
  const createToken = useCreateAgentToken()
  const deleteToken = useDeleteAgentToken()
  const [name, setName] = useState('')
  const [workspaceId, setWorkspaceId] = useState<string>(ALL_WORKSPACES)
  const [error, setError] = useState('')
  const [created, setCreated] = useState<CreatedAgentToken | null>(null)
  const [revoking, setRevoking] = useState<number | null>(null)

  const mcpURL = `${window.location.origin}/mcp`
  const workspaceName = (id: number | null) =>
    id === null ? 'All workspaces' : (workspaces?.find((item) => item.id === id)?.name ?? 'Unknown')

  function submit() {
    const trimmed = name.trim()
    if (!trimmed) {
      setError('Give the key a name so you can tell it apart later.')
      return
    }
    setError('')
    createToken.mutate(
      {
        name: trimmed,
        workspace_id: workspaceId === ALL_WORKSPACES ? undefined : Number(workspaceId),
      },
      {
        onSuccess: (result) => {
          setCreated(result)
          setName('')
          setWorkspaceId(ALL_WORKSPACES)
        },
        onError: (err) => setError(serverError(err)),
      },
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Agents</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm text-muted-foreground">
          Keys let an external agent use this workspace over MCP. The agent can read boards, tasks,
          notes, reminders and time, and can change them, so treat a key like a password. A key
          limited to one workspace cannot see the others.
        </p>

        <div className="flex flex-col gap-1.5">
          <span className="text-sm font-medium">MCP endpoint</span>
          <CopyField value={mcpURL} label="MCP endpoint" />
        </div>

        {created && (
          <div
            role="status"
            aria-live="polite"
            className="flex flex-col gap-2 rounded-md border border-primary/40 bg-primary/5 p-3"
          >
            <p className="text-sm font-medium">Copy this key now. It is shown once.</p>
            <CopyField value={created.secret} label={`Key for ${created.token.name}`} />
            <Button size="sm" variant="ghost" onClick={() => setCreated(null)}>
              Done
            </Button>
          </div>
        )}

        <div className="flex flex-col gap-1">
          {tokens?.map((token: AgentToken) => (
            <div key={token.id} className="flex flex-col gap-1 rounded-md border px-2 py-1.5">
              <div className="flex items-center gap-2">
                <KeyRound className="size-4 shrink-0 text-muted-foreground" />
                <span className="min-w-0 flex-1 truncate text-sm">{token.name}</span>
                <span className="hidden font-mono text-xs text-muted-foreground sm:inline">
                  {token.prefix}
                </span>
                <Button
                  size="icon-xs"
                  variant="ghost"
                  aria-label={`Revoke ${token.name}`}
                  onClick={() => setRevoking(token.id)}
                >
                  <Trash2 />
                </Button>
              </div>
              <div className="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
                <span>{workspaceName(token.workspace_id)}</span>
                <span>Created {new Date(token.created_at).toLocaleDateString()}</span>
                <span>{formatMoment(token.last_used_at)}</span>
              </div>
              {revoking === token.id && (
                <div className="flex flex-wrap items-center gap-2 border-t pt-1.5">
                  <span className="text-xs">
                    Revoke {token.name}? Agents using it stop working immediately.
                  </span>
                  <Button size="xs" variant="ghost" onClick={() => setRevoking(null)}>
                    Cancel
                  </Button>
                  <Button
                    size="xs"
                    variant="destructive"
                    disabled={deleteToken.isPending}
                    onClick={() =>
                      deleteToken.mutate(token.id, {
                        onSuccess: () => {
                          setRevoking(null)
                          if (created?.token.id === token.id) setCreated(null)
                          toast.success('Key revoked')
                        },
                        onError: (err) => toast.error(serverError(err)),
                      })
                    }
                  >
                    Revoke
                  </Button>
                </div>
              )}
            </div>
          ))}
          {!tokens?.length && <p className="text-sm text-muted-foreground">No agent keys yet.</p>}
        </div>

        <div className="flex flex-col gap-2 border-t pt-3">
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium" htmlFor="agent-key-name">
              New key
            </label>
            <Input
              id="agent-key-name"
              value={name}
              placeholder="Agent name, for example Hermes"
              onChange={(event) => setName(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && !event.nativeEvent.isComposing) submit()
              }}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-sm text-muted-foreground" htmlFor="agent-key-workspace">
              Scope
            </label>
            <Select
              items={[
                { value: ALL_WORKSPACES, label: 'All workspaces' },
                ...(workspaces ?? []).map((item) => ({
                  value: String(item.id),
                  label: `${item.name} (${item.tag})`,
                })),
              ]}
              value={workspaceId}
              onValueChange={(value) => value && setWorkspaceId(value)}
            >
              <SelectTrigger id="agent-key-workspace" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_WORKSPACES}>All workspaces</SelectItem>
                {workspaces?.map((item) => (
                  <SelectItem key={item.id} value={String(item.id)}>
                    {item.name}
                    <span className="text-muted-foreground">({item.tag})</span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {error && (
            <p role="alert" className="text-xs text-destructive">
              {error}
            </p>
          )}
          <Button size="sm" disabled={createToken.isPending} onClick={submit}>
            {createToken.isPending ? 'Creating…' : 'Create key'}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
