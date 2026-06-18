#!/usr/bin/env bash
#
# HiGoOS deployment script
# -------------------------
# Cross-compiles the Go backend (higo-api / higo-worker / higoctl), builds the
# web-pc frontend, ships both to the target NAS host, installs runtime deps,
# and restarts the systemd services.
#
# Usage:
#   SSH_PASS=secret ./deploy/deploy.sh [host] [ssh_user]
#
# Environment overrides:
#   HOST         target host          (default: 10.211.55.3, or $1)
#   SSH_USER     ssh login user       (default: hiveton,     or $2)
#   SSH_PASS     ssh/sudo password    (required; uses sshpass)
#   TARGET_ARCH  go GOARCH for target (default: arm64)
#   SKIP_DEPS    set to 1 to skip apt package install
#
# Requires on the build machine: go (>= go.mod version), node/npm, rsync, sshpass.

set -euo pipefail

# --- config -----------------------------------------------------------------
HOST="${HOST:-${1:-10.211.55.3}}"
SSH_USER="${SSH_USER:-${2:-hiveton}}"
TARGET_ARCH="${TARGET_ARCH:-arm64}"
SKIP_DEPS="${SKIP_DEPS:-0}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER_DIR="$REPO_ROOT/server-go"
WEB_DIR="$REPO_ROOT/web-pc"
DEPLOY_DIR="$REPO_ROOT/deploy"
BUILD_DIR="$(mktemp -d /tmp/higo-build.XXXXXX)"

REMOTE_STAGE="/home/$SSH_USER/higo-deploy"
INSTALL_BIN="/opt/higoos/bin"
INSTALL_WEB="/opt/higoos/web"
HEALTH_URL="http://127.0.0.1:8080/healthz"

RUNTIME_PKGS=(ca-certificates curl smartmontools util-linux e2fsprogs \
  btrfs-progs lm-sensors rsync ffmpeg aria2 docker.io docker-compose-v2)

if [[ -z "${SSH_PASS:-}" ]]; then
  echo "ERROR: SSH_PASS env var is required (ssh/sudo password)." >&2
  exit 1
fi

command -v sshpass >/dev/null || { echo "ERROR: sshpass not installed (brew install hudochenkov/sshpass/sshpass)"; exit 1; }

export SSHPASS="$SSH_PASS"
SSH_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
  -o ConnectTimeout=10 -o LogLevel=ERROR)

rsh() { sshpass -e ssh "${SSH_OPTS[@]}" "$SSH_USER@$HOST" "$@"; }
# run a command on the remote as root via sudo -S (password on stdin)
rsudo() { sshpass -e ssh "${SSH_OPTS[@]}" "$SSH_USER@$HOST" "echo '$SSH_PASS' | sudo -S -p '' bash -c \"$1\""; }

log() { printf '\033[1;36m==>\033[0m %s\n' "$*"; }

cleanup() { rm -rf "$BUILD_DIR"; }
trap cleanup EXIT

# --- 1. build backend -------------------------------------------------------
log "Cross-compiling Go backend for linux/$TARGET_ARCH"
mkdir -p "$BUILD_DIR/bin"
( cd "$SERVER_DIR"
  for cmd in higo-api higo-worker higoctl; do
    CGO_ENABLED=0 GOOS=linux GOARCH="$TARGET_ARCH" \
      go build -trimpath -ldflags="-s -w" -o "$BUILD_DIR/bin/$cmd" "./cmd/$cmd"
  done )

# --- 2. build frontend ------------------------------------------------------
log "Building web-pc frontend"
( cd "$WEB_DIR"
  [[ -d node_modules ]] || npm ci
  npm run build )

# --- 3. ship artifacts to remote staging ------------------------------------
log "Syncing artifacts to $SSH_USER@$HOST:$REMOTE_STAGE"
rsh "rm -rf $REMOTE_STAGE && mkdir -p $REMOTE_STAGE/bin $REMOTE_STAGE/web"
sshpass -e rsync -az --delete -e "ssh ${SSH_OPTS[*]}" \
  "$BUILD_DIR/bin/" "$SSH_USER@$HOST:$REMOTE_STAGE/bin/"
sshpass -e rsync -az --delete -e "ssh ${SSH_OPTS[*]}" \
  "$WEB_DIR/dist/" "$SSH_USER@$HOST:$REMOTE_STAGE/web/"
sshpass -e rsync -az -e "ssh ${SSH_OPTS[*]}" \
  "$DEPLOY_DIR/higo-api.service" "$DEPLOY_DIR/higo-worker.service" \
  "$DEPLOY_DIR/server.env" "$SSH_USER@$HOST:$REMOTE_STAGE/"

# --- 4. runtime packages ----------------------------------------------------
if [[ "$SKIP_DEPS" != "1" ]]; then
  log "Installing/verifying runtime packages"
  rsudo "DEBIAN_FRONTEND=noninteractive apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y ${RUNTIME_PKGS[*]}" \
    || echo "WARN: package install reported errors (continuing)"
fi

# --- 5. install + restart ---------------------------------------------------
log "Installing binaries, web, units and restarting services"
rsudo "set -e
  install -d $INSTALL_BIN $INSTALL_WEB /etc/higoos /var/lib/higoos/state /srv/higoos/nas
  systemctl stop higo-worker higo-api 2>/dev/null || true
  install -m 0755 $REMOTE_STAGE/bin/higo-api    $INSTALL_BIN/higo-api
  install -m 0755 $REMOTE_STAGE/bin/higo-worker $INSTALL_BIN/higo-worker
  install -m 0755 $REMOTE_STAGE/bin/higoctl      $INSTALL_BIN/higoctl
  rsync -a --delete $REMOTE_STAGE/web/ $INSTALL_WEB/
  [ -f /etc/higoos/server.env ] || install -m 0644 $REMOTE_STAGE/server.env /etc/higoos/server.env
  install -m 0644 $REMOTE_STAGE/higo-api.service    /etc/systemd/system/higo-api.service
  install -m 0644 $REMOTE_STAGE/higo-worker.service /etc/systemd/system/higo-worker.service
  systemctl daemon-reload
  systemctl enable --now higo-api higo-worker"

# --- 6. health check --------------------------------------------------------
log "Health check"
sleep 2
rsudo "systemctl is-active higo-api higo-worker; echo '--- health ---'; curl -fsS $HEALTH_URL || curl -fsS http://127.0.0.1:8080/ -o /dev/null -w 'http %{http_code}\n'"

log "Deploy complete: http://$HOST:8080/"
