// Shared types + helpers for the Cockpit. No data fetching, no side effects —
// pure derivations over data the Cockpit already loads from existing endpoints.
import type { BackupJob, System } from '../../api'

/** Aggregated live state for a single system, derived from its jobs. */
export interface SystemStatus {
  system:  System
  running: BackupJob | null  // currently running
  pending: BackupJob | null  // queued / waiting for agent
  lastJob: BackupJob | null  // most recent finished job (success/failed/cancelled)
}

const ONLINE_WINDOW_MS = 5 * 60_000 // a system seen within 5 min counts as online

/** True when the system reported in recently (LastSeen within the online window). */
export function isOnline(sys: System): boolean {
  if (!sys.LastSeen) return false
  return Date.now() - new Date(sys.LastSeen).getTime() < ONLINE_WINDOW_MS
}

/** Left-accent / status colour for a system card, per the cockpit colour rules. */
export function systemAccent(ss: SystemStatus): string {
  if (ss.running) return 'var(--success)'                 // green — backup running
  if (ss.pending) return 'var(--running)'                 // blue — waiting (info)
  if (ss.lastJob?.Status === 'failed') return 'var(--error)'
  if (ss.lastJob?.Status === 'success') return 'var(--success)'
  return 'var(--running)'                                  // never run yet → info
}

/** Throughput formatter (bytes/s → human readable). */
export function fmtBps(bps: number): string {
  if (bps < 1024)        return `${bps} B/s`
  if (bps < 1024 * 1024) return `${(bps / 1024).toFixed(1)} KB/s`
  return `${(bps / 1024 / 1024).toFixed(1)} MB/s`
}

/** Shorten a repository location for display without leaking the full path. */
export function friendlyLoc(loc: string): string {
  if (/^(s3|b2|azure|gs):/.test(loc)) return loc
  const parts = loc.replace(/\\/g, '/').split('/').filter(Boolean)
  return parts.length > 2 ? `…/${parts.slice(-2).join('/')}` : loc
}

/** Capitalise a job status for display ("success" → "Erfolgreich" mapping handled by caller). */
export function statusColor(status?: string): string {
  switch (status) {
    case 'success':   return 'var(--success)'
    case 'failed':    return 'var(--error)'
    case 'running':   return 'var(--success)'
    case 'pending':   return 'var(--running)'
    case 'cancelled': return 'var(--warning)'
    default:          return 'var(--text-dim)'
  }
}
