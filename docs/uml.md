# UML — OpensourceBackup Diagramme

> Aktuelle Diagramme in Mermaid-Syntax.
> Stand: B1–B9.7 implementiert.

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
        AUTH_MW[AgentAuth Middleware\nBearer Token → system_id]
        API[REST API\n/v1/systems · /v1/jobs · /v1/policies\n/v1/repositories · /v1/snapshots]
        AGENT_API[Agent API\n/v1/agent/jobs · start · complete · fail\n/v1/agent/enroll]
        SCHED[Scheduler\nCron + Dead-Man's Switch]
        CATALOG[Catalog Layer\n5 Stores]
        AUTH[Auth Layer\nEnrollmentTokenStore\nAgentTokenStore]
    end

    subgraph Storage
        PG[(PostgreSQL 16)]
        REDIS[(Redis)]
    end

    WEB -->|HTTPS REST| MW
    AGENT -->|Bearer Token| MW
    MW --> AUTH_MW
    AUTH_MW --> AGENT_API
    MW --> API
    API --> CATALOG
    AGENT_API --> CATALOG
    AGENT_API --> AUTH
    SCHED --> CATALOG
    CATALOG --> PG
    AUTH --> PG
    SCHED --> REDIS
    AGENT -->|restic backup| RESTIC[Restic Engine]
```

---

## 2. Klassendiagramm — Catalog Models

```mermaid
classDiagram
    class System {
        +UUID ID
        +string Hostname
        +string RiskClass
        +map Tags
        +time.Time CreatedAt
    }
    class BackupRepository {
        +UUID ID
        +string Type
        +string Location
        +bool ObjectLockEnabled
        +time.Time CreatedAt
    }
    class BackupPolicy {
        +UUID ID
        +string Name
        +string Engine
        +string Schedule
        +UUID RepositoryID
        +time.Time CreatedAt
    }
    class BackupJob {
        +UUID ID
        +UUID SystemID
        +UUID PolicyID
        +string Status
        +int64 BytesUploaded
        +time.Time CreatedAt
    }
    class Snapshot {
        +UUID ID
        +UUID JobID
        +UUID RepositoryID
        +string EngineSnapshotID
        +string ChecksumStatus
        +time.Time CreatedAt
    }
    System "1" --> "0..*" BackupJob
    BackupPolicy "1" --> "0..*" BackupJob
    BackupPolicy --> BackupRepository : RepositoryID
    BackupJob "1" --> "0..*" Snapshot
    BackupRepository "1" --> "0..*" Snapshot
```

---

## 3. Klassendiagramm — Auth Models

```mermaid
classDiagram
    class EnrollmentToken {
        +UUID ID
        +UUID SystemID
        +string TokenHash
        +time.Time ExpiresAt
        +time.Time UsedAt
        +time.Time RevokedAt
        +time.Time CreatedAt
    }
    class AgentToken {
        +UUID ID
        +UUID SystemID
        +string TokenHash
        +time.Time LastUsedAt
        +time.Time RevokedAt
        +time.Time CreatedAt
    }
    class EnrollmentTokenStore {
        <<interface>>
        +Create(systemID, hash, expiresAt) EnrollmentToken
        +Consume(hash) EnrollmentToken
        +Revoke(id) error
    }
    class AgentTokenStore {
        <<interface>>
        +Create(systemID, hash) AgentToken
        +ValidateAndTouch(hash) UUID
        +Revoke(id) error
    }
    EnrollmentTokenStore <|.. pgEnrollmentTokenStore
    AgentTokenStore <|.. pgAgentTokenStore
    EnrollmentToken --* EnrollmentTokenStore
    AgentToken --* AgentTokenStore
```

---

## 4. Klassendiagramm — Agent (DIP)

```mermaid
classDiagram
    class ControlPlaneClient {
        <<interface>>
        +ListPendingJobs(ctx) []BackupJob
        +GetPolicy(ctx, id) BackupPolicy
        +StartJob(ctx, id) error
        +CompleteJob(ctx, id, snapshotID, bytes, paths) error
        +FailJob(ctx, id, reason) error
    }
    class Client {
        -baseURL string
        -token string
        +ListPendingJobs()
        +StartJob()
        +CompleteJob()
        +FailJob()
        +Enroll(enrollmentToken) string
    }
    class Agent {
        -cfg Config
        -cp ControlPlaneClient
        -runner Runner
        +Run(ctx) error
    }
    class Runner {
        -bin string
        +Backup(opts) BackupResult
    }
    ControlPlaneClient <|.. Client
    Agent --> ControlPlaneClient
    Agent --> Runner
```

---

## 5. Sequenzdiagramm — Enrollment Flow

```mermaid
sequenceDiagram
    participant ADM as Admin
    participant CP as Control Plane
    participant DB as PostgreSQL
    participant A as Agent

    ADM->>CP: POST /v1/systems/{id}/enrollment-token
    CP->>CP: GenerateToken() → raw (43 chars)
    CP->>CP: HashToken(raw) → SHA-256
    CP->>DB: INSERT agent_enrollment_tokens (hash, expires+30min)
    CP-->>ADM: {"token": raw, "expires_at": "..."}
    Note over ADM,CP: raw Token nur einmal ausgegeben — nie loggen

    A->>CP: POST /v1/agent/enroll {"enrollment_token": raw}
    CP->>CP: HashToken(raw)
    CP->>DB: SELECT WHERE hash=? AND used_at IS NULL AND expires_at > NOW()
    DB-->>CP: EnrollmentToken
    CP->>DB: UPDATE used_at = NOW()
    CP->>CP: GenerateToken() → agentToken
    CP->>DB: INSERT agent_tokens (system_id, HashToken(agentToken))
    CP-->>A: {"token": agentToken, "system_id": "..."}
    A->>A: SaveToken("data/agent-token", agentToken, 0600)
```

---

## 6. Sequenzdiagramm — Gesicherter Backup-Flow

```mermaid
sequenceDiagram
    participant CR as Cron
    participant SC as Scheduler
    participant CP as Control Plane
    participant A as Agent
    participant R as Restic

    CR->>SC: Policy-Schedule auslösen
    SC->>CP: Create BackupJob{status=pending}

    loop alle 30s
        A->>CP: GET /v1/agent/jobs\nAuthorization: Bearer <token>
        CP->>CP: HashToken → ValidateAndTouch → system_id
        CP-->>A: [pending jobs für diese system_id]
    end

    A->>CP: PUT /v1/agent/jobs/{id}/start
    A->>R: restic init + restic backup --json
    R-->>A: {"message_type":"summary","snapshot_id":"abc","data_added":1234}
    A->>CP: PUT /v1/agent/jobs/{id}/complete\n{snapshot_id, bytes, paths}
    CP->>CP: Job=success + Snapshot erstellen\n(policy.repository_id → snapshot.repository_id)
```

---

## 7. Sequenzdiagramm — AgentAuth Middleware

```mermaid
sequenceDiagram
    participant A as Agent
    participant MW as AgentAuth
    participant DB as PostgreSQL
    participant H as Handler

    A->>MW: GET /v1/agent/jobs\nAuthorization: Bearer xyz
    MW->>MW: extractBearer("Bearer xyz") → "xyz"
    MW->>MW: HashToken("xyz") → hash
    MW->>DB: UPDATE agent_tokens SET last_used_at=NOW()\nWHERE hash=? AND revoked_at IS NULL\nRETURNING system_id
    DB-->>MW: system_id (or no rows → 401)
    MW->>H: r.WithContext(ctx + system_id)
    H->>DB: SELECT jobs WHERE system_id=$1 AND status='pending'
    DB-->>A: []BackupJob
```

---

## 8. Deploymentdiagramm — Entwicklung

```mermaid
graph LR
    subgraph Dev["Entwickler-Maschine"]
        CP[control-plane :8080]
        PG[(PostgreSQL :5432)]
        RD[(Redis :6379)]
    end
    subgraph Target["Zielsystem (Agent)"]
        AG[agent\nENROLLMENT_TOKEN=...\nREStic_REPO=s3:...]
    end
    CP -->|pgx| PG
    AG -->|Bearer Token| CP
    AG -->|restic| S3[(S3/MinIO)]
```

---

## 9. Deploymentdiagramm — Produktion (Ziel)

```mermaid
graph TB
    subgraph K8s["Kubernetes Cluster"]
        ING[Ingress TLS 1.3]
        CP1[control-plane Pod 1]
        CP2[control-plane Pod 2]
        PG[(PostgreSQL)]
        RD[(Redis)]
    end
    subgraph Agents["Zielsysteme"]
        A1[Agent 1\ndata/agent-token]
        A2[Agent 2\ndata/agent-token]
    end
    A1 -->|Bearer Token| ING
    A2 -->|Bearer Token| ING
    ING --> CP1 & CP2
    CP1 & CP2 --> PG & RD
```
