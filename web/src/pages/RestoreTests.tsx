import { useEffect, useState } from 'react'
import { api, duration, timeAgo, type RestoreTest, type Snapshot, type BackupRepository } from '../api'
import { Card, SectionHeader } from '../components/Card'
import { StatusBadge } from '../components/StatusBadge'
import { Table } from '../components/Table'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { Modal } from '../components/Modal'
import { assessRestorePath, type PathRisk } from '../lib/pathSafety'

// Colour per advisory path-risk level (design-system functional colours).
const RISK_COLOR: Record<PathRisk, string> = {
  empty:   'var(--text-dim)',
  safe:    'var(--success)',
  caution: 'var(--warning)',
  danger:  'var(--error)',
}

export function RestoreTests() {
  const [tests,      setTests]      = useState<RestoreTest[]>([])
  const [snapshots,  setSnapshots]  = useState<Snapshot[]>([])
  const [repos,      setRepos]      = useState<BackupRepository[]>([])
  const [loading,    setLoading]    = useState(true)
  const [showNew,    setShowNew]    = useState(false)
  const [selSnap,    setSelSnap]    = useState('')
  const [selRepo,    setSelRepo]    = useState('')  // optional repo override
  const [targetPath, setTargetPath] = useState('')
  const [creating,   setCreating]   = useState(false)
  const [deleteFor,  setDeleteFor]  = useState<RestoreTest|null>(null)
  const [err,        setErr]        = useState<string|null>(null)

  const load = () => Promise.all([api.restoreTests(), api.snapshots(), api.repositories()])
    .then(([t,s,r]) => { setTests(t); setSnapshots(s); setRepos(r) })
    .finally(() => setLoading(false))

  useEffect(() => { load() }, [])

  // When a snapshot is selected, pre-select its default repository
  function onSnapChange(snapID: string) {
    setSelSnap(snapID)
    if (snapID) {
      const snap = snapshots.find(s => s.ID === snapID)
      setSelRepo(snap?.RepositoryID ?? '')
    } else {
      setSelRepo('')
    }
  }

  async function create() {
    if (!selSnap) { setErr('Select a snapshot.'); return }
    if (assessRestorePath(targetPath).risk === 'danger') {
      setErr('Der Zielpfad ist nicht erlaubt. Lass das Feld leer oder wähle ein Sandbox-Verzeichnis.')
      return
    }
    setCreating(true); setErr(null)
    try {
      await api.createRestoreTest(selSnap, targetPath || undefined, selRepo || undefined)
      setShowNew(false); setSelSnap(''); setSelRepo(''); setTargetPath(''); await load()
    } catch { setErr('Failed to create restore test. Is the snapshot valid?') }
    finally { setCreating(false) }
  }

  function repoName(id: string): string {
    const r = repos.find(r => r.ID === id)
    if (!r) return id.slice(0, 8) + '…'
    const loc = r.Location.replace(/\\/g, '/').split('/').pop() ?? r.Location
    return `${r.Type} — ${loc.length > 30 ? '…' + loc.slice(-28) : loc}`
  }

  const snapName = (id: string) => {
    const sn = snapshots.find(s => s.ID === id)
    return sn ? sn.EngineSnapshotID.slice(0,12)+'…' : id.slice(0,8)+'…'
  }

  const total  = snapshots.length
  const tested = snapshots.filter(s => tests.some(t => t.SnapshotID===s.ID && t.Status==='success')).length
  const pct    = total > 0 ? Math.round((tested/total)*100) : 0

  // Advisory target-path safety check (the agent enforces the real boundary).
  const pathCheck = assessRestorePath(targetPath)
  const pathBlocked = pathCheck.risk === 'danger'

  return (
    <div style={s.page}>
      <div style={s.topRow}>
        <SectionHeader title="Restore Tests" count={tests.length} />
        <button onClick={() => { setShowNew(true); setErr(null) }} style={s.newBtn}>
          + New Restore Test
        </button>
      </div>

      <div style={s.summary}>
        <div style={s.summaryCard}>
          <div style={{ fontSize:32, fontWeight:700, color: pct>0?'var(--success)':'var(--warning)' }}>{pct}%</div>
          <div style={s.summaryLabel}>Snapshots restore-tested</div>
          <div style={s.summaryDetail}>{tested} of {total} snapshots verified</div>
        </div>
        <div style={s.summaryInfo}>
          <h3 style={s.infoTitle}>Why restore tests matter</h3>
          <p style={s.infoText}>
            A backup is only proven when a restore succeeds.
            Create a restore test for each snapshot to verify
            your data can actually be recovered.
          </p>
          <p style={s.infoText}>
            After B14 (Restore Runner), tests will run automatically
            and verify file counts and checksums.
          </p>
        </div>
      </div>

      <Card>
        {loading ? <div style={s.load}>Loading…</div> : (
          <Table
            cols={[
              { header:'Status',         render:t => <StatusBadge status={t.Status} />, width:'110px' },
              { header:'Snapshot',       render:t => <span style={s.mono}>{snapName(t.SnapshotID)}</span> },
              { header:'Verified Files', render:t => t.VerifiedFiles != null
                  ? <span style={{color:'var(--success)'}}>{t.VerifiedFiles} files</span>
                  : <span style={s.dim}>—</span> },
              { header:'Verified Bytes', render:t => t.VerifiedBytes != null
                  ? <span style={{color:'var(--success)'}}>{(t.VerifiedBytes/1024).toFixed(1)} KB</span>
                  : <span style={s.dim}>—</span> },
              { header:'Duration',       render:t => duration(t.StartedAt, t.FinishedAt) },
              { header:'Created',        render:t => timeAgo(t.CreatedAt) },
              { header:'Error',          render:t => t.ErrorSummary
                  ? <span style={{color:'var(--error)',fontSize:11}}>{t.ErrorSummary}</span>
                  : <span style={s.dim}>—</span> },
              { header:'', render:t => (
                  <button onClick={() => setDeleteFor(t)} style={s.delBtn}>🗑</button>
                ), width:'40px' },
            ]}
            rows={tests} keyFn={t => t.ID}
            empty="No restore tests yet. Create one to verify your backups can be restored."
          />
        )}
      </Card>

      {showNew && (
        <Modal title="New Restore Test" onClose={() => { setShowNew(false); setErr(null) }}>
          <div style={s.field}>
            <label style={s.label}>Snapshot <span style={{color:'var(--error)'}}>*</span></label>
            <select style={s.select} value={selSnap} onChange={e => onSnapChange(e.target.value)}>
              <option value="">— select snapshot —</option>
              {snapshots.map(sn => (
                <option key={sn.ID} value={sn.ID}>
                  {sn.EngineSnapshotID.slice(0,16)}… — {new Date(sn.CreatedAt).toLocaleDateString()}
                  {sn.Paths?.length ? ` (${sn.Paths[0]})` : ''}
                </option>
              ))}
            </select>
          </div>
          <div style={s.field}>
            <label style={s.label}>
              Restore from Repository
              <span style={{fontWeight:400, fontSize:10, color:'var(--text-dim)', marginLeft:6}}>
                (default: snapshot's own repo)
              </span>
            </label>
            <select style={s.select} value={selRepo} onChange={e => setSelRepo(e.target.value)}>
              <option value="">— use snapshot's default repository —</option>
              {repos.map(r => (
                <option key={r.ID} value={r.ID}>{repoName(r.ID)}</option>
              ))}
            </select>
            {selRepo && selSnap && (
              <div style={s.repoHint}>
                {repos.find(r => r.ID === selRepo)?.Location ?? ''}
              </div>
            )}
          </div>
          <div style={s.field}>
            <label style={s.label}>Target Path <span style={{fontWeight:400,fontSize:10,color:'var(--text-dim)',marginLeft:6}}>(optional sandbox path)</span></label>
            <input
              style={{ ...s.input, ...(pathBlocked ? s.inputDanger : {}) }}
              value={targetPath}
              onChange={e => setTargetPath(e.target.value)}
              placeholder="leer lassen für automatische Sandbox — oder z. B. C:/tmp/restore-test"
            />
            <div style={{ ...s.pathBox, borderColor: RISK_COLOR[pathCheck.risk], color: RISK_COLOR[pathCheck.risk] }}>
              <span style={s.pathIcon}>
                {pathCheck.risk === 'danger' ? '⛔' : pathCheck.risk === 'caution' ? '⚠' : pathCheck.risk === 'safe' ? '✓' : 'ℹ'}
              </span>
              <span>
                <strong style={s.pathTitle}>{pathCheck.title}</strong>
                <span style={s.pathDetail}>{pathCheck.detail}</span>
              </span>
            </div>
          </div>
          {err && <div style={s.errBox}>{err}</div>}
          <div style={s.actions}>
            <button onClick={() => { setShowNew(false); setErr(null) }} style={s.cancelBtn}>Cancel</button>
            <button onClick={create} disabled={creating || !selSnap || pathBlocked}
              style={{ ...s.submitBtn, ...(creating || !selSnap || pathBlocked ? s.submitOff : {}) }}>
              {creating ? 'Creating…' : '✓ Create Restore Test'}
            </button>
          </div>
        </Modal>
      )}

      {deleteFor && (
        <ConfirmDialog
          title="Delete restore test?"
          message={`Delete this ${deleteFor.Status} restore test? The snapshot itself is not affected.`}
          confirmLabel="Delete" danger
          onConfirm={async () => { await api.deleteRestoreTest(deleteFor.ID); setDeleteFor(null); await load() }}
          onCancel={() => setDeleteFor(null)}
        />
      )}
    </div>
  )
}

const s: Record<string,React.CSSProperties> = {
  page:         { padding:'28px 36px', maxWidth:1200 },
  topRow:       { display:'flex', justifyContent:'space-between', alignItems:'flex-start', marginBottom:16 },
  newBtn:       { padding:'7px 16px', borderRadius:6, background:'var(--accent)', color:'#fff', border:'none', fontSize:13, fontWeight:600, cursor:'pointer' },
  summary:      { display:'grid', gridTemplateColumns:'200px 1fr', gap:16, marginBottom:20, background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:10, padding:'20px 24px' },
  summaryCard:  { textAlign:'center' as const, padding:'8px 0' },
  summaryLabel: { fontSize:12, fontWeight:600, color:'var(--text-muted)', textTransform:'uppercase' as const, letterSpacing:'0.06em', marginTop:6 },
  summaryDetail:{ fontSize:11, color:'var(--text-dim)', marginTop:4 },
  summaryInfo:  { borderLeft:'1px solid var(--border)', paddingLeft:20 },
  infoTitle:    { fontSize:14, fontWeight:700, color:'var(--text)', marginBottom:8 },
  infoText:     { fontSize:13, color:'var(--text-muted)', lineHeight:1.6, marginBottom:8 },
  load:         { padding:40, color:'var(--text-muted)', textAlign:'center' },
  mono:         { fontFamily:'var(--font-mono)', fontSize:12, color:'var(--accent)' },
  dim:          { color:'var(--text-dim)', fontSize:12 },
  delBtn:       { padding:'3px 8px', borderRadius:5, background:'rgba(244,63,94,0.08)', color:'var(--error)', border:'1px solid rgba(244,63,94,0.2)', fontSize:12, cursor:'pointer' },
  field:        { marginBottom:16 },
  label:        { display:'block', fontSize:11, fontWeight:700, color:'var(--text-dim)', textTransform:'uppercase' as const, letterSpacing:'0.08em', marginBottom:6 },
  hint:         { fontWeight:400, textTransform:'none' as const, fontSize:10, color:'var(--text-dim)', letterSpacing:0 },
  hint2:        { fontSize:11, color:'var(--text-dim)', marginTop:4 },
  repoHint:     { fontSize:10, color:'var(--text-dim)', marginTop:4, fontFamily:'var(--font-mono)' },
  select:       { width:'100%', padding:'8px 11px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, cursor:'pointer' },
  input:        { width:'100%', padding:'8px 11px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, outline:'none' },
  inputDanger:  { borderColor:'var(--error)' },
  pathBox:      { display:'flex', gap:8, alignItems:'flex-start', marginTop:8, padding:'8px 11px', borderRadius:6, border:'1px solid var(--border)', background:'rgba(255,255,255,0.02)', fontSize:11, lineHeight:1.5 },
  pathIcon:     { flexShrink:0, fontSize:13 },
  pathTitle:    { display:'block', fontWeight:700, marginBottom:2 },
  pathDetail:   { display:'block', color:'var(--text-muted)', fontWeight:400 },
  submitOff:    { opacity:0.4, cursor:'not-allowed' },
  errBox:       { background:'rgba(244,63,94,0.1)', border:'1px solid rgba(244,63,94,0.25)', borderRadius:6, padding:'8px 12px', fontSize:13, color:'var(--error)', marginBottom:8 },
  actions:      { display:'flex', gap:8, justifyContent:'flex-end', paddingTop:16, borderTop:'1px solid var(--border)' },
  cancelBtn:    { padding:'7px 16px', borderRadius:6, background:'transparent', border:'1px solid var(--border)', color:'var(--text-muted)', fontSize:13, cursor:'pointer' },
  submitBtn:    { padding:'7px 20px', borderRadius:6, background:'var(--success)', color:'#000', border:'none', fontSize:13, fontWeight:700, cursor:'pointer' },
}
