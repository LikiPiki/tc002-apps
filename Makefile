DEV_COMPOSE = docker compose -p tc002-dev -f dev/compose.yaml
TEXT ?= Hello from local Go
export TEXT

.PHONY: help check build docker-build dev dev-build dev-down dev-logs dev-notify dev-check

help:
	@printf '%s\n' 'make check         Go format, vet and tests' 'make build         Build local binaries into bin/' 'make docker-build  Build production Go images' 'make dev           Build/start virtual AWTRIX at http://localhost:8080' 'make dev-notify    Send a notification (TEXT="...")' 'make dev-check     Build/start AWTRIX and send a smoke notification' 'make dev-logs      Follow AWTRIX logs' 'make dev-down      Stop dev environment, preserving data'

check:
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal; exit 1)
	go vet ./...
	go test ./...

build:
	go build -o bin/worker ./cmd/worker
	go build -o bin/clockctl ./cmd/clockctl

docker-build:
	docker compose --env-file deploy/.env.example -f deploy/compose.yaml --profile tools build

dev-build:
	$(DEV_COMPOSE) --profile tools build

dev:
	$(DEV_COMPOSE) up -d --build --wait awtrix
	@printf '%s\n' 'Virtual TC002: http://localhost:8080'

dev-down:
	$(DEV_COMPOSE) down

dev-logs:
	$(DEV_COMPOSE) logs -f awtrix

dev-notify:
	$(DEV_COMPOSE) run --rm --build clockctl --text "$$TEXT"

dev-check: dev
	$(DEV_COMPOSE) run --rm --build worker --once
	$(DEV_COMPOSE) run --rm --build clockctl --text 'Local integration check passed'
