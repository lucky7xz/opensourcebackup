# OpenSourceBackup

> **Creating backups is easy. Proving recoverability is the difference.**

🇩🇪 [Deutsche Version](README.de.md) | 🇬🇧 English

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.1.0-blue)](CHANGELOG.md)

---

OpenSourceBackup is an open-source **Backup Control Plane** that orchestrates backup agents across your systems — Windows, Linux, FreeBSD, and NAS. It tracks every backup, runs automated restore tests, and gives you a single dashboard to **prove your data is actually recoverable**.

> **Not a backup engine.** OpenSourceBackup orchestrates [Restic](https://restic.net), Borg, pgBackRest, and Velero. You bring the storage — OpenSourceBackup brings the control.

---

## What it does

```
Agent (on your systems)
  └── backs up files/DBs → Repository (NAS / S3 / local)
        └── reports to Control Plane
              └── Web Dashboard shows health, jobs, snapshots
                    └── Restore Tests prove recoverability
```

- 📦 **Backup orchestration** — schedule policies, run jobs, track results across 100+ systems
- 🔄 **Restore verification** — automated restore tests with file-count and size validation
- 🖥 **Multi-platform agents** — Windows, Linux x64/ARM64, FreeBSD/OPNsense — all as system services
- 📊 **Single dashboard** — health overview, live job progress, snapshot history
- 🔒 **Security-first** — bcrypt auth, CSRF protection, audit log, GDPR export/purge

---

## Quick Start

### Option A — Windows (one command)

```powershell
# PowerShell as Administrator
$env:RESTIC_PASSWORD="your-backup-password"
$env:RESTIC_REPO="C:\Backups"
irm https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-local.ps1 | iex
```

Installs Control Plane + Agent + PostgreSQL + Redis as Windows Services. Opens the dashboard automatically.

### Option B — Proxmox (auto-creates LXC container)

```bash
# On the Proxmox host as root
bash <(curl -fsSL https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-proxmox.sh)
```

Automatically finds a free container ID (starting at 200), downloads the Debian 12 template, creates and starts the LXC, and installs everything inside it. Prints the dashboard URL at the end.

### Option C — Linux server / LXC manually

```bash
curl -fsSL https://raw.githubusercontent.com/cerberus8484/opensourcebackup/main/scripts/install-server.sh | bash
```

### After installation

1. Open `http://<your-ip>:8080/ui/`
2. Set `ADMIN_PASSWORD` in `/etc/opensourcebackup/server.env` and restart
3. Go to **Agents → + Enroll Agent** → follow the wizard
4. Create a **Repository** (where backups go)
5. Create a **Policy** (what to back up, when)
6. Run your first **Job** and watch the live progress

---

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                   Web Dashboard                       │
│            React 18 + TypeScript + Vite              │
└──────────────────────┬───────────────────────────────┘
                       │ HTTP REST + session cookie
┌──────────────────────▼───────────────────────────────┐
│                  Control Plane (Go 1.22+)             │
│  ┌─────────────┐  ┌───────────┐  ┌─────────────────┐ │
│  │  Scheduler  │  │  REST API │  │ Auth / Audit    │ │
│  │  (cron)     │  │  /v1/*    │  │ bcrypt · CSRF   │ │
│  └─────────────┘  └───────────┘  └─────────────────┘ │
└──────────────┬──────────────────────────┬────────────┘
               │                          │
┌──────────────▼──────────┐  ┌────────────▼────────────┐
│     PostgreSQL 16        │  │        Redis 7           │
│  Catalog · Audit (RLS)  │  │                          │
└─────────────────────────┘  └─────────────────────────┘
               │
               │ Bearer Token (SHA-256 hash)
┌──────────────▼──────────────────────────────────────┐
│                    Agent                             │
│  Windows Service / systemd / rc.d (FreeBSD)         │
│  ┌──────────────────────────────────────────────┐   │
│  │               Restic Runner                  │   │
│  │  backup → repository (NAS / S3 / local)     │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

---

## Platform Support

### Agent platforms

| Platform | Service | Installer |
|---|---|---|
| Windows x64 | Windows Service | `install-agent.ps1` / MSI / EXE |
| Linux x64 | systemd | `install-agent.sh` |
| Linux ARM64 | systemd | `install-agent.sh` |
| FreeBSD x64 / OPNsense | rc.d | `install-agent-freebsd.sh` |

### Repository types

| Type | Example |
|---|---|
| Local path | `/var/backups`, `C:\Backups` |
| NAS / SMB | `Z:\OpenSourceBackup`, Synology via CIFS |
| NAS / NFS | `/mnt/nas/backups`, QNAP |
| MinIO / S3 | Self-hosted MinIO, AWS S3, Azure Blob, B2 |
| Proxmox Storage | `/mnt/pve/Backup`, `/mnt/pve/NAS` |
| Borg via SSH | `user@host:./backups` |
| pgBackRest | PostgreSQL WAL archiving, PITR |
| Velero | Kubernetes deployments + volumes |

---

## Security

OpenSourceBackup implements **technical controls supporting GDPR-compliant operation**.
Actual GDPR compliance requires legal basis, processes, and documentation by the operator.

| Control | Implementation |
|---|---|
| Authentication | bcrypt admin password (cost 12), 8h sessions |
| Session security | HttpOnly + SameSite=Strict cookies |
| Brute-force | 5 attempts/min per hashed IP |
| CSRF | Double-Submit Cookie (`X-CSRF-Token`) |
| Transport | TLS via `TLS_CERT_FILE` + `TLS_KEY_FILE` |
| Backup encryption | Restic AES-256-CTR + Poly1305 (client-side) |
| Token storage | SHA-256 hashes only — never plaintext |
| Audit log | Append-only, IP hashed, PostgreSQL RLS |
| GDPR Art. 20 | `GET /v1/gdpr/systems/{id}/export` |
| GDPR Art. 17 | `DELETE /v1/gdpr/systems/{id}/purge` |
| Security headers | CSP, HSTS, X-Frame-Options, Permissions-Policy |

→ Full details: [SECURITY.md](SECURITY.md)

---

## Configuration

### Control Plane

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | — | PostgreSQL DSN (**required**) |
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `ADMIN_PASSWORD` | — | Dashboard password (empty = no auth, **dev only**) |
| `CORS_ORIGIN` | `http://localhost:5173` | Allowed CORS origin |
| `TLS_CERT_FILE` | — | TLS certificate (enables HTTPS) |
| `TLS_KEY_FILE` | — | TLS private key |
| `WEB_UI_DIR` | — | Path to web UI `dist/` directory |

### Agent

| Variable | Description |
|---|---|
| `CONTROL_PLANE_URL` | Control Plane URL (**required**) |
| `RESTIC_PASSWORD` | Backup encryption password (**required**) |
| `RESTIC_REPO` | Backup destination (**required**) |
| `ENROLLMENT_TOKEN` | One-time token (first run only) |
| `AGENT_POLL_INTERVAL` | Poll interval (default: `30s`) |
| `AGENT_TOKEN_FILE` | Saved token path (default: `data/agent-token`) |
| `RESTORE_TEST_ROOT` | Sandbox for restore tests |
| `RESTIC_BIN` | Path to restic binary |

### Agent commands

```bash
opensourcebackup-agent install    # Register as system service
opensourcebackup-agent start      # Start service
opensourcebackup-agent stop       # Stop service
opensourcebackup-agent restart    # Restart service
opensourcebackup-agent status     # Show service status
opensourcebackup-agent uninstall  # Remove service
opensourcebackup-agent            # Run interactively (dev/debug)
```

---

## Development

### Prerequisites

- Go 1.22+
- Docker Desktop (PostgreSQL + Redis)
- Node.js 20 LTS (web UI)

### Local setup

```bash
git clone https://github.com/cerberus8484/opensourcebackup.git
cd opensourcebackup

# Start database stack
make dev-up

# Run migrations
make migrate-up

# Start server (http://localhost:8080)
make run

# Web UI dev server (http://localhost:5173, with HMR)
cd web && npm install && npm run dev
```

### Tests

```bash
make test                # unit tests
make test-integration    # requires running PostgreSQL
make lint                # hard rules (blocks CI)
make lint-warn           # soft rules (informational)
```

### Build

```bash
make build-all                    # all binaries (agent + server, all platforms)
make build-agent-all              # agent only
make build-agent-windows          # Windows agent
make build-agent-linux            # Linux x64 agent
make build-agent-freebsd          # FreeBSD agent
.\scripts\build-release.ps1       # full release: binaries + MSI + EXE + checksums
```

---

## Roadmap

| # | Feature | Status |
|---|---|---|
| B_RBAC | Login UI + Admin/Operator/Auditor roles | 🔜 Next |
| B_RET | Retention policies + automatic prune | 🔜 Planned |
| B15 | Prometheus metrics endpoint | 📋 Planned |
| R-01 | Remove CSP `unsafe-inline` | 📋 Backlog |
| R-03 | TLS enforcement flag | 📋 Backlog |

---

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for the full history.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[Apache-2.0](LICENSE) — © 2026 cerberus8484

---

*OpenSourceBackup is not a backup engine. It orchestrates your existing tools and proves your data is recoverable.*
