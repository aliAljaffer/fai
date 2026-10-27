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
  `X-Forwarded-User`/`X-Auth-Request-User` header set by a fronting oauth2-proxy
  instead of validating a token itself. Only safe when the backend is unreachable
  except through that proxy (enforced by docker-compose's network: the backend's
  port is never published to the host).

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
bundled ntfy, oauth2-proxy + Caddy for auth and TLS.

**Frontend image note:** `VITE_OIDC_AUTHORITY` is baked into the frontend bundle at
*build* time (Vite inlines `VITE_*` vars), so the published
`ghcr.io/alialjaffer/fai-backend`/`fai-frontend:latest` images — built with that
authority set for the kubernetes deployment — aren't quite right for compose: here,
oauth2-proxy + Caddy gate every request (including the initial page load) before the
browser ever reaches the SPA, so the frontend must do *no* client-side OIDC of its
own (it otherwise detects a baked-in authority and tries its own login, which would
conflict with the session oauth2-proxy already established). Build a compose-specific
frontend image with that build arg omitted:

```sh
make build-frontend-compose   # ghcr.io/alialjaffer/fai-frontend:compose
# update docker-compose.prod.yml's frontend image to the :compose tag, or push it
# to your own registry
```

Then:

```sh
cp .env.prod.example .env.prod   # fill in FAI_DOMAIN and an oauth2-proxy OIDC client
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

`--env-file .env.prod` is required (not just `env_file:` inside the compose file) so
Caddy's `${FAI_DOMAIN}`-based site address and oauth2-proxy's redirect URL resolve —
compose only auto-loads a file literally named `.env` otherwise.

`.env.prod` needs a confidential OIDC client registered with your identity provider
for oauth2-proxy (standard flow, not PKCE — oauth2-proxy runs server-side and can hold
a secret), redirect URI `https://<FAI_DOMAIN>/oauth2/callback`.

**Unverified**: the full Caddy → oauth2-proxy → OIDC provider login round-trip hasn't
been exercised against a real domain/DNS/OIDC client in this environment. The
Caddyfile's syntax is validated (`caddy validate`) and the `forward_auth` +
`handle_response` redirect-to-login pattern follows Caddy's documented oauth2-proxy
recipe, but the live flow is unverified.
