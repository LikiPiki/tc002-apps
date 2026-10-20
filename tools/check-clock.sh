#!/bin/sh
# Each invocation gets its own network and an ephemeral, port-free AWTRIX.
set -eu
PROJECT="tc002-clock-check-$$"
compose() {
    docker compose -p "$PROJECT" -f dev/compose.yaml --profile checks "$@"
}
cleanup() {
    status=$?
    trap - EXIT
    if [ "$status" -ne 0 ]; then
        compose logs --tail=80 awtrix-check >&2 || true
    fi
    compose down --volumes --remove-orphans >/dev/null 2>&1 || true
    exit "$status"
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

compose build awtrix-check clock-check
compose up -d --wait awtrix-check
compose run --rm --no-deps clock-check check \
    --url http://awtrix-check:8080 \
    --config "${CONFIG:-configs/config.toml}" \
    --local-config "${LOCAL_CONFIG-configs/config.local.toml}" \
    --app "${APP:-}" --duration "${CHECK_DURATION:-2s}"
