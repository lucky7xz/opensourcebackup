import { useEffect, useState } from 'react'
import { api, timeAgo, type BackupJob, type BackupPolicy, type System } from '../api'
import { Card, SectionHeader } from '../components/Card'
import { StatusBadge } from '../components/StatusBadge'
import { Table } from '../components/Table'
import { Modal } from '../components/Modal'
import { ConfirmDialog } from '../components/ConfirmDialog'

function systemStatus(s: System, jobs: BackupJob[]): string {
  const last = jobs.filter(j => j.SystemID===s.ID).sort((a,b) => new Date(b.CreatedAt).getTime()-new Date(a.CreatedAt).getTime())[0]
  if (!last) return 'unknown'
  return last.Status==='success' ? 'healthy' : last.Status==='failed' ? 'failed' : last.Status
}

export function Systems() {
  const [systems,  setSystems]  = useState<System[]>([])
  const [jobs,     setJobs]     = useState<BackupJob[]>([])
  const [policies, setPolicies] = useState<BackupPolicy[]>([])
  const [loading,  setLoading]  = useState(true)
  const [runFor,     setRunFor]     = useState<System|null>(null)
  const [deleteFor,  setDeleteFor]  = useState<System|null>(null)
  const [selPolicy,  setSelPolicy]  = useState('')
  const [creating,   setCreating]   = useState(false)
  const [deleting,   setDeleting]   = useState(false)
  const [msg,        setMsg]        = useState<string|null>(null)

  const load = () => Promise.all([api.systems(), api.jobs(), api.policies()])
    .then(([s,j,p]) => { setSystems(s); setJobs(j); setPolicies(p) })
    .finally(() => setLoading(false))

  useEffect(() => { load() }, [])

  async function deleteSystem() {
    if (!deleteFor) return
    setDeleting(true)
    try {
      await api.deleteSystem(deleteFor.ID)
      setMsg(`${deleteFor.Hostname} removed.`)
      setDeleteFor(null)
      await load()
    } catch {
      setMsg('Failed to delete system.')
    } finally { setDeleting(false) }
  }

  async function runBackup() {
    if (!runFor || !selPolicy) return
    setCreating(true)
    try {
      await api.createJob(runFor.ID, selPolicy)
      setMsg(`Job created for ${runFor.Hostname}`)
      setRunFor(null); setSelPolicy('')
      await load()
    } catch {
      setMsg('Failed to create job.')
    } finally { setCreating(false) }
  }

  return (
    <div style={s.page}>
      <SectionHeader title="Systems" count={systems.length} />
      {msg && <div style={s.msgBox} onClick={() => setMsg(null)}>{msg} ✕</div>}
      <Card>
        {loading ? <div style={s.load}>Loading…</div> : (
          <Table
            cols={[
              { header:'Hostname',       render:sys=><span style={s.name}>{sys.Hostname}</span> },
              { header:'Status',         render:sys=><StatusBadge status={systemStatus(sys, jobs)} />, width:'110px' },
              { header:'OS',             render:sys=>sys.OS??'—' },
              { header:'Risk',           render:sys=><StatusBadge status={sys.RiskClass||'standard'} />, width:'100px' },
              { header:'Last Backup',    render:sys=>{
                const last = jobs.filter(j=>j.SystemID===sys.ID).sort((a,b)=>new Date(b.CreatedAt).getTime()-new Date(a.CreatedAt).getTime())[0]
                return last
                  ? <><StatusBadge status={last.Status} /><span style={{fontSize:11,color:'var(--text-dim)',marginLeft:6}}>{timeAgo(last.CreatedAt)}</span></>
                  : <span style={{color:'var(--text-dim)'}}>never</span>
              }},
              { header:'Restore Tested', render:_=><span style={{color:'var(--text-dim)',fontSize:12}}>not tested</span> },
              { header:'',               render:sys=>(
                <div style={{display:'flex',gap:6}}>
                  <button onClick={() => { setRunFor(sys); setSelPolicy('') }} style={s.runBtn}>▶ Run</button>
                  <button onClick={() => setDeleteFor(sys)} style={s.delBtn}>🗑</button>
                </div>
              ), width:'110px' },
            ]}
            rows={systems} keyFn={sys=>sys.ID}
            empty="No systems registered. Install the agent to get started."
          />
        )}
      </Card>

      {deleteFor && (
        <ConfirmDialog
          title={`Remove ${deleteFor.Hostname}?`}
          message={`This will delete the system record and revoke all agent tokens for ${deleteFor.Hostname}. The agent process itself will stop authenticating on the next poll. This cannot be undone.`}
          confirmLabel={deleting ? 'Removing…' : 'Remove Agent'}
          danger
          onConfirm={deleteSystem}
          onCancel={() => setDeleteFor(null)}
        />
      )}

      {runFor && (
        <Modal title={`Run Backup — ${runFor.Hostname}`} onClose={() => setRunFor(null)}>
          <div style={s.field}>
            <label style={s.label}>Select Policy</label>
            <select value={selPolicy} onChange={e => setSelPolicy(e.target.value)} style={s.select}>
              <option value="">— select policy —</option>
              {policies.map(p => (
                <option key={p.ID} value={p.ID}>
                  {p.Name} ({p.Engine}){!p.RepositoryID ? ' ⚠ no repository' : ''}
                </option>
              ))}
            </select>
          </div>
          <div style={s.actions}>
            <button onClick={() => setRunFor(null)} style={s.cancelBtn}>Cancel</button>
            <button onClick={runBackup} disabled={creating || !selPolicy} style={s.submitBtn}>
              {creating ? 'Creating…' : '▶ Run Backup'}
            </button>
          </div>
        </Modal>
      )}
    </div>
  )
}

const s: Record<string,React.CSSProperties> = {
  page:      { padding:'28px 36px', maxWidth:1200 },
  load:      { padding:40, color:'var(--text-muted)', textAlign:'center' },
  name:      { fontWeight:600, color:'var(--text)' },
  runBtn:    { padding:'4px 10px', borderRadius:5, background:'var(--accent-dim)', color:'var(--accent)', border:'1px solid rgba(59,130,246,0.3)', fontSize:11, fontWeight:600, cursor:'pointer' },
  delBtn:    { padding:'4px 8px', borderRadius:5, background:'rgba(244,63,94,0.08)', color:'var(--error)', border:'1px solid rgba(244,63,94,0.2)', fontSize:12, cursor:'pointer' },
  msgBox:    { background:'rgba(34,197,94,0.1)', border:'1px solid rgba(34,197,94,0.3)', borderRadius:6, padding:'8px 14px', fontSize:13, color:'var(--success)', marginBottom:12, cursor:'pointer' },
  field:     { marginBottom:16 },
  label:     { display:'block', fontSize:12, fontWeight:600, color:'var(--text-muted)', textTransform:'uppercase' as const, letterSpacing:'0.06em', marginBottom:6 },
  select:    { width:'100%', padding:'8px 12px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, cursor:'pointer' },
  actions:   { display:'flex', gap:8, justifyContent:'flex-end', marginTop:20, paddingTop:16, borderTop:'1px solid var(--border)' },
  cancelBtn: { padding:'7px 16px', borderRadius:6, background:'transparent', border:'1px solid var(--border)', color:'var(--text-muted)', fontSize:13, cursor:'pointer' },
  submitBtn: { padding:'7px 20px', borderRadius:6, background:'var(--success)', color:'#000', border:'none', fontSize:13, fontWeight:700, cursor:'pointer' },
}
