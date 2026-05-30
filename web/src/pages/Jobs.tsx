import { useEffect, useState } from 'react'
import { api, type BackupJob } from '../api'
import { StatusBadge } from '../components/StatusBadge'
import { Table } from '../components/Table'

function fmt(bytes?: number) {
  if (!bytes) return '—'
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function duration(start?: string, end?: string) {
  if (!start || !end) return '—'
  const ms = new Date(end).getTime() - new Date(start).getTime()
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`
}

export function Jobs() {
  const [jobs, setJobs] = useState<BackupJob[]>([])
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState<string>('all')

  useEffect(() => { api.jobs().then(setJobs).finally(() => setLoading(false)) }, [])

  const sorted = [...jobs].sort((a,b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())
  const filtered = filter === 'all' ? sorted : sorted.filter(j => j.Status === filter)

  return (
    <div style={styles.page}>
      <h1 style={styles.h1}>Jobs <span style={styles.count}>{jobs.length}</span></h1>

      <div style={styles.filters}>
        {['all','success','running','pending','failed'].map(f => (
          <button key={f} onClick={() => setFilter(f)} style={{ ...styles.btn, ...(filter === f ? styles.btnActive : {}) }}>
            {f}
          </button>
        ))}
      </div>

      <div style={styles.tableWrap}>
        {loading ? <div style={styles.loading}>Loading…</div> : (
          <Table
            columns={[
              { header: 'Status',   render: j => <StatusBadge status={j.Status} />, width: '110px' },
              { header: 'System',   render: j => <span style={styles.mono}>{j.SystemID.slice(0,8)}…</span> },
              { header: 'Policy',   render: j => <span style={styles.mono}>{j.PolicyID.slice(0,8)}…</span> },
              { header: 'Size',     render: j => fmt(j.BytesUploaded) },
              { header: 'Duration', render: j => duration(j.StartedAt, j.FinishedAt) },
              { header: 'Created',  render: j => new Date(j.CreatedAt).toLocaleString() },
              { header: 'Error',    render: j => j.ErrorSummary
                  ? <span style={styles.err}>{j.ErrorSummary}</span>
                  : '—' },
            ]}
            rows={filtered}
            keyFn={j => j.ID}
            emptyMsg="No jobs found"
          />
        )}
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  page: { padding: '32px 40px', maxWidth: 1100 },
  h1: { fontSize: 24, fontWeight: 700, color: 'var(--text-primary)', marginBottom: 20, display: 'flex', alignItems: 'center', gap: 12 },
  count: { fontSize: 14, fontWeight: 600, background: 'rgba(0,212,255,0.12)', color: 'var(--accent-cyan)', padding: '2px 10px', borderRadius: 20 },
  filters: { display: 'flex', gap: 8, marginBottom: 16 },
  btn: { padding: '6px 14px', borderRadius: 6, border: '1px solid var(--border)', background: 'transparent', color: 'var(--text-secondary)', fontSize: 12, cursor: 'pointer', fontWeight: 500 },
  btnActive: { background: 'rgba(0,212,255,0.1)', color: 'var(--accent-cyan)', borderColor: 'rgba(0,212,255,0.3)' },
  tableWrap: { background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 10, overflow: 'hidden' },
  mono: { fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--accent-cyan)' },
  err: { color: 'var(--accent-red)', fontSize: 11 },
  loading: { padding: 40, color: 'var(--text-secondary)', textAlign: 'center' },
}
