import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, timeAgo, type BackupJob, type BackupPolicy, type BackupRepository, type System } from '../api'

// ─── Types ─────────────────────────────────────────────────────────────────

interface SystemStatus {
  system:  System
  running: BackupJob | null  // currently running
  pending: BackupJob | null  // queued / waiting for agent
  lastJob: BackupJob | null  // most recent finished job
}

interface CancelTarget {
  job:    BackupJob
  system: System
  policy: BackupPolicy | null
}

// ─── Cockpit ───────────────────────────────────────────────────────────────

export function Cockpit() {
  const nav = useNavigate()

  const [systems,      setSystems]      = useState<System[]>([])
  const [jobs,         setJobs]         = useState<BackupJob[]>([])
  const [policies,     setPolicies]     = useState<BackupPolicy[]>([])
  const [repos,        setRepos]        = useState<BackupRepository[]>([])
  const [loading,      setLoading]      = useState(true)
  const [selSystem,    setSelSystem]    = useState('')
  const [selPolicy,    setSelPolicy]    = useState('')
  const [starting,     setStarting]     = useState(false)
  const [startErr,     setStartErr]     = useState<string | null>(null)
  const [cancelTarget, setCancelTarget] = useState<CancelTarget | null>(null)
  const [cancelReason, setCancelReason] = useState('Windows-Update')
  const [cancelling,   setCancelling]   = useState(false)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const load = useCallback(() =>
    Promise.all([api.systems(), api.jobs(), api.policies(), api.repositories()])
      .then(([sys, j, p, r]) => { setSystems(sys); setJobs(j); setPolicies(p); setRepos(r) })
      .finally(() => setLoading(false))
  , [])

  useEffect(() => { load() }, [load])

  // Auto-refresh every 3 s while a job is active
  useEffect(() => {
    if (timerRef.current) clearInterval(timerRef.current)
    const hasActive = jobs.some(j => j.Status === 'running' || j.Status === 'pending')
    if (hasActive) timerRef.current = setInterval(load, 3000)
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [jobs, load])

  // ─── Derived ────────────────────────────────────────────────────────────

  const statusPerSystem: SystemStatus[] = systems.map(sys => {
    const sysJobs = jobs.filter(j => j.SystemID === sys.ID)
    const running = sysJobs.find(j => j.Status === 'running') ?? null
    const pending = sysJobs.find(j => j.Status === 'pending') ?? null
    const finished = sysJobs
      .filter(j => j.Status === 'success' || j.Status === 'failed' || j.Status === 'cancelled')
      .sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())
    return { system: sys, running, pending, lastJob: finished[0] ?? null }
  })

  const policyName = (id?: string) =>
    id ? (policies.find(p => p.ID === id)?.Name ?? id.slice(0, 8) + '…') : '—'

  const hasActive = jobs.some(j => j.Status === 'running' || j.Status === 'pending')

  // ─── Actions ────────────────────────────────────────────────────────────

  async function startBackup() {
    if (!selSystem || !selPolicy) { setStartErr('Bitte System und Policy auswählen.'); return }
    setStarting(true); setStartErr(null)
    try {
      await api.createJob(selSystem, selPolicy)
      setSelSystem(''); setSelPolicy('')
      await load()
    } catch { setStartErr('Backup konnte nicht gestartet werden.') }
    finally { setStarting(false) }
  }

  async function confirmCancel() {
    if (!cancelTarget) return
    setCancelling(true)
    try {
      await api.cancelJob(cancelTarget.job.ID, cancelReason)
      setCancelTarget(null)
      await load()
    } catch { /* job may already have finished */ }
    finally { setCancelling(false) }
  }

  // ─── Render ─────────────────────────────────────────────────────────────

  return (
    <div style={s.root}>

      {/* Header */}
      <div style={s.header}>
        <div>
          <div style={s.title}>Cockpit</div>
          <div style={s.subtitle}>Backup Operations · Alltags-Steuerung</div>
        </div>
        <div style={s.liveChip}>
          <div style={{ ...s.liveDot, ...(hasActive ? s.livePulse : {}) }} />
          <span>{hasActive ? 'Live · 3s' : 'Idle'}</span>
        </div>
      </div>

      <div style={s.body}>

        {/* ── SECTION 1: Live Status ── */}
        <div style={s.section}>
          <div style={s.sectionLabel}>LIVE STATUS</div>
          <div style={s.card}>
            {loading ? (
              <div style={s.emptyMsg}>Lade…</div>
            ) : statusPerSystem.length === 0 ? (
              <div style={s.emptyState}>
                <div style={s.emptyIcon}>🖥</div>
                <div style={s.emptyTitle}>Keine Systeme registriert</div>
                <div style={s.emptySub}>Zuerst ein System hinzufügen und den Agent starten.</div>
                <button style={s.emptyBtn} onClick={() => nav('/systems')}>System hinzufügen →</button>
              </div>
            ) : (
              <table style={s.table}>
                <thead>
                  <tr>
                    <th style={s.th}>System</th>
                    <th style={s.th}>Status</th>
                    <th style={{ ...s.th, minWidth: 160 }}>Fortschritt</th>
                    <th style={s.th}>Tempo</th>
                    <th style={s.th}>Letztes Backup</th>
                    <th style={{ ...s.th, width: 80 }}></th>
                  </tr>
                </thead>
                <tbody>
                  {statusPerSystem.map(ss => (
                    <SystemRow
                      key={ss.system.ID}
                      ss={ss}
                      policyName={policyName}
                      onStop={job => {
                        const pol = policies.find(p => p.ID === job.PolicyID) ?? null
                        setCancelTarget({ job, system: ss.system, policy: pol })
                        setCancelReason('Windows-Update')
                      }}
                    />
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

        {/* ── SECTION 2: Run Backup ── */}
        <div style={s.section}>
          <div style={s.sectionLabel}>JETZT BACKUPEN</div>
          <div style={s.card}>
            <div style={s.runRow}>
              <div style={s.field}>
                <label style={s.label}>System</label>
                <select
                  value={selSystem}
                  onChange={e => { setSelSystem(e.target.value); setStartErr(null) }}
                  style={s.select}
                >
                  <option value="">— wählen —</option>
                  {systems.map(sys => (
                    <option key={sys.ID} value={sys.ID}>{sys.Hostname}</option>
                  ))}
                </select>
              </div>

              <div style={s.field}>
                <label style={s.label}>Policy</label>
                <select
                  value={selPolicy}
                  onChange={e => { setSelPolicy(e.target.value); setStartErr(null) }}
                  style={s.select}
                >
                  <option value="">— wählen —</option>
                  {policies.map(p => (
                    <option key={p.ID} value={p.ID}>{p.Name}</option>
                  ))}
                </select>
              </div>

              <div style={s.startWrap}>
                <label style={s.label}>&nbsp;</label>
                <button
                  onClick={startBackup}
                  disabled={starting || !selSystem || !selPolicy}
                  style={{
                    ...s.startBtn,
                    ...(!selSystem || !selPolicy || starting ? s.startBtnOff : {}),
                  }}
                >
                  {starting ? '…' : '▶ Start'}
                </button>
              </div>
            </div>
            {startErr && <div style={s.errBox}>{startErr}</div>}
          </div>
        </div>

        {/* ── SECTION 3: Repositories ── */}
        <div style={s.section}>
          <div style={s.sectionLabel}>REPOSITORIES</div>
          <div style={s.card}>
            {repos.length === 0 ? (
              <div style={s.emptyState}>
                <div style={s.emptyIcon}>▤</div>
                <div style={s.emptyTitle}>Kein Repository konfiguriert</div>
                <div style={s.emptySub}>Ohne Repository können keine Backups gestartet werden.</div>
                <button style={s.emptyBtn} onClick={() => nav('/repositories')}>Repository hinzufügen →</button>
              </div>
            ) : (
              <div style={s.repoList}>
                {repos.map(repo => (
                  <RepoRow key={repo.ID} repo={repo} onDetails={() => nav('/repositories')} />
                ))}
              </div>
            )}
          </div>
        </div>

      </div>

      {/* ── Cancel Dialog ─────────────────────────────────────────────── */}
      {cancelTarget && (
        <div style={s.overlay}>
          <div style={s.dialog}>
            <div style={s.dlgHead}>
              <span style={s.dlgTitle}>⏹ Backup stoppen</span>
            </div>
            <div style={s.dlgBody}>
              <div style={s.dlgRow}>
                <span style={s.dlgKey}>System</span>
                <span style={s.dlgVal}>{cancelTarget.system.Hostname}</span>
              </div>
              {cancelTarget.policy && (
                <div style={s.dlgRow}>
                  <span style={s.dlgKey}>Policy</span>
                  <span style={s.dlgVal}>{cancelTarget.policy.Name}</span>
                </div>
              )}
              <div style={{ marginTop: 18 }}>
                <div style={s.label}>Grund</div>
                <div style={s.reasonGrid}>
                  {CANCEL_REASONS.map(r => (
                    <button
                      key={r}
                      onClick={() => setCancelReason(r)}
                      style={{ ...s.reasonBtn, ...(cancelReason === r ? s.reasonOn : {}) }}
                    >
                      {r}
                    </button>
                  ))}
                </div>
              </div>
              <div style={s.dlgNote}>
                Stop = kontrollierter Abbruch. Der Job erhält Status „cancelled" — kein Fehler.
                Der nächste geplante Lauf startet normal.
              </div>
            </div>
            <div style={s.dlgFoot}>
              <button
                onClick={() => setCancelTarget(null)}
                disabled={cancelling}
                style={s.dlgDismiss}
              >
                Abbrechen
              </button>
              <button onClick={confirmCancel} disabled={cancelling} style={s.dlgStop}>
                {cancelling ? 'Stoppe…' : '⏹ Backup stoppen'}
              </button>
            </div>
          </div>
        </div>
      )}

      <style>{`
        @keyframes pulse-ring {
          0%,100% { box-shadow: 0 0 0 0 rgba(56,189,248,.5) }
          60%     { box-shadow: 0 0 0 5px rgba(56,189,248,0) }
        }
      `}</style>
    </div>
  )
}

// ─── SystemRow ─────────────────────────────────────────────────────────────

interface RowProps {
  ss:         SystemStatus
  policyName: (id?: string) => string
  onStop:     (job: BackupJob) => void
}

function SystemRow({ ss, policyName, onStop }: RowProps) {
  const { system, running, pending, lastJob } = ss
  const active = running ?? pending

  if (active) {
    const isRunning = active.Status === 'running'
    const pct = active.ProgressPercent ?? 0
    const bps = active.ProgressThroughputBps ?? 0

    return (
      <tr style={s.tr}>
        <td style={s.td}><span style={s.host}>{system.Hostname}</span></td>
        <td style={s.td}>
          <span
            style={{
              ...s.statusDot,
              background: isRunning ? 'var(--running)' : 'var(--warning)',
              boxShadow: isRunning ? '0 0 5px var(--running)' : 'none',
            }}
          />
          <span style={{ fontSize: 12, fontWeight: 600, color: isRunning ? 'var(--running)' : 'var(--warning)' }}>
            {isRunning ? policyName(active.PolicyID) : 'Warte auf Agent…'}
          </span>
        </td>
        <td style={s.td}>
          {isRunning ? (
            <div style={s.progWrap}>
              <div style={s.progBar}>
                <div style={{ ...s.progFill, width: `${Math.min(Math.max(pct, 0), 100)}%` }} />
              </div>
              <span style={s.pct}>{pct > 0 ? `${Math.round(pct)}%` : '…'}</span>
            </div>
          ) : (
            <span style={s.dim}>—</span>
          )}
        </td>
        <td style={s.td}>
          {isRunning && bps > 0
            ? <span style={s.throughput}>{fmtBps(bps)}</span>
            : <span style={s.dim}>—</span>}
        </td>
        <td style={s.td}>
          <span style={s.dim}>{lastJob ? timeAgo(lastJob.FinishedAt ?? lastJob.CreatedAt) : '—'}</span>
        </td>
        <td style={s.tdRight}>
          {isRunning && (
            <button onClick={() => onStop(active)} style={s.stopBtn}>⏹ Stop</button>
          )}
        </td>
      </tr>
    )
  }

  // Finished / never run
  const status = lastJob?.Status ?? 'none'
  const isOk   = status === 'success'
  const isFail = status === 'failed'

  return (
    <tr style={s.tr}>
      <td style={s.td}><span style={s.host}>{system.Hostname}</span></td>
      <td style={s.td}>
        {status === 'none'      && <span style={s.dim}>Noch kein Backup</span>}
        {isOk                   && <span style={{ fontSize:12, fontWeight:600, color:'var(--success)' }}>● Idle · OK</span>}
        {isFail                 && <span style={{ fontSize:12, fontWeight:600, color:'var(--error)' }}>⚠ Fehlgeschlagen</span>}
        {status === 'cancelled' && <span style={{ fontSize:12, fontWeight:600, color:'var(--warning)' }}>○ Abgebrochen</span>}
      </td>
      <td style={s.td}><span style={s.dim}>—</span></td>
      <td style={s.td}><span style={s.dim}>—</span></td>
      <td style={s.td}>
        {lastJob && (
          <span style={{ ...s.dim, ...(isOk ? { color:'var(--success)' } : {}), ...(isFail ? { color:'var(--error)' } : {}) }}>
            {timeAgo(lastJob.FinishedAt ?? lastJob.CreatedAt)}{isOk ? ' ✓' : ''}
          </span>
        )}
        {!lastJob && <span style={s.dim}>—</span>}
        {isFail && lastJob?.ErrorSummary && (
          <div style={s.errSnip} title={lastJob.ErrorSummary}>
            {lastJob.ErrorSummary.slice(0, 64)}{lastJob.ErrorSummary.length > 64 ? '…' : ''}
          </div>
        )}
      </td>
      <td style={s.tdRight} />
    </tr>
  )
}

// ─── RepoRow ───────────────────────────────────────────────────────────────

function RepoRow({ repo, onDetails }: { repo: BackupRepository; onDetails: () => void }) {
  const immutable = repo.ImmutableMode !== 'none' && repo.ImmutableMode !== 'unknown'
  const encrypted = !!repo.EncryptionMode && repo.EncryptionMode !== 'none'

  return (
    <div style={s.repoRow}>
      <div style={s.repoInfo}>
        <span style={s.repoName}>{friendlyLoc(repo.Location)}</span>
        <span style={s.repoType}>{repo.Type}</span>
      </div>
      <div style={s.repoBadges}>
        {immutable && <span style={{ ...s.badge, ...s.badgePurple }}>🔒 immutable</span>}
        {encrypted  && <span style={{ ...s.badge, ...s.badgeGreen }}>🔑 encrypted</span>}
        {!immutable && <span style={{ ...s.badge, ...s.badgeOrange }}>⚠ kein Lock</span>}
      </div>
      <button onClick={onDetails} style={s.repoBtn}>Details →</button>
    </div>
  )
}

// ─── Helpers ────────────────────────────────────────────────────────────────

const CANCEL_REASONS = ['Windows-Update', 'Maschine instabil', 'Arbeitsbetrieb', 'Sonstiges']

function fmtBps(bps: number): string {
  if (bps < 1024)          return `${bps} B/s`
  if (bps < 1024 * 1024)   return `${(bps / 1024).toFixed(1)} KB/s`
  return `${(bps / 1024 / 1024).toFixed(1)} MB/s`
}

function friendlyLoc(loc: string): string {
  if (/^(s3|b2|azure|gs):/.test(loc)) return loc
  const parts = loc.replace(/\\/g, '/').split('/').filter(Boolean)
  return parts.length > 2 ? `…/${parts.slice(-2).join('/')}` : loc
}

// ─── Styles ─────────────────────────────────────────────────────────────────

const s: Record<string, React.CSSProperties> = {
  root:      { padding: '28px 36px', maxWidth: 960 },

  // Header
  header:    { display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 28 },
  title:     { fontSize: 22, fontWeight: 800, color: 'var(--text)', lineHeight: 1.2 },
  subtitle:  { fontSize: 13, color: 'var(--text-muted)', marginTop: 4 },
  liveChip:  { display: 'flex', alignItems: 'center', gap: 7, padding: '6px 14px', borderRadius: 20, background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', fontSize: 12, color: 'var(--text-muted)' },
  liveDot:   { width: 7, height: 7, borderRadius: '50%', background: 'var(--text-dim)', flexShrink: 0 },
  livePulse: { background: 'var(--running)', animation: 'pulse-ring 1.5s infinite' },

  // Layout
  body:        { display: 'flex', flexDirection: 'column', gap: 20 },
  section:     {},
  sectionLabel:{ fontSize: 9, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.14em', marginBottom: 8, paddingLeft: 2 },
  card:        { background: 'linear-gradient(180deg, rgba(21,28,46,0.95) 0%, rgba(10,15,27,0.95) 100%)', border: '1px solid var(--border)', borderRadius: 12, overflow: 'hidden' },

  // Table
  table:   { width: '100%', borderCollapse: 'collapse' },
  th:      { padding: '9px 16px', fontSize: 9, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.1em', borderBottom: '1px solid var(--border)', textAlign: 'left', background: 'rgba(255,255,255,0.02)' },
  tr:      { borderBottom: '1px solid rgba(255,255,255,0.035)' },
  td:      { padding: '12px 16px', fontSize: 13, color: 'var(--text-muted)', verticalAlign: 'middle' },
  tdRight: { padding: '8px 16px', verticalAlign: 'middle', textAlign: 'right' },

  host:      { fontWeight: 700, color: 'var(--text)', fontSize: 13 },
  statusDot: { display: 'inline-block', width: 7, height: 7, borderRadius: '50%', marginRight: 7, verticalAlign: 'middle', flexShrink: 0 },
  dim:       { color: 'var(--text-dim)', fontSize: 12 },
  errSnip:   { marginTop: 3, fontSize: 10, color: 'var(--error)', fontFamily: 'var(--font-mono)', opacity: 0.85 },

  // Progress
  progWrap:  { display: 'flex', alignItems: 'center', gap: 8 },
  progBar:   { flex: 1, height: 4, background: 'var(--border)', borderRadius: 2, overflow: 'hidden', minWidth: 80 },
  progFill:  { height: '100%', background: 'var(--running)', borderRadius: 2, transition: 'width 0.4s ease' },
  pct:       { fontSize: 11, color: 'var(--running)', fontWeight: 700, width: 32, textAlign: 'right', flexShrink: 0 },
  throughput:{ fontSize: 12, color: 'var(--accent-teal)', fontWeight: 600, fontFamily: 'var(--font-mono)' },
  stopBtn:   { padding: '5px 12px', borderRadius: 6, background: 'rgba(239,68,68,0.1)', color: 'var(--error)', border: '1px solid rgba(239,68,68,0.3)', fontSize: 12, fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap' },

  // Run backup
  runRow:    { display: 'flex', gap: 16, padding: '20px 20px 20px', alignItems: 'flex-end', flexWrap: 'wrap' },
  field:     { display: 'flex', flexDirection: 'column', gap: 6, flex: 1, minWidth: 180 },
  label:     { fontSize: 10, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.1em' },
  select:    { padding: '9px 12px', background: 'var(--bg)', border: '1px solid var(--border)', borderRadius: 7, color: 'var(--text)', fontSize: 13, cursor: 'pointer' },
  startWrap: { display: 'flex', flexDirection: 'column', gap: 6, flexShrink: 0 },
  startBtn:  { padding: '9px 28px', borderRadius: 7, background: 'var(--accent)', color: '#fff', border: 'none', fontSize: 14, fontWeight: 700, cursor: 'pointer', whiteSpace: 'nowrap' },
  startBtnOff:{ opacity: 0.4, cursor: 'not-allowed' },
  errBox:    { margin: '0 20px 20px', background: 'rgba(239,68,68,0.08)', border: '1px solid rgba(239,68,68,0.2)', borderRadius: 7, padding: '8px 12px', fontSize: 13, color: 'var(--error)' },

  // Repositories
  repoList:  { display: 'flex', flexDirection: 'column' },
  repoRow:   { display: 'flex', alignItems: 'center', gap: 14, padding: '14px 20px', borderBottom: '1px solid rgba(255,255,255,0.035)' },
  repoInfo:  { flex: 1, minWidth: 0 },
  repoName:  { display: 'block', fontWeight: 700, color: 'var(--text)', fontSize: 13, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' },
  repoType:  { display: 'block', fontSize: 10, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.1em', marginTop: 2 },
  repoBadges:{ display: 'flex', gap: 6, flexShrink: 0 },
  badge:     { padding: '3px 8px', borderRadius: 4, fontSize: 10, fontWeight: 700, letterSpacing: '0.03em' },
  badgePurple:{ background: 'rgba(139,92,246,0.12)', color: '#a78bfa', border: '1px solid rgba(139,92,246,0.2)' },
  badgeGreen: { background: 'rgba(137,189,40,0.1)',  color: 'var(--accent)', border: '1px solid rgba(137,189,40,0.25)' },
  badgeOrange:{ background: 'rgba(245,158,11,0.1)',  color: 'var(--warning)', border: '1px solid rgba(245,158,11,0.2)' },
  repoBtn:   { padding: '5px 12px', borderRadius: 6, background: 'transparent', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 12, cursor: 'pointer', flexShrink: 0 },

  // Empty states
  emptyMsg:    { padding: 40, color: 'var(--text-muted)', textAlign: 'center', fontSize: 13 },
  emptyState:  { padding: '40px 20px', textAlign: 'center' },
  emptyIcon:   { fontSize: 32, opacity: 0.3, marginBottom: 12 },
  emptyTitle:  { fontSize: 14, fontWeight: 700, color: 'var(--text-muted)', marginBottom: 4 },
  emptySub:    { fontSize: 12, color: 'var(--text-dim)', marginBottom: 16 },
  emptyBtn:    { padding: '7px 16px', borderRadius: 6, background: 'transparent', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 12, cursor: 'pointer' },

  // Cancel dialog
  overlay:    { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.65)', backdropFilter: 'blur(4px)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 },
  dialog:     { width: 440, background: 'linear-gradient(180deg, rgba(21,28,46,0.99) 0%, rgba(10,15,27,0.99) 100%)', border: '1px solid var(--border)', borderRadius: 14, overflow: 'hidden' },
  dlgHead:    { padding: '18px 20px', borderBottom: '1px solid var(--border)', background: 'rgba(239,68,68,0.06)' },
  dlgTitle:   { fontSize: 14, fontWeight: 700, color: 'var(--error)', letterSpacing: '0.04em' },
  dlgBody:    { padding: '20px' },
  dlgRow:     { display: 'flex', gap: 12, alignItems: 'center', marginBottom: 8 },
  dlgKey:     { fontSize: 11, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em', width: 52, flexShrink: 0 },
  dlgVal:     { fontSize: 13, color: 'var(--text)', fontWeight: 600 },
  reasonGrid: { display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginTop: 8 },
  reasonBtn:  { padding: '9px 12px', borderRadius: 7, background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 12, fontWeight: 500, cursor: 'pointer', textAlign: 'left' },
  reasonOn:   { background: 'rgba(239,68,68,0.12)', border: '1px solid rgba(239,68,68,0.35)', color: 'var(--error)', fontWeight: 700 },
  dlgNote:    { marginTop: 16, fontSize: 11, color: 'var(--text-dim)', lineHeight: 1.6, padding: '10px 12px', background: 'rgba(255,255,255,0.02)', borderRadius: 6 },
  dlgFoot:    { display: 'flex', justifyContent: 'flex-end', gap: 10, padding: '16px 20px', borderTop: '1px solid var(--border)' },
  dlgDismiss: { padding: '8px 18px', borderRadius: 7, background: 'transparent', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 13, cursor: 'pointer' },
  dlgStop:    { padding: '8px 22px', borderRadius: 7, background: 'var(--error)', color: '#fff', border: 'none', fontSize: 13, fontWeight: 700, cursor: 'pointer' },
}
