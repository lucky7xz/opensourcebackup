#!/usr/bin/env bash
# ==============================================================================
# OpenSourceBackup — Proxmox LXC Auto-Installer
#
# Läuft auf dem PROXMOX HOST (nicht im Container).
# Erstellt automatisch einen Debian 12 LXC Container und installiert
# OpenSourceBackup darin.
#
# Usage (auf dem Proxmox-Host als root):
#   bash <(curl -fsSL https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-proxmox.sh)
#
# Optionen (Umgebungsvariablen):
#   CTID=201              Container-ID (Standard: nächste freie ab 200)
#   CT_HOSTNAME=osb       Hostname des Containers
#   CT_CORES=2            CPU-Kerne
#   CT_MEMORY=2048        RAM in MB
#   CT_DISK=20            Disk in GB
#   CT_STORAGE=local-lvm  Proxmox-Storage für den Container
#   CT_BRIDGE=vmbr0       Netzwerk-Bridge
#   CT_IP=dhcp            IP-Adresse (dhcp oder 192.168.1.100/24)
#   CT_GW=                Gateway (nur bei statischer IP)
#   OSB_PORT=8080         Port für das Web-Dashboard
# ==============================================================================

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✓${NC} $*"; }
info() { echo -e "${CYAN}→${NC} $*"; }
warn() { echo -e "${YELLOW}⚠${NC} $*"; }
die()  { echo -e "${RED}✗ FEHLER:${NC} $*" >&2; exit 1; }
step() { echo -e "\n${CYAN}━━━ $* ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║   OpenSourceBackup — Proxmox LXC Auto-Installer  ║${NC}"
echo -e "${CYAN}║   Creating backups is easy.                       ║${NC}"
echo -e "${CYAN}║   Proving recoverability is the difference.       ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════╝${NC}"
echo ""

# ── Prüfungen ─────────────────────────────────────────────────────────────────

[[ $EUID -eq 0 ]] || die "Als root ausführen!"
command -v pct    &>/dev/null || die "pct nicht gefunden — läuft dieses Script auf einem Proxmox-Host?"
command -v pveam  &>/dev/null || die "pveam nicht gefunden — Proxmox-Host erforderlich"

# ── Konfiguration ─────────────────────────────────────────────────────────────

# Nächste freie Container-ID ab 200 ermitteln (falls nicht gesetzt)
if [ -z "${CTID:-}" ]; then
  CTID=200
  while pct status "$CTID" &>/dev/null 2>&1; do
    CTID=$((CTID + 1))
  done
  info "Verwende Container-ID: $CTID (erste freie ab 200)"
fi

CT_HOSTNAME="${CT_HOSTNAME:-opensourcebackup}"
CT_CORES="${CT_CORES:-2}"
CT_MEMORY="${CT_MEMORY:-2048}"
CT_DISK="${CT_DISK:-20}"
CT_BRIDGE="${CT_BRIDGE:-vmbr0}"
CT_IP="${CT_IP:-dhcp}"
CT_GW="${CT_GW:-}"
OSB_PORT="${OSB_PORT:-8080}"

# Storage ermitteln: bevorzuge local-lvm, sonst erstes verfügbares
if [ -z "${CT_STORAGE:-}" ]; then
  if pvesm status | grep -q "local-lvm"; then
    CT_STORAGE="local-lvm"
  elif pvesm status | grep -q "local"; then
    CT_STORAGE="local"
  else
    CT_STORAGE=$(pvesm status | awk 'NR>1 {print $1}' | head -1)
  fi
  info "Verwende Storage: $CT_STORAGE"
fi

echo ""
echo -e "  ${YELLOW}Container-Konfiguration:${NC}"
echo -e "  ID:        ${CYAN}${CTID}${NC}"
echo -e "  Hostname:  ${CYAN}${CT_HOSTNAME}${NC}"
echo -e "  Cores:     ${CYAN}${CT_CORES}${NC}"
echo -e "  RAM:       ${CYAN}${CT_MEMORY} MB${NC}"
echo -e "  Disk:      ${CYAN}${CT_DISK} GB${NC}"
echo -e "  Storage:   ${CYAN}${CT_STORAGE}${NC}"
echo -e "  Bridge:    ${CYAN}${CT_BRIDGE}${NC}"
echo -e "  IP:        ${CYAN}${CT_IP}${NC}"
echo -e "  OSB Port:  ${CYAN}${OSB_PORT}${NC}"
echo ""

# ── Step 1: Debian 12 Template ────────────────────────────────────────────────

step "Step 1/5: Debian 12 LXC Template"

info "Template-Liste aktualisieren..."
pveam update 2>/dev/null | tail -1 || true

# Verfügbares Debian-12-Template suchen
TEMPLATE_NAME=$(pveam available 2>/dev/null \
  | grep "debian-12-standard" \
  | awk '{print $2}' \
  | sort -V | tail -1 || true)

if [ -z "$TEMPLATE_NAME" ]; then
  # Fallback: bereits heruntergeladenes Template suchen
  TEMPLATE_NAME=$(pveam list local 2>/dev/null \
    | grep "debian-12-standard" \
    | awk '{print $1}' \
    | sort -V | tail -1 || true)
fi

[ -n "$TEMPLATE_NAME" ] || die "Kein Debian 12 Template gefunden. Manuell herunterladen:\n  pveam download local debian-12-standard_12.7-1_amd64.tar.zst"

# Template herunterladen falls noch nicht lokal
LOCAL_TEMPLATE=$(pveam list local 2>/dev/null | grep "debian-12-standard" | awk '{print $1}' | tail -1 || true)
if [ -z "$LOCAL_TEMPLATE" ]; then
  info "Lade Template herunter: $TEMPLATE_NAME ..."
  pveam download local "$TEMPLATE_NAME"
  ok "Template heruntergeladen"
  LOCAL_TEMPLATE=$(pveam list local | grep "debian-12-standard" | awk '{print $1}' | tail -1)
else
  ok "Template bereits lokal verfügbar: $LOCAL_TEMPLATE"
fi

# ── Step 2: LXC Container erstellen ──────────────────────────────────────────

step "Step 2/5: LXC Container erstellen"

# Prüfen ob Container schon existiert
if pct status "$CTID" &>/dev/null 2>&1; then
  warn "Container $CTID existiert bereits!"
  read -r -p "Container $CTID löschen und neu erstellen? [y/N] " CONFIRM
  if [[ "${CONFIRM:-n}" =~ ^[Yy]$ ]]; then
    pct stop "$CTID" 2>/dev/null || true
    sleep 2
    pct destroy "$CTID" --purge
    ok "Alter Container gelöscht"
  else
    die "Abgebrochen. Setze CTID=<andere ID> um eine andere ID zu verwenden."
  fi
fi

# Netzwerk-Parameter bauen
NET_CONFIG="name=eth0,bridge=${CT_BRIDGE}"
if [ "$CT_IP" = "dhcp" ]; then
  NET_CONFIG="${NET_CONFIG},ip=dhcp"
else
  NET_CONFIG="${NET_CONFIG},ip=${CT_IP}"
  [ -n "$CT_GW" ] && NET_CONFIG="${NET_CONFIG},gw=${CT_GW}"
fi

info "Erstelle LXC Container $CTID ($CT_HOSTNAME)..."
pct create "$CTID" "$LOCAL_TEMPLATE" \
  --hostname   "$CT_HOSTNAME" \
  --cores      "$CT_CORES" \
  --memory     "$CT_MEMORY" \
  --rootfs     "${CT_STORAGE}:${CT_DISK}" \
  --net0       "$NET_CONFIG" \
  --features   "nesting=1" \
  --unprivileged 1 \
  --ostype     debian \
  --start      0

ok "Container $CTID erstellt"

# ── Step 3: Container starten und bereit machen ───────────────────────────────

step "Step 3/5: Container starten"

info "Starte Container $CTID..."
pct start "$CTID"

# Warten bis Container gestartet ist
info "Warte bis Container bereit ist..."
for i in {1..30}; do
  if pct exec "$CTID" -- echo "ready" &>/dev/null 2>&1; then
    break
  fi
  echo -n "."
  sleep 2
done
echo ""
ok "Container läuft"

# Netzwerk-Verbindung prüfen
info "Prüfe Internet-Verbindung im Container..."
for i in {1..15}; do
  if pct exec "$CTID" -- curl -fsSL --max-time 5 https://github.com &>/dev/null 2>&1; then
    ok "Internet-Verbindung OK"
    break
  fi
  echo -n "."
  sleep 3
done
echo ""

# Basis-Pakete installieren
info "Installiere Basis-Pakete..."
pct exec "$CTID" -- bash -c "apt-get update -qq && apt-get install -y -qq curl git openssl ca-certificates" 2>&1 | tail -3
ok "Basis-Pakete installiert"

# ── Step 4: OpenSourceBackup im Container installieren ───────────────────────

step "Step 4/5: OpenSourceBackup installieren"

info "Starte install-server.sh im Container..."
pct exec "$CTID" -- bash -c "
  export OSB_PORT='${OSB_PORT}'
  curl -fsSL https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh | bash
"

ok "OpenSourceBackup installiert"

# ── Step 5: IP-Adresse und Ergebnis ──────────────────────────────────────────

step "Step 5/5: Fertig"

# IP-Adresse des Containers ermitteln
sleep 3
CT_ACTUAL_IP=$(pct exec "$CTID" -- hostname -I 2>/dev/null | awk '{print $1}' || echo "unbekannt")

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║       ✓  OpenSourceBackup LXC Container bereit!                 ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║                                                                  ║${NC}"
echo -e "${GREEN}║   📦  CONTAINER                                                  ║${NC}"
echo -e "${GREEN}║       ID:       ${CYAN}${CTID}${GREEN}$(printf '%*s' $((52 - ${#CTID})) '')║${NC}"
echo -e "${GREEN}║       Hostname: ${CYAN}${CT_HOSTNAME}${GREEN}$(printf '%*s' $((52 - ${#CT_HOSTNAME})) '')║${NC}"
echo -e "${GREEN}║                                                                  ║${NC}"
echo -e "${GREEN}║   🌐  WEB DASHBOARD                                              ║${NC}"
echo -e "${GREEN}║       ${CYAN}http://${CT_ACTUAL_IP}:${OSB_PORT}/ui/${GREEN}$(printf '%*s' $((47 - ${#CT_ACTUAL_IP} - ${#OSB_PORT})) '')║${NC}"
echo -e "${GREEN}║                                                                  ║${NC}"
echo -e "${GREEN}║   🔑  LOGIN                                                      ║${NC}"
echo -e "${GREEN}║       ${YELLOW}(noch nicht erforderlich — kommt in v2)${GREEN}               ║${NC}"
echo -e "${GREEN}║                                                                  ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║   Container-Verwaltung (auf Proxmox-Host):                       ║${NC}"
echo -e "${GREEN}║   Shell:     pct enter ${CTID}${GREEN}$(printf '%*s' $((41 - ${#CTID})) '')║${NC}"
echo -e "${GREEN}║   Stop:      pct stop ${CTID}${GREEN}$(printf '%*s' $((42 - ${#CTID})) '')║${NC}"
echo -e "${GREEN}║   Start:     pct start ${CTID}${GREEN}$(printf '%*s' $((41 - ${#CTID})) '')║${NC}"
echo -e "${GREEN}║   Logs OSB:  pct exec ${CTID} -- journalctl -u opensourcebackup -f${NC}  ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${YELLOW}Agent auf einem System installieren:${NC}"
echo -e "  Im Dashboard: Agents → + Enroll Agent → Wizard folgen"
echo ""
echo -e "  ${YELLOW}Zugangsdaten im Container:${NC}"
echo -e "  ${CYAN}pct exec ${CTID} -- cat /root/opensourcebackup-credentials.txt${NC}"
echo ""
