# OpensourceBackup

> Open-Source Backup Control Plane — zentrales Orchestrierungs- und Verwaltungssystem für 100+ Systeme.

🇩🇪 Deutsch | 🇬🇧 [English](README.md)

---

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
git clone https://github.com/cerberus8484/opensourcebackup.git
cd opensourcebackup

# Lokale Entwicklungsumgebung starten
make dev-up

# Datenbank migrieren
make migrate-up

# Control Plane starten
make run
```

## Entwicklung

```bash
# Abhängigkeiten installieren
make deps

# Unit-Tests ausführen
make test

# Integration-Tests (benötigt laufende PostgreSQL)
make test-integration

# Lint (hart — blockiert bei Verletzung)
make lint

# Lint (Warnungen — zeigt Baustellen, blockiert nie)
make lint-warn

# Format + Lint + Test in einem
make check
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
├── cmd/
│   └── control-plane/      # Einstiegspunkt Control Plane
├── internal/
│   ├── api/                # HTTP REST API Handler + Middleware
│   ├── catalog/            # PostgreSQL Datenzugriffsschicht
│   └── scheduler/          # Cron-Scheduler + Dead-Man's-Switch
├── migrations/             # SQL-Migrationen (golang-migrate)
├── deployments/
│   └── docker-compose/     # Lokaler Dev-Stack (PostgreSQL + Redis)
└── docs/
    ├── architecture/       # Architekturdokumentation
    ├── developer-guide/    # Developer Guide + Clean-Code-Prinzipien
    ├── quality/            # Lint-Strategie
    └── adr/                # Architecture Decision Records
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
