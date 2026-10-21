DEV_COMPOSE = docker compose -p tc002-dev -f dev/compose.yaml
TEXT ?= Hello from local Go
CONFIG ?= configs/config.toml
LOCAL_CONFIG ?= configs/config.local.toml
APP ?=
KIND ?= all
GROUP ?= all
CHECK_DURATION ?= 2s
CHECK_BUILD ?= 1
export TEXT CONFIG LOCAL_CONFIG APP KIND GROUP CHECK_DURATION CHECK_BUILD

.PHONY: help check build docker-build docker-save list-apps dev dev-build dev-down dev-logs dev-notify dev-check clock-list clock-deploy clock-show clock-start clock-stop clock-check

help:
	@printf '%s\n' 'make list-apps      Show selected Go artifacts (GROUP=all|server|tools)' 'make check         Go format, vet and tests' 'make build         Build selected Go binaries into bin/<group>/' 'make docker-build  Build isolated server/tool images' 'make docker-save   Export selected images to dist/<group>/<app>.tar.gz' 'make dev           Build/start virtual AWTRIX at http://localhost:8080' 'make dev-notify    Send a notification (TEXT="...")' 'make dev-check     Build/start AWTRIX and send a smoke notification' 'make dev-logs      Follow AWTRIX logs' 'make dev-down      Stop dev environment, preserving data'
	@printf '%s\n' 'make clock-list    Show target and selected Berry apps' 'make clock-deploy  Upload Berry apps (APP=<name> selects one)' 'make clock-show APP=<name>  Show an installed Berry app'
	@printf '%s\n' 'make clock-check   Check Berry apps in disposable local AWTRIX (APP=<name>, CHECK_DURATION=2s)'
	@printf '%s\n' 'make clock-start APP=<name>  Start an installed on-demand app' 'make clock-stop APP=<name>   Stop the current on-demand app' 'Use KIND=screens|services|on-demand to filter clock-list, clock-deploy, or clock-check'

clock-check:
	sh scripts/check-clock.sh

.PHONY: status-check clock-validate
status-check:
	sh scripts/check-status.sh

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
		go build -o "bin/$$target" "./apps/$$group/cmd/$$app"; done

docker-build:
	@set -eu; targets=$$(sh scripts/select-apps.sh "$$CONFIG" "$$LOCAL_CONFIG"); \
	for target in $$targets; do group=$${target%%/*}; app=$${target#*/}; \
		docker build -f "apps/$$group/Dockerfile" --build-arg "APP=$$app" -t "tc002-apps/$$target:local" .; done

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
