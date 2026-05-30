const BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`)
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
  return res.json()
}

export interface System {
  ID: string
  Hostname: string
  OS?: string
  AgentVersion?: string
  LastSeen?: string
  RiskClass: string
  Tags?: Record<string, string>
  CreatedAt: string
}

export interface BackupRepository {
  ID: string
  Type: string
  Location: string
  EncryptionMode?: string
  ObjectLockEnabled: boolean
  CreatedAt: string
}

export interface BackupPolicy {
  ID: string
  Name: string
  Engine: string
  Schedule?: string
  Includes?: string[]
  Excludes?: string[]
  RepositoryID?: string
  CreatedAt: string
}

export interface BackupJob {
  ID: string
  SystemID: string
  PolicyID: string
  Status: string
  BytesUploaded?: number
  BytesScanned?: number
  StartedAt?: string
  FinishedAt?: string
  ErrorSummary?: string
  CreatedAt: string
}

export interface Snapshot {
  ID: string
  JobID: string
  RepositoryID: string
  EngineSnapshotID: string
  Hostname?: string
  Paths?: string[]
  ChecksumStatus: string
  CreatedAt: string
}

export const api = {
  health: () => get<{ status: string }>('/health'),
  systems: () => get<System[]>('/v1/systems'),
  repositories: () => get<BackupRepository[]>('/v1/repositories'),
  policies: () => get<BackupPolicy[]>('/v1/policies'),
  jobs: () => get<BackupJob[]>('/v1/jobs'),
  snapshots: () => get<Snapshot[]>('/v1/snapshots'),
}
