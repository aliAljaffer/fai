# fai

Self-hosted personal task tracker / Kanban / notetaking app. See `PLAN.md` for the full design and `PROGRESS.md` for build status.

## Local dev

```sh
make dev-db-up      # Postgres via Docker
make migrate-up      # apply schema
cp backend/.env.example backend/.env
make backend-run      # go run ./cmd/api, DISABLE_AUTH=true
make frontend-run      # vite dev, proxies /api to :8080
```

Backend: Go (chi, pgx, sqlc, goose). Frontend: Vite + React SPA.
