#!/bin/sh
# Run the weather producer and both real Berry consumers in an isolated clock.
set -eu

PROJECT="tc002-weather-check-$$"
compose() {
    docker compose -p "$PROJECT" -f dev/compose.yaml --profile checks "$@"
}

cleanup() {
    status=$?
    trap - EXIT
    if [ "$status" -ne 0 ]; then
        compose logs awtrix-check >&2 || true
    fi
    compose down --volumes --remove-orphans >/dev/null 2>&1 || true
    exit "$status"
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

case "${CHECK_BUILD:-1}" in
    1) compose build awtrix-check weather-check ;;
    0) : ;;
    *)
        printf '%s\n' 'CHECK_BUILD must be 0 or 1' >&2
        exit 2
        ;;
esac

compose up -d --no-build --pull never --wait awtrix-check
compose run --rm --no-deps weather-check
