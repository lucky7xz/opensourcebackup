import { useEffect, useState } from 'react'
import { api, type BackupPolicy } from '../api'
import { Table } from '../components/Table'

export function Policies() {
  const [policies, setPolicies] = useState<BackupPolicy[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => { api.policies().then(setPolicies).finally(() => setLoading(false)) }, [])

  return (
    <div style={styles.page}>
      <h1 style={styles.h1}>Policies <span style={styles.count}>{policies.length}</span></h1>
      <div style={styles.tableWrap}>
        {loading ? <div style={styles.loading}>Loading…</div> : (
          <Table
            columns={[
              { header: 'Name',       render: p => <span style={styles.name}>{p.Name}</span> },
              { header: 'Engine',     render: p => <span style={styles.badge}>{p.Engine}</span>, width: '100px' },
              { header: 'Schedule',   render: p => p.Schedule
                  ? <span style={styles.mono}>{p.Schedule}</span>
                  : <span style={styles.dim}>manual</span> },
              { header: 'Includes',   render: p => (
                  <div>{(p.Includes ?? []).map(i => <div key={i} style={styles.path}>{i}</div>)}</div>
                ) },
              { header: 'Repository', render: p => p.RepositoryID
                  ? <span style={styles.mono}>{p.RepositoryID.slice(0,8)}…</span>
                  : <span style={styles.warn}>⚠ not set</span> },
              { header: 'Created',    render: p => new Date(p.CreatedAt).toLocaleString() },
            ]}
            rows={policies}
            keyFn={p => p.ID}
            emptyMsg="No policies yet"
          />
        )}
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  page: { padding: '32px 40px', maxWidth: 1100 },
  h1: { fontSize: 24, fontWeight: 700, color: 'var(--text-primary)', marginBottom: 24, display: 'flex', alignItems: 'center', gap: 12 },
  count: { fontSize: 14, fontWeight: 600, background: 'rgba(0,212,255,0.12)', color: 'var(--accent-cyan)', padding: '2px 10px', borderRadius: 20 },
  tableWrap: { background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 10, overflow: 'hidden' },
  name: { fontWeight: 600, color: 'var(--text-primary)' },
  mono: { fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--accent-cyan)' },
  badge: { display: 'inline-block', padding: '2px 8px', borderRadius: 4, background: 'rgba(0,212,255,0.1)', color: 'var(--accent-cyan)', fontSize: 11, fontWeight: 600 },
  path: { fontSize: 12, color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)' },
  dim: { color: 'var(--text-secondary)', fontSize: 12 },
  warn: { color: 'var(--accent-orange)', fontSize: 12 },
  loading: { padding: 40, color: 'var(--text-secondary)', textAlign: 'center' },
}
