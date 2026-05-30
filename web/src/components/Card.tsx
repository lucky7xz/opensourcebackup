import type { ReactNode, CSSProperties } from 'react'

export function Card({ children, style }: { children: ReactNode; style?: CSSProperties }) {
  return <div style={{ background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:'var(--radius)', ...style }}>{children}</div>
}

export function HealthCard({
  label, value, sub, color, warn
}: { label:string; value:string|number; sub?:string; color?:string; warn?:boolean }) {
  return (
    <div style={{
      background:'var(--bg-card)', border:`1px solid ${warn ? 'rgba(245,158,11,0.3)' : 'var(--border)'}`,
      borderRadius:'var(--radius)', padding:'18px 20px',
    }}>
      <div style={{ fontSize:28, fontWeight:700, color: color ?? 'var(--text)', lineHeight:1 }}>{value}</div>
      <div style={{ fontSize:12, fontWeight:600, color:'var(--text-muted)', marginTop:6, textTransform:'uppercase' as const, letterSpacing:'0.06em' }}>{label}</div>
      {sub && <div style={{ fontSize:11, color:'var(--text-dim)', marginTop:3 }}>{sub}</div>}
    </div>
  )
}

export function SectionHeader({ title, count }: { title:string; count?:number }) {
  return (
    <div style={{ display:'flex', alignItems:'center', gap:10, marginBottom:12 }}>
      <h2 style={{ fontSize:13, fontWeight:700, color:'var(--text-muted)', textTransform:'uppercase' as const, letterSpacing:'0.08em' }}>{title}</h2>
      {count !== undefined && (
        <span style={{ fontSize:11, fontWeight:600, background:'rgba(59,130,246,0.12)', color:'var(--accent)', padding:'1px 8px', borderRadius:10 }}>{count}</span>
      )}
    </div>
  )
}
