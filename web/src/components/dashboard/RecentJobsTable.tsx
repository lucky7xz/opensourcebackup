import { useNavigate } from 'react-router-dom'
import type { BackupJob } from '../../api'
import { StatusBadge } from '../StatusBadge'
import { timeAgo, duration } from '../../api'

interface Props { jobs: BackupJob[]; systems: Record<string, string> }

export function RecentJobsTable({ jobs, systems }: Props) {
  const nav = useNavigate()
  return (
    <div className="dash-card" style={s.card}>
      <div style={s.header}>
        <span style={s.title}>Recent Jobs</span>
        <button style={s.link} onClick={() => nav('/jobs')}>View all →</button>
      </div>
      <table style={s.table}>
        <thead>
          <tr>{['Job','System','Type','Status','Duration','Completed'].map(h =>
            <th key={h} style={s.th}>{h}</th>)}</tr>
        </thead>
        <tbody>
          {jobs.length === 0
            ? <tr><td colSpan={6} style={s.empty}>No jobs yet</td></tr>
            : jobs.map(j => (
            <tr key={j.ID} style={s.tr}>
              <td style={s.td}><span style={s.mono}>{j.ID.slice(0,10)}…</span></td>
              <td style={s.td}><span style={s.sys}>{systems[j.SystemID] ?? j.SystemID.slice(0,8)}</span></td>
              <td style={s.td}><TypeBadge type={j.Type ?? 'backup'} /></td>
              <td style={s.td}><StatusBadge status={j.Status} /></td>
              <td style={s.td}><span style={s.mono}>{duration(j.StartedAt, j.FinishedAt)}</span></td>
              <td style={s.td}><span style={{ color: 'var(--text-muted)', fontSize: 12 }}>{timeAgo(j.FinishedAt ?? j.CreatedAt)}</span></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function TypeBadge({ type }: { type: string }) {
  const color = type === 'retention' ? 'var(--warning)' : type === 'restore_test' ? 'var(--accent-teal)' : 'var(--accent-blue)'
  return <span style={{ fontSize: 11, padding: '2px 7px', borderRadius: 4, background: `${color}18`, color, fontWeight: 600 }}>{type}</span>
}

const s: Record<string, React.CSSProperties> = {
  card:   { display: 'flex', flexDirection: 'column', overflow: 'hidden' },
  header: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '14px 18px 10px', borderBottom: '1px solid var(--border)' },
  title:  { fontSize: 13, fontWeight: 700 },
  link:   { background: 'none', border: 'none', color: 'var(--accent)', fontSize: 11, cursor: 'pointer', padding: 0 },
  table:  { width: '100%', borderCollapse: 'collapse' as const },
  th:     { padding: '7px 14px', fontSize: 10, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase' as const, letterSpacing: '0.08em', textAlign: 'left' as const, borderBottom: '1px solid var(--border)', background: 'rgba(255,255,255,0.02)' },
  tr:     { borderBottom: '1px solid rgba(255,255,255,0.04)' },
  td:     { padding: '8px 14px', fontSize: 12, color: 'var(--text-muted)', verticalAlign: 'middle' as const },
  mono:   { fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' },
  sys:    { fontSize: 12, color: 'var(--text)', fontWeight: 500 },
  empty:  { padding: 24, textAlign: 'center' as const, color: 'var(--text-dim)', fontSize: 12 },
}
