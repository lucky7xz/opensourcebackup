// "Backup jetzt starten" — large action panel. System + Policy dropdowns and a
// prominent start button. The repository is shown read-only because it is fixed
// by the chosen policy (the run-now API takes only system + policy).
import type { BackupPolicy, System } from '../../api'

interface RunBackupPanelProps {
  systems:      System[]
  policies:     BackupPolicy[]
  selSystem:    string
  selPolicy:    string
  repoLabel:    string | null   // resolved from the selected policy's repository
  starting:     boolean
  startErr:     string | null
  onSystem:     (id: string) => void
  onPolicy:     (id: string) => void
  onStart:      () => void
}

export function RunBackupPanel(p: RunBackupPanelProps) {
  const disabled = p.starting || !p.selSystem || !p.selPolicy

  return (
    <div style={s.card}>
      <div style={s.grid}>
        <Field label="System">
          <select style={s.select} value={p.selSystem} onChange={e => p.onSystem(e.target.value)}>
            <option value="">— wählen —</option>
            {p.systems.map(sys => <option key={sys.ID} value={sys.ID}>{sys.Hostname}</option>)}
          </select>
        </Field>

        <Field label="Policy">
          <select style={s.select} value={p.selPolicy} onChange={e => p.onPolicy(e.target.value)}>
            <option value="">— wählen —</option>
            {p.policies.map(pol => <option key={pol.ID} value={pol.ID}>{pol.Name}</option>)}
          </select>
        </Field>

        <Field label="Repository">
          <div style={s.readonly} title="Wird durch die gewählte Policy bestimmt">
            {p.repoLabel ?? <span style={s.placeholder}>— aus Policy —</span>}
          </div>
        </Field>

        <div style={s.startWrap}>
          <button style={{ ...s.startBtn, ...(disabled ? s.startOff : {}) }} disabled={disabled} onClick={p.onStart}>
            {p.starting ? '…' : '▶ Backup starten'}
          </button>
          <span style={s.hint}>Jetzt einmalig ausführen</span>
        </div>
      </div>

      {p.startErr && <div style={s.errBox}>{p.startErr}</div>}
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div style={s.field}>
      <label style={s.label}>{label}</label>
      {children}
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  card:       { background: 'linear-gradient(180deg, rgba(21,28,46,0.95) 0%, rgba(10,15,27,0.95) 100%)', border: '1px solid var(--border)', borderRadius: 16, padding: 20, boxShadow: '0 4px 20px rgba(0,0,0,0.18)' },
  grid:       { display: 'flex', gap: 16, alignItems: 'flex-end', flexWrap: 'wrap' },
  field:      { display: 'flex', flexDirection: 'column', gap: 7, flex: 1, minWidth: 180 },
  label:      { fontSize: 10, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.1em' },
  select:     { padding: '11px 12px', background: 'var(--bg)', border: '1px solid var(--border)', borderRadius: 10, color: 'var(--text)', fontSize: 14, cursor: 'pointer' },
  readonly:   { padding: '11px 12px', background: 'rgba(255,255,255,0.02)', border: '1px solid var(--border)', borderRadius: 10, color: 'var(--text)', fontSize: 14, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' },
  placeholder:{ color: 'var(--text-dim)' },
  startWrap:  { display: 'flex', flexDirection: 'column', gap: 6, flexShrink: 0, alignItems: 'stretch' },
  startBtn:   { padding: '12px 28px', borderRadius: 10, background: 'var(--accent)', color: '#000', border: 'none', fontSize: 14, fontWeight: 700, cursor: 'pointer', whiteSpace: 'nowrap' },
  startOff:   { opacity: 0.4, cursor: 'not-allowed' },
  hint:       { fontSize: 11, color: 'var(--text-dim)', textAlign: 'center' },
  errBox:     { marginTop: 16, background: 'rgba(239,68,68,0.08)', border: '1px solid rgba(239,68,68,0.2)', borderRadius: 10, padding: '10px 14px', fontSize: 13, color: 'var(--error)' },
}
