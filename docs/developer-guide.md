# Developer Guide

> Verbindlicher Leitfaden für alle Entwickler am OpensourceBackup-Projekt.
> Vollständige Regeln: [DEVELOPER_GUIDE.md](developer-guide/DEVELOPER_GUIDE.md)

---

## Voraussetzungen

```bash
go 1.22+
docker 24+ / docker compose 2.20+
git 2.40+
restic                 # auf Zielsystemen für Agent-Betrieb
golangci-lint v2       # make lint-install
golang-migrate         # go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
goimports              # go install golang.org/x/tools/cmd/goimports@latest
```

## Setup

```bash
git clone https://github.com/cerberus8484/opensourcebackup.git
cd opensourcebackup

make deps          # go mod download
make dev-up        # PostgreSQL + Redis (Docker)
make migrate-up    # Schema 000001–000008
make run           # → http://localhost:8080/health → {"status":"ok"}
```

## Projektstruktur

```
opensourcebackup/
├── cmd/
│   ├── control-plane/main.go   # Server + Scheduler + Auth + Middleware
│   └── agent/main.go           # Agent: Enrollment-Flow + Poll-Loop
├── internal/
│   ├── api/
│   │   ├── handler.go          # Handler-Struct, DI, decode, ErrBodyTooLarge
│   │   ├── middleware.go       # Recovery, SecurityHeaders, BodyLimit, Logging, Timeout
│   │   ├── agent_auth.go       # AgentAuth-Middleware, SystemIDFromContext
│   │   ├── enrollment.go       # POST /enrollment-token, POST /agent/enroll
│   │   ├── agent_jobs.go       # GET/start/complete/fail /v1/agent/jobs/*
│   │   ├── routes.go           # Routing: öffentlich + /v1/agent/* (geschützt)
│   │   ├── systems.go          # CRUD /v1/systems
│   │   ├── repositories.go     # CRUD /v1/repositories
│   │   ├── policies.go         # CRUD /v1/policies
│   │   ├── jobs.go             # CRUD /v1/jobs (inkl. ?system_id=&status=pending)
│   │   └── snapshots.go        # CRD /v1/snapshots
│   ├── auth/
│   │   ├── token.go            # GenerateToken (256-bit), HashToken (SHA-256)
│   │   ├── enrollment_store.go # OTP: Create, Consume, Revoke
│   │   └── agent_token_store.go# Bearer: Create, ValidateAndTouch, Revoke
│   ├── agent/
│   │   ├── agent.go            # Poll-Loop, Job-Flow, 401→Abbruch
│   │   ├── tokenfile.go        # SaveToken(0600), LoadToken
│   │   ├── client/client.go    # HTTP-Client /v1/agent/* mit Bearer-Token
│   │   └── restic/runner.go    # Restic-Wrapper: init, backup --json, parse
│   ├── catalog/
│   │   ├── models.go           # System, BackupRepository, BackupPolicy (mit RepositoryID),
│   │   │                       # BackupJob, Snapshot
│   │   ├── errors.go           # ErrNotFound, ErrConflict
│   │   └── *.go                # 5 Store-Interfaces + pgx-Implementierungen
│   └── scheduler/
│       └── scheduler.go        # Cron-Dispatcher + Dead-Man's-Switch
├── migrations/
│   ├── 000001–000005           # systems, repositories, backup_policies, jobs, snapshots
│   ├── 000006                  # repository_id in backup_policies
│   ├── 000007                  # agent_enrollment_tokens
│   └── 000008                  # agent_tokens
├── .golangci.hard.yml          # Lint Schicht 1 — blockiert CI
├── .golangci.warn.yml          # Lint Schicht 2 — informativ
├── .github/workflows/ci.yml
└── Makefile
```

## Make-Targets

```bash
make deps            # go mod download
make test            # Unit-Tests
make test-integration# Integration-Tests (DATABASE_URL nötig)
make lint            # Schicht 1 — blockiert
make lint-warn       # Schicht 2 — informativ
make fmt             # gofmt + goimports
make check           # fmt + lint + test
make run             # Control Plane
make run-agent       # Agent (CONTROL_PLANE_URL + Token nötig)
make dev-up          # Docker Compose starten
make dev-down        # Docker Compose stoppen
make migrate-up      # Migrationen ausführen (inkl. 000006–000008)
make migrate-down    # Migrationen rückgängig
make lint-install    # golangci-lint v2 installieren
```

## Umgebungsvariablen

### Control Plane

```bash
DATABASE_URL=postgres://opensourcebackup:dev_password@localhost:5432/opensourcebackup?sslmode=disable
LISTEN_ADDR=:8080
```

### Agent (eine Token-Option wählen)

```bash
CONTROL_PLANE_URL=http://localhost:8080
RESTIC_PASSWORD=geheimes-passwort
RESTIC_REPO=s3:mein-bucket/backups/system-name
RESTIC_BIN=restic              # optional, Default: restic im PATH
AGENT_POLL_INTERVAL=30s        # optional, Default: 30s

# Token-Priorität:
AGENT_TOKEN=<token>            # 1. direkter Token
AGENT_TOKEN_FILE=data/agent-token  # 2. Token aus Datei
ENROLLMENT_TOKEN=<token>       # 3. Einmal-Token zum Enrollen
```

## Agent-Enrollment (einmalig)

```bash
# 1. Admin erzeugt Enrollment-Token für ein System
curl -X POST http://localhost:8080/v1/systems/{system_id}/enrollment-token
# → {"token": "Xk3mNp...", "expires_at": "...+30min"}

# 2. Agent enrollt sich (token wird in data/agent-token gespeichert)
CONTROL_PLANE_URL=http://localhost:8080 \
ENROLLMENT_TOKEN=Xk3mNp... \
RESTIC_PASSWORD=secret \
RESTIC_REPO=s3:bucket/backups \
make run-agent
```

## Tests schreiben

```go
// Unit-Test — kein Build-Tag, kein Docker
func TestAgent_FailsJob_WhenPolicyHasNoRepository(t *testing.T) { ... }

// Integration-Test — Build-Tag + DATABASE_URL
//go:build integration
func TestEnrollmentTokenStore_Consume_RejectsExpired(t *testing.T) { ... }
```

## Test-Übersicht (~70 Tests)

| Paket | Typ | Inhalt |
|---|---|---|
| `internal/auth` | Unit | GenerateToken, HashToken (5 Tests) |
| `internal/auth` | Integration | Enrollment + Agent-Token Stores (7 Tests) |
| `internal/api` | Unit | Handler, Middleware, AgentAuth (16+ Tests) |
| `internal/agent` | Unit | Poll-Loop, 401-Handling, Transient-Error (4 Tests) |
| `internal/catalog` | Integration | Alle 5 Stores, B12-Tests (50+ Tests) |
| `internal/scheduler` | Unit | Scheduler Start (2 Tests) |

## Definition of Done

- [ ] `go build ./...` — kein Fehler
- [ ] `make lint` — 0 Issues
- [ ] `make test` + `make test-integration` — alle grün
- [ ] Tokens werden nie geloggt (Code-Review-Pflicht)
- [ ] `CHANGELOG.md` aktualisiert (feat/fix)
- [ ] Beide READMEs (EN + DE) synchron halten
- [ ] Keine Credentials im Code

## Coding-Regeln

| Prinzip | Beispiel im Projekt |
|---|---|
| DIP | `ControlPlaneClient` Interface — Agent kennt keinen konkreten HTTP-Client |
| SRP | `auth/token.go` — nur Token-Hashing, sonst nichts |
| IOSP | `agent.go` orchestriert, `restic/runner.go` führt aus |
| KISS | stdlib net/http, keine Router-Frameworks |
| YAGNI | Keine DTOs bis sie wirklich gebraucht werden |
| Security | Tokens nie loggen — `// Never log this` im Code |
