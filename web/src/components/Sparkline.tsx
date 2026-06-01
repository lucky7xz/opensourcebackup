interface Props {
  data:   number[]   // values to plot
  width?: number
  height?: number
  color?: string
  filled?: boolean
}

/**
 * Minimal SVG sparkline — no external library.
 * Renders a smooth line (and optional fill) for trend display in KPI cards.
 */
export function Sparkline({ data, width = 80, height = 28, color = '#22c55e', filled = true }: Props) {
  if (!data || data.length < 2) return null

  const min = Math.min(...data)
  const max = Math.max(...data)
  const range = max - min || 1

  const pts = data.map((v, i) => {
    const x = (i / (data.length - 1)) * width
    const y = height - ((v - min) / range) * (height - 4) - 2
    return `${x.toFixed(1)},${y.toFixed(1)}`
  })

  const polyline = pts.join(' ')
  const fillPath  = `M0,${height} L${pts[0]} L${pts.slice(1).join(' L')} L${width},${height} Z`

  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`}
      style={{ display: 'block', overflow: 'visible' }}>
      {filled && (
        <path d={fillPath} fill={color} opacity={0.12} />
      )}
      <polyline
        points={polyline}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}
