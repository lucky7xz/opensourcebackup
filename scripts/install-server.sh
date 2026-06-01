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

# Fix locale warnings in minimal containers (LXC)
export LANG=C.UTF-8 LC_ALL=C.UTF-8 DEBIAN_FRONTEND=noninteractive

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

DB_USER="opensourcebackup"
DB_NAME="opensourcebackup"
CREDS_FILE="/etc/opensourcebackup/.db_password"
LOCAL_IP=$(hostname -I | awk '{print $1}')

# Generate password once and persist it — never regenerate on re-runs
mkdir -p /etc/opensourcebackup
if [ -f "$CREDS_FILE" ]; then
  DB_PASSWORD=$(cat "$CREDS_FILE")
  ok "Using existing database password"
else
  DB_PASSWORD=$(openssl rand -hex 32)
  echo "$DB_PASSWORD" > "$CREDS_FILE"
  chmod 600 "$CREDS_FILE"
  ok "New database password generated"
fi

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
info "Waiting for PostgreSQL to be ready (up to 60s)..."
READY=0
for i in {1..30}; do
  if docker exec opensourcebackup-postgres-1 \
      pg_isready -U "$DB_USER" &>/dev/null 2>&1; then
    READY=1; break
  fi
  echo -n "."
  sleep 2
done
echo ""
[ "$READY" -eq 1 ] && ok "PostgreSQL + Redis running" || die "PostgreSQL did not become ready in time. Check: docker compose -f $OSB_INSTALL_DIR/docker-compose.yml logs postgres"

# ── Step 4: Download Server Binary ───────────────────────────────────────────

info "Step 4/6: Downloading OpenSourceBackup server..."
SERVER_BIN="$OSB_INSTALL_DIR/opensourcebackup-server"
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
DOWNLOAD_URL="https://github.com/cerberus8484/opensourcebackup/releases/download/${OSB_VERSION}/opensourcebackup-server-linux-${ARCH}"

# Ensure Go 1.22+ is available (Debian apt only ships 1.19 which is too old)
ensure_go_122() {
  local GO_MIN="1.22"
  local GO_BINARY="go"
  # Check if a sufficient version is already installed
  if command -v go &>/dev/null; then
    local VER
    VER=$(go version | grep -oP '\d+\.\d+' | head -1)
    if awk -v v="$VER" -v m="$GO_MIN" 'BEGIN{if(v+0>=m+0)exit 0; exit 1}'; then
      ok "Go $VER already installed"; return 0
    fi
  fi
  info "Installing Go 1.22.5 (Debian ships 1.19 which is too old)..."
  curl -fsSL "https://go.dev/dl/go1.22.5.linux-amd64.tar.gz" | tar -C /usr/local -xz
  export PATH="/usr/local/go/bin:$PATH"
  ok "Go $(go version | grep -oP '\d+\.\d+\.\d+' | head -1) installed"
}

# Fallback: build from source if release not available
if ! curl -fsSL --head "$DOWNLOAD_URL" &>/dev/null; then
  warn "Binary release not found. Building from source..."
  ensure_go_122
  rm -rf /tmp/osb-build
  git clone --depth=1 https://github.com/cerberus8484/opensourcebackup.git /tmp/osb-build
  cd /tmp/osb-build
  /usr/local/go/bin/go build -o "$SERVER_BIN" ./cmd/control-plane || go build -o "$SERVER_BIN" ./cmd/control-plane
  cd /
  rm -rf /tmp/osb-build
else
  curl -fsSL "$DOWNLOAD_URL" -o "$SERVER_BIN"
fi

chmod +x "$SERVER_BIN"
chown "$OSB_USER:$OSB_USER" "$SERVER_BIN"
ok "Server binary ready: $SERVER_BIN"

# ── Step 4b: Web UI (optional — pre-built or from source) ────────────────────

info "Step 4b/6: Installing Web UI..."
WEB_UI_DIR="$OSB_INSTALL_DIR/web-ui"
mkdir -p "$WEB_UI_DIR"

# Try to download pre-built web UI (future releases will publish it)
WEB_UI_URL="https://github.com/cerberus8484/opensourcebackup/releases/download/${OSB_VERSION}/web-ui.tar.gz"
if curl -fsSL --head "$WEB_UI_URL" &>/dev/null; then
  curl -fsSL "$WEB_UI_URL" | tar -xz -C "$WEB_UI_DIR"
  ok "Web UI downloaded"
else
  # Install Node.js if not available
  if ! command -v node &>/dev/null; then
    info "Installing Node.js 20 LTS..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - >/dev/null 2>&1
    apt-get install -y -qq nodejs
    ok "Node.js $(node --version) installed"
  fi

  # Build Web UI from source
  rm -rf /tmp/osb-web-build
  git clone --depth=1 https://github.com/cerberus8484/opensourcebackup.git /tmp/osb-web-build
  if [ -d /tmp/osb-web-build/web ]; then
    cd /tmp/osb-web-build/web
    VITE_API_URL="" npm install --silent
    VITE_API_URL="" npm run build --silent
    cp -r dist/. "$WEB_UI_DIR/"
    ok "Web UI built and ready"
  fi
  cd /; rm -rf /tmp/osb-web-build
fi

chown -R "$OSB_USER:$OSB_USER" "$WEB_UI_DIR" 2>/dev/null || true

# ── Step 5: Migrations ───────────────────────────────────────────────────────

info "Step 5/6: Running database migrations..."
DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@127.0.0.1:5432/${DB_NAME}?sslmode=disable"

# Install migrate tool — download explicitly to avoid pipe/tar issues
MIGRATE_BIN="/usr/local/bin/migrate"
if ! command -v migrate &>/dev/null; then
  info "Installing golang-migrate..."
  MIGRATE_URL="https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-amd64.tar.gz"
  curl -fsSL "$MIGRATE_URL" -o /tmp/migrate.tar.gz
  tar -xzf /tmp/migrate.tar.gz -C /tmp
  # Binary may be named 'migrate' or 'migrate.linux-amd64'
  if [ -f /tmp/migrate ]; then
    mv /tmp/migrate "$MIGRATE_BIN"
  elif [ -f /tmp/migrate.linux-amd64 ]; then
    mv /tmp/migrate.linux-amd64 "$MIGRATE_BIN"
  fi
  chmod +x "$MIGRATE_BIN"
  rm -f /tmp/migrate.tar.gz /tmp/migrate.linux-amd64 2>/dev/null || true
fi

# Verify migrate is available
if ! command -v migrate &>/dev/null && [ ! -x "$MIGRATE_BIN" ]; then
  warn "migrate not found — skipping automatic migrations. Run manually after start:"
  warn "  migrate -path $OSB_INSTALL_DIR/migrations -database \"\$DATABASE_URL\" up"
else
  # Clone migrations if not building from source
  if [ ! -d "$OSB_INSTALL_DIR/migrations" ]; then
    git clone --depth=1 https://github.com/cerberus8484/opensourcebackup.git /tmp/osb-migrations
    cp -r /tmp/osb-migrations/migrations "$OSB_INSTALL_DIR/"
    rm -rf /tmp/osb-migrations
  fi

  "$MIGRATE_BIN" -path "$OSB_INSTALL_DIR/migrations" -database "$DATABASE_URL" up
  ok "Database migrations complete"
fi

# ── Step 6: systemd Service ──────────────────────────────────────────────────

info "Step 6/6: Creating systemd service..."

# Write environment file
cat > "/etc/opensourcebackup/server.env" <<EOF
DATABASE_URL=postgres://${DB_USER}:${DB_PASSWORD}@127.0.0.1:5432/${DB_NAME}?sslmode=disable
LISTEN_ADDR=:${OSB_PORT}
CORS_ORIGIN=*
WEB_UI_DIR=${WEB_UI_DIR}
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

WEB_UI_URL="http://${LOCAL_IP}:${OSB_PORT}/ui/"
API_URL="http://${LOCAL_IP}:${OSB_PORT}"

# Save credentials file — KEEP SECURE
cat > /root/opensourcebackup-credentials.txt <<CREDS
==============================================================
  OpenSourceBackup — Zugangsdaten / Access Credentials
  Installiert: $(date)
  DIESE DATEI SICHER AUFBEWAHREN
==============================================================

  Web Dashboard URL : http://${LOCAL_IP}:${OSB_PORT}/ui/
  API URL           : http://${LOCAL_IP}:${OSB_PORT}

  Benutzername      : (noch nicht erforderlich — kommt in Release 2)
  Passwort          : (noch nicht erforderlich — kommt in Release 2)

  Hinweis: Aktuell kein Login nötig.
  Nur im internen Netzwerk (Proxmox LAN) verwenden.
  Nicht ohne Auth + TLS ins Internet exposen.

  Datenbank
  Host     : 127.0.0.1:5432
  User     : ${DB_USER}
  Passwort : ${DB_PASSWORD}
  DB-Name  : ${DB_NAME}

==============================================================
CREDS
chmod 600 /root/opensourcebackup-credentials.txt

# ── Abschluss-Ausgabe ─────────────────────────────────────────────────────────

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║       ✓  OpenSourceBackup erfolgreich installiert!               ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║                                                                  ║${NC}"
echo -e "${GREEN}║   🌐  WEB DASHBOARD                                              ║${NC}"
echo -e "${GREEN}║       ${CYAN}${WEB_UI_URL}${GREEN}$(printf '%*s' $((48 - ${#WEB_UI_URL})) '')║${NC}"
echo -e "${GREEN}║                                                                  ║${NC}"
echo -e "${GREEN}║   🔑  ZUGANGSDATEN / LOGIN                                       ║${NC}"
echo -e "${GREEN}║       Benutzername : ${YELLOW}(kein Login nötig — kommt in v2)${GREEN}          ║${NC}"
echo -e "${GREEN}║       Passwort     : ${YELLOW}(kein Login nötig — kommt in v2)${GREEN}          ║${NC}"
echo -e "${GREEN}║                                                                  ║${NC}"
echo -e "${GREEN}║   🖥️  IP-ADRESSE                                                 ║${NC}"
echo -e "${GREEN}║       ${CYAN}${LOCAL_IP}${GREEN}$(printf '%*s' $((55 - ${#LOCAL_IP})) '')║${NC}"
echo -e "${GREEN}║                                                                  ║${NC}"
echo -e "${GREEN}║   ⚙️  PORT                                                       ║${NC}"
echo -e "${GREEN}║       ${CYAN}${OSB_PORT}${GREEN}$(printf '%*s' $((59 - ${#OSB_PORT})) '')║${NC}"
echo -e "${GREEN}║                                                                  ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║   ⚠  Aktuell kein Login erforderlich.                            ║${NC}"
echo -e "${GREEN}║      Nur im internen Netzwerk verwenden!                         ║${NC}"
echo -e "${GREEN}║      Auth (RBAC + Login) kommt im nächsten Release.              ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${YELLOW}Next step — install the agent on a system to back up:${NC}"
echo ""
echo -e "  ${CYAN}CONTROL_PLANE_URL=http://${LOCAL_IP}:${OSB_PORT} \\${NC}"
echo -e "  ${CYAN}ENROLLMENT_TOKEN=<token-from-dashboard> \\${NC}"
echo -e "  ${CYAN}RESTIC_PASSWORD=<your-backup-password> \\${NC}"
echo -e "  ${CYAN}RESTIC_REPO=/mnt/your-backup-target \\${NC}"
echo -e "  ${CYAN}bash <(curl -fsSL http://${LOCAL_IP}:${OSB_PORT}/scripts/install-agent.sh)${NC}"
echo ""
echo -e "  ${YELLOW}Manage service:${NC}"
echo -e "  Logs:     ${CYAN}journalctl -u opensourcebackup -f${NC}"
echo -e "  Restart:  ${CYAN}systemctl restart opensourcebackup${NC}"
echo -e "  Stop:     ${CYAN}systemctl stop opensourcebackup${NC}"
echo ""
echo -e "  ${YELLOW}Credentials saved to:${NC} /root/opensourcebackup-credentials.txt"
echo ""
