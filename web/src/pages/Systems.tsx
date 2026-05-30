import { useEffect, useState } from 'react'
import { api, type System } from '../api'
import { StatusBadge } from '../components/StatusBadge'
import { Table } from '../components/Table'

function timeAgo(iso?: string) {
  if (!iso) return 'never'
  const diff = Date.now() - new Date(iso).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

export function Systems() {
  const [systems, setSystems] = useState<System[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => { api.systems().then(setSystems).finally(() => setLoading(false)) }, [])

  return (
    <div style={styles.page}>
      <h1 style={styles.h1}>Systems <span style={styles.count}>{systems.length}</span></h1>
      <div style={styles.tableWrap}>
        {loading ? <div style={styles.loading}>Loading…</div> : (
          <Table
            columns={[
              { header: 'Hostname', render: s => <span style={styles.hostname}>{s.Hostname}</span> },
              { header: 'Risk',     render: s => <StatusBadge status={s.RiskClass || 'standard'} />, width: '100px' },
              { header: 'OS',       render: s => s.OS ?? '—' },
              { header: 'Agent',    render: s => s.AgentVersion ?? '—' },
              { header: 'Last Seen',render: s => timeAgo(s.LastSeen) },
              { header: 'Tags',     render: s => s.Tags
                  ? Object.entries(s.Tags).map(([k,v]) => (
                      <span key={k} style={styles.tag}>{k}={v}</span>
                    ))
                  : '—' },
              { header: 'ID',       render: s => <span style={styles.mono}>{s.ID.slice(0,8)}…</span> },
            ]}
            rows={systems}
            keyFn={s => s.ID}
            emptyMsg="No systems registered yet"
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
  hostname: { fontWeight: 600, color: 'var(--text-primary)' },
  mono: { fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--accent-cyan)' },
  tag: { display: 'inline-block', background: 'rgba(148,163,184,0.1)', color: 'var(--text-secondary)', padding: '1px 7px', borderRadius: 4, fontSize: 11, marginRight: 4 },
  loading: { padding: 40, color: 'var(--text-secondary)', textAlign: 'center' },
}
