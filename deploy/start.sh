#!/bin/bash
# mqvi — Start both LiveKit SFU and mqvi backend
#
# Usage: chmod +x start.sh mqvi-server && ./start.sh
#
# Ctrl+C gracefully stops both processes.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# ─── Check .env ───
if [ ! -f .env ]; then
    echo "ERROR: .env file not found!"
    echo "Copy the example: cp .env.example .env"
    echo "Then set JWT_SECRET and LIVEKIT_API_SECRET."
    exit 1
fi

# ─── LiveKit: only run locally if livekit.yaml exists (otherwise a remote SFU is assumed) ───
RUN_LIVEKIT=false
if [ -f livekit.yaml ]; then
    RUN_LIVEKIT=true
fi

# ─── Check / download LiveKit binary (only when we'll actually run it) ───
if [ "$RUN_LIVEKIT" = true ] && [ ! -f ./livekit-server ]; then
    echo "LiveKit server not found. Downloading..."
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64) LK_ARCH="amd64" ;;
        aarch64|arm64) LK_ARCH="arm64" ;;
        *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    LK_VERSION="v1.9.12"
    LK_URL="https://github.com/livekit/livekit/releases/download/${LK_VERSION}/livekit_${LK_VERSION#v}_linux_${LK_ARCH}.tar.gz"
    echo "Downloading: $LK_URL"
    curl -fsSL "$LK_URL" | tar xz livekit-server
    chmod +x livekit-server
    echo "LiveKit server downloaded."
fi

# ─── Create data directories ───
mkdir -p data/uploads

env_value() {
    # $1 = key, $2 = default
    local key="$1"
    local default="$2"
    local line value
    line="$(grep -E "^${key}=" .env 2>/dev/null | tail -n 1 || true)"
    if [ -z "$line" ]; then
        printf '%s' "$default"
        return
    fi
    value="${line#*=}"
    value="${value%\"}"
    value="${value#\"}"
    printf '%s' "$value"
}

wait_for_clamav() {
    # $1 = addr (unix:/path, /path, or host:port), $2 = attempts
    local addr="$1" attempts="$2" i path host port
    case "$addr" in
        unix:*) path="${addr#unix:}" ;;
        /*)     path="$addr" ;;
    esac
    for i in $(seq 1 "$attempts"); do
        if [ -n "$path" ]; then
            [ -S "$path" ] && return 0
        else
            host="${addr%:*}"; port="${addr##*:}"
            if (echo >"/dev/tcp/${host}/${port}") >/dev/null 2>&1; then
                return 0
            fi
        fi
        sleep 1
    done
    return 1
}

install_clamav_if_missing() {
    # Idempotent — fast no-op when already installed. Uses Debian defaults
    # (Unix socket at /run/clamav/clamd.ctl); no TCP/socket config edits.
    if command -v clamd >/dev/null 2>&1 || command -v clamdscan >/dev/null 2>&1; then
        return 0
    fi
    if [ "$(id -u)" -ne 0 ]; then
        echo "[start] ClamAV not installed and not running as root; skipping install."
        return 1
    fi
    echo "[start] ClamAV not installed. Installing clamav-daemon + freshclam..."
    export DEBIAN_FRONTEND=noninteractive
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update -qq >/dev/null 2>&1 || true
        apt-get install -y clamav-daemon clamav-freshclam >/dev/null 2>&1 || {
            echo "[start] WARNING: apt install of clamav failed; uploads will follow unavailable policy."
            return 1
        }
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y clamav clamd clamav-update >/dev/null 2>&1 || return 1
    elif command -v yum >/dev/null 2>&1; then
        yum install -y clamav clamd clamav-update >/dev/null 2>&1 || return 1
    else
        echo "[start] WARNING: no supported package manager; install clamav manually."
        return 1
    fi
    systemctl enable --now clamav-freshclam >/dev/null 2>&1 || true
    systemctl enable --now clamav-daemon >/dev/null 2>&1 || true
    echo "[start] ClamAV installed. Signature DB may take ~2 minutes to populate on first run."
    return 0
}

ensure_clamav_ready() {
    local enabled addr
    enabled="$(env_value MQVI_ANTIVIRUS_ENABLED true)"
    case "$(printf '%s' "$enabled" | tr '[:upper:]' '[:lower:]')" in
        false|0|no|off)
            echo "[start] Antivirus scanning disabled."
            return
            ;;
    esac

    addr="$(env_value MQVI_CLAMAV_ADDR unix:/run/clamav/clamd.ctl)"

    install_clamav_if_missing || true
    if command -v systemctl >/dev/null 2>&1; then
        systemctl start clamav-daemon >/dev/null 2>&1 || true
    fi

    echo "[start] Waiting for ClamAV at ${addr}..."
    if wait_for_clamav "$addr" 60; then
        echo "[start] ClamAV is reachable."
    else
        echo "[start] WARNING: ClamAV is not reachable at ${addr}; uploads will follow antivirus unavailable policy."
    fi
}

sync_caddy_upload_size() {
    # Sync Caddy's request_body max_size with UPLOAD_MAX_SIZE in .env.
    # Only touches Caddyfiles marked "# managed by mqvi" — user-owned configs
    # (mode=existing) are left alone.
    local caddyfile="/etc/caddy/Caddyfile"
    [ -f "$caddyfile" ] || return 0
    grep -q "managed by mqvi" "$caddyfile" || return 0
    [ "$(id -u)" -eq 0 ] || return 0

    local size_bytes
    size_bytes="$(env_value UPLOAD_MAX_SIZE 104857600)"
    local current
    current="$(grep -oE 'max_size[[:space:]]+[^[:space:]}]+' "$caddyfile" | awk '{print $2}' | head -n 1)"
    if [ "$current" = "$size_bytes" ]; then
        return 0
    fi
    sed -i -E "s/(max_size[[:space:]]+)[^[:space:]}]+/\1${size_bytes}/" "$caddyfile"
    systemctl reload caddy >/dev/null 2>&1 || true
    echo "[start] Caddy max_size synced to ${size_bytes} bytes."
}

echo "========================================="
echo "  Starting mqvi server..."
echo "========================================="
echo ""

# ─── Start LiveKit in the background (skipped when no livekit.yaml — remote SFU mode) ───
LIVEKIT_PID=""
if [ "$RUN_LIVEKIT" = true ]; then
    echo "[start] Starting LiveKit SFU (port 7880)..."
    ./livekit-server --config livekit.yaml &
    LIVEKIT_PID=$!
else
    echo "[start] No livekit.yaml found — skipping local LiveKit (assuming remote SFU)."
fi

# ─── Cleanup trap — stop both on Ctrl+C or SIGTERM ───
cleanup() {
    echo ""
    echo "[start] Stopping servers..."
    kill $LIVEKIT_PID 2>/dev/null || true
    kill $MQVI_PID 2>/dev/null || true
    wait $LIVEKIT_PID 2>/dev/null || true
    wait $MQVI_PID 2>/dev/null || true
    echo "[start] Clean shutdown complete."
    exit 0
}
trap cleanup SIGINT SIGTERM

# Wait briefly for LiveKit to start
sleep 1

ensure_clamav_ready
sync_caddy_upload_size

# ─── Start mqvi backend ───
echo "[start] Starting mqvi backend (port 9090)..."
./mqvi-server &
MQVI_PID=$!

echo ""
echo "========================================="
echo "  mqvi is running!"
echo "  Web UI:  http://$(hostname -I | awk '{print $1}'):9090"
if [ "$RUN_LIVEKIT" = true ]; then
    echo "  LiveKit: ws://localhost:7880"
fi
echo "  Stop with: Ctrl+C"
echo "========================================="
echo ""

# Wait for either process — if one dies, stop both
if [ -n "$LIVEKIT_PID" ]; then
    wait -n $LIVEKIT_PID $MQVI_PID 2>/dev/null || true
else
    wait $MQVI_PID 2>/dev/null || true
fi
cleanup
