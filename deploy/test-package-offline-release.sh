#!/usr/bin/env bash
# Offline integration test for deploy/package-offline-release.sh.
# Replaces curl with a local fixture so CI never downloads LiveKit.

set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

mkdir -p "$TMP/bin" "$TMP/release"
for arch in amd64 arm64; do
  printf '#!/bin/sh\necho fake mqvi %s\n' "$arch" > "$TMP/release/mqvi-server-linux-$arch"
  chmod 0755 "$TMP/release/mqvi-server-linux-$arch"
done

cat > "$TMP/bin/curl" <<'CURL'
#!/usr/bin/env bash
set -Eeuo pipefail
output=""
while (($#)); do
  if [[ "$1" == "-o" ]]; then
    output="$2"
    shift 2
  else
    shift
  fi
done
[[ -n "$output" ]]
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
printf '#!/bin/sh\necho fake livekit\n' > "$fixture/livekit-server"
chmod 0755 "$fixture/livekit-server"
tar -C "$fixture" -czf "$output" livekit-server
CURL
chmod 0755 "$TMP/bin/curl"

PATH="$TMP/bin:$PATH" GITHUB_SHA=offline-package-test \
  bash "$ROOT/deploy/package-offline-release.sh" "$TMP/release"

for arch in amd64 arm64; do
  archive="oblivions-offline-linux-$arch.tar.gz"
  (
    cd "$TMP/release"
    sha256sum --check --strict "$archive.sha256"
  )
  mkdir "$TMP/unpack-$arch"
  tar -xzf "$TMP/release/$archive" -C "$TMP/unpack-$arch"
  (
    cd "$TMP/unpack-$arch"
    grep -Fxq "Target: linux/$arch" README.txt
    sha256sum --check --strict SHA256SUMS
    test -x mqvi-server
    test -x livekit-server
  )
done

echo "Offline release package test passed."
