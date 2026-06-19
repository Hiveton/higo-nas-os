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
#   SKIP_WEB     set to 1 to skip building/deploying the web-pc frontend
#                (backend-only deploy)
#
# Requires on the build machine: go (>= go.mod version), node/npm, rsync, sshpass.

set -euo pipefail

# --- config -----------------------------------------------------------------
HOST="${HOST:-${1:-10.211.55.3}}"
SSH_USER="${SSH_USER:-${2:-hiveton}}"
TARGET_ARCH="${TARGET_ARCH:-arm64}"
SKIP_DEPS="${SKIP_DEPS:-0}"
SKIP_WEB="${SKIP_WEB:-0}"

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
  btrfs-progs lm-sensors rsync ffmpeg aria2 docker.io docker-compose-v2 \
  samba samba-common-bin nfs-kernel-server minidlna apache2 apache2-utils)

if [[ -z "${SSH_PASS:-}" ]]; then
  echo "ERROR: SSH_PASS env var is required (ssh/sudo password)." >&2
  exit 1
fi

command -v sshpass >/dev/null || { echo "ERROR: sshpass not installed (brew install hudochenkov/sshpass/sshpass)"; exit 1; }

export SSHPASS="$SSH_PASS"
SSH_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
  -o ConnectTimeout=10 -o LogLevel=ERROR)

rsh() { sshpass -e ssh "${SSH_OPTS[@]}" "$SSH_USER@$HOST" "$@"; }
# Run a (possibly multi-line) script on the remote as root.
# The script is shipped as a file and executed with `sudo bash <file>` instead
# of being wrapped in `sudo bash -c "$1"`. The old form re-parsed the script
# through an extra shell layer, which broke on embedded single quotes /
# redirections (e.g. the HIGO_DATABASE_URL injection). The remote temp file is
# always removed afterwards (it may contain config such as the DB DSN).
rsudo() {
  local local_script remote_script
  local_script="$(mktemp "${TMPDIR:-/tmp}/higo-rsudo.XXXXXX")"
  remote_script="/tmp/higo-rsudo.$$.sh"
  printf '%s\n' "$1" > "$local_script"
  sshpass -e scp "${SSH_OPTS[@]}" "$local_script" "$SSH_USER@$HOST:$remote_script" >/dev/null
  rm -f "$local_script"
  sshpass -e ssh "${SSH_OPTS[@]}" "$SSH_USER@$HOST" \
    "echo '$SSH_PASS' | sudo -S -p '' bash '$remote_script'; rc=\$?; rm -f '$remote_script'; exit \$rc"
}

log() { printf '\033[1;36m==>\033[0m %s\n' "$*"; }

cleanup() { rm -rf "$BUILD_DIR"; }
trap cleanup EXIT

# --- 1. build backend -------------------------------------------------------
log "Cross-compiling Go backend for linux/$TARGET_ARCH"
mkdir -p "$BUILD_DIR/bin"
( cd "$SERVER_DIR"
  for cmd in higo-api higo-worker higoctl higo-mcp; do
    CGO_ENABLED=0 GOOS=linux GOARCH="$TARGET_ARCH" \
      go build -trimpath -ldflags="-s -w" -o "$BUILD_DIR/bin/$cmd" "./cmd/$cmd"
  done )

# --- 2. build frontend ------------------------------------------------------
if [[ "$SKIP_WEB" != "1" ]]; then
  log "Building web-pc frontend"
  ( cd "$WEB_DIR"
    [[ -d node_modules ]] || npm ci
    npm run build )
else
  log "SKIP_WEB=1 — skipping frontend build (backend-only deploy)"
fi

# --- 3. ship artifacts to remote staging ------------------------------------
log "Syncing artifacts to $SSH_USER@$HOST:$REMOTE_STAGE"
rsh "rm -rf $REMOTE_STAGE && mkdir -p $REMOTE_STAGE/bin $REMOTE_STAGE/web"
sshpass -e rsync -az --delete -e "ssh ${SSH_OPTS[*]}" \
  "$BUILD_DIR/bin/" "$SSH_USER@$HOST:$REMOTE_STAGE/bin/"
if [[ "$SKIP_WEB" != "1" ]]; then
  sshpass -e rsync -az --delete -e "ssh ${SSH_OPTS[*]}" \
    "$WEB_DIR/dist/" "$SSH_USER@$HOST:$REMOTE_STAGE/web/"
fi
sshpass -e rsync -az -e "ssh ${SSH_OPTS[*]}" \
  "$DEPLOY_DIR/higo-api.service" "$DEPLOY_DIR/higo-worker.service" \
  "$DEPLOY_DIR/higoos-webdav.service" "$DEPLOY_DIR/higoos-webdav.conf" \
  "$DEPLOY_DIR/server.env" "$SSH_USER@$HOST:$REMOTE_STAGE/"

# --- 4. runtime packages ----------------------------------------------------
if [[ "$SKIP_DEPS" != "1" ]]; then
  log "Installing/verifying runtime packages"
  rsudo "DEBIAN_FRONTEND=noninteractive apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y ${RUNTIME_PKGS[*]}" \
    || echo "WARN: package install reported errors (continuing)"
fi

# --- 5. install + restart ---------------------------------------------------
log "Installing binaries, web, units and restarting services"
WEB_INSTALL="rsync -a --delete $REMOTE_STAGE/web/ $INSTALL_WEB/"
if [[ "$SKIP_WEB" == "1" ]]; then
  WEB_INSTALL="echo 'SKIP_WEB=1 — leaving installed frontend untouched'"
fi
rsudo "set -e
  install -d $INSTALL_BIN $INSTALL_WEB /etc/higoos /var/lib/higoos/state /srv/higoos/nas
  systemctl stop higo-worker higo-api 2>/dev/null || true
  install -m 0755 $REMOTE_STAGE/bin/higo-api    $INSTALL_BIN/higo-api
  install -m 0755 $REMOTE_STAGE/bin/higo-worker $INSTALL_BIN/higo-worker
  install -m 0755 $REMOTE_STAGE/bin/higoctl      $INSTALL_BIN/higoctl
  install -m 0755 $REMOTE_STAGE/bin/higo-mcp     $INSTALL_BIN/higo-mcp
  $WEB_INSTALL
  [ -f /etc/higoos/server.env ] || install -m 0644 $REMOTE_STAGE/server.env /etc/higoos/server.env
  # AI index/search backbone: pgvector container (idempotent) + inject DSN.
  docker inspect higo-postgres >/dev/null 2>&1 || docker run -d --name higo-postgres --restart unless-stopped -e POSTGRES_USER=higo -e POSTGRES_PASSWORD=higo -e POSTGRES_DB=higo -v higo-pgdata:/var/lib/postgresql/data -p 127.0.0.1:5433:5432 pgvector/pgvector:pg16
  for i in \$(seq 1 30); do docker exec higo-postgres pg_isready -U higo >/dev/null 2>&1 && break; sleep 1; done
  grep -q '^HIGO_DATABASE_URL=' /etc/higoos/server.env || echo 'HIGO_DATABASE_URL=postgres://higo:higo@127.0.0.1:5433/higo?sslmode=disable' >> /etc/higoos/server.env
  install -m 0644 $REMOTE_STAGE/higo-api.service    /etc/systemd/system/higo-api.service
  install -m 0644 $REMOTE_STAGE/higo-worker.service /etc/systemd/system/higo-worker.service
  # Sharing-protocol scaffolding. The protocols domain enables these services
  # on-demand through the governance flow, so we install config + units but do
  # NOT enable them here. Guarded with '|| true' so optional bits never abort
  # the core api/worker deploy.
  install -d /etc/samba/smb.conf.d /etc/exports.d /etc/higoos/webdav /var/lib/higoos/webdav 2>/dev/null || true
  a2enmod dav dav_fs >/dev/null 2>&1 || true
  install -m 0644 $REMOTE_STAGE/higoos-webdav.conf    /etc/higoos/webdav/higoos-webdav.conf 2>/dev/null || true
  install -m 0644 $REMOTE_STAGE/higoos-webdav.service /etc/systemd/system/higoos-webdav.service 2>/dev/null || true
  systemctl daemon-reload
  systemctl enable --now higo-api higo-worker"

# --- 6. health check --------------------------------------------------------
log "Health check"
sleep 2
rsudo "systemctl is-active higo-api higo-worker; echo '--- health ---'; curl -fsS $HEALTH_URL || curl -fsS http://127.0.0.1:8080/ -o /dev/null -w 'http %{http_code}\n'"

log "Deploy complete: http://$HOST:8080/"
