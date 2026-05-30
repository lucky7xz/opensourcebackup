import { useEffect, useState } from 'react'
import { api, type Snapshot } from '../api'
import { StatusBadge } from '../components/StatusBadge'
import { Table } from '../components/Table'

export function Snapshots() {
  const [snapshots, setSnapshots] = useState<Snapshot[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => { api.snapshots().then(setSnapshots).finally(() => setLoading(false)) }, [])

  const sorted = [...snapshots].sort((a,b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())

  return (
    <div style={styles.page}>
      <h1 style={styles.h1}>Snapshots <span style={styles.count}>{snapshots.length}</span></h1>
      <div style={styles.tableWrap}>
        {loading ? <div style={styles.loading}>Loading…</div> : (
          <Table
            columns={[
              { header: 'Checksum',   render: s => <StatusBadge status={s.ChecksumStatus} />, width: '120px' },
              { header: 'Snapshot ID',render: s => <span style={styles.mono}>{s.EngineSnapshotID.slice(0,12)}…</span> },
              { header: 'Paths',      render: s => (
                  <div>{(s.Paths ?? []).map(p => <div key={p} style={styles.path}>{p}</div>)}</div>
                ) },
              { header: 'Repository', render: s => <span style={styles.mono}>{s.RepositoryID.slice(0,8)}…</span> },
              { header: 'Job',        render: s => <span style={styles.mono}>{s.JobID.slice(0,8)}…</span> },
              { header: 'Created',    render: s => new Date(s.CreatedAt).toLocaleString() },
            ]}
            rows={sorted}
            keyFn={s => s.ID}
            emptyMsg="No snapshots yet — run a backup first"
          />
        )}
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  page: { padding: '32px 40px', maxWidth: 1100 },
  h1: { fontSize: 24, fontWeight: 700, color: 'var(--text-primary)', marginBottom: 24, display: 'flex', alignItems: 'center', gap: 12 },
  count: { fontSize: 14, fontWeight: 600, background: 'rgba(0,255,136,0.12)', color: 'var(--accent-green)', padding: '2px 10px', borderRadius: 20 },
  tableWrap: { background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 10, overflow: 'hidden' },
  mono: { fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--accent-cyan)' },
  path: { fontSize: 12, color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)' },
  loading: { padding: 40, color: 'var(--text-secondary)', textAlign: 'center' },
}
