import { useNavigate } from 'react-router-dom'

interface Alert { code: string; severity: 'critical'|'warning'|'info'; title: string; category: string; points: number }
interface Evidence { ID: number; Action: string; ResourceType: string; Timestamp: string }
interface Props { alerts: Alert[]; evidence: Evidence[] }

import { timeAgo } from '../../api'

const SEV_ICON  = { critical: '🔴', warning: '⚠️', info: 'ℹ️' }
const SEV_COLOR = { critical: 'var(--error)', warning: 'var(--warning)', info: 'var(--accent-blue)' }

export function AlertsPreview({ alerts, evidence }: Props) {
  const nav = useNavigate()
  return (
    <div className="dash-card" style={s.card}>
      <div style={s.header}>
        <span style={s.title}>Recent Alerts</span>
        <button style={s.link} onClick={() => nav('/alerts')}>View all →</button>
      </div>

      <div style={s.section}>
        {alerts.length === 0 ? (
          <div style={s.allClear}>
            <span style={{ fontSize: 18 }}>✅</span>
            <span style={{ fontSize: 12, color: 'var(--text-dim)' }}>No active alerts</span>
          </div>
        ) : alerts.slice(0, 5).map(a => (
          <div key={a.code} style={s.row}>
            <span style={{ fontSize: 13, flexShrink: 0 }}>{SEV_ICON[a.severity]}</span>
            <div style={s.rowBody}>
              <div style={{ fontSize: 12, color: 'var(--text)', fontWeight: 600 }}>{a.title}</div>
              <div style={{ fontSize: 10, color: 'var(--text-dim)', marginTop: 1 }}>{a.category} · −{a.points} pts</div>
            </div>
            <span style={{ ...s.sevDot, color: SEV_COLOR[a.severity] }}>{a.severity}</span>
          </div>
        ))}
      </div>

      {evidence.length > 0 && (
        <>
          <div style={s.divider} />
          <div style={s.evidHeader}>
            <span style={s.evidTitle}>Recent Evidence</span>
            <button style={s.link} onClick={() => nav('/evidence')}>View all →</button>
          </div>
          <div style={{ padding: '0 18px 14px' }}>
            {evidence.slice(0, 4).map(e => (
              <div key={e.ID} style={s.evidRow}>
                <span style={{ fontSize: 11, color: 'var(--success)' }}>✓</span>
                <span style={s.evidAction}>{e.Action}</span>
                <span style={{ fontSize: 10, color: 'var(--text-dim)', flexShrink: 0 }}>{timeAgo(e.Timestamp)}</span>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  card:      { display: 'flex', flexDirection: 'column' },
  header:    { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '14px 18px 10px', borderBottom: '1px solid var(--border)' },
  title:     { fontSize: 13, fontWeight: 700 },
  link:      { background: 'none', border: 'none', color: 'var(--accent)', fontSize: 11, cursor: 'pointer', padding: 0 },
  section:   { padding: '10px 18px', display: 'flex', flexDirection: 'column', gap: 8 },
  allClear:  { display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6, padding: '16px 0', color: 'var(--text-dim)' },
  row:       { display: 'flex', alignItems: 'flex-start', gap: 10, padding: '6px 0', borderBottom: '1px solid rgba(255,255,255,0.04)' },
  rowBody:   { flex: 1, minWidth: 0 },
  sevDot:    { fontSize: 9, fontWeight: 700, textTransform: 'uppercase' as const, letterSpacing: '0.05em', flexShrink: 0, marginTop: 2 },
  divider:   { height: 1, background: 'var(--border)', margin: '4px 0' },
  evidHeader:{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 18px 6px' },
  evidTitle: { fontSize: 11, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase' as const, letterSpacing: '0.08em' },
  evidRow:   { display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0', borderBottom: '1px solid rgba(255,255,255,0.04)' },
  evidAction:{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--accent)', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' as const },
}
