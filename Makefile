DEV_COMPOSE = docker compose -p tc002-dev -f dev/compose.yaml
LAN_COMPOSE = docker compose -p tc002-lan -f dev/compose.lan.yaml
SERVICE ?= weather-provider
TEXT ?= Hello from local Go
CONFIG ?= configs/config.toml
LOCAL_CONFIG ?= configs/config.local.toml
APP ?=
KIND ?= all
GROUP ?= all
CHECK_DURATION ?= 2s
CHECK_BUILD ?= 1
CHECK_ALL ?= false
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
LDFLAGS = -s -w \
	-X github.com/LikiPiki/tc002-apps/internal/buildinfo.Version=$(VERSION) \
	-X github.com/LikiPiki/tc002-apps/internal/buildinfo.Commit=$(COMMIT)
export TEXT CONFIG LOCAL_CONFIG APP KIND GROUP CHECK_DURATION CHECK_BUILD CHECK_ALL VERSION COMMIT

.PHONY: help check build docker-build docker-save list-apps dev dev-build dev-down dev-logs dev-notify dev-check clock-list clock-deploy clock-show clock-start clock-stop clock-check

help:
	@printf '%s\n' \
		'make list-apps      Show selected Go artifacts (GROUP=all|server|tools)' \
		'make check          Go format, vet and tests' \
		'make build          Build selected Go binaries into bin/<group>/' \
		'make docker-build   Build isolated server/tool images' \
		'make docker-save    Export selected images to dist/<group>/<app>.tar.gz' \
		'make release        Build cross-platform clockctl archives and checksums' \
		'make dev            Build/start virtual AWTRIX at http://localhost:8080' \
		'make dev-notify     Send a notification (TEXT="...")' \
		'make dev-check      Build/start AWTRIX and send a smoke notification' \
		'make dev-logs       Follow AWTRIX logs' \
		'make dev-down       Stop dev environment, preserving data' \
		'make lan            Build/start a local server for the physical clock (SERVICE=weather-provider)' \
		'make lan-logs       Follow local server logs' \
		'make lan-once       Fetch/publish once from a local container' \
		'make lan-restart    Reload local server config' \
		'make lan-doctor     Diagnose the physical clock from the container network' \
		'make lan-down       Stop local LAN servers' \
		'make clock-list     Show target and selected Berry apps' \
		'make clock-validate Validate clock settings offline' \
		'make clock-deploy   Upload Berry apps (APP=<name> selects one)' \
		'make clock-check    Check Berry apps in disposable AWTRIX (CHECK_ALL=true for all apps)' \
		'make weather-check  Check weather producer and both weather screens' \
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

.PHONY: weather-check
weather-check:
	sh scripts/check-weather.sh

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

list-apps:
	@sh scripts/select-apps.sh "$$CONFIG" "$$LOCAL_CONFIG"

check:
	@test -z "$$(gofmt -l apps/server apps/tools internal)" || (gofmt -l apps/server apps/tools internal; exit 1)
	go vet ./...
	go test ./...

build:
	@set -eu; targets=$$(sh scripts/select-apps.sh "$$CONFIG" "$$LOCAL_CONFIG"); \
	for target in $$targets; do group=$${target%%/*}; app=$${target#*/}; \
		go build -trimpath -ldflags "$(LDFLAGS)" -o "bin/$$target" "./apps/$$group/cmd/$$app"; done

docker-build:
	@set -eu; targets=$$(sh scripts/select-apps.sh "$$CONFIG" "$$LOCAL_CONFIG"); \
	for target in $$targets; do group=$${target%%/*}; app=$${target#*/}; \
		docker build -f "apps/$$group/Dockerfile" --build-arg "APP=$$app" \
			--build-arg "VERSION=$$VERSION" --build-arg "COMMIT=$$COMMIT" \
			-t "tc002-apps/$$target:local" .; done

.PHONY: release
release:
	sh scripts/release.sh

docker-save: docker-build
	@set -eu; targets=$$(sh scripts/select-apps.sh "$$CONFIG" "$$LOCAL_CONFIG"); \
	for target in $$targets; do group=$${target%%/*}; mkdir -p "dist/$$group"; \
		docker image save -o "dist/$$target.tar" "tc002-apps/$$target:local"; \
		gzip -f "dist/$$target.tar"; \
	done

dev-build:
	$(DEV_COMPOSE) --profile tools --profile server build

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

.PHONY: lan lan-logs lan-once lan-restart lan-doctor lan-down

lan:
	$(LAN_COMPOSE) up -d --build --wait "$(SERVICE)"

lan-logs:
	$(LAN_COMPOSE) logs -f --tail=100 "$(SERVICE)"

lan-once:
	$(LAN_COMPOSE) run --rm --build --no-deps "$(SERVICE)" --once

lan-restart:
	$(LAN_COMPOSE) restart "$(SERVICE)"

lan-doctor:
	$(LAN_COMPOSE) run --rm --build --no-deps clockctl doctor

lan-down:
	$(LAN_COMPOSE) --profile '*' down
