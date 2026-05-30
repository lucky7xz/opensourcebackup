# Arc42 — OpensourceBackup Architektur

> Architekturdokumentation nach arc42-Template.
> Stand: B1–B9.7 implementiert (Catalog, API, Scheduler, Security Middleware, Auth, Agent, Restic Runner).

---

## 1. Einführung und Ziele

### Aufgabenstellung

OpensourceBackup ist eine **Backup Control Plane** — eine Plattform zur Orchestrierung,
Verwaltung und Überwachung bewährter Backup-Engines (Restic, Borg, pgBackRest, Velero)
über 100+ Systeme hinweg. Der Agent läuft auf Zielsystemen, führt Restic aus und meldet
Ergebnisse sicher über Bearer-Token-Authentifizierung zurück.

### Wesentliche Ziele

| Priorität | Ziel |
|---|---|
| 1 | Zentrales Management von Backup-Jobs für heterogene Systeme |
| 2 | Automatische Verifikation der Restore-Integrität |
| 3 | Dead-Man's-Switch: Alert wenn erwartete Jobs ausbleiben |
| 4 | Sicherer Agent-Flow: Enrollment → Bearer Token → geschützte Routen |
| 5 | Betrieb auf Kubernetes und Docker Compose |

### Qualitätsziele

| Qualitätsmerkmal | Beschreibung |
|---|---|
| Korrektheit | Backup-Daten zuverlässig verarbeiten, Repository-Verknüpfung vollständig |
| Sicherheit | Token-basierte Auth, SHA-256 Hashes, nie Tokens loggen, Security Headers |
| Wandelbarkeit | CCD-Prinzipien, Interface-basiert, testgetrieben |
| Betriebsbereitschaft | Graceful Shutdown, Health-Endpoint, Middleware-Chain |

---

## 2. Randbedingungen

| Randbedingung | Hintergrund |
|---|---|
| Go 1.22+ | Single Binary, Cross-Compilation, Concurrency |
| PostgreSQL 16 | JSONB, UUID, Arrays, FK-Constraints |
| Docker + Kubernetes | Standard-Betriebsumgebung |
| Apache 2.0 Lizenz | Open-Source |
| Conventional Commits | Automatisches Changelog |

---

## 3. Kontextabgrenzung

```
┌────────────────────────────────────────────────────────────┐
│                     OpensourceBackup                        │
│                                                             │
│  ┌──────────┐   REST API   ┌──────────────────────────────┐│
│  │ Web-GUI  │◄────────────►│        Control Plane         ││
│  └──────────┘              │  Scheduler · Catalog         ││
│                            │  API · Policy Engine         ││
│  ┌──────────┐   mTLS/HTTPS │  Auth · Agent Routes         ││
│  │  Agent   │◄────────────►└──────────────────────────────┘│
│  │ (Restic) │                                               │
│  └──────────┘                                               │
└────────────────────────────────────────────────────────────┘
         │ orchestriert
  Restic / Borg / pgBackRest / Velero
         │ schreibt auf
  ZFS / MinIO / S3 / GCS / Azure
```

### Externe Schnittstellen

| System | Richtung | Protokoll |
|---|---|---|
| Backup-Agent | bidirektional | HTTPS + Bearer Token |
| Web-GUI | eingehend | HTTPS / REST |
| PostgreSQL | ausgehend | pgx/v5 |
| Redis | ausgehend | Redis Streams |
| Prometheus | eingehend (scrape) | HTTP |

---

## 4. Lösungsstrategie

| Entscheidung | Begründung |
|---|---|
| Go für Agent + Server | Single Binary, Cross-Compilation |
| PostgreSQL als Katalog | JSONB für flexible Metadaten |
| Restic als Standard-Engine | Deduplizierung, Verschlüsselung, aktive Community |
| stdlib net/http | Kein Framework-Overhead, Go 1.22 Pattern-Routing |
| Interface-basierter DB-Zugriff | Testbarkeit mit Stubs, DIP-konform |
| SHA-256 Token-Hashes | Einfach, sicher für zufällige Tokens (256-bit Entropy) |
| Bearer-Token statt mTLS (MVP) | Einfacher Einstieg; mTLS kommt in B16 |

---

## 5. Bausteinsicht

### Ebene 1 — Gesamtsystem

```
opensourcebackup/
├── cmd/
│   ├── control-plane/   # HTTP-Server + Scheduler + Auth
│   └── agent/           # Backup-Agent mit Enrollment-Flow
├── internal/
│   ├── api/             # HTTP-Handler, Middleware, Agent-Routen
│   ├── auth/            # Token-Hashing, Enrollment Store, Agent Token Store
│   ├── agent/           # Poll-Loop, ControlPlaneClient Interface, Restic Runner
│   │   ├── client/      # HTTP-Client für /v1/agent/* Routen
│   │   └── restic/      # Restic-Wrapper: init, backup --json, parse summary
│   ├── catalog/         # DB-Stores für alle 5 Entitäten
│   ├── scheduler/       # Cron-Dispatcher + Dead-Man's-Switch
│   └── selfbackup/      # (Stub — kommt in B_SB)
└── migrations/          # SQL 000001–000008
```

### Ebene 2 — internal/api

| Baustein | Verantwortlichkeit |
|---|---|
| `middleware.go` | Recovery, SecurityHeaders, BodyLimit, Logging, Timeout, Chain |
| `agent_auth.go` | AgentAuth-Middleware: Bearer-Token → system_id im Context |
| `enrollment.go` | POST /v1/systems/{id}/enrollment-token, POST /v1/agent/enroll |
| `agent_jobs.go` | GET /v1/agent/jobs, start/complete/fail — system_id-Isolation |
| `handler.go` | Handler-Struct mit allen Stores inkl. EnrollmentTokenStore + AgentTokenStore |
| `routes.go` | URL-Routing: öffentlich + /v1/agent/* (AgentAuth-geschützt) |

### Ebene 2 — internal/auth

| Baustein | Verantwortlichkeit |
|---|---|
| `token.go` | GenerateToken (32 random bytes, base64url), HashToken (SHA-256) |
| `enrollment_store.go` | OTP-Enrollment: Create, Consume (Einmalnutzung + Ablauf), Revoke |
| `agent_token_store.go` | Langlebige Bearer-Tokens: Create, ValidateAndTouch, Revoke |

### Ebene 2 — internal/agent

| Baustein | Verantwortlichkeit |
|---|---|
| `agent.go` | Poll-Loop, Job-Ausführung, 401→Abbruch, transient→weiter |
| `client/client.go` | HTTP-Client: ListPendingJobs, StartJob, CompleteJob, FailJob, Enroll |
| `restic/runner.go` | Restic-Wrapper: init repo, backup --json, Summary parsen |
| `tokenfile.go` | SaveToken (0600), LoadToken für persistente Token-Datei |

### Ebene 2 — internal/catalog

| Baustein | Stores |
|---|---|
| Systems, Repositories, Policies | CRUD + List |
| Jobs | CRUD + ListBySystemID + ListPendingBySystemID + LatestByPolicyID |
| Snapshots | CR + List + ListByJobID + Delete |

---

## 6. Laufzeitsicht

### Szenario: API-Request

```
HTTP Request
  → Recovery → SecurityHeaders → BodyLimit(1MB) → Logging → Timeout(30s)
  → Handler → Store → PostgreSQL
  ← JSON Response + Security Headers
```

### Szenario: Agent-Request (authenticated)

```
HTTP Request + Authorization: Bearer <token>
  → Recovery → SecurityHeaders → BodyLimit → Logging → Timeout
  → AgentAuth Middleware:
      HashToken(raw) → AgentTokenStore.ValidateAndTouch()
      → system_id in Context injizieren
  → AgentJobHandler:
      system_id aus Context → nur eigene Jobs sichtbar
      fremde system_id → 404 (kein Info-Leak)
```

### Szenario: Agent Enrollment

```
Admin: POST /v1/systems/{id}/enrollment-token
  → GenerateToken() → HashToken() → DB speichern (expires 30 Min)
  → raw Token EINMAL zurückgeben (nie loggen)

Agent: POST /v1/agent/enroll {"enrollment_token": "..."}
  → HashToken() → DB suchen
  → prüfen: nicht abgelaufen, nicht benutzt, nicht revoked
  → used_at setzen (OTP-Semantik)
  → neuen Agent-Token erzeugen → Hash speichern
  → raw Agent-Token EINMAL zurückgeben
  → Token in data/agent-token (0600) speichern
```

### Szenario: Scheduled Backup Flow

```
Cron-Trigger (Policy-Schedule)
  → Scheduler.dispatchJob() → BackupJob{status=pending} in DB

Agent pollt GET /v1/agent/jobs (alle 30s)
  → pending Job gefunden
  → PUT /v1/agent/jobs/{id}/start
  → restic init + restic backup --json
  → PUT /v1/agent/jobs/{id}/complete {snapshot_id, bytes, paths}
      → Control Plane: Job=success + Snapshot registriert
         (mit policy.repository_id → snapshot.repository_id)
```

### Szenario: Dead-Man's-Switch

```
Ticker alle 5 Minuten
  → für jede scheduled Policy:
      → cronInterval() → erwartetes Intervall
      → LatestByPolicyID() → letzter Job
      → wenn älter als Intervall × 1.5 → slog.Warn
```

---

## 7. Verteilungssicht

### Entwicklung

```
make dev-up      → PostgreSQL:5432 + Redis:6379 (Docker)
make migrate-up  → Schema 000001–000008
make run         → Control Plane :8080

# Agent auf Zielsystem:
CONTROL_PLANE_URL=http://... ENROLLMENT_TOKEN=... make run-agent
```

### Produktion (Ziel)

```
Kubernetes: Ingress (TLS) → control-plane (2+ Pods) → PostgreSQL StatefulSet
Agent → Bearer Token → Ingress → Control Plane
```

---

## 8. Querschnittliche Konzepte

### Security (aktueller Stand)

| Maßnahme | Status |
|---|---|
| Security Headers (6) | ✅ Middleware |
| Request Body Limit 1 MB | ✅ Middleware |
| SQL nur parametrisiert | ✅ pgx $1, $2 |
| Token-Hashes (SHA-256) | ✅ nie Klartext in DB |
| Tokens nie geloggt | ✅ explizite Regel im Code |
| Bearer Token Auth | ✅ /v1/agent/* geschützt |
| Agent sieht nur eigene Jobs | ✅ system_id Isolation |
| OTP Enrollment Token | ✅ Einmalnutzung, TTL 30 Min |
| Token Revocation | ✅ beide Token-Typen |
| TLS / mTLS | ❌ B16 |
| Rate Limiting | ❌ offen |
| Audit-Logging | 🔧 Basis vorhanden |

### Fehlerbehandlung

- `ErrNotFound` → 404, `ErrConflict` → 409
- `ErrBodyTooLarge` → 413
- `ErrUnauthorized` (client) → Agent stoppt, re-enrollment nötig
- Panics → Recovery-Middleware → 500

### Testing

| Ebene | Tool | Scope |
|---|---|---|
| Unit (API) | httptest + Stubs | Handler, Middleware, Auth |
| Unit (Agent) | Stubs | Poll-Loop, Job-Flow, 401-Handling |
| Unit (Auth) | direkt | GenerateToken, HashToken |
| Integration (Catalog) | echte PostgreSQL | Alle Store-Methoden |
| Integration (Auth) | echte PostgreSQL | Enrollment + Agent-Token Stores |

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
| Job bleibt aus | Dead-Man's-Switch → Warn-Log |
| Token revoked | Agent stoppt sofort, loggt Fehler |
| Fremder Job-Zugriff | 404 (kein 403 — kein Info-Leak) |
| Policy ohne Repository | Job schlägt fehl mit Erklärung |
| DB bricht ab | pgxpool reconnect, /health → 503 |
| Zu großer Request | BodyLimit → 413 |

---

## 11. Risiken und technische Schulden

| Risiko | Status | Maßnahme |
|---|---|---|
| Kein TLS | ❌ Produktion blockiert | B16 |
| Rate Limiting fehlt | ❌ | offen |
| Restore noch nicht implementiert | 📋 | B13/B14 |
| gosec noch in Schicht 2 | 🔧 | jetzt hochziehbar |
| noctx false-positive in Tests | 🔧 warn | dokumentiert |

---

## 12. Glossar

| Begriff | Bedeutung |
|---|---|
| Control Plane | Zentraler Server: API, Scheduler, Katalog, Auth |
| Agent | Go-Binary auf Zielsystem — enrollt sich, führt Restic aus |
| Enrollment Token | Einmal-Token (30 Min TTL) zum Registrieren eines Agents |
| Agent Token | Langlebiger Bearer-Token für den laufenden Agent |
| Catalog | PostgreSQL-DB mit allen Backup-Entitäten |
| Policy | Engine, Schedule, Retention, Repository-Verknüpfung |
| Dead-Man's Switch | Alert wenn erwarteter Job ausbleibt |
| Snapshot | Ergebnis eines erfolgreichen Backup-Jobs |
