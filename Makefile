SHELL := /bin/bash

.PHONY: dev-db-up dev-db-down migrate-up migrate-new sqlc-generate sqlc-generate-sqlite backend-run frontend-run build build-images push-images build-frontend-compose hash-password compose-prod-up compose-prod-down

GOOSE := $(HOME)/go/bin/goose
SQLC := $(HOME)/go/bin/sqlc
DB_URL := postgres://fai:fai@localhost:5432/fai?sslmode=disable
REGISTRY := ghcr.io/alialjaffer
OIDC_AUTHORITY ?= https://kc.alialjaffer.com/realms/homelab

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

sqlc-generate-sqlite:
	cd backend && $(SQLC) generate -f sqlc.sqlite.yaml

compose-prod-up:
	docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

compose-prod-down:
	docker compose -f docker-compose.prod.yml --env-file .env.prod down

backend-run:
	cd backend && go run ./cmd/api

frontend-run:
	cd frontend && npm run dev

build:
	cd backend && go build ./...
	cd frontend && npm run build

# Kubernetes images bake in the OIDC authority; CI publishes the same thing
# tagged by commit SHA. The compose deployment needs the separate target below
# because oauth2-proxy already gates every request there.
build-images:
	docker build -t $(REGISTRY)/fai-backend:latest backend/
	docker build -t $(REGISTRY)/fai-frontend:latest --build-arg VITE_OIDC_AUTHORITY=$(OIDC_AUTHORITY) frontend/

push-images:
	docker login $(REGISTRY) -u alialjaffer --password-stdin < $(HOME)/ghcr.txt
	docker push $(REGISTRY)/fai-backend:latest
	docker push $(REGISTRY)/fai-frontend:latest

build-frontend-compose:
	docker build -t $(REGISTRY)/fai-frontend:compose frontend/

# Prints the bcrypt hash already $$-escaped for Compose's env_file interpolation.
hash-password:
	@read -rsp 'Password: ' pw; echo; \
	printf '%s\n' "$$pw" | docker run --rm -i caddy:alpine caddy hash-password | sed 's/\$$/\$$\$$/g'
