# Developer Guide

> Stand: B1–B16 — Control Plane, Agent, Web-UI, Auth, Downloads.

---

## Voraussetzungen

```bash
go 1.22+  ·  node 20+  ·  docker 24+  ·  git 2.40+  ·  restic
golangci-lint v2  →  make lint-install
golang-migrate    →  go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
goimports         →  go install golang.org/x/tools/cmd/goimports@latest
```

## Setup

```bash
git clone https://github.com/cerberus8484/opensourcebackup.git
cd opensourcebackup

make deps && make dev-up && make migrate-up && make run
# Control Plane → http://localhost:8080

cd web && npm install && npm run dev
# Web-UI → http://localhost:5173
```

## Projektstruktur

```
cmd/
  control-plane/main.go    Server + Scheduler + Auth + Middleware
  agent/main.go            Enrollment-Flow + Poll-Loop

internal/
  api/
    downloads.go           GET /downloads/agent/{version}/{platform}
    middleware.go          Recovery, CORS, SecurityHeaders, BodyLimit, Logging, Timeout
    agent_auth.go          Bearer Token → system_id im Context
    enrollment.go          Enrollment-Token + Enroll-Endpoint
    agent_jobs.go          /v1/agent/jobs start/complete/fail
    handler.go + routes.go
    systems/repositories/policies/jobs/snapshots.go
  auth/
    token.go               GenerateToken (256-bit), HashToken (SHA-256)
    enrollment_store.go    OTP: Create, Consume, Revoke
    agent_token_store.go   Bearer: Create, ValidateAndTouch, Revoke
  agent/
    agent.go               Poll-Loop, Job-Flow, 401→Abbruch
    tokenfile.go           SaveToken(0600), LoadToken
    client/client.go       HTTP-Client /v1/agent/*
    restic/runner.go       init + backup --json + parse
  catalog/                 5 Store-Interfaces + pgx
  scheduler/               Cron + Dead-Man's-Switch

web/src/
  pages/                   Dashboard, Systems, Agents, Policies, Jobs,
                           Snapshots, RestoreTests, Repositories
  components/              Sidebar, StatusBadge, Table, Modal, ConfirmDialog, Card

migrations/                000001–000009
dist/agent/                Pre-built Binaries (in .gitignore)
```

## Make-Targets

```bash
make deps                  go mod download
make test                  Unit-Tests
make test-integration      Integration-Tests (DATABASE_URL)
make lint                  Schicht 1 — blockiert
make lint-warn             Schicht 2 — informativ
make check                 fmt + lint + test
make run                   Control Plane
make run-agent             Agent
make dev-up / dev-down     Docker Compose
make migrate-up / down     Migrationen 000001–000009
make build-agent-windows   Windows AMD64 Binary → dist/
make build-agent-linux     Linux AMD64 Binary → dist/
make build-agent-all       Alle Platforms
make lint-install          golangci-lint v2
```

## Umgebungsvariablen

### Control Plane
```bash
DATABASE_URL=postgres://opensourcebackup:dev_password@localhost:5432/opensourcebackup?sslmode=disable
LISTEN_ADDR=:8080
CORS_ORIGIN=http://localhost:5173   # Web-UI Origin
```

### Agent
```bash
CONTROL_PLANE_URL=http://localhost:8080
RESTIC_PASSWORD=<passwort>
RESTIC_REPO=<pfad-oder-url>

# Token (eine Option):
AGENT_TOKEN=<token>              # direkter Token
AGENT_TOKEN_FILE=data/agent-token
ENROLLMENT_TOKEN=<token>         # einmaliges Enrollen
```

## Agent-Binary bauen

```bash
# Alle Platforms:
make build-agent-all

# Einzeln:
make build-agent-windows   # → dist/agent/v0.1.0/opensourcebackup-agent-windows-amd64.exe
make build-agent-linux     # → dist/agent/v0.1.0/opensourcebackup-agent-linux-amd64
```

Binaries werden über `GET /downloads/agent/v0.1.0/{platform}` ausgeliefert.

## Web-UI

```bash
cd web
npm install
npm run dev          # → http://localhost:5173
npm run build        # → web/dist/ für Produktion
```

## Tests (~70+)

| Paket | Typ | Tests |
|---|---|---|
| `internal/auth` | Unit + Integration | Token-Hashing, Enrollment, Agent-Token |
| `internal/api` | Unit | Handler, Middleware, AgentAuth, Modal-Actions |
| `internal/agent` | Unit | Poll, 401, transient errors |
| `internal/catalog` | Integration | Alle 5 Stores + B12 |
| `internal/scheduler` | Unit | Scheduler Start |

## Migrationen

| Nr | Inhalt |
|---|---|
| 000001–000005 | systems, repositories, backup_policies, backup_jobs, snapshots |
| 000006 | repository_id in backup_policies |
| 000007 | agent_enrollment_tokens |
| 000008 | agent_tokens |
| 000009 | CASCADE DELETE auf Token-Tabellen |

## Definition of Done

- [ ] `go build ./...` grün
- [ ] `make lint` — 0 Issues
- [ ] `make test` + `make test-integration` grün
- [ ] Tokens nie geloggt
- [ ] `CHANGELOG.md` aktualisiert
- [ ] Beide READMEs (EN + DE) synchron
- [ ] `make build-agent-all` nach Agent-Änderungen
