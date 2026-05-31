import { useEffect, useState } from 'react'
import { api, type BackupPolicy, type BackupRepository } from '../api'
import { Card, SectionHeader } from '../components/Card'
import { Table } from '../components/Table'
import { Modal } from '../components/Modal'
import { ConfirmDialog } from '../components/ConfirmDialog'

const ENGINES = [
  { value: 'restic',     label: 'Restic',     desc: 'Dateien & Ordner — Windows, Linux, NAS, S3' },
  { value: 'borg',       label: 'Borg',        desc: 'Linux-Server via SSH — sehr effizient' },
  { value: 'pgbackrest', label: 'pgBackRest',  desc: 'PostgreSQL-Datenbanken — WAL & Point-in-Time' },
  { value: 'velero',     label: 'Velero',      desc: 'Kubernetes-Cluster — Deployments & Volumes' },
]
const SCHEDULES = [
  { label: 'Daily at 02:00',    value: '0 2 * * *' },
  { label: 'Daily at midnight', value: '0 0 * * *' },
  { label: 'Every 6 hours',     value: '0 */6 * * *' },
  { label: 'Every hour',        value: '0 * * * *' },
  { label: 'Weekly (Sun 2am)',  value: '0 2 * * 0' },
  { label: 'Manual only',       value: '' },
  { label: 'Custom…',           value: '__custom' },
]

export function Policies() {
  const [policies,   setPolicies]   = useState<BackupPolicy[]>([])
  const [repos,      setRepos]      = useState<BackupRepository[]>([])
  const [loading,    setLoading]    = useState(true)
  const [showForm,   setShowForm]   = useState(false)
  const [editingId,  setEditingId]  = useState<string|null>(null) // null = create, id = update
  const [deleteFor,  setDeleteFor]  = useState<BackupPolicy|null>(null)
  const [saving,     setSaving]     = useState(false)
  const [err,        setErr]        = useState<string|null>(null)

  // form
  const [name,       setName]       = useState('')
  const [engine,     setEngine]     = useState('restic')
  const [repoID,     setRepoID]     = useState('')
  const [schedSel,   setSchedSel]   = useState('0 2 * * *')
  const [schedCustom,setSchedCustom]= useState('')
  const [includes,   setIncludes]   = useState<string[]>([''])
  const [excludes,   setExcludes]   = useState<string[]>([])
  const [retDaily,   setRetDaily]   = useState('7')
  const [retWeekly,  setRetWeekly]  = useState('4')
  const [retMonthly, setRetMonthly] = useState('12')

  const load = () => Promise.all([api.policies(), api.repositories()])
    .then(([p,r]) => { setPolicies(p); setRepos(r) })
    .finally(() => setLoading(false))

  useEffect(() => { load() }, [])

  function resetForm() {
    setName(''); setEngine('restic'); setRepoID('')
    setSchedSel('0 2 * * *'); setSchedCustom('')
    setIncludes(['']); setExcludes([]); setErr(null)
    setRetDaily('7'); setRetWeekly('4'); setRetMonthly('12')
  }

  async function save() {
    if (!name.trim()) { setErr('Policy name is required.'); return }
    const schedule     = schedSel === '__custom' ? schedCustom.trim() : schedSel
    const cleanIncludes = includes.map(p => p.trim()).filter(Boolean)
    if (cleanIncludes.length === 0) { setErr('At least one include path is required.'); return }
    setSaving(true); setErr(null)
    try {
      const data = {
        Name:         name.trim(),
        Engine:       engine,
        RepositoryID: repoID || undefined,
        Schedule:     schedule || undefined,
        Includes:     cleanIncludes,
        Excludes:     excludes.map(p => p.trim()).filter(Boolean),
        Retention:    { daily: Number(retDaily), weekly: Number(retWeekly), monthly: Number(retMonthly) },
      }
      if (editingId) {
        await api.updatePolicy(editingId, data)
      } else {
        await api.createPolicy(data)
      }
      setShowForm(false); setEditingId(null); resetForm(); await load()
    } catch { setErr('Failed to save. Check all values.') }
    finally { setSaving(false) }
  }

  return (
    <div style={s.page}>
      <div style={s.topRow}>
        <SectionHeader title="Policies" count={policies.length} />
        <button onClick={() => { resetForm(); setShowForm(true) }} style={s.newBtn}>+ New Policy</button>
      </div>

      <Card>
        {loading ? <div style={s.load}>Loading…</div> : (
          <Table
            cols={[
              { header:'Name',       render:p => <span style={s.name}>{p.Name}</span> },
              { header:'Engine',     render:p => {
                const e = ENGINES.find(e=>e.value===p.Engine)
                return <span style={s.badge} title={e?.desc}>{p.Engine}</span>
              }, width:'90px' },
              { header:'Schedule',   render:p => p.Schedule ? <span style={s.mono}>{p.Schedule}</span> : <span style={s.dim}>manual</span> },
              { header:'Includes',   render:p => <div>{(p.Includes??[]).map(i=><div key={i} style={s.path}>{i}</div>)}</div> },
              { header:'Repository', render:p => p.RepositoryID
                  ? <span style={s.mono}>{repos.find(r=>r.ID===p.RepositoryID)?.Location ?? p.RepositoryID.slice(0,8)+'…'}</span>
                  : <span style={s.warn}>⚠ no repository</span> },
              { header:'', render:p => (
                <div style={{display:'flex',gap:6}}>
                  <button onClick={() => {
                    setEditingId(p.ID); setShowForm(true)
                    setName(p.Name); setEngine(p.Engine)
                    setRepoID(p.RepositoryID??'')
                    const sc = p.Schedule ?? ''
                    const known = SCHEDULES.find(s=>s.value===sc)
                    setSchedSel(known ? sc : (sc ? '__custom' : ''))
                    setSchedCustom(sc)
                    setIncludes(p.Includes?.length ? p.Includes : [''])
                    setExcludes(p.Excludes ?? [])
                  }} style={s.editBtn}>✏</button>
                  <button onClick={()=>setDeleteFor(p)} style={s.delBtn}>🗑</button>
                </div>
              ), width:'70px' },
            ]}
            rows={policies} keyFn={p=>p.ID}
            empty="No policies yet. Click '+ New Policy' to create one."
          />
        )}
      </Card>

      {showForm && (
        <Modal title={editingId ? 'Edit Policy' : 'New Backup Policy'} onClose={()=>{setShowForm(false);setEditingId(null);resetForm()}}>
          <div>
            <div style={s.row2}>
              <div style={s.field}>
                <label style={s.label}>Policy Name</label>
                <input style={s.input} value={name} onChange={e=>setName(e.target.value)}
                  placeholder="e.g. nightly-documents" autoFocus />
              </div>
              <div style={s.field}>
                <label style={s.label}>Engine</label>
                <select style={s.select} value={engine} onChange={e=>setEngine(e.target.value)}>
                  {ENGINES.map(e=><option key={e.value} value={e.value}>{e.label} — {e.desc}</option>)}
                </select>
              </div>
            </div>

            <div style={s.field}>
              <label style={s.label}>Repository <span style={s.hint2}>(where to store backups)</span></label>
              <select style={s.select} value={repoID} onChange={e=>setRepoID(e.target.value)}>
                <option value="">— select repository —</option>
                {repos.map(r=><option key={r.ID} value={r.ID}>{r.Location} ({r.Type})</option>)}
              </select>
              {!repoID && <div style={s.warnText}>⚠ Without a repository the agent cannot run this policy.</div>}
            </div>

            <div style={s.field}>
              <label style={s.label}>Schedule</label>
              <select style={s.select} value={schedSel} onChange={e=>setSchedSel(e.target.value)}>
                {SCHEDULES.map(sc=><option key={sc.value} value={sc.value}>{sc.label}{sc.value&&sc.value!=='__custom'?` — ${sc.value}`:''}</option>)}
              </select>
              {schedSel==='__custom' && (
                <input style={{...s.input,marginTop:6}} value={schedCustom} onChange={e=>setSchedCustom(e.target.value)}
                  placeholder="cron: 0 3 * * 1-5" />
              )}
            </div>

            <div style={s.field}>
              <label style={s.label}>Include Paths <span style={s.hint2}>(folders to back up)</span></label>
              {includes.map((p,i)=>(
                <div key={i} style={s.pathRow}>
                  <input style={s.input} value={p}
                    onChange={e=>setIncludes(prev=>prev.map((v,j)=>j===i?e.target.value:v))}
                    placeholder={i===0?'e.g. C:/Users/Admin/Documents':'Another path…'} />
                  {includes.length>1 && (
                    <button onClick={()=>setIncludes(prev=>prev.filter((_,j)=>j!==i))} style={s.removeBtn}>✕</button>
                  )}
                </div>
              ))}
              <button onClick={()=>setIncludes(prev=>[...prev,''])} style={s.addBtn}>+ Add path</button>
            </div>

            <div style={s.field}>
              <label style={s.label}>Exclude Paths <span style={s.hint2}>(optional)</span></label>
              {excludes.map((p,i)=>(
                <div key={i} style={s.pathRow}>
                  <input style={s.input} value={p}
                    onChange={e=>setExcludes(prev=>prev.map((v,j)=>j===i?e.target.value:v))}
                    placeholder="e.g. C:/Users/Admin/AppData" />
                  <button onClick={()=>setExcludes(prev=>prev.filter((_,j)=>j!==i))} style={s.removeBtn}>✕</button>
                </div>
              ))}
              <button onClick={()=>setExcludes(prev=>[...prev,''])} style={s.addBtn}>+ Add exclusion</button>
            </div>

            <div style={s.field}>
              <label style={s.label}>Retention (keep X snapshots)</label>
              <div style={s.row3}>
                {[['Daily',retDaily,setRetDaily],['Weekly',retWeekly,setRetWeekly],['Monthly',retMonthly,setRetMonthly]].map(([lbl,val,set])=>(
                  <div key={lbl as string}>
                    <div style={s.retLabel}>{lbl as string}</div>
                    <input style={s.retInput} type="number" min="0" value={val as string}
                      onChange={e=>(set as (v:string)=>void)(e.target.value)} />
                  </div>
                ))}
              </div>
            </div>

            {err && <div style={s.errBox}>{err}</div>}

            <div style={s.actions}>
              <button onClick={()=>{setShowForm(false);setEditingId(null);resetForm()}} style={s.cancelBtn}>Cancel</button>
              <button onClick={save} disabled={saving} style={s.submitBtn}>
                {saving ? 'Saving…' : editingId ? '✓ Save Changes' : '✓ Create Policy'}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {deleteFor && (
        <ConfirmDialog
          title={`Delete "${deleteFor.Name}"?`}
          message="Existing jobs and snapshots will remain, but no new backups will be scheduled."
          confirmLabel="Delete Policy" danger
          onConfirm={async()=>{await api.deletePolicy(deleteFor.ID);setDeleteFor(null);await load()}}
          onCancel={()=>setDeleteFor(null)}
        />
      )}
    </div>
  )
}

const s: Record<string,React.CSSProperties> = {
  page:      { padding:'28px 36px', maxWidth:1200 },
  topRow:    { display:'flex', justifyContent:'space-between', alignItems:'flex-start', marginBottom:4 },
  load:      { padding:40, color:'var(--text-muted)', textAlign:'center' },
  name:      { fontWeight:600, color:'var(--text)' },
  mono:      { fontFamily:'var(--font-mono)', fontSize:12, color:'var(--accent)' },
  badge:     { display:'inline-block', padding:'2px 8px', borderRadius:4, background:'rgba(59,130,246,0.1)', color:'var(--accent)', fontSize:11, fontWeight:600 },
  path:      { fontFamily:'var(--font-mono)', fontSize:11, color:'var(--text-muted)' },
  dim:       { color:'var(--text-dim)', fontSize:12 },
  warn:      { color:'var(--warning)', fontSize:12 },
  editBtn:   { padding:'3px 8px', borderRadius:5, background:'rgba(245,158,11,0.08)', color:'var(--warning)', border:'1px solid rgba(245,158,11,0.2)', fontSize:12, cursor:'pointer' },
  delBtn:    { padding:'3px 8px', borderRadius:5, background:'rgba(244,63,94,0.08)', color:'var(--error)', border:'1px solid rgba(244,63,94,0.2)', fontSize:12, cursor:'pointer' },
  newBtn:    { padding:'7px 16px', borderRadius:6, background:'var(--accent)', color:'#fff', border:'none', fontSize:13, fontWeight:600, cursor:'pointer' },
  row2:      { display:'grid', gridTemplateColumns:'1fr 1fr', gap:12, marginBottom:4 },
  row3:      { display:'flex', gap:12 },
  field:     { marginBottom:16 },
  label:     { display:'block', fontSize:11, fontWeight:700, color:'var(--text-dim)', textTransform:'uppercase' as const, letterSpacing:'0.08em', marginBottom:6 },
  hint2:     { fontWeight:400, textTransform:'none' as const, color:'var(--text-dim)', letterSpacing:0, fontSize:10 },
  input:     { width:'100%', padding:'8px 11px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, outline:'none' },
  select:    { width:'100%', padding:'8px 11px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, cursor:'pointer' },
  warnText:  { fontSize:11, color:'var(--warning)', marginTop:4 },
  pathRow:   { display:'flex', gap:6, marginBottom:6, alignItems:'center' },
  removeBtn: { padding:'6px 10px', borderRadius:5, background:'rgba(244,63,94,0.08)', color:'var(--error)', border:'1px solid rgba(244,63,94,0.2)', fontSize:12, cursor:'pointer', flexShrink:0 },
  addBtn:    { padding:'5px 12px', borderRadius:5, background:'transparent', color:'var(--accent)', border:'1px dashed rgba(59,130,246,0.4)', fontSize:12, cursor:'pointer', marginTop:2 },
  retLabel:  { fontSize:11, color:'var(--text-dim)', marginBottom:4 },
  retInput:  { width:70, padding:'7px 10px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, textAlign:'center' as const },
  errBox:    { background:'rgba(244,63,94,0.1)', border:'1px solid rgba(244,63,94,0.25)', borderRadius:6, padding:'8px 12px', fontSize:13, color:'var(--error)', marginBottom:8 },
  actions:   { display:'flex', gap:8, justifyContent:'flex-end', paddingTop:16, borderTop:'1px solid var(--border)' },
  cancelBtn: { padding:'7px 16px', borderRadius:6, background:'transparent', border:'1px solid var(--border)', color:'var(--text-muted)', fontSize:13, cursor:'pointer' },
  submitBtn: { padding:'7px 20px', borderRadius:6, background:'var(--success)', color:'#000', border:'none', fontSize:13, fontWeight:700, cursor:'pointer' },
}
