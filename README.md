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

- `postgres://...` (kubernetes deployment) — standard Postgres via CNPG.
- `sqlite://path/to/file.db` (docker-compose deployment) — a single SQLite file; the
  backend applies its schema itself on first run, no separate migration step.

Both are driven through the same `internal/store.Store` interface, so the API behaves
identically regardless of which one is active — see `internal/store/pg` and
`internal/store/lite`.

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
docker build -t ghcr.io/alialjaffer/fai-frontend:compose frontend/
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
