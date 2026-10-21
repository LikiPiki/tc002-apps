#!/bin/sh
# Build standalone clockctl archives; publishing is handled by the release workflow.
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
VERSION=${VERSION:-dev}
COMMIT=${COMMIT:-$(git -C "$ROOT" rev-parse HEAD)}
case "$VERSION" in
    ''|*[!A-Za-z0-9._-]*)
        printf '%s\n' 'VERSION must contain only letters, digits, dots, underscores or hyphens' >&2
        exit 2
        ;;
esac

OUTPUT="$ROOT/dist/release/$VERSION"
mkdir -p "$OUTPUT"
BUILD_DIR=$(mktemp -d)
trap 'rm -rf "$BUILD_DIR"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
    OS=${target%/*}
    ARCH=${target#*/}
    BINARY=clockctl
    if [ "$OS" = windows ]; then
        BINARY=clockctl.exe
    fi
    CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" go -C "$ROOT" build -trimpath \
        -ldflags="-s -w -X github.com/LikiPiki/tc002-apps/internal/buildinfo.Version=$VERSION -X github.com/LikiPiki/tc002-apps/internal/buildinfo.Commit=$COMMIT" \
        -o "$BUILD_DIR/$BINARY" ./apps/tools/cmd/clockctl
    tar -czf "$OUTPUT/clockctl-$VERSION-$OS-$ARCH.tar.gz" \
        -C "$BUILD_DIR" "$BINARY" -C "$ROOT" LICENSE
done

# Use relative asset names so downloaded checksums can be verified in any directory.
(
    cd "$OUTPUT"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum ./*.tar.gz >SHA256SUMS
    else
        shasum -a 256 ./*.tar.gz >SHA256SUMS
    fi
)
printf 'Release assets: %s\n' "$OUTPUT"
