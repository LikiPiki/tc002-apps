#!/bin/sh
# Run on the VPS from any directory: sh /path/to/tc002-apps/deploy/deploy.sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
if [ ! -f "$ROOT/deploy/.env" ]; then
    printf '%s\n' 'Create deploy/.env from deploy/.env.example before deployment.' >&2
    exit 1
fi

docker compose --env-file "$ROOT/deploy/.env" -f "$ROOT/deploy/compose.yaml" config --quiet
docker compose --env-file "$ROOT/deploy/.env" -f "$ROOT/deploy/compose.yaml" up -d --build --remove-orphans
docker compose --env-file "$ROOT/deploy/.env" -f "$ROOT/deploy/compose.yaml" ps
