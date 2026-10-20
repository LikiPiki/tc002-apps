#!/bin/sh
# Read server.apps and tools.apps using the shared runtime TOML parser.
set -eu
CONFIG=${1:-${CONFIG:-configs/config.toml}}
LOCAL_CONFIG=${2-${LOCAL_CONFIG-configs/config.local.toml}}

if [ "${CONFIG_RUNNER:-auto}" != docker ] && command -v go >/dev/null 2>&1; then
    exec go run ./apps/tools/cmd/clockctl build-list --group "${GROUP:-all}" --config "$CONFIG" --local-config "$LOCAL_CONFIG"
fi

# Docker-only users do not need Go installed on the build machine.
docker build -f apps/tools/Dockerfile --build-arg APP=clockctl -t tc002-apps/tools/clockctl:config . >/dev/null
exec docker run --rm --workdir /repo \
    --mount "type=bind,source=$(pwd),target=/repo,readonly" \
    tc002-apps/tools/clockctl:config build-list --group "${GROUP:-all}" --config "$CONFIG" --local-config "$LOCAL_CONFIG"
