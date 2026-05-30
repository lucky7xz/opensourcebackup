const map: Record<string, {bg:string; color:string; dot:string}> = {
  success:    { bg:'rgba(34,197,94,0.10)',  color:'#22c55e', dot:'#22c55e' },
  running:    { bg:'rgba(96,165,250,0.10)', color:'#60a5fa', dot:'#60a5fa' },
  pending:    { bg:'rgba(156,163,175,0.10)',color:'#9ca3af', dot:'#9ca3af' },
  failed:     { bg:'rgba(244,63,94,0.10)',  color:'#f43f5e', dot:'#f43f5e' },
  warning:    { bg:'rgba(245,158,11,0.10)', color:'#f59e0b', dot:'#f59e0b' },
  overdue:    { bg:'rgba(245,158,11,0.10)', color:'#f59e0b', dot:'#f59e0b' },
  verified:   { bg:'rgba(34,197,94,0.10)',  color:'#22c55e', dot:'#22c55e' },
  unverified: { bg:'rgba(156,163,175,0.10)',color:'#9ca3af', dot:'none'    },
  online:     { bg:'rgba(34,197,94,0.10)',  color:'#22c55e', dot:'#22c55e' },
  offline:    { bg:'rgba(244,63,94,0.10)',  color:'#f43f5e', dot:'#f43f5e' },
  healthy:    { bg:'rgba(34,197,94,0.10)',  color:'#22c55e', dot:'#22c55e' },
  critical:   { bg:'rgba(244,63,94,0.10)',  color:'#f43f5e', dot:'#f43f5e' },
  standard:   { bg:'rgba(96,165,250,0.10)', color:'#60a5fa', dot:'none'    },
  passed:     { bg:'rgba(34,197,94,0.10)',  color:'#22c55e', dot:'#22c55e' },
  'not tested':{ bg:'rgba(156,163,175,0.10)',color:'#9ca3af', dot:'none'  },
}

export function StatusBadge({ status }: { status: string }) {
  const key = status.toLowerCase()
  const s = map[key] ?? { bg:'rgba(75,85,99,0.2)', color:'#9ca3af', dot:'none' }
  return (
    <span style={{
      display:'inline-flex', alignItems:'center', gap:5,
      padding:'3px 9px', borderRadius:20, fontSize:11, fontWeight:600,
      background:s.bg, color:s.color, textTransform:'uppercase', letterSpacing:'0.05em',
    }}>
      {s.dot !== 'none' && (
        <span style={{ width:5, height:5, borderRadius:'50%', background:s.dot, flexShrink:0 }} />
      )}
      {status}
    </span>
  )
}
