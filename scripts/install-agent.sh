#!/usr/bin/env bash
# ==============================================================================
# OpenSourceBackup — Agent Install Script (Linux / systemd)
#
# Usage:
#   CONTROL_PLANE_URL=http://192.168.1.10:8080 \
#   ENROLLMENT_TOKEN=<token> \
#   RESTIC_PASSWORD=<password> \
#   RESTIC_REPO=/mnt/backup/restic-repo \
#   bash install-agent.sh
#
# Required:
#   CONTROL_PLANE_URL   URL of your Control Plane
#   ENROLLMENT_TOKEN    One-time enrollment token from the UI
#   RESTIC_PASSWORD     Encryption password for backups
#   RESTIC_REPO         Backup destination (path, S3, NFS mount, etc.)
#
# Optional:
#   AGENT_POLL_INTERVAL  Default: 30s
#   RESTORE_TEST_ROOT    Default: /var/lib/opensourcebackup/restore-tests
#   OSB_VERSION          Default: v0.1.0
# ==============================================================================

set -euo pipefail

OSB_VERSION="${OSB_VERSION:-v0.1.0}"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/opensourcebackup/agent"
SERVICE_NAME="opensourcebackup-agent"

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✓${NC} $*"; }
info() { echo -e "${CYAN}→${NC} $*"; }
die()  { echo -e "${RED}✗ ERROR:${NC} $*" >&2; exit 1; }

echo ""
echo -e "${CYAN}OpenSourceBackup — Agent Installer (Linux)${NC}"
echo ""

[[ $EUID -eq 0 ]] || die "Run as root: sudo bash $0"

[[ -n "${CONTROL_PLANE_URL:-}" ]] || die "CONTROL_PLANE_URL is required"
[[ -n "${ENROLLMENT_TOKEN:-}"  ]] || die "ENROLLMENT_TOKEN is required"
[[ -n "${RESTIC_PASSWORD:-}"   ]] || die "RESTIC_PASSWORD is required"
[[ -n "${RESTIC_REPO:-}"       ]] || die "RESTIC_REPO is required"

# ── Step 1: Download Agent ────────────────────────────────────────────────────

info "Detecting architecture..."
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
AGENT_BIN="$INSTALL_DIR/opensourcebackup-agent"
info "Downloading agent (linux-${ARCH})..."

curl -fsSL "${CONTROL_PLANE_URL}/downloads/agent/${OSB_VERSION}/linux-${ARCH}" \
  -o "$AGENT_BIN"
chmod +x "$AGENT_BIN"
ok "Agent downloaded: $AGENT_BIN"

# ── Step 2: Install restic ────────────────────────────────────────────────────

if ! command -v restic &>/dev/null; then
  info "Installing restic..."
  RESTIC_VERSION="0.17.3"
  curl -fsSL "https://github.com/restic/restic/releases/download/v${RESTIC_VERSION}/restic_${RESTIC_VERSION}_linux_${ARCH}.bz2" \
    | bunzip2 > /usr/local/bin/restic
  chmod +x /usr/local/bin/restic
  ok "restic installed"
else
  ok "restic already installed ($(restic version | head -1))"
fi

# ── Step 3: Directories ──────────────────────────────────────────────────────

info "Creating data directory..."
mkdir -p "$DATA_DIR"
chmod 700 "$DATA_DIR"
ok "Data directory: $DATA_DIR"

# ── Step 4: Install service via agent binary ─────────────────────────────────

info "Installing systemd service..."

export CONTROL_PLANE_URL
export ENROLLMENT_TOKEN
export RESTIC_PASSWORD
export RESTIC_REPO
export RESTIC_BIN="/usr/local/bin/restic"
export AGENT_POLL_INTERVAL="${AGENT_POLL_INTERVAL:-30s}"
export RESTORE_TEST_ROOT="${RESTORE_TEST_ROOT:-${DATA_DIR}/restore-tests}"
export AGENT_TOKEN_FILE="${DATA_DIR}/agent-token"

# The agent binary self-installs as a systemd service,
# embedding the current environment variables.
"$AGENT_BIN" install
ok "systemd service installed (opensourcebackup-agent)"

# ── Step 5: Start ─────────────────────────────────────────────────────────────

info "Starting service..."
"$AGENT_BIN" start
sleep 2
"$AGENT_BIN" status

# ── Done ─────────────────────────────────────────────────────────────────────
echo ""
echo -e "  ${GREEN}✓ Agent is running as a systemd service!${NC}"
echo ""
echo -e "  ${YELLOW}Commands:${NC}"
echo -e "  Logs:     ${CYAN}journalctl -u ${SERVICE_NAME} -f${NC}"
echo -e "  Status:   ${CYAN}${AGENT_BIN} status${NC}"
echo -e "  Stop:     ${CYAN}${AGENT_BIN} stop${NC}"
echo -e "  Restart:  ${CYAN}${AGENT_BIN} restart${NC}"
echo -e "  Remove:   ${CYAN}${AGENT_BIN} stop && ${AGENT_BIN} uninstall${NC}"
echo ""
echo -e "  Token:    ${CYAN}${DATA_DIR}/agent-token${NC}"
echo ""
