import { NavLink } from 'react-router-dom'

const sections = [
  { links: [
    { to:'/',             label:'Dashboard',     icon:'▦' },
  ]},
  { label: 'INVENTORY', links: [
    { to:'/systems',      label:'Systems',       icon:'⬡' },
    { to:'/agents',       label:'Agents',        icon:'⬡' },
  ]},
  { label: 'OPERATIONS', links: [
    { to:'/policies',     label:'Policies',      icon:'▤' },
    { to:'/jobs',         label:'Jobs',          icon:'⊙' },
  ]},
  { label: 'EVIDENCE', links: [
    { to:'/snapshots',    label:'Snapshots',     icon:'◫' },
    { to:'/restore-tests',label:'Restore Tests', icon:'✓' },
  ]},
  { label: 'STORAGE', links: [
    { to:'/repositories', label:'Repositories',  icon:'▭' },
  ]},
  { label: 'SYSTEM', links: [
    { to:'/settings',     label:'Settings',      icon:'⚙' },
  ]},
]

export function Sidebar() {
  return (
    <aside style={s.sidebar}>
      <div style={s.brand}>
        <div style={s.brandName}>OpenSourceBackup</div>
        <div style={s.brandSub}>Backup Control Plane</div>
      </div>

      <nav style={s.nav}>
        {sections.map((sec, i) => (
          <div key={i}>
            {sec.label && <div style={s.section}>{sec.label}</div>}
            {sec.links.map(l => (
              <NavLink key={l.to} to={l.to} end={l.to==='/'} style={({isActive}) => ({
                ...s.link, ...(isActive ? s.active : {})
              })}>
                <span style={s.icon}>{l.icon}</span>
                {l.label}
              </NavLink>
            ))}
          </div>
        ))}
      </nav>

      <div style={s.footer}>
        <span style={s.dot} />
        <span style={{color:'var(--text-muted)', fontSize:12}}>Control Plane · online</span>
      </div>
    </aside>
  )
}

const s: Record<string, React.CSSProperties> = {
  sidebar: {
    width:220, minWidth:220, background:'var(--bg-sidebar)',
    borderRight:'1px solid var(--border)', display:'flex', flexDirection:'column',
    height:'100vh', position:'sticky', top:0, overflowY:'auto',
  },
  brand: { padding:'20px 16px 16px', borderBottom:'1px solid var(--border)' },
  brandName: { fontSize:14, fontWeight:700, color:'var(--text)', letterSpacing:0.5 },
  brandSub:  { fontSize:11, color:'var(--text-dim)', marginTop:2 },
  nav: { flex:1, padding:'8px 0' },
  section: {
    padding:'14px 16px 4px', fontSize:10, fontWeight:700,
    color:'var(--text-dim)', letterSpacing:'0.1em',
  },
  link: {
    display:'flex', alignItems:'center', gap:8, padding:'7px 16px',
    color:'var(--text-muted)', fontSize:13, fontWeight:500,
    borderLeft:'2px solid transparent', transition:'all 0.12s',
  },
  active: {
    color:'var(--text)', borderLeftColor:'var(--accent)',
    background:'var(--accent-dim)',
  },
  icon: { fontSize:13, width:16, textAlign:'center' as const, opacity:0.7 },
  footer: {
    padding:'12px 16px', borderTop:'1px solid var(--border)',
    display:'flex', alignItems:'center', gap:6,
  },
  dot: {
    width:7, height:7, borderRadius:'50%',
    background:'var(--success)', boxShadow:'0 0 6px rgba(34,197,94,0.5)',
  },
}
