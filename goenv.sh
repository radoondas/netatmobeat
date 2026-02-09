#!/usr/bin/env bash
# Project-local Go environment wrapper.
# Usage:
#   ./goenv.sh go version
#   ./goenv.sh go build github.com/radoondas/netatmobeat
#   ./goenv.sh go test ./beater/...
#   ./goenv.sh make

MIN_GO_VERSION="1.24"
LOCAL_GOROOT="/Users/rado/opt/go-1.24.13"

# Check if a Go installation meets the minimum version requirement.
# Returns 0 (true) if the version is sufficient, 1 otherwise.
check_go_version() {
  local go_bin="$1/bin/go"
  [ -x "$go_bin" ] || return 1
  local ver
  ver=$("$go_bin" version 2>/dev/null | grep -oE 'go[0-9]+\.[0-9]+' | head -1 | sed 's/go//')
  [ -z "$ver" ] && return 1
  local major minor
  major=$(echo "$ver" | cut -d. -f1)
  minor=$(echo "$ver" | cut -d. -f2)
  local min_major min_minor
  min_major=$(echo "$MIN_GO_VERSION" | cut -d. -f1)
  min_minor=$(echo "$MIN_GO_VERSION" | cut -d. -f2)
  [ "$major" -gt "$min_major" ] && return 0
  [ "$major" -eq "$min_major" ] && [ "$minor" -ge "$min_minor" ] && return 0
  return 1
}

# Priority: GOROOT env var (if sufficient) > well-known local path > go in PATH
if [ -n "$GOROOT" ] && check_go_version "$GOROOT"; then
  # GOROOT already set and meets minimum version — use it
  :
elif [ -d "$LOCAL_GOROOT" ] && check_go_version "$LOCAL_GOROOT"; then
  export GOROOT="$LOCAL_GOROOT"
elif command -v go &>/dev/null && check_go_version "$(go env GOROOT)"; then
  export GOROOT="$(go env GOROOT)"
else
  echo "ERROR: Go >= ${MIN_GO_VERSION} not found."
  echo "  - Set GOROOT to a Go >= ${MIN_GO_VERSION} installation, or"
  echo "  - Install Go >= ${MIN_GO_VERSION} at ${LOCAL_GOROOT}, or"
  echo "  - Ensure 'go' >= ${MIN_GO_VERSION} is in your PATH"
  exit 1
fi

export PATH="$GOROOT/bin:$PATH"

exec "$@"