import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, fmt, timeAgo, duration, type BackupJob, type RestoreTest, type Snapshot, type System, type BackupRepository, type RepositoryHealth, type HealthScore, type ActivityBucket } from '../api'
import { StatusBadge } from '../components/StatusBadge'
import { DonutChart, DonutLegend } from '../components/DonutChart'
import { ActivityChart, ActivityLegend } from '../components/ActivityChart'

// ── Component ─────────────────────────────────────────────────────────────────

export function Dashboard() {
  const navigate = useNavigate()
  const [systems,      setSystems]      = useState<System[]>([])
  const [jobs,         setJobs]         = useState<BackupJob[]>([])
  const [snapshots,    setSnapshots]    = useState<Snapshot[]>([])
  const [restoreTests, setRestoreTests] = useState<RestoreTest[]>([])
  const [repos,       setRepos]       = useState<BackupRepository[]>([])
  const [repoHealth,  setRepoHealth]  = useState<RepositoryHealth[]>([])
  const [healthScore, setHealthScore] = useState<HealthScore|null>(null)
  const [activity,    setActivity]    = useState<ActivityBucket[]>([])
  const [alerts,      setAlerts]      = useState<any[]>([])
  const [evidence,    setEvidence]    = useState<any[]>([])
  const [loading,     setLoading]     = useState(true)

  useEffect(() => {
    Promise.all([
      api.systems(), api.jobs(), api.snapshots(), api.restoreTests(),
      api.repositories(),
      api.repositoryHealth().catch(() => [] as RepositoryHealth[]),
      api.healthScore().catch(() => null),
      api.healthActivity(24).catch(() => []),
      api.healthAlerts().catch(() => ({ alerts: [], summary: {} })),
      api.auditLog(6).catch(() => []),
    ]).then(([s, j, sn, rt, r, rh, hs, act, al, ev]) => {
      setSystems(s); setJobs(j); setSnapshots(sn); setRestoreTests(rt)
      setRepos(r); setRepoHealth(rh); setHealthScore(hs)
      setActivity(act as ActivityBucket[])
      setAlerts((al as any)?.alerts ?? [])
      setEvidence(ev as any[])
    }).finally(() => setLoading(false))
  }, [])

  if (loading) return <div style={s.loading}>Loading…</div>

  // ── Derived metrics ───────────────────────────────────────────────────────

  const successJobs = jobs.filter(j => j.Status === 'success').length
  const failedJobs  = jobs.filter(j => j.Status === 'failed').length

  const now = Date.now()
  const ms24h = 24 * 60 * 60 * 1000
  const failedLast24h = jobs.filter(j =>
    j.Status === 'failed' && (now - new Date(j.CreatedAt).getTime()) < ms24h
  ).length

  const successRate = jobs.length > 0
    ? Math.round((successJobs / jobs.length) * 100)
    : 0

  // Snapshot restore coverage
  const verifiedSnaps = snapshots.filter(sn =>
    restoreTests.some(rt => rt.SnapshotID === sn.ID && rt.Status === 'success')
  )
  const failedOnlySnaps = snapshots.filter(sn =>
    !restoreTests.some(rt => rt.SnapshotID === sn.ID && rt.Status === 'success') &&
     restoreTests.some(rt => rt.SnapshotID === sn.ID && rt.Status === 'failed')
  )
  const untestedSnaps = snapshots.filter(sn =>
    !restoreTests.some(rt => rt.SnapshotID === sn.ID)
  )
  const restoreVerifiedPct = snapshots.length > 0
    ? Math.round((verifiedSnaps.length / snapshots.length) * 100)
    : 0

  // ── Agent activity ────────────────────────────────────────────────────────
  // Thresholds: Online ≤ 2min · Idle ≤ 15min · Offline > 15min or never seen
  const MS_ONLINE  = 2  * 60 * 1000
  const MS_IDLE    = 15 * 60 * 1000

  function agentStatus(sys: { LastSeen?: string }): 'online' | 'idle' | 'offline' {
    if (!sys.LastSeen) return 'offline'
    const age = now - new Date(sys.LastSeen).getTime()
    if (age <= MS_ONLINE) return 'online'
    if (age <= MS_IDLE)   return 'idle'
    return 'offline'
  }

  const onlineSystems  = systems.filter(s => agentStatus(s) === 'online')
  const idleSystems    = systems.filter(s => agentStatus(s) === 'idle')
  const offlineSystems = systems.filter(s => agentStatus(s) === 'offline')

  const agentDonut = [
    { value: onlineSystems.length,  color: 'var(--success)',  label: 'Online'  },
    { value: idleSystems.length,    color: 'var(--warning)',  label: 'Idle'    },
    { value: offlineSystems.length, color: 'var(--error)',    label: 'Offline' },
  ]

  // Last-seen list — show all, sorted by most recent first
  const systemsByLastSeen = [...systems].sort((a, b) => {
    if (!a.LastSeen && !b.LastSeen) return 0
    if (!a.LastSeen) return 1
    if (!b.LastSeen) return -1
    return new Date(b.LastSeen).getTime() - new Date(a.LastSeen).getTime()
  })

  // Recent jobs
  const recentJobs = [...jobs]
    .sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())
    .slice(0, 8)

  // Donut segments
  const restoreDonut = [
    { value: verifiedSnaps.length,  color: 'var(--success)',  label: 'Verified' },
    { value: untestedSnaps.length,  color: 'var(--text-dim)', label: 'Not Tested' },
    { value: failedOnlySnaps.length,color: 'var(--error)',    label: 'Failed' },
  ]
  const donutTotal = snapshots.length

  return (
    <div style={s.page}>

      {/* ── Page header ───────────────────────────────────────────────────── */}
      <div style={s.header}>
        <div>
          <h1 style={s.h1}>Dashboard</h1>
          <p style={s.sub}>Real-time overview of your backup posture and restore assurance.</p>
        </div>
      </div>

      {/* ── KPI row ───────────────────────────────────────────────────────── */}
      <div style={s.kpiRow}>

        <KpiCard
          icon="🖥"
          label="Protected Systems"
          value={`${systems.length}`}
          sub={systems.length === 0 ? 'No agents enrolled' : `${systems.length} system${systems.length !== 1 ? 's' : ''} enrolled`}
          color="var(--accent)"
        />

        <KpiCard
          icon="✓"
          label="Backup Success Rate"
          value={jobs.length > 0 ? `${successRate}%` : '—'}
          sub={jobs.length > 0 ? `${successJobs} of ${jobs.length} jobs` : 'No jobs yet'}
          color={successRate >= 90 ? 'var(--success)' : successRate >= 70 ? 'var(--warning)' : 'var(--error)'}
        />

        <KpiCard
          icon="🔄"
          label="Restore Verified"
          value={snapshots.length > 0 ? `${restoreVerifiedPct}%` : '—'}
          sub={snapshots.length > 0
            ? `${verifiedSnaps.length} of ${snapshots.length} snapshots`
            : 'No snapshots yet'}
          color={restoreVerifiedPct === 100 ? 'var(--success)'
            : restoreVerifiedPct > 0 ? 'var(--warning)'
            : snapshots.length > 0 ? 'var(--error)' : 'var(--text-dim)'}
          warn={snapshots.length > 0 && restoreVerifiedPct === 0}
        />

        <KpiCard
          icon="⚠"
          label="Failed Jobs"
          value={`${failedJobs}`}
          sub={failedLast24h > 0 ? `${failedLast24h} in last 24h` : 'None in last 24h'}
          color={failedJobs > 0 ? 'var(--error)' : 'var(--text-dim)'}
          warn={failedJobs > 0}
        />

        {/* Recovery Score — from backend canonical calculation */}
        <div style={{ ...s.kpiCard, flexDirection: 'row', gap: 20, alignItems: 'center' }}>
          {healthScore ? (
            <>
              <div style={{ ...s.scoreRing, borderColor: healthScore.color }}>
                <span style={{ fontSize: 22, fontWeight: 800, color: healthScore.color }}>
                  {healthScore.score}
                </span>
                <span style={{ fontSize: 10, color: 'var(--text-dim)', letterSpacing: '0.05em' }}>/ 100</span>
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 11, color: 'var(--text-dim)', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: 2 }}>
                  Backup Health Score
                </div>
                <div style={{ fontSize: 17, fontWeight: 700, color: healthScore.color, marginBottom: 6 }}>
                  {healthScore.label}
                </div>
                {healthScore.deductions.length === 0 ? (
                  <div style={{ fontSize: 11, color: 'var(--success)' }}>✓ All checks passed</div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                    {healthScore.deductions.map((d, i) => (
                      <div key={i} style={{ fontSize: 11, color: 'var(--text-muted)', display: 'flex', gap: 5 }}>
                        <span style={{ color: 'var(--error)', fontWeight: 700, flexShrink: 0 }}>−{d.points}</span>
                        <span>{d.reason}</span>
                      </div>
                    ))}
                  </div>
                )}
                {healthScore.factors.length > 0 && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 2, marginTop: 4 }}>
                    {healthScore.factors.slice(0, 2).map((f, i) => (
                      <div key={i} style={{ fontSize: 10, color: 'var(--success)' }}>✓ {f}</div>
                    ))}
                  </div>
                )}
                <div style={{ fontSize: 9, color: 'var(--text-dim)', marginTop: 6, fontStyle: 'italic' }}>
                  Score v{healthScore.version} · Prometheus: opensourcebackup_recovery_score
                </div>
              </div>
            </>
          ) : (
            <div style={{ color: 'var(--text-dim)', fontSize: 13 }}>Loading score…</div>
          )}
        </div>

      </div>

      {/* ── Main content — 3-column fill ─────────────────────────────────── */}
      <div style={s.mainGrid}>

        {/* Recent Jobs */}
        <div style={s.wideCard}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>Recent Jobs</span>
            <button onClick={() => navigate('/jobs')} style={s.viewAll}>View all →</button>
          </div>
          <table style={s.table}>
            <thead>
              <tr>
                {['Job ID', 'System', 'Type', 'Status', 'Duration', 'Completed'].map(h => (
                  <th key={h} style={s.th}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {recentJobs.length === 0 ? (
                <tr><td colSpan={6} style={s.empty}>No jobs yet</td></tr>
              ) : recentJobs.map(j => (
                <tr key={j.ID} style={s.tr}>
                  <td style={s.td}><span style={s.mono}>{j.ID.slice(0, 8)}…</span></td>
                  <td style={s.td}><span style={s.mono}>{j.SystemID.slice(0, 8)}…</span></td>
                  <td style={s.td}><span style={s.tag}>Backup</span></td>
                  <td style={s.td}><StatusBadge status={j.Status} /></td>
                  <td style={s.td}>{duration(j.StartedAt, j.FinishedAt)}</td>
                  <td style={s.td} >{timeAgo(j.FinishedAt || j.CreatedAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Restore Verification donut */}
        <div style={s.card}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>Restore Verification</span>
            <button onClick={() => navigate('/restore-tests')} style={s.viewAll}>View all →</button>
          </div>

          {snapshots.length === 0 ? (
            <div style={s.emptyState}>
              <div style={{ fontSize: 32, marginBottom: 8 }}>🔄</div>
              <div style={{ fontSize: 13, color: 'var(--text-muted)', textAlign: 'center' }}>
                No snapshots yet.<br />Run a backup first, then configure restore tests.
              </div>
            </div>
          ) : (
            <>
              <div style={{ display: 'flex', gap: 20, alignItems: 'center', padding: '8px 0 16px' }}>
                <DonutChart
                  segments={restoreDonut}
                  size={110}
                  thickness={16}
                  center={
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: 20, fontWeight: 800, color: 'var(--text)' }}>
                        {snapshots.length}
                      </div>
                      <div style={{ fontSize: 10, color: 'var(--text-dim)' }}>Snapshots</div>
                    </div>
                  }
                />
                <DonutLegend segments={restoreDonut} total={donutTotal} />
              </div>

              {restoreVerifiedPct < 100 && (
                <div style={s.verifyNotice}>
                  <span style={{ color: 'var(--warning)', fontWeight: 600 }}>
                    {untestedSnaps.length > 0
                      ? `${untestedSnaps.length} snapshot${untestedSnaps.length > 1 ? 's' : ''} not yet tested`
                      : `${failedOnlySnaps.length} restore test${failedOnlySnaps.length > 1 ? 's' : ''} failed`
                    }
                  </span>
                  {' — '}
                  <button onClick={() => navigate('/restore-tests')} style={s.inlineLink}>
                    schedule restore tests →
                  </button>
                </div>
              )}
            </>
          )}
        </div>

        {/* Storage summary */}
        <div style={s.card}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>Storage Used</span>
          </div>
          <div style={{ padding: '8px 0' }}>
            <div style={s.storageTotal}>
              {fmt(jobs.reduce((a, j) => a + (j.BytesUploaded ?? 0), 0))}
            </div>
            <div style={{ fontSize: 12, color: 'var(--text-dim)', marginTop: 2 }}>
              across {snapshots.length} snapshot{snapshots.length !== 1 ? 's' : ''}
            </div>
            <div style={s.divider} />
            <StatRow label="Successful jobs" value={`${successJobs}`} color="var(--success)" />
            <StatRow label="Failed jobs"     value={`${failedJobs}`}  color={failedJobs > 0 ? 'var(--error)' : 'var(--text-dim)'} />
            <StatRow label="Total jobs"      value={`${jobs.length}`} color="var(--text-muted)" />
          </div>
        </div>

      </div>

      {/* ── Repository Health ─────────────────────────────────────────────── */}
      {repos.length > 0 && (
        <div style={{ ...s.card, marginTop: 16 }}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>Repository Health</span>
            <button onClick={() => navigate('/repositories')} style={s.viewAll}>Manage →</button>
          </div>
          <table style={s.table}>
            <thead>
              <tr>
                {['Repository', 'Type', 'Immutability', 'Encryption', 'Snapshots', 'Verified', 'Last Backup', 'Last Restore Test'].map(h => (
                  <th key={h} style={s.th}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {repos.map(repo => {
                const health = repoHealth.find(h => h.RepositoryID === repo.ID)
                const imm = repo.ImmutableMode ?? 'none'
                const immColor = (imm === 'object_lock' || imm === 'worm') ? 'var(--success)'
                  : imm === 'append_only' ? '#22c55e'
                  : imm === 'unknown' ? 'var(--warning)'
                  : 'var(--text-dim)'
                const immLabel = {
                  object_lock: '🔒 Object Lock', worm: '🔒 WORM',
                  append_only: '📎 Append-Only', unknown: '? Unknown', none: '— None',
                }[imm] ?? '— None'

                return (
                  <tr key={repo.ID} style={s.tr}>
                    <td style={s.td}><span style={s.mono}>{repo.Location.length > 28 ? repo.Location.slice(-28) : repo.Location}</span></td>
                    <td style={s.td}><span style={s.tag}>{repo.Type}</span></td>
                    <td style={s.td}><span style={{ fontSize: 12, color: immColor, fontWeight: 600 }}>{immLabel}</span></td>
                    <td style={s.td}>
                      {health?.EncryptionEnabled
                        ? <span style={{ color: 'var(--success)', fontSize: 12 }}>✓ AES-256</span>
                        : <span style={{ color: 'var(--warning)', fontSize: 12 }}>⚠ Off</span>}
                    </td>
                    <td style={s.td}>{health?.SnapshotCount ?? '—'}</td>
                    <td style={s.td}>
                      {health && health.SnapshotCount > 0
                        ? <span style={{ color: health.VerifiedCount === health.SnapshotCount ? 'var(--success)' : 'var(--warning)', fontSize: 12, fontWeight: 600 }}>
                            {health.VerifiedCount}/{health.SnapshotCount}
                          </span>
                        : <span style={{ color: 'var(--text-dim)', fontSize: 12 }}>—</span>
                      }
                    </td>
                    <td style={s.td}>{timeAgo(health?.LastBackupAt)}</td>
                    <td style={s.td}>{timeAgo(health?.LastRestoreTestAt)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* ── Activity Chart + Alerts + Evidence ────────────────────────────── */}
      <div style={s.threeCol}>

        {/* Backup & Restore Activity Chart */}
        <div style={{ ...s.card, gridColumn: 'span 2' }}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>Backup & Restore Activity (24h)</span>
            <ActivityLegend />
          </div>
          <ActivityChart data={activity} height={150} />
          <div style={s.activityStats}>
            <span>Backups: <strong style={{ color: '#00d4ff' }}>{activity.reduce((a, b) => a + b.backups, 0)}</strong></span>
            <span>Restore Tests: <strong style={{ color: '#00ff88' }}>{activity.reduce((a, b) => a + b.restore_tests, 0)}</strong></span>
            <span>Failures: <strong style={{ color: '#ef4444' }}>{activity.reduce((a, b) => a + b.failures, 0)}</strong></span>
          </div>
        </div>

        {/* Alerts Preview */}
        <div style={s.card}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>Alerts</span>
            <button onClick={() => navigate('/alerts')} style={s.viewAll}>View all →</button>
          </div>
          {alerts.length === 0 ? (
            <div style={s.panelEmpty}>
              <span style={{ fontSize: 18 }}>✅</span>
              <span style={{ fontSize: 12, color: 'var(--text-dim)' }}>All checks passed</span>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginTop: 8 }}>
              {alerts.slice(0, 4).map((a: any) => (
                <div key={a.code} style={s.alertPreviewItem}>
                  <span style={{ fontSize: 14 }}>
                    {a.severity === 'critical' ? '🔴' : a.severity === 'warning' ? '⚠️' : 'ℹ️'}
                  </span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 12, color: 'var(--text)', fontWeight: 600,
                      overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {a.title}
                    </div>
                    <div style={{ fontSize: 10, color: 'var(--text-dim)', marginTop: 1 }}>
                      {a.category} · −{a.points} pts
                    </div>
                  </div>
                </div>
              ))}
              {alerts.length > 4 && (
                <button onClick={() => navigate('/alerts')} style={s.viewAll}>
                  +{alerts.length - 4} more →
                </button>
              )}
            </div>
          )}
        </div>

      </div>

      {/* ── Recent Evidence ────────────────────────────────────────────────── */}
      {evidence.length > 0 && (
        <div style={{ ...s.card, marginTop: 16 }}>
          <div style={s.cardHeader}>
            <span style={s.cardTitle}>Recent Evidence</span>
            <button onClick={() => navigate('/evidence')} style={s.viewAll}>View all →</button>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
            {evidence.map((e: any) => (
              <div key={e.ID} style={s.evidenceItem}>
                <span style={{ fontSize: 12, color: '#00ff88' }}>✓</span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <span style={s.mono}>{e.Action}</span>
                  {e.ResourceType && (
                    <span style={{ fontSize: 11, color: 'var(--text-dim)', marginLeft: 8 }}>
                      {e.ResourceType}
                    </span>
                  )}
                </div>
                <span style={{ fontSize: 11, color: 'var(--text-dim)', flexShrink: 0 }}>
                  {timeAgo(e.Timestamp)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ── Agent Activity ─────────────────────────────────────────────────── */}
      <div style={s.agentRow}>

        {/* Donut */}
        <div style={{ ...s.card, display: 'flex', gap: 20, alignItems: 'center' }}>
          <DonutChart
            segments={agentDonut}
            size={100}
            thickness={15}
            center={
              <div style={{ textAlign: 'center' }}>
                <div style={{ fontSize: 18, fontWeight: 800, color: 'var(--text)' }}>
                  {systems.length}
                </div>
                <div style={{ fontSize: 9, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  Agents
                </div>
              </div>
            }
          />
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 11, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: 10 }}>
              Agent Activity
            </div>
            <DonutLegend segments={agentDonut} total={systems.length} />
            <div style={{ fontSize: 10, color: 'var(--text-dim)', marginTop: 8, fontStyle: 'italic' }}>
              Online ≤ 2min · Idle ≤ 15min · Offline = no heartbeat
            </div>
          </div>
        </div>

        {/* Last Seen list */}
        <div style={{ ...s.card, flex: 2 }}>
          <div style={{ ...s.cardHeader, paddingLeft: 0, paddingTop: 0 }}>
            <span style={s.cardTitle}>Last Seen</span>
          </div>
          {systems.length === 0 ? (
            <div style={{ fontSize: 12, color: 'var(--text-dim)', padding: '8px 0' }}>
              No agents enrolled yet.
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4, marginTop: 8 }}>
              {systemsByLastSeen.slice(0, 8).map(sys => {
                const st = agentStatus(sys)
                const dot = st === 'online'  ? 'var(--success)'
                          : st === 'idle'    ? 'var(--warning)'
                          : 'var(--error)'
                const age = sys.LastSeen
                  ? timeAgo(sys.LastSeen)
                  : 'never'
                return (
                  <div key={sys.ID} style={s.agentRow2}>
                    <span style={{ ...s.statusDot, background: dot, boxShadow: st === 'online' ? `0 0 5px ${dot}` : 'none' }} />
                    <span style={{ fontSize: 12, color: 'var(--text)', flex: 1 }}>{sys.Hostname}</span>
                    <span style={{ fontSize: 11, color: 'var(--text-dim)' }}>{age}</span>
                  </div>
                )
              })}
            </div>
          )}
        </div>

      </div>
    </div>
  )
}

// ── Sub-components ─────────────────────────────────────────────────────────

function KpiCard({ icon, label, value, sub, color, warn }: {
  icon: string; label: string; value: string
  sub?: string; color: string; warn?: boolean
}) {
  return (
    <div style={{ ...s.kpiCard, borderColor: warn ? 'rgba(245,158,11,0.3)' : 'var(--border)' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
        <span style={{ fontSize: 16 }}>{icon}</span>
        <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>
          {label}
        </span>
      </div>
      <div style={{ fontSize: 28, fontWeight: 800, color, lineHeight: 1, marginBottom: 4 }}>
        {value}
      </div>
      {sub && <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>{sub}</div>}
    </div>
  )
}

function StatRow({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', padding: '5px 0', borderBottom: '1px solid rgba(255,255,255,0.04)' }}>
      <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{label}</span>
      <span style={{ fontSize: 12, fontWeight: 600, color }}>{value}</span>
    </div>
  )
}

// ── Styles ─────────────────────────────────────────────────────────────────

const s: Record<string, React.CSSProperties> = {
  page:        { padding: '28px 36px' },
  loading:     { padding: 40, color: 'var(--text-muted)', textAlign: 'center' },
  header:      { marginBottom: 24 },
  h1:          { fontSize: 22, fontWeight: 700, color: 'var(--text)', marginBottom: 2 },
  sub:         { fontSize: 13, color: 'var(--text-muted)' },

  kpiRow: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr 1fr 1fr 2fr',
    gap: 14,
    marginBottom: 24,
  },
  kpiCard: {
    background: 'var(--bg-card)', border: '1px solid var(--border)',
    borderRadius: 10, padding: '16px 20px',
    display: 'flex', flexDirection: 'column',
  },
  scoreRing: {
    width: 72, height: 72, borderRadius: '50%',
    border: '3px solid',
    display: 'flex', flexDirection: 'column',
    alignItems: 'center', justifyContent: 'center',
    flexShrink: 0,
  },

  mainGrid: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr 1fr',
    gap: 16,
    alignItems: 'start',
  },

  wideCard: {
    background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 10,
    overflow: 'hidden',
  },
  card: {
    background: 'var(--bg-card)', border: '1px solid var(--border)',
    borderRadius: 10, padding: '16px 20px',
  },
  cardHeader: {
    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
    padding: '14px 20px 10px', borderBottom: '1px solid var(--border)',
  },
  cardTitle:   { fontSize: 13, fontWeight: 700, color: 'var(--text)' },
  viewAll:     {
    fontSize: 11, color: 'var(--accent)', background: 'none',
    border: 'none', cursor: 'pointer', padding: 0,
  },

  table:  { width: '100%', borderCollapse: 'collapse' as const },
  th:     { padding: '8px 16px', fontSize: 11, fontWeight: 700, color: 'var(--text-dim)', textAlign: 'left' as const, textTransform: 'uppercase' as const, letterSpacing: '0.06em', borderBottom: '1px solid var(--border)', background: 'rgba(255,255,255,0.02)' },
  tr:     { borderBottom: '1px solid rgba(255,255,255,0.04)' },
  td:     { padding: '9px 16px', fontSize: 13, color: 'var(--text-muted)', verticalAlign: 'middle' as const },
  empty:  { padding: '24px 16px', textAlign: 'center' as const, color: 'var(--text-dim)', fontSize: 13 },
  mono:   { fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--accent)' },
  tag:    { fontSize: 11, padding: '2px 7px', borderRadius: 4, background: 'rgba(59,130,246,0.1)', color: 'var(--accent)', fontWeight: 600 },

  emptyState: {
    display: 'flex', flexDirection: 'column', alignItems: 'center',
    justifyContent: 'center', padding: '24px 0', minHeight: 120,
  },
  verifyNotice: {
    background: 'rgba(245,158,11,0.07)', border: '1px solid rgba(245,158,11,0.2)',
    borderRadius: 6, padding: '8px 12px', fontSize: 12, color: 'var(--text-muted)',
    marginTop: 4,
  },
  inlineLink:  { background: 'none', border: 'none', color: 'var(--accent)', fontSize: 12, cursor: 'pointer', padding: 0 },

  storageTotal: { fontSize: 26, fontWeight: 800, color: 'var(--text)', marginBottom: 2 },
  divider:      { height: 1, background: 'var(--border)', margin: '12px 0' },

  threeCol: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr 1fr',
    gap: 16,
    marginTop: 16,
  },
  activityStats: {
    display: 'flex', gap: 20, padding: '8px 20px 12px',
    fontSize: 12, color: 'var(--text-dim)',
  },
  panelEmpty: {
    display: 'flex', flexDirection: 'column' as const, alignItems: 'center',
    justifyContent: 'center', gap: 6, padding: '20px 0', minHeight: 80,
  },
  alertPreviewItem: {
    display: 'flex', alignItems: 'flex-start', gap: 8,
    padding: '6px 8px', borderRadius: 6,
    background: 'rgba(255,255,255,0.02)',
    border: '1px solid rgba(255,255,255,0.05)',
  },
  evidenceItem: {
    display: 'flex', alignItems: 'center', gap: 10,
    padding: '7px 0', borderBottom: '1px solid rgba(255,255,255,0.04)',
  },
  agentRow: {
    display: 'grid',
    gridTemplateColumns: '300px 1fr',
    gap: 16,
    marginTop: 16,
  },
  agentRow2: {
    display: 'flex', alignItems: 'center', gap: 8,
    padding: '4px 0', borderBottom: '1px solid rgba(255,255,255,0.04)',
  },
  statusDot: {
    width: 7, height: 7, borderRadius: '50%', flexShrink: 0,
  },
}
