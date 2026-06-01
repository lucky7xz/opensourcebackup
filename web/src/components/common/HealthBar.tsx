interface Props {
  pct:   number | null  // 0–100, null = no data
  width?: number        // px — default 60
}

export function HealthBar({ pct, width = 60 }: Props) {
  if (pct === null) return <span style={{ color: 'var(--text-dim)', fontSize: 11 }}>—</span>
  const color = pct >= 90 ? 'var(--success)' : pct >= 70 ? 'var(--warning)' : 'var(--error)'
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
      <div style={{ width, height: 4, borderRadius: 2, background: 'rgba(255,255,255,0.08)', overflow: 'hidden', flexShrink: 0 }}>
        <div style={{ width: `${pct}%`, height: '100%', background: color, borderRadius: 2, transition: 'width 0.3s' }} />
      </div>
      <span style={{ fontSize: 11, fontWeight: 600, color, minWidth: 32 }}>{pct}%</span>
    </div>
  )
}
