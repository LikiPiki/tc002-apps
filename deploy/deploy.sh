#!/bin/sh
# Run on the VPS: sh deploy/deploy.sh [--no-build]
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
if [ ! -f "$ROOT/configs/config.toml" ]; then
    printf '%s\n' 'Missing configs/config.toml; run make init-config and configure your clocks/location first.' >&2
    exit 1
fi

BUILD_FLAG=--build
if [ "${1:-}" = "--no-build" ]; then
    BUILD_FLAG=--no-build
    shift
fi
if [ "$#" -ne 0 ]; then
    printf '%s\n' 'Usage: sh deploy/deploy.sh [--no-build]' >&2
    exit 2
fi

compose() {
    docker compose -f "$ROOT/deploy/compose.yaml" "$@"
}

compose config --quiet
compose up -d "$BUILD_FLAG" --remove-orphans server
compose ps server
