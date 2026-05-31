# OpenSourceBackup — Installations-Anleitung

> *Backups zu erstellen ist einfach. Wiederherstellbarkeit zu beweisen ist der Unterschied.*

🇩🇪 Deutsch | 🇬🇧 [English Version](SETUP_EN.md)

---

## Installationsart wählen

| | Lokal | Proxmox |
|---|---|---|
| **Ideal für** | Testen, einzelner Rechner | Heimlabor, dauerhaft laufender Server |
| **Voraussetzungen** | Windows oder Linux, Docker | Proxmox VE 7/8 |
| **Aufwand** | ~5 Minuten | ~10 Minuten |

---

# Option A — Lokale Installation

Control Plane direkt auf dem eigenen Windows- oder Linux-Rechner betreiben — ideal zum Testen.

## Voraussetzungen

- **Docker Desktop** (Windows/Mac) oder Docker (Linux)
- Port **8080** muss frei sein

## Schritt 1 — Herunterladen und starten

**Windows (PowerShell):**
```powershell
# Ordner anlegen
mkdir C:\opensourcebackup
cd C:\opensourcebackup

# Docker Compose Datei herunterladen
Invoke-WebRequest "https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/deployments/docker-compose/dev.yml" -OutFile docker-compose.yml

# PostgreSQL + Redis starten
docker compose -f docker-compose.yml up -d
```

**Linux:**
```bash
mkdir ~/opensourcebackup && cd ~/opensourcebackup
curl -fsSL https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/deployments/docker-compose/dev.yml -o docker-compose.yml
docker compose -f docker-compose.yml up -d
```

## Schritt 2 — Server herunterladen und starten

**Windows:**
```powershell
# Server-Binary herunterladen
Invoke-WebRequest "https://github.com/cerberus8484/opensourcebackup/releases/latest/download/opensourcebackup-server-windows-amd64.exe" -OutFile opensourcebackup-server.exe

# Konfiguration setzen
$env:DATABASE_URL = "postgres://opensourcebackup:dev_password@localhost:5432/opensourcebackup?sslmode=disable"
$env:LISTEN_ADDR  = ":8080"

# Starten
.\opensourcebackup-server.exe
```

**Linux:**
```bash
# Server-Binary herunterladen
curl -fsSL https://github.com/cerberus8484/opensourcebackup/releases/latest/download/opensourcebackup-server-linux-amd64 \
  -o opensourcebackup-server && chmod +x opensourcebackup-server

# Starten
DATABASE_URL="postgres://opensourcebackup:dev_password@localhost:5432/opensourcebackup?sslmode=disable" \
LISTEN_ADDR=":8080" \
./opensourcebackup-server
```

## Schritt 3 — Dashboard öffnen

```
http://localhost:8080/ui/
```

> ✅ Das **Backup Health** Dashboard sollte erscheinen.

## Schritt 4 — Agent installieren

Im Dashboard: **Agents → + Enroll Agent** → Wizard folgen

---

# Option B — Proxmox Installation

Control Plane auf dem Proxmox-Server installieren — läuft dauerhaft und kann alle Systeme sichern.

## Empfohlen: Debian 12 LXC Container

Ein leichtgewichtiger Container hält OpenSourceBackup vom Proxmox-Host getrennt.

## Schritt 1 — LXC Container erstellen

In der **Proxmox Shell** (oder Datacenter → Node → Shell):

```bash
# Template-Liste aktualisieren
pveam update

# Verfügbare Debian 12 Templates anzeigen
pveam available | grep "debian-12-standard"

# Template herunterladen (Namen aus der Ausgabe oben verwenden)
pveam download local debian-12-standard_12.12-1_amd64.tar.zst

# Container erstellen
TEMPLATE=$(pveam list local | grep "debian-12-standard" | awk '{print $1}' | tail -1)

pct create 200 $TEMPLATE \
  --hostname opensourcebackup \
  --memory 2048 \
  --cores 2 \
  --rootfs local-lvm:20 \
  --net0 name=eth0,bridge=vmbr0,ip=dhcp \
  --features nesting=1 \
  --unprivileged 1

# Starten und betreten
pct start 200
pct enter 200
```

> 💡 **Container-ID 200** — ändern wenn bereits belegt.
> **nesting=1** ist erforderlich damit Docker im Container läuft.

## Schritt 2 — Install-Script ausführen

Im Container (du bist jetzt root im LXC):

```bash
curl -fsSL \
  https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh \
  | bash
```

**Das Script macht automatisch:**

| Schritt | Was passiert |
|---|---|
| 1 | Docker installieren |
| 2 | PostgreSQL 16 + Redis 7 starten |
| 3 | Server-Binary bauen (Go 1.22) |
| 4 | Web-UI bauen |
| 5 | Datenbank-Migrationen ausführen |
| 6 | systemd-Service einrichten |
| 7 | Zugangsdaten + URL anzeigen |

**Das dauert ca. 5–10 Minuten** (Binary wird aus dem Quellcode gebaut).

## Schritt 3 — Zugangsdaten notieren

Am Ende erscheint:

```
╔══════════════════════════════════════════════════════════╗
║   ✓  OpenSourceBackup — Installation abgeschlossen!     ║
╠══════════════════════════════════════════════════════════╣
║   🌐  Web Dashboard: http://192.168.x.x:8080/ui/         ║
║   🔑  Benutzername: (nicht nötig — kommt in v2)          ║
║   🔑  Passwort:     (nicht nötig — kommt in v2)          ║
╚══════════════════════════════════════════════════════════╝
```

Die Zugangsdaten werden auch gespeichert unter:
```bash
cat /root/opensourcebackup-credentials.txt
```

## Schritt 4 — Dashboard öffnen

Im Browser aufrufen (eigene Proxmox-IP einsetzen):

```
http://192.168.x.x:8080/ui/
```

## Alternative: Direkt auf dem Proxmox-Host

Falls die Installation direkt auf dem Proxmox-Host gewünscht wird (für Produktion nicht empfohlen):

```bash
# Auf dem Proxmox-Host als root
curl -fsSL \
  https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh \
  | bash
```

> ⚠️ Dabei teilt OpenSourceBackup das Betriebssystem mit Proxmox. Für Produktionseinsatz LXC verwenden.

---

# Agent installieren

Sobald das Dashboard läuft, Agent auf den zu sichernden Systemen installieren.

## Windows Agent

**Im Dashboard:** Agents → + Enroll Agent → System wählen → Windows → Befehl kopieren

```powershell
# Agent herunterladen
Invoke-WebRequest "http://<server-ip>:8080/downloads/agent/v0.1.0/windows-amd64" `
  -OutFile opensourcebackup-agent.exe

# Variablen setzen (Werte aus dem Wizard verwenden)
$env:CONTROL_PLANE_URL  = "http://<server-ip>:8080"
$env:ENROLLMENT_TOKEN   = "<Token aus dem Wizard>"
$env:RESTIC_PASSWORD    = "<dein Backup-Passwort>"
$env:RESTIC_REPO        = "Z:\Backup-Ordner"

# Starten — enrollt sich beim ersten Start automatisch
.\opensourcebackup-agent.exe
```

## Linux Agent

```bash
CONTROL_PLANE_URL=http://<server-ip>:8080 \
ENROLLMENT_TOKEN=<Token aus dem Wizard> \
RESTIC_PASSWORD=<dein Backup-Passwort> \
RESTIC_REPO=/mnt/nas/backups \
bash <(curl -fsSL http://<server-ip>:8080/scripts/install-agent.sh)
```

---

# Erstes Backup

1. **Repositories** → `+ New Repository` → Typ wählen (Lokal, NAS, S3…) → Pfad eingeben
2. **Policies** → `+ New Policy` → Repository, Pfade, Zeitplan festlegen
3. **Jobs** → `+ New Job` → System und Policy wählen → ▶ Backup starten
4. Live-Fortschritt im Job-Detail-Panel beobachten

---

# Repository-Typen

| Typ | Wofür |
|---|---|
| 💾 **Lokal** | Lokale Festplatte, USB-Stick, eingebundenes Volume |
| 🖥 **Proxmox Storage** | `/mnt/pve/Backup`, `/mnt/pve/NAS` — Proxmox-Speicher |
| 🗄 **NAS / NFS** | Synology, QNAP, TrueNAS via NFS |
| 🗄 **NAS / SMB** | Windows-Netzlaufwerk (z.B. `Z:\`), Synology via SMB |
| ☁ **MinIO / S3** | Self-hosted MinIO, AWS S3, Azure, Google Cloud |
| ⚙ **Restic REST** | Eigener Restic REST-Server |
| 🔒 **Borg (SSH)** | Linux-Server über SSH — sehr effiziente Deduplizierung |
| 🐘 **pgBackRest** | Nur PostgreSQL — WAL-Archivierung, Point-in-Time |
| ☸ **Velero** | Kubernetes-Cluster |

---

# Backup-Engines

| Engine | Ideal für |
|---|---|
| **Restic** | Dateien & Ordner — Windows, Linux, NAS, S3 |
| **Borg** | Linux-Server via SSH — sehr effiziente Deduplizierung |
| **pgBackRest** | PostgreSQL-Datenbanken — WAL & Point-in-Time Recovery |
| **Velero** | Kubernetes-Cluster — Deployments, Volumes, ConfigMaps |

---

# Verwaltungs-Befehle

## Server (Linux/Proxmox)

```bash
journalctl -u opensourcebackup -f          # Logs anzeigen
systemctl restart opensourcebackup         # Neustart
systemctl stop opensourcebackup            # Stoppen
cat /root/opensourcebackup-credentials.txt # Zugangsdaten anzeigen
```

## Agent (Windows)

```powershell
Stop-Process -Name "opensourcebackup-agent" -Force  # Stoppen
Remove-Item "data\agent-token" -Force               # Token löschen (Re-Enrollment)
```

## Agent (Linux)

```bash
journalctl -u opensourcebackup-agent -f    # Logs
systemctl restart opensourcebackup-agent   # Neustart
systemctl stop opensourcebackup-agent      # Stoppen
```

---

# Häufige Probleme

| Problem | Lösung |
|---|---|
| Dashboard zeigt leere Seite | Web UI neu bauen mit `VITE_API_URL=http://<ip>:<port>` |
| "token revoked or invalid" | `data\agent-token` löschen, neu enrollen |
| Port 8080 belegt | In `/etc/opensourcebackup/server.env` ändern: `LISTEN_ADDR=:8090` |
| PostgreSQL startet nicht | `chown -R 999:999 /var/lib/opensourcebackup/postgres` |
| Restore schlägt fehl (Windows) | Agent v0.1.0+ verwenden — UtimesNano-Fehler werden ignoriert |

---

*[github.com/cerberus8484/opensourcebackup](https://github.com/cerberus8484/opensourcebackup)*
