#!/usr/bin/env bash
# ==============================================================================
# OpenSourceBackup — Vollautomatischer LXC-Installer für Proxmox VE
#
# Läuft auf dem PROXMOX HOST als root.
# Erstellt einen Debian-12-LXC-Container und installiert OpenSourceBackup
# vollständig, inklusive:
#   - Docker + PostgreSQL 16 + Redis 7
#   - Control Plane (Go-Binary)
#   - Web UI (React)
#   - Datenbank-Migrationen
#   - Systemd-Dienst
#   - Admin-Benutzer
#   - Credentials-Ausgabe am Ende
#
# Usage:
#   bash <(curl -fsSL https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-lxc.sh)
#
# Oder mit eigener Konfiguration:
#   CT_ID=201 ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=sicher123 \
#   bash install-lxc.sh
# ==============================================================================

set -euo pipefail

# ╔══════════════════════════════════════════════════════════════════════════╗
# ║                    KONFIGURATION — HIER ANPASSEN                        ║
# ╚══════════════════════════════════════════════════════════════════════════╝

# Container-ID (leer = automatisch, nächste freie ab 200)
CT_ID="${CT_ID:-}"

# Hostname des Containers
CT_HOSTNAME="${CT_HOSTNAME:-opensourcebackup}"

# Container-Ressourcen
CT_MEMORY="${CT_MEMORY:-2048}"   # MB RAM
CT_CORES="${CT_CORES:-2}"        # CPU-Kerne
CT_DISK="${CT_DISK:-20}"         # GB Disk

# Proxmox-Storage (leer = automatisch erkannt)
CT_STORAGE="${CT_STORAGE:-}"

# Netzwerk
CT_BRIDGE="${CT_BRIDGE:-vmbr0}"
CT_IP="${CT_IP:-dhcp}"           # "dhcp" oder "192.168.1.50/24"
CT_GW="${CT_GW:-}"               # Gateway (nur bei statischer IP nötig)

# Dashboard-Port
OSB_PORT="${OSB_PORT:-8080}"

# Admin-Zugangsdaten (leer = werden zufällig generiert)
ADMIN_EMAIL="${ADMIN_EMAIL:-}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"

# ╔══════════════════════════════════════════════════════════════════════════╗
# ║                      SCRIPT — NICHT ÄNDERN                              ║
# ╚══════════════════════════════════════════════════════════════════════════╝

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'
BOLD='\033[1m'; NC='\033[0m'

ok()    { echo -e "  ${GREEN}✓${NC} $*"; }
info()  { echo -e "  ${CYAN}→${NC} $*"; }
warn()  { echo -e "  ${YELLOW}⚠${NC} $*"; }
die()   { echo -e "\n  ${RED}✗ FEHLER:${NC} $*\n" >&2; exit 1; }
title() { echo -e "\n${CYAN}━━━ $* ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

echo ""
echo -e "${BOLD}${CYAN}  ╔═══════════════════════════════════════════════════╗${NC}"
echo -e "${BOLD}${CYAN}  ║  OpenSourceBackup — LXC Installer                ║${NC}"
echo -e "${BOLD}${CYAN}  ║  Creating backups is easy.                        ║${NC}"
echo -e "${BOLD}${CYAN}  ║  Proving recoverability is the difference.        ║${NC}"
echo -e "${BOLD}${CYAN}  ╚═══════════════════════════════════════════════════╝${NC}"
echo ""

# ── Voraussetzungen ───────────────────────────────────────────────────────────

title "Schritt 1/7: Voraussetzungen prüfen"

[[ $EUID -eq 0 ]] || die "Als root ausführen!"
command -v pct   &>/dev/null || die "pct nicht gefunden — kein Proxmox-Host?"
command -v pveam &>/dev/null || die "pveam nicht gefunden — kein Proxmox-Host?"

# Container-ID automatisch ermitteln
if [ -z "$CT_ID" ]; then
  CT_ID=200
  while pct status "$CT_ID" &>/dev/null 2>&1; do CT_ID=$((CT_ID+1)); done
  info "Verwende Container-ID: $CT_ID (automatisch)"
else
  if pct status "$CT_ID" &>/dev/null 2>&1; then
    warn "Container $CT_ID existiert bereits — wird gelöscht!"
    pct stop "$CT_ID" 2>/dev/null || true
    sleep 2
    pct destroy "$CT_ID" --purge
    ok "Alter Container entfernt"
  fi
fi

# Storage automatisch ermitteln
if [ -z "$CT_STORAGE" ]; then
  if pvesm status 2>/dev/null | grep -q "local-lvm"; then
    CT_STORAGE="local-lvm"
  elif pvesm status 2>/dev/null | grep -q "local"; then
    CT_STORAGE="local"
  else
    CT_STORAGE=$(pvesm status 2>/dev/null | awk 'NR>1{print $1}' | head -1)
  fi
  info "Verwende Storage: $CT_STORAGE (automatisch)"
fi

# Admin-Credentials generieren falls nicht gesetzt
if [ -z "$ADMIN_EMAIL" ]; then
  ADMIN_EMAIL="admin@opensourcebackup.local"
  info "Admin-E-Mail: $ADMIN_EMAIL (Standard)"
fi
if [ -z "$ADMIN_PASSWORD" ]; then
  ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
  info "Admin-Passwort: wird automatisch generiert"
fi

ok "Alle Voraussetzungen erfüllt"

# ── Template ──────────────────────────────────────────────────────────────────

title "Schritt 2/7: Debian 12 Template"

info "Template-Liste aktualisieren..."
pveam update 2>/dev/null | tail -1 || true

LOCAL_TPL=$(pveam list local 2>/dev/null | grep "debian-12-standard" | awk '{print $1}' | sort -V | tail -1 || true)
if [ -z "$LOCAL_TPL" ]; then
  AVAIL=$(pveam available 2>/dev/null | grep "debian-12-standard" | awk '{print $2}' | sort -V | tail -1 || true)
  [ -n "$AVAIL" ] || die "Kein Debian-12-Template verfügbar.\nManuell herunterladen:\n  pveam download local debian-12-standard_12.7-1_amd64.tar.zst"
  info "Lade Template herunter: $AVAIL"
  pveam download local "$AVAIL"
  LOCAL_TPL=$(pveam list local | grep "debian-12-standard" | awk '{print $1}' | tail -1)
fi
ok "Template: $LOCAL_TPL"

# ── Container erstellen ────────────────────────────────────────────────────────

title "Schritt 3/7: LXC-Container erstellen"

NET_CONFIG="name=eth0,bridge=${CT_BRIDGE}"
if [ "$CT_IP" = "dhcp" ]; then
  NET_CONFIG="${NET_CONFIG},ip=dhcp"
else
  NET_CONFIG="${NET_CONFIG},ip=${CT_IP}"
  [ -n "$CT_GW" ] && NET_CONFIG="${NET_CONFIG},gw=${CT_GW}"
fi

info "Erstelle Container $CT_ID ($CT_HOSTNAME)..."
pct create "$CT_ID" "$LOCAL_TPL" \
  --hostname    "$CT_HOSTNAME" \
  --cores       "$CT_CORES" \
  --memory      "$CT_MEMORY" \
  --rootfs      "${CT_STORAGE}:${CT_DISK}" \
  --net0        "$NET_CONFIG" \
  --features    "nesting=1" \
  --unprivileged 1 \
  --ostype      debian \
  --start       0

info "Starte Container..."
pct start "$CT_ID"
sleep 6

for i in {1..25}; do
  pct exec "$CT_ID" -- echo ok &>/dev/null 2>&1 && break || { echo -n "."; sleep 2; }
done
echo ""
ok "Container $CT_ID läuft"

# ── Installation im Container ─────────────────────────────────────────────────

title "Schritt 4/7: System vorbereiten"

pct exec "$CT_ID" -- bash -c "
  export LANG=C.UTF-8 LC_ALL=C.UTF-8 DEBIAN_FRONTEND=noninteractive

  # Locales konfigurieren (unterdrückt perl-Warnungen)
  apt-get install -y -qq locales 2>/dev/null
  locale-gen en_US.UTF-8 C.UTF-8 2>/dev/null || true
  update-locale LANG=C.UTF-8 2>/dev/null || true

  apt-get update -qq
  apt-get install -y -qq curl git ca-certificates openssl
  echo 'DONE'
" | tail -1 | grep -q DONE && ok "Basispakete installiert" || warn "Basispakete — Warnungen ignoriert"

# ── Docker + PostgreSQL + Redis ───────────────────────────────────────────────

title "Schritt 5/7: Docker + PostgreSQL + Redis"

pct exec "$CT_ID" -- bash -c "
  export LANG=C.UTF-8 LC_ALL=C.UTF-8 DEBIAN_FRONTEND=noninteractive
  set -e

  # Docker installieren
  apt-get install -y -qq ca-certificates curl gnupg lsb-release
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo \"deb [arch=\$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian \$(lsb_release -cs) stable\" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -qq
  apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin
  systemctl enable docker
  systemctl start docker

  # Warten bis Docker-Daemon bereit ist
  for i in \$(seq 1 30); do
    docker info &>/dev/null 2>&1 && break
    sleep 2
  done
  docker info &>/dev/null || { echo 'Docker daemon not ready'; exit 1; }

  # System-User für Service
  id osb &>/dev/null || useradd --system --home-dir /opt/opensourcebackup --shell /usr/sbin/nologin osb

  # Verzeichnisse
  mkdir -p /opt/opensourcebackup /var/lib/opensourcebackup/certs /etc/opensourcebackup

  # DB-Passwort generieren (einmalig)
  CREDS_DIR=/etc/opensourcebackup
  if [ ! -f \"\$CREDS_DIR/.db_password\" ]; then
    openssl rand -hex 32 > \"\$CREDS_DIR/.db_password\"
    chmod 600 \"\$CREDS_DIR/.db_password\"
  fi
  DB_PASS=\$(cat \"\$CREDS_DIR/.db_password\")

  # docker-compose.yml mit Named Volumes (keine Permissions-Probleme)
  cat > /opt/opensourcebackup/docker-compose.yml << DCEOF
services:
  postgres:
    image: postgres:16-alpine
    restart: always
    environment:
      POSTGRES_USER: opensourcebackup
      POSTGRES_PASSWORD: \${DB_PASS}
      POSTGRES_DB: opensourcebackup
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - '127.0.0.1:5432:5432'
    healthcheck:
      test: [\"CMD-SHELL\", \"pg_isready -U opensourcebackup\"]
      interval: 5s
      timeout: 5s
      retries: 20
  redis:
    image: redis:7-alpine
    restart: always
    ports:
      - '127.0.0.1:6379:6379'
    volumes:
      - redisdata:/data
volumes:
  pgdata:
  redisdata:
DCEOF

  # .env Datei für docker-compose (damit Variable verfügbar ist)
  echo \"DB_PASS=\${DB_PASS}\" > /opt/opensourcebackup/.env
  chmod 600 /opt/opensourcebackup/.env

  # Container starten
  docker compose -f /opt/opensourcebackup/docker-compose.yml up -d
  echo 'DOCKER_OK'
"
ok "Docker + PostgreSQL + Redis gestartet"

# PostgreSQL warten — von außen prüfen
info "Warte auf PostgreSQL..."
for i in {1..50}; do
  pct exec "$CT_ID" -- docker exec opensourcebackup-postgres-1 \
    pg_isready -U opensourcebackup &>/dev/null 2>&1 && break || { echo -n "."; sleep 3; }
done
echo ""

# Prüfen ob PostgreSQL wirklich läuft
if ! pct exec "$CT_ID" -- docker exec opensourcebackup-postgres-1 pg_isready -U opensourcebackup &>/dev/null 2>&1; then
  warn "PostgreSQL antwortet nicht — Container-Logs:"
  pct exec "$CT_ID" -- docker logs opensourcebackup-postgres-1 2>&1 | tail -10
  die "PostgreSQL konnte nicht gestartet werden"
fi
ok "PostgreSQL bereit"

# ── Server-Binary + Web UI ────────────────────────────────────────────────────

title "Schritt 6/7: Control Plane + Web UI bauen"

pct exec "$CT_ID" -- bash -c "
  export LANG=C.UTF-8 LC_ALL=C.UTF-8 DEBIAN_FRONTEND=noninteractive
  set -e

  # Go 1.25 installieren
  if ! /usr/local/go/bin/go version 2>/dev/null | grep -qE '1\.(2[2-9]|[3-9][0-9])'; then
    echo 'Installiere Go 1.25...'
    curl -fsSL https://go.dev/dl/go1.25.0.linux-amd64.tar.gz | tar -C /usr/local -xz
  fi
  export PATH=\"/usr/local/go/bin:\$PATH\"

  # Source klonen
  rm -rf /tmp/osb-src
  git clone --depth=1 https://github.com/cerberus8484/opensourcebackup.git /tmp/osb-src

  # Server bauen
  cd /tmp/osb-src
  CGO_ENABLED=0 go build -ldflags '-s -w' -o /opt/opensourcebackup/opensourcebackup-server ./cmd/control-plane/
  chown osb:osb /opt/opensourcebackup/opensourcebackup-server

  # Node.js + Web UI
  if ! command -v node &>/dev/null; then
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - >/dev/null 2>&1
    apt-get install -y -qq nodejs
  fi
  mkdir -p /opt/opensourcebackup/web-ui
  cd /tmp/osb-src/web
  npm install --silent
  npm run build
  cp -r dist/. /opt/opensourcebackup/web-ui/

  # Migrations-Tool
  if ! command -v migrate &>/dev/null; then
    curl -fsSL https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-amd64.tar.gz \
      -o /tmp/migrate.tar.gz
    tar -xzf /tmp/migrate.tar.gz -C /tmp
    mv /tmp/migrate /usr/local/bin/migrate 2>/dev/null || \
      mv /tmp/migrate.linux-amd64 /usr/local/bin/migrate 2>/dev/null || true
    chmod +x /usr/local/bin/migrate
    rm -f /tmp/migrate.tar.gz
  fi

  # Migrationen
  DB_PASSWORD=\$(cat /etc/opensourcebackup/.db_password)
  DB_URL=\"postgres://opensourcebackup:\${DB_PASSWORD}@127.0.0.1:5432/opensourcebackup?sslmode=disable\"
  migrate -path /tmp/osb-src/migrations -database \"\$DB_URL\" up

  # Migrations-Ordner kopieren
  cp -r /tmp/osb-src/migrations /opt/opensourcebackup/

  rm -rf /tmp/osb-src

  echo 'BUILD_OK'
" 2>&1 | grep -E "^(✓|→|⚠|Installie|Build|Go|Node|Migrat|error|Error|FAIL|BUILD_OK)" || true
ok "Control Plane + Web UI gebaut und Migrationen ausgeführt"

# ── Konfiguration + Service ───────────────────────────────────────────────────

title "Schritt 7/7: Konfiguration + Systemd-Dienst"

pct exec "$CT_ID" -- bash -c "
  set -e
  export LANG=C.UTF-8 LC_ALL=C.UTF-8

  DB_PASSWORD=\$(cat /etc/opensourcebackup/.db_password)
  LOCAL_IP=\$(hostname -I | awk '{print \$1}')

  # server.env erstellen
  cat > /etc/opensourcebackup/server.env << ENVEOF
DATABASE_URL=postgres://opensourcebackup:\${DB_PASSWORD}@127.0.0.1:5432/opensourcebackup?sslmode=disable
LISTEN_ADDR=:${OSB_PORT}
CORS_ORIGIN=*
WEB_UI_DIR=/opt/opensourcebackup/web-ui
ADMIN_EMAIL=${ADMIN_EMAIL}
ADMIN_PASSWORD=${ADMIN_PASSWORD}
ENVEOF
  chmod 600 /etc/opensourcebackup/server.env

  # Systemd-Service
  cat > /etc/systemd/system/opensourcebackup.service << SVCEOF
[Unit]
Description=OpenSourceBackup Control Plane
Documentation=https://github.com/cerberus8484/opensourcebackup
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=osb
WorkingDirectory=/opt/opensourcebackup
EnvironmentFile=/etc/opensourcebackup/server.env
ExecStart=/opt/opensourcebackup/opensourcebackup-server
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=opensourcebackup

[Install]
WantedBy=multi-user.target
SVCEOF

  chown -R osb:osb /opt/opensourcebackup /var/lib/opensourcebackup
  systemctl daemon-reload
  systemctl enable opensourcebackup
  systemctl start opensourcebackup
  sleep 5

  # Bootstrap-Admin warten
  for i in {1..15}; do
    STATUS=\$(curl -sf http://127.0.0.1:${OSB_PORT}/health 2>/dev/null | grep -c 'ok' || true)
    [ \"\$STATUS\" -ge 1 ] && break || sleep 3
  done

  # Credentials-Datei
  cat > /root/opensourcebackup-credentials.txt << CREDEOF
=================================================================
  OpenSourceBackup — Zugangsdaten
  Installiert: \$(date)
  DIESE DATEI SICHER AUFBEWAHREN UND DANN LÖSCHEN!
=================================================================

  Web Dashboard : http://\${LOCAL_IP}:${OSB_PORT}/ui/
  Admin E-Mail  : ${ADMIN_EMAIL}
  Admin Passwort: ${ADMIN_PASSWORD}

  Datenbank
  Host    : 127.0.0.1:5432
  User    : opensourcebackup
  Passwort: \${DB_PASSWORD}

  Dienst verwalten:
  Status  : systemctl status opensourcebackup
  Logs    : journalctl -u opensourcebackup -f
  Neustart: systemctl restart opensourcebackup

=================================================================
CREDEOF
  chmod 600 /root/opensourcebackup-credentials.txt

  echo 'SERVICE_OK'
"
ok "Dienst konfiguriert und gestartet"

# ── IP + DNS ermitteln ────────────────────────────────────────────────────────

CT_IP_ACTUAL=$(pct exec "$CT_ID" -- hostname -I 2>/dev/null | awk '{print $1}' || echo "unbekannt")
CT_HOSTNAME_ACTUAL=$(pct exec "$CT_ID" -- hostname 2>/dev/null || echo "$CT_HOSTNAME")

# Health-Check
if curl -sf "http://${CT_IP_ACTUAL}:${OSB_PORT}/health" &>/dev/null; then
  HEALTH="${GREEN}✓ erreichbar${NC}"
else
  HEALTH="${YELLOW}⚠ noch nicht erreichbar (kurz warten)${NC}"
fi

# ── Abschluss-Ausgabe ─────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BOLD}${GREEN}║   ✓  OpenSourceBackup erfolgreich installiert!                ║${NC}"
echo -e "${BOLD}${GREEN}╠═══════════════════════════════════════════════════════════════╣${NC}"
echo -e "${BOLD}${GREEN}║                                                               ║${NC}"
echo -e "${BOLD}${GREEN}║   🌐  WEB DASHBOARD                                           ║${NC}"
echo -e "${BOLD}${GREEN}║       ${CYAN}http://${CT_IP_ACTUAL}:${OSB_PORT}/ui/${NC}"
echo -e "${BOLD}${GREEN}║                                                               ║${NC}"
echo -e "${BOLD}${GREEN}║   🔑  ZUGANGSDATEN                                            ║${NC}"
echo -e "${BOLD}${GREEN}║       E-Mail  : ${CYAN}${ADMIN_EMAIL}${NC}"
echo -e "${BOLD}${GREEN}║       Passwort: ${CYAN}${ADMIN_PASSWORD}${NC}"
echo -e "${BOLD}${GREEN}║                                                               ║${NC}"
echo -e "${BOLD}${GREEN}║   🖥  CONTAINER                                               ║${NC}"
echo -e "${BOLD}${GREEN}║       ID       : ${CYAN}${CT_ID}${NC}"
echo -e "${BOLD}${GREEN}║       Hostname : ${CYAN}${CT_HOSTNAME_ACTUAL}${NC}"
echo -e "${BOLD}${GREEN}║       IP       : ${CYAN}${CT_IP_ACTUAL}${NC}"
echo -e "${BOLD}${GREEN}║       Status   : $(echo -e "$HEALTH")"
echo -e "${BOLD}${GREEN}║                                                               ║${NC}"
echo -e "${BOLD}${GREEN}║   📋  Container verwalten (auf Proxmox-Host):                 ║${NC}"
echo -e "${BOLD}${GREEN}║       Shell  : ${CYAN}pct enter ${CT_ID}${NC}"
echo -e "${BOLD}${GREEN}║       Stop   : ${CYAN}pct stop ${CT_ID}${NC}"
echo -e "${BOLD}${GREEN}║       Logs   : ${CYAN}pct exec ${CT_ID} -- journalctl -u opensourcebackup -f${NC}"
echo -e "${BOLD}${GREEN}║                                                               ║${NC}"
echo -e "${BOLD}${GREEN}║   📄  Credentials gespeichert unter:                          ║${NC}"
echo -e "${BOLD}${GREEN}║       ${CYAN}pct exec ${CT_ID} -- cat /root/opensourcebackup-credentials.txt${NC}"
echo -e "${BOLD}${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}  ⚠  Passwort sofort ändern unter:${NC}"
echo -e "     ${CYAN}http://${CT_IP_ACTUAL}:${OSB_PORT}/ui/ → Users${NC}"
echo ""
