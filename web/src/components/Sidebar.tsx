import { NavLink, useNavigate } from 'react-router-dom'

// ── Navigation structure ───────────────────────────────────────────────────

const NAV = [
  {
    links: [
      { to: '/',              label: 'Overview',      icon: '▦' },
    ],
  },
  {
    label: 'INVENTORY',
    links: [
      { to: '/systems',       label: 'Systems',       icon: '🖥' },
      { to: '/agents',        label: 'Agents',        icon: '⬡' },
    ],
  },
  {
    label: 'OPERATIONS',
    links: [
      { to: '/repositories',  label: 'Repositories',  icon: '▭' },
      { to: '/policies',      label: 'Policies',      icon: '▤' },
      { to: '/jobs',          label: 'Jobs',          icon: '⊙' },
    ],
  },
  {
    label: 'ASSURANCE',
    links: [
      { to: '/snapshots',     label: 'Snapshots',     icon: '◫' },
      { to: '/restore-tests', label: 'Restore Tests', icon: '✓' },
    ],
  },
  {
    label: 'COMPLIANCE',
    links: [
      { to: '/evidence',      label: 'Evidence',      icon: '🔍' },
    ],
  },
  {
    label: 'COMPLIANCE',
    links: [
      { to: '/evidence',      label: 'Evidence',      icon: '🔍' },
      { to: '/alerts',        label: 'Alerts',        icon: '🔔' },
    ],
  },
  {
    label: 'COMING SOON',
    links: [
      { to: '/reports',       label: 'Reports',       icon: '📄', soon: true },
    ],
  },
  {
    label: 'SYSTEM',
    links: [
      { to: '/settings',      label: 'Settings',      icon: '⚙' },
    ],
  },
]

// ── Quick actions ──────────────────────────────────────────────────────────

interface QuickAction {
  label: string
  icon:  string
  to:    string
}

const QUICK_ACTIONS: QuickAction[] = [
  { label: 'New System',      icon: '＋', to: '/systems' },
  { label: 'Enroll Agent',    icon: '⬡', to: '/agents' },
  { label: 'Run Restore Test',icon: '✓', to: '/restore-tests' },
  { label: 'Create Policy',   icon: '▤', to: '/policies' },
  { label: 'Add Repository',  icon: '▭', to: '/repositories' },
]

// ── Component ─────────────────────────────────────────────────────────────

export function Sidebar() {
  const navigate = useNavigate()

  return (
    <aside style={s.sidebar}>

      {/* Brand */}
      <div style={s.brand}>
        <div style={s.brandIcon}>OSB</div>
        <div>
          <div style={s.brandName}>OpenSourceBackup</div>
          <div style={s.brandSub}>Restore Assured</div>
        </div>
      </div>

      {/* Navigation */}
      <nav style={s.nav}>
        {NAV.map((sec, i) => (
          <div key={i}>
            {sec.label && <div style={s.section}>{sec.label}</div>}
            {sec.links.map(l => (
              (l as {soon?: boolean}).soon ? (
                <div key={l.to} style={s.soonLink} title="Coming soon">
                  <span style={s.icon}>{l.icon}</span>
                  <span>{l.label}</span>
                  <span style={s.soonBadge}>Soon</span>
                </div>
              ) : (
                <NavLink
                  key={l.to} to={l.to} end={l.to === '/'}
                  style={({ isActive }) => ({ ...s.link, ...(isActive ? s.active : {}) })}
                >
                  <span style={s.icon}>{l.icon}</span>
                  {l.label}
                </NavLink>
              )
            ))}
          </div>
        ))}
      </nav>

      {/* Quick Actions */}
      <div style={s.quickSection}>
        <div style={s.section}>QUICK ACTIONS</div>
        {QUICK_ACTIONS.map(a => (
          <button key={a.to} style={s.quickBtn} onClick={() => navigate(a.to)}>
            <span style={s.icon}>{a.icon}</span>
            {a.label}
          </button>
        ))}
      </div>

      {/* Footer */}
      <div style={s.footer}>
        <span style={s.dot} />
        <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>Control Plane · online</span>
      </div>

    </aside>
  )
}

// ── Styles ─────────────────────────────────────────────────────────────────

const s: Record<string, React.CSSProperties> = {
  sidebar: {
    width: 220, minWidth: 220, background: 'var(--bg-sidebar)',
    borderRight: '1px solid var(--border)', display: 'flex',
    flexDirection: 'column', height: '100vh',
    position: 'sticky', top: 0, overflowY: 'auto',
  },
  brand: {
    padding: '16px 14px', borderBottom: '1px solid var(--border)',
    display: 'flex', alignItems: 'center', gap: 10,
  },
  brandIcon: {
    width: 32, height: 32, borderRadius: 8,
    background: 'var(--accent)', color: '#fff',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    fontSize: 10, fontWeight: 800, letterSpacing: '0.05em', flexShrink: 0,
  },
  brandName: { fontSize: 12, fontWeight: 700, color: 'var(--text)', lineHeight: 1.3 },
  brandSub:  { fontSize: 10, color: 'var(--text-dim)', marginTop: 1 },

  nav:     { flex: 1, padding: '6px 0' },
  section: {
    padding: '12px 14px 4px', fontSize: 9, fontWeight: 700,
    color: 'var(--text-dim)', letterSpacing: '0.12em', textTransform: 'uppercase' as const,
  },
  link: {
    display: 'flex', alignItems: 'center', gap: 8,
    padding: '6px 14px', color: 'var(--text-muted)',
    fontSize: 13, fontWeight: 500, borderLeft: '2px solid transparent',
    transition: 'all 0.1s', textDecoration: 'none',
  },
  active: {
    color: 'var(--text)', borderLeftColor: 'var(--accent)',
    background: 'var(--accent-dim)',
  },
  soonLink: {
    display: 'flex', alignItems: 'center', gap: 8,
    padding: '6px 14px', color: 'var(--text-dim)',
    fontSize: 13, opacity: 0.5, cursor: 'default',
    borderLeft: '2px solid transparent',
  },
  soonBadge: {
    marginLeft: 'auto', fontSize: 9, padding: '1px 5px',
    borderRadius: 3, background: 'rgba(99,102,241,0.15)',
    color: '#818cf8', fontWeight: 700, letterSpacing: '0.05em',
  },
  icon: { fontSize: 12, width: 14, textAlign: 'center' as const, flexShrink: 0 },

  quickSection: {
    borderTop: '1px solid var(--border)', paddingBottom: 8,
  },
  quickBtn: {
    display: 'flex', alignItems: 'center', gap: 8,
    width: '100%', padding: '6px 14px', textAlign: 'left' as const,
    background: 'none', border: 'none', cursor: 'pointer',
    color: 'var(--text-muted)', fontSize: 12, fontWeight: 500,
    transition: 'color 0.1s',
  },

  footer: {
    padding: '10px 14px', borderTop: '1px solid var(--border)',
    display: 'flex', alignItems: 'center', gap: 6,
  },
  dot: {
    width: 6, height: 6, borderRadius: '50%',
    background: 'var(--success)', boxShadow: '0 0 5px rgba(34,197,94,0.5)',
  },
}
