# fai

Self-hosted personal task tracker / Kanban / notetaking app. See `PLAN.md` for the full design and `PROGRESS.md` for build status.

## Local dev

```sh
make dev-db-up      # Postgres via Docker
make migrate-up      # apply schema
cp backend/.env.example backend/.env
make backend-run      # go run ./cmd/api, DISABLE_AUTH=true
make frontend-run      # vite dev, proxies /api to :8080 (VITE_DISABLE_AUTH=true via .env.development)
```

Backend: Go (chi, pgx, sqlc, goose). Frontend: Vite + React SPA.

## Auth

Two modes, selected by `AUTH_MODE`:

- `oidc` (default, used by the kubernetes deployment) — the backend validates bearer
  JWTs itself via go-oidc against the `fai` public client in Keycloak's `homelab`
  realm (`https://kc.alialjaffer.com/realms/homelab`), frontend does PKCE login.
- `proxy-header` (used by the docker-compose deployment) — the backend trusts an
  `X-Forwarded-User` header set by Caddy after it validates HTTP Basic auth, instead
  of validating a token itself. Only safe when the backend is unreachable except
  through that proxy (enforced by docker-compose's network: the backend's port is
  never published to the host).

`DISABLE_AUTH`/`VITE_DISABLE_AUTH` bypass auth entirely for local dev only — never set
either in a real deployment; the backend fails closed (refuses to start) if auth is
enabled without the required config for whichever mode is selected.

## Database

Two engines, selected by `DATABASE_URL`'s scheme:

- `postgres://...` (kubernetes deployment) — standard Postgres via CNPG; the backend
  applies pending goose migrations itself on startup.
- `sqlite://path/to/file.db` (docker-compose deployment) — a single SQLite file; the
  backend applies its schema itself on first run.

Both are driven through the same `internal/store.Store` interface, so the API behaves
identically regardless of which one is active — see `internal/store/pg` and
`internal/store/lite`.

## Attachments

Tasks and notes accept file attachments (25 MiB each) over
`POST /api/tasks/{id}/attachments` and `POST /api/notes/{id}/attachments`; metadata
lives in the database, bytes live under `ATTACHMENTS_DIR`. The kubernetes deployment
mounts a Longhorn `ReadWriteMany` PVC at `/data/attachments`; docker-compose uses
`/data/attachments` inside the existing `faidata` volume. Leave `ATTACHMENTS_DIR`
unset to disable uploads.

## Calendar subscription

`GET /api/calendar.ics?token=<ICS_TOKEN>` is a read-only ICS feed of open tasks with
a due date, suitable for Apple Calendar (File → New Calendar Subscription). It sits
outside OIDC because calendar clients cannot log in, so the token is the credential;
the Settings page shows the full URL. Set `ICS_TOKEN` (`openssl rand -hex 32`) and
`PUBLIC_URL` (used for the task links inside events); rotating the token invalidates
existing subscriptions. In docker-compose, Caddy routes this one path around
`forward_auth`.

## Navigation

`⌘K`/`Ctrl+K` opens the command palette (pages, tasks, new task). `n` creates a task,
`e` focuses the open task's title, and `Esc` closes sheets/dialogs. The List page
filters by text, column, and tag; rows open the task sheet.

## Notifications

Reminders and meeting-outcome prompts go out over every configured channel; both are
optional, and the scheduler no-ops when neither is set. If one channel fails while the
other delivers, the item is marked sent so the working channel is not re-notified every
minute.

- **ntfy** — `NTFY_URL`, `NTFY_TOPIC`, `NTFY_USERNAME`, `NTFY_PASSWORD`. The
  kubernetes deployment publishes to the in-cluster `ntfy` Service.
- **Email** — `SMTP_ADDR`, `SMTP_FROM`, `SMTP_TO` (comma-separated recipients). The
  kubernetes deployment points at the cluster's rate-limited
  `smtp-relay.mail.svc.cluster.local:587`, which accepts unauthenticated mail from
  allowed namespaces and handles the upstream provider's credentials; fai itself holds
  no SMTP credentials.

## AI

`AI_API_KEY` enables the optional task-breakdown and chat features; `AI_PROVIDER=opencode-zen`
talks to OpenCode Zen's OpenAI-compatible endpoint with `AI_MODEL` (for example
`claude-haiku-4-5`), while any other value uses Anthropic's Messages API. The endpoints
are rate-limited per authenticated subject. Unset means the UI hides the AI actions.

## Jira

`JIRA_BASE_URL`, `JIRA_EMAIL`, `JIRA_API_TOKEN` enable the bridge.
`JIRA_DEFAULT_PROJECT_KEY` is used when an import omits JQL and an export omits a
project key; `JIRA_STATUS_COLUMN_MAP` is a comma-separated `Jira Status:Board Column`
list applied on import (unmapped statuses fall into the board's first column). Jira
Cloud's `/rest/api/3/search/jql` endpoint is required — the older `/rest/api/3/search`
returns `410 Gone`.

## Docker Compose deployment

A complete non-Kubernetes alternative: SQLite (no separate database container),
bundled ntfy, and Caddy for HTTP Basic auth and TLS. Prerequisites are just Docker
and Docker Compose -- no identity provider or extra auth service.

**Frontend image note:** `VITE_OIDC_AUTHORITY` is baked into the frontend bundle at
*build* time (Vite inlines `VITE_*` vars), so the published
`ghcr.io/alialjaffer/fai-backend`/`fai-frontend:latest` images — built with that
authority set for the kubernetes deployment — aren't quite right for compose: here,
Caddy gates every request (including the initial page load) with Basic auth before
the browser ever reaches the SPA, so the frontend must do *no* client-side OIDC of
its own (a baked-in authority would make it try its own login). CI therefore also
publishes `ghcr.io/alialjaffer/fai-frontend:compose` with that build arg omitted, and
the compose file uses it; `make build-frontend-compose` rebuilds it locally if
needed.

Then:

```sh
cp .env.prod.example .env.prod   # fill in FAI_DOMAIN, FAI_AUTH_USER, FAI_AUTH_HASH
make hash-password               # prompts, prints the bcrypt hash for FAI_AUTH_HASH
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

`--env-file .env.prod` is required (not just `env_file:` inside the compose file) so
the `${FAI_DOMAIN}` used for Caddy's site address and `PUBLIC_URL` resolves — compose
only auto-loads a file literally named `.env` otherwise. The browser gets a native
Basic auth prompt; credentials are cached for the origin and sent on the SPA's API
calls. Caddy sets `X-Forwarded-User` for `/api`, which the backend trusts in
`proxy-header` mode while remaining unpublished outside the compose network. The
`/api/calendar.ics` feed deliberately bypasses Basic auth because calendar clients
use its URL token instead.
