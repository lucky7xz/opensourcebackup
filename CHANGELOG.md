# Changelog

All notable changes to OpenSourceBackup are documented in this file.
Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning follows [Semantic Versioning](https://semver.org/).

🇩🇪 [Deutsche Version](#deutsch) weiter unten.

---

## [English]

---

## [0.1.0] — 2026-05-31

### Added — Core Platform

- **Control Plane** (Go 1.22+): REST API for managing systems, repositories, policies, jobs, snapshots, and restore tests
- **PostgreSQL 16** catalog with 13 database migrations
- **Redis 7** for future caching/pub-sub
- **Scheduler**: cron-based policy execution, Dead-Man's Switch for missed backups
- **Snapshot catalog**: tracks every successful backup with size, duration, and file count

### Added — Backup Agent

- **Cross-platform agent binary** (single codebase, four targets):
  - Windows x64 — runs as a Windows Service
  - Linux x64 / ARM64 — runs as a systemd unit
  - FreeBSD x64 — runs as an rc.d service (OPNsense / pfSense)
- **Enrollment protocol**: one-time token → permanent agent token (SHA-256 hashed in DB)
- **Restic runner**: backup, restore, restore-test with sandbox isolation
- **Windows UtimesNano fix**: treats timestamp-only exit-code-1 as success

### Added — Web Dashboard

- **React 18 + TypeScript + Vite** SPA served directly from the Go binary
- Pages: Dashboard (health overview), Systems, Agents, Repositories, Policies, Jobs, Restore Tests
- **Job Detail Panel** with live auto-refresh every 3s during active jobs
- **Enrollment Wizard**: 4-step guide with platform-specific install commands (Windows/Linux/FreeBSD)
- 9 repository types with descriptions and setup hints
- Engine selector: Restic, Borg, pgBackRest, Velero

### Added — Installer Suite

- `scripts/install-proxmox.sh` — auto-creates Debian 12 LXC container and installs inside it
- `scripts/install-server.sh` — Linux/Debian server install (Docker + PostgreSQL + Redis + service)
- `scripts/install-local.ps1` — Windows all-in-one: Control Plane + Agent + Docker stack
- `scripts/install-agent.sh` — Linux agent (systemd service)
- `scripts/install-agent-freebsd.sh` — FreeBSD/OPNsense agent (rc.d service)
- `scripts/install-agent.ps1` — Windows agent (Windows Service)
- `scripts/build-release.ps1` — builds all platform binaries + MSI + EXE
- `build/windows/agent.wxs` — WiX MSI installer definition
- `build/windows/agent-installer.nsi` — NSIS EXE installer with config wizard
- `build/windows/local-installer.nsi` — NSIS all-in-one local installer

### Added — Security

- **Web authentication**: bcrypt admin password (cost 12), 8h sessions, HttpOnly + SameSite=Strict cookies
- **CSRF protection**: Double-Submit-Cookie pattern; agent routes and `/auth/login` exempt
- **Rate limiting**: token-bucket per IP — 5 req/min on auth endpoints, 20 req/s global
- **Security headers**: CSP, HSTS (1 year), X-Frame-Options DENY, Referrer-Policy, Permissions-Policy
- **Input protection**: parameterized queries (pgx), 1 MB body limit, 30s request timeout
- **Token security**: SHA-256 hashes only in DB, never plaintext in logs or responses
- `internal/security/` package: `IPRateLimiter`, `CSRFProtect`, `ClientIPHashed`
- `SECURITY.md`: public security policy with correct GDPR language

### Added — GDPR / DSGVO

- **Audit log** (migration 000011): append-only table, indexed by resource/actor/time
- **IP hashing**: SHA-256 8-byte hash stored instead of plaintext IP
- **PostgreSQL Row Security Policy** (migration 000013): app user may only INSERT + SELECT on `audit_log` — UPDATE/DELETE blocked at DB level even if application code is compromised
- **GDPR fields on systems** (migration 000012): `data_owner`, `retention_days`, `gdpr_note`
- `GET /v1/gdpr/systems/{id}/export` — Art. 20 data portability (JSON)
- `DELETE /v1/gdpr/systems/{id}/purge` — Art. 17 right to erasure (catalog data)
- `GET /v1/audit` — transparency endpoint
- `docs/security/`: threat-model, TOM, gdpr-notes, audit-log (operator-private, in .gitignore)

### Added — Developer Tooling

- Global skill system: `clean-code`, `ccd-wertesystem`, `security`, `dsgvo` skills in `~/.claude/`
- `Makefile`: all build targets including `build-agent-freebsd`, `installer-windows`, `release`
- TLS dev certificate generator (`internal/tools/gencert`)
- `docs/setup/SETUP_EN.md` + `SETUP_DE.md`: step-by-step setup guides
- `INSTALL.md`: comprehensive installation reference

### Fixed

- SPA handler: `filepath.Join` with absolute path dropped base directory — fixed with `strings.TrimPrefix`
- PostgreSQL password mismatch on re-runs — password now persisted in `/etc/opensourcebackup/.db_password`
- `docker compose exec` vs `docker exec` in non-interactive shell — switched to `docker exec`
- Go 1.19 on Debian apt (too old) — install script now installs Go 1.22.5 from go.dev if needed
- Windows UtimesNano error: restic exits code 1 when it cannot set timestamps on protected Windows directories but data was fully restored — treated as success
- TypeScript duplicate style keys in Systems.tsx and Agents.tsx
- Dashboard.tsx variable shadowing (`s` style object vs render callback)
- Vite `--silent` flag not supported in installed version — removed

### Known Limitations (v0.1.0)

- No RBAC — single global admin password (B_RBAC planned)
- TLS is opt-in, not enforced (R-03)
- CSP contains `unsafe-inline` (required for Vite/Google Fonts — R-01)
- GDPR purge deletes catalog data only; Restic repository content requires separate manual process
- No retention/prune automation (B_RET planned)
- No Prometheus metrics (B15 planned)

---

---

## [Deutsch]

---

## [0.1.0] — 2026-05-31

### Hinzugefügt — Core Platform

- **Control Plane** (Go 1.22+): REST-API für Systeme, Repositories, Policies, Jobs, Snapshots und Restore-Tests
- **PostgreSQL 16** Katalog mit 13 Datenbank-Migrationen
- **Redis 7** für zukünftiges Caching/Pub-Sub
- **Scheduler**: Cron-basierte Policy-Ausführung, Dead-Man's-Switch für verpasste Backups
- **Snapshot-Katalog**: verfolgt jedes erfolgreiche Backup mit Größe, Dauer und Dateianzahl

### Hinzugefügt — Backup-Agent

- **Plattformübergreifende Agent-Binary** (eine Codebasis, vier Zielplattformen):
  - Windows x64 — läuft als Windows-Dienst
  - Linux x64 / ARM64 — läuft als systemd-Unit
  - FreeBSD x64 — läuft als rc.d-Dienst (OPNsense / pfSense)
- **Enrollment-Protokoll**: Einmal-Token → dauerhafter Agent-Token (SHA-256-gehasht in DB)
- **Restic Runner**: Backup, Restore, Restore-Test mit Sandbox-Isolierung
- **Windows UtimesNano-Fix**: Exit-Code 1 bei reinen Timestamp-Fehlern wird als Erfolg gewertet

### Hinzugefügt — Web-Dashboard

- **React 18 + TypeScript + Vite** SPA, direkt aus der Go-Binary ausgeliefert
- Seiten: Dashboard (Gesundheitsübersicht), Systeme, Agents, Repositories, Policies, Jobs, Restore-Tests
- **Job-Detail-Panel** mit Live-Aktualisierung alle 3 Sekunden bei aktiven Jobs
- **Enrollment-Wizard**: 4-Schritte-Assistent mit plattformspezifischen Installationsbefehlen
- 9 Repository-Typen mit Beschreibungen und Einrichtungshinweisen
- Engine-Auswahl: Restic, Borg, pgBackRest, Velero

### Hinzugefügt — Installer-Suite

- `scripts/install-proxmox.sh` — erstellt automatisch Debian-12-LXC-Container und installiert darin
- `scripts/install-server.sh` — Linux/Debian-Server-Installation (Docker + PostgreSQL + Redis + Dienst)
- `scripts/install-local.ps1` — Windows alles-in-einem: Control Plane + Agent + Docker-Stack
- `scripts/install-agent.sh` — Linux-Agent (systemd-Dienst)
- `scripts/install-agent-freebsd.sh` — FreeBSD/OPNsense-Agent (rc.d-Dienst)
- `scripts/install-agent.ps1` — Windows-Agent (Windows-Dienst)
- `scripts/build-release.ps1` — baut alle Plattform-Binaries + MSI + EXE
- `build/windows/agent.wxs` — WiX-MSI-Installer-Definition
- `build/windows/agent-installer.nsi` — NSIS-EXE-Installer mit Konfigurations-Assistent
- `build/windows/local-installer.nsi` — NSIS alles-in-einem lokaler Installer

### Hinzugefügt — Sicherheit

- **Web-Authentifizierung**: bcrypt Admin-Passwort (Cost 12), 8h Sessions, HttpOnly + SameSite=Strict Cookies
- **CSRF-Schutz**: Double-Submit-Cookie-Muster; Agent-Routen und `/auth/login` ausgenommen
- **Rate-Limiting**: Token-Bucket pro IP — 5 Anfragen/Min auf Auth-Endpunkten, 20 Anfragen/s global
- **Security-Header**: CSP, HSTS (1 Jahr), X-Frame-Options DENY, Referrer-Policy, Permissions-Policy
- **Eingabeschutz**: parametrisierte Queries (pgx), 1-MB-Body-Limit, 30s Request-Timeout
- **Token-Sicherheit**: nur SHA-256-Hashes in DB, niemals Klartext in Logs oder Antworten
- `internal/security/` Paket: `IPRateLimiter`, `CSRFProtect`, `ClientIPHashed`
- `SECURITY.md`: öffentliche Security-Policy mit korrekter DSGVO-Sprache

### Hinzugefügt — DSGVO

- **Audit-Log** (Migration 000011): Append-Only-Tabelle, indiziert nach Ressource/Akteur/Zeit
- **IP-Hashing**: SHA-256-8-Byte-Hash statt Klartext-IP
- **PostgreSQL Row Security Policy** (Migration 000013): App-User darf nur INSERT + SELECT auf `audit_log` — UPDATE/DELETE auf DB-Ebene gesperrt, auch bei kompromittiertem Anwendungscode
- **DSGVO-Felder an Systemen** (Migration 000012): `data_owner`, `retention_days`, `gdpr_note`
- `GET /v1/gdpr/systems/{id}/export` — Art. 20 Datenportabilität (JSON)
- `DELETE /v1/gdpr/systems/{id}/purge` — Art. 17 Recht auf Löschung (Katalogdaten)
- `GET /v1/audit` — Transparenz-Endpunkt
- `docs/security/`: Threat-Model, TOM, DSGVO-Hinweise, Audit-Log (in .gitignore, Betreiber-privat)

### Behoben

- SPA-Handler: `filepath.Join` mit absolutem Pfad überschrieb Basisverzeichnis — behoben mit `strings.TrimPrefix`
- PostgreSQL-Passwort-Konflikt bei Neuinstallationen — Passwort wird jetzt in `/etc/opensourcebackup/.db_password` dauerhaft gespeichert
- `docker compose exec` vs `docker exec` in nicht-interaktiver Shell — auf `docker exec` umgestellt
- Go 1.19 von Debian apt (zu alt) — Install-Script installiert jetzt Go 1.22.5 von go.dev
- Windows-UtimesNano-Fehler: Restic beendet sich mit Code 1 bei Timestamp-Fehlern auf geschützten Windows-Verzeichnissen, obwohl Daten vollständig wiederhergestellt wurden — wird jetzt als Erfolg gewertet
- TypeScript-Duplikat-Style-Keys in Systems.tsx und Agents.tsx
- Dashboard.tsx Variable-Shadowing (`s` Style-Objekt vs. Render-Callback)
- Vite-Flag `--silent` in installierter Version nicht unterstützt — entfernt

### Bekannte Einschränkungen (v0.1.0)

- Kein RBAC — globales Admin-Passwort (B_RBAC geplant)
- TLS ist opt-in, nicht erzwungen (R-03)
- CSP enthält `unsafe-inline` (erforderlich für Vite/Google Fonts — R-01)
- DSGVO-Purge löscht nur Katalogdaten; Restic-Repository-Inhalte erfordern separaten manuellen Prozess
- Keine Retention/Prune-Automatisierung (B_RET geplant)
- Keine Prometheus-Metriken (B15 geplant)
