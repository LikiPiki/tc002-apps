#!/bin/sh
# Create a local working config once; never replace an existing path.
set -eu

TARGET=${1:-${CONFIG:-configs/config.toml}}
TEMPLATE=${2:-configs/config.toml.example}

if [ -e "$TARGET" ] || [ -L "$TARGET" ]; then
    printf 'Keeping existing config: %s\n' "$TARGET"
    exit 0
fi

if [ ! -f "$TEMPLATE" ]; then
    printf 'Missing config template: %s\n' "$TEMPLATE" >&2
    exit 1
fi
if [ ! -d "$(dirname -- "$TARGET")" ]; then
    printf 'Config parent directory does not exist: %s\n' "$TARGET" >&2
    exit 1
fi

# A bind-mounted config must be readable by the non-root container user.
# noclobber also preserves a config created by another concurrent initializer.
if (
    umask 022
    set -C
    cat "$TEMPLATE" >"$TARGET"
) 2>/dev/null; then
    printf 'Created config: %s\n' "$TARGET"
elif [ -e "$TARGET" ] || [ -L "$TARGET" ]; then
    printf 'Keeping existing config: %s\n' "$TARGET"
else
    printf 'Unable to create config: %s\n' "$TARGET" >&2
    exit 1
fi
