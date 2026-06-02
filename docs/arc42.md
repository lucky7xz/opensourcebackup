# Arc42 — OpensourceBackup Architektur

> Stand: B1–B16 implementiert (Catalog, API, Scheduler, Security, Auth, Agent, Web-UI). Agent läuft auf Windows (cerberus), Linux (192.168.0.72) und FreeBSD/OPNsense (192.168.0.41).

---

## 1. Einführung und Ziele

OpensourceBackup ist eine **Backup Control Plane** — Orchestrierung von Restic, Borg, pgBackRest, Velero über 100+ Systeme. Der Agent läuft auf Zielsystemen, führt Backups aus und meldet Ergebnisse zurück.

**Kern-Aussage der UI:** *"Sind meine Systeme gesichert — und wurde ein Restore erfolgreich getestet?"*

| Priorität | Ziel |
|---|---|
| 1 | Zentrales Management für heterogene Systeme |
| 2 | Restore-Verifikation als primäre Qualitätsmessung |
| 3 | Dead-Man's-Switch bei ausbleibenden Jobs |
| 4 | Sicherer Agent-Flow: Enrollment → Bearer Token → geschützte Routen |
| 5 | Browser-basiertes Operations-Dashboard |

---

## 2. Randbedingungen

| Randbedingung | Begründung |
|---|---|
| Go 1.22+ | Agent + Control Plane: Single Binary, Cross-Compilation |
| PostgreSQL 16 | JSONB, UUID, FK-Constraints, CASCADE |
| React 18 + TypeScript | Web-Dashboard |
| Vite | Frontend Dev-Server + Build |
| Apache 2.0 | Open-Source |

---

## 3. Kontextabgrenzung

```
Browser (React)  ──── HTTP/CORS ────► Control Plane ◄──── Bearer Token ── Agent
                                      (API + Auth +         (polls jobs,
                                       Scheduler +           runs restic,
                                       Catalog +             meldet zurück)
                                       Downloads)
                                           │
                                     PostgreSQL + Redis
                                           │
                                    dist/agent/v0.1.0/
                                    (Binary-Downloads)
```

---

## 4. Lösungsstrategie

| Entscheidung | Begründung |
|---|---|
| SHA-256 Token-Hashes | Sicher für 256-bit zufällige Tokens |
| Bearer-Token (MVP) | Einfacher Einstieg; mTLS kommt in B_TLS |
| React ohne CSS-Framework | Vollständige Kontrolle über Dark-Theme |
| `/downloads/agent/{v}/{os}` | Control Plane serviert eigene Binaries |
| CASCADE FK auf Token-Tabellen | System löschen räumt Tokens automatisch auf |

---

## 5. Bausteinsicht

```
cmd/
  control-plane/   HTTP-Server + Scheduler + Auth + Downloads
  agent/           Enrollment-Flow + Poll-Loop

internal/
  api/             Handler, Middleware (CORS, Security, Auth), Downloads
  auth/            Token-Hashing, OTP-Enrollment, Agent-Bearer
  agent/           Poll, Job-Flow, Restic Runner, TokenFile
  catalog/         5 Store-Interfaces + pgx
  scheduler/       Cron + Dead-Man's-Switch

web/               React 18 + TypeScript + Vite
  src/
    pages/         Dashboard, Systems, Agents, Policies, Jobs,
                   Snapshots, RestoreTests, Repositories
    components/    Sidebar, StatusBadge, Table, Modal, ConfirmDialog, Card

migrations/        000001–000009
dist/agent/        Pre-built Binaries (nicht in Git — make build-agent-all)
```

### Web-UI Seiten

| Seite | Inhalt |
|---|---|
| Dashboard | Health-Cards, Restore-Status prominent, Recent Jobs + Failures |
| Systems | Tabelle mit Last Backup + Run Backup Button |
| Agents | 4-Schritt-Wizard: System → Platform → Config → Install |
| Policies | Tabelle + New Policy Formular (Pfade, Schedule, Retention) |
| Jobs | Filter, + New Job, 🗑 Delete für pending/failed |
| Snapshots | Restore-Test-Status pro Snapshot |
| Restore Tests | Placeholder — kommt in B13/B14 |
| Repositories | WORM-Lock Status |

---

## 6. Laufzeitsicht

### API-Request-Chain

```
HTTP → Recovery → CORS → SecurityHeaders → BodyLimit → Logging → Timeout → Handler
```

### Agent-Enrollment

```
Admin: POST /v1/systems/{id}/enrollment-token → OTP (30 Min TTL)
Agent: POST /v1/agent/enroll {token} → Bearer-Token → data/agent-token (0600)
```

### Backup-Flow

```
Scheduler → BackupJob{pending}
Agent pollt GET /v1/agent/jobs (Bearer)
→ PUT start → restic backup --json → PUT complete {snapshot_id, bytes}
→ Snapshot mit policy.repository_id registriert
```

### Agent-Download

```
make build-agent-windows/linux/linux-arm64/darwin
→ dist/agent/v0.1.0/opensourcebackup-agent-{platform}[.exe]

GET /downloads/agent          → JSON Liste verfügbarer Binaries
GET /downloads/agent/{v}/{os} → Binary-Download (path traversal geschützt)
```

---

## 7. Verteilungssicht

### Entwicklung

```
make dev-up + make migrate-up (000001–000009)
make run         → Control Plane :8080
cd web && npm run dev → Web-UI :5173
make run-agent   → Agent (token aus data/agent-token)
```

### Produktion — aktuelle Infrastruktur

```
Control Plane
  Host: 192.168.0.72 (Debian LXC auf Proxmox)
  Binary: /opt/opensourcebackup/opensourcebackup-server
  Service: systemd (opensourcebackup.service)
  Deploy: systemctl stop → cp → systemctl start
  SSH-Zugang: root@192.168.0.72 via C:\Users\Admin\.ssh\id_ed25519

Agent — cerberus (Windows)
  Binary: C:\ProgramData\OpenSourceBackup\opensourcebackup-agent.exe
  Token: C:\ProgramData\OpenSourceBackup\agent-token
  Autostart: HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Run

Agent — OPNsense (FreeBSD 14 / OPNsense 26.1.2)
  Host: 192.168.0.41 (WAN) / 192.168.1.1 (LAN) / 192.168.2.1 (WIFI)
  Binary: /usr/local/bin/opensourcebackup-agent (FreeBSD amd64)
  Service: rc.d (/usr/local/etc/rc.d/opensourcebackup)
  Autostart: /etc/rc.conf.d/opensourcebackup → opensourcebackup_enable="YES"
  Logs: /var/log/opensourcebackup.log
  System-ID: 8e390e79-5acb-4ec5-a533-7de4f4e5e339
  Restic: noch nicht konfiguriert (SMB-Mount + Passwort offen)
```

---

## 8. Querschnittliche Konzepte

### Security

| Maßnahme | Status |
|---|---|
| Security Headers (6) | ✅ |
| CORS (configurable) | ✅ |
| SQL parametrisiert | ✅ |
| Token-Hashes SHA-256 | ✅ |
| Bearer Token Auth | ✅ |
| CASCADE delete Tokens | ✅ Migration 000009 |
| TLS / mTLS | ❌ B_TLS |
| Rate Limiting | ❌ offen |

### UI Design-Prinzipien

- Keine Panik-Texte — sachlich, präzise, überprüfbar
- Restore-Verifikation prominenter als Backup-Erfolg
- Rote Farbe nur für echte Fehler
- Aktions-Buttons mit Bestätigungs-Dialog bei destruktiven Aktionen

---

## 9. Architekturentscheidungen

| ADR | Entscheidung |
|---|---|
| ADR-001 | Restic als Standard Backup-Engine |
| ADR-002 | PostgreSQL als Katalog-Datenbank |
| ADR-003 | Go für Agent und Control Plane |

---

## 10. Qualitätsanforderungen

| Szenario | Maßnahme |
|---|---|
| System gelöscht | CASCADE räumt Tokens auf |
| Agent-Token revoked | Agent stoppt beim nächsten Poll (401) |
| Policy ohne Repository | Job schlägt fehl mit Erklärung |
| Pending Job stapelt sich | 🗑 Delete-Button in Jobs-GUI |

---

## 11. Risiken

| Risiko | Status |
|---|---|
| Kein TLS | ❌ B_TLS |
| Restore nicht implementiert | 📋 B13/B14 |
| gosec noch in Schicht 2 | 🔧 jetzt aktivierbar |

---

## 12. Glossar

| Begriff | Bedeutung |
|---|---|
| Control Plane | Zentraler Server: API, Scheduler, Katalog, Auth, Downloads |
| Agent | Go-Binary auf Zielsystem |
| Enrollment Token | OTP (30 Min) zum einmaligen Registrieren |
| Agent Token | Langlebiger Bearer-Token (data/agent-token) |
| Policy | Engine, Schedule, Includes/Excludes, Retention, Repository |
| Restore Tests | Automatische Snapshot-Verifikation (B13/B14) |
