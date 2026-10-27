.PHONY: dev-db-up dev-db-down migrate-up migrate-new sqlc-generate backend-run frontend-run build

GOOSE := $(HOME)/go/bin/goose
SQLC := $(HOME)/go/bin/sqlc
DB_URL := postgres://fai:fai@localhost:5432/fai?sslmode=disable

dev-db-up:
	docker compose up -d postgres

dev-db-down:
	docker compose down

migrate-up:
	$(GOOSE) -dir backend/migrations postgres "$(DB_URL)" up

migrate-new:
	$(GOOSE) -dir backend/migrations create $(name) sql

sqlc-generate:
	cd backend && $(SQLC) generate

backend-run:
	cd backend && go run ./cmd/api

frontend-run:
	cd frontend && npm run dev

build:
	cd backend && go build ./...
	cd frontend && npm run build
