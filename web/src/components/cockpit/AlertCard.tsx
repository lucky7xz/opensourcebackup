// Compact alert card shown in the sidebar when one or more systems have a
// failed last backup. Calm tone (no panic banners) per the design rules.
interface AlertCardProps {
  errorCount: number
  onDetails:  () => void
}

export function AlertCard({ errorCount, onDetails }: AlertCardProps) {
  if (errorCount <= 0) return null
  const label = errorCount === 1 ? '1 System mit Fehlern' : `${errorCount} Systeme mit Fehlern`

  return (
    <div style={s.card}>
      <div style={s.head}>
        <span style={s.icon}>⚠</span>
        <span style={s.title}>{label}</span>
      </div>
      <div style={s.text}>Bitte überprüfen und beheben.</div>
      <button style={s.btn} onClick={onDetails}>Details</button>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  card:  { background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.22)', borderRadius: 14, padding: 16, display: 'flex', flexDirection: 'column', gap: 8 },
  head:  { display: 'flex', alignItems: 'center', gap: 8 },
  icon:  { fontSize: 15, color: 'var(--error)' },
  title: { fontSize: 14, fontWeight: 700, color: 'var(--error)' },
  text:  { fontSize: 12, color: 'var(--text-muted)' },
  btn:   { alignSelf: 'flex-start', marginTop: 2, padding: '6px 16px', borderRadius: 8, background: 'rgba(239,68,68,0.1)', border: '1px solid rgba(239,68,68,0.3)', color: 'var(--error)', fontSize: 12, fontWeight: 700, cursor: 'pointer' },
}
