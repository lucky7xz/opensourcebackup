import { useEffect, useState } from 'react'

const BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

interface AuditEntry {
  ID:           number
  Timestamp:    string
  Action:       string
  ResourceType: string
  ResourceID:   string
  ActorType:    string
  Actor:        string
  IP:           string
  Details:      string
  Severity:     'info' | 'warning' | 'critical'
  Success:      boolean
}

const SEVERITY_COLOR: Record<string, string> = {
  info:     'var(--text-muted)',
  warning:  'var(--warning)',
  critical: 'var(--error)',
}

const SEVERITY_ICON: Record<string, string> = {
  info:     'ℹ',
  warning:  '⚠',
  critical: '🔴',
}

const ACTOR_ICON: Record<string, string> = {
  admin:     '👤',
  agent:     '⬡',
  scheduler: '⏰',
  system:    '⚙',
}

export function Evidence() {
  const [entries,   setEntries]   = useState<AuditEntry[]>([])
  const [loading,   setLoading]   = useState(true)
  const [err,       setErr]       = useState<string|null>(null)
  const [filter,    setFilter]    = useState('')
  const [sevFilter, setSevFilter] = useState<string>('all')

  useEffect(() => {
    fetch(`${BASE}/v1/audit?limit=200`)
      .then(r => r.ok ? r.json() : Promise.reject(r.status))
      .then(data => setEntries(data ?? []))
      .catch(() => setErr('Could not load audit log.'))
      .finally(() => setLoading(false))
  }, [])

  const visible = entries.filter(e => {
    const matchSev = sevFilter === 'all' || e.Severity === sevFilter
    const q = filter.toLowerCase()
    const matchText = !q || e.Action.toLowerCase().includes(q) ||
      e.ResourceType.toLowerCase().includes(q) ||
      e.Actor?.toLowerCase().includes(q) ||
      e.Details?.toLowerCase().includes(q)
    return matchSev && matchText
  })

  return (
    <div style={s.page}>
      <div style={s.header}>
        <div>
          <h1 style={s.h1}>Evidence & Audit Log</h1>
          <p style={s.sub}>
            Append-only record of all security-relevant and data-lifecycle actions.
            Proving recoverability requires knowing what happened, when, and by whom.
          </p>
        </div>
      </div>

      {/* Filters */}
      <div style={s.filters}>
        <input
          style={s.search}
          placeholder="Filter by action, resource, actor, details…"
          value={filter}
          onChange={e => setFilter(e.target.value)}
        />
        {['all', 'info', 'warning', 'critical'].map(sev => (
          <button
            key={sev}
            onClick={() => setSevFilter(sev)}
            style={{
              ...s.sevBtn,
              ...(sevFilter === sev ? s.sevBtnActive : {}),
              color: sev === 'all' ? 'var(--text)' : SEVERITY_COLOR[sev],
            }}
          >
            {sev === 'all' ? 'All' : `${SEVERITY_ICON[sev]} ${sev}`}
          </button>
        ))}
        <span style={s.count}>{visible.length} entries</span>
      </div>

      {/* Table */}
      <div style={s.tableWrap}>
        {loading && <div style={s.empty}>Loading…</div>}
        {err    && <div style={{ ...s.empty, color: 'var(--error)' }}>{err}</div>}
        {!loading && !err && visible.length === 0 && (
          <div style={s.empty}>
            {entries.length === 0
              ? 'No audit events yet — events are written as you use the system.'
              : 'No events match the current filter.'}
          </div>
        )}
        {!loading && !err && visible.length > 0 && (
          <table style={s.table}>
            <thead>
              <tr>
                {['Time', 'Severity', 'Actor', 'Action', 'Resource', 'Details', 'Result'].map(h => (
                  <th key={h} style={s.th}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {visible.map(e => (
                <tr key={e.ID} style={s.tr}>
                  <td style={s.td}>
                    <span style={s.time}>{new Date(e.Timestamp).toLocaleString()}</span>
                  </td>
                  <td style={s.td}>
                    <span style={{ color: SEVERITY_COLOR[e.Severity] ?? 'var(--text-dim)', fontSize: 13 }}>
                      {SEVERITY_ICON[e.Severity] ?? 'ℹ'} {e.Severity}
                    </span>
                  </td>
                  <td style={s.td}>
                    <span style={s.actor}>
                      {ACTOR_ICON[e.ActorType] ?? '?'} {e.Actor || e.ActorType}
                    </span>
                  </td>
                  <td style={s.td}>
                    <span style={s.action}>{e.Action}</span>
                  </td>
                  <td style={s.td}>
                    <span style={s.resource}>
                      {e.ResourceType}
                      {e.ResourceID && (
                        <span style={s.resourceId}> {e.ResourceID.slice(0, 8)}…</span>
                      )}
                    </span>
                  </td>
                  <td style={s.td}>
                    <span style={s.details} title={e.Details}>{e.Details || '—'}</span>
                  </td>
                  <td style={s.td}>
                    {e.Success
                      ? <span style={{ color: 'var(--success)', fontSize: 12 }}>✓</span>
                      : <span style={{ color: 'var(--error)', fontSize: 12 }}>✗</span>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div style={s.note}>
        🔒 This log is append-only — rows are never modified or deleted by the application.
        PostgreSQL Row Security Policy prevents UPDATE/DELETE for the app user.
      </div>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  page:       { padding: '28px 36px', maxWidth: 1400 },
  header:     { marginBottom: 20 },
  h1:         { fontSize: 22, fontWeight: 700, color: 'var(--text)', marginBottom: 4 },
  sub:        { fontSize: 13, color: 'var(--text-muted)', maxWidth: 700 },
  filters:    { display: 'flex', gap: 8, alignItems: 'center', marginBottom: 16, flexWrap: 'wrap' as const },
  search:     { padding: '7px 12px', background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 6, color: 'var(--text)', fontSize: 13, outline: 'none', flex: '1 1 300px', maxWidth: 400 },
  sevBtn:     { padding: '5px 12px', borderRadius: 6, background: 'var(--bg-card)', border: '1px solid var(--border)', fontSize: 12, fontWeight: 600, cursor: 'pointer' },
  sevBtnActive:{ background: 'var(--accent-dim)', borderColor: 'var(--accent)' },
  count:      { fontSize: 12, color: 'var(--text-dim)', marginLeft: 'auto' },
  tableWrap:  { background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 10, overflow: 'auto' },
  table:      { width: '100%', borderCollapse: 'collapse' as const },
  th:         { padding: '8px 14px', fontSize: 11, fontWeight: 700, color: 'var(--text-dim)', textAlign: 'left' as const, textTransform: 'uppercase' as const, letterSpacing: '0.06em', borderBottom: '1px solid var(--border)', background: 'rgba(255,255,255,0.02)', whiteSpace: 'nowrap' as const },
  tr:         { borderBottom: '1px solid rgba(255,255,255,0.04)' },
  td:         { padding: '8px 14px', fontSize: 12, color: 'var(--text-muted)', verticalAlign: 'top' as const, maxWidth: 200 },
  empty:      { padding: 40, textAlign: 'center' as const, color: 'var(--text-dim)', fontSize: 13 },
  time:       { fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-dim)', whiteSpace: 'nowrap' as const },
  actor:      { fontSize: 12, color: 'var(--text)' },
  action:     { fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--accent)', whiteSpace: 'nowrap' as const },
  resource:   { fontSize: 12, color: 'var(--text-muted)' },
  resourceId: { fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-dim)' },
  details:    { fontSize: 11, color: 'var(--text-dim)', maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' as const, display: 'block' },
  note:       { marginTop: 12, fontSize: 11, color: 'var(--text-dim)', fontStyle: 'italic' },
}
