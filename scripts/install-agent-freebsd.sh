#!/bin/sh
# ==============================================================================
# OpenSourceBackup — Agent Install Script (FreeBSD / OPNsense)
#
# Usage:
#   CONTROL_PLANE_URL=http://192.168.1.10:8080 \
#   ENROLLMENT_TOKEN=<token> \
#   RESTIC_PASSWORD=<password> \
#   RESTIC_REPO=/mnt/backup/restic-repo \
#   sh install-agent-freebsd.sh
#
# Required:
#   CONTROL_PLANE_URL   URL of your Control Plane
#   ENROLLMENT_TOKEN    One-time enrollment token from the UI
#   RESTIC_PASSWORD     Encryption password for backups
#   RESTIC_REPO         Backup destination
#
# Optional:
#   AGENT_POLL_INTERVAL  Default: 30s
#   RESTORE_TEST_ROOT    Default: /var/db/opensourcebackup/restore-tests
#   OSB_VERSION          Default: v0.1.0
# ==============================================================================

set -eu

OSB_VERSION="${OSB_VERSION:-v0.1.0}"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/db/opensourcebackup/agent"
RC_SCRIPT="/usr/local/etc/rc.d/opensourcebackup_agent"

ok()   { echo "[OK]  $*"; }
info() { echo "[-->] $*"; }
die()  { echo "[ERR] $*" >&2; exit 1; }

echo ""
echo "OpenSourceBackup — Agent Installer (FreeBSD / OPNsense)"
echo ""

[ "$(id -u)" -eq 0 ] || die "Run as root"

[ -n "${CONTROL_PLANE_URL:-}" ] || die "CONTROL_PLANE_URL is required"
[ -n "${ENROLLMENT_TOKEN:-}"  ] || die "ENROLLMENT_TOKEN is required"
[ -n "${RESTIC_PASSWORD:-}"   ] || die "RESTIC_PASSWORD is required"
[ -n "${RESTIC_REPO:-}"       ] || die "RESTIC_REPO is required"

# ── Step 1: Download Agent ────────────────────────────────────────────────────

AGENT_BIN="${INSTALL_DIR}/opensourcebackup-agent"
info "Downloading agent (freebsd-amd64)..."

# Use fetch (FreeBSD native) or curl if available
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "${CONTROL_PLANE_URL}/downloads/agent/${OSB_VERSION}/freebsd-amd64" \
    -o "$AGENT_BIN"
else
  fetch -q -o "$AGENT_BIN" \
    "${CONTROL_PLANE_URL}/downloads/agent/${OSB_VERSION}/freebsd-amd64"
fi
chmod 755 "$AGENT_BIN"
ok "Agent downloaded: $AGENT_BIN"

# ── Step 2: Install restic ────────────────────────────────────────────────────

if ! command -v restic >/dev/null 2>&1; then
  info "Installing restic via pkg..."
  pkg install -y restic || {
    info "pkg install failed, trying ports..."
    RESTIC_VERSION="0.17.3"
    if command -v curl >/dev/null 2>&1; then
      curl -fsSL "https://github.com/restic/restic/releases/download/v${RESTIC_VERSION}/restic_${RESTIC_VERSION}_freebsd_amd64.bz2" \
        | bunzip2 > /usr/local/bin/restic
    else
      fetch -q -o - \
        "https://github.com/restic/restic/releases/download/v${RESTIC_VERSION}/restic_${RESTIC_VERSION}_freebsd_amd64.bz2" \
        | bunzip2 > /usr/local/bin/restic
    fi
    chmod 755 /usr/local/bin/restic
  }
  ok "restic installed"
else
  ok "restic already installed"
fi

# ── Step 3: Directories ──────────────────────────────────────────────────────

mkdir -p "$DATA_DIR"
chmod 700 "$DATA_DIR"
ok "Data directory: $DATA_DIR"

# ── Step 4: Install rc.d service via agent binary ────────────────────────────

info "Installing rc.d service..."

export CONTROL_PLANE_URL
export ENROLLMENT_TOKEN
export RESTIC_PASSWORD
export RESTIC_REPO
export RESTIC_BIN="/usr/local/bin/restic"
export AGENT_POLL_INTERVAL="${AGENT_POLL_INTERVAL:-30s}"
export RESTORE_TEST_ROOT="${RESTORE_TEST_ROOT:-${DATA_DIR}/restore-tests}"
export AGENT_TOKEN_FILE="${DATA_DIR}/agent-token"

# The agent binary self-registers as an rc.d service (kardianos/service).
"$AGENT_BIN" install
ok "rc.d service registered"

# Enable in /etc/rc.conf
if ! grep -q "opensourcebackup_agent_enable" /etc/rc.conf 2>/dev/null; then
  echo 'opensourcebackup_agent_enable="YES"' >> /etc/rc.conf
fi

# ── Step 5: Start ─────────────────────────────────────────────────────────────

info "Starting service..."
"$AGENT_BIN" start
sleep 2
"$AGENT_BIN" status

# ── Done ─────────────────────────────────────────────────────────────────────
echo ""
echo "✓ Agent is running as an rc.d service!"
echo ""
echo "  Commands:"
echo "  Logs:     tail -f /var/log/opensourcebackup-agent.log"
echo "  Status:   ${AGENT_BIN} status"
echo "  Stop:     ${AGENT_BIN} stop"
echo "  Restart:  ${AGENT_BIN} restart"
echo "  Remove:   ${AGENT_BIN} stop && ${AGENT_BIN} uninstall"
echo ""
echo "  Token:    ${DATA_DIR}/agent-token"
echo ""
