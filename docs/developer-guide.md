# Developer Guide

> Verbindlicher Leitfaden für alle Entwickler am OpensourceBackup-Projekt.
> Vollständige Regeln: [DEVELOPER_GUIDE.md](developer-guide/DEVELOPER_GUIDE.md)

---

## Voraussetzungen

```bash
go 1.22+
docker 24+ / docker compose 2.20+
git 2.40+
golangci-lint v2 (make lint-install)
golang-migrate   (go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest)
goimports        (go install golang.org/x/tools/cmd/goimports@latest)
```

## Setup

```bash
git clone https://github.com/cerberus8484/opensourcebackup.git
cd opensourcebackup

# Abhängigkeiten
make deps

# Dev-Stack starten (PostgreSQL + Redis)
make dev-up

# Datenbank migrieren
make migrate-up

# Control Plane starten
make run
# → http://localhost:8080/health → {"status":"ok"}
```

## Projektstruktur

```
opensourcebackup/
├── cmd/
│   └── control-plane/main.go   # Einstiegspunkt: Server + Scheduler + Middleware
├── internal/
│   ├── api/                    # HTTP REST API
│   │   ├── doc.go              # Package-Kommentar
│   │   ├── handler.go          # Handler-Struct, DI, decode/writeJSON/ErrBodyTooLarge
│   │   ├── middleware.go       # Recovery, SecurityHeaders, BodyLimit, Logging, Timeout, Chain
│   │   ├── routes.go           # URL-Routing /health + /v1/...
│   │   ├── systems.go          # CRUD /v1/systems
│   │   ├── repositories.go     # CRUD /v1/repositories
│   │   ├── policies.go         # CRUD /v1/policies
│   │   ├── jobs.go             # CRUD /v1/jobs
│   │   └── snapshots.go        # CRD  /v1/snapshots
│   ├── catalog/                # Datenbankschicht
│   │   ├── db.go               # pgxpool.Pool, Open/Ping/Close
│   │   ├── models.go           # System, BackupRepository, BackupPolicy, BackupJob, Snapshot
│   │   ├── errors.go           # ErrNotFound, ErrConflict
│   │   ├── systems.go          # SystemStore
│   │   ├── repositories.go     # RepositoryStore
│   │   ├── policies.go         # PolicyStore
│   │   ├── jobs.go             # JobStore
│   │   └── snapshots.go        # SnapshotStore
│   └── scheduler/
│       └── scheduler.go        # Cron-Dispatcher + Dead-Man's-Switch
├── migrations/
│   ├── 000001_create_systems.{up,down}.sql
│   ├── 000002_create_repositories.{up,down}.sql
│   ├── 000003_create_backup_policies.{up,down}.sql
│   ├── 000004_create_backup_jobs.{up,down}.sql
│   └── 000005_create_snapshots.{up,down}.sql
├── deployments/
│   └── docker-compose/dev.yml  # PostgreSQL 16 + Redis 7
├── docs/
│   ├── arc42.md / arc42.html
│   ├── uml.md / uml.html
│   ├── developer-guide.md / developer-guide.html
│   ├── USER_GUIDE.md / USER_GUIDE.html
│   ├── architecture/ARCHITECTURE.md
│   ├── developer-guide/DEVELOPER_GUIDE.md
│   ├── developer-guide/CLEAN_CODE.md
│   ├── quality/lint-strategy.md
│   └── adr/
├── .golangci.hard.yml          # Lint Schicht 1 — blockiert CI
├── .golangci.warn.yml          # Lint Schicht 2 — zeigt Baustellen
├── .github/workflows/ci.yml    # CI: Test + Lint
├── Makefile
└── go.mod
```

## Make-Targets

```bash
make deps           # go mod download
make test           # Unit-Tests
make test-integration  # Integration-Tests gegen PostgreSQL (DATABASE_URL nötig)
make lint           # golangci-lint Schicht 1 — blockiert
make lint-warn      # golangci-lint Schicht 2 — informativ
make fmt            # gofmt + goimports
make check          # fmt + lint + test
make run            # Control Plane starten
make dev-up         # Docker Compose starten
make dev-down       # Docker Compose stoppen
make migrate-up     # Alle Migrationen ausführen
make migrate-down   # Alle Migrationen rückgängig
make migrate-status # Aktuelle Migration-Version
make lint-install   # golangci-lint v2 installieren
```

## Umgebungsvariablen

```bash
# Pflicht
DATABASE_URL=postgres://opensourcebackup:dev_password@localhost:5432/opensourcebackup?sslmode=disable

# Optional
LISTEN_ADDR=:8080       # Standard: :8080
REDIS_URL=redis://localhost:6379
```

Vorlage: `.env.example` — niemals `.env.local` committen.

## Branching & Commits

**GitHub Flow** — ein `main`-Branch, kurzlebige Feature-Branches:

```
feature/OB-123-kurzbeschreibung
fix/OB-456-beschreibung
docs/OB-789-beschreibung
```

**Conventional Commits:**

```
feat(catalog): add SystemStore with CRUD operations
fix(api): return 404 on missing system instead of 500
docs(adr): add ADR-004 for scheduler design
chore(lint): promote revive to hard layer
```

## Tests schreiben

```go
// Unit-Test (kein Build-Tag nötig) — mit Stubs
func TestCreateSystem_Returns201_WithID(t *testing.T) { ... }

// Integration-Test — Build-Tag + DATABASE_URL
//go:build integration
func TestSystemStore_Create_AssignsIDAndCreatedAt(t *testing.T) { ... }
```

```bash
# Nur Unit-Tests
go test ./...

# Integration-Tests
DATABASE_URL=... go test -tags=integration ./internal/catalog/...
```

## Definition of Done

- [ ] `go build ./...` — kein Fehler
- [ ] `make lint` — 0 Issues
- [ ] `make test` — alle grün
- [ ] Integration-Tests für neue DB-Stores
- [ ] `CHANGELOG.md` aktualisiert (bei feat/fix)
- [ ] Beide READMEs (EN + DE) aktualisiert wenn nötig
- [ ] ADR erstellt wenn Architekturentscheidung getroffen wurde

## Coding-Regeln (Kurzfassung)

Vollständige Regeln: [CLEAN_CODE.md](developer-guide/CLEAN_CODE.md)

- **DIP**: Handler hängen von Interfaces ab, nicht von pgx-Implementierungen
- **SRP**: Eine Datei — eine Verantwortlichkeit
- **IOSP**: `main.go` orchestriert, `catalog/` führt aus
- **KISS**: stdlib `net/http` statt Router-Framework
- **YAGNI**: Keine vorauseilenden Abstraktionen

## CI/CD

`.github/workflows/ci.yml` läuft bei jedem Push/PR:

```
1. Test     — Unit + Integration gegen PostgreSQL
2. Lint     — golangci-lint hard (blockiert Merge)
3. Lint-Warn — golangci-lint warn (non-blocking, informativ)
```
