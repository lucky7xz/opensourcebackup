# OpensourceBackup

> Open-Source Backup Control Plane mit nachweisbarer Wiederherstellbarkeit

> **Backups zu erstellen ist einfach. Wiederherstellbarkeit zu beweisen ist der Unterschied.**

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

# Entwicklungsumgebung starten
make dev-up && make migrate-up && make run
# → http://localhost:8080/health → {"status":"ok"}
```

## Agent auf einem Zielsystem installieren

```bash
# 1. System in der Control Plane anlegen
curl -X POST http://localhost:8080/v1/systems \
  -d '{"Hostname":"mein-server","RiskClass":"standard"}'

# 2. Einmal-Token erzeugen (gilt 30 Min)
curl -X POST http://localhost:8080/v1/systems/{id}/enrollment-token

# 3. Agent starten — enrollt sich automatisch
CONTROL_PLANE_URL=http://localhost:8080 \
ENROLLMENT_TOKEN=<token> \
RESTIC_PASSWORD=<geheimnis> \
RESTIC_REPO=s3:mein-bucket/backups \
./agent
```

## Web-Dashboard

```bash
cd web && npm install && npm run dev
# → http://localhost:5173
```

**Dashboard zeigt:** Geschützte Systeme, Job-Status, Restore-Verifikation, aktuelle Fehler.

## Entwicklung

```bash
make deps              # Abhängigkeiten herunterladen
make test              # Unit-Tests
make test-integration  # Integration-Tests (PostgreSQL nötig)
make lint              # Hart — blockiert bei Verletzung
make lint-warn         # Weich — informativ
make check             # fmt + lint + test
make run               # Control Plane starten
make run-agent         # Backup-Agent starten
make build-agent-all   # Agent-Binaries für alle Platforms bauen
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
