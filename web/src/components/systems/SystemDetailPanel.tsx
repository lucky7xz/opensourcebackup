import type { System, BackupJob, BackupPolicy, RestoreTest, Snapshot } from '../../api'
import { timeAgo, fmt, duration } from '../../api'

interface Props {
  system:       System
  jobs:         BackupJob[]
  policies:     BackupPolicy[]
  restoreTests: RestoreTest[]
  snapshots:    Snapshot[]
  onClose:      () => void
  onRunBackup:  (sys: System) => void
}

type PanelTab = 'overview' | 'backups' | 'restore' | 'alerts' | 'activity'

// ── Helpers ───────────────────────────────────────────────────────────────────

function agentStatus(sys: System): 'online' | 'idle' | 'offline' {
  if (!sys.LastSeen) return 'offline'
  const mins = (Date.now() - new Date(sys.LastSeen).getTime()) / 60000
  return mins <= 2 ? 'online' : mins <= 15 ? 'idle' : 'offline'
}

function statusColor(s: 'online' | 'idle' | 'offline') {
  return s === 'online' ? '#22c55e' : s === 'idle' ? '#f59e0b' : '#64748b'
}

import { useState } from 'react'

export function SystemDetailPanel({ system, jobs, policies, restoreTests, snapshots, onClose, onRunBackup }: Props) {
  const [tab, setTab] = useState<PanelTab>('overview')

  const status    = agentStatus(system)
  const sysJobs   = jobs
    .filter(j => j.SystemID === system.ID && j.Type !== 'retention')
    .sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())
  const lastJob   = sysJobs[0]
  const lastRT    = restoreTests
    .filter(rt => rt.SystemID === system.ID)
    .sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())[0]
  const policy    = lastJob ? policies.find(p => p.ID === lastJob.PolicyID) : undefined
  const snapCount = snapshots.filter(sn => sysJobs.some(j => j.ID === sn.JobID)).length

  const TABS: { id: PanelTab; label: string }[] = [
    { id: 'overview',  label: 'Overview' },
    { id: 'backups',   label: 'Backups' },
    { id: 'restore',   label: 'Restore Tests' },
    { id: 'alerts',    label: 'Alerts' },
    { id: 'activity',  label: 'Activity' },
  ]

  return (
    <aside style={s.panel}>

      {/* Header */}
      <div style={s.header}>
        <div style={s.headerLeft}>
          <span style={{ fontSize: 16, marginRight: 6 }}>🌐</span>
          <span style={s.hostName}>{system.Hostname}</span>
          <span style={{ color: statusColor(status), fontSize: 10, marginLeft: 8 }}>● {status.charAt(0).toUpperCase() + status.slice(1)}</span>
        </div>
        <button style={s.closeBtn} onClick={onClose} title="Close">✕</button>
      </div>

      {/* Tabs */}
      <div style={s.tabs}>
        {TABS.map(t => (
          <button key={t.id}
            style={{ ...s.tabBtn, ...(tab === t.id ? s.tabActive : {}) }}
            onClick={() => setTab(t.id)}>
            {t.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div style={s.body}>
        {tab === 'overview' && (
          <>
            {/* System Information */}
            <div style={s.card}>
              <div style={s.cardHeader}>
                <span style={s.cardTitle}>System Information</span>
                <span style={s.editLink}>Edit</span>
              </div>
              <table style={s.infoTable}>
                <tbody>
                  <InfoRow label="Hostname"      value={system.Hostname} />
                  <InfoRow label="Type / OS"     value={system.OS ?? '—'} />
                  <InfoRow label="IP Address"    value="—" />
                  <InfoRow label="Agent Version" value={system.AgentVersion ? `${system.AgentVersion}` : '—'} />
                  <InfoRow label="First Seen"    value={fmtDate(system.CreatedAt)} />
                  <InfoRow label="Last Seen"     value={timeAgo(system.LastSeen)} />
                </tbody>
              </table>
              {system.Tags && Object.keys(system.Tags).length > 0 && (
                <div style={s.tagsRow}>
                  {Object.entries(system.Tags).map(([k, v]) => (
                    <span key={k} style={s.tag}>{k}={v}</span>
                  ))}
                </div>
              )}
            </div>

            {/* Policy Assignment */}
            <div style={s.card}>
              <div style={s.cardHeader}>
                <span style={s.cardTitle}>Policy Assignment</span>
                <span style={s.editLink}>Change Policy</span>
              </div>
              {policy ? (
                <>
                  <div style={s.policyName}>
                    <span style={s.policyBadge}>★</span>
                    <span style={{ fontWeight: 700, color: 'var(--text)' }}>{policy.Name}</span>
                    <span style={s.activeBadge}>Active</span>
                  </div>
                  <div style={s.policyDesc}>
                    {policy.Engine} engine
                    {policy.Schedule ? ` · ${policy.Schedule}` : ''}
                  </div>
                  {policy.RetentionPlan && (
                    <div style={s.retentionGrid}>
                      {policy.RetentionPlan.KeepDaily > 0 && <div style={s.retCell}><span style={s.retVal}>{policy.RetentionPlan.KeepDaily}</span><span style={s.retLbl}>Daily</span></div>}
                      {policy.RetentionPlan.KeepWeekly > 0 && <div style={s.retCell}><span style={s.retVal}>{policy.RetentionPlan.KeepWeekly}</span><span style={s.retLbl}>Weekly</span></div>}
                      {policy.RetentionPlan.KeepMonthly > 0 && <div style={s.retCell}><span style={s.retVal}>{policy.RetentionPlan.KeepMonthly}</span><span style={s.retLbl}>Monthly</span></div>}
                      {policy.RetentionPlan.KeepYearly > 0 && <div style={s.retCell}><span style={s.retVal}>{policy.RetentionPlan.KeepYearly}</span><span style={s.retLbl}>Yearly</span></div>}
                    </div>
                  )}
                  <div style={s.nextBackup}>
                    Next full backup: <span style={{ color: 'var(--text)' }}>Scheduled by policy</span>
                  </div>
                </>
              ) : (
                <div style={s.noData}>No policy assigned</div>
              )}
            </div>

            {/* Backup Summary */}
            <div style={s.card}>
              <div style={s.cardHeader}>
                <span style={s.cardTitle}>Backup Summary</span>
                <span style={s.editLink}>View All Backups</span>
              </div>
              {lastJob ? (
                <table style={s.infoTable}>
                  <tbody>
                    <InfoRow label="Last Backup"     value={timeAgo(lastJob.CreatedAt)} />
                    <InfoRow label="Data Backed Up"  value={fmt(lastJob.BytesUploaded)} />
                    <InfoRow label="Backup Status"   value={
                      <span style={{ color: lastJob.Status === 'success' ? 'var(--success)' : lastJob.Status === 'failed' ? 'var(--error)' : 'var(--warning)', fontWeight: 600 }}>
                        {lastJob.Status.charAt(0).toUpperCase() + lastJob.Status.slice(1)}
                      </span>
                    } />
                    <InfoRow label="Duration"        value={duration(lastJob.StartedAt, lastJob.FinishedAt)} />
                    <InfoRow label="Snapshots"       value={String(snapCount)} />
                  </tbody>
                </table>
              ) : (
                <div style={s.noData}>No backups have completed for this system yet.</div>
              )}
            </div>

            {/* Restore Test Summary */}
            <div style={s.card}>
              <div style={s.cardHeader}>
                <span style={s.cardTitle}>Restore Test Summary</span>
                <span style={s.editLink}>View All Tests</span>
              </div>
              {lastRT ? (
                <table style={s.infoTable}>
                  <tbody>
                    <InfoRow label="Last Test"      value={timeAgo(lastRT.CreatedAt)} />
                    <InfoRow label="Result"         value={
                      <span style={{ color: lastRT.Status === 'success' ? 'var(--success)' : 'var(--error)', fontWeight: 600 }}>
                        {lastRT.Status === 'success' ? 'Successful' : 'Failed'}
                      </span>
                    } />
                    <InfoRow label="Data Restored"  value={fmt(lastRT.VerifiedBytes)} />
                    <InfoRow label="Duration"       value={duration(lastRT.StartedAt, lastRT.FinishedAt)} />
                  </tbody>
                </table>
              ) : (
                <div style={s.noData}>No restore tests have been run for this system.</div>
              )}
            </div>

            {/* Quick Actions */}
            <div style={s.card}>
              <div style={{ padding: '0 0 10px', fontSize: 11, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.1em' }}>Quick Actions</div>
              <div style={s.quickGrid}>
                <QuickAction icon="▶" label="Run Backup"       onClick={() => onRunBackup(system)} />
                <QuickAction icon="✓" label="Restore Test"     onClick={() => {}} disabled />
                <QuickAction icon="📁" label="Recover Files"   onClick={() => {}} disabled />
                <QuickAction icon="📸" label="View Snapshots"  onClick={() => {}} disabled />
              </div>
            </div>
          </>
        )}

        {tab !== 'overview' && (
          <div style={s.comingSoon}>
            <div style={{ fontSize: 28, opacity: 0.3, marginBottom: 10 }}>
              {tab === 'backups' ? '🗂' : tab === 'restore' ? '✓' : tab === 'alerts' ? '🔔' : '📊'}
            </div>
            <div style={{ fontSize: 13, color: 'var(--text-muted)' }}>
              {tab.charAt(0).toUpperCase() + tab.slice(1)} view coming soon
            </div>
          </div>
        )}
      </div>
    </aside>
  )
}

// ── Sub-components ────────────────────────────────────────────────────────────

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <tr>
      <td style={{ padding: '5px 0', fontSize: 11, color: 'var(--text-dim)', width: '40%', verticalAlign: 'top' }}>{label}</td>
      <td style={{ padding: '5px 0 5px 8px', fontSize: 11, color: 'var(--text)', fontWeight: 500 }}>{value}</td>
    </tr>
  )
}

function QuickAction({ icon, label, onClick, disabled }: { icon: string; label: string; onClick: () => void; disabled?: boolean }) {
  return (
    <button onClick={onClick} disabled={disabled} style={{
      display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4,
      padding: '10px 8px', borderRadius: 8,
      background: disabled ? 'rgba(255,255,255,0.02)' : 'rgba(137,189,40,0.06)',
      border: '1px solid ' + (disabled ? 'var(--border)' : 'rgba(137,189,40,0.2)'),
      color: disabled ? 'var(--text-dim)' : 'var(--text-muted)',
      fontSize: 11, cursor: disabled ? 'default' : 'pointer',
      opacity: disabled ? 0.5 : 1, transition: 'all 0.1s',
      flex: 1, minWidth: 60,
    }}>
      <span style={{ fontSize: 18 }}>{icon}</span>
      <span style={{ fontSize: 10, fontWeight: 500, textAlign: 'center', lineHeight: 1.3 }}>{label}</span>
    </button>
  )
}

function fmtDate(iso?: string): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

// ── Styles ────────────────────────────────────────────────────────────────────

const s: Record<string, React.CSSProperties> = {
  panel: {
    width: 680, minWidth: 640, maxWidth: 720,
    borderLeft: '1px solid var(--border)',
    background: 'var(--bg-sidebar)',
    display: 'flex', flexDirection: 'column',
    height: '100%', overflowY: 'hidden',
    flexShrink: 0,
  },
  header: {
    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
    padding: '16px 16px 12px',
    borderBottom: '1px solid var(--border)',
    flexShrink: 0,
  },
  headerLeft: { display: 'flex', alignItems: 'center', minWidth: 0 },
  hostName:   { fontSize: 14, fontWeight: 700, color: 'var(--text)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' },
  closeBtn:   { background: 'none', border: 'none', color: 'var(--text-dim)', fontSize: 14, cursor: 'pointer', padding: '2px 6px', borderRadius: 4 },
  tabs: {
    display: 'flex', overflowX: 'auto', flexShrink: 0,
    borderBottom: '1px solid var(--border)',
    padding: '0 8px',
    gap: 2,
  },
  tabBtn: {
    padding: '9px 10px', border: 'none', background: 'none',
    color: 'var(--text-dim)', fontSize: 11, fontWeight: 600,
    cursor: 'pointer', borderBottom: '2px solid transparent',
    whiteSpace: 'nowrap', transition: 'all 0.12s',
  },
  tabActive: { color: 'var(--accent)', borderBottomColor: 'var(--accent)' },
  body:     { flex: 1, overflowY: 'auto', padding: '14px 18px', display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, alignContent: 'start' },
  card: {
    background: 'linear-gradient(180deg, rgba(21,28,46,0.95), rgba(10,15,27,0.95))',
    border: '1px solid rgba(148,163,184,0.1)',
    borderRadius: 10, padding: '13px 14px',
  },
  cardHeader:  { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 },
  cardTitle:   { fontSize: 11, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.1em' },
  editLink:    { fontSize: 11, color: 'var(--accent)', cursor: 'pointer', background: 'none', border: 'none', padding: 0 },
  infoTable:   { width: '100%', borderCollapse: 'collapse' },
  tagsRow:     { display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 8 },
  tag:         { fontSize: 10, padding: '2px 7px', borderRadius: 4, background: 'rgba(137,189,40,0.1)', color: 'var(--accent)', border: '1px solid rgba(137,189,40,0.2)' },
  noData:      { fontSize: 12, color: 'var(--text-dim)', fontStyle: 'italic', padding: '4px 0' },
  policyName:  { display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 },
  policyBadge: { color: '#f59e0b', fontSize: 14 },
  activeBadge: { fontSize: 10, padding: '2px 7px', borderRadius: 4, background: 'rgba(34,197,94,0.1)', color: 'var(--success)', border: '1px solid rgba(34,197,94,0.2)', marginLeft: 'auto' },
  policyDesc:  { fontSize: 11, color: 'var(--text-dim)', marginBottom: 8 },
  retentionGrid: { display: 'flex', gap: 8, marginTop: 6, marginBottom: 8 },
  retCell:     { flex: 1, background: 'rgba(255,255,255,0.03)', borderRadius: 6, padding: '6px 8px', textAlign: 'center' },
  retVal:      { display: 'block', fontSize: 14, fontWeight: 700, color: 'var(--text)' },
  retLbl:      { display: 'block', fontSize: 9, color: 'var(--text-dim)', marginTop: 1 },
  nextBackup:  { fontSize: 11, color: 'var(--text-dim)', marginTop: 6 },
  quickGrid:   { display: 'flex', gap: 6, flexWrap: 'wrap' },
  comingSoon:  { flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: 40 },
}
