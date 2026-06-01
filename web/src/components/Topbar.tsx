import { useNavigate } from 'react-router-dom'

interface Props {
  title:       string
  sub?:        string
  alertCount?: number
}

export function Topbar({ title, sub, alertCount = 0 }: Props) {
  const nav = useNavigate()
  return (
    <div style={s.bar}>
      <div style={s.left}>
        <h1 style={s.title}>{title}</h1>
        {sub && <p style={s.sub}>{sub}</p>}
      </div>
      <div style={s.right}>
        {/* Search */}
        <div style={s.searchWrap}>
          <span style={s.searchIcon}>🔍</span>
          <input style={s.search} placeholder="Search systems, jobs, repositories…" disabled />
          <span style={s.searchKey}>⌘K</span>
        </div>

        {/* Environment badge */}
        <div style={s.envBadge}>
          🖥 Local
        </div>

        {/* Notifications */}
        <div style={s.iconBtn} onClick={() => nav('/alerts')} title="Alerts">
          🔔
          {alertCount > 0 && (
            <span style={s.badge}>{alertCount > 9 ? '9+' : alertCount}</span>
          )}
        </div>

        {/* User */}
        <div style={s.userBtn}>
          <div style={s.avatar}>A</div>
          <span style={s.userName}>Admin</span>
        </div>
      </div>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  bar: {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '14px 28px 10px',
    borderBottom: '1px solid var(--border)',
    background: 'var(--bg-topbar)',
    gap: 16,
    flexShrink: 0,
  },
  left:     { minWidth: 0 },
  title:    { fontSize: 20, fontWeight: 700, color: 'var(--text)', lineHeight: 1.2 },
  sub:      { fontSize: 12, color: 'var(--text-muted)', marginTop: 2 },
  right:    { display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0 },
  searchWrap:{ display: 'flex', alignItems: 'center', gap: 8, background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 8, padding: '6px 12px', minWidth: 220 },
  searchIcon:{ fontSize: 12, color: 'var(--text-dim)' },
  search:   { background: 'none', border: 'none', color: 'var(--text-muted)', fontSize: 12, outline: 'none', flex: 1, cursor: 'default' },
  searchKey:{ fontSize: 10, color: 'var(--text-dim)', background: 'var(--border-soft)', borderRadius: 4, padding: '1px 5px' },
  envBadge: { fontSize: 11, fontWeight: 600, color: 'var(--accent)', background: 'var(--accent-dim)', border: '1px solid rgba(137,189,40,0.25)', borderRadius: 6, padding: '4px 10px' },
  iconBtn:  { position: 'relative' as const, fontSize: 16, cursor: 'pointer', padding: '4px 8px', borderRadius: 6, color: 'var(--text-muted)', userSelect: 'none' as const },
  badge:    { position: 'absolute' as const, top: 0, right: 0, background: 'var(--error)', color: '#fff', fontSize: 9, fontWeight: 800, borderRadius: '50%', width: 14, height: 14, display: 'flex', alignItems: 'center', justifyContent: 'center', lineHeight: 1 },
  userBtn:  { display: 'flex', alignItems: 'center', gap: 8, background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 8, padding: '5px 12px', cursor: 'pointer' },
  avatar:   { width: 24, height: 24, borderRadius: '50%', background: 'var(--accent)', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12, fontWeight: 700 },
  userName: { fontSize: 12, fontWeight: 600, color: 'var(--text)' },
}
