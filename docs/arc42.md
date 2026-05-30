# Arc42 — OpensourceBackup Architektur

> Architekturdokumentation nach arc42-Template.
> Stand: B1–B7 implementiert (Catalog, API, Scheduler, Security Middleware).

---

## 1. Einführung und Ziele

### Aufgabenstellung

OpensourceBackup ist eine **Backup Control Plane** — kein neues Backup-Tool, sondern eine
Plattform zur Orchestrierung, Verwaltung und Überwachung bewährter Backup-Engines
(Restic, Borg, pgBackRest, Velero) über 100+ Systeme hinweg.

### Wesentliche Ziele

| Priorität | Ziel |
|---|---|
| 1 | Zentrales Management von Backup-Jobs für heterogene Systeme |
| 2 | Automatische Verifikation der Restore-Integrität |
| 3 | Dead-Man's-Switch: Alert wenn erwartete Jobs ausbleiben |
| 4 | Nachvollziehbare Beweiskette: jedes Finding hat Belege |
| 5 | Betrieb auf Kubernetes und Docker Compose |

### Qualitätsziele

| Qualitätsmerkmal | Beschreibung |
|---|---|
| Korrektheit | Backup-Daten müssen zuverlässig verarbeitet werden |
| Sicherheit | mTLS, RBAC, keine Credentials im Code, TLS 1.3 |
| Wandelbarkeit | Clean Code, CCD-Prinzipien, testgetrieben |
| Betriebsbereitschaft | Graceful Shutdown, Health-Endpoint, Metrics |

---

## 2. Randbedingungen

### Technische Randbedingungen

| Randbedingung | Hintergrund |
|---|---|
| Go 1.22+ für Agent und Server | Einfache Cross-Compilation, Single Binary |
| PostgreSQL 16 als Katalog-DB | JSONB, UUID, Arrays nativ |
| Docker + Kubernetes Deployment | Standard-Betriebsumgebung |
| Apache 2.0 Lizenz | Open-Source |

### Organisatorische Randbedingungen

- Conventional Commits für automatisches Changelog
- Semantic Versioning
- Kein Merge ohne grüne CI (lint + tests)

---

## 3. Kontextabgrenzung

### Fachlicher Kontext

```
┌─────────────────────────────────────────────────────────┐
│                    OpensourceBackup                      │
│                                                          │
│  ┌──────────┐   REST API   ┌──────────────────────────┐ │
│  │ Web-GUI  │◄────────────►│     Control Plane        │ │
│  └──────────┘              │  (Scheduler, Catalog,    │ │
│                            │   API, Policy Engine)    │ │
│  ┌──────────┐   mTLS/HTTPS └────────────┬─────────────┘ │
│  │  Agent   │◄───────────────────────────┘               │
│  └──────────┘                                            │
└─────────────────────────────────────────────────────────┘
         │
         ▼ orchestriert
  Restic / Borg / pgBackRest / Velero
         │
         ▼ schreibt auf
  ZFS / MinIO / S3 / GCS / Azure
```

### Externe Schnittstellen

| System | Richtung | Protokoll |
|---|---|---|
| Backup-Agent | bidirektional | HTTPS + mTLS |
| Web-GUI | eingehend | HTTPS / REST |
| PostgreSQL | ausgehend | pgx/v5 |
| Redis | ausgehend | Redis Streams |
| Prometheus | eingehend (scrape) | HTTP |
| Alertmanager | ausgehend | HTTP Webhook |

---

## 4. Lösungsstrategie

| Entscheidung | Begründung |
|---|---|
| Go für Agent + Server | Single Binary, Cross-Compilation, gute Concurrency |
| PostgreSQL als Katalog | JSONB für flexible Metadaten, solide FK-Constraints |
| Restic als Standard-Engine | Deduplizierung, Verschlüsselung, aktive Community |
| REST API (stdlib net/http) | Kein Framework-Overhead, Go 1.22 Pattern-Routing |
| golang-migrate für Schema | Versionierte, rückwärtskompatible Migrationen |
| Interface-basierter DB-Zugriff | Testbarkeit mit Stubs, DIP-konform |

Detaillierte Entscheidungen: [ADR Index](adr/README.md)

---

## 5. Bausteinsicht

### Ebene 1 — Gesamtsystem

```
opensourcebackup/
├── cmd/control-plane/     # Einstiegspunkt: HTTP-Server + Scheduler
├── internal/
│   ├── api/               # HTTP-Handler, Middleware, Routing
│   ├── catalog/           # DB-Stores für alle 5 Entitäten
│   └── scheduler/         # Cron-Dispatcher + Dead-Man's-Switch
└── migrations/            # SQL-Migrationen (001–005)
```

### Ebene 2 — internal/api

| Baustein | Verantwortlichkeit |
|---|---|
| `handler.go` | Handler-Struct, DI, Hilfsfunktionen |
| `routes.go` | URL-Pattern → Handler-Mapping |
| `middleware.go` | Recovery, Logging, Timeout, SecurityHeaders, BodyLimit |
| `systems.go` | CRUD-Handler für `/v1/systems` |
| `repositories.go` | CRUD-Handler für `/v1/repositories` |
| `policies.go` | CRUD-Handler für `/v1/policies` |
| `jobs.go` | CRUD-Handler für `/v1/jobs` |
| `snapshots.go` | CRD-Handler für `/v1/snapshots` |

### Ebene 2 — internal/catalog

| Baustein | Verantwortlichkeit |
|---|---|
| `db.go` | pgxpool.Pool, Open/Ping/Close |
| `models.go` | System, BackupRepository, BackupPolicy, BackupJob, Snapshot |
| `errors.go` | ErrNotFound, ErrConflict |
| `systems.go` | SystemStore Interface + pgx-Implementierung |
| `repositories.go` | RepositoryStore Interface + pgx-Implementierung |
| `policies.go` | PolicyStore Interface + pgx-Implementierung |
| `jobs.go` | JobStore Interface + pgx-Implementierung |
| `snapshots.go` | SnapshotStore Interface + pgx-Implementierung |

### Ebene 2 — internal/scheduler

| Baustein | Verantwortlichkeit |
|---|---|
| `scheduler.go` | Cron-Einträge aus Policies laden, Jobs dispatchen |
| Dead-Man's-Switch | Ticker alle 5 Min: überfällige Jobs erkennen |

---

## 6. Laufzeitsicht

### Szenario: API-Request

```
HTTP Request
  → Recovery Middleware      (panic-safe)
  → SecurityHeaders          (6 HTTP Security Headers)
  → RequestBodyLimit (1 MB)  (413 bei Überschreitung)
  → Logging                  (method, path, status, duration)
  → Timeout (30s)            (503 bei Überschreitung)
  → Handler
      → Store.Method(ctx)
          → pgxpool.QueryRow / Exec
              → PostgreSQL
  ← Response (JSON)
```

### Szenario: Scheduled Job Dispatch

```
Cron-Ticker (Policy-Schedule)
  → Scheduler.dispatchJob()
      → JobStore.Create(ctx, &BackupJob{status: "pending"})
          → PostgreSQL INSERT

Ticker alle 5 Minuten
  → Scheduler.checkDeadMan()
      → JobStore.LatestByPolicyID(ctx, policyID)
      → Wenn last_job.created_at > now - interval*1.5
          → slog.Warn("dead-man: overdue job detected")
```

---

## 7. Verteilungssicht

### Entwicklung

```
Docker Compose (deployments/docker-compose/dev.yml)
  ├── postgres:16-alpine  → localhost:5432
  └── redis:7-alpine      → localhost:6379

Control Plane  → go run ./cmd/control-plane  → localhost:8080
```

### Produktion (Ziel)

```
Kubernetes Cluster
  ├── Deployment: control-plane  (2+ Replicas)
  ├── Service: LoadBalancer / Ingress (TLS termination)
  ├── StatefulSet: PostgreSQL 16
  ├── Deployment: Redis
  └── DaemonSet/Deployment: Prometheus + Grafana

Agent (auf Zielsystemen)
  └── systemd-Service / Docker Container → HTTPS → Ingress
```

---

## 8. Querschnittliche Konzepte

### Fehlerbehandlung

- Catalog-Fehler: `ErrNotFound`, `ErrConflict` — gemappt auf HTTP 404/409
- pgx-Fehler: nicht nach außen gegeben — nur intern geloggt
- Panics: Recovery-Middleware fängt alle ab → HTTP 500

### Logging

Strukturiertes JSON-Logging via `log/slog`:

```json
{"time":"...","level":"INFO","msg":"request","method":"GET","path":"/v1/systems","status":200,"duration_ms":3}
```

### Testing-Strategie

| Ebene | Tool | Scope |
|---|---|---|
| Unit (API) | `net/http/httptest` + Stubs | Handler-Logik ohne DB |
| Unit (Scheduler) | Stubs | Dispatch-Logik |
| Integration (Catalog) | echte PostgreSQL | Alle Store-Methoden |
| Build-Tag | `//go:build integration` | Trennung Unit/Integration |

### Security (aktueller Stand)

| Maßnahme | Status |
|---|---|
| Security Headers (6) | ✅ Middleware |
| Request Body Limit 1 MB | ✅ Middleware |
| SQL nur parametrisiert | ✅ pgx `$1, $2` |
| TLS / mTLS | ❌ B9 |
| Auth / RBAC | ❌ B9 |
| Rate Limiting | ❌ B9 |

---

## 9. Architekturentscheidungen

Vollständige Liste: [ADR Index](adr/README.md)

| ADR | Entscheidung |
|---|---|
| ADR-001 | Restic als Standard Backup-Engine |
| ADR-002 | PostgreSQL als Katalog-Datenbank |
| ADR-003 | Go für Agent und Control Plane |

---

## 10. Qualitätsanforderungen

| Szenario | Maßnahme |
|---|---|
| Backup-Job bleibt aus | Dead-Man's-Switch → Alert |
| DB-Verbindung bricht ab | pgxpool reconnect, Health-Endpoint gibt 503 |
| Panic im Handler | Recovery-Middleware → 500, keine Unterbrechung |
| Zu großer Request | Body-Limit → 413 |
| Langsamer Handler | Timeout-Middleware → 503 |
| Lint-Verletzung | CI blockiert Merge |

---

## 11. Risiken und technische Schulden

| Risiko | Maßnahme |
|---|---|
| Keine Auth (API offen) | ❌ Blockiert Produktion — B9 |
| Kein TLS | ❌ Blockiert Produktion |
| `SystemID` in Jobs ist nullable | Beim Agent-MVP füllen (B10) |
| noctx in Tests false-positive | In lint-warn, dokumentiert in lint-strategy.md |

---

## 12. Glossar

| Begriff | Bedeutung |
|---|---|
| Control Plane | Zentraler Server: API, Scheduler, Katalog |
| Agent | Go-Binary auf Zielsystem — führt Backup-Engines aus |
| Catalog | PostgreSQL-DB mit allen Backup-Entitäten |
| Policy | Konfiguration: Engine, Schedule, Retention, Include/Exclude |
| Dead-Man's Switch | Alert wenn erwarteter Job ausbleibt |
| Snapshot | Ergebnis eines erfolgreich ausgeführten Backup-Jobs |
| Restore Test | Automatische Verifikation eines Snapshots |
