interface Props {
  title: string
  message: string
  confirmLabel?: string
  danger?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({ title, message, confirmLabel = 'Confirm', danger = false, onConfirm, onCancel }: Props) {
  return (
    <div style={s.overlay} onClick={onCancel}>
      <div style={s.box} onClick={e => e.stopPropagation()}>
        <h3 style={s.title}>{title}</h3>
        <p style={s.msg}>{message}</p>
        <div style={s.actions}>
          <button onClick={onCancel} style={s.cancel}>Cancel</button>
          <button onClick={onConfirm} style={{ ...s.confirm, ...(danger ? s.danger : {}) }}>
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  overlay: { position:'fixed', inset:0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:200 },
  box:     { background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:10, padding:'24px 28px', width:400, maxWidth:'90vw', boxShadow:'0 24px 48px rgba(0,0,0,0.5)' },
  title:   { fontSize:16, fontWeight:700, color:'var(--text)', marginBottom:10 },
  msg:     { fontSize:13, color:'var(--text-muted)', lineHeight:1.6, marginBottom:20 },
  actions: { display:'flex', gap:8, justifyContent:'flex-end' },
  cancel:  { padding:'7px 16px', borderRadius:6, background:'transparent', border:'1px solid var(--border)', color:'var(--text-muted)', fontSize:13, cursor:'pointer' },
  confirm: { padding:'7px 18px', borderRadius:6, border:'none', fontSize:13, fontWeight:600, cursor:'pointer', background:'var(--accent)', color:'#fff' },
  danger:  { background:'var(--error)', color:'#fff' },
}
