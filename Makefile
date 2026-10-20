DEV_COMPOSE = docker compose -p tc002-dev -f dev/compose.yaml
TEXT ?= Hello from local Go
CONFIG ?= configs/apps.list
APP ?=
CLOCK_CONFIG ?= configs/clock.toml
CLOCK_LOCAL_CONFIG ?= configs/clock.local.toml
export TEXT CONFIG APP CLOCK_CONFIG CLOCK_LOCAL_CONFIG

.PHONY: help check build docker-build docker-save list-apps dev dev-build dev-down dev-logs dev-notify dev-check clock-list clock-deploy clock-show

help:
	@printf '%s\n' 'make list-apps      Show selected apps (configs/apps.list; absent = all)' 'make check         Go format, vet and tests' 'make build         Build selected Go binaries into bin/' 'make docker-build  Build selected Go images' 'make docker-save   Export selected images to dist/<app>.tar.gz' 'make dev           Build/start virtual AWTRIX at http://localhost:8080' 'make dev-notify    Send a notification (TEXT="...")' 'make dev-check     Build/start AWTRIX and send a smoke notification' 'make dev-logs      Follow AWTRIX logs' 'make dev-down      Stop dev environment, preserving data'
	@printf '%s\n' 'make clock-list    Show target and selected Berry apps' 'make clock-deploy  Upload Berry apps (APP=<name> selects one)' 'make clock-show APP=<name>  Show an installed Berry app'

clock-list clock-deploy clock-show:
	go run ./apps/server/cmd/clockctl $(patsubst clock-%,%,$@) --config "$$CLOCK_CONFIG" --local-config "$$CLOCK_LOCAL_CONFIG" --app "$$APP"

list-apps:
	@sh tools/select-apps.sh "$$CONFIG"

check:
	@test -z "$$(gofmt -l apps/server)" || (gofmt -l apps/server; exit 1)
	go vet ./...
	go test ./...

build:
	@set -eu; apps=$$(sh tools/select-apps.sh "$$CONFIG"); \
	for app in $$apps; do go build -o "bin/$$app" "./apps/server/cmd/$$app"; done

docker-build:
	@set -eu; apps=$$(sh tools/select-apps.sh "$$CONFIG"); \
	for app in $$apps; do docker build --build-arg "APP=$$app" -t "tc002-apps/$$app:local" .; done

docker-save: docker-build
	@set -eu; apps=$$(sh tools/select-apps.sh "$$CONFIG"); mkdir -p dist; \
	for app in $$apps; do \
		docker image save -o "dist/$$app.tar" "tc002-apps/$$app:local"; \
		gzip -f "dist/$$app.tar"; \
	done

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
