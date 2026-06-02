# User Guide — OpensourceBackup

> Stand: B1–B16 — Control Plane, Agent, Web-Dashboard.

---

## Was ist OpensourceBackup?

OpensourceBackup sichert Dateien, Ordner und Datenbanken auf deinen Servern und Clients — zentral verwaltet, automatisch überwacht.

```
Control Plane + Web-UI  → verwaltet Systeme, Policies, Jobs, Snapshots
Agent (auf Zielsystem)  → führt restic backup aus, meldet zurück
```

**Kern-Frage des Dashboards:** *Sind meine Systeme gesichert — und wurde ein Restore getestet?*

---

## Installation auf Proxmox (Empfehlung)

Der empfohlene Weg für Heimlabs und kleine Umgebungen: **Debian 12 LXC Container** auf Proxmox VE.

### Warum LXC?

```
Proxmox VE
└── LXC Container (Debian 12, 2 CPU, 2 GB RAM, 20 GB)
    └── opensourcebackup-server  ← Control Plane + API + Web-UI
    └── PostgreSQL 16 + Redis 7  ← Docker (automatisch installiert)
```

- Kleiner Footprint (~500 MB RAM im Betrieb)
- Vollständig isoliert vom Proxmox-Host
- Einfach snapshot-bar und migrierbar

### Schritt 1 — LXC Container erstellen

Im Proxmox Shell oder über das Datacenter-Terminal:

```bash
# Verfügbare Debian-Templates anzeigen
pveam update
pveam available | grep debian

# Neuestes Debian 12 Template herunterladen
# (Name prüfen: pveam available | grep "debian-12-standard")
pveam download local debian-12-standard_12.12-1_amd64.tar.zst

# Template-Name automatisch ermitteln (funktioniert mit jeder Version)
TEMPLATE=$(pveam list local | grep "debian-12-standard" | awk '{print $1}' | tail -1)
echo "Verwende Template: $TEMPLATE"

# LXC erstellen
pct create 200 $TEMPLATE \
  --hostname opensourcebackup \
  --memory 2048 \
  --cores 2 \
  --rootfs local-lvm:20 \
  --net0 name=eth0,bridge=vmbr0,ip=dhcp \
  --features nesting=1 \
  --unprivileged 1

# Container starten
pct start 200
pct enter 200
```

### Schritt 2 — Install Script ausführen

Im Container (als root):

```bash
curl -fsSL \
  https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh \
  | bash
```

Das Script macht automatisch:
- Docker installieren (falls nicht vorhanden)
- PostgreSQL 16 + Redis 7 starten
- Control Plane Binary herunterladen
- Datenbank-Migrationen ausführen
- systemd-Service einrichten (`opensourcebackup`)
- Zugangsdaten in `/root/opensourcebackup-credentials.txt` speichern

### Schritt 3 — Web-UI verbinden

Auf deinem Laptop/PC (wo du die Web-UI nutzt):

```bash
# Statt localhost:8080 die IP des Containers verwenden
VITE_API_URL=http://192.168.1.xxx:8080 npm run dev
# → http://localhost:5173
```

Oder in der Web-UI unter **Settings** → API URL anpassen.

### Schritt 4 — Agent installieren (Linux)

Auf dem zu sichernden System:

```bash
# Enrollment-Token aus der Web-UI holen (Systems → Enrollment Token)
CONTROL_PLANE_URL=http://192.168.1.xxx:8080 \
ENROLLMENT_TOKEN=<token-aus-dem-dashboard> \
RESTIC_PASSWORD=<dein-backup-passwort> \
RESTIC_REPO=/mnt/nas/backups \
bash <(curl -fsSL http://192.168.1.xxx:8080/scripts/install-agent.sh)
```

### Schritt 5 — Agent installieren (Windows)

```powershell
# Agent herunterladen
Invoke-WebRequest "http://192.168.1.xxx:8080/downloads/agent/v0.1.0/windows-amd64" `
  -OutFile opensourcebackup-agent.exe

# Token aus der Web-UI (Agents-Seite → Enroll Agent → Schritt 4)
$env:CONTROL_PLANE_URL  = "http://192.168.1.xxx:8080"
$env:ENROLLMENT_TOKEN   = "<token>"
$env:RESTIC_PASSWORD    = "<backup-passwort>"
$env:RESTIC_REPO        = "Z:\OpenSourceBackup"  # NAS-Pfad
.\opensourcebackup-agent.exe
```

### Windows Agent — Autostart einrichten

Nach der ersten erfolgreichen Enrollierung den Agent für automatischen Start konfigurieren.

**Option A — Autostart bei Benutzer-Login (kein Admin nötig):**

```powershell
# Umgebungsvariablen systemweit setzen (einmalig als Admin):
[System.Environment]::SetEnvironmentVariable("OSB_SERVER_URL","http://<server-ip>:8080","Machine")
[System.Environment]::SetEnvironmentVariable("OSB_TOKEN_FILE","C:\ProgramData\OpenSourceBackup\agent-token","Machine")

# Autostart-Eintrag fuer aktuellen Benutzer (kein Admin):
Set-ItemProperty `
  -Path "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" `
  -Name "OpenSourceBackupAgent" `
  -Value "C:\ProgramData\OpenSourceBackup\opensourcebackup-agent.exe"
```

Startet automatisch bei jedem Windows-Login. Prüfen:

```powershell
Get-ItemProperty "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" -Name "OpenSourceBackupAgent"
Get-Process "opensourcebackup-agent" -ErrorAction SilentlyContinue
```

**Option B — Task Scheduler (startet auch ohne Login, empfohlen fuer Server):**

```powershell
# Als Administrator ausfuehren:
schtasks /Create `
  /TN "OpenSourceBackupAgent" `
  /TR "C:\ProgramData\OpenSourceBackup\opensourcebackup-agent.exe" `
  /SC ONSTART `
  /RU SYSTEM `
  /RL HIGHEST `
  /F

schtasks /Run /TN "OpenSourceBackupAgent"
```

> **Hinweis:** Das Agent-Binary unterstuetzt keine Windows Service Control API (`sc.exe create` haengt sich auf).
> Immer Task Scheduler verwenden.

### Schritt 6 — Agent installieren (FreeBSD / OPNsense)

Für OPNsense und andere FreeBSD-Systeme wird ein FreeBSD-spezifisches Binary benötigt (CGO_ENABLED=0, GOOS=freebsd).

```sh
# 1. Binary auf den Router übertragen (z.B. via SCP)
scp opensourcebackup-agent-freebsd root@192.168.0.41:/usr/local/bin/opensourcebackup-agent
ssh root@192.168.0.41 chmod +x /usr/local/bin/opensourcebackup-agent

# 2. rc.d Service-Datei anlegen
cat > /usr/local/etc/rc.d/opensourcebackup << 'EOF'
#!/bin/sh
# PROVIDE: opensourcebackup
# REQUIRE: NETWORKING
# KEYWORD: shutdown

. /etc/rc.subr

name="opensourcebackup"
rcvar="opensourcebackup_enable"
command="/usr/local/bin/opensourcebackup-agent"
pidfile="/var/run/${name}.pid"

load_rc_config $name
run_rc_command "$1"
EOF
chmod +x /usr/local/etc/rc.d/opensourcebackup

# 3. Autostart aktivieren
echo 'opensourcebackup_enable="YES"' > /etc/rc.conf.d/opensourcebackup

# 4. Umgebungsvariablen setzen (in /etc/rc.conf.d/opensourcebackup ergänzen)
echo 'opensourcebackup_env="CONTROL_PLANE_URL=http://192.168.0.72:8080 AGENT_TOKEN_FILE=/usr/local/etc/opensourcebackup-token"' >> /etc/rc.conf.d/opensourcebackup

# 5. Enrollment-Token aus der Web-UI holen und eintragen
echo -n "<enrollment-token>" > /usr/local/etc/opensourcebackup-token
chmod 600 /usr/local/etc/opensourcebackup-token

# 6. Service starten
service opensourcebackup start

# Logs
tail -f /var/log/opensourcebackup.log
```

> **Hinweis RESTIC_REPO / RESTIC_PASSWORD:** Beide Variablen sind optional — der Agent startet auch ohne sie und sendet Heartbeats. Restic-Backups werden erst ausgeführt wenn beide konfiguriert sind.

### Alternative: Direkt auf Proxmox-Host

Das Script läuft auch direkt auf dem Proxmox-Host (Debian 12), ist aber für Produktion nicht empfohlen (teilt das OS mit Proxmox).

```bash
# Auf dem Proxmox-Host als root:
curl -fsSL \
  https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh \
  | bash
```

### Verwaltungs-Befehle

```bash
# Service-Status
systemctl status opensourcebackup

# Logs
journalctl -u opensourcebackup -f

# Neustart
systemctl restart opensourcebackup

# Datenbank-Container
docker compose -f /opt/opensourcebackup/docker-compose.yml ps
```

---

## Lokaler Start (Entwicklung)

```bash
# 1. Control Plane + Datenbank
make dev-up && make migrate-up && make run

# 2. Web-UI
cd web && npm install && npm run dev
# → http://localhost:5173

# 3. Gesundheit prüfen
curl http://localhost:8080/health  # → {"status":"ok"}
```

---

## Erster Backup in 4 Schritten

### Schritt 1 — System registrieren
**Systems-Seite → (kommt in B_FORM)**  
Aktuell per API:
```powershell
Invoke-WebRequest "http://localhost:8080/v1/systems" -Method POST `
  -ContentType "application/json" `
  -Body '{"Hostname":"mein-server","RiskClass":"standard"}' `
  -UseBasicParsing
```

### Schritt 2 — Repository anlegen
**Repositories-Seite** (Create-Formular kommt in B_FORM)
```powershell
Invoke-WebRequest "http://localhost:8080/v1/repositories" -Method POST `
  -ContentType "application/json" `
  -Body '{"Type":"restic","Location":"C:/tmp/backup-repo"}' `
  -UseBasicParsing
```

### Schritt 3 — Policy erstellen
**Policies-Seite → „+ New Policy"**

| Feld | Beschreibung |
|---|---|
| Name | z.B. `nightly-documents` |
| Engine | restic / borg / pgbackrest / velero |
| Repository | Dropdown aus vorhandenen Repos |
| Include Paths | Ordner zum Sichern, z.B. `C:/Users/Admin/Documents` |
| Exclude Paths | Optional: `C:/Users/Admin/AppData` |
| Schedule | Preset oder eigener Cron-Ausdruck |
| Retention | Wie viele Snapshots behalten (Daily / Weekly / Monthly) |

### Schritt 4 — Agent installieren
**Agents-Seite → Install Agent Wizard**

1. System aus Liste wählen
2. Platform wählen (Windows / Linux)
3. Repository-Pfad + Passwort eingeben
4. Fertigen Installationsbefehl kopieren und auf Zielsystem ausführen

Der Agent enrollt sich automatisch und startet den Backup-Zyklus.

---

## Web-UI Übersicht

| Seite | URL | Was du siehst |
|---|---|---|
| Dashboard | `/` | Health-Cards, Restore-Status, Recent Jobs |
| Systems | `/systems` | Alle Systeme, Last Backup, ▶ Run Backup |
| Agents | `/agents` | Install-Wizard, Connected Systems, Remove |
| Policies | `/policies` | Backup-Regeln, + New Policy, 🗑 Delete |
| Jobs | `/jobs` | Jobs mit Filter, + New Job, 🗑 Delete (pending/failed) |
| Snapshots | `/snapshots` | Alle Snapshots, Restore-Test-Status |
| Restore Tests | `/restore-tests` | Kommt in B13/B14 |
| Repositories | `/repositories` | Storage-Ziele |

---

## Agent verwalten

### Agent herunterladen
```powershell
# Windows
Invoke-WebRequest "http://localhost:8080/downloads/agent/v0.1.0/windows-amd64" `
  -OutFile opensourcebackup-agent.exe

# Linux
curl -fsSL http://localhost:8080/downloads/agent/v0.1.0/linux-amd64 `
  -o opensourcebackup-agent && chmod +x opensourcebackup-agent
```

### Agent starten (nach Enrollment)
```powershell
$env:CONTROL_PLANE_URL = "http://localhost:8080"
$env:RESTIC_PASSWORD   = "<passwort>"
$env:RESTIC_REPO       = "<pfad-oder-url>"
.\opensourcebackup-agent.exe
```

### Agent stoppen
```powershell
Stop-Process -Name "opensourcebackup-agent" -Force
```

### Agent entfernen
**Agents-Seite** oder **Systems-Seite** → 🗑 Remove → Bestätigung

Der Agent stoppt beim nächsten Poll (max. 30s) mit 401.

---

## Backup manuell auslösen

**Jobs-Seite → „+ New Job"** → System + Policy wählen → Run Backup

Oder auf der **Systems-Seite** → ▶ Run Backup in der jeweiligen Zeile.

---

## Policy-Pfade konfigurieren

**Policies-Seite → „+ New Policy"**

Include-Pfade:
```
C:/Users/Admin/Documents
C:/Users/Admin/Desktop
C:/ProgramData/myapp
```

Exclude-Pfade (optional):
```
C:/Users/Admin/AppData
C:/Users/Admin/Documents/temp
```

Nach dem Erstellen: Job manuell oder per Schedule auslösen.

---

## Scheduler

Alle Policies mit einem Cron-Schedule werden beim Start der Control Plane geladen.
**Neue Policy → Control Plane neu starten damit der Schedule aktiv wird.**

**Dead-Man's-Switch:** Alle 5 Min prüft der Scheduler ob Jobs termingerecht liefen:
```json
{"level":"WARN","msg":"dead-man: overdue job detected","policy_name":"nightly"}
```

---

## HTTP-Statuscodes

| Code | Bedeutung |
|---|---|
| 200 / 201 / 204 | OK / Angelegt / Gelöscht |
| 400 | Ungültige Eingabe (z.B. fehlende Pflichtfelder) |
| 401 | Kein oder ungültiger Bearer-Token |
| 404 | Nicht gefunden |
| 413 | Request Body > 1 MB |
| 503 | DB nicht erreichbar oder Timeout |

---

## Sicherheitshinweise

- Agent-Token in `data/agent-token` mit Rechten `0600`
- Enrollment-Token gilt nur 30 Minuten, nur einmal verwendbar
- `RESTIC_PASSWORD` und `DATABASE_URL` nie in Logs ausgeben
- System löschen → alle Tokens werden automatisch mitgelöscht (CASCADE)
