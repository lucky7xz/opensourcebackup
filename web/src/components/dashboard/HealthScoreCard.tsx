import type { HealthScore } from '../../api'

interface Props { score: HealthScore | null }

const LABEL_COLOR: Record<string, string> = {
  Excellent: '#22c55e',
  Good:      '#4ade80',
  Fair:      '#f59e0b',
  'At Risk': '#ef4444',
}

export function HealthScoreCard({ score }: Props) {
  if (!score) return (
    <div style={s.card}>
      <div style={s.loadingText}>Loading score…</div>
    </div>
  )

  const { score: val, label, deductions, factors } = score
  const color = LABEL_COLOR[label] ?? '#94a3b8'

  // SVG ring
  const R = 38, SW = 8
  const C = 2 * Math.PI * R
  const progress = Math.max(0, Math.min(val / 100, 1)) * C

  return (
    <div style={s.card}>
      <div style={s.topRow}>
        <span style={s.cardLabel}>Backup Health Score</span>
        <span style={{ fontSize: 9, color: 'var(--text-dim)' }}>v{score.version}</span>
      </div>

      <div style={s.body}>
        {/* Ring gauge */}
        <div style={s.ringBox}>
          <svg width={92} height={92} viewBox="0 0 92 92">
            {/* Track */}
            <circle cx={46} cy={46} r={R} fill="none"
              stroke="rgba(255,255,255,0.06)" strokeWidth={SW} />
            {/* Progress */}
            <circle cx={46} cy={46} r={R} fill="none"
              stroke={color} strokeWidth={SW}
              strokeDasharray={`${progress.toFixed(1)} ${C.toFixed(1)}`}
              strokeDashoffset={(C * 0.25).toFixed(1)}
              strokeLinecap="round"
              style={{ transition: 'stroke-dasharray 0.8s ease, stroke 0.4s' }}
            />
            {/* Score number */}
            <text x={46} y={41} textAnchor="middle"
              fontSize={20} fontWeight={800} fill="var(--text)">{val}</text>
            <text x={46} y={54} textAnchor="middle"
              fontSize={8} fill="var(--text-dim)">/ 100</text>
          </svg>
          {/* Label below ring */}
          <div style={{ fontSize: 12, fontWeight: 700, color, marginTop: -2 }}>{label}</div>
        </div>

        {/* Evidence lines */}
        <div style={s.evidence}>
          {deductions.slice(0, 3).map((d, i) => (
            <div key={i} style={s.deductLine}>
              <span style={{ color: 'var(--error)', fontWeight: 700, fontSize: 11, minWidth: 22 }}>
                −{d.points}
              </span>
              <span style={s.deductText}>{d.reason}</span>
            </div>
          ))}
          {factors.slice(0, 2).map((f, i) => (
            <div key={i} style={s.factorLine}>
              <span style={{ color: 'var(--success)', fontSize: 11, minWidth: 22 }}>✓</span>
              <span style={s.deductText}>{f}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  card: {
    background: 'linear-gradient(160deg, var(--bg-card-soft) 0%, var(--bg-card) 100%)',
    border: '1px solid var(--border)',
    borderRadius: 14,
    padding: '16px 18px',
    boxShadow: '0 4px 20px rgba(0,0,0,0.18)',
    display: 'flex', flexDirection: 'column', gap: 8,
    height: '100%',
  },
  loadingText: { color: 'var(--text-muted)', fontSize: 12 },
  topRow:   { display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  cardLabel:{ fontSize: 10, fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.12em' },
  body:     { display: 'flex', gap: 12, alignItems: 'flex-start', flex: 1 },
  ringBox:  { display: 'flex', flexDirection: 'column', alignItems: 'center', flexShrink: 0, gap: 2 },
  evidence: { flex: 1, display: 'flex', flexDirection: 'column', gap: 5, minWidth: 0 },
  deductLine:{ display: 'flex', gap: 6, alignItems: 'flex-start' },
  factorLine:{ display: 'flex', gap: 6, alignItems: 'flex-start' },
  deductText:{ fontSize: 10, color: 'var(--text-muted)', lineHeight: 1.5, overflow: 'hidden', textOverflow: 'ellipsis' },
}
