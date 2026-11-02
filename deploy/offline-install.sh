#!/usr/bin/env bash
# Install Oblivions from a prebuilt offline bundle on a freshly provisioned Linux host.
# This script makes no network requests and never installs OS packages.

set -Eeuo pipefail

BUNDLE=""
APP_DIR="/opt/mqvi"
PUBLIC_IP=""
DOMAIN=""
LIVEKIT_URL=""
PORT="9090"
while (($#)); do
  case "$1" in
    --bundle) BUNDLE="${2:-}"; shift 2 ;;
    --app-dir) APP_DIR="${2:-}"; shift 2 ;;
    --public-ip) PUBLIC_IP="${2:-}"; shift 2 ;;
    --domain) DOMAIN="${2:-}"; shift 2 ;;
    --livekit-url) LIVEKIT_URL="${2:-}"; shift 2 ;;
    --port) PORT="${2:-}"; shift 2 ;;
    -h|--help)
      echo "Usage: sudo deploy/offline-install.sh --bundle FILE --public-ip IPV4 [--domain HOST] --livekit-url wss://livekit.example.ir [--port 9090]"
      echo "Requires preinstalled systemd, curl, openssl, tar, sha256sum and a TLS reverse proxy."
      exit 0 ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

die() { echo "ERROR: $*" >&2; exit 1; }
[[ "$(id -u)" -eq 0 ]] || die "Run as root."
[[ -n "$BUNDLE" && -s "$BUNDLE" ]] || die "--bundle must point to a non-empty bundle archive."
[[ "$LIVEKIT_URL" =~ ^wss://[A-Za-z0-9.-]+$ ]] || die "--livekit-url must be a WSS hostname without a path or port."
[[ "$PUBLIC_IP" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] || die "--public-ip must be the server's public IPv4 address."
[[ "$PORT" =~ ^[0-9]+$ ]] && ((PORT > 0 && PORT < 65536)) || die "Invalid port."
for tool in systemctl useradd install tar sha256sum openssl curl; do
  command -v "$tool" >/dev/null 2>&1 || die "Missing preinstalled requirement: $tool"
done
[[ ! -e "$APP_DIR/mqvi-server" ]] || die "$APP_DIR/mqvi-server already exists; use the separate domestic deployment/upgrade helper."
systemctl cat mqvi-server.service >/dev/null 2>&1 && die "mqvi-server.service already exists; refusing to replace an existing installation."

TMP="$(mktemp -d)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT
tar -xzf "$BUNDLE" -C "$TMP"
[[ -s "$TMP/mqvi-server" && -x "$TMP/mqvi-server" ]] || die "Bundle is missing mqvi-server."
[[ -s "$TMP/livekit-server" && -x "$TMP/livekit-server" ]] || die "Bundle is missing livekit-server."
[[ -s "$TMP/SHA256SUMS" && -s "$TMP/README.txt" ]] || die "Bundle is missing SHA256SUMS or README.txt."
(cd "$TMP" && sha256sum --check --strict SHA256SUMS) || die "Bundle integrity check failed."
case "$(uname -m)" in
  x86_64|amd64) HOST_ARCH=amd64 ;;
  aarch64|arm64) HOST_ARCH=arm64 ;;
  *) die "Unsupported host architecture: $(uname -m)" ;;
esac
grep -Fxq "Target: linux/$HOST_ARCH" "$TMP/README.txt" || die "Bundle architecture does not match host $HOST_ARCH."

if ! id mqvi >/dev/null 2>&1; then
  useradd --system --home-dir "$APP_DIR" --shell /usr/sbin/nologin mqvi
fi
install -d -o mqvi -g mqvi -m 0750 "$APP_DIR" "$APP_DIR/data" "$APP_DIR/data/uploads"
install -o root -g root -m 0755 "$TMP/mqvi-server" "$APP_DIR/mqvi-server"
install -o root -g root -m 0755 "$TMP/livekit-server" "$APP_DIR/livekit-server"

if [[ ! -s "$APP_DIR/.env" ]]; then
  JWT_SECRET="$(openssl rand -hex 32)"
  ENCRYPTION_KEY="$(openssl rand -hex 32)"
  SIGNED_URL_SECRET="$(openssl rand -base64 32)"
  LK_API_KEY="APIkey$(openssl rand -hex 6)"
  LK_API_SECRET="$(openssl rand -hex 32)"
  APP_URL=""
  if [[ -n "$DOMAIN" ]]; then APP_URL="https://$DOMAIN"; fi
  cat > "$APP_DIR/.env" <<EOF
SERVER_HOST=127.0.0.1
SERVER_PORT=$PORT
MQVI_READINESS_ADDR=127.0.0.1:9091
DATABASE_PATH=$APP_DIR/data/mqvi.db
UPLOAD_DIR=$APP_DIR/data/uploads
JWT_SECRET=$JWT_SECRET
JWT_ACCESS_EXPIRY_MINUTES=15
JWT_REFRESH_EXPIRY_DAYS=90
JWT_FILE_EXPIRY_HOURS=24
ENCRYPTION_KEY=$ENCRYPTION_KEY
MQVI_SIGNED_URL_SECRET=$SIGNED_URL_SECRET
MQVI_ANTIVIRUS_ENABLED=false
MQVI_PASSWORD_BREACH_CHECK=false
MQVI_GEOIP_AUTO_UPDATE=false
MQVI_LINK_PREVIEWS_ENABLED=false
STUN_URLS=
TURN_URLS=
LIVEKIT_URL=$LIVEKIT_URL
LIVEKIT_API_KEY=$LK_API_KEY
LIVEKIT_API_SECRET=$LK_API_SECRET
APP_URL=$APP_URL
CORS_ORIGINS=$APP_URL
EOF
fi

# Keep the backend SDK on the same public WSS endpoint that browser clients use.
if grep -q "^LIVEKIT_URL=" "$APP_DIR/.env"; then
  sed -i "s|^LIVEKIT_URL=.*|LIVEKIT_URL=$LIVEKIT_URL|" "$APP_DIR/.env"
else
  printf "LIVEKIT_URL=%s\\n" "$LIVEKIT_URL" >> "$APP_DIR/.env"
fi

if [[ ! -s "$APP_DIR/livekit.yaml" ]]; then
  # Explicit node_ip disables LiveKit's external-IP/STUN discovery.
  LK_KEY="$(grep '^LIVEKIT_API_KEY=' "$APP_DIR/.env" | tail -n1 | cut -d= -f2-)"
  LK_SECRET="$(grep '^LIVEKIT_API_SECRET=' "$APP_DIR/.env" | tail -n1 | cut -d= -f2-)"
  [[ -n "$LK_KEY" && -n "$LK_SECRET" ]] || die "Existing .env must define LIVEKIT_API_KEY and LIVEKIT_API_SECRET."
  cat > "$APP_DIR/livekit.yaml" <<EOF
port: 7880
rtc:
  tcp_port: 7881
  port_range_start: 50000
  port_range_end: 50200
  use_external_ip: false
  node_ip: $PUBLIC_IP
keys:
  $LK_KEY: $LK_SECRET
logging:
  level: info
EOF
fi

chown root:mqvi "$APP_DIR/.env" "$APP_DIR/livekit.yaml"
chmod 0640 "$APP_DIR/.env" "$APP_DIR/livekit.yaml"
chown mqvi:mqvi "$APP_DIR/data" "$APP_DIR/data/uploads"

cat > /etc/systemd/system/mqvi-livekit.service <<EOF
[Unit]
Description=Oblivions local LiveKit SFU
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=mqvi
Group=mqvi
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/livekit-server --config $APP_DIR/livekit.yaml
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=$APP_DIR

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/mqvi-server.service <<EOF
[Unit]
Description=Oblivions application server
After=network-online.target mqvi-livekit.service
Wants=network-online.target
Requires=mqvi-livekit.service

[Service]
Type=simple
User=mqvi
Group=mqvi
WorkingDirectory=$APP_DIR
EnvironmentFile=$APP_DIR/.env
ExecStart=$APP_DIR/mqvi-server
Restart=on-failure
RestartSec=5
KillSignal=SIGTERM
TimeoutStopSec=20
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=$APP_DIR/data

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now mqvi-livekit.service mqvi-server.service
READY=0
for attempt in $(seq 1 30); do
  if curl --fail --silent --max-time 3 http://127.0.0.1:9091/health/ready >/dev/null; then
    READY=1
    break
  fi
  sleep 2
done
if [[ "$READY" -ne 1 ]]; then
  systemctl status mqvi-server.service --no-pager >&2 || true
  journalctl -u mqvi-server.service -n 50 --no-pager >&2 || true
  die "Server did not become ready. Services are left installed for diagnosis."
fi

echo "Oblivions is installed and ready."
echo "App directory: $APP_DIR"
echo "Local app: http://127.0.0.1:$PORT"
echo "Configure TLS proxy: $DOMAIN -> 127.0.0.1:$PORT and the LiveKit WSS hostname -> 127.0.0.1:7880."
echo "Open TCP 80/443 (proxy), TCP 7880/7881, and UDP 50000-50200 in host/provider firewalls."
echo "Add the LiveKit WSS hostname to /etc/hosts as 127.0.0.1 on this server so backend SDK calls use the local TLS proxy."
echo "No Caddy, certificate, coturn, or OS packages were installed."
echo "For restrictive client networks, provision domestic TURN/coturn separately and set TURN_URLS/TURN_SECRET."
