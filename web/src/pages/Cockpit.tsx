import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  api, type BackupJob, type BackupPolicy, type BackupRepository, type RestoreTest, type System,
} from '../api'
import { KpiCard } from '../components/cockpit/KpiCard'
import { LiveStatusCard } from '../components/cockpit/LiveStatusCard'
import { RunBackupPanel } from '../components/cockpit/RunBackupPanel'
import { RepositoryCard, type RepoLastSuccess } from '../components/cockpit/RepositoryCard'
import { RecentJobsTable } from '../components/cockpit/RecentJobsTable'
import { AlertCard } from '../components/cockpit/AlertCard'
import { CancelDialog, type CancelTarget } from '../components/cockpit/CancelDialog'
import { friendlyLoc, isOnline, type SystemStatus } from '../components/cockpit/util'

const QUICK_ACTIONS: { label: string; icon: string; to: string }[] = [
  { label: 'Restore-Tests',      icon: '🛡', to: '/restore-tests' },
  { label: 'Policies verwalten', icon: '📋', to: '/policies' },
  { label: 'Systeme verwalten',  icon: '🖥', to: '/systems' },
  { label: 'Repository verwalten', icon: '🗄', to: '/repositories' },
  { label: 'Einstellungen',      icon: '⚙', to: '/settings' },
]

export function Cockpit() {
  const nav = useNavigate()

  const [systems,      setSystems]      = useState<System[]>([])
  const [jobs,         setJobs]         = useState<BackupJob[]>([])
  const [policies,     setPolicies]     = useState<BackupPolicy[]>([])
  const [repos,        setRepos]        = useState<BackupRepository[]>([])
  const [restoreTests, setRestoreTests] = useState<RestoreTest[]>([])
  const [loading,      setLoading]      = useState(true)
  const [now,          setNow]          = useState(() => new Date())

  const [selSystem,    setSelSystem]    = useState('')
  const [selPolicy,    setSelPolicy]    = useState('')
  const [starting,     setStarting]     = useState(false)
  const [startErr,     setStartErr]     = useState<string | null>(null)

  const [cancelTarget, setCancelTarget] = useState<CancelTarget | null>(null)
  const [cancelReason, setCancelReason] = useState('Windows-Update')
  const [cancelling,   setCancelling]   = useState(false)

  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const load = useCallback(() =>
    Promise.all([api.systems(), api.jobs(), api.policies(), api.repositories(), api.restoreTests()])
      .then(([sys, j, p, r, rt]) => { setSystems(sys); setJobs(j); setPolicies(p); setRepos(r); setRestoreTests(rt) })
      .catch(() => { /* keep last good data; per-section empty states handle the rest */ })
      .finally(() => setLoading(false))
  , [])

  useEffect(() => { load() }, [load])

  // Live clock (header)
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 30_000)
    return () => clearInterval(t)
  }, [])

  // Auto-refresh every 3 s while a job is active
  const hasActive = jobs.some(j => j.Status === 'running' || j.Status === 'pending')
  useEffect(() => {
    if (timerRef.current) clearInterval(timerRef.current)
    if (hasActive) timerRef.current = setInterval(load, 3000)
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [hasActive, load])

  // ─── Derived ──────────────────────────────────────────────────────────────

  const statusPerSystem: SystemStatus[] = useMemo(() => systems.map(sys => {
    const sysJobs = jobs.filter(j => j.SystemID === sys.ID)
    const running = sysJobs.find(j => j.Status === 'running') ?? null
    const pending = sysJobs.find(j => j.Status === 'pending') ?? null
    const finished = sysJobs
      .filter(j => j.Status === 'success' || j.Status === 'failed' || j.Status === 'cancelled')
      .sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())
    return { system: sys, running, pending, lastJob: finished[0] ?? null }
  }), [systems, jobs])

  const policyName = useCallback((id?: string) =>
    id ? (policies.find(p => p.ID === id)?.Name ?? id.slice(0, 8) + '…') : '—', [policies])
  const systemName = useCallback((id: string) =>
    systems.find(sy => sy.ID === id)?.Hostname ?? id.slice(0, 8) + '…', [systems])

  // KPI numbers — all derived from real data
  const onlineCount  = systems.filter(isOnline).length
  const runningCount = jobs.filter(j => j.Status === 'running').length
  const pendingCount = jobs.filter(j => j.Status === 'pending').length
  const errorCount   = statusPerSystem.filter(ss => !ss.running && !ss.pending && ss.lastJob?.Status === 'failed').length
  const rtTotal      = restoreTests.length
  const rtOk         = restoreTests.filter(t => t.Status === 'success').length
  const rtPct        = rtTotal > 0 ? Math.round((rtOk / rtTotal) * 100) : null

  const recentJobs = useMemo(() => [...jobs]
    .filter(j => j.Status !== 'pending')
    .sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())
    .slice(0, 6), [jobs])

  // Last successful backup per repository (job → policy → repository)
  const repoLastSuccess = useMemo(() => {
    const map = new Map<string, RepoLastSuccess>()
    for (const job of jobs) {
      if (job.Status !== 'success') continue
      const repoID = policies.find(p => p.ID === job.PolicyID)?.RepositoryID
      if (!repoID) continue
      const at = job.FinishedAt ?? job.CreatedAt
      const prev = map.get(repoID)
      if (!prev || new Date(at).getTime() > new Date(prev.at).getTime()) {
        map.set(repoID, { at, bytes: job.BytesUploaded })
      }
    }
    return map
  }, [jobs, policies])

  const selectedRepoLabel = useMemo(() => {
    const repoID = policies.find(p => p.ID === selPolicy)?.RepositoryID
    if (!repoID) return null
    const repo = repos.find(r => r.ID === repoID)
    return repo ? friendlyLoc(repo.Location) : null
  }, [selPolicy, policies, repos])

  const timeStr = now.toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })
  const dateStr = now.toLocaleDateString('de-DE', { day: '2-digit', month: 'long', year: 'numeric' })

  // ─── Actions ──────────────────────────────────────────────────────────────

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

  function requestStop(job: BackupJob, system: System) {
    setCancelTarget({ job, system, policy: policies.find(p => p.ID === job.PolicyID) ?? null })
    setCancelReason('Windows-Update')
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

  // ─── Render ───────────────────────────────────────────────────────────────

  return (
    <div style={s.root}>

      {/* Header */}
      <header style={s.header}>
        <div>
          <div style={s.titleRow}>
            <span style={s.brand}>OpenSourceBackup</span>
            <span style={s.brandAccent}>Cockpit</span>
          </div>
          <div style={s.subtitle}>Backup Operations · Alltags-Steuerung</div>
        </div>
        <div style={s.headerRight}>
          <div style={s.liveChip}>
            <div style={{ ...s.liveDot, ...(hasActive ? s.livePulse : {}) }} />
            <span>{hasActive ? 'Live · 3s' : 'Idle'}</span>
          </div>
          <div style={s.clock}>
            <div style={s.clockTime}>{timeStr}</div>
            <div style={s.clockDate}>{dateStr}</div>
          </div>
          <button style={s.refreshBtn} onClick={() => load()} title="Aktualisieren">↻</button>
        </div>
      </header>

      {/* KPI row */}
      <div className="ck-kpis">
        <KpiCard
          icon="🖥" tone="systems" label="Systeme" value={systems.length}
          sub={`${onlineCount} von ${systems.length} online`}
          onDetails={() => nav('/systems')}
        />
        <KpiCard
          icon="▶" tone="running" label="Laufende Jobs" value={runningCount}
          sub={pendingCount > 0 ? `${pendingCount} wartend` : 'aktuell aktiv'}
          onDetails={() => nav('/jobs')}
        />
        <KpiCard
          icon="⚠" tone="error" label="Fehler" value={errorCount} emphasise={errorCount > 0}
          sub={errorCount > 0 ? 'benötigt Aufmerksamkeit' : 'alles in Ordnung'}
          onDetails={() => nav('/jobs')}
        />
        <KpiCard
          icon="🛡" tone="verify" label="Restore-Verifikation"
          value={rtPct !== null ? `${rtPct}%` : '—'}
          sub={rtTotal > 0 ? `${rtOk} von ${rtTotal} erfolgreich` : 'keine Tests'}
          onDetails={() => nav('/restore-tests')}
        />
      </div>

      {/* Main: content + sidebar */}
      <div className="ck-main">

        {/* Left column */}
        <div style={s.col}>
          {/* Live status */}
          <section>
            <div style={s.sectionLabel}>Live-Status</div>
            {loading && statusPerSystem.length === 0 ? (
              <div style={s.panel}><div style={s.loadingMsg}>Lade…</div></div>
            ) : statusPerSystem.length === 0 ? (
              <div style={s.panel}>
                <EmptyState icon="🖥" title="Keine Systeme registriert"
                  sub="Zuerst ein System hinzufügen und den Agent starten."
                  btn="System hinzufügen →" onClick={() => nav('/systems')} />
              </div>
            ) : (
              <div style={s.liveList}>
                {statusPerSystem.map(ss => (
                  <LiveStatusCard key={ss.system.ID} ss={ss} policyName={policyName}
                    onStop={job => requestStop(job, ss.system)} />
                ))}
              </div>
            )}
          </section>

          {/* Run backup */}
          <section>
            <div style={s.sectionLabel}>Backup jetzt starten</div>
            <RunBackupPanel
              systems={systems} policies={policies}
              selSystem={selSystem} selPolicy={selPolicy} repoLabel={selectedRepoLabel}
              starting={starting} startErr={startErr}
              onSystem={v => { setSelSystem(v); setStartErr(null) }}
              onPolicy={v => { setSelPolicy(v); setStartErr(null) }}
              onStart={startBackup}
            />
          </section>

          {/* Recent jobs */}
          <section>
            <div style={s.sectionLabel}>Letzte Jobs</div>
            <div style={s.panel}>
              <RecentJobsTable jobs={recentJobs} systemName={systemName} policyName={policyName}
                onDetails={() => nav('/jobs')} />
              {recentJobs.length > 0 && (
                <button style={s.allJobsBtn} onClick={() => nav('/jobs')}>Alle Jobs anzeigen →</button>
              )}
            </div>
          </section>
        </div>

        {/* Right sidebar */}
        <aside style={s.sidebar}>
          {/* Repositories */}
          <section>
            <div style={s.sectionLabel}>Repositories</div>
            {repos.length === 0 ? (
              <div style={s.panel}>
                <EmptyState icon="🗄" title="Kein Repository konfiguriert"
                  sub="Ohne Repository können keine Backups gestartet werden."
                  btn="Repository hinzufügen →" onClick={() => nav('/repositories')} />
              </div>
            ) : (
              <div style={s.repoList}>
                {repos.map(repo => (
                  <RepositoryCard key={repo.ID} repo={repo}
                    lastSuccess={repoLastSuccess.get(repo.ID) ?? null}
                    onDetails={() => nav('/repositories')} />
                ))}
              </div>
            )}
          </section>

          {/* Quick actions */}
          <section>
            <div style={s.sectionLabel}>Schnellaktionen</div>
            <div style={s.quickCard}>
              {QUICK_ACTIONS.map(a => (
                <button key={a.to} style={s.quickRow} onClick={() => nav(a.to)}>
                  <span style={s.quickIcon}>{a.icon}</span>
                  <span style={s.quickLabel}>{a.label}</span>
                  <span style={s.quickArrow}>›</span>
                </button>
              ))}
            </div>
          </section>

          {/* Alerts */}
          {errorCount > 0 && (
            <section>
              <div style={s.sectionLabel}>Hinweise</div>
              <AlertCard errorCount={errorCount} onDetails={() => nav('/jobs')} />
            </section>
          )}
        </aside>
      </div>

      {/* Cancel dialog */}
      {cancelTarget && (
        <CancelDialog
          target={cancelTarget} reason={cancelReason} cancelling={cancelling}
          onReason={setCancelReason} onDismiss={() => setCancelTarget(null)} onConfirm={confirmCancel}
        />
      )}

      <style>{`
        @keyframes pulse-ring {
          0%,100% { box-shadow: 0 0 0 0 rgba(56,189,248,.5) }
          60%     { box-shadow: 0 0 0 5px rgba(56,189,248,0) }
        }
        .ck-kpis { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin-bottom: 22px; }
        .ck-main { display: grid; grid-template-columns: minmax(0,1fr) 340px; gap: 22px; align-items: start; }
        @media (max-width: 1180px) { .ck-kpis { grid-template-columns: repeat(2, 1fr); } }
        @media (max-width: 1024px) { .ck-main { grid-template-columns: 1fr; } }
        @media (max-width: 560px)  { .ck-kpis { grid-template-columns: 1fr; } }
      `}</style>
    </div>
  )
}

// ─── EmptyState ──────────────────────────────────────────────────────────────

function EmptyState({ icon, title, sub, btn, onClick }: { icon: string; title: string; sub: string; btn: string; onClick: () => void }) {
  return (
    <div style={s.empty}>
      <div style={s.emptyIcon}>{icon}</div>
      <div style={s.emptyTitle}>{title}</div>
      <div style={s.emptySub}>{sub}</div>
      <button style={s.emptyBtn} onClick={onClick}>{btn}</button>
    </div>
  )
}

// ─── Styles ──────────────────────────────────────────────────────────────────

const s: Record<string, React.CSSProperties> = {
  root:       { padding: '28px 32px', maxWidth: 1440, margin: '0 auto' },

  // Header
  header:     { display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 16, marginBottom: 24, flexWrap: 'wrap' },
  titleRow:   { display: 'flex', alignItems: 'baseline', gap: 10 },
  brand:      { fontSize: 24, fontWeight: 800, color: 'var(--text)', letterSpacing: '-0.4px' },
  brandAccent:{ fontSize: 24, fontWeight: 800, color: 'var(--accent)', letterSpacing: '-0.4px' },
  subtitle:   { fontSize: 13, color: 'var(--text-muted)', marginTop: 4 },
  headerRight:{ display: 'flex', alignItems: 'center', gap: 14 },
  liveChip:   { display: 'flex', alignItems: 'center', gap: 7, padding: '6px 14px', borderRadius: 20, background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', fontSize: 12, color: 'var(--text-muted)' },
  liveDot:    { width: 7, height: 7, borderRadius: '50%', background: 'var(--text-dim)', flexShrink: 0 },
  livePulse:  { background: 'var(--running)', animation: 'pulse-ring 1.5s infinite' },
  clock:      { textAlign: 'right' },
  clockTime:  { fontSize: 18, fontWeight: 700, color: 'var(--text)', lineHeight: 1.1, fontVariantNumeric: 'tabular-nums' },
  clockDate:  { fontSize: 12, color: 'var(--text-dim)', marginTop: 2 },
  refreshBtn: { width: 38, height: 38, borderRadius: 10, background: 'var(--bg-card)', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 18, cursor: 'pointer', flexShrink: 0 },

  // Columns
  col:        { display: 'flex', flexDirection: 'column', gap: 22, minWidth: 0 },
  sidebar:    { display: 'flex', flexDirection: 'column', gap: 18, minWidth: 0 },
  sectionLabel:{ fontSize: 13, fontWeight: 700, color: 'var(--text)', marginBottom: 12 },

  panel:      { background: 'linear-gradient(180deg, rgba(21,28,46,0.95) 0%, rgba(10,15,27,0.95) 100%)', border: '1px solid var(--border)', borderRadius: 16, overflow: 'hidden' },
  loadingMsg: { padding: 40, textAlign: 'center', color: 'var(--text-muted)', fontSize: 13 },

  liveList:   { display: 'flex', flexDirection: 'column', gap: 12 },
  repoList:   { display: 'flex', flexDirection: 'column', gap: 12 },

  allJobsBtn: { display: 'block', width: '100%', padding: '12px', background: 'rgba(255,255,255,0.02)', border: 'none', borderTop: '1px solid var(--border)', color: 'var(--running)', fontSize: 13, fontWeight: 600, cursor: 'pointer' },

  // Quick actions
  quickCard:  { background: 'linear-gradient(180deg, rgba(21,28,46,0.95) 0%, rgba(10,15,27,0.95) 100%)', border: '1px solid var(--border)', borderRadius: 14, overflow: 'hidden' },
  quickRow:   { display: 'flex', alignItems: 'center', gap: 12, width: '100%', padding: '13px 16px', background: 'none', border: 'none', borderBottom: '1px solid rgba(255,255,255,0.04)', color: 'var(--text)', fontSize: 13, cursor: 'pointer', textAlign: 'left' },
  quickIcon:  { fontSize: 16, width: 22, textAlign: 'center', flexShrink: 0, opacity: 0.9 },
  quickLabel: { flex: 1, fontWeight: 500 },
  quickArrow: { color: 'var(--text-dim)', fontSize: 16 },

  // Empty states
  empty:      { padding: '40px 20px', textAlign: 'center' },
  emptyIcon:  { fontSize: 32, opacity: 0.3, marginBottom: 12 },
  emptyTitle: { fontSize: 14, fontWeight: 700, color: 'var(--text-muted)', marginBottom: 4 },
  emptySub:   { fontSize: 12, color: 'var(--text-dim)', marginBottom: 16 },
  emptyBtn:   { padding: '8px 16px', borderRadius: 8, background: 'transparent', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 12, cursor: 'pointer' },
}
