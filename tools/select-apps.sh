#!/bin/sh
# Print the selected Go commands, one per line. Run from the repository root.
set -eu
CONFIG=${1:-configs/apps.list}

validate() {
    case "$1" in
        ''|*[!a-zA-Z0-9_-]*) printf 'Invalid app name: %s\n' "$1" >&2; exit 1 ;;
    esac
    if [ ! -f "cmd/$1/main.go" ]; then
        printf 'Unknown Go app: %s\n' "$1" >&2
        exit 1
    fi
}

APPS=
if [ -e "$CONFIG" ]; then
    # Format: one name per line; blank lines and # comments are allowed.
    while IFS= read -r line || [ -n "$line" ]; do
        line=${line%%#*}
        # Word splitting trims whitespace; glob expansion is disabled.
        set -f
        for app in $line; do
            validate "$app"
            case " $APPS " in *" $app "*) ;; *) APPS="$APPS $app" ;; esac
        done
    done < "$CONFIG"
else
    for dir in cmd/*; do
        [ -f "$dir/main.go" ] || continue
        app=${dir#cmd/}
        validate "$app"
        APPS="$APPS $app"
    done
fi

if [ -z "$APPS" ]; then
    printf 'No apps selected in %s\n' "$CONFIG" >&2
    exit 1
fi
for app in $APPS; do printf '%s\n' "$app"; done
