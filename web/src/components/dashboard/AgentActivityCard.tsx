import { useNavigate } from 'react-router-dom'
import type { System } from '../../api'
import { timeAgo } from '../../api'

interface Props { systems: System[] }

const MS_ONLINE = 2  * 60 * 1000
const MS_IDLE   = 15 * 60 * 1000

function status(s: System): 'online'|'idle'|'offline' {
  if (!s.LastSeen) return 'offline'
  const age = Date.now() - new Date(s.LastSeen).getTime()
  return age <= MS_ONLINE ? 'online' : age <= MS_IDLE ? 'idle' : 'offline'
}

const DOT_COLOR = { online: 'var(--success)', idle: 'var(--warning)', offline: 'var(--error)' }
const STATUS_LABEL = { online: 'Online', idle: 'Idle', offline: 'Offline' }

export function AgentActivityCard({ systems }: Props) {
  const nav = useNavigate()
  const online  = systems.filter(s => status(s) === 'online').length
  const idle    = systems.filter(s => status(s) === 'idle').length
  const offline = systems.filter(s => status(s) === 'offline').length
  const total   = systems.length

  const sorted = [...systems].sort((a, b) => {
    if (!a.LastSeen && !b.LastSeen) return 0
    if (!a.LastSeen) return 1
    if (!b.LastSeen) return -1
    return new Date(b.LastSeen).getTime() - new Date(a.LastSeen).getTime()
  })

  // Mini ring SVG
  const R = 30, C = 2 * Math.PI * R
  const onlinePct  = total > 0 ? (online  / total) * C : 0
  const idlePct    = total > 0 ? (idle    / total) * C : 0
  const offlinePct = total > 0 ? (offline / total) * C : 0

  const segments = [
    { len: onlinePct,  color: 'var(--success)', gap: total > 1 ? 2 : 0 },
    { len: idlePct,    color: 'var(--warning)', gap: total > 1 ? 2 : 0 },
    { len: offlinePct, color: 'var(--error)',   gap: 0 },
  ]
  let offset = C / 4
  const arcs = segments.map(seg => {
    const el = { ...seg, offset }
    offset -= seg.len + seg.gap
    return el
  })

  return (
    <div className="dash-card" style={s.card}>
      <div style={s.header}>
        <span style={s.title}>Agent Activity</span>
        <button style={s.link} onClick={() => nav('/agents')}>View all →</button>
      </div>
      <div style={s.body}>
        {/* Ring */}
        <div style={s.ringWrap}>
          <svg width={76} height={76} viewBox="0 0 76 76">
            <circle cx={38} cy={38} r={R} fill="none" stroke="var(--border)" strokeWidth={7} />
            {arcs.map((a, i) => a.len > 0 && (
              <circle key={i} cx={38} cy={38} r={R} fill="none"
                stroke={a.color} strokeWidth={7}
                strokeDasharray={`${Math.max(0,a.len-a.gap)} ${C}`}
                strokeDashoffset={a.offset}
                strokeLinecap="butt"
              />
            ))}
            <text x={38} y={34} textAnchor="middle" fontSize={14} fontWeight={800} fill="var(--text)">{total}</text>
            <text x={38} y={46} textAnchor="middle" fontSize={8} fill="var(--text-muted)">Agents</text>
          </svg>
          <div style={s.legend}>
            {[['online',online,'var(--success)'],['idle',idle,'var(--warning)'],['offline',offline,'var(--error)']].map(([k,v,c]) => (
              <div key={k as string} style={s.legendRow}>
                <span style={{ width:8, height:8, borderRadius:'50%', background: c as string, flexShrink:0, display:'block' }} />
                <span style={{ fontSize:11, color:'var(--text-muted)', flex:1 }}>{STATUS_LABEL[k as 'online'|'idle'|'offline']}</span>
                <span style={{ fontSize:11, fontWeight:700, color:'var(--text)' }}>{v as number}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Last seen list */}
        <div style={s.list}>
          <div style={s.listHeader}>Last Seen</div>
          {sorted.slice(0, 6).map(sys => {
            const st = status(sys)
            return (
              <div key={sys.ID} style={s.listRow}>
                <span style={{ width:7, height:7, borderRadius:'50%', background:DOT_COLOR[st], flexShrink:0, display:'block', boxShadow: st==='online' ? `0 0 4px ${DOT_COLOR[st]}` : 'none' }} />
                <span style={{ flex:1, fontSize:12, color:'var(--text)', fontWeight:500, overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap' }}>{sys.Hostname}</span>
                <span style={{ fontSize:11, color:'var(--text-muted)', flexShrink:0 }}>{sys.LastSeen ? timeAgo(sys.LastSeen) : 'never'}</span>
              </div>
            )
          })}
          {systems.length === 0 && <div style={{ fontSize:12, color:'var(--text-dim)', padding:'8px 0' }}>No agents enrolled yet.</div>}
        </div>
      </div>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  card:      { display: 'flex', flexDirection: 'column' },
  header:    { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '14px 18px 10px', borderBottom: '1px solid var(--border)' },
  title:     { fontSize: 13, fontWeight: 700 },
  link:      { background: 'none', border: 'none', color: 'var(--accent)', fontSize: 11, cursor: 'pointer', padding: 0 },
  body:      { display: 'flex', gap: 16, padding: '14px 18px', flex: 1 },
  ringWrap:  { display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8, flexShrink: 0 },
  legend:    { display: 'flex', flexDirection: 'column', gap: 5, minWidth: 80 },
  legendRow: { display: 'flex', alignItems: 'center', gap: 7 },
  list:      { flex: 1, display: 'flex', flexDirection: 'column', gap: 0 },
  listHeader:{ fontSize: 10, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: 8 },
  listRow:   { display: 'flex', alignItems: 'center', gap: 8, padding: '5px 0', borderBottom: '1px solid rgba(255,255,255,0.04)' },
}
