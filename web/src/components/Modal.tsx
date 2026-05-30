import type { ReactNode } from 'react'

interface ModalProps {
  title: string
  onClose: () => void
  children: ReactNode
}

export function Modal({ title, onClose, children }: ModalProps) {
  return (
    <div style={s.overlay} onClick={onClose}>
      <div style={s.box} onClick={e => e.stopPropagation()}>
        <div style={s.header}>
          <h2 style={s.title}>{title}</h2>
          <button onClick={onClose} style={s.close}>✕</button>
        </div>
        <div style={s.body}>{children}</div>
      </div>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  overlay: {
    position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
    display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100,
  },
  box: {
    background: 'var(--bg-card)', border: '1px solid var(--border)',
    borderRadius: 10, width: 480, maxWidth: '90vw', boxShadow: '0 24px 48px rgba(0,0,0,0.5)',
  },
  header: {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '18px 20px', borderBottom: '1px solid var(--border)',
  },
  title: { fontSize: 15, fontWeight: 700, color: 'var(--text)' },
  close: {
    background: 'none', border: 'none', color: 'var(--text-muted)',
    fontSize: 16, cursor: 'pointer', padding: '2px 6px', borderRadius: 4,
  },
  body: { padding: '20px' },
}
