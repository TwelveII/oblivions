#!/usr/bin/env bash
# Package already-built Oblivions binaries with matching LiveKit binaries for offline install.
# This runs during a tagged release while internet access is available.

set -Eeuo pipefail

RELEASE_DIR="${1:-release}"
LIVEKIT_VERSION="${LIVEKIT_VERSION:-v1.9.12}"
STAGE=""

cleanup() {
  if [[ -n "$STAGE" && -d "$STAGE" ]]; then
    rm -rf "$STAGE"
  fi
}
trap cleanup EXIT

for tool in curl tar install sha256sum; do
  command -v "$tool" >/dev/null 2>&1 || {
    echo "Missing required tool: $tool" >&2
    exit 1
  }
done

[[ -d "$RELEASE_DIR" ]] || {
  echo "Release directory does not exist: $RELEASE_DIR" >&2
  exit 1
}

for arch in amd64 arm64; do
  server_binary="$RELEASE_DIR/mqvi-server-linux-$arch"
  [[ -s "$server_binary" && -x "$server_binary" ]] || {
    echo "Missing executable server binary: $server_binary" >&2
    exit 1
  }

  STAGE="$(mktemp -d)"
  install -m 0755 "$server_binary" "$STAGE/mqvi-server"

  livekit_archive="$STAGE/livekit.tar.gz"
  livekit_url="https://github.com/livekit/livekit/releases/download/$LIVEKIT_VERSION/livekit_${LIVEKIT_VERSION#v}_linux_${arch}.tar.gz"
  echo "Downloading LiveKit $LIVEKIT_VERSION for linux/$arch..."
  curl -fsSL --retry 3 "$livekit_url" -o "$livekit_archive"
  tar -xzf "$livekit_archive" -C "$STAGE" livekit-server
  rm -f "$livekit_archive"
  chmod 0755 "$STAGE/livekit-server"

  cat > "$STAGE/README.txt" <<EOF
Oblivions offline runtime bundle
Source revision: ${GITHUB_SHA:-unknown}
Target: linux/$arch
LiveKit version: $LIVEKIT_VERSION

Contents:
- mqvi-server: Oblivions server with embedded frontend
- livekit-server: local LiveKit SFU

This archive does not include OS packages, Caddy, certificates, DNS,
firewall configuration, or coturn. Provision those domestically in advance.
EOF

  (
    cd "$STAGE"
    sha256sum README.txt mqvi-server livekit-server > SHA256SUMS
  )

  archive="$RELEASE_DIR/oblivions-offline-linux-$arch.tar.gz"
  tar -C "$STAGE" -czf "$archive" .
  (
    cd "$RELEASE_DIR"
    sha256sum "$(basename "$archive")" > "$(basename "$archive").sha256"
  )

  rm -rf "$STAGE"
  STAGE=""
  echo "Created $archive"
done
