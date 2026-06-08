import { describe, it, expect } from 'vitest'
import type { BackupJob, System } from '../../api'
import { fmtBps, friendlyLoc, isOnline, statusColor, systemAccent, type SystemStatus } from './util'

function sys(over: Partial<System> = {}): System {
  return { ID: 's1', Hostname: 'host', RiskClass: 'standard', CreatedAt: '2026-01-01T00:00:00Z', ...over }
}
function job(status: string): BackupJob {
  return { ID: 'j', SystemID: 's1', PolicyID: 'p', Status: status, CreatedAt: '2026-01-01T00:00:00Z' }
}
function status(over: Partial<SystemStatus>): SystemStatus {
  return { system: sys(), running: null, pending: null, lastJob: null, ...over }
}

describe('isOnline', () => {
  it('is false without a LastSeen', () => {
    expect(isOnline(sys())).toBe(false)
  })
  it('is true when seen within the 5-minute window', () => {
    expect(isOnline(sys({ LastSeen: new Date(Date.now() - 60_000).toISOString() }))).toBe(true)
  })
  it('is false when last seen too long ago', () => {
    expect(isOnline(sys({ LastSeen: new Date(Date.now() - 10 * 60_000).toISOString() }))).toBe(false)
  })
})

describe('systemAccent', () => {
  it('is success (green) while running', () => {
    expect(systemAccent(status({ running: job('running') }))).toBe('var(--success)')
  })
  it('is running (blue) while pending', () => {
    expect(systemAccent(status({ pending: job('pending') }))).toBe('var(--running)')
  })
  it('is error (red) when the last job failed', () => {
    expect(systemAccent(status({ lastJob: job('failed') }))).toBe('var(--error)')
  })
  it('is success (green) when the last job succeeded', () => {
    expect(systemAccent(status({ lastJob: job('success') }))).toBe('var(--success)')
  })
  it('is info (blue) when nothing has run yet', () => {
    expect(systemAccent(status({}))).toBe('var(--running)')
  })
  it('prioritises a running job over a failed history', () => {
    expect(systemAccent(status({ running: job('running'), lastJob: job('failed') }))).toBe('var(--success)')
  })
})

describe('fmtBps', () => {
  it('formats bytes per second', () => {
    expect(fmtBps(512)).toBe('512 B/s')
    expect(fmtBps(2048)).toBe('2.0 KB/s')
    expect(fmtBps(5 * 1024 * 1024)).toBe('5.0 MB/s')
  })
})

describe('friendlyLoc', () => {
  it('keeps cloud URIs intact', () => {
    expect(friendlyLoc('s3:bucket/path')).toBe('s3:bucket/path')
  })
  it('shortens deep windows UNC paths to the last two segments', () => {
    expect(friendlyLoc('\\\\192.168.0.32\\Public\\OpenSourceBackup')).toBe('…/Public/OpenSourceBackup')
  })
  it('leaves short paths unchanged', () => {
    expect(friendlyLoc('/mnt/backup')).toBe('/mnt/backup')
  })
})

describe('statusColor', () => {
  it('maps known statuses to functional colours', () => {
    expect(statusColor('success')).toBe('var(--success)')
    expect(statusColor('failed')).toBe('var(--error)')
    expect(statusColor('running')).toBe('var(--success)')
    expect(statusColor('pending')).toBe('var(--running)')
    expect(statusColor('cancelled')).toBe('var(--warning)')
  })
  it('falls back to dim for unknown statuses', () => {
    expect(statusColor('weird')).toBe('var(--text-dim)')
    expect(statusColor(undefined)).toBe('var(--text-dim)')
  })
})
