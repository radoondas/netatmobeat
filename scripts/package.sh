#!/usr/bin/env bash
# Package netatmobeat binaries into distributable archives (tar.gz, zip, deb, rpm).
#
# Prerequisites:
#   - Run scripts/build.sh first (binaries must exist in dist/)
#   - nfpm must be installed for deb/rpm (go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.41.1)
#
# Usage:
#   ./scripts/package.sh [VERSION]
#
# Environment variables:
#   BEAT_VERSION  — override version
#   SKIP_DEB_RPM  — set to "true" to skip deb/rpm packaging (when nfpm not available)

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

if [ -z "$VERSION" ]; then
  echo "ERROR: Could not determine version."
  exit 1
fi

DIST_DIR="$PROJECT_ROOT/dist"
PACKAGED=0

echo "Packaging netatmobeat v${VERSION}"

# Arch name mapping: GOARCH -> display name for tar.gz/zip
arch_name() {
  case "$1" in
    amd64) echo "x86_64" ;;
    arm64) echo "aarch64" ;;
    *)     echo "$1" ;;
  esac
}

# Arch mapping for deb packages: GOARCH -> deb arch
deb_arch() {
  case "$1" in
    amd64) echo "amd64" ;;
    arm64) echo "arm64" ;;
    *)     echo "$1" ;;
  esac
}

# Arch mapping for rpm packages: GOARCH -> rpm arch
rpm_arch() {
  case "$1" in
    amd64) echo "x86_64" ;;
    arm64) echo "aarch64" ;;
    *)     echo "$1" ;;
  esac
}

# Files to include in tar.gz/zip packages
PAYLOAD_FILES=(
  "netatmobeat.yml"
  "netatmobeat.reference.yml"
  "netatmobeat.template.publicdata.json"
  "netatmobeat.template.stastiondata.json"
  "fields.yml"
  "LICENSE.txt"
  "NOTICE.txt"
  "README.md"
)

# Generate SHA512 checksum sidecar
sha512_checksum() {
  local file="$1"
  if command -v sha512sum &>/dev/null; then
    (cd "$(dirname "$file")" && sha512sum "$(basename "$file")") > "${file}.sha512"
  else
    # macOS: shasum -a 512
    (cd "$(dirname "$file")" && shasum -a 512 "$(basename "$file")") > "${file}.sha512"
  fi
}

# --- tar.gz packages (linux, darwin) ---
for platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  ARCH_NAME="$(arch_name "$GOARCH")"

  BUILD_DIR="$DIST_DIR/netatmobeat-${VERSION}-${GOOS}-${GOARCH}"
  if [ ! -d "$BUILD_DIR" ]; then
    echo "  SKIP ${GOOS}/${GOARCH} — build directory not found"
    continue
  fi

  ARCHIVE_NAME="netatmobeat-${VERSION}-${GOOS}-${ARCH_NAME}.tar.gz"
  STAGING="$DIST_DIR/_staging/netatmobeat-${VERSION}-${GOOS}-${ARCH_NAME}"
  rm -rf "$STAGING"
  mkdir -p "$STAGING"

  # Copy binary
  cp "$BUILD_DIR/netatmobeat" "$STAGING/"

  # Copy payload files
  for f in "${PAYLOAD_FILES[@]}"; do
    if [ -f "$PROJECT_ROOT/$f" ]; then
      cp "$PROJECT_ROOT/$f" "$STAGING/"
    fi
  done

  # Copy docs subdirectory
  if [ -d "$PROJECT_ROOT/docs" ] && [ -f "$PROJECT_ROOT/docs/RUNBOOK.md" ]; then
    mkdir -p "$STAGING/docs"
    cp "$PROJECT_ROOT/docs/RUNBOOK.md" "$STAGING/docs/"
  fi

  echo "  Creating ${ARCHIVE_NAME}..."
  tar -czf "$DIST_DIR/$ARCHIVE_NAME" -C "$DIST_DIR/_staging" "netatmobeat-${VERSION}-${GOOS}-${ARCH_NAME}"
  sha512_checksum "$DIST_DIR/$ARCHIVE_NAME"
  PACKAGED=$((PACKAGED + 1))

  rm -rf "$STAGING"
done

# --- zip packages (windows) ---
for platform in windows/amd64; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  ARCH_NAME="$(arch_name "$GOARCH")"

  BUILD_DIR="$DIST_DIR/netatmobeat-${VERSION}-${GOOS}-${GOARCH}"
  if [ ! -d "$BUILD_DIR" ]; then
    echo "  SKIP ${GOOS}/${GOARCH} — build directory not found"
    continue
  fi

  ARCHIVE_NAME="netatmobeat-${VERSION}-${GOOS}-${ARCH_NAME}.zip"
  STAGING="$DIST_DIR/_staging/netatmobeat-${VERSION}-${GOOS}-${ARCH_NAME}"
  rm -rf "$STAGING"
  mkdir -p "$STAGING"

  # Copy binary
  cp "$BUILD_DIR/netatmobeat.exe" "$STAGING/"

  # Copy payload files
  for f in "${PAYLOAD_FILES[@]}"; do
    if [ -f "$PROJECT_ROOT/$f" ]; then
      cp "$PROJECT_ROOT/$f" "$STAGING/"
    fi
  done

  if [ -d "$PROJECT_ROOT/docs" ] && [ -f "$PROJECT_ROOT/docs/RUNBOOK.md" ]; then
    mkdir -p "$STAGING/docs"
    cp "$PROJECT_ROOT/docs/RUNBOOK.md" "$STAGING/docs/"
  fi

  echo "  Creating ${ARCHIVE_NAME}..."
  (cd "$DIST_DIR/_staging" && zip -qr "$DIST_DIR/$ARCHIVE_NAME" "netatmobeat-${VERSION}-${GOOS}-${ARCH_NAME}")
  sha512_checksum "$DIST_DIR/$ARCHIVE_NAME"
  PACKAGED=$((PACKAGED + 1))

  rm -rf "$STAGING"
done

# --- deb/rpm packages via nfpm (linux only) ---
if [ "${SKIP_DEB_RPM:-}" = "true" ]; then
  echo "  Skipping deb/rpm (SKIP_DEB_RPM=true)"
elif ! command -v nfpm &>/dev/null; then
  echo "  Skipping deb/rpm (nfpm not found — install with: go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.41.1)"
else
  for GOARCH in amd64 arm64; do
    BUILD_DIR="$DIST_DIR/netatmobeat-${VERSION}-linux-${GOARCH}"
    if [ ! -d "$BUILD_DIR" ]; then
      echo "  SKIP deb/rpm ${GOARCH} — build directory not found"
      continue
    fi

    DEB_A="$(deb_arch "$GOARCH")"
    RPM_A="$(rpm_arch "$GOARCH")"
    DEB_NAME="netatmobeat-${VERSION}-${DEB_A}.deb"
    RPM_NAME="netatmobeat-${VERSION}-${RPM_A}.rpm"

    echo "  Creating ${DEB_NAME}..."
    ARCH="$DEB_A" VERSION="$VERSION" GOARCH="$GOARCH" \
      nfpm package -p deb -f "$PROJECT_ROOT/packaging/nfpm.yaml" -t "$DIST_DIR/$DEB_NAME"
    sha512_checksum "$DIST_DIR/$DEB_NAME"
    PACKAGED=$((PACKAGED + 1))

    echo "  Creating ${RPM_NAME}..."
    ARCH="$RPM_A" VERSION="$VERSION" GOARCH="$GOARCH" \
      nfpm package -p rpm -f "$PROJECT_ROOT/packaging/nfpm.yaml" -t "$DIST_DIR/$RPM_NAME"
    sha512_checksum "$DIST_DIR/$RPM_NAME"
    PACKAGED=$((PACKAGED + 1))
  done
fi

# Cleanup staging
rm -rf "$DIST_DIR/_staging"

if [ "$PACKAGED" -eq 0 ]; then
  echo "ERROR: No artifacts were packaged. Did you run scripts/build.sh first?"
  exit 1
fi

echo "Packaging complete. ${PACKAGED} artifact(s) in $DIST_DIR/"
ls -1 "$DIST_DIR/"*.{tar.gz,zip,deb,rpm,sha512} 2>/dev/null || true