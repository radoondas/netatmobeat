#!/usr/bin/env bash
# Cross-compile netatmobeat for all supported platforms.
#
# Usage:
#   ./scripts/build.sh [VERSION]
#
# Environment variables:
#   BEAT_VERSION  — override version (takes priority over argument and .beat-version)
#   PLATFORMS     — space-separated list of os/arch pairs (default: all)
#
# Output:
#   dist/netatmobeat-{VERSION}-{os}-{arch}/netatmobeat[.exe]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Resolve version: BEAT_VERSION env > $1 argument > .beat-version file
if [ -n "${BEAT_VERSION:-}" ]; then
  VERSION="$BEAT_VERSION"
elif [ -n "${1:-}" ]; then
  VERSION="$1"
else
  VERSION="$(sed -n 's/^version *= *"\(.*\)"/\1/p' "$PROJECT_ROOT/.beat-version")"
fi

if [ -z "$VERSION" ]; then
  echo "ERROR: Could not determine version. Set BEAT_VERSION, pass as argument, or check .beat-version"
  exit 1
fi

echo "Building netatmobeat v${VERSION}"

# Default platform matrix
DEFAULT_PLATFORMS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64"
PLATFORMS="${PLATFORMS:-$DEFAULT_PLATFORMS}"

# Build metadata
COMMIT="$(git -C "$PROJECT_ROOT" rev-parse --short HEAD 2>/dev/null || echo "unknown")"
BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

LDFLAGS="-s -w"
LDFLAGS="$LDFLAGS -X github.com/elastic/beats/v7/libbeat/version.buildTime=$BUILD_TIME"
LDFLAGS="$LDFLAGS -X github.com/elastic/beats/v7/libbeat/version.commit=$COMMIT"

DIST_DIR="$PROJECT_ROOT/dist"
mkdir -p "$DIST_DIR"

for platform in $PLATFORMS; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"

  BINARY="netatmobeat"
  if [ "$GOOS" = "windows" ]; then
    BINARY="netatmobeat.exe"
  fi

  OUT_DIR="$DIST_DIR/netatmobeat-${VERSION}-${GOOS}-${GOARCH}"
  mkdir -p "$OUT_DIR"

  echo "  Building ${GOOS}/${GOARCH}..."
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -ldflags "$LDFLAGS" -o "$OUT_DIR/$BINARY" "$PROJECT_ROOT"
done

echo "Build complete. Binaries in $DIST_DIR/"