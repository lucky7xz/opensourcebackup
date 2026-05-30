# OpensourceBackup

> Open-Source Backup Control Plane — central orchestration and management system for 100+ systems.

🇩🇪 [Deutsche Version](README.de.md) | 🇬🇧 English

---

## Overview

OpensourceBackup is not a new backup tool. It is a **Backup Control Plane**:
a platform that orchestrates proven backup engines (Restic, Borg, pgBackRest, Velero),
manages them centrally, monitors their status, and automatically verifies restore integrity.

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
make dev-up

# Run database migrations
make migrate-up

# Start control plane
make run
```

## Development

```bash
# Install dependencies
make deps

# Run unit tests
make test

# Run integration tests (requires running PostgreSQL)
make test-integration

# Lint (hard — blocks on violation)
make lint

# Lint (warn — shows issues, never blocks)
make lint-warn

# Format + lint + test in one
make check
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
│   └── control-plane/      # Control Plane entry point
├── internal/
│   ├── api/                # HTTP REST API handlers + middleware
│   ├── catalog/            # PostgreSQL data access layer
│   └── scheduler/          # Cron scheduler + dead-man's switch
├── migrations/             # SQL migrations (golang-migrate)
├── deployments/
│   └── docker-compose/     # Local dev stack (PostgreSQL + Redis)
└── docs/
    ├── architecture/       # Architecture documentation
    ├── developer-guide/    # Developer guide + clean code principles
    ├── quality/            # Lint strategy
    └── adr/                # Architecture Decision Records
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
