import { useEffect, useState } from 'react'
import { api, fmt, duration, timeAgo, type BackupJob, type System, type BackupPolicy } from '../api'
import { Card, SectionHeader } from '../components/Card'
import { StatusBadge } from '../components/StatusBadge'
import { Table } from '../components/Table'
import { Modal } from '../components/Modal'
import { ConfirmDialog } from '../components/ConfirmDialog'

const FILTERS = ['all', 'success', 'running', 'pending', 'failed']

export function Jobs() {
  const [jobs,     setJobs]     = useState<BackupJob[]>([])
  const [systems,  setSystems]  = useState<System[]>([])
  const [policies, setPolicies] = useState<BackupPolicy[]>([])
  const [filter,   setFilter]   = useState('all')
  const [loading,  setLoading]  = useState(true)
  const [showModal,  setShowModal]  = useState(false)
  const [deleteJob,  setDeleteJob]  = useState<BackupJob|null>(null)
  const [selSystem,  setSelSystem]  = useState('')
  const [selPolicy,  setSelPolicy]  = useState('')
  const [creating,   setCreating]   = useState(false)
  const [error,      setError]      = useState<string|null>(null)

  const load = () => Promise.all([api.jobs(), api.systems(), api.policies()])
    .then(([j,s,p]) => { setJobs(j); setSystems(s); setPolicies(p) })
    .finally(() => setLoading(false))

  useEffect(() => { load() }, [])

  async function createJob() {
    if (!selSystem || !selPolicy) { setError('Please select a system and policy.'); return }
    setCreating(true); setError(null)
    try {
      await api.createJob(selSystem, selPolicy)
      setShowModal(false); setSelSystem(''); setSelPolicy('')
      await load()
    } catch {
      setError('Failed to create job. Check that system and policy exist.')
    } finally { setCreating(false) }
  }

  const sorted   = [...jobs].sort((a,b) => new Date(b.CreatedAt).getTime()-new Date(a.CreatedAt).getTime())
  const filtered = filter==='all' ? sorted : sorted.filter(j => j.Status===filter)
  const counts   = FILTERS.reduce((acc,f) => ({
    ...acc, [f]: f==='all' ? jobs.length : jobs.filter(j => j.Status===f).length
  }), {} as Record<string,number>)

  const hostname   = (id: string) => systems.find(s  => s.ID===id)?.Hostname ?? id.slice(0,8)+'…'
  const policyName = (id: string) => policies.find(p => p.ID===id)?.Name    ?? id.slice(0,8)+'…'

  return (
    <div style={s.page}>
      <div style={s.topRow}>
        <SectionHeader title="Jobs" count={jobs.length} />
        <button onClick={() => setShowModal(true)} style={s.newBtn}>+ New Job</button>
      </div>

      <div style={s.filters}>
        {FILTERS.map(f => (
          <button key={f} onClick={() => setFilter(f)}
            style={{...s.btn, ...(filter===f ? s.btnOn : {})}}>
            {f} <span style={s.cnt}>{counts[f]}</span>
          </button>
        ))}
      </div>

      <Card>
        {loading ? <div style={s.load}>Loading…</div> : (
          <Table
            cols={[
              { header:'Status',   render:j => <StatusBadge status={j.Status} />, width:'110px' },
              { header:'System',   render:j => <span style={s.strong}>{hostname(j.SystemID)}</span> },
              { header:'Policy',   render:j => <span style={s.muted}>{policyName(j.PolicyID)}</span> },
              { header:'Size',     render:j => fmt(j.BytesUploaded) },
              { header:'Duration', render:j => duration(j.StartedAt, j.FinishedAt) },
              { header:'Started',  render:j => j.StartedAt ? new Date(j.StartedAt).toLocaleString() : timeAgo(j.CreatedAt) },
              { header:'Error',    render:j => j.ErrorSummary
                  ? <span style={s.err}>{j.ErrorSummary}</span>
                  : <span style={s.dim}>—</span> },
              { header:'',        render:j => (j.Status==='pending'||j.Status==='failed')
                  ? <button onClick={() => setDeleteJob(j)} style={s.delBtn}>🗑</button>
                  : null, width:'40px' },
            ]}
            rows={filtered} keyFn={j => j.ID}
            empty={`No ${filter==='all' ? '' : filter+' '}jobs found`}
          />
        )}
      </Card>

      {deleteJob && (
        <ConfirmDialog
          title="Delete Job?"
          message={`Delete this ${deleteJob.Status} job for ${hostname(deleteJob.SystemID)}? The agent will no longer pick it up.`}
          confirmLabel="Delete Job"
          danger
          onConfirm={async () => {
            await api.deleteJob(deleteJob.ID)
            setDeleteJob(null)
            await load()
          }}
          onCancel={() => setDeleteJob(null)}
        />
      )}

      {showModal && (
        <Modal title="Run Backup Job" onClose={() => { setShowModal(false); setError(null) }}>
          <div style={s.field}>
            <label style={s.label}>System</label>
            <select value={selSystem} onChange={e => setSelSystem(e.target.value)} style={s.select}>
              <option value="">— select system —</option>
              {systems.map(sys => (
                <option key={sys.ID} value={sys.ID}>{sys.Hostname}</option>
              ))}
            </select>
          </div>
          <div style={s.field}>
            <label style={s.label}>Policy</label>
            <select value={selPolicy} onChange={e => setSelPolicy(e.target.value)} style={s.select}>
              <option value="">— select policy —</option>
              {policies.map(p => (
                <option key={p.ID} value={p.ID}>{p.Name} ({p.Engine})</option>
              ))}
            </select>
          </div>
          {error && <div style={s.errBox}>{error}</div>}
          <div style={s.actions}>
            <button onClick={() => { setShowModal(false); setError(null) }} style={s.cancelBtn}>Cancel</button>
            <button onClick={createJob} disabled={creating} style={s.submitBtn}>
              {creating ? 'Creating…' : '▶ Run Backup'}
            </button>
          </div>
        </Modal>
      )}
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  page:      { padding:'28px 36px', maxWidth:1200 },
  topRow:    { display:'flex', justifyContent:'space-between', alignItems:'flex-start', marginBottom:4 },
  load:      { padding:40, color:'var(--text-muted)', textAlign:'center' },
  filters:   { display:'flex', gap:6, marginBottom:14 },
  btn:       { padding:'5px 12px', borderRadius:6, border:'1px solid var(--border)', background:'transparent', color:'var(--text-muted)', fontSize:12, cursor:'pointer', fontWeight:500, display:'flex', alignItems:'center', gap:5 },
  btnOn:     { background:'var(--accent-dim)', color:'var(--accent)', borderColor:'rgba(59,130,246,0.3)' },
  cnt:       { background:'rgba(255,255,255,0.06)', borderRadius:10, padding:'0 5px', fontSize:10 },
  newBtn:    { padding:'7px 16px', borderRadius:6, background:'var(--accent)', color:'#fff', border:'none', fontSize:13, fontWeight:600, cursor:'pointer' },
  strong:    { fontWeight:600, color:'var(--text)' },
  muted:     { color:'var(--text-muted)' },
  err:       { color:'var(--error)', fontSize:12 },
  dim:       { color:'var(--text-dim)' },
  field:     { marginBottom:16 },
  label:     { display:'block', fontSize:12, fontWeight:600, color:'var(--text-muted)', textTransform:'uppercase' as const, letterSpacing:'0.06em', marginBottom:6 },
  select:    { width:'100%', padding:'8px 12px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, cursor:'pointer' },
  errBox:    { background:'rgba(244,63,94,0.1)', border:'1px solid rgba(244,63,94,0.3)', borderRadius:6, padding:'8px 12px', fontSize:13, color:'var(--error)', marginBottom:12 },
  actions:   { display:'flex', gap:8, justifyContent:'flex-end', marginTop:20, paddingTop:16, borderTop:'1px solid var(--border)' },
  cancelBtn: { padding:'7px 16px', borderRadius:6, background:'transparent', border:'1px solid var(--border)', color:'var(--text-muted)', fontSize:13, cursor:'pointer' },
  submitBtn: { padding:'7px 20px', borderRadius:6, background:'var(--success)', color:'#000', border:'none', fontSize:13, fontWeight:700, cursor:'pointer' },
  delBtn:    { padding:'3px 8px', borderRadius:5, background:'rgba(244,63,94,0.08)', color:'var(--error)', border:'1px solid rgba(244,63,94,0.2)', fontSize:12, cursor:'pointer' },
}
