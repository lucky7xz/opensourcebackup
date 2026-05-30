const colors: Record<string, { bg: string; text: string }> = {
  success:  { bg: 'rgba(0,255,136,0.12)', text: 'var(--accent-green)' },
  running:  { bg: 'rgba(0,212,255,0.12)', text: 'var(--accent-cyan)' },
  pending:  { bg: 'rgba(255,149,0,0.12)', text: 'var(--accent-orange)' },
  failed:   { bg: 'rgba(255,71,87,0.12)', text: 'var(--accent-red)' },
  warning:  { bg: 'rgba(255,149,0,0.12)', text: 'var(--accent-orange)' },
  verified: { bg: 'rgba(0,255,136,0.12)', text: 'var(--accent-green)' },
  unverified: { bg: 'rgba(255,149,0,0.12)', text: 'var(--accent-orange)' },
  standard: { bg: 'rgba(148,163,184,0.12)', text: 'var(--text-secondary)' },
  critical: { bg: 'rgba(255,71,87,0.12)', text: 'var(--accent-red)' },
}

export function StatusBadge({ status }: { status: string }) {
  const c = colors[status.toLowerCase()] ?? { bg: 'rgba(148,163,184,0.12)', text: 'var(--text-secondary)' }
  return (
    <span style={{
      display: 'inline-block',
      padding: '2px 10px',
      borderRadius: 20,
      fontSize: 11,
      fontWeight: 600,
      background: c.bg,
      color: c.text,
      textTransform: 'uppercase',
      letterSpacing: '0.05em',
    }}>
      {status}
    </span>
  )
}
