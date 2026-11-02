#!/usr/bin/env bash
# Build a self-contained Oblivions application + LiveKit bundle without network access.
#
# Prepare the LiveKit binary separately before the outage, then run:
#   LIVEKIT_BINARY=/path/to/livekit-server deploy/prepare-offline-bundle.sh
#
# Requirements: cached npm dependencies (including SheetJS), cached Go modules,
# Node/npm, Go, tar, sha256sum. No downloads are performed by this script.

set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) GOARCH=amd64 ;;
  aarch64|arm64) GOARCH=arm64 ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

LIVEKIT_BINARY="${LIVEKIT_BINARY:-$ROOT/deploy/package/livekit-server}"
OUT_DIR="${OUT_DIR:-$ROOT/dist/offline}"
VERSION="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || printf 'local')"
STAGE="$(mktemp -d)"
STATIC_DIR="$ROOT/server/static/dist"
STATIC_BACKUP=""

cleanup() {
  if [[ -n "$STATIC_BACKUP" && -d "$STATIC_BACKUP" ]]; then
    rm -rf "$STATIC_DIR"
    mv "$STATIC_BACKUP" "$STATIC_DIR"
  elif [[ -d "$STATIC_DIR" ]]; then
    rm -rf "$STATIC_DIR"
  fi
  rm -rf "$STAGE"
}
trap cleanup EXIT

for tool in npm go tar sha256sum; do
  command -v "$tool" >/dev/null 2>&1 || { echo "Missing required tool: $tool" >&2; exit 1; }
done
[[ -s "$LIVEKIT_BINARY" && -x "$LIVEKIT_BINARY" ]] || {
  echo "Set LIVEKIT_BINARY to an executable, pre-downloaded LiveKit server binary." >&2
  exit 1
}

if [[ -d "$STATIC_DIR" ]]; then
  STATIC_BACKUP="$(mktemp -d)"
  rmdir "$STATIC_BACKUP"
  mv "$STATIC_DIR" "$STATIC_BACKUP"
fi

echo "[1/4] Installing frontend dependencies from the local npm cache only..."
(
  cd "$ROOT/client"
  npm ci --offline --no-audit --no-fund
  npm run build
)

echo "[2/4] Embedding the frontend and compiling with cached Go modules only..."
mkdir -p "$STATIC_DIR"
cp -a "$ROOT/client/dist/." "$STATIC_DIR/"
(
  cd "$ROOT/server"
  GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" \
    go build -trimpath -buildvcs=false -o "$STAGE/mqvi-server" .
)

echo "[3/4] Staging LiveKit and checksums..."
install -m 0755 "$LIVEKIT_BINARY" "$STAGE/livekit-server"
cat > "$STAGE/README.txt" <<EOF
Oblivions offline runtime bundle
Source revision: $VERSION
Target: linux/$GOARCH

Contents:
- mqvi-server: Go server with embedded frontend
- livekit-server: local LiveKit SFU
- SHA256SUMS: integrity hashes

This bundle does not contain OS packages, Caddy, certificates, DNS, firewall rules,
or coturn. Those must be provisioned on the domestic host before an outage.
EOF
(
  cd "$STAGE"
  sha256sum mqvi-server livekit-server > SHA256SUMS
)

echo "[4/4] Creating archive..."
mkdir -p "$OUT_DIR"
ARCHIVE="$OUT_DIR/oblivions-offline-linux-$GOARCH-$VERSION.tar.gz"
tar -C "$STAGE" -czf "$ARCHIVE" .
sha256sum "$ARCHIVE" > "$ARCHIVE.sha256"
echo "Created: $ARCHIVE"
echo "Archive checksum: $ARCHIVE.sha256"
