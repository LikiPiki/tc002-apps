#!/bin/sh
# Run on the VPS: sh deploy/deploy.sh [--no-build] [service ...]
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
if [ ! -f "$ROOT/configs/config.toml" ]; then
    printf '%s\n' 'Missing configs/config.toml; deploy the shared project config first.' >&2
    exit 1
fi

BUILD_FLAG=--build
if [ "${1:-}" = "--no-build" ]; then
    BUILD_FLAG=--no-build
    shift
fi
# Never implicitly start all services as the project grows.
if [ "$#" -eq 0 ]; then
    set -- worker
fi

compose() {
    docker compose -f "$ROOT/deploy/compose.yaml" "$@"
}

compose config --quiet
SERVICES=$(compose --profile tools config --services)
for service in "$@"; do
    FOUND=false
    for known in $SERVICES; do
        if [ "$service" = "$known" ]; then FOUND=true; break; fi
    done
    if [ "$FOUND" = false ]; then
        printf 'Unknown service: %s\n' "$service" >&2
        exit 1
    fi
done

compose up -d "$BUILD_FLAG" --no-deps "$@"
compose ps "$@"
