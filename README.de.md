# OpenSourceBackup

> **Backups zu erstellen ist einfach. Wiederherstellbarkeit zu beweisen ist der Unterschied.**

🇩🇪 Deutsch | 🇬🇧 [English Version](README.md)

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.1.0-blue)](CHANGELOG.md)

---

OpenSourceBackup ist eine Open-Source **Backup Control Plane**, die Backup-Agenten auf deinen Systemen orchestriert — Windows, Linux, FreeBSD und NAS. Es verfolgt jedes Backup, führt automatische Restore-Tests durch und gibt dir ein einziges Dashboard, um **zu beweisen, dass deine Daten wirklich wiederherstellbar sind**.

> **Keine Backup-Engine.** OpenSourceBackup orchestriert [Restic](https://restic.net), Borg, pgBackRest und Velero. Du bringst den Speicher — OpenSourceBackup bringt die Kontrolle.

---

## Was es macht

```
Agent (auf deinen Systemen)
  └── sichert Dateien/DBs → Repository (NAS / S3 / lokal)
        └── meldet an Control Plane
              └── Web-Dashboard zeigt Gesundheit, Jobs, Snapshots
                    └── Restore-Tests beweisen Wiederherstellbarkeit
```

- 📦 **Backup-Orchestrierung** — Policies planen, Jobs ausführen, Ergebnisse über 100+ Systeme verfolgen
- 🔄 **Restore-Verifizierung** — automatisierte Restore-Tests mit Dateianzahl- und Größenvalidierung
- 🖥 **Multi-Plattform-Agenten** — Windows, Linux x64/ARM64, FreeBSD/OPNsense — alle als Systemdienste
- 📊 **Zentrales Dashboard** — Gesundheitsübersicht, Live-Job-Fortschritt, Snapshot-Historie
- 🔒 **Security-first** — bcrypt-Authentifizierung, CSRF-Schutz, Audit-Log, DSGVO Export/Löschung

---

## Schnellstart

### Option A — Windows (ein Befehl)

```powershell
# PowerShell als Administrator
$env:RESTIC_PASSWORD="dein-backup-passwort"
$env:RESTIC_REPO="C:\Backups"
irm https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-local.ps1 | iex
```

Installiert Control Plane + Agent + PostgreSQL + Redis als Windows-Dienste. Öffnet das Dashboard automatisch.

### Option B — Proxmox (erstellt LXC-Container automatisch)

```bash
# Auf dem Proxmox-Host als root
bash <(curl -fsSL https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-proxmox.sh)
```

Findet automatisch eine freie Container-ID (ab 200), lädt das Debian-12-Template herunter, erstellt und startet den LXC und installiert alles darin. Gibt am Ende die Dashboard-URL aus.

### Option C — Linux-Server / LXC manuell

```bash
curl -fsSL https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh | bash
```

### Nach der Installation

1. `http://<deine-ip>:8080/ui/` öffnen
2. `ADMIN_PASSWORD` in `/etc/opensourcebackup/server.env` setzen und neu starten
3. **Agents → + Enroll Agent** → Wizard folgen
4. **Repository** anlegen (wohin Backups gespeichert werden)
5. **Policy** anlegen (was gesichert wird, wann)
6. Ersten **Job** starten und Live-Fortschritt beobachten

---

## Architektur

```
┌──────────────────────────────────────────────────────┐
│                   Web-Dashboard                       │
│            React 18 + TypeScript + Vite              │
└──────────────────────┬───────────────────────────────┘
                       │ HTTP REST + Session-Cookie
┌──────────────────────▼───────────────────────────────┐
│               Control Plane (Go 1.22+)                │
│  ┌─────────────┐  ┌───────────┐  ┌─────────────────┐ │
│  │  Scheduler  │  │  REST API │  │ Auth / Audit    │ │
│  │  (Cron)     │  │  /v1/*    │  │ bcrypt · CSRF   │ │
│  └─────────────┘  └───────────┘  └─────────────────┘ │
└──────────────┬──────────────────────────┬────────────┘
               │                          │
┌──────────────▼──────────┐  ┌────────────▼────────────┐
│     PostgreSQL 16        │  │        Redis 7           │
│  Katalog · Audit (RLS)  │  │                          │
└─────────────────────────┘  └─────────────────────────┘
               │
               │ Bearer-Token (SHA-256-Hash)
┌──────────────▼──────────────────────────────────────┐
│                    Agent                             │
│  Windows-Dienst / systemd / rc.d (FreeBSD)          │
│  ┌──────────────────────────────────────────────┐   │
│  │               Restic Runner                  │   │
│  │  Backup → Repository (NAS / S3 / lokal)     │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

---

## Plattform-Unterstützung

### Agent-Plattformen

| Plattform | Dienst | Installer |
|---|---|---|
| Windows x64 | Windows-Dienst | `install-agent.ps1` / MSI / EXE |
| Linux x64 | systemd | `install-agent.sh` |
| Linux ARM64 | systemd | `install-agent.sh` |
| FreeBSD x64 / OPNsense | rc.d | `install-agent-freebsd.sh` |

### Repository-Typen

| Typ | Beispiel |
|---|---|
| Lokaler Pfad | `/var/backups`, `C:\Backups` |
| NAS / SMB | `Z:\OpenSourceBackup`, Synology via CIFS |
| NAS / NFS | `/mnt/nas/backups`, QNAP |
| MinIO / S3 | Self-hosted MinIO, AWS S3, Azure Blob, B2 |
| Proxmox Storage | `/mnt/pve/Backup`, `/mnt/pve/NAS` |
| Borg via SSH | `user@host:./backups` |
| pgBackRest | PostgreSQL WAL-Archivierung, Point-in-Time Recovery |
| Velero | Kubernetes Deployments + Volumes |

---

## Sicherheit & DSGVO

OpenSourceBackup implementiert **technische Bausteine zur Unterstützung eines DSGVO-konformen Betriebs**. Tatsächliche Konformität erfordert Rechtsgrundlage, Prozesse und Dokumentation durch den Betreiber.

| Maßnahme | Umsetzung |
|---|---|
| Authentifizierung | bcrypt Admin-Passwort (Cost 12), 8h Sessions |
| Session-Sicherheit | HttpOnly + SameSite=Strict Cookies |
| Brute-Force-Schutz | 5 Versuche/Min pro gehashter IP |
| CSRF-Schutz | Double-Submit-Cookie (`X-CSRF-Token`) |
| Transport | TLS über `TLS_CERT_FILE` + `TLS_KEY_FILE` |
| Backup-Verschlüsselung | Restic AES-256-CTR + Poly1305 (clientseitig) |
| Token-Speicherung | Nur SHA-256-Hashes — niemals Klartext |
| Audit-Log | Append-only, IP gehasht, PostgreSQL RLS |
| DSGVO Art. 20 | `GET /v1/gdpr/systems/{id}/export` |
| DSGVO Art. 17 | `DELETE /v1/gdpr/systems/{id}/purge` |
| Security-Header | CSP, HSTS, X-Frame-Options, Permissions-Policy |

→ Details: [SECURITY.md](SECURITY.md)

---

## Konfiguration

### Control Plane (Umgebungsvariablen)

| Variable | Standard | Beschreibung |
|---|---|---|
| `DATABASE_URL` | — | PostgreSQL DSN (**erforderlich**) |
| `LISTEN_ADDR` | `:8080` | HTTP-Adresse |
| `ADMIN_PASSWORD` | — | Dashboard-Passwort (leer = kein Auth, **nur Dev**) |
| `CORS_ORIGIN` | `http://localhost:5173` | Erlaubter CORS-Origin |
| `TLS_CERT_FILE` | — | TLS-Zertifikat (aktiviert HTTPS) |
| `TLS_KEY_FILE` | — | TLS-Schlüssel |
| `WEB_UI_DIR` | — | Pfad zum Web-UI `dist/`-Verzeichnis |

### Agent (Umgebungsvariablen)

| Variable | Beschreibung |
|---|---|
| `CONTROL_PLANE_URL` | Control-Plane-URL (**erforderlich**) |
| `RESTIC_PASSWORD` | Backup-Verschlüsselungspasswort (**erforderlich**) |
| `RESTIC_REPO` | Backup-Ziel (**erforderlich**) |
| `ENROLLMENT_TOKEN` | Einmal-Token (nur erster Start) |
| `AGENT_POLL_INTERVAL` | Poll-Intervall (Standard: `30s`) |
| `AGENT_TOKEN_FILE` | Gespeicherter Token-Pfad (Standard: `data/agent-token`) |
| `RESTORE_TEST_ROOT` | Sandbox für Restore-Tests |
| `RESTIC_BIN` | Pfad zur Restic-Binary |

### Agent-Befehle

```bash
opensourcebackup-agent install    # Als Systemdienst registrieren
opensourcebackup-agent start      # Dienst starten
opensourcebackup-agent stop       # Dienst stoppen
opensourcebackup-agent restart    # Dienst neustarten
opensourcebackup-agent status     # Status anzeigen
opensourcebackup-agent uninstall  # Dienst entfernen
opensourcebackup-agent            # Interaktiv starten (Dev/Debug)
```

---

## Entwicklung

### Voraussetzungen

- Go 1.22+
- Docker Desktop (PostgreSQL + Redis)
- Node.js 20 LTS (Web-UI)

### Lokales Setup

```bash
git clone https://github.com/cerberus8484/opensourcebackup.git
cd opensourcebackup

# Datenbank starten
make dev-up

# Migrationen ausführen
make migrate-up

# Server starten (http://localhost:8080)
make run

# Web-UI Dev-Server (http://localhost:5173, mit HMR)
cd web && npm install && npm run dev
```

### Tests

```bash
make test                # Unit-Tests
make test-integration    # Benötigt laufendes PostgreSQL
make lint                # Harte Regeln (blockiert CI)
make lint-warn           # Weiche Regeln (informativ)
```

### Bauen

```bash
make build-all                    # Alle Binaries (Agent + Server, alle Plattformen)
make build-agent-all              # Nur Agent-Binaries
make build-agent-windows          # Windows-Agent
make build-agent-linux            # Linux-x64-Agent
make build-agent-freebsd          # FreeBSD-Agent
.\scripts\build-release.ps1       # Vollständiger Release: Binaries + MSI + EXE + Checksums
```

---

## Roadmap

| # | Feature | Status |
|---|---|---|
| B_RBAC | Login-UI + Admin/Operator/Auditor-Rollen | 🔜 Nächster Schritt |
| B_RET | Retention-Policies + automatisches Prune | 🔜 Geplant |
| B15 | Prometheus-Metriken-Endpunkt | 📋 Geplant |
| R-01 | CSP `unsafe-inline` entfernen (Self-hosted Fonts) | 📋 Backlog |
| R-03 | TLS-Enforcement-Flag | 📋 Backlog |

---

## Changelog

Siehe [CHANGELOG.md](CHANGELOG.md) für die vollständige Historie.

## Mitwirken

Siehe [CONTRIBUTING.md](CONTRIBUTING.md).

## Lizenz

[MIT](LICENSE) — © 2026 cerberus8484

---

*OpenSourceBackup ist keine Backup-Engine. Es orchestriert deine bestehenden Tools und beweist, dass deine Daten tatsächlich wiederherstellbar sind.*
