#!/usr/bin/env bash
# Verify that all expected release artifacts exist in dist/ and have SHA512 checksums.
#
# Usage:
#   ./scripts/verify-assets.sh [VERSION]
#
# Environment variables:
#   BEAT_VERSION  — override version
#   SKIP_DEB_RPM  — set to "true" to skip deb/rpm verification

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Resolve version
if [ -n "${BEAT_VERSION:-}" ]; then
  VERSION="$BEAT_VERSION"
elif [ -n "${1:-}" ]; then
  VERSION="$1"
else
  VERSION="$(sed -n 's/^version *= *"\(.*\)"/\1/p' "$PROJECT_ROOT/.beat-version")"
fi

DIST_DIR="$PROJECT_ROOT/dist"
ERRORS=0

check_file() {
  local file="$1"
  if [ -f "$DIST_DIR/$file" ]; then
    echo "  OK  $file"
  else
    echo "  MISSING  $file"
    ERRORS=$((ERRORS + 1))
  fi
}

echo "Verifying release artifacts for netatmobeat v${VERSION}..."
echo

# tar.gz archives + checksums
echo "tar.gz archives:"
for pair in "linux-x86_64" "linux-aarch64" "darwin-x86_64" "darwin-aarch64"; do
  check_file "netatmobeat-${VERSION}-${pair}.tar.gz"
  check_file "netatmobeat-${VERSION}-${pair}.tar.gz.sha512"
done

echo
echo "zip archives:"
check_file "netatmobeat-${VERSION}-windows-x86_64.zip"
check_file "netatmobeat-${VERSION}-windows-x86_64.zip.sha512"

if [ "${SKIP_DEB_RPM:-}" != "true" ]; then
  echo
  echo "deb packages:"
  for arch in amd64 arm64; do
    check_file "netatmobeat-${VERSION}-${arch}.deb"
    check_file "netatmobeat-${VERSION}-${arch}.deb.sha512"
  done

  echo
  echo "rpm packages:"
  for arch in x86_64 aarch64; do
    check_file "netatmobeat-${VERSION}-${arch}.rpm"
    check_file "netatmobeat-${VERSION}-${arch}.rpm.sha512"
  done
fi

echo
if [ "$ERRORS" -gt 0 ]; then
  echo "FAIL: $ERRORS missing artifact(s)"
  exit 1
else
  echo "PASS: All expected artifacts present"
fi