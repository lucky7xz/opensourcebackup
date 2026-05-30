import { useEffect, useState } from 'react'
import { api, type BackupRepository } from '../api'
import { Table } from '../components/Table'

export function Repositories() {
  const [repos, setRepos] = useState<BackupRepository[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => { api.repositories().then(setRepos).finally(() => setLoading(false)) }, [])

  return (
    <div style={styles.page}>
      <h1 style={styles.h1}>Repositories <span style={styles.count}>{repos.length}</span></h1>
      <div style={styles.tableWrap}>
        {loading ? <div style={styles.loading}>Loading…</div> : (
          <Table
            columns={[
              { header: 'Type',     render: r => <span style={styles.badge}>{r.Type}</span>, width: '90px' },
              { header: 'Location', render: r => <span style={styles.mono}>{r.Location}</span> },
              { header: 'Encrypt',  render: r => r.EncryptionMode ?? '—', width: '100px' },
              { header: 'WORM',     render: r => r.ObjectLockEnabled
                  ? <span style={styles.green}>✓ enabled</span>
                  : <span style={styles.dim}>—</span>, width: '90px' },
              { header: 'ID',       render: r => <span style={styles.mono}>{r.ID.slice(0,8)}…</span> },
              { header: 'Created',  render: r => new Date(r.CreatedAt).toLocaleString() },
            ]}
            rows={repos}
            keyFn={r => r.ID}
            emptyMsg="No repositories yet"
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
  mono: { fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--accent-cyan)' },
  badge: { display: 'inline-block', padding: '2px 8px', borderRadius: 4, background: 'rgba(0,212,255,0.1)', color: 'var(--accent-cyan)', fontSize: 11, fontWeight: 600 },
  green: { color: 'var(--accent-green)', fontSize: 12 },
  dim: { color: 'var(--text-secondary)', fontSize: 12 },
  loading: { padding: 40, color: 'var(--text-secondary)', textAlign: 'center' },
}
