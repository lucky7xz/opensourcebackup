// Slim, modern "Letzte Jobs" table — secondary information, not the focal point.
import { duration, fmt, timeAgo, type BackupJob } from '../../api'
import { statusColor } from './util'

interface RecentJobsTableProps {
  jobs:       BackupJob[]                  // pre-sorted (newest first), already sliced
  systemName: (id: string) => string
  policyName: (id?: string) => string
  onDetails:  (job: BackupJob) => void
}

const STATUS_LABEL: Record<string, string> = {
  success:   'Erfolgreich',
  failed:    'Fehlgeschlagen',
  running:   'Läuft',
  pending:   'Wartet',
  cancelled: 'Abgebrochen',
}

export function RecentJobsTable({ jobs, systemName, policyName, onDetails }: RecentJobsTableProps) {
  if (jobs.length === 0) {
    return (
      <div style={s.empty}>
        <div style={s.emptyIcon}>↻</div>
        <div style={s.emptyTitle}>Noch keine Jobs</div>
        <div style={s.emptySub}>Starte ein Backup, um hier die Historie zu sehen.</div>
      </div>
    )
  }

  return (
    <div style={s.wrap}>
      <table style={s.table}>
        <thead>
          <tr>
            <th style={s.th}>Zeit</th>
            <th style={s.th}>System</th>
            <th style={s.th}>Policy</th>
            <th style={s.th}>Status</th>
            <th style={s.th}>Dauer</th>
            <th style={s.th}>Daten</th>
            <th style={{ ...s.th, textAlign: 'right' }}></th>
          </tr>
        </thead>
        <tbody>
          {jobs.map(job => {
            const color = statusColor(job.Status)
            return (
              <tr key={job.ID} style={s.tr}>
                <td style={s.td}>vor {timeAgo(job.FinishedAt ?? job.CreatedAt)}</td>
                <td style={{ ...s.td, color: 'var(--text)', fontWeight: 600 }}>{systemName(job.SystemID)}</td>
                <td style={s.td}>{policyName(job.PolicyID)}</td>
                <td style={s.td}>
                  <span style={{ ...s.statusDot, background: color }} />
                  <span style={{ color, fontWeight: 600 }}>{STATUS_LABEL[job.Status] ?? job.Status}</span>
                </td>
                <td style={s.tdMono}>{duration(job.StartedAt, job.FinishedAt)}</td>
                <td style={s.tdMono}>{fmt(job.BytesUploaded)}</td>
                <td style={{ ...s.td, textAlign: 'right' }}>
                  <button style={s.detailBtn} onClick={() => onDetails(job)}>Details</button>
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
  th:        { padding: '10px 14px', fontSize: 10, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.1em', borderBottom: '1px solid var(--border)', textAlign: 'left', whiteSpace: 'nowrap' },
  tr:        { borderBottom: '1px solid rgba(255,255,255,0.04)' },
  td:        { padding: '11px 14px', fontSize: 13, color: 'var(--text-muted)', whiteSpace: 'nowrap', verticalAlign: 'middle' },
  tdMono:    { padding: '11px 14px', fontSize: 12, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', whiteSpace: 'nowrap', verticalAlign: 'middle' },
  statusDot: { display: 'inline-block', width: 7, height: 7, borderRadius: '50%', marginRight: 7, verticalAlign: 'middle' },
  detailBtn: { padding: '5px 14px', borderRadius: 8, background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 12, fontWeight: 600, cursor: 'pointer' },
  empty:     { padding: '40px 20px', textAlign: 'center' },
  emptyIcon: { fontSize: 30, opacity: 0.3, marginBottom: 10 },
  emptyTitle:{ fontSize: 14, fontWeight: 700, color: 'var(--text-muted)', marginBottom: 4 },
  emptySub:  { fontSize: 12, color: 'var(--text-dim)' },
}
