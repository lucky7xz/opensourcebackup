import { Sparkline } from '../Sparkline'

interface Props {
  icon:      React.ReactNode
  label:     string
  value:     string
  sub?:      string
  color?:    string
  warn?:     boolean
  sparkData?: number[]
  sparkColor?: string
  trend?:    string   // e.g. "▲ 3 since yesterday"
  trendUp?:  boolean  // green if true, red if false
}

export function KpiCard({ icon, label, value, sub, color = 'var(--text)', warn, sparkData, sparkColor, trend, trendUp }: Props) {
  return (
    <div className="dash-card" style={s.card}>
      <div style={s.top}>
        <div style={s.iconWrap}>{icon}</div>
        <span style={s.label}>{label}</span>
      </div>
      <div style={{ ...s.value, color: warn ? 'var(--error)' : color }}>{value}</div>
      {sub && <div style={s.sub}>{sub}</div>}
      {sparkData && sparkData.length > 1 && (
        <div style={s.spark}>
          <Sparkline data={sparkData} width={100} height={24} color={sparkColor ?? color} filled />
        </div>
      )}
      {trend && (
        <div style={{ ...s.trend, color: trendUp ? 'var(--success)' : trendUp === false ? 'var(--error)' : 'var(--text-muted)' }}>
          {trend}
        </div>
      )}
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  card:    { padding: '18px 20px', display: 'flex', flexDirection: 'column', gap: 4, minWidth: 0 },
  top:     { display: 'flex', alignItems: 'center', gap: 10, marginBottom: 6 },
  iconWrap:{ fontSize: 18, lineHeight: 1, flexShrink: 0 },
  label:   { fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.1em' },
  value:   { fontSize: 32, fontWeight: 800, lineHeight: 1.1 },
  sub:     { fontSize: 12, color: 'var(--text-muted)', marginTop: 2 },
  spark:   { marginTop: 8 },
  trend:   { fontSize: 11, marginTop: 4, fontWeight: 500 },
}
