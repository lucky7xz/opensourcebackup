import type { HealthScore } from '../../api'

interface Props { score: HealthScore | null }

export function HealthScoreCard({ score }: Props) {
  if (!score) return (
    <div className="dash-card" style={s.card}>
      <div style={s.loading}>Loading score…</div>
    </div>
  )

  const { score: val, label, color, deductions, factors } = score
  const strokeColor = color.startsWith('var') ? undefined : color

  // SVG ring
  const R = 36, C = 2 * Math.PI * R
  const progress = (val / 100) * C

  return (
    <div className="dash-card" style={s.card}>
      <div style={s.top}>
        <div style={s.label}>Backup Health Score</div>
      </div>

      <div style={s.body}>
        {/* Ring gauge */}
        <div style={s.ringWrap}>
          <svg width={90} height={90} viewBox="0 0 90 90">
            <circle cx={45} cy={45} r={R} fill="none" stroke="var(--border)" strokeWidth={8} />
            <circle cx={45} cy={45} r={R} fill="none"
              stroke={strokeColor ?? 'var(--accent)'}
              strokeWidth={8}
              strokeDasharray={`${progress} ${C}`}
              strokeDashoffset={C / 4}
              strokeLinecap="round"
              style={{ transition: 'stroke-dasharray 0.6s ease' }}
            />
            <text x={45} y={42} textAnchor="middle" fontSize={18} fontWeight={800} fill="var(--text)">{val}</text>
            <text x={45} y={56} textAnchor="middle" fontSize={9} fill="var(--text-muted)">/ 100</text>
          </svg>
          <div style={{ ...s.scoreLabel, color: strokeColor ?? 'var(--accent)' }}>{label}</div>
          <div style={s.scoreVersion}>Score v{score.version}</div>
        </div>

        {/* Evidence lines */}
        <div style={s.evidence}>
          {deductions.slice(0, 3).map((d, i) => (
            <div key={i} style={s.evidLine}>
              <span style={{ color: 'var(--error)', fontWeight: 700, minWidth: 26 }}>−{d.points}</span>
              <span style={s.evidText}>{d.reason}</span>
            </div>
          ))}
          {factors.slice(0, 2).map((f, i) => (
            <div key={i} style={s.evidLine}>
              <span style={{ color: 'var(--success)', minWidth: 26 }}>✓</span>
              <span style={s.evidText}>{f}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  card:       { padding: '18px 20px', display: 'flex', flexDirection: 'column', gap: 10, minWidth: 0, height: '100%' },
  loading:    { color: 'var(--text-muted)', fontSize: 13 },
  top:        { display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  label:      { fontSize: 11, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.1em' },
  body:       { display: 'flex', gap: 16, alignItems: 'flex-start' },
  ringWrap:   { display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4, flexShrink: 0 },
  scoreLabel: { fontSize: 13, fontWeight: 700 },
  scoreVersion:{ fontSize: 10, color: 'var(--text-dim)' },
  evidence:   { flex: 1, display: 'flex', flexDirection: 'column', gap: 5 },
  evidLine:   { display: 'flex', gap: 8, alignItems: 'flex-start' },
  evidText:   { fontSize: 11, color: 'var(--text-muted)', lineHeight: 1.4 },
}
