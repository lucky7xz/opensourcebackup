import { Sparkline } from '../Sparkline'

interface Props {
  icon:       React.ReactNode
  label:      string
  value:      string
  sub?:       string
  color?:     string
  warn?:      boolean
  sparkData?: number[]
  sparkColor?:string
  trend?:     string
  trendUp?:   boolean
}

export function KpiCard({ icon, label, value, sub, color = 'var(--text)', warn, sparkData, sparkColor, trend, trendUp }: Props) {
  const borderColor = warn ? 'rgba(239,68,68,0.3)' : 'var(--border)'

  return (
    <div style={{ ...s.card, borderColor }}>
      {/* Header */}
      <div style={s.header}>
        <span style={s.iconWrap}>{icon}</span>
        <span style={s.label}>{label}</span>
      </div>

      {/* Value */}
      <div style={{ ...s.value, color: warn ? 'var(--error)' : color }}>
        {value}
      </div>

      {/* Sub text */}
      {sub && <div style={s.sub}>{sub}</div>}

      {/* Sparkline */}
      {sparkData && sparkData.length > 1 && (
        <div style={s.sparkWrap}>
          <Sparkline data={sparkData} width={100} height={26} color={sparkColor ?? color} filled />
        </div>
      )}

      {/* Trend */}
      {trend && (
        <div style={{ ...s.trend, color: trendUp === true ? 'var(--success)' : trendUp === false ? 'var(--error)' : 'var(--text-muted)' }}>
          {trendUp === true ? '▲' : trendUp === false ? '▼' : ''} {trend}
        </div>
      )}
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  card:     {
    background: 'linear-gradient(160deg, var(--bg-card-soft) 0%, var(--bg-card) 100%)',
    border: '1px solid',
    borderRadius: 14,
    padding: '16px 18px',
    display: 'flex', flexDirection: 'column', gap: 3,
    boxShadow: '0 4px 20px rgba(0,0,0,0.18)',
    minWidth: 0,
  },
  header:   { display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 },
  iconWrap: { fontSize: 17, lineHeight: 1, flexShrink: 0, opacity: 0.85 },
  label:    { fontSize: 10, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.12em' },
  value:    { fontSize: 30, fontWeight: 800, lineHeight: 1.1, letterSpacing: '-0.5px' },
  sub:      { fontSize: 11, color: 'var(--text-muted)', marginTop: 1 },
  sparkWrap:{ marginTop: 6 },
  trend:    { fontSize: 10, marginTop: 3, fontWeight: 600, letterSpacing: '0.03em' },
}
