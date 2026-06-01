export interface ActivityBucket {
  hour:          string
  backups:       number
  restore_tests: number
  failures:      number
  bytes_added?:  number
}

interface Props {
  data:   ActivityBucket[]
  height?: number
}

const COLORS = {
  backups:      '#38bdf8',
  restoreTests: '#22c55e',
  failures:     '#ef4444',
}

/**
 * Smooth area/line chart for backup activity.
 * Uses SVG polylines with fill areas — no external library.
 */
export function ActivityChart({ data, height = 160 }: Props) {
  if (!data || data.length < 2) {
    return (
      <div style={{ height, display:'flex', alignItems:'center', justifyContent:'center' }}>
        <span style={{ fontSize:12, color:'var(--text-dim)', fontStyle:'italic' }}>
          Activity history will appear as jobs and restore tests are collected.
        </span>
      </div>
    )
  }

  const W = 1000  // viewBox width
  const padL = 8, padR = 8, padT = 12, padB = 4
  const chartW = W - padL - padR
  const chartH = height - padT - padB

  const maxVal = Math.max(1,
    ...data.map(d => d.backups),
    ...data.map(d => d.restore_tests),
    ...data.map(d => d.failures),
  )

  const n = data.length
  const xOf = (i: number) => padL + (i / (n - 1)) * chartW
  const yOf = (v: number) => padT + chartH - (v / maxVal) * chartH

  const area = (key: keyof ActivityBucket, color: string) => {
    const pts = data.map((d, i) => `${xOf(i).toFixed(1)},${yOf(Number(d[key] ?? 0)).toFixed(1)}`)
    const bot = `${xOf(n-1).toFixed(1)},${(padT+chartH).toFixed(1)} ${padL},${(padT+chartH).toFixed(1)}`
    return (
      <g key={key as string}>
        <polygon
          points={pts.join(' ') + ' ' + bot}
          fill={color} fillOpacity={0.08}
        />
        <polyline
          points={pts.join(' ')}
          fill="none" stroke={color} strokeWidth={1.5}
          strokeLinecap="round" strokeLinejoin="round"
        />
      </g>
    )
  }

  // Label every Nth point
  const labelEvery = n <= 12 ? 2 : n <= 24 ? 4 : n <= 48 ? 8 : 12

  return (
    <div style={{ width:'100%', display:'flex', flexDirection:'column', gap:0 }}>
      <svg
        viewBox={`0 0 ${W} ${height}`}
        preserveAspectRatio="none"
        style={{ width:'100%', height, display:'block' }}
      >
        {/* Grid lines */}
        {[0.25, 0.5, 0.75, 1].map(f => (
          <line key={f}
            x1={padL} y1={(padT + chartH * (1-f)).toFixed(1)}
            x2={W-padR} y2={(padT + chartH * (1-f)).toFixed(1)}
            stroke="rgba(255,255,255,0.05)" strokeWidth={0.5} />
        ))}

        {/* Areas + lines */}
        {area('failures',     COLORS.failures)}
        {area('restore_tests',COLORS.restoreTests)}
        {area('backups',      COLORS.backups)}

        {/* Data dots on hover — invisible hit targets */}
        {data.map((d, i) => (
          <circle key={i}
            cx={xOf(i)} cy={yOf(d.backups)}
            r={3} fill={COLORS.backups} opacity={0}
          />
        ))}

        {/* Baseline */}
        <line
          x1={padL} y1={padT+chartH}
          x2={W-padR} y2={padT+chartH}
          stroke="rgba(255,255,255,0.08)" strokeWidth={0.8} />
      </svg>

      {/* X-axis labels */}
      <div style={{
        display: 'flex', justifyContent: 'space-between',
        padding: '2px 8px 0', overflow: 'hidden',
      }}>
        {data
          .map((d, i) => ({ d, i }))
          .filter(({ i }) => i % labelEvery === 0)
          .map(({ d, i }) => (
            <span key={i} style={{ fontSize:10, color:'rgba(255,255,255,0.45)', lineHeight:1 }}>
              {d.hour}
            </span>
          ))
        }
      </div>
    </div>
  )
}

export function ActivityLegend() {
  return (
    <div style={{ display:'flex', gap:14, alignItems:'center' }}>
      {[
        { color: COLORS.backups,      label: 'Backups' },
        { color: COLORS.restoreTests, label: 'Restore Tests' },
        { color: COLORS.failures,     label: 'Failures' },
      ].map(({ color, label }) => (
        <div key={label} style={{ display:'flex', alignItems:'center', gap:5 }}>
          <span style={{ width:10, height:2, borderRadius:1, background:color, display:'block' }} />
          <span style={{ fontSize:11, color:'var(--text-muted)' }}>{label}</span>
        </div>
      ))}
    </div>
  )
}
