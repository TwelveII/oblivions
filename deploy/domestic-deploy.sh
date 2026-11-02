#!/usr/bin/env bash
# Deploy a prebuilt Oblivions server binary to a domestic host without GitHub Actions.
#
# This script performs no downloads and does not build the application. Prepare the
# Linux binary before an international outage, then run this from a machine that can
# reach the domestic server over SSH.
#
# Usage:
#   deploy/domestic-deploy.sh --host 203.0.113.10 --artifact deploy/package/mqvi-server
#
# Optional:
#   --user root
#   --port 22
#   --key ~/.ssh/id_ed25519
#   --app-dir /opt/mqvi
#   --service mqvi-server.service
#   --ready-addr 127.0.0.1:9091
#
# The script preserves .env, the database, uploads, LiveKit and all other server data.
# It backs up the previous binary and automatically restores it if restart/readiness fails.

set -Eeuo pipefail

SSH_HOST=""
SSH_USER="root"
SSH_PORT="22"
SSH_KEY=""
ARTIFACT=""
APP_DIR="/opt/mqvi"
SERVICE="mqvi-server.service"
READY_ADDR="127.0.0.1:9091"

usage() {
  sed -n '2,24p' "$0" | sed 's/^# \?//'
}

die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

while (($#)); do
  case "$1" in
    --host) SSH_HOST="${2:-}"; shift 2 ;;
    --user) SSH_USER="${2:-}"; shift 2 ;;
    --port) SSH_PORT="${2:-}"; shift 2 ;;
    --key) SSH_KEY="${2:-}"; shift 2 ;;
    --artifact) ARTIFACT="${2:-}"; shift 2 ;;
    --app-dir) APP_DIR="${2:-}"; shift 2 ;;
    --service) SERVICE="${2:-}"; shift 2 ;;
    --ready-addr) READY_ADDR="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) die "Unknown argument: $1 (use --help)" ;;
  esac
done

[[ -n "$SSH_HOST" ]] || die "--host is required"
[[ -n "$ARTIFACT" ]] || die "--artifact is required"
[[ "$SSH_HOST" =~ ^[A-Za-z0-9._-]+$ ]] || die "Invalid host name or IPv4 address"
[[ "$SSH_USER" == "root" ]] || die "Use --user root; the installed service and binary are root-owned"
[[ "$SSH_PORT" =~ ^[0-9]+$ ]] && ((SSH_PORT > 0 && SSH_PORT < 65536)) || die "Invalid SSH port"
[[ -f "$ARTIFACT" && -s "$ARTIFACT" ]] || die "Artifact does not exist or is empty: $ARTIFACT"
if [[ -n "$SSH_KEY" ]]; then
  [[ -f "$SSH_KEY" ]] || die "SSH key not found: $SSH_KEY"
fi

REMOTE="${SSH_USER}@${SSH_HOST}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
REMOTE_NEW="/tmp/oblivions-mqvi-server-${STAMP}.new"
BACKUP_NAME="mqvi-server-${STAMP}"
SSH_OPTS=(-p "$SSH_PORT" -o BatchMode=yes -o StrictHostKeyChecking=yes)
SCP_OPTS=(-P "$SSH_PORT" -o BatchMode=yes -o StrictHostKeyChecking=yes)
if [[ -n "$SSH_KEY" ]]; then
  SSH_OPTS+=(-i "$SSH_KEY")
  SCP_OPTS+=(-i "$SSH_KEY")
fi

echo "[1/4] Checking SSH access and service layout..."
ssh "${SSH_OPTS[@]}" "$REMOTE" "test -d $(printf '%q' "$APP_DIR") && systemctl cat $(printf '%q' "$SERVICE") >/dev/null"

echo "[2/4] Uploading prebuilt binary..."
scp "${SCP_OPTS[@]}" "$ARTIFACT" "$REMOTE:$REMOTE_NEW"

echo "[3/4] Installing with backup and automatic rollback..."
printf -v REMOTE_COMMAND 'APP_DIR=%q SERVICE=%q READY_ADDR=%q NEW_BINARY=%q BACKUP_NAME=%q bash -s' \
  "$APP_DIR" "$SERVICE" "$READY_ADDR" "$REMOTE_NEW" "$BACKUP_NAME"

ssh "${SSH_OPTS[@]}" "$REMOTE" "$REMOTE_COMMAND" <<'REMOTE'
set -Eeuo pipefail

CURRENT="$APP_DIR/mqvi-server"
BACKUPS="$APP_DIR/deploy-backups"
BACKUP="$BACKUPS/$BACKUP_NAME"

test -s "$NEW_BINARY"
test -x "$CURRENT"
test -f "$APP_DIR/.env"
install -d -o root -g root -m 700 "$BACKUPS"
cp -a "$CURRENT" "$BACKUP"

DB_BACKUP_TMP="$BACKUP.database.tmp"
DB_REL="$(sed -n 's/^DATABASE_PATH=//p' "$APP_DIR/.env" | tail -n 1 | sed 's/^"//; s/"$//; s/\r$//')"
DB_REL="${DB_REL:-./data/mqvi.db}"
if [[ "$DB_REL" = /* ]]; then DB_FILE="$DB_REL"; else DB_FILE="$APP_DIR/$DB_REL"; fi
test -s "$DB_FILE" || { echo "Database not found at $DB_FILE; refusing deployment" >&2; exit 1; }

systemctl stop "$SERVICE"

rollback() {
  echo "Deployment failed; restoring previous binary and SQLite backup from $BACKUP" >&2
  systemctl stop "$SERVICE" || true
  cp -a "$BACKUP" "$APP_DIR/mqvi-server.rollback"
  mv -f "$APP_DIR/mqvi-server.rollback" "$CURRENT"
  if [ -s "$BACKUP.database/database" ]; then
    rm -f "$DB_FILE-wal" "$DB_FILE-shm"
    cp -a "$BACKUP.database/database" "$DB_FILE"
    for suffix in -wal -shm; do
      if [ -f "$BACKUP.database/database$suffix" ]; then
        cp -a "$BACKUP.database/database$suffix" "$DB_FILE$suffix"
      fi
    done
  fi
  systemctl start "$SERVICE" || true
  rm -f "$NEW_BINARY" "$DB_BACKUP_TMP"
}
trap 'status=$?; if [ "$status" -ne 0 ]; then rollback; fi; exit "$status"' EXIT

# The application is stopped, so the database and any WAL/SHM sidecars form a consistent backup.
install -d -o root -g root -m 700 "$DB_BACKUP_TMP"
cp -a "$DB_FILE" "$DB_BACKUP_TMP/database"
for suffix in -wal -shm; do
  if [ -f "$DB_FILE$suffix" ]; then cp -a "$DB_FILE$suffix" "$DB_BACKUP_TMP/database$suffix"; fi
done
mv -f "$DB_BACKUP_TMP" "$BACKUP.database"

install -o root -g root -m 755 "$NEW_BINARY" "$APP_DIR/mqvi-server.installing"
mv -f "$APP_DIR/mqvi-server.installing" "$CURRENT"
rm -f "$NEW_BINARY"
systemctl start "$SERVICE"

READY=0
for attempt in $(seq 1 30); do
  if curl --fail --silent --show-error --max-time 3 \
    "http://$READY_ADDR/health/ready" >/dev/null 2>&1; then
    READY=1
    break
  fi
  sleep 2
done

if [ "$READY" -ne 1 ]; then
  echo "Readiness check failed at http://$READY_ADDR/health/ready" >&2
  exit 1
fi

trap - EXIT
echo "Deployment succeeded. Previous binary backup: $BACKUP"
REMOTE

echo "[4/4] Domestic deployment completed."
