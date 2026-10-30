# fai Installation Guide

Full setup and configuration details for fai. The [README](../README.md) has
the short install commands.

## Features

| Area | Features |
|---|---|
| Tasks | Kanban and list views, filters, bulk actions, subtasks, labels, priorities, due dates, comments, and attachments |
| Task structure | Arcs group work, tasks track work, and steps break tasks down. Each task has a reference such as `FAI-42`. |
| Workspaces | Separate tasks, notes, reminders, and calendars by workspace |
| Time tracking | Start timers from a task or the navigation bar. View totals on tasks and in Stats. |
| Notes | Markdown, `[[wikilinks]]`, backlinks, attachments, and AI task plans |
| Calendar | Import iCalendar feeds, show recurring events, and create tasks from meeting action items |
| Search and stats | Search tasks and notes. View completion heatmaps and tracked-time charts. |
| Reminders | Show reminders on the board. Send notifications through ntfy or email. |
| AI | Chat creates tasks with steps and arcs with nested tasks. Also generates label suggestions, note plans, and meeting action items. |
| Jira | Import issues with JQL, export tasks, and map Jira statuses to board columns |
| Themes | Built-in palettes, custom palettes, and workspace colors. Workspace themes are stored in the database. |
| Live updates | `GET /api/events` streams server-sent events. Every open client sees changes made on another device. |
| Backup | `GET /api/backup` downloads a zip with the database and attachments. `POST /api/backup/restore` restores it after you type `I understand`. |

## Requirements

- Go 1.27
- Node 22
- Docker
- SQLite or PostgreSQL
- `goose` and `sqlc` for schema changes
- Keycloak for OIDC mode, optional for local development

## Local development

```sh
make dev-db-up
make migrate-up
cp backend/.env.example backend/.env
make backend-run
make frontend-run
```

The local backend uses `DISABLE_AUTH=true`. The frontend dev server proxies
`/api` to `:8080`.

The backend uses Go, chi, pgx, sqlc, and goose. The frontend uses React and
Vite. The desktop shell uses Electron. Storage uses SQLite or PostgreSQL.

## Docker Compose

The production Compose stack uses SQLite, a bundled ntfy service, and Caddy for
HTTP Basic authentication and TLS. It does not require an identity provider.

1. Copy `.env.prod.example` to `.env.prod`.
2. Set `FAI_DOMAIN`, `FAI_AUTH_USER`, `FAI_AUTH_HASH`, and `ICS_TOKEN`.
3. Generate the password hash with `make hash-password`.
4. Start the stack:

```sh
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

Pass `--env-file .env.prod` on the command line. Compose does not load that file
automatically, and values such as `FAI_DOMAIN` and `PUBLIC_URL` would be empty.

Caddy shows the browser's Basic authentication prompt and sets
`X-Forwarded-User` for the API. The backend trusts that header in
`proxy-header` mode and is not published outside the Compose network. Caddy
bypasses Basic authentication for the ICS feed, which uses its URL token.

Vite embeds `VITE_*` values when it builds the frontend. The Kubernetes image
includes `VITE_OIDC_AUTHORITY`, but the Compose stack needs a frontend without
client-side OIDC. CI publishes `ghcr.io/alialjaffer/fai-frontend:compose` for
this deployment. Build it locally with `make build-frontend-compose`.

## Desktop app

The Electron app bundles the backend and frontend. It uses one SQLite database,
listens on a random loopback port, and runs without authentication. Attachments
are stored beside the database.

```sh
make electron-dev
make electron-dist
```

The desktop app checks for updates with `electron-updater`.

Data locations:

- macOS: `~/Library/Application Support/fai/`
- Windows: `%APPDATA%\fai\`
- Linux: `~/.config/fai/`

The first launch creates `fai.db`, `attachments/`, and `backend.log`. Optional
integrations read settings from the process environment or a `.env` file in the
application-data directory. The app sets `DATABASE_URL`, `BIND_ADDR`, `PORT`,
and `DISABLE_AUTH` itself.

## Releases

Tag a release with a `v*` version. The workflow builds macOS arm64, macOS x64,
Linux x64, and Windows x64 packages, attaches them to the GitHub release, and
publishes the Homebrew cask. Release notes come from `.github/changelog.py`, which
lists the commits since the previous tag and prepends the same entry to
`CHANGELOG.md` on `main`.

The macOS app is signed and notarized when these repository secrets are set:
`CSC_LINK` (base64 Developer ID Application certificate), `CSC_KEY_PASSWORD`,
`APPLE_ID`, `APPLE_APP_SPECIFIC_PASSWORD`, and `APPLE_TEAM_ID`. Without them the
macOS job builds an unsigned package and skips notarization. Add
`--config.mac.notarize=false` to a local `electron-builder` run for the same
fallback.

Installed copies check GitHub Releases every six hours and show an update badge.
Downloads wait for approval by default. Settings can enable automatic
downloads, show the current version, and start a manual check.

Install on macOS:

```sh
brew install --cask alialjaffer/tap/fai
```

Linux users can install the `tar.gz` or AppImage from the release page.

## Configuration

### Authentication

Set `AUTH_MODE` to choose how the API authenticates requests:

- `oidc` is the default. It checks bearer JWTs against the `fai` client in the
  `homelab` Keycloak realm. The frontend uses PKCE login.
- `proxy-header` trusts `X-Forwarded-User` after Caddy validates Basic
  authentication. Use this mode only when clients cannot reach the backend
  except through the trusted proxy.

`DISABLE_AUTH` and `VITE_DISABLE_AUTH` bypass authentication for local
development. Do not set either in a real deployment. The backend refuses to
start if the selected auth mode lacks its required configuration.

### Database

Set `DATABASE_URL` to select the database:

- `postgres://...` selects PostgreSQL. The backend applies pending goose
  migrations at startup.
- `sqlite://path/to/file.db` selects SQLite. The backend creates the schema on
  first run.

Both use the `internal/store.Store` interface. Implementations are in
`backend/internal/store/pg` and `backend/internal/store/lite`.

### Attachments

Tasks and notes accept files up to 25 MiB through
`POST /api/tasks/{id}/attachments` and `POST /api/notes/{id}/attachments`.
Metadata is stored in the database. File bytes are stored under
`ATTACHMENTS_DIR`. Uploads require `ATTACHMENTS_DIR` and stop when storage
reaches 10 GiB.

### Calendar feed

Subscribe to `GET /api/calendar.ics?token=<ICS_TOKEN>` from Apple Calendar or
another ICS client. The read-only feed contains open tasks with due dates. It
does not include calendars imported into the web app.

Set `ICS_TOKEN` and `PUBLIC_URL`. Generate the token with
`openssl rand -hex 32`. `PUBLIC_URL` supplies task links inside events.
Changing the token breaks existing subscriptions. In Docker Compose, Caddy
routes the ICS path around Basic authentication.

### Notifications

Reminders and meeting-outcome prompts use every configured channel. Both
channels are optional. The scheduler does nothing when neither is configured.
If one channel fails but another succeeds, fai marks the item sent.

- ntfy: `NTFY_URL`, `NTFY_TOPIC`, `NTFY_USERNAME`, and `NTFY_PASSWORD`
- Email: `SMTP_ADDR`, `SMTP_FROM`, and comma-separated `SMTP_TO`

### AI

Set `AI_API_KEY` to enable task breakdowns, note plans, meeting extraction, and
chat. `AI_PROVIDER` accepts `anthropic`, `openai`, and `openai-compat` (also
called `opencode-zen`). Set `AI_MODEL` and, for OpenAI-compatible services,
`AI_BASE_URL`.

Settings can also store the provider, model, base URL, and key in the database.
Saved settings override environment values. AI endpoints are rate-limited per
user. The UI hides AI actions when neither source has a key.

Set `ELEVENLABS_API_KEY` to use ElevenLabs for voice-note transcription instead
of the chat provider. `ELEVENLABS_STT_MODEL` defaults to `scribe_v2`. The key
needs the `speech_to_text` permission.

### Telegram assistant

The Telegram assistant can create tasks and reminders, find tasks, and move a
task after confirmation. It runs inside the API process and needs an AI key.

Set an owner allowlist. Only private chats are accepted. Voice notes are
transcribed and echoed as text. ElevenLabs handles transcription when
`ELEVENLABS_API_KEY` is set. Otherwise, the chat provider must accept audio;
Anthropic does not.

If the transcription service rejects Telegram OGG/Opus, the backend converts
the audio to mono MP3 with the bundled ffmpeg binary and retries. The last 10
messages per chat stay in memory and are lost on restart. Message IDs are
remembered for 10 minutes to prevent duplicate tasks after transport retries.

Every inbound and outbound bot message is written to the API log as one line:

```
telegram: chat <chat_id> user: <message>
telegram: chat <chat_id> bot: <reply>
```

Newlines are flattened so each message stays on one line. Query a chat with:

```sh
docker compose -f docker-compose.prod.yml logs backend | grep 'telegram: chat 123456789'
```

Telegram uses `getUpdates` long polling and needs no public route. Configure:

- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_ALLOWED_USER_ID`
- `TELEGRAM_ALLOWED_CHAT_ID`
- `TELEGRAM_STATE_FILE` (optional, defaults to `fai-telegram-offset`)

### Jira

Set `JIRA_BASE_URL`, `JIRA_EMAIL`, and `JIRA_API_TOKEN` to enable the bridge.
`JIRA_DEFAULT_PROJECT_KEY` supplies the project when an import has no JQL or an
export has no project key. Set `JIRA_STATUS_COLUMN_MAP` as a comma-separated
`Jira Status:Board Column` list. Unmapped statuses use the first board column.

Jira Cloud must support `/rest/api/3/search/jql`. The older
`/rest/api/3/search` endpoint returns `410 Gone`.

Issue types map between Jira and fai:

- Jira Epic maps to Arc.
- Sub-task maps to Step.
- Other types, including Story and Bug, map to Task.

Set `JIRA_EPIC_ISSUE_TYPE` (default `Epic`), `JIRA_TASK_ISSUE_TYPE` (default
`Task`), and `JIRA_SUBTASK_ISSUE_TYPE` (default `Sub-task`) to name target
types. `JIRA_EPIC_LINK_FIELD` defaults to `parent`. For company-managed
projects, set it to the project's field ID, often `customfield_10014`.

### Metrics

Set `METRICS_ADDR` (for example, `:9090`) to serve Prometheus metrics on a
separate listener. Metrics are disabled by default. The Compose stack does not
publish an extra port. The Kubernetes deployment sets this value and scrapes it
through the `fai-backend` ServiceMonitor.

The listener has no authentication. Bind it to a private interface and restrict
access at the network level.

HTTP metrics:

- `fai_http_requests_total`
- `fai_http_request_duration_seconds`
- `fai_http_requests_in_flight`

Business gauges:

- `fai_tasks`
- `fai_tasks_overdue`
- `fai_tasks_by_priority`
- `fai_tasks_completed_7d`
- `fai_notes_total`
- `fai_reminders_pending`

## Development checks

```sh
cd backend && go vet ./... && go test ./... && go build ./...
cd frontend && npm run lint && npm run build
```

## Privacy

fai stores data in the configured database and attachments directory. It has no
telemetry. Optional AI calls send requested text to the provider you configure.
Jira and calendar integrations connect only to the servers you configure. See
the [security policy](../SECURITY.md).
