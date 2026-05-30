# OpensourceBackup

> Open-Source Backup Control Plane — zentrales Orchestrierungs- und Verwaltungssystem für 100+ Systeme.

## Übersicht

OpensourceBackup ist kein neues Backup-Tool. Es ist eine **Backup Control Plane**:
eine Plattform, die bewährte Backup-Engines (Restic, Borg, pgBackRest, Velero) orchestriert,
zentral verwaltet, überwacht und Restore-Integrität automatisch verifiziert.

```
Quellen (Server, VMs, DBs, Endgeräte)
    ↓
Backup-Agent (Restic / Borg / pgBackRest / Velero)
    ↓
Control Plane (Scheduler, Katalog, API, Web-UI)
    ↓
Storage (MinIO / ZFS / S3 / GCS / Azure)
    ↓
Monitoring (Prometheus / Grafana / Alertmanager / Loki)
```

## Schnellstart

```bash
# Repository klonen
git clone https://github.com/your-org/opensourcebackup.git
cd opensourcebackup

# Lokale Entwicklungsumgebung starten
docker compose up -d

# Agent auf einem Zielsystem installieren
./scripts/agent-install.sh --server https://your-control-plane
```

## Dokumentation

| Dokument | Beschreibung |
|---|---|
| [Developer Guide](docs/developer-guide/DEVELOPER_GUIDE.md) | Einstieg, Setup, Prozesse, Regeln |
| [Clean Code & Wertesystem](docs/developer-guide/CLEAN_CODE.md) | Verbindliche Qualitätsprinzipien |
| [Architektur](docs/architecture/ARCHITECTURE.md) | Zielarchitektur, Komponenten, Entscheidungen |
| [Changelog](CHANGELOG.md) | Versionshistorie nach Keep a Changelog |
| [Contributing](CONTRIBUTING.md) | Wie man beiträgt |
| [ADR Index](docs/adr/README.md) | Architecture Decision Records |

## Projektstruktur

```
opensourcebackup/
├── agent/                  # Backup-Agent (Go)
│   ├── engines/            # Engine-Wrapper (Restic, Borg, pgBackRest, Velero)
│   ├── collectors/         # System-Inventar, Metriken
│   ├── metrics/            # Prometheus-Exporter
│   └── config/             # Agent-Konfiguration
├── server/                 # Control Plane (Go)
│   ├── api/                # REST API
│   ├── scheduler/          # Job-Scheduler
│   ├── catalog/            # PostgreSQL-Katalog
│   ├── auth/               # Authentifizierung / RBAC
│   └── policies/           # Policy-Engine
├── web/                    # Web-Dashboard (React / TypeScript)
│   ├── dashboard/
│   ├── systems/
│   ├── jobs/
│   ├── restores/
│   └── alerts/
├── deployments/            # Deployment-Konfigurationen
│   ├── docker-compose/
│   ├── helm/
│   └── ansible/
├── docs/                   # Dokumentation
│   ├── architecture/
│   ├── developer-guide/
│   ├── adr/
│   └── changelog/
└── scripts/                # Hilfsskripte
```

## Technologie-Stack

| Schicht | Technologie |
|---|---|
| Agent / Server | Go 1.22+ |
| Web-UI | React 18 + TypeScript 5 |
| Datenbank | PostgreSQL 16 |
| Message Queue | Redis Streams |
| Monitoring | Prometheus + Grafana + Loki |
| Container | Docker + Kubernetes (Helm) |
| IaC | Terraform + Ansible |
| Secrets | HashiCorp Vault / SOPS |
| Backup-Engines | Restic, Borg, pgBackRest, Velero |

## Lizenz

Apache 2.0 — siehe [LICENSE](LICENSE)
