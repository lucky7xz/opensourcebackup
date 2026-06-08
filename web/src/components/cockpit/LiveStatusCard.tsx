// One large status card per system. Replaces the old debug-style table row.
// Shows a left accent bar, system identity, prominent status, and — while a
// backup runs — a progress bar, throughput, data and a Stop button.
import { fmt, timeAgo, type BackupJob } from '../../api'
import { fmtBps, systemAccent, type SystemStatus } from './util'

interface LiveStatusCardProps {
  ss:         SystemStatus
  policyName: (id?: string) => string
  onStop:     (job: BackupJob) => void
}

export function LiveStatusCard({ ss, policyName, onStop }: LiveStatusCardProps) {
  const { system, running, pending, lastJob } = ss
  const accent = systemAccent(ss)

  return (
    <div style={{ ...s.card, borderLeft: `3px solid ${accent}` }}>
      {/* Identity */}
      <div style={s.identity}>
        <div style={{ ...s.iconCircle, color: accent, borderColor: accent }}>🖥</div>
        <div style={s.idText}>
          <div style={s.host}>{system.Hostname}</div>
          <div style={s.meta}>
            {system.OS ?? 'Unbekanntes OS'}
            {system.AgentVersion ? ` · Agent ${system.AgentVersion}` : ''}
          </div>
        </div>
      </div>

      {/* Status / progress / error */}
      <div style={s.body}>
        {running ? (
          <RunningBody job={running} policyName={policyName} onStop={onStop} accent={accent} />
        ) : pending ? (
          <div style={s.statusLine}>
            <span style={{ ...s.dot, background: 'var(--running)' }} />
            <span style={{ ...s.statusText, color: 'var(--running)' }}>Wartet auf Agent…</span>
          </div>
        ) : (
          <IdleBody lastJob={lastJob} />
        )}
      </div>
    </div>
  )
}

// ── Running ──────────────────────────────────────────────────────────────────

interface RunningBodyProps {
  job:        BackupJob
  policyName: (id?: string) => string
  onStop:     (job: BackupJob) => void
  accent:     string
}

function RunningBody({ job, policyName, onStop, accent }: RunningBodyProps) {
  const pct  = Math.min(Math.max(job.ProgressPercent ?? 0, 0), 100)
  const bps  = job.ProgressThroughputBps ?? 0
  const done = job.ProgressBytesDone ?? 0
  const tot  = job.ProgressBytesTotal ?? 0
  const files = job.ProgressFilesDone ?? 0

  return (
    <>
      <div style={s.runTop}>
        <div style={s.statusLine}>
          <span style={{ ...s.dot, background: accent, boxShadow: `0 0 6px ${accent}` }} />
          <span style={{ ...s.statusText, color: accent }}>Backup läuft</span>
          <span style={s.policy}>Policy: {policyName(job.PolicyID)}</span>
        </div>
        <button style={s.stopBtn} onClick={() => onStop(job)}>⏹ Stoppen</button>
      </div>

      <div style={s.progRow}>
        <div style={s.progBar}>
          <div style={{ ...s.progFill, width: `${pct}%`, background: accent }} />
        </div>
        <span style={{ ...s.pct, color: accent }}>{pct > 0 ? `${Math.round(pct)}%` : '…'}</span>
      </div>

      <div style={s.runMeta}>
        <span>{bps > 0 ? fmtBps(bps) : '—'}</span>
        <span style={s.sep}>·</span>
        <span>{fmt(done)} / {fmt(tot)}</span>
        {files > 0 && <><span style={s.sep}>·</span><span>{files.toLocaleString('de-DE')} Dateien</span></>}
        {job.StartedAt && <><span style={s.sep}>·</span><span>seit {timeAgo(job.StartedAt)}</span></>}
      </div>
    </>
  )
}

// ── Idle / finished ──────────────────────────────────────────────────────────

function IdleBody({ lastJob }: { lastJob: BackupJob | null }) {
  if (!lastJob) {
    return (
      <div style={s.statusLine}>
        <span style={{ ...s.dot, background: 'var(--text-dim)' }} />
        <span style={{ ...s.statusText, color: 'var(--text-dim)' }}>Noch kein Backup</span>
      </div>
    )
  }

  const when = timeAgo(lastJob.FinishedAt ?? lastJob.CreatedAt)

  if (lastJob.Status === 'failed') {
    return (
      <div>
        <div style={s.statusLine}>
          <span style={{ ...s.dot, background: 'var(--error)' }} />
          <span style={{ ...s.statusText, color: 'var(--error)' }}>Fehlgeschlagen</span>
          <span style={s.policyDim}>vor {when}</span>
        </div>
        {lastJob.ErrorSummary && <div style={s.errBox}>{lastJob.ErrorSummary}</div>}
      </div>
    )
  }

  if (lastJob.Status === 'cancelled') {
    return (
      <div style={s.statusLine}>
        <span style={{ ...s.dot, background: 'var(--warning)' }} />
        <span style={{ ...s.statusText, color: 'var(--warning)' }}>Abgebrochen</span>
        <span style={s.policyDim}>vor {when}</span>
      </div>
    )
  }

  // success
  return (
    <div style={s.statusLine}>
      <span style={{ ...s.dot, background: 'var(--success)' }} />
      <span style={{ ...s.statusText, color: 'var(--success)' }}>Idle · OK</span>
      <span style={s.policyDim}>letztes Backup vor {when}</span>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  card:       { background: 'linear-gradient(180deg, rgba(21,28,46,0.95) 0%, rgba(10,15,27,0.95) 100%)', border: '1px solid var(--border)', borderRadius: 16, padding: '16px 18px', display: 'flex', flexDirection: 'column', gap: 12, boxShadow: '0 4px 20px rgba(0,0,0,0.18)' },
  identity:   { display: 'flex', alignItems: 'center', gap: 12 },
  iconCircle: { width: 40, height: 40, borderRadius: 12, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 18, border: '1px solid transparent', background: 'rgba(255,255,255,0.03)', flexShrink: 0 },
  idText:     { minWidth: 0 },
  host:       { fontSize: 15, fontWeight: 700, color: 'var(--text)', lineHeight: 1.2 },
  meta:       { fontSize: 12, color: 'var(--text-dim)', marginTop: 2 },

  body:       { display: 'flex', flexDirection: 'column', gap: 10 },
  runTop:     { display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap' },
  statusLine: { display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' },
  dot:        { width: 8, height: 8, borderRadius: '50%', flexShrink: 0 },
  statusText: { fontSize: 13, fontWeight: 700 },
  policy:     { fontSize: 12, color: 'var(--text-muted)', marginLeft: 4 },
  policyDim:  { fontSize: 12, color: 'var(--text-dim)', marginLeft: 4 },

  progRow:    { display: 'flex', alignItems: 'center', gap: 12 },
  progBar:    { flex: 1, height: 8, background: 'rgba(255,255,255,0.06)', borderRadius: 4, overflow: 'hidden' },
  progFill:   { height: '100%', borderRadius: 4, transition: 'width 0.4s ease' },
  pct:        { fontSize: 13, fontWeight: 700, width: 42, textAlign: 'right', flexShrink: 0 },

  runMeta:    { display: 'flex', alignItems: 'center', gap: 8, fontSize: 12, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', flexWrap: 'wrap' },
  sep:        { color: 'var(--text-dim)' },

  stopBtn:    { padding: '7px 16px', borderRadius: 8, background: 'rgba(239,68,68,0.1)', color: 'var(--error)', border: '1px solid rgba(239,68,68,0.3)', fontSize: 13, fontWeight: 700, cursor: 'pointer', whiteSpace: 'nowrap' },
  errBox:     { marginTop: 8, padding: '10px 12px', background: 'rgba(239,68,68,0.07)', border: '1px solid rgba(239,68,68,0.18)', borderRadius: 10, fontSize: 13, color: 'var(--error)', lineHeight: 1.5, wordBreak: 'break-word' },
}
