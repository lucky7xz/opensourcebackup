#!/usr/bin/env bash
# ==============================================================================
# OpenSourceBackup — Migrate from Proxmox Host to LXC Container
#
# Läuft auf dem PROXMOX HOST als root.
# Migriert eine laufende OpenSourceBackup-Installation vom Host in einen
# neuen Debian-12-LXC-Container.
#
# Usage:
#   bash migrate-to-lxc.sh
#
# Was das Script macht:
#   1. Konfiguration prüfen
#   2. PostgreSQL-Dump vom laufenden System erstellen
#   3. Neuen LXC-Container erstellen
#   4. OpenSourceBackup im Container installieren
#   5. Daten importieren
#   6. Konfiguration übertragen
#   7. Container testen
#   8. Optional: Host-Service stoppen
#
# Voraussetzungen:
#   - Proxmox VE 7 oder 8
#   - OpenSourceBackup läuft auf dem Host (systemd-Dienst + Docker)
#   - Internetverbindung für apt/Docker-Installation im Container
# ==============================================================================

set -euo pipefail

# ╔══════════════════════════════════════════════════════════════════════════╗
# ║                    KONFIGURATION — HIER ANPASSEN                        ║
# ╚══════════════════════════════════════════════════════════════════════════╝

# Container-ID für den neuen LXC-Container
# Prüfe mit: pct list — wähle eine freie ID
CT_ID="${CT_ID:-200}"

# Hostname des neuen Containers
CT_HOSTNAME="opensourcebackup"

# Ressourcen des Containers
CT_MEMORY=2048    # MB RAM
CT_CORES=2        # CPU-Kerne
CT_DISK=20        # GB Disk

# Proxmox-Storage für den Container-Rootfs
# Prüfe verfügbare Storages mit: pvesm status
CT_STORAGE="${CT_STORAGE:-local-lvm}"

# Netzwerk-Bridge
CT_BRIDGE="vmbr0"

# IP-Konfiguration
# "dhcp" = automatisch per DHCP (Standard)
# Oder statisch: "192.168.1.50/24" + CT_GW="192.168.1.1"
CT_IP="dhcp"
CT_GW=""

# Port auf dem der Control Plane lauscht
OSB_PORT="${OSB_PORT:-8080}"

# ── Bestehende Installation (Host) ───────────────────────────────────────────

# Pfad zur server.env auf dem Host
HOST_ENV_FILE="/etc/opensourcebackup/server.env"

# Pfad zur DB-Passwort-Datei auf dem Host
HOST_DB_PASS_FILE="/etc/opensourcebackup/.db_password"

# Docker-Container-Name der PostgreSQL-Instanz auf dem Host
HOST_PG_CONTAINER="opensourcebackup-postgres-1"

# PostgreSQL-Benutzer und Datenbankname auf dem Host
HOST_PG_USER="opensourcebackup"
HOST_PG_DB="opensourcebackup"

# Temporäre Dateien für Migration
DUMP_FILE="/tmp/osb-migration-dump.sql"
ENV_BACKUP="/tmp/osb-migration-server.env"
PASS_BACKUP="/tmp/osb-migration-db.password"

# ╔══════════════════════════════════════════════════════════════════════════╗
# ║                      SCRIPT — NICHT ÄNDERN                              ║
# ╚══════════════════════════════════════════════════════════════════════════╝

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✓${NC} $*"; }
info() { echo -e "${CYAN}→${NC} $*"; }
warn() { echo -e "${YELLOW}⚠${NC} $*"; }
die()  { echo -e "${RED}✗ FEHLER:${NC} $*" >&2; exit 1; }
step() { echo -e "\n${CYAN}━━━ $* ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║  OpenSourceBackup — Migration zu LXC-Container   ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════╝${NC}"
echo ""

# ── Voraussetzungen prüfen ────────────────────────────────────────────────────

step "Schritt 1/8: Voraussetzungen prüfen"

[[ $EUID -eq 0 ]] || die "Als root ausführen!"
command -v pct    &>/dev/null || die "pct nicht gefunden — kein Proxmox-Host?"
command -v docker &>/dev/null || die "Docker nicht gefunden — läuft OpenSourceBackup auf diesem Host?"

# Bestehende Installation prüfen
[[ -f "$HOST_ENV_FILE" ]]     || die "server.env nicht gefunden: $HOST_ENV_FILE"
[[ -f "$HOST_DB_PASS_FILE" ]] || die "DB-Passwort nicht gefunden: $HOST_DB_PASS_FILE"

if ! docker ps --format '{{.Names}}' | grep -q "$HOST_PG_CONTAINER"; then
  die "PostgreSQL-Container '$HOST_PG_CONTAINER' läuft nicht. Bitte zuerst starten:\n  systemctl start opensourcebackup"
fi

# Container-ID prüfen
if pct status "$CT_ID" &>/dev/null 2>&1; then
  warn "Container $CT_ID existiert bereits!"
  read -r -p "Container $CT_ID löschen und neu erstellen? [y/N] " CONFIRM
  if [[ "${CONFIRM:-n}" =~ ^[Yy]$ ]]; then
    pct stop "$CT_ID" 2>/dev/null || true
    sleep 2
    pct destroy "$CT_ID" --purge
    ok "Alter Container gelöscht"
  else
    die "Abgebrochen. Setze CT_ID=<andere-ID> in diesem Script."
  fi
fi

ok "Voraussetzungen erfüllt"
info "Container-ID:  $CT_ID"
info "Hostname:      $CT_HOSTNAME"
info "Storage:       $CT_STORAGE"
info "RAM:           ${CT_MEMORY} MB"
info "Disk:          ${CT_DISK} GB"

# ── Daten sichern ─────────────────────────────────────────────────────────────

step "Schritt 2/8: PostgreSQL-Dump erstellen"

DB_PASS=$(cat "$HOST_DB_PASS_FILE")

info "Erstelle Datenbankdump…"
docker exec "$HOST_PG_CONTAINER" \
  pg_dump -U "$HOST_PG_USER" "$HOST_PG_DB" > "$DUMP_FILE"

LINES=$(wc -l < "$DUMP_FILE")
ok "Dump erstellt: $DUMP_FILE ($LINES Zeilen)"

# Konfiguration sichern
cp "$HOST_ENV_FILE"      "$ENV_BACKUP"
cp "$HOST_DB_PASS_FILE"  "$PASS_BACKUP"
ok "Konfiguration gesichert"

# ── LXC-Container erstellen ───────────────────────────────────────────────────

step "Schritt 3/8: LXC-Container erstellen"

info "Template-Liste aktualisieren…"
pveam update 2>/dev/null | tail -1 || true

# Debian-12-Template suchen oder herunterladen
LOCAL_TPL=$(pveam list local 2>/dev/null | grep "debian-12-standard" | awk '{print $1}' | tail -1 || true)
if [ -z "$LOCAL_TPL" ]; then
  AVAIL_TPL=$(pveam available 2>/dev/null | grep "debian-12-standard" | awk '{print $2}' | sort -V | tail -1 || true)
  [ -n "$AVAIL_TPL" ] || die "Kein Debian-12-Template verfügbar. Bitte manuell herunterladen:\n  pveam download local debian-12-standard_12.7-1_amd64.tar.zst"
  info "Lade Template herunter: $AVAIL_TPL"
  pveam download local "$AVAIL_TPL"
  LOCAL_TPL=$(pveam list local | grep "debian-12-standard" | awk '{print $1}' | tail -1)
fi
ok "Template: $LOCAL_TPL"

# Netzwerk-Parameter
NET_CONFIG="name=eth0,bridge=${CT_BRIDGE}"
if [ "$CT_IP" = "dhcp" ]; then
  NET_CONFIG="${NET_CONFIG},ip=dhcp"
else
  NET_CONFIG="${NET_CONFIG},ip=${CT_IP}"
  [ -n "$CT_GW" ] && NET_CONFIG="${NET_CONFIG},gw=${CT_GW}"
fi

info "Erstelle Container $CT_ID ($CT_HOSTNAME)…"
pct create "$CT_ID" "$LOCAL_TPL" \
  --hostname   "$CT_HOSTNAME" \
  --cores      "$CT_CORES" \
  --memory     "$CT_MEMORY" \
  --rootfs     "${CT_STORAGE}:${CT_DISK}" \
  --net0       "$NET_CONFIG" \
  --features   "nesting=1" \
  --unprivileged 1 \
  --ostype     debian \
  --start      0

ok "Container erstellt"

info "Starte Container…"
pct start "$CT_ID"
sleep 8

# Warten bis Container erreichbar
for i in {1..20}; do
  if pct exec "$CT_ID" -- echo "ok" &>/dev/null 2>&1; then break; fi
  echo -n "."; sleep 2
done
echo ""
ok "Container läuft"

# ── Basispakete installieren ──────────────────────────────────────────────────

step "Schritt 4/8: Basispakete im Container installieren"

pct exec "$CT_ID" -- bash -c "
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get install -y -qq curl git ca-certificates openssl
" 2>&1 | tail -3
ok "Basispakete installiert"

# ── OpenSourceBackup installieren ────────────────────────────────────────────

step "Schritt 5/8: OpenSourceBackup im Container installieren"

info "Führe install-server.sh im Container aus (dauert 5–10 Minuten)…"

# WICHTIG: Das Install-Script generiert ein NEUES DB-Passwort.
# Wir überschreiben es danach mit dem alten Passwort aus dem Backup.
pct exec "$CT_ID" -- bash -c "
  OSB_PORT='${OSB_PORT}' \
  curl -fsSL https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh \
  | bash
" || die "Installation im Container fehlgeschlagen"

ok "OpenSourceBackup installiert"

# ── Daten migrieren ───────────────────────────────────────────────────────────

step "Schritt 6/8: Daten in Container importieren"

# Dump in Container kopieren
info "Kopiere Datenbankdump in Container…"
pct push "$CT_ID" "$DUMP_FILE" /tmp/osb-migration-dump.sql
pct push "$CT_ID" "$PASS_BACKUP" /tmp/osb-migration-db.password

# Altes DB-Passwort übernehmen
info "Übernehme bestehendes DB-Passwort…"
pct exec "$CT_ID" -- bash -c "
  # Altes Passwort als neues setzen
  OLD_PASS=\$(cat /tmp/osb-migration-db.password)
  NEW_PASS=\$(cat /etc/opensourcebackup/.db_password)

  if [ \"\$OLD_PASS\" != \"\$NEW_PASS\" ]; then
    # Passwort in PostgreSQL ändern
    docker exec opensourcebackup-postgres-1 \
      psql -U opensourcebackup -c \"ALTER USER opensourcebackup PASSWORD '\$OLD_PASS';\"

    # Passwort-Datei und server.env aktualisieren
    echo \"\$OLD_PASS\" > /etc/opensourcebackup/.db_password
    sed -i \"s|:${NEW_PASS}@|:\${OLD_PASS}@|g\" /etc/opensourcebackup/server.env
    echo 'Passwort übernommen'
  else
    echo 'Passwort ist identisch — keine Änderung nötig'
  fi
"

# Warten bis PostgreSQL bereit
info "Warte auf PostgreSQL…"
for i in {1..20}; do
  if pct exec "$CT_ID" -- docker exec opensourcebackup-postgres-1 pg_isready -U opensourcebackup &>/dev/null 2>&1; then
    break
  fi
  echo -n "."; sleep 3
done
echo ""

# Dump einspielen (nach Migrationen!)
info "Importiere Daten…"
pct exec "$CT_ID" -- bash -c "
  # Bestehende Daten löschen und neu einspielen
  docker exec -i opensourcebackup-postgres-1 \
    psql -U opensourcebackup opensourcebackup < /tmp/osb-migration-dump.sql 2>&1 | tail -5
" || warn "Dump-Import hatte Warnungen — prüfe manuell"

ok "Daten importiert"

# ── Container testen ──────────────────────────────────────────────────────────

step "Schritt 7/8: Container testen"

sleep 5

CT_IP_ACTUAL=$(pct exec "$CT_ID" -- hostname -I 2>/dev/null | awk '{print $1}' || echo "unbekannt")

# Health-Check
if curl -sf "http://${CT_IP_ACTUAL}:${OSB_PORT}/health" &>/dev/null; then
  ok "Health-Check erfolgreich: http://${CT_IP_ACTUAL}:${OSB_PORT}/health"
else
  warn "Health-Check fehlgeschlagen — prüfe Logs im Container:"
  echo "  pct exec $CT_ID -- journalctl -u opensourcebackup -n 20 --no-pager"
fi

# Datenbankinhalt prüfen
SYSTEM_COUNT=$(pct exec "$CT_ID" -- docker exec opensourcebackup-postgres-1 \
  psql -U opensourcebackup opensourcebackup -t -c "SELECT COUNT(*) FROM systems;" 2>/dev/null | tr -d ' \n' || echo "?")
JOB_COUNT=$(pct exec "$CT_ID" -- docker exec opensourcebackup-postgres-1 \
  psql -U opensourcebackup opensourcebackup -t -c "SELECT COUNT(*) FROM backup_jobs;" 2>/dev/null | tr -d ' \n' || echo "?")

ok "Systeme in Container-DB: $SYSTEM_COUNT"
ok "Jobs in Container-DB:    $JOB_COUNT"

# ── Host-Service deaktivieren ─────────────────────────────────────────────────

step "Schritt 8/8: Host-Service"

echo ""
echo -e "${YELLOW}Der Container läuft und wurde getestet.${NC}"
echo ""
echo -e "Container-Dashboard: ${CYAN}http://${CT_IP_ACTUAL}:${OSB_PORT}/ui/${NC}"
echo ""
echo -e "${YELLOW}Soll der OpenSourceBackup-Service auf dem HOST jetzt gestoppt werden?${NC}"
read -r -p "Host-Service stoppen und deaktivieren? [y/N] " STOP_HOST

if [[ "${STOP_HOST:-n}" =~ ^[Yy]$ ]]; then
  systemctl stop opensourcebackup
  systemctl disable opensourcebackup
  docker compose -f /opt/opensourcebackup/docker-compose.yml down 2>/dev/null || true
  ok "Host-Service gestoppt und deaktiviert"
  warn "Die Dateien auf dem Host bleiben erhalten unter /opt/opensourcebackup/"
  warn "Zum endgültigen Entfernen: rm -rf /opt/opensourcebackup /etc/opensourcebackup"
else
  warn "Host-Service läuft noch. Beide Instanzen laufen parallel!"
  warn "Stoppe den Host-Service manuell wenn der Container stabil läuft:"
  echo "  systemctl stop opensourcebackup && systemctl disable opensourcebackup"
fi

# ── Zusammenfassung ────────────────────────────────────────────────────────────

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   ✓  Migration abgeschlossen!                                ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║                                                              ║${NC}"
echo -e "${GREEN}║   Container-ID:   ${CYAN}${CT_ID}${GREEN}$(printf '%*s' $((46-${#CT_ID})) '')║${NC}"
echo -e "${GREEN}║   Hostname:       ${CYAN}${CT_HOSTNAME}${GREEN}$(printf '%*s' $((46-${#CT_HOSTNAME})) '')║${NC}"
echo -e "${GREEN}║   IP-Adresse:     ${CYAN}${CT_IP_ACTUAL}${GREEN}$(printf '%*s' $((46-${#CT_IP_ACTUAL})) '')║${NC}"
echo -e "${GREEN}║   Dashboard:      ${CYAN}http://${CT_IP_ACTUAL}:${OSB_PORT}/ui/${GREEN}$(printf '%*s' $((39-${#CT_IP_ACTUAL}-${#OSB_PORT})) '')║${NC}"
echo -e "${GREEN}║                                                              ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║   Container-Verwaltung:                                      ║${NC}"
echo -e "${GREEN}║   Shell:     pct enter ${CT_ID}$(printf '%*s' $((38-${#CT_ID})) '')║${NC}"
echo -e "${GREEN}║   Stop:      pct stop ${CT_ID}$(printf '%*s' $((39-${#CT_ID})) '')║${NC}"
echo -e "${GREEN}║   Logs:      pct exec ${CT_ID} -- journalctl -u opensourcebackup -f$(printf '%*s' $((7-${#CT_ID})) '')║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Temporäre Dateien aufräumen
rm -f "$DUMP_FILE" "$ENV_BACKUP" "$PASS_BACKUP"
info "Temporäre Dateien gelöscht"
