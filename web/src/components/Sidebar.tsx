import { NavLink, useNavigate } from 'react-router-dom'

const NAV = [
  {
    label: 'OPERATIONS',
    links: [
      { to: '/cockpit',       label: 'Cockpit',       icon: '⊙' },
      { to: '/',              label: 'Overview',      icon: '⊞' },
      { to: '/systems',       label: 'Systems',       icon: '🖥' },
      { to: '/agents',        label: 'Agents',        icon: '◉' },
      { to: '/policies',      label: 'Policies',      icon: '≡' },
      { to: '/jobs',          label: 'Jobs',          icon: '↻' },
    ],
  },
  {
    label: 'RECOVERY',
    links: [
      { to: '/snapshots',     label: 'Snapshots',     icon: '⬡' },
      { to: '/restore-tests', label: 'Restore Tests', icon: '✓' },
      { to: '/repositories',  label: 'Repositories',  icon: '▤' },
    ],
  },
  {
    label: 'COMPLIANCE',
    links: [
      { to: '/evidence',      label: 'Evidence',      icon: '🔍' },
      { to: '/alerts',        label: 'Alerts',        icon: '🔔' },
      { to: '/reports',       label: 'Reports',       icon: '📊', soon: true },
    ],
  },
  {
    label: 'ADMIN',
    links: [
      { to: '/users',         label: 'Users',         icon: '👤' },
      { to: '/settings',      label: 'Settings',      icon: '⚙' },
    ],
  },
]

const QUICK = [
  { label: '+ New System',       to: '/systems' },
  { label: 'Enroll Agent',       to: '/agents' },
  { label: 'Run Restore Test',   to: '/restore-tests' },
  { label: 'Create Policy',      to: '/policies' },
  { label: 'Add Repository',     to: '/repositories' },
]

export function Sidebar() {
  const nav = useNavigate()
  return (
    <aside style={s.sidebar}>

      {/* Brand */}
      <div style={s.brand}>
        <div style={s.brandIcon}>
          <span style={{ fontSize: 13, fontWeight: 800, color: '#fff' }}>OSB</span>
        </div>
        <div>
          <div style={s.brandName}>OpenSourceBackup</div>
          <div style={s.brandSub}>Restore Assured</div>
        </div>
      </div>

      {/* Navigation */}
      <nav style={s.nav}>
        {NAV.map((sec, i) => (
          <div key={i} style={s.section}>
            <div style={s.sectionLabel}>{sec.label}</div>
            {sec.links.map(l => (
              (l as {soon?:boolean}).soon ? (
                <div key={l.to} style={s.soonLink} title="Coming soon">
                  <span style={s.linkIcon}>{l.icon}</span>
                  <span>{l.label}</span>
                  <span style={s.soonBadge}>Soon</span>
                </div>
              ) : (
                <NavLink key={l.to} to={l.to} end={l.to === '/'}
                  style={({ isActive }) => ({
                    ...s.link,
                    ...(isActive ? s.linkActive : {}),
                  })}>
                  {({ isActive }) => (
                    <>
                      <span style={{ ...s.linkIcon, color: isActive ? 'var(--accent)' : 'var(--text-dim)' }}>{l.icon}</span>
                      <span>{l.label}</span>
                    </>
                  )}
                </NavLink>
              )
            ))}
          </div>
        ))}
      </nav>

      {/* Quick Actions */}
      <div style={s.quickSection}>
        <div style={s.sectionLabel}>QUICK ACTIONS</div>
        {QUICK.map(a => (
          <button key={a.to} style={s.quickBtn} onClick={() => nav(a.to)}>
            {a.label}
          </button>
        ))}
      </div>

      {/* Instance info */}
      <div style={s.instance}>
        <div style={s.instanceDot} />
        <div>
          <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted)' }}>Local Infrastructure</div>
          <div style={{ fontSize: 10, color: 'var(--text-dim)' }}>Self-hosted · Control Plane online</div>
        </div>
      </div>

    </aside>
  )
}

const s: Record<string, React.CSSProperties> = {
  sidebar: {
    width: 224, minWidth: 224,
    background: 'var(--bg-sidebar)',
    borderRight: '1px solid var(--border)',
    display: 'flex', flexDirection: 'column',
    height: '100vh', position: 'sticky', top: 0,
    overflowY: 'auto',
  },
  brand: {
    display: 'flex', alignItems: 'center', gap: 10,
    padding: '18px 16px 16px',
    borderBottom: '1px solid var(--border)',
    flexShrink: 0,
  },
  brandIcon: {
    width: 34, height: 34, borderRadius: 9,
    background: 'linear-gradient(135deg, var(--accent) 0%, #5a8f1a 100%)',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    flexShrink: 0,
    boxShadow: '0 0 12px rgba(137,189,40,0.3)',
  },
  brandName: { fontSize: 12, fontWeight: 700, color: 'var(--text)', lineHeight: 1.3 },
  brandSub:  { fontSize: 9, color: 'var(--text-dim)', marginTop: 1 },

  nav:         { flex: 1, padding: '8px 0', overflowY: 'auto' },
  section:     { marginBottom: 4 },
  sectionLabel:{ padding: '10px 16px 4px', fontSize: 9, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase' as const, letterSpacing: '0.12em' },

  link: {
    display: 'flex', alignItems: 'center', gap: 9,
    padding: '7px 16px', margin: '1px 8px',
    color: 'var(--text-muted)', fontSize: 13, fontWeight: 500,
    borderRadius: 7, textDecoration: 'none',
    transition: 'all 0.12s',
    cursor: 'pointer',
  },
  linkActive: {
    color: 'var(--text)',
    background: 'rgba(137,189,40,0.1)',
    boxShadow: 'inset 2px 0 0 var(--accent)',
  },
  linkIcon: { fontSize: 13, width: 16, textAlign: 'center' as const, flexShrink: 0 },

  soonLink: {
    display: 'flex', alignItems: 'center', gap: 9,
    padding: '7px 16px', margin: '1px 8px',
    color: 'var(--text-dim)', fontSize: 13, opacity: 0.5,
    cursor: 'default', borderRadius: 7,
  },
  soonBadge: {
    marginLeft: 'auto', fontSize: 9, padding: '1px 5px',
    borderRadius: 3, background: 'rgba(99,102,241,0.15)',
    color: '#818cf8', fontWeight: 700, letterSpacing: '0.05em',
  },

  quickSection: { borderTop: '1px solid var(--border)', padding: '10px 8px 8px', flexShrink: 0 },
  quickBtn: {
    display: 'block', width: '100%', textAlign: 'left' as const,
    padding: '6px 10px', borderRadius: 6,
    background: 'none', border: '1px solid var(--border)',
    color: 'var(--text-muted)', fontSize: 11, fontWeight: 500,
    cursor: 'pointer', marginBottom: 4,
    transition: 'all 0.1s',
  },

  instance: {
    display: 'flex', alignItems: 'center', gap: 8,
    padding: '12px 16px',
    borderTop: '1px solid var(--border)',
    flexShrink: 0,
  },
  instanceDot: {
    width: 7, height: 7, borderRadius: '50%',
    background: 'var(--success)',
    boxShadow: '0 0 6px var(--success)',
    flexShrink: 0,
  },
}
