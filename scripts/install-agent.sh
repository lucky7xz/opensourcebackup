#!/usr/bin/env bash
# ==============================================================================
# OpenSourceBackup — Agent Install Script (Linux)
#
# Usage:
#   CONTROL_PLANE_URL=http://192.168.1.10:8080 \
#   ENROLLMENT_TOKEN=<token> \
#   RESTIC_PASSWORD=<password> \
#   RESTIC_REPO=/mnt/backup/restic-repo \
#   bash install-agent.sh
#
# Required variables:
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
ENV_FILE="/etc/opensourcebackup/agent.env"
SERVICE_NAME="opensourcebackup-agent"

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✓${NC} $*"; }
info() { echo -e "${CYAN}→${NC} $*"; }
die()  { echo -e "${RED}✗ ERROR:${NC} $*" >&2; exit 1; }

echo ""
echo -e "${CYAN}OpenSourceBackup — Agent Installer${NC}"
echo ""

[[ $EUID -eq 0 ]] || die "Run as root: sudo bash $0"

# Validate required vars
[[ -n "${CONTROL_PLANE_URL:-}" ]] || die "CONTROL_PLANE_URL is required"
[[ -n "${ENROLLMENT_TOKEN:-}"  ]] || die "ENROLLMENT_TOKEN is required"
[[ -n "${RESTIC_PASSWORD:-}"   ]] || die "RESTIC_PASSWORD is required"
[[ -n "${RESTIC_REPO:-}"       ]] || die "RESTIC_REPO is required"

# ── Step 1: Download Agent ────────────────────────────────────────────────────

info "Downloading agent binary..."
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
AGENT_BIN="$INSTALL_DIR/opensourcebackup-agent"

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

# ── Step 3: Directories + Env ────────────────────────────────────────────────

info "Creating directories..."
mkdir -p "$DATA_DIR" /etc/opensourcebackup
chmod 700 "$DATA_DIR"

cat > "$ENV_FILE" <<EOF
CONTROL_PLANE_URL=${CONTROL_PLANE_URL}
ENROLLMENT_TOKEN=${ENROLLMENT_TOKEN}
RESTIC_PASSWORD=${RESTIC_PASSWORD}
RESTIC_REPO=${RESTIC_REPO}
RESTIC_BIN=/usr/local/bin/restic
AGENT_POLL_INTERVAL=${AGENT_POLL_INTERVAL:-30s}
RESTORE_TEST_ROOT=${RESTORE_TEST_ROOT:-${DATA_DIR}/restore-tests}
AGENT_TOKEN_FILE=${DATA_DIR}/agent-token
EOF
chmod 600 "$ENV_FILE"
ok "Environment file: $ENV_FILE"

# ── Step 4: systemd Service ──────────────────────────────────────────────────

info "Creating systemd service..."
cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=OpenSourceBackup Agent
Documentation=https://github.com/cerberus8484/opensourcebackup
After=network.target

[Service]
Type=simple
WorkingDirectory=${DATA_DIR}
EnvironmentFile=${ENV_FILE}
ExecStart=${AGENT_BIN}
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=${SERVICE_NAME}

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl start "$SERVICE_NAME"
sleep 3

# ── Done ─────────────────────────────────────────────────────────────────────
echo ""
ok "Agent installed and started!"
echo ""
echo -e "  ${YELLOW}Commands:${NC}"
echo -e "  Logs:     ${CYAN}journalctl -u ${SERVICE_NAME} -f${NC}"
echo -e "  Status:   ${CYAN}systemctl status ${SERVICE_NAME}${NC}"
echo -e "  Stop:     ${CYAN}systemctl stop ${SERVICE_NAME}${NC}"
echo -e "  Restart:  ${CYAN}systemctl restart ${SERVICE_NAME}${NC}"
echo ""
echo -e "  Token stored at: ${CYAN}${DATA_DIR}/agent-token${NC}"
echo ""
