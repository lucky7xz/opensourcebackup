import { useEffect, useState } from 'react'
import { api, type BackupRepository } from '../api'
import { Card, SectionHeader } from '../components/Card'
import { Table } from '../components/Table'
import { Modal } from '../components/Modal'
import { ConfirmDialog } from '../components/ConfirmDialog'

const TYPES = [
  { value: 'restic',     label: 'Restic',       hint: 'Local, S3, SFTP, B2, Azure, GCS' },
  { value: 'borg',       label: 'Borg',          hint: 'SSH / local — deduplicated' },
  { value: 'pgbackrest', label: 'pgBackRest',    hint: 'PostgreSQL — WAL archiving' },
  { value: 'velero',     label: 'Velero',        hint: 'Kubernetes cluster backup' },
]

const LOCATION_HINTS: Record<string, string> = {
  restic:     's3:http://minio:9000/bucket  or  /local/path  or  sftp:user@host:/path',
  borg:       'ssh://user@host/./backups  or  /local/path',
  pgbackrest: 'path=/var/lib/pgbackrest  or  s3:bucket',
  velero:     's3://bucket/velero  or  azure://container',
}

export function Repositories() {
  const [repos,      setRepos]      = useState<BackupRepository[]>([])
  const [loading,    setLoading]    = useState(true)
  const [showForm,   setShowForm]   = useState(false)
  const [deleteFor,  setDeleteFor]  = useState<BackupRepository|null>(null)
  const [saving,     setSaving]     = useState(false)
  const [err,        setErr]        = useState<string|null>(null)

  // form
  const [type,       setType]       = useState('restic')
  const [location,   setLocation]   = useState('')
  const [encryption, setEncryption] = useState('aes256')
  const [worm,       setWorm]       = useState(false)

  const load = () => api.repositories().then(setRepos).finally(() => setLoading(false))
  useEffect(() => { load() }, [])

  function resetForm() {
    setType('restic'); setLocation(''); setEncryption('aes256'); setWorm(false); setErr(null)
  }

  async function save() {
    if (!location.trim()) { setErr('Location is required.'); return }
    setSaving(true); setErr(null)
    try {
      await api.createRepository({
        Type:              type,
        Location:          location.trim(),
        EncryptionMode:    encryption.trim() || undefined,
        ObjectLockEnabled: worm,
      })
      setShowForm(false); resetForm(); await load()
    } catch { setErr('Could not create repository. Check the location format.') }
    finally { setSaving(false) }
  }

  return (
    <div style={s.page}>
      <div style={s.topRow}>
        <SectionHeader title="Repositories" count={repos.length} />
        <button onClick={() => { resetForm(); setShowForm(true) }} style={s.newBtn}>
          + New Repository
        </button>
      </div>

      <p style={s.sub}>
        Repositories define <strong>where</strong> backups are stored.
        Every policy must reference a repository.
      </p>

      <Card>
        {loading ? <div style={s.load}>Loading…</div> : (
          <Table
            cols={[
              { header:'Type',       render:r => <span style={s.badge}>{r.Type}</span>, width:'100px' },
              { header:'Location',   render:r => <span style={s.mono}>{r.Location}</span> },
              { header:'Encryption', render:r => r.EncryptionMode
                  ? <span style={s.enc}>{r.EncryptionMode}</span>
                  : <span style={s.dim}>—</span>, width:'100px' },
              { header:'WORM Lock',  render:r => r.ObjectLockEnabled
                  ? <span style={s.worm}>✓ enabled</span>
                  : <span style={s.dim}>—</span>, width:'100px' },
              { header:'ID',         render:r => <span style={s.mono}>{r.ID.slice(0,8)}…</span> },
              { header:'Created',    render:r => new Date(r.CreatedAt).toLocaleDateString() },
              { header:'',           render:r => (
                  <button onClick={() => setDeleteFor(r)} style={s.delBtn}>🗑</button>
                ), width:'40px' },
            ]}
            rows={repos} keyFn={r => r.ID}
            empty="No repositories yet. Click '+ New Repository' to add your first backup destination."
          />
        )}
      </Card>

      {/* ── New Repository Modal ── */}
      {showForm && (
        <Modal title="Register New Repository" onClose={() => { setShowForm(false); resetForm() }}>
          <div>
            <div style={s.field}>
              <label style={s.label}>Type <span style={s.req}>*</span></label>
              <div style={s.typeGrid}>
                {TYPES.map(t => (
                  <div key={t.value} onClick={() => setType(t.value)}
                    style={{...s.typeCard, ...(type===t.value ? s.typeCardOn : {})}}>
                    <div style={s.typeName}>{t.label}</div>
                    <div style={s.typeHint}>{t.hint}</div>
                  </div>
                ))}
              </div>
            </div>

            <div style={s.field}>
              <label style={s.label}>Location <span style={s.req}>*</span></label>
              <input style={s.input} value={location} onChange={e => setLocation(e.target.value)}
                placeholder={LOCATION_HINTS[type]} />
              <div style={s.hint2}>{LOCATION_HINTS[type]}</div>
            </div>

            <div style={s.row2}>
              <div style={s.field}>
                <label style={s.label}>Encryption Mode</label>
                <input style={s.input} value={encryption} onChange={e => setEncryption(e.target.value)}
                  placeholder="aes256" />
                <div style={s.hint2}>Leave blank to disable encryption (not recommended)</div>
              </div>
              <div style={s.field}>
                <label style={s.label}>WORM / Object Lock</label>
                <div style={s.toggle} onClick={() => setWorm(v => !v)}>
                  <div style={{...s.toggleDot, ...(worm ? s.toggleOn : {})}}>
                    {worm ? '✓' : ''}
                  </div>
                  <span style={{ fontSize:13, color: worm ? 'var(--success)' : 'var(--text-dim)' }}>
                    {worm ? 'Enabled — ransomware protection' : 'Disabled'}
                  </span>
                </div>
                <div style={s.hint2}>Requires MinIO or S3 Object Lock support</div>
              </div>
            </div>

            {err && <div style={s.errBox}>{err}</div>}
            <div style={s.actions}>
              <button onClick={() => { setShowForm(false); resetForm() }} style={s.cancelBtn}>Cancel</button>
              <button onClick={save} disabled={saving || !location.trim()} style={s.submitBtn}>
                {saving ? 'Saving…' : '✓ Register Repository'}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {deleteFor && (
        <ConfirmDialog
          title={`Delete repository?`}
          message={`Delete "${deleteFor.Location}"? Policies linked to this repository will lose their backup destination.`}
          confirmLabel="Delete Repository" danger
          onConfirm={async () => { await api.deleteRepository(deleteFor.ID); setDeleteFor(null); await load() }}
          onCancel={() => setDeleteFor(null)}
        />
      )}
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  page:       { padding:'28px 36px', maxWidth:1100 },
  topRow:     { display:'flex', justifyContent:'space-between', alignItems:'flex-start', marginBottom:6 },
  sub:        { fontSize:13, color:'var(--text-muted)', marginBottom:16 },
  load:       { padding:40, color:'var(--text-muted)', textAlign:'center' },
  newBtn:     { padding:'7px 16px', borderRadius:6, background:'var(--accent)', color:'#fff', border:'none', fontSize:13, fontWeight:600, cursor:'pointer' },
  mono:       { fontFamily:'var(--font-mono)', fontSize:12, color:'var(--accent)' },
  badge:      { display:'inline-block', padding:'2px 8px', borderRadius:4, background:'rgba(59,130,246,0.1)', color:'var(--accent)', fontSize:11, fontWeight:600 },
  enc:        { fontSize:12, color:'var(--success)' },
  worm:       { fontSize:12, color:'var(--success)' },
  dim:        { color:'var(--text-dim)', fontSize:12 },
  delBtn:     { padding:'3px 8px', borderRadius:5, background:'rgba(244,63,94,0.08)', color:'var(--error)', border:'1px solid rgba(244,63,94,0.2)', fontSize:12, cursor:'pointer' },
  field:      { marginBottom:16 },
  row2:       { display:'grid', gridTemplateColumns:'1fr 1fr', gap:14 },
  label:      { display:'block', fontSize:11, fontWeight:700, color:'var(--text-dim)', textTransform:'uppercase' as const, letterSpacing:'0.08em', marginBottom:6 },
  req:        { color:'var(--error)' },
  hint2:      { fontSize:11, color:'var(--text-dim)', marginTop:4 },
  input:      { width:'100%', padding:'8px 11px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, outline:'none' },
  typeGrid:   { display:'grid', gridTemplateColumns:'repeat(2,1fr)', gap:8 },
  typeCard:   { padding:'10px 12px', borderRadius:7, border:'1px solid var(--border)', cursor:'pointer', transition:'all 0.12s' },
  typeCardOn: { borderColor:'var(--accent)', background:'var(--accent-dim)' },
  typeName:   { fontWeight:600, color:'var(--text)', fontSize:13, marginBottom:2 },
  typeHint:   { fontSize:11, color:'var(--text-dim)' },
  toggle:     { display:'flex', alignItems:'center', gap:10, padding:'8px 0', cursor:'pointer' },
  toggleDot:  { width:32, height:18, borderRadius:9, background:'var(--border)', display:'flex', alignItems:'center', justifyContent:'center', fontSize:10, color:'#fff', transition:'background 0.15s' },
  toggleOn:   { background:'var(--success)' },
  errBox:     { background:'rgba(244,63,94,0.1)', border:'1px solid rgba(244,63,94,0.25)', borderRadius:6, padding:'8px 12px', fontSize:13, color:'var(--error)', marginBottom:8 },
  actions:    { display:'flex', gap:8, justifyContent:'flex-end', paddingTop:16, borderTop:'1px solid var(--border)' },
  cancelBtn:  { padding:'7px 16px', borderRadius:6, background:'transparent', border:'1px solid var(--border)', color:'var(--text-muted)', fontSize:13, cursor:'pointer' },
  submitBtn:  { padding:'7px 20px', borderRadius:6, background:'var(--success)', color:'#000', border:'none', fontSize:13, fontWeight:700, cursor:'pointer' },
}
