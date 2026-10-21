DEV_COMPOSE = docker compose -p tc002-dev -f dev/compose.yaml
LAN_COMPOSE = docker compose -p tc002-lan -f dev/compose.lan.yaml
TEXT ?= Hello from local Go
CONFIG ?= configs/config.toml
LOCAL_CONFIG ?= configs/config.local.toml
APP ?=
KIND ?= all
CHECK_DURATION ?= 2s
CHECK_BUILD ?= 1
CHECK_ALL ?= false
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
LDFLAGS = -s -w \
	-X github.com/LikiPiki/tc002-apps/internal/buildinfo.Version=$(VERSION) \
	-X github.com/LikiPiki/tc002-apps/internal/buildinfo.Commit=$(COMMIT)
export TEXT CONFIG LOCAL_CONFIG APP KIND CHECK_DURATION CHECK_BUILD CHECK_ALL VERSION COMMIT

.PHONY: help check build build-server build-tools docker-build docker-build-server docker-build-tools docker-save docker-save-server docker-save-tools list-tools dev dev-build dev-server dev-down dev-logs dev-notify dev-check clock-list clock-deploy clock-show clock-start clock-stop clock-check

help:
	@printf '%s\n' \
		'make list-tools     Show selected utility commands' \
		'make check          Go format, vet and tests' \
		'make build          Build the complete server' \
		'make build-server   Build bin/server/tc002-server with all server features' \
		'make build-tools    Build selected utilities into bin/tools/' \
		'make docker-build   Build the complete server image' \
		'make docker-build-server  Build tc002-apps/server:local' \
		'make docker-save-server   Export dist/server/tc002-server.tar.gz' \
		'make release        Build cross-platform clockctl archives and checksums' \
		'make dev            Build/start virtual AWTRIX at http://localhost:8080' \
		'make dev-server     Run the complete server against virtual AWTRIX' \
		'make dev-notify     Send a notification (TEXT="...")' \
		'make dev-check      Build/start AWTRIX and send a smoke notification' \
		'make dev-logs       Follow AWTRIX logs' \
		'make dev-down       Stop dev environment, preserving data' \
		'make lan            Build/start the complete server for physical clocks' \
		'make lan-logs       Follow local server logs' \
		'make lan-once       Fetch/publish once from a local container' \
		'make lan-restart    Reload local server config' \
		'make lan-doctor     Diagnose the physical clock from the container network' \
		'make lan-down       Stop local LAN servers' \
		'make clock-list     Show target and selected Berry apps' \
		'make clock-validate Validate clock settings offline' \
		'make clock-deploy   Upload Berry apps (APP=<name> selects one)' \
		'make clock-check    Check Berry apps in disposable AWTRIX (CHECK_ALL=true for all apps)' \
		'make server-check   Check server data processing and Berry consumers' \
		'make sprites        Generate Berry arrays from sprite PNGs' \
		'make sprites-check  Verify generated sprite arrays' \
		'make previews       Refresh README GIFs from runtime artifacts' \
		'make clock-show APP=<name>   Show an installed Berry app' \
		'make clock-start APP=<name>  Start an installed on-demand app' \
		'make clock-stop APP=<name>   Stop the current on-demand app' \
		'Use KIND=screens|services|on-demand to filter catalog commands and clock-check'

clock-check:
	sh scripts/check-clock.sh

.PHONY: clock-validate

.PHONY: server-check
server-check:
	sh scripts/check-server.sh

.PHONY: sprites sprites-check
sprites:
	go run ./apps/tools/cmd/spritegen

sprites-check:
	go run ./apps/tools/cmd/spritegen --check

.PHONY: previews
previews:
	go run ./apps/tools/cmd/previewgen

clock-list clock-deploy clock-show clock-start clock-stop clock-validate:
	go run ./apps/tools/cmd/clockctl $(patsubst clock-%,%,$@) --config "$$CONFIG" --local-config "$$LOCAL_CONFIG" --app "$$APP" --kind "$$KIND"

list-tools:
	@sh scripts/select-tools.sh "$$CONFIG" "$$LOCAL_CONFIG"

check:
	@test -z "$$(gofmt -l apps/server apps/tools internal)" || (gofmt -l apps/server apps/tools internal; exit 1)
	go vet ./...
	go test ./...

build: build-server

build-server:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/server/tc002-server ./apps/server/cmd/server

build-tools:
	@set -eu; tools=$$(sh scripts/select-tools.sh "$$CONFIG" "$$LOCAL_CONFIG"); \
	for tool in $$tools; do \
		go build -trimpath -ldflags "$(LDFLAGS)" -o "bin/tools/$$tool" "./apps/tools/cmd/$$tool"; \
	done

docker-build: docker-build-server

docker-build-server:
	docker build -f apps/server/Dockerfile \
		--build-arg "VERSION=$$VERSION" --build-arg "COMMIT=$$COMMIT" \
		-t tc002-apps/server:local .

docker-build-tools:
	@set -eu; tools=$$(sh scripts/select-tools.sh "$$CONFIG" "$$LOCAL_CONFIG"); \
	for tool in $$tools; do \
		docker build -f apps/tools/Dockerfile --build-arg "APP=$$tool" \
			--build-arg "VERSION=$$VERSION" --build-arg "COMMIT=$$COMMIT" \
			-t "tc002-apps/tools/$$tool:local" .; \
	done

.PHONY: release
release:
	sh scripts/release.sh

docker-save: docker-save-server

docker-save-server: docker-build-server
	mkdir -p dist/server
	docker image save -o dist/server/tc002-server.tar tc002-apps/server:local
	gzip -f dist/server/tc002-server.tar

docker-save-tools: docker-build-tools
	@set -eu; tools=$$(sh scripts/select-tools.sh "$$CONFIG" "$$LOCAL_CONFIG"); \
	mkdir -p dist/tools; \
	for tool in $$tools; do \
		docker image save -o "dist/tools/$$tool.tar" "tc002-apps/tools/$$tool:local"; \
		gzip -f "dist/tools/$$tool.tar"; \
	done

dev-build:
	$(DEV_COMPOSE) --profile tools --profile server build

dev:
	$(DEV_COMPOSE) up -d --build --wait awtrix
	@printf '%s\n' 'Virtual TC002: http://localhost:8080'

dev-server:
	$(DEV_COMPOSE) --profile server up -d --build --wait server

dev-down:
	$(DEV_COMPOSE) down

dev-logs:
	$(DEV_COMPOSE) logs -f awtrix

dev-notify:
	$(DEV_COMPOSE) run --rm --build clockctl --text "$$TEXT"

dev-check: dev
	$(DEV_COMPOSE) run --rm --build clockctl --text 'Local integration check passed'

.PHONY: lan lan-logs lan-once lan-restart lan-doctor lan-down

lan:
	$(LAN_COMPOSE) up -d --build --wait --remove-orphans server

lan-logs:
	$(LAN_COMPOSE) logs -f --tail=100 server

lan-once:
	$(LAN_COMPOSE) run --rm --build --no-deps server --once

lan-restart:
	$(LAN_COMPOSE) restart server

lan-doctor:
	$(LAN_COMPOSE) run --rm --build --no-deps clockctl doctor

lan-down:
	$(LAN_COMPOSE) --profile '*' down
