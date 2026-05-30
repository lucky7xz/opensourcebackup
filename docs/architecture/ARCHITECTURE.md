# Architektur

> Zielarchitektur des OpensourceBackup-Projekts.
> Letzte Aktualisierung: siehe Git-Historie.

---

## Leitprinzip

OpensourceBackup baut **keine** neue Backup-Engine.
Das Projekt ist eine **Backup Control Plane**: Orchestrierung, Verwaltung, Monitoring
und Restore-Verifikation — auf Basis bewährter Engines.

```
╔══════════════════════════════════════════════════════╗
║              OpensourceBackup Control Plane          ║
║   Scheduler │ Katalog │ API │ Web-UI │ Policy Engine ║
╚══════════════════════╤═══════════════════════════════╝
                       │ orchestriert
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
   Restic          pgBackRest      Velero
 (Dateien)        (PostgreSQL)  (Kubernetes)
        │              │              │
        └──────────────┴──────────────┘
                       │ schreibt auf
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
  ZFS / NAS         MinIO          S3 / GCS
  (lokal)      (Object Store)      (Cloud)
```

---

## Komponenten

### Backup-Agent

Läuft auf jedem zu sichernden System.

**Verantwortlichkeiten:**
- Engine-Wrapper ausführen (Restic, Borg, pgBackRest, Velero)
- Job-Status und Metriken an Control Plane melden
- Restore-Tests lokal durchführen
- System-Inventar (OS, Hostname, Tags) melden

**Technologie:** Go — Single Binary, kross-kompiliert für Linux/Windows (amd64, arm64)

**Kommunikation:** HTTPS + mTLS → Control Plane API

---

### Control Plane

Zentraler Server. Kann hochverfügbar betrieben werden.

#### API (`server/api/`)
- REST-API für Agent-Kommunikation und Web-UI
- Authentifizierung: JWT + mTLS für Agenten, OAuth2/OIDC für Benutzer
- Versioniert: `/v1/...`

#### Scheduler (`server/scheduler/`)
- Cron-basierte Job-Planung pro Policy
- Dead-Man's-Switch: Alert wenn erwarteter Job ausbleibt
- Retry-Logik mit exponential backoff

#### Policy Engine (`server/policies/`)
- Retention-Policies (daily/weekly/monthly/yearly)
- Include/Exclude-Regeln
- Pre/Post-Hooks
- Risk-Class-basierte Backup-Frequenz

#### Katalog (`server/catalog/`)
- PostgreSQL 16
- Speichert: Systeme, Repositories, Jobs, Snapshots, Restore-Tests, Audit-Log
- Migrations-Tool: `golang-migrate`

---

### Storage

#### Lokal
- **ZFS/NAS/SAN:** Schnelle Restores, Snapshots, lokale Kontrolle
- **MinIO:** S3-kompatibler Object Store, selbstgehostet
- **Object Lock:** WORM-Schutz gegen Ransomware (MinIO + S3)

#### Cloud
- S3, GCS, Azure Blob — über Restic-Native-Backends
- Verschlüsselung bleibt beim Agenten — kein Vertrauen in Cloud-Storage

#### 3-2-1-Regel (Empfehlung)
3 Kopien, 2 verschiedene Medien, 1 offsite.

---

### Monitoring

| Komponente | Zweck |
|---|---|
| Prometheus | Metriken-Sammlung |
| Grafana | Dashboards |
| Alertmanager | Alert-Routing (E-Mail, Slack, PagerDuty) |
| Loki | Zentrales Log-Aggregat |

**Kern-Metriken:**

```
backup_job_last_success_timestamp{system, policy}
backup_job_duration_seconds{system, policy}
backup_job_bytes_uploaded{system, policy}
backup_snapshot_count{system, repository}
backup_restore_test_last_success_timestamp{system, snapshot}
backup_jobs_failed_total{system, policy, reason}
```

**Dead-Man's-Switch:** Alert wenn `backup_job_last_success_timestamp` älter als
`expected_interval * 1.5` für eine Policy mit definiertem Schedule.

---

## Datenbankschema (Kern)

```sql
-- Zu sichernde Systeme
systems (
  id UUID PRIMARY KEY,
  hostname TEXT NOT NULL,
  os TEXT,
  agent_version TEXT,
  last_seen TIMESTAMPTZ,
  tags JSONB,
  risk_class TEXT DEFAULT 'standard'
)

-- Backup-Repositories (Ziele)
repositories (
  id UUID PRIMARY KEY,
  type TEXT NOT NULL,           -- 'restic', 'borg'
  location TEXT NOT NULL,       -- s3:bucket/path, sftp:host:/path
  encryption_mode TEXT,
  object_lock_enabled BOOLEAN DEFAULT false,
  retention_policy_id UUID
)

-- Backup-Policies
backup_policies (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  includes TEXT[],
  excludes TEXT[],
  schedule TEXT,                -- cron expression
  retention JSONB,              -- {daily: 7, weekly: 4, monthly: 12}
  engine TEXT NOT NULL,         -- 'restic', 'borg', 'pgbackrest', 'velero'
  pre_hooks TEXT[],
  post_hooks TEXT[]
)

-- Ausgeführte Backup-Jobs
backup_jobs (
  id UUID PRIMARY KEY,
  system_id UUID REFERENCES systems,
  policy_id UUID REFERENCES backup_policies,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  status TEXT,                  -- 'running', 'success', 'failed', 'warning'
  bytes_scanned BIGINT,
  bytes_uploaded BIGINT,
  error_summary TEXT,
  raw_output JSONB
)

-- Engine-Snapshots
snapshots (
  id UUID PRIMARY KEY,
  job_id UUID REFERENCES backup_jobs,
  engine_snapshot_id TEXT NOT NULL,  -- Restic/Borg Snapshot-ID
  repository_id UUID REFERENCES repositories,
  created_at TIMESTAMPTZ,
  hostname TEXT,
  paths TEXT[],
  checksum_status TEXT           -- 'verified', 'unverified', 'failed'
)

-- Restore-Tests
restore_tests (
  id UUID PRIMARY KEY,
  snapshot_id UUID REFERENCES snapshots,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  status TEXT,                  -- 'success', 'failed'
  target TEXT,                  -- Sandbox-Ziel
  verified_files INTEGER,
  error_summary TEXT
)
```

---

## Sicherheitsarchitektur

```
Internet / Agenten
      │
      │ HTTPS + mTLS (TLS 1.3)
      ▼
  Load Balancer / Ingress
      │
      ▼
  Control Plane API
      │
      ├── JWT-Validierung (Benutzer)
      ├── mTLS-Validierung (Agenten)
      │
      ▼
  PostgreSQL (verschlüsselt at rest)
      │
  Vault / SOPS (Secrets)
```

**Prinzipien:**
- Keine eigene Kryptografie — nur Go-Stdlib und bewährte Libraries
- Agenten-Credentials: kurzlebige JWTs, pro System
- Repository-Credentials: per System/Mandant getrennt, in Vault gespeichert
- Audit-Log: unveränderlich, jede Policy-Änderung und jeder Restore-Zugriff geloggt
- Restore-Rechte: getrennt von Backup-Rechten (RBAC)

---

## Deployment

### Entwicklung
```bash
docker compose -f deployments/docker-compose/dev.yml up -d
```

### Produktion (Kubernetes)
```bash
helm install opensourcebackup deployments/helm/opensourcebackup \
  --values values.production.yaml
```

### Agenten-Verteilung
```bash
# Ansible-Role für Masseninstallation
ansible-playbook deployments/ansible/agent-install.yml \
  -i inventory/production.yml \
  -e "control_plane_url=https://backup.your-org.com"
```
