import type { System, BackupJob, BackupPolicy, RestoreTest } from '../../api'
import { timeAgo, fmt } from '../../api'
import { HealthBar } from '../common/HealthBar'

interface Props {
  systems:      System[]
  jobs:         BackupJob[]
  policies:     BackupPolicy[]
  restoreTests: RestoreTest[]
  selected:     System | null
  onSelect:     (sys: System) => void
  onRun:        (sys: System) => void
  onEdit:       (sys: System) => void
  onDelete:     (sys: System) => void
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function agentStatus(sys: System): 'online' | 'idle' | 'offline' {
  if (!sys.LastSeen) return 'offline'
  const mins = (Date.now() - new Date(sys.LastSeen).getTime()) / 60000
  return mins <= 2 ? 'online' : mins <= 15 ? 'idle' : 'offline'
}

function sysIcon(sys: System): string {
  const os = (sys.OS ?? '').toLowerCase()
  const h  = sys.Hostname.toLowerCase()
  if (h.includes('k8s') || h.includes('kube') || os.includes('kubernetes')) return '☸'
  if (os.includes('vmware') || os.includes('esxi') || h.includes('esxi'))   return '⬡'
  if (os.includes('truenas') || os.includes('freenas') || h.includes('nas')) return '🗄'
  if (os.includes('postgresql') || os.includes('mysql') || h.includes('db-') || h.includes('-db')) return '🗃'
  if (os.includes('windows')) return '⊞'
  return '🖥'
}

function riskConfig(rc: string): { label: string; color: string; bg: string } {
  switch ((rc ?? '').toLowerCase()) {
    case 'critical': return { label: 'Critical', color: '#ef4444', bg: 'rgba(239,68,68,0.12)' }
    case 'high':     return { label: 'High',     color: '#f97316', bg: 'rgba(249,115,22,0.12)' }
    case 'medium':   return { label: 'Medium',   color: '#f59e0b', bg: 'rgba(245,158,11,0.12)' }
    default:         return { label: 'Low',      color: '#22c55e', bg: 'rgba(34,197,94,0.1)' }
  }
}

function statusDot(status: 'online' | 'idle' | 'offline'): string {
  return status === 'online' ? '#22c55e' : status === 'idle' ? '#f59e0b' : '#64748b'
}

function systemHealth(sysId: string, jobs: BackupJob[]): number | null {
  const sj = jobs
    .filter(j => j.SystemID === sysId && (j.Type !== 'retention'))
    .sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())
    .slice(0, 10)
  if (!sj.length) return null
  return Math.round(sj.filter(j => j.Status === 'success').length / sj.length * 100)
}

function lastJobFor(sysId: string, jobs: BackupJob[]): BackupJob | undefined {
  return jobs
    .filter(j => j.SystemID === sysId && (j.Type !== 'retention'))
    .sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())[0]
}

function lastRestoreFor(sysId: string, restoreTests: RestoreTest[]): RestoreTest | undefined {
  return restoreTests
    .filter(rt => rt.SystemID === sysId)
    .sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())[0]
}

function policyFor(sysId: string, jobs: BackupJob[], policies: BackupPolicy[]): BackupPolicy | undefined {
  const pid = lastJobFor(sysId, jobs)?.PolicyID
  return pid ? policies.find(p => p.ID === pid) : undefined
}

// ── Component ─────────────────────────────────────────────────────────────────

export function SystemsTable({ systems, jobs, policies, restoreTests, selected, onSelect, onRun, onEdit, onDelete }: Props) {
  if (systems.length === 0) {
    return (
      <div style={s.empty}>
        <div style={s.emptyIcon}>🖥</div>
        <div style={s.emptyTitle}>No systems registered yet.</div>
        <div style={s.emptySub}>Add your first system to start protecting it.</div>
      </div>
    )
  }

  return (
    <div style={s.wrap}>
      <table style={s.table}>
        <thead>
          <tr>
            <th style={s.th}><input type="checkbox" style={{ accentColor: 'var(--accent)' }} /></th>
            {['System', 'Risk Class', 'Agent Status', 'Last Backup', 'Last Restore Test', 'Policy', 'Health', ''].map(h =>
              <th key={h} style={s.th}>{h}</th>
            )}
          </tr>
        </thead>
        <tbody>
          {systems.map(sys => {
            const status  = agentStatus(sys)
            const risk    = riskConfig(sys.RiskClass)
            const lastJob = lastJobFor(sys.ID, jobs)
            const lastRT  = lastRestoreFor(sys.ID, restoreTests)
            const policy  = policyFor(sys.ID, jobs, policies)
            const health  = systemHealth(sys.ID, jobs)
            const isOn    = selected?.ID === sys.ID

            return (
              <tr key={sys.ID}
                onClick={() => onSelect(sys)}
                style={{ ...s.tr, ...(isOn ? s.trActive : {}) }}>

                {/* Checkbox */}
                <td style={s.td} onClick={e => e.stopPropagation()}>
                  <input type="checkbox" style={{ accentColor: 'var(--accent)' }} />
                </td>

                {/* System */}
                <td style={s.td}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <span style={s.sysIcon}>{sysIcon(sys)}</span>
                    <div>
                      <div style={s.sysName}>{sys.Hostname}</div>
                      <div style={s.sysSub}>{sys.OS ?? 'Unknown OS'}</div>
                    </div>
                  </div>
                </td>

                {/* Risk Class */}
                <td style={s.td}>
                  <span style={{ ...s.badge, color: risk.color, background: risk.bg }}>
                    {risk.label}
                  </span>
                </td>

                {/* Agent Status */}
                <td style={s.td}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span style={{ color: statusDot(status), fontSize: 10 }}>●</span>
                    <div>
                      <div style={{ fontSize: 12, fontWeight: 600, color: statusDot(status), textTransform: 'capitalize' }}>{status}</div>
                      {sys.AgentVersion && <div style={{ fontSize: 10, color: 'var(--text-dim)' }}>{sys.AgentVersion}</div>}
                    </div>
                  </div>
                </td>

                {/* Last Backup */}
                <td style={s.td}>
                  {lastJob ? (
                    <div>
                      <div style={{ fontSize: 12, color: 'var(--text)' }}>{timeAgo(lastJob.CreatedAt)}</div>
                      <div style={{ fontSize: 10, color: 'var(--text-dim)' }}>{fmt(lastJob.BytesUploaded)}</div>
                    </div>
                  ) : (
                    <span style={{ fontSize: 11, color: 'var(--text-dim)' }}>never</span>
                  )}
                </td>

                {/* Last Restore Test */}
                <td style={s.td}>
                  {lastRT ? (
                    <div>
                      <div style={{ fontSize: 12, color: 'var(--text)' }}>{timeAgo(lastRT.CreatedAt)}</div>
                      <div style={{ fontSize: 10, color: lastRT.Status === 'success' ? 'var(--success)' : 'var(--error)' }}>
                        {lastRT.Status === 'success' ? 'Successful' : 'Failed'}
                      </div>
                    </div>
                  ) : (
                    <span style={{ fontSize: 11, color: 'var(--text-dim)' }}>Not tested</span>
                  )}
                </td>

                {/* Policy */}
                <td style={s.td}>
                  {policy ? (
                    <div>
                      <div style={{ fontSize: 12, color: 'var(--text)', fontWeight: 500 }}>{policy.Name}</div>
                      <div style={{ fontSize: 10, color: 'var(--text-dim)' }}>{policy.Engine}</div>
                    </div>
                  ) : (
                    <span style={{ fontSize: 11, color: 'var(--text-dim)' }}>No policy</span>
                  )}
                </td>

                {/* Health */}
                <td style={s.td}><HealthBar pct={health} /></td>

                {/* Actions */}
                <td style={s.td} onClick={e => e.stopPropagation()}>
                  <div style={{ display: 'flex', gap: 4 }}>
                    <button style={s.actBtn} title="Run Backup" onClick={() => onRun(sys)}>▶</button>
                    <button style={s.actBtn} title="Edit" onClick={() => onEdit(sys)}>✏</button>
                    <button style={{ ...s.actBtn, color: 'var(--error)' }} title="Delete" onClick={() => onDelete(sys)}>🗑</button>
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  wrap:      { overflowX: 'auto' },
  table:     { width: '100%', borderCollapse: 'collapse' },
  th:        { padding: '9px 12px', fontSize: 10, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.1em', textAlign: 'left', borderBottom: '1px solid var(--border)', background: 'rgba(255,255,255,0.015)', whiteSpace: 'nowrap' },
  tr:        { borderBottom: '1px solid rgba(255,255,255,0.04)', cursor: 'pointer', transition: 'background 0.1s' },
  trActive:  { background: 'rgba(137,189,40,0.07)', borderLeft: '2px solid var(--accent)' },
  td:        { padding: '11px 12px', fontSize: 12, color: 'var(--text-muted)', verticalAlign: 'middle' },
  badge:     { display: 'inline-block', fontSize: 10, fontWeight: 700, padding: '2px 8px', borderRadius: 4 },
  sysIcon:   { fontSize: 18, opacity: 0.85, flexShrink: 0 },
  sysName:   { fontWeight: 600, color: 'var(--text)', fontSize: 13 },
  sysSub:    { fontSize: 10, color: 'var(--text-dim)', marginTop: 1 },
  actBtn:    { padding: '4px 8px', borderRadius: 5, background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 11, cursor: 'pointer' },
  empty:     { padding: '52px 24px', textAlign: 'center' },
  emptyIcon: { fontSize: 40, opacity: 0.3, marginBottom: 12 },
  emptyTitle:{ fontSize: 15, fontWeight: 600, color: 'var(--text-muted)' },
  emptySub:  { fontSize: 12, color: 'var(--text-dim)', marginTop: 4 },
}
