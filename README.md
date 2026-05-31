# OpensourceBackup

> Open-Source Backup Control Plane with Restore Assurance

🇩🇪 [Deutsche Version](README.de.md) | 🇬🇧 English

> **Creating backups is easy. Proving recoverability is the difference.**

---

## Overview

OpensourceBackup orchestrates Restic, Borg, pgBackRest, and Velero across 100+ systems —
and proves that your data can actually be restored.

The central question answered every day:
**Are my systems protected — and has a restore been successfully tested?**

```
System → Policy → Agent → Backup → Snapshot → Restore Test → Evidence
```

Not just "Backup success ✅" — but **Restore verified ✅, files checked ✅, bytes validated ✅**.

```
Sources (Servers, VMs, DBs, Endpoints)
    ↓
Backup Agent (Restic / Borg / pgBackRest / Velero)
    ↓
Control Plane (Scheduler, Catalog, API, Web-UI)
    ↓
Storage (MinIO / ZFS / S3 / GCS / Azure)
    ↓
Monitoring (Prometheus / Grafana / Alertmanager / Loki)
```

## Quick Start

```bash
# Clone repository
git clone https://github.com/cerberus8484/opensourcebackup.git
cd opensourcebackup

# Start local development environment
make dev-up && make migrate-up && make run
# → http://localhost:8080/health → {"status":"ok"}
```

## Install Agent on a Target System

```bash
# 1. Create a system in the control plane
curl -X POST http://localhost:8080/v1/systems \
  -d '{"Hostname":"my-server","RiskClass":"standard"}'

# 2. Generate a one-time enrollment token (valid 30 min)
curl -X POST http://localhost:8080/v1/systems/{id}/enrollment-token

# 3. Start agent — enrolls automatically, saves token to data/agent-token
CONTROL_PLANE_URL=http://localhost:8080 \
ENROLLMENT_TOKEN=<token> \
RESTIC_PASSWORD=<secret> \
RESTIC_REPO=s3:my-bucket/backups \
./agent
```

## Web Dashboard

```bash
cd web && npm install && npm run dev
# → http://localhost:5173
```

**Dashboard shows:** Protected systems, job status, restore verification status, recent failures.

## Development

```bash
make deps              # Download dependencies
make test              # Unit tests
make test-integration  # Integration tests (requires PostgreSQL)
make lint              # Hard lint (blocks on violation)
make lint-warn         # Soft lint (informational)
make check             # fmt + lint + test
make run               # Start control plane
make run-agent         # Start backup agent
make build-agent-all   # Build agent binaries for all platforms
```

## Documentation

| Document | Description |
|---|---|
| [Developer Guide](docs/developer-guide/DEVELOPER_GUIDE.md) | Setup, workflow, processes, rules |
| [Clean Code & Values](docs/developer-guide/CLEAN_CODE.md) | Mandatory quality principles |
| [Architecture](docs/architecture/ARCHITECTURE.md) | Target architecture, components, decisions |
| [Changelog](CHANGELOG.md) | Version history following Keep a Changelog |
| [Contributing](CONTRIBUTING.md) | How to contribute |
| [ADR Index](docs/adr/README.md) | Architecture Decision Records |

## Project Structure

```
opensourcebackup/
├── cmd/
│   ├── control-plane/      # Control Plane: API + Scheduler + Auth + Downloads
│   └── agent/              # Backup Agent: enrollment + poll + restic runner
├── internal/
│   ├── api/                # REST API, Middleware, AgentAuth, Downloads
│   ├── auth/               # Token hashing, Enrollment Store, Agent Token Store
│   ├── agent/              # Poll loop, Restic runner, TokenFile
│   ├── catalog/            # PostgreSQL data access (5 stores)
│   └── scheduler/          # Cron scheduler + dead-man's switch
├── web/                    # React 18 + TypeScript + Vite dashboard
│   └── src/pages/          # Dashboard, Systems, Agents, Policies, Jobs, Snapshots
├── migrations/             # SQL 000001–000009
├── deployments/
│   └── docker-compose/     # Local dev stack (PostgreSQL + Redis)
└── docs/                   # arc42, UML, Developer Guide, User Guide, ADRs
```

## Technology Stack

| Layer | Technology |
|---|---|
| Agent / Server | Go 1.22+ |
| Web-UI | React 18 + TypeScript 5 |
| Database | PostgreSQL 16 |
| Message Queue | Redis Streams |
| Monitoring | Prometheus + Grafana + Loki |
| Container | Docker + Kubernetes (Helm) |
| IaC | Terraform + Ansible |
| Secrets | HashiCorp Vault / SOPS |
| Backup Engines | Restic, Borg, pgBackRest, Velero |

## License

Apache 2.0 — see [LICENSE](LICENSE)
