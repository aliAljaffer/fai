SHELL := /bin/bash

.PHONY: up down up-prod down-prod dev-db-up dev-db-down migrate-up migrate-new sqlc-generate sqlc-generate-sqlite backend-run frontend-run build build-images push-images compose-build electron-dev electron-dist

GOOSE := $(HOME)/go/bin/goose
SQLC := $(HOME)/go/bin/sqlc
DB_URL := postgres://fai:fai@localhost:5432/fai?sslmode=disable
REGISTRY := ghcr.io/alialjaffer
OIDC_AUTHORITY ?= https://kc.alialjaffer.com/realms/homelab

# Local stack, from this checkout: http://localhost:8080
up:
	docker compose up -d --build

down:
	docker compose down

# Production stack. Needs FAI_DOMAIN and FAI_AUTH_PASSWORD in .env.
up-prod:
	docker compose -f docker-compose.prod.yml up -d

down-prod:
	docker compose -f docker-compose.prod.yml down

dev-db-up:
	docker compose --profile postgres up -d postgres

dev-db-down:
	docker compose stop postgres

migrate-up:
	$(GOOSE) -dir backend/migrations postgres "$(DB_URL)" up

migrate-new:
	$(GOOSE) -dir backend/migrations create $(name) sql

sqlc-generate:
	cd backend && $(SQLC) generate

sqlc-generate-sqlite:
	cd backend && $(SQLC) generate -f sqlc.sqlite.yaml

backend-run:
	cd backend && DISABLE_AUTH=true go run ./cmd/api

frontend-run:
	cd frontend && npm run dev

build:
	cd backend && go build ./...
	cd frontend && npm run build

# Kubernetes images bake in the OIDC authority; CI publishes the same thing
# tagged by commit SHA. The compose deployment needs the separate target below
# because Caddy already gates every request there.
build-images:
	docker build -t $(REGISTRY)/fai-backend:latest backend/
	docker build -t $(REGISTRY)/fai-frontend:latest --build-arg VITE_OIDC_AUTHORITY=$(OIDC_AUTHORITY) frontend/

push-images:
	docker login $(REGISTRY) -u alialjaffer --password-stdin < $(HOME)/ghcr.txt
	docker push $(REGISTRY)/fai-backend:latest
	docker push $(REGISTRY)/fai-frontend:latest

# Builds the images docker-compose.prod.yml uses, for running it from source.
compose-build:
	docker build -t $(REGISTRY)/fai-backend:latest backend/
	docker build -t $(REGISTRY)/fai-frontend:compose frontend/

electron-dev:
	cd frontend && npm run build
	cd backend && go build -o bin/fai-backend ./cmd/api
	cd electron && npm install && npm start

electron-dist:
	cd frontend && npm run build
	cd backend && go build -trimpath -ldflags="-s -w" -o bin/fai-backend ./cmd/api
	cd electron && npm install && npm run dist
