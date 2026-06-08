// A single KPI card: icon circle + large value + label + subtext + optional link.
type Tone = 'systems' | 'running' | 'error' | 'verify'

interface KpiCardProps {
  icon:          string
  label:         string
  value:         string | number
  sub:           string
  tone:          Tone
  emphasise?:    boolean            // visually highlight (e.g. errors present)
  onDetails?:    () => void
  detailsLabel?: string
}

const TONES: Record<Tone, { color: string; bg: string; border: string }> = {
  systems: { color: 'var(--success)', bg: 'rgba(34,197,94,0.12)',  border: 'rgba(34,197,94,0.25)' },
  running: { color: 'var(--running)', bg: 'rgba(56,189,248,0.12)', border: 'rgba(56,189,248,0.25)' },
  error:   { color: 'var(--error)',   bg: 'rgba(239,68,68,0.12)',  border: 'rgba(239,68,68,0.25)' },
  verify:  { color: '#a78bfa',        bg: 'rgba(139,92,246,0.12)', border: 'rgba(139,92,246,0.25)' },
}

export function KpiCard({ icon, label, value, sub, tone, emphasise, onDetails, detailsLabel }: KpiCardProps) {
  const t = TONES[tone]
  return (
    <div style={{ ...s.card, ...(emphasise ? { borderColor: t.border } : {}) }}>
      <div style={s.row}>
        <div style={{ ...s.icon, background: t.bg, color: t.color, borderColor: t.border }}>{icon}</div>
        <div style={s.text}>
          <div style={s.label}>{label}</div>
          <div style={{ ...s.value, ...(emphasise ? { color: t.color } : {}) }}>{value}</div>
        </div>
      </div>
      <div style={s.sub}>{sub}</div>
      {onDetails && (
        <button style={s.link} onClick={onDetails}>{detailsLabel ?? 'Details anzeigen'}</button>
      )}
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  card:   { background: 'linear-gradient(160deg, var(--bg-card-soft) 0%, var(--bg-card) 100%)', border: '1px solid var(--border)', borderRadius: 16, padding: '18px 20px', boxShadow: '0 4px 20px rgba(0,0,0,0.18)', display: 'flex', flexDirection: 'column', gap: 10, minWidth: 0 },
  row:    { display: 'flex', alignItems: 'center', gap: 14 },
  icon:   { width: 46, height: 46, borderRadius: 14, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 20, border: '1px solid transparent', flexShrink: 0 },
  text:   { minWidth: 0 },
  label:  { fontSize: 12, color: 'var(--text-muted)', fontWeight: 600, marginBottom: 2 },
  value:  { fontSize: 30, fontWeight: 800, color: 'var(--text)', letterSpacing: '-0.5px', lineHeight: 1 },
  sub:    { fontSize: 12, color: 'var(--text-dim)' },
  link:   { alignSelf: 'flex-start', background: 'none', border: 'none', padding: 0, color: 'var(--running)', fontSize: 12, fontWeight: 600, cursor: 'pointer' },
}
