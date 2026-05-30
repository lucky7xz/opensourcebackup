const BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`)
  if (!res.ok) throw new Error(`${res.status}`)
  return res.json()
}

export interface System {
  ID: string; Hostname: string; OS?: string; AgentVersion?: string
  LastSeen?: string; RiskClass: string; Tags?: Record<string,string>; CreatedAt: string
}
export interface BackupRepository {
  ID: string; Type: string; Location: string
  EncryptionMode?: string; ObjectLockEnabled: boolean; CreatedAt: string
}
export interface BackupPolicy {
  ID: string; Name: string; Engine: string; Schedule?: string
  Includes?: string[]; Excludes?: string[]; RepositoryID?: string; CreatedAt: string
}
export interface BackupJob {
  ID: string; SystemID: string; PolicyID: string; Status: string
  BytesUploaded?: number; StartedAt?: string; FinishedAt?: string
  ErrorSummary?: string; CreatedAt: string
}
export interface Snapshot {
  ID: string; JobID: string; RepositoryID: string; EngineSnapshotID: string
  Hostname?: string; Paths?: string[]; ChecksumStatus: string; CreatedAt: string
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`${res.status}`)
  return res.json()
}

export const api = {
  health:       () => get<{status:string}>('/health'),
  systems:      () => get<System[]>('/v1/systems'),
  repositories: () => get<BackupRepository[]>('/v1/repositories'),
  policies:     () => get<BackupPolicy[]>('/v1/policies'),
  jobs:         () => get<BackupJob[]>('/v1/jobs'),
  snapshots:    () => get<Snapshot[]>('/v1/snapshots'),
  createJob:    (systemID: string, policyID: string) =>
    post<BackupJob>('/v1/jobs', { SystemID: systemID, PolicyID: policyID, Status: 'pending' }),
  createEnrollmentToken: (systemID: string) =>
    post<{token:string; expires_at:string}>(`/v1/systems/${systemID}/enrollment-token`, {}),
}

export function fmt(bytes?: number) {
  if (!bytes) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024**2) return `${(bytes/1024).toFixed(1)} KB`
  if (bytes < 1024**3) return `${(bytes/1024**2).toFixed(1)} MB`
  return `${(bytes/1024**3).toFixed(2)} GB`
}

export function timeAgo(iso?: string) {
  if (!iso) return 'never'
  const diff = Date.now() - new Date(iso).getTime()
  const m = Math.floor(diff/60000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m/60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h/24)}d ago`
}

export function duration(start?: string, end?: string) {
  if (!start || !end) return '—'
  const ms = new Date(end).getTime() - new Date(start).getTime()
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms/1000).toFixed(1)}s`
  const m = Math.floor(ms/60000); const s = Math.floor((ms%60000)/1000)
  return `${m}m ${s}s`
}
