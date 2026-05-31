#!/usr/bin/env bash
# ==============================================================================
# OpenSourceBackup — Server Install Script
# Installs the Control Plane on Debian 12 (Proxmox VE 8.x)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh | sudo bash
#
# Or with custom port:
#   OSB_PORT=8443 bash install-server.sh
#
# What this script does:
#   1. Install Docker + Docker Compose
#   2. Start PostgreSQL 16 + Redis 7 via Docker
#   3. Download opensourcebackup-server binary
#   4. Run database migrations
#   5. Create systemd service
#   6. Print first-run instructions
# ==============================================================================

set -euo pipefail

OSB_VERSION="${OSB_VERSION:-v0.1.0}"
OSB_PORT="${OSB_PORT:-8080}"
OSB_INSTALL_DIR="${OSB_INSTALL_DIR:-/opt/opensourcebackup}"
OSB_DATA_DIR="${OSB_DATA_DIR:-/var/lib/opensourcebackup}"
OSB_USER="${OSB_USER:-osb}"

# Colors
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✓${NC} $*"; }
info() { echo -e "${CYAN}→${NC} $*"; }
warn() { echo -e "${YELLOW}⚠${NC} $*"; }
die()  { echo -e "${RED}✗ ERROR:${NC} $*" >&2; exit 1; }

echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║   OpenSourceBackup — Control Plane Installer     ║${NC}"
echo -e "${CYAN}║   Creating backups is easy.                      ║${NC}"
echo -e "${CYAN}║   Proving recoverability is the difference.      ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════╝${NC}"
echo ""

# ── Checks ────────────────────────────────────────────────────────────────────

[[ $EUID -eq 0 ]] || die "Run as root: sudo bash $0"

# Detect Proxmox / Debian
if [ -f /etc/debian_version ]; then
  ok "Debian-based system detected"
else
  warn "Not a Debian system. Script tested on Proxmox VE 8 / Debian 12."
  warn "Continuing anyway — manual fixes may be needed."
fi

# Generate secure passwords
DB_PASSWORD=$(openssl rand -hex 32)
DB_USER="opensourcebackup"
DB_NAME="opensourcebackup"

# ── Step 1: Docker ────────────────────────────────────────────────────────────

info "Step 1/6: Installing Docker..."
if command -v docker &>/dev/null; then
  ok "Docker already installed ($(docker --version | cut -d' ' -f3 | tr -d ','))"
else
  apt-get update -qq
  apt-get install -y -qq ca-certificates curl gnupg lsb-release
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
    https://download.docker.com/linux/debian $(lsb_release -cs) stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -qq
  apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin
  systemctl enable --now docker
  ok "Docker installed"
fi

# ── Step 2: Directories + User ───────────────────────────────────────────────

info "Step 2/6: Creating directories and user..."
mkdir -p "$OSB_INSTALL_DIR" "$OSB_DATA_DIR/postgres" "$OSB_DATA_DIR/certs"

# Create system user
if ! id "$OSB_USER" &>/dev/null; then
  useradd --system --home-dir "$OSB_INSTALL_DIR" --shell /usr/sbin/nologin "$OSB_USER"
  ok "User '$OSB_USER' created"
fi
chown -R "$OSB_USER:$OSB_USER" "$OSB_INSTALL_DIR" "$OSB_DATA_DIR"
ok "Directories ready"

# ── Step 3: PostgreSQL + Redis ────────────────────────────────────────────────

info "Step 3/6: Starting PostgreSQL 16 + Redis 7..."
cat > "$OSB_INSTALL_DIR/docker-compose.yml" <<EOF
services:
  postgres:
    image: postgres:16-alpine
    restart: always
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - ${OSB_DATA_DIR}/postgres:/var/lib/postgresql/data
    ports:
      - "127.0.0.1:5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 5s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7-alpine
    restart: always
    ports:
      - "127.0.0.1:6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 10
EOF

docker compose -f "$OSB_INSTALL_DIR/docker-compose.yml" up -d
info "Waiting for PostgreSQL to be ready..."
sleep 5
for i in {1..20}; do
  docker compose -f "$OSB_INSTALL_DIR/docker-compose.yml" exec -T postgres \
    pg_isready -U "$DB_USER" &>/dev/null && break
  sleep 2
done
ok "PostgreSQL + Redis running"

# ── Step 4: Download Server Binary ───────────────────────────────────────────

info "Step 4/6: Downloading OpenSourceBackup server..."
SERVER_BIN="$OSB_INSTALL_DIR/opensourcebackup-server"
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
DOWNLOAD_URL="https://github.com/cerberus8484/opensourcebackup/releases/download/${OSB_VERSION}/opensourcebackup-server-linux-${ARCH}"

# Fallback: build from source if release not available
if ! curl -fsSL --head "$DOWNLOAD_URL" &>/dev/null; then
  warn "Binary release not found. Building from source (requires Go 1.22+)..."
  apt-get install -y -qq golang-go 2>/dev/null || {
    warn "Go not available via apt. Installing Go 1.22 manually..."
    curl -fsSL "https://go.dev/dl/go1.22.5.linux-amd64.tar.gz" | tar -C /usr/local -xz
    export PATH="$PATH:/usr/local/go/bin"
  }
  cd /tmp
  git clone --depth=1 https://github.com/cerberus8484/opensourcebackup.git osb-build
  cd osb-build
  go build -o "$SERVER_BIN" ./cmd/control-plane
  cd /
  rm -rf /tmp/osb-build
else
  curl -fsSL "$DOWNLOAD_URL" -o "$SERVER_BIN"
fi

chmod +x "$SERVER_BIN"
chown "$OSB_USER:$OSB_USER" "$SERVER_BIN"
ok "Server binary ready: $SERVER_BIN"

# ── Step 5: Migrations ───────────────────────────────────────────────────────

info "Step 5/6: Running database migrations..."
DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@127.0.0.1:5432/${DB_NAME}?sslmode=disable"

# Install migrate tool
if ! command -v migrate &>/dev/null; then
  curl -fsSL "https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-amd64.tar.gz" \
    | tar -xz -C /usr/local/bin migrate
fi

# Clone migrations if not building from source
if [ ! -d "$OSB_INSTALL_DIR/migrations" ]; then
  git clone --depth=1 https://github.com/cerberus8484/opensourcebackup.git /tmp/osb-migrations
  cp -r /tmp/osb-migrations/migrations "$OSB_INSTALL_DIR/"
  rm -rf /tmp/osb-migrations
fi

migrate -path "$OSB_INSTALL_DIR/migrations" -database "$DATABASE_URL" up
ok "Database migrations complete"

# ── Step 6: systemd Service ──────────────────────────────────────────────────

info "Step 6/6: Creating systemd service..."

# Write environment file
cat > "/etc/opensourcebackup/server.env" <<EOF
DATABASE_URL=postgres://${DB_USER}:${DB_PASSWORD}@127.0.0.1:5432/${DB_NAME}?sslmode=disable
LISTEN_ADDR=:${OSB_PORT}
CORS_ORIGIN=*
# TLS (optional — fill in to enable HTTPS)
TLS_CERT_FILE=
TLS_KEY_FILE=
EOF
chmod 600 /etc/opensourcebackup/server.env
mkdir -p /etc/opensourcebackup

cat > /etc/systemd/system/opensourcebackup.service <<EOF
[Unit]
Description=OpenSourceBackup Control Plane
Documentation=https://github.com/cerberus8484/opensourcebackup
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=${OSB_USER}
WorkingDirectory=${OSB_INSTALL_DIR}
EnvironmentFile=/etc/opensourcebackup/server.env
ExecStart=${SERVER_BIN}
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=opensourcebackup

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable opensourcebackup
systemctl start opensourcebackup
sleep 3

# ── Done ─────────────────────────────────────────────────────────────────────

LOCAL_IP=$(hostname -I | awk '{print $1}')
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   Installation complete!                          ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  Control Plane:  ${CYAN}http://${LOCAL_IP}:${OSB_PORT}${NC}"
echo -e "  Health check:   ${CYAN}http://${LOCAL_IP}:${OSB_PORT}/health${NC}"
echo ""
echo -e "  ${YELLOW}Next steps:${NC}"
echo -e "  1. Open the Web-UI (run on your laptop):"
echo -e "     ${CYAN}VITE_API_URL=http://${LOCAL_IP}:${OSB_PORT} npm run dev${NC}"
echo ""
echo -e "  2. Register a system + generate enrollment token:"
echo -e "     ${CYAN}curl -X POST http://${LOCAL_IP}:${OSB_PORT}/v1/systems \\${NC}"
echo -e "     ${CYAN}  -d '{\"Hostname\":\"my-system\",\"RiskClass\":\"standard\"}'${NC}"
echo ""
echo -e "  3. Install agent on target systems:"
echo -e "     ${CYAN}CONTROL_PLANE_URL=http://${LOCAL_IP}:${OSB_PORT} \\${NC}"
echo -e "     ${CYAN}ENROLLMENT_TOKEN=<token> \\${NC}"
echo -e "     ${CYAN}RESTIC_PASSWORD=<password> \\${NC}"
echo -e "     ${CYAN}RESTIC_REPO=<path-or-s3-url> \\${NC}"
echo -e "     ${CYAN}./opensourcebackup-agent${NC}"
echo ""
echo -e "  ${YELLOW}Logs:${NC}  journalctl -u opensourcebackup -f"
echo -e "  ${YELLOW}Stop:${NC}  systemctl stop opensourcebackup"
echo ""

# Save credentials
cat > /root/opensourcebackup-credentials.txt <<EOF
# OpenSourceBackup — Installation Credentials
# $(date)

CONTROL_PLANE_URL=http://${LOCAL_IP}:${OSB_PORT}

DATABASE_URL=postgres://${DB_USER}:${DB_PASSWORD}@127.0.0.1:5432/${DB_NAME}?sslmode=disable
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}
EOF
chmod 600 /root/opensourcebackup-credentials.txt
echo -e "  ${YELLOW}Credentials saved to:${NC} /root/opensourcebackup-credentials.txt"
echo ""
