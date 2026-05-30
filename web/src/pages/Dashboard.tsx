import { useEffect, useState } from 'react'
import { api, type BackupJob, type Snapshot, type System } from '../api'
import { StatCard } from '../components/Card'
import { StatusBadge } from '../components/StatusBadge'
import { Table } from '../components/Table'

function fmt(bytes?: number) {
  if (!bytes) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function timeAgo(iso?: string) {
  if (!iso) return '—'
  const diff = Date.now() - new Date(iso).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

export function Dashboard() {
  const [systems, setSystems] = useState<System[]>([])
  const [jobs, setJobs] = useState<BackupJob[]>([])
  const [snapshots, setSnapshots] = useState<Snapshot[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([api.systems(), api.jobs(), api.snapshots()])
      .then(([s, j, sn]) => { setSystems(s); setJobs(j); setSnapshots(sn) })
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div style={styles.loading}>Loading…</div>

  const success = jobs.filter(j => j.Status === 'success').length
  const failed  = jobs.filter(j => j.Status === 'failed').length
  const pending = jobs.filter(j => j.Status === 'pending' || j.Status === 'running').length

  const totalBytes = snapshots.reduce((a, s) => {
    const job = jobs.find(j => j.ID === s.JobID)
    return a + (job?.BytesUploaded ?? 0)
  }, 0)

  const recentJobs = [...jobs].sort((a, b) =>
    new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime()
  ).slice(0, 10)

  return (
    <div style={styles.page}>
      <h1 style={styles.h1}>Dashboard</h1>

      <div style={styles.grid4}>
        <StatCard label="Systems" value={systems.length} color="var(--accent-cyan)" />
        <StatCard label="Snapshots" value={snapshots.length} sub={fmt(totalBytes) + ' total'} color="var(--accent-green)" />
        <StatCard label="Jobs success" value={success} color="var(--accent-green)" />
        <StatCard label="Jobs failed" value={failed} color={failed > 0 ? 'var(--accent-red)' : 'var(--text-secondary)'} />
      </div>

      {pending > 0 && (
        <div style={styles.alert}>
          ⚙ {pending} job{pending > 1 ? 's' : ''} running or pending
        </div>
      )}

      <h2 style={styles.h2}>Recent Jobs</h2>
      <div style={styles.tableWrap}>
        <Table
          columns={[
            { header: 'Status',  render: j => <StatusBadge status={j.Status} />, width: '110px' },
            { header: 'System',  render: j => <span style={styles.mono}>{j.SystemID.slice(0, 8)}…</span> },
            { header: 'Policy',  render: j => <span style={styles.mono}>{j.PolicyID.slice(0, 8)}…</span> },
            { header: 'Size',    render: j => fmt(j.BytesUploaded) },
            { header: 'When',    render: j => timeAgo(j.CreatedAt) },
            { header: 'Error',   render: j => j.ErrorSummary
                ? <span style={{ color: 'var(--accent-red)', fontSize: 11 }}>{j.ErrorSummary}</span>
                : '—' },
          ]}
          rows={recentJobs}
          keyFn={j => j.ID}
          emptyMsg="No jobs yet"
        />
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  page: { padding: '32px 40px', maxWidth: 1100 },
  h1: { fontSize: 24, fontWeight: 700, color: 'var(--text-primary)', marginBottom: 24 },
  h2: { fontSize: 16, fontWeight: 600, color: 'var(--text-secondary)', margin: '32px 0 16px', textTransform: 'uppercase', letterSpacing: '0.06em' },
  grid4: { display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16, marginBottom: 16 },
  tableWrap: { background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 10, overflow: 'hidden' },
  mono: { fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--accent-cyan)' },
  loading: { padding: 40, color: 'var(--text-secondary)' },
  alert: { background: 'rgba(0,212,255,0.08)', border: '1px solid rgba(0,212,255,0.2)', borderRadius: 8, padding: '10px 16px', fontSize: 13, color: 'var(--accent-cyan)', marginBottom: 8 },
}
