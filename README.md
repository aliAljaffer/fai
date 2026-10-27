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

Keycloak OIDC, PKCE, against the `fai` public client in the `homelab` realm (`https://kc.alialjaffer.com/realms/homelab`). `DISABLE_AUTH`/`VITE_DISABLE_AUTH` bypass it for local dev only — never set either in a real deployment; the backend fails closed (refuses to start) if auth is enabled without `OIDC_ISSUER_URL`/`OIDC_AUDIENCE` configured.
