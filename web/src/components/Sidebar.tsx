import { NavLink } from 'react-router-dom'

const links = [
  { to: '/',            icon: '⬡', label: 'Dashboard' },
  { to: '/systems',     icon: '🖥', label: 'Systems' },
  { to: '/policies',    icon: '📋', label: 'Policies' },
  { to: '/jobs',        icon: '⚙', label: 'Jobs' },
  { to: '/snapshots',   icon: '📦', label: 'Snapshots' },
  { to: '/repositories',icon: '🗄', label: 'Repositories' },
]

export function Sidebar() {
  return (
    <aside style={styles.sidebar}>
      <div style={styles.logo}>
        <span style={styles.logoAccent}>OSB</span>
        <span style={styles.logoText}>OpensourceBackup</span>
      </div>
      <nav>
        {links.map(l => (
          <NavLink
            key={l.to}
            to={l.to}
            end={l.to === '/'}
            style={({ isActive }) => ({
              ...styles.link,
              ...(isActive ? styles.linkActive : {}),
            })}
          >
            <span style={styles.icon}>{l.icon}</span>
            {l.label}
          </NavLink>
        ))}
      </nav>
      <div style={styles.footer}>
        <div style={styles.footerDot} />
        <span>Control Plane</span>
      </div>
    </aside>
  )
}

const styles: Record<string, React.CSSProperties> = {
  sidebar: {
    width: 220,
    minWidth: 220,
    background: 'var(--bg-sidebar)',
    borderRight: '1px solid var(--border)',
    display: 'flex',
    flexDirection: 'column',
    padding: '24px 0',
    height: '100vh',
    position: 'sticky',
    top: 0,
  },
  logo: {
    padding: '0 20px 20px',
    borderBottom: '1px solid var(--border)',
    marginBottom: 12,
  },
  logoAccent: {
    display: 'block',
    fontSize: 22,
    fontWeight: 800,
    color: 'var(--accent-cyan)',
    letterSpacing: 2,
  },
  logoText: {
    fontSize: 11,
    color: 'var(--text-secondary)',
    letterSpacing: 1,
    textTransform: 'uppercase' as const,
  },
  link: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    padding: '10px 20px',
    color: 'var(--text-secondary)',
    fontSize: 13,
    fontWeight: 500,
    borderLeft: '2px solid transparent',
    transition: 'all 0.15s',
  },
  linkActive: {
    color: 'var(--accent-cyan)',
    borderLeftColor: 'var(--accent-cyan)',
    background: 'rgba(0,212,255,0.06)',
  },
  icon: { fontSize: 16, width: 20, textAlign: 'center' as const },
  footer: {
    marginTop: 'auto',
    padding: '16px 20px',
    borderTop: '1px solid var(--border)',
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    fontSize: 12,
    color: 'var(--text-secondary)',
  },
  footerDot: {
    width: 8,
    height: 8,
    borderRadius: '50%',
    background: 'var(--accent-green)',
    boxShadow: '0 0 6px var(--accent-green)',
  },
}
