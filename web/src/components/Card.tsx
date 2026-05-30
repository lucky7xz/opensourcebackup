import type { ReactNode } from 'react'

interface CardProps {
  title?: string
  children: ReactNode
  accent?: string
}

export function Card({ title, children, accent }: CardProps) {
  return (
    <div style={{ ...styles.card, ...(accent ? { borderLeftColor: accent, borderLeftWidth: 3 } : {}) }}>
      {title && <h3 style={styles.title}>{title}</h3>}
      {children}
    </div>
  )
}

export function StatCard({ label, value, sub, color }: { label: string; value: string | number; sub?: string; color?: string }) {
  return (
    <div style={styles.card}>
      <div style={{ fontSize: 28, fontWeight: 700, color: color ?? 'var(--accent-cyan)' }}>{value}</div>
      <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary)', marginTop: 4 }}>{label}</div>
      {sub && <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 2 }}>{sub}</div>}
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  card: {
    background: 'var(--bg-card)',
    border: '1px solid var(--border)',
    borderRadius: 10,
    padding: '20px 24px',
  },
  title: {
    fontSize: 13,
    fontWeight: 600,
    color: 'var(--text-secondary)',
    textTransform: 'uppercase',
    letterSpacing: '0.06em',
    marginBottom: 16,
  },
}
