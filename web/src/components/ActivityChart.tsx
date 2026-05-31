import { useEffect, useRef } from 'react'

export interface ActivityBucket {
  hour:          string
  backups:       number
  restore_tests: number
  failures:      number
}

interface Props {
  data:   ActivityBucket[]
  height?: number
}

const COLORS = {
  backups:      '#00d4ff',
  restoreTests: '#00ff88',
  failures:     '#ef4444',
}

/**
 * Lightweight SVG bar chart for backup/restore/failure activity.
 * No external chart library — keeps bundle size minimal.
 */
export function ActivityChart({ data, height = 140 }: Props) {
  const canvasRef = useRef<HTMLDivElement>(null)

  if (!data || data.length === 0) {
    return (
      <div style={{ height, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <span style={{ fontSize: 12, color: 'var(--text-dim)' }}>No activity data yet</span>
      </div>
    )
  }

  const w      = 100  // SVG viewBox width
  const padB   = 18   // bottom padding for labels
  const padT   = 6    // top padding
  const chartH = height - padB - padT
  const n      = data.length
  const barW   = (w / n) * 0.7
  const gap    = (w / n) * 0.3

  const maxVal = Math.max(1, ...data.map(d => d.backups + d.restore_tests + d.failures))

  // Show every Nth label to avoid crowding
  const labelEvery = n <= 12 ? 1 : n <= 24 ? 4 : 6

  const toY = (v: number) => padT + chartH - (v / maxVal) * chartH

  return (
    <div ref={canvasRef} style={{ width: '100%', height }}>
      <svg
        viewBox={`0 0 100 ${height}`}
        preserveAspectRatio="none"
        style={{ width: '100%', height: '100%' }}
      >
        {/* Grid lines */}
        {[0, 0.5, 1].map((f, i) => (
          <line
            key={i}
            x1={0} y1={padT + chartH * (1 - f)}
            x2={100} y2={padT + chartH * (1 - f)}
            stroke="rgba(255,255,255,0.06)" strokeWidth={0.3}
          />
        ))}

        {/* Bars — stacked: backups + restore_tests + failures */}
        {data.map((d, i) => {
          const x    = (i / n) * w + gap / 2
          const bH   = (d.backups / maxVal) * chartH
          const rtH  = (d.restore_tests / maxVal) * chartH
          const fH   = (d.failures / maxVal) * chartH
          const totalH = bH + rtH + fH
          const baseY  = toY(d.backups + d.restore_tests + d.failures)

          return (
            <g key={i}>
              {/* Backups (bottom) */}
              {bH > 0 && (
                <rect x={x} y={baseY + rtH + fH} width={barW} height={bH}
                  fill={COLORS.backups} opacity={0.8} rx={0.4} />
              )}
              {/* Restore tests (middle) */}
              {rtH > 0 && (
                <rect x={x} y={baseY + fH} width={barW} height={rtH}
                  fill={COLORS.restoreTests} opacity={0.8} rx={0.4} />
              )}
              {/* Failures (top — red) */}
              {fH > 0 && (
                <rect x={x} y={baseY} width={barW} height={fH}
                  fill={COLORS.failures} opacity={0.9} rx={0.4} />
              )}
              {/* X-axis label */}
              {i % labelEvery === 0 && (
                <text
                  x={x + barW / 2} y={height - 3}
                  textAnchor="middle" fontSize={3}
                  fill="rgba(255,255,255,0.35)"
                >
                  {d.hour}
                </text>
              )}
            </g>
          )
        })}

        {/* Baseline */}
        <line x1={0} y1={padT + chartH} x2={100} y2={padT + chartH}
          stroke="rgba(255,255,255,0.1)" strokeWidth={0.3} />
      </svg>
    </div>
  )
}

export function ActivityLegend() {
  return (
    <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
      {[
        { color: COLORS.backups,      label: 'Backups' },
        { color: COLORS.restoreTests, label: 'Restore Tests' },
        { color: COLORS.failures,     label: 'Failures' },
      ].map(({ color, label }) => (
        <div key={label} style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
          <span style={{ width: 10, height: 10, borderRadius: 2, background: color, display: 'block' }} />
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{label}</span>
        </div>
      ))}
    </div>
  )
}
