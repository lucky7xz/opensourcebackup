# UML — OpensourceBackup Diagramme

> Aktuelle Diagramme in Mermaid-Syntax.
> Stand: B1–B7 implementiert.

---

## 1. Komponentendiagramm — Gesamtsystem

```mermaid
graph TB
    subgraph Client
        WEB[Web-GUI\nReact/TypeScript]
        AGENT[Backup-Agent\nGo Binary]
    end

    subgraph ControlPlane["Control Plane (Go)"]
        MW[Middleware\nRecovery · Headers · Limit · Logging · Timeout]
        API[REST API\n/v1/systems · /v1/jobs · /v1/policies\n/v1/repositories · /v1/snapshots]
        SCHED[Scheduler\nCron + Dead-Man's Switch]
        CATALOG[Catalog Layer\nSystemStore · JobStore · PolicyStore\nRepositoryStore · SnapshotStore]
    end

    subgraph Storage
        PG[(PostgreSQL 16\nCatalog DB)]
        REDIS[(Redis\nMessage Queue)]
    end

    subgraph BackupEngines["Backup Engines (auf Agent-System)"]
        RESTIC[Restic]
        BORG[Borg]
        PGBR[pgBackRest]
        VELERO[Velero]
    end

    WEB -->|HTTPS REST| MW
    AGENT -->|HTTPS mTLS| MW
    MW --> API
    API --> CATALOG
    SCHED --> CATALOG
    CATALOG --> PG
    SCHED --> REDIS
    AGENT --> RESTIC
    AGENT --> BORG
    AGENT --> PGBR
    AGENT --> VELERO
```

---

## 2. Klassendiagramm — Catalog Models

```mermaid
classDiagram
    class System {
        +UUID ID
        +string Hostname
        +string OS
        +string AgentVersion
        +time.Time LastSeen
        +map Tags
        +string RiskClass
        +time.Time CreatedAt
    }

    class BackupRepository {
        +UUID ID
        +string Type
        +string Location
        +string EncryptionMode
        +bool ObjectLockEnabled
        +UUID RetentionPolicyID
        +time.Time CreatedAt
    }

    class BackupPolicy {
        +UUID ID
        +string Name
        +[]string Includes
        +[]string Excludes
        +string Schedule
        +map Retention
        +string Engine
        +[]string PreHooks
        +[]string PostHooks
        +time.Time CreatedAt
    }

    class BackupJob {
        +UUID ID
        +UUID SystemID
        +UUID PolicyID
        +time.Time StartedAt
        +time.Time FinishedAt
        +string Status
        +int64 BytesScanned
        +int64 BytesUploaded
        +string ErrorSummary
        +map RawOutput
        +time.Time CreatedAt
    }

    class Snapshot {
        +UUID ID
        +UUID JobID
        +string EngineSnapshotID
        +UUID RepositoryID
        +time.Time CreatedAt
        +string Hostname
        +[]string Paths
        +string ChecksumStatus
    }

    System "1" --> "0..*" BackupJob : has
    BackupPolicy "1" --> "0..*" BackupJob : triggers
    BackupJob "1" --> "0..*" Snapshot : produces
    BackupRepository "1" --> "0..*" Snapshot : stores
```

---

## 3. Klassendiagramm — Store Interfaces (DIP)

```mermaid
classDiagram
    class SystemStore {
        <<interface>>
        +Create(ctx, System) error
        +GetByID(ctx, UUID) System
        +List(ctx) []System
        +Update(ctx, System) error
        +Delete(ctx, UUID) error
    }

    class JobStore {
        <<interface>>
        +Create(ctx, BackupJob) error
        +GetByID(ctx, UUID) BackupJob
        +List(ctx) []BackupJob
        +ListBySystemID(ctx, UUID) []BackupJob
        +LatestByPolicyID(ctx, UUID) BackupJob
        +Update(ctx, BackupJob) error
        +Delete(ctx, UUID) error
    }

    class pgSystemStore {
        -db DB
        +Create()
        +GetByID()
        +List()
        +Update()
        +Delete()
    }

    class Handler {
        -systems SystemStore
        -repositories RepositoryStore
        -policies PolicyStore
        -jobs JobStore
        -snapshots SnapshotStore
        -log Logger
    }

    class Scheduler {
        -policies PolicyStore
        -jobs JobStore
        -cron Cron
        -log Logger
        +Start(ctx) error
    }

    SystemStore <|.. pgSystemStore : implements
    Handler --> SystemStore : depends on
    Handler --> JobStore : depends on
    Scheduler --> JobStore : depends on
```

---

## 4. Sequenzdiagramm — API Request (mit Middleware-Chain)

```mermaid
sequenceDiagram
    participant C as Client
    participant REC as Recovery
    participant SEC as SecurityHeaders
    participant LIM as BodyLimit
    participant LOG as Logging
    participant TO as Timeout
    participant H as Handler
    participant S as Store
    participant DB as PostgreSQL

    C->>REC: HTTP Request
    REC->>SEC: next
    SEC->>SEC: Set 6 Security Headers
    SEC->>LIM: next
    LIM->>LIM: MaxBytesReader(1MB)
    LIM->>LOG: next
    LOG->>TO: next (start timer)
    TO->>H: next
    H->>S: Store.GetByID(ctx, id)
    S->>DB: SELECT ... WHERE id = $1
    DB-->>S: Row
    S-->>H: *Model / ErrNotFound
    H-->>TO: Response 200 / 404
    TO-->>LOG: Response
    LOG->>LOG: log method/path/status/duration
    LOG-->>SEC: Response
    SEC-->>C: Response + Security Headers
```

---

## 5. Sequenzdiagramm — Scheduled Job Dispatch

```mermaid
sequenceDiagram
    participant CR as Cron
    participant SC as Scheduler
    participant JS as JobStore
    participant DB as PostgreSQL
    participant TK as Ticker (5min)

    Note over CR: Policy-Schedule auslösen
    CR->>SC: dispatchJob(policy)
    SC->>JS: Create(ctx, BackupJob{status: "pending"})
    JS->>DB: INSERT INTO backup_jobs
    DB-->>JS: id, created_at
    JS-->>SC: job.ID gesetzt

    Note over TK: Alle 5 Minuten
    TK->>SC: checkDeadMan(policies)
    loop für jede scheduled Policy
        SC->>SC: cronInterval(schedule)
        SC->>JS: LatestByPolicyID(ctx, policyID)
        JS->>DB: SELECT ... ORDER BY created_at DESC LIMIT 1
        DB-->>JS: BackupJob / ErrNotFound
        alt Job überfällig
            SC->>SC: slog.Warn("dead-man: overdue")
        end
    end
```

---

## 6. Deploymentdiagramm — Entwicklung

```mermaid
graph LR
    subgraph Developer["Entwickler-Maschine"]
        CP[control-plane\n:8080]
        PG[(PostgreSQL\n:5432)]
        RD[(Redis\n:6379)]
    end

    CP -->|pgx/v5| PG
    CP -->|redis| RD

    DEV[make dev-up\nmake migrate-up\nmake run] --> Developer
```

---

## 7. Deploymentdiagramm — Produktion (Ziel)

```mermaid
graph TB
    subgraph Internet
        CLI[Agent / Web-GUI]
    end

    subgraph K8s["Kubernetes Cluster"]
        ING[Ingress\nTLS 1.3]
        CP1[control-plane Pod 1]
        CP2[control-plane Pod 2]
        PG[(PostgreSQL\nStatefulSet)]
        RD[(Redis\nDeployment)]
        PROM[Prometheus\nGrafana\nLoki]
    end

    subgraph Agent["Zielsysteme"]
        A1[Agent 1\nsystemd]
        A2[Agent 2\nDocker]
    end

    CLI -->|HTTPS| ING
    A1 -->|mTLS| ING
    A2 -->|mTLS| ING
    ING --> CP1
    ING --> CP2
    CP1 --> PG
    CP2 --> PG
    CP1 --> RD
    CP2 --> RD
    PROM -.->|scrape| CP1
    PROM -.->|scrape| CP2
```
