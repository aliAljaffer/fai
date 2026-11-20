# Agent integration

Fai exposes one MCP endpoint, so any agent that speaks the Model Context
Protocol can work your boards, tasks, notes, reminders and time entries. Hermes
Agent is the reference client. The same endpoint works with Claude Code,
Command Code, OpenCode, and anything else that supports remote MCP over HTTP.

## 1. Create an agent key

Open **Settings → Agents** and click **Create key**. Give it a name and, if you
want, limit it to one workspace. Copy the key when it appears: Fai stores only a
hash and cannot show it again.

The MCP endpoint is `https://<your-fai-host>/mcp`. It authenticates with the
agent key instead of the site login, so Caddy routes `/mcp` around Basic auth.

You can also seed keys from the environment with `FAI_AGENT_TOKENS`, for
deployments that never open the UI:

```sh
# name:secret[:WORKSPACE], comma separated. Secrets need 16+ characters.
FAI_AGENT_TOKENS=hermes:9f2c1b7a4d8e3f6a,claude:7ab1c3d5e7f90123:WORK
```

Environment keys cannot be revoked without a restart. Prefer keys created in the
UI, which can be revoked at any time.

## 2. Connect the agent

### Hermes Agent

Add the server to `~/.hermes/config.yaml` and put the key in `~/.hermes/.env`:

```yaml
mcp_servers:
  fai:
    url: "https://fai.example.com/mcp"
    headers:
      Authorization: "Bearer ${FAI_AGENT_KEY}"
    tools:
      exclude: ["delete_task"]
```

```sh
echo 'FAI_AGENT_KEY=fai_...' >> ~/.hermes/.env
hermes mcp test fai
```

Start a new session (or `/reload-mcp`) and ask Hermes what is on the board. Hermes
registers the tools as `mcp_fai_get_context`, `mcp_fai_create_task`, and so on.

Install the skill that teaches the workflow:

```sh
hermes skills install https://raw.githubusercontent.com/alialjaffer/fai/main/skills/fai/SKILL.md
```

### Claude Code

```sh
claude mcp add --transport http fai https://fai.example.com/mcp \
  --header "Authorization: Bearer ${FAI_AGENT_KEY}"
```

Or commit a project `.mcp.json`:

```json
{
  "mcpServers": {
    "fai": {
      "type": "http",
      "url": "https://fai.example.com/mcp",
      "headers": { "Authorization": "Bearer ${FAI_AGENT_KEY}" }
    }
  }
}
```

### Command Code

Flags go before the server name:

```sh
cmd mcp add --transport http fai https://fai.example.com/mcp \
  --header "Authorization: Bearer $FAI_AGENT_KEY"
```

Or use JSON:

```sh
cmd mcp add-json fai '{
  "type": "http",
  "url": "https://fai.example.com/mcp",
  "headers": { "Authorization": "Bearer ${FAI_AGENT_KEY}" }
}'
```

`cmd mcp list` and `cmd mcp get fai` confirm the connection.

### OpenCode

Add the server to `opencode.json` or `~/.config/opencode/opencode.jsonc`, with
`FAI_AGENT_KEY` exported in the environment OpenCode runs in:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "fai": {
      "type": "remote",
      "url": "https://fai.example.com/mcp",
      "headers": { "Authorization": "Bearer {env:FAI_AGENT_KEY}" },
      "enabled": true
    }
  }
}
```

## 3. What the agent can do

| Tool | Purpose |
| --- | --- |
| `get_context` | Today's date, workspaces, boards, columns, tags, running timer |
| `list_boards`, `create_board`, `create_column` | Board structure |
| `create_workspace`, `create_tag` | Workspace and tag setup |
| `list_tasks`, `get_task`, `search`, `list_upcoming` | Read work |
| `create_task`, `update_task`, `move_task`, `complete_task`, `delete_task` | Change work |
| `add_steps`, `add_comment`, `tag_task` | Break work down and hand it off |
| `create_note`, `get_note`, `append_note` | Notes |
| `start_timer`, `stop_timer`, `log_time` | Time tracking |
| `create_reminder`, `list_reminders`, `complete_reminder` | Reminders |

Every task carries its ref (`WORK-12`). An agent that writes `@WORK-12` in a
comment or note creates a link the UI renders, so the human can follow what the
agent did.

Not exposed on purpose: backup and restore, workspace reset, Jira import and
export, appearance, and Fai's own AI endpoints. An agent brings its own model.

## 4. Orchestration: wake the agent when work arrives

Fai can POST every task change to an agent endpoint. Requests are signed with
HMAC-SHA256 over `<unix timestamp>.<body>`, the scheme Hermes calls `generic`
V2, in the `X-Webhook-Signature-V2` header. `X-Webhook-Timestamp` carries the
timestamp and `X-Request-ID` a per-event id for de-duplication.

Set the endpoint on the Fai server:

```sh
FAI_WEBHOOK_URL=https://hermes.example.com:8644/webhooks/fai-tasks
FAI_WEBHOOK_SECRET=<openssl rand -hex 32>
```

Fai sends `task.created`, `task.updated`, `task.moved`, `task.completed` and
`task.commented`:

```json
{
  "event": "task.created",
  "event_type": "task.created",
  "id": "9f2c1b7a4d8e3f6a",
  "at": "2026-11-20T09:12:44Z",
  "actor": "agent:hermes",
  "task": {
    "id": 42,
    "ref": "WORK-42",
    "title": "Rotate the staging certificates",
    "kind": "task",
    "priority": "high",
    "due_date": "2026-11-25",
    "completed": false,
    "board": "Ops",
    "column": "Ready",
    "workspace": "WORK"
  }
}
```

Deliveries retry at 2s and 10s, then drop. A dead endpoint never blocks a save.

On the Hermes side, add a route:

```yaml
platforms:
  webhook:
    enabled: true
    extra:
      port: 8644
      # Paste the same value as FAI_WEBHOOK_SECRET.
      secret: "<webhook secret>"
      routes:
        fai-tasks:
          events: ["task.created"]
          signature: generic
          prompt: |
            A task was created in Fai.
            Ref: {task.ref}
            Title: {task.title}
            Board: {task.board} / {task.column}
            Priority: {task.priority}
            Due: {task.due_date}

            Read @{task.ref} with the fai skill, do the work, and report back with
            add_comment. Move it to the done column only when it is finished.
          skills: ["fai"]
          deliver: "telegram"
```

Use `events` to pick which changes wake the agent. Without it, every change does.
For a queue-style board, create a `Ready` column and filter on
`{task.column}` equalling `Ready`.

If the agent endpoint is not reachable from Fai, use a Hermes cron job instead:

```sh
hermes cron create "every 10 minutes" \
  --prompt "Check the Fai Ready column with list_tasks, work one task, and comment the result."
```

## 5. Clients without MCP

Agent keys work on `/mcp` only. The REST API under `/api` stays behind the site
login, because that surface can export backups and spend the AI key, and because
a workspace-scoped key cannot be enforced on it. For a client without MCP, run
the agent through a shell that can call the MCP endpoint, or open an issue for a
narrower REST token mode.

## 6. Limits and safety

- A key scopes to one workspace or all of them. Scoped keys cannot read or write
  other workspaces, including through title search.
- Tool results include text the agent did not write: task titles, comments and
  notes. Treat that as untrusted input. Scope the key and prune tools with
  `tools.exclude` when the agent does not need to change anything.
- `delete_task` archives; the task can be restored from the UI.
- Attachments and calendar events are read through the web app, not MCP yet.
  Task descriptions, comments and notes carry the context an agent normally
  needs; attaching a file to a task is a human step in this version.
- One key per agent, so revoking one does not disturb the others. `last_used_at`
  in Settings shows when a key was last presented.
- Restoring a backup replaces the workspaces, so keys scoped to a workspace are
  removed with it. Create those keys again after a restore. Keys with access to
  all workspaces survive, and backup files never contain key material.
