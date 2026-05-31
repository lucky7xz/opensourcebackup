import { useEffect, useRef, useState } from 'react'
import { api, fmt, duration, timeAgo, type BackupJob, type System, type BackupPolicy } from '../api'
import { Card, SectionHeader } from '../components/Card'
import { StatusBadge } from '../components/StatusBadge'
import { Table } from '../components/Table'
import { Modal } from '../components/Modal'
import { ConfirmDialog } from '../components/ConfirmDialog'

const FILTERS = ['all', 'success', 'running', 'pending', 'failed']

export function Jobs() {
  const [jobs,      setJobs]      = useState<BackupJob[]>([])
  const [systems,   setSystems]   = useState<System[]>([])
  const [policies,  setPolicies]  = useState<BackupPolicy[]>([])
  const [filter,    setFilter]    = useState('all')
  const [loading,   setLoading]   = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [deleteJob, setDeleteJob] = useState<BackupJob|null>(null)
  const [selSystem, setSelSystem] = useState('')
  const [selPolicy, setSelPolicy] = useState('')
  const [creating,  setCreating]  = useState(false)
  const [error,     setError]     = useState<string|null>(null)
  const [selected,  setSelected]  = useState<BackupJob|null>(null)
  const timerRef = useRef<ReturnType<typeof setInterval>|null>(null)

  const load = () => Promise.all([api.jobs(), api.systems(), api.policies()])
    .then(([j,s,p]) => { setJobs(j); setSystems(s); setPolicies(p) })
    .finally(() => setLoading(false))

  useEffect(() => { load() }, [])

  // Auto-refresh every 3s when a running/pending job is visible
  useEffect(() => {
    const hasActive = jobs.some(j => j.Status === 'running' || j.Status === 'pending')
    if (hasActive) {
      timerRef.current = setInterval(() => {
        load()
        // Update selected job if open
        if (selected) {
          api.jobs().then(all => {
            const updated = all.find(j => j.ID === selected.ID)
            if (updated) setSelected(updated)
          })
        }
      }, 3000)
    }
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [jobs, selected])

  async function createJob() {
    if (!selSystem || !selPolicy) { setError('Please select a system and policy.'); return }
    setCreating(true); setError(null)
    try {
      await api.createJob(selSystem, selPolicy)
      setShowModal(false); setSelSystem(''); setSelPolicy('')
      await load()
    } catch { setError('Failed to create job.') }
    finally { setCreating(false) }
  }

  const sorted   = [...jobs].sort((a,b) => new Date(b.CreatedAt).getTime()-new Date(a.CreatedAt).getTime())
  const filtered = filter==='all' ? sorted : sorted.filter(j => j.Status===filter)
  const counts   = FILTERS.reduce((acc,f) => ({
    ...acc, [f]: f==='all' ? jobs.length : jobs.filter(j => j.Status===f).length
  }), {} as Record<string,number>)

  const hostname   = (id: string) => systems.find(s => s.ID===id)?.Hostname ?? id.slice(0,8)+'…'
  const policyName = (id: string) => policies.find(p => p.ID===id)?.Name    ?? id.slice(0,8)+'…'

  return (
    <div style={s.root}>
      {/* ── Left: Job List ── */}
      <div style={s.left}>
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
                { header:'When',     render:j => j.StartedAt ? new Date(j.StartedAt).toLocaleString() : timeAgo(j.CreatedAt) },
                { header:'',        render:j => (
                  <div style={{display:'flex',gap:4}}>
                    <button onClick={() => setSelected(j)} style={{...s.detailBtn, ...(selected?.ID===j.ID?s.detailBtnOn:{})}}>
                      {selected?.ID===j.ID ? '▶' : '⬡'}
                    </button>
                    {(j.Status==='pending'||j.Status==='failed') && (
                      <button onClick={e=>{e.stopPropagation();setDeleteJob(j)}} style={s.delBtn}>🗑</button>
                    )}
                  </div>
                ), width:'70px' },
              ]}
              rows={filtered} keyFn={j => j.ID}
              empty={`No ${filter==='all' ? '' : filter+' '}jobs found`}
            />
          )}
        </Card>
      </div>

      {/* ── Right: Job Detail Panel ── */}
      {selected && (
        <div style={s.detail}>
          <div style={s.detailHeader}>
            <span style={s.detailTitle}>Job Details</span>
            <button onClick={() => setSelected(null)} style={s.closeBtn}>✕</button>
          </div>

          <div style={s.detailBody}>
            <div style={s.detailRow}>
              <span style={s.dk}>Status</span>
              <StatusBadge status={selected.Status} />
              {(selected.Status==='running'||selected.Status==='pending') && (
                <span style={s.pulse}>⟳ live</span>
              )}
            </div>
            <div style={s.detailRow}>
              <span style={s.dk}>System</span>
              <span style={s.dv}>{hostname(selected.SystemID)}</span>
            </div>
            <div style={s.detailRow}>
              <span style={s.dk}>Policy</span>
              <span style={s.dv}>{policyName(selected.PolicyID)}</span>
            </div>
            <div style={s.detailRow}>
              <span style={s.dk}>Started</span>
              <span style={s.dv}>{selected.StartedAt ? new Date(selected.StartedAt).toLocaleString() : '—'}</span>
            </div>
            <div style={s.detailRow}>
              <span style={s.dk}>Finished</span>
              <span style={s.dv}>{selected.FinishedAt ? new Date(selected.FinishedAt).toLocaleString() : '—'}</span>
            </div>
            <div style={s.detailRow}>
              <span style={s.dk}>Duration</span>
              <span style={s.dv}>{duration(selected.StartedAt, selected.FinishedAt)}</span>
            </div>
            <div style={s.detailRow}>
              <span style={s.dk}>Size</span>
              <span style={s.dv}>{fmt(selected.BytesUploaded)}</span>
            </div>
            {selected.ErrorSummary && (
              <div style={s.errBox}>
                <div style={s.errTitle}>Error</div>
                <div style={s.errMsg}>{selected.ErrorSummary}</div>
              </div>
            )}

            {/* Progress indicator for active jobs */}
            {(selected.Status==='running'||selected.Status==='pending') && (
              <div style={s.progressBox}>
                <div style={s.progressBar}>
                  <div style={{...s.progressFill, animation:'progress 1.5s ease-in-out infinite'}} />
                </div>
                <div style={s.progressText}>
                  {selected.Status==='pending' ? 'Waiting for agent…' : 'Backup running…'}
                </div>
                <div style={s.progressSub}>Auto-refreshing every 3s</div>
              </div>
            )}

            {selected.Status==='success' && (
              <div style={s.successBox}>
                ✓ Backup completed successfully
              </div>
            )}

            <div style={s.idRow}>
              <span style={s.dk}>Job ID</span>
              <span style={s.mono}>{selected.ID.slice(0,16)}…</span>
            </div>
          </div>
        </div>
      )}

      <style>{`
        @keyframes progress {
          0%   { transform: translateX(-100%) }
          100% { transform: translateX(400%) }
        }
        @keyframes blink {
          0%,100% { opacity:1 } 50% { opacity:0.3 }
        }
      `}</style>

      {deleteJob && (
        <ConfirmDialog
          title="Delete Job?"
          message={`Delete this ${deleteJob.Status} job for ${hostname(deleteJob.SystemID)}?`}
          confirmLabel="Delete Job" danger
          onConfirm={async () => { await api.deleteJob(deleteJob.ID); setDeleteJob(null); await load() }}
          onCancel={() => setDeleteJob(null)}
        />
      )}

      {showModal && (
        <Modal title="Run Backup Job" onClose={() => { setShowModal(false); setError(null) }}>
          <div style={s.field}>
            <label style={s.label}>System</label>
            <select value={selSystem} onChange={e => setSelSystem(e.target.value)} style={s.select}>
              <option value="">— select system —</option>
              {systems.map(sys => (<option key={sys.ID} value={sys.ID}>{sys.Hostname}</option>))}
            </select>
          </div>
          <div style={s.field}>
            <label style={s.label}>Policy</label>
            <select value={selPolicy} onChange={e => setSelPolicy(e.target.value)} style={s.select}>
              <option value="">— select policy —</option>
              {policies.map(p => (<option key={p.ID} value={p.ID}>{p.Name} ({p.Engine})</option>))}
            </select>
          </div>
          {error && <div style={s.errBox2}>{error}</div>}
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
  root:        { display:'flex', gap:0, padding:'28px 36px', maxWidth:1400, alignItems:'flex-start' },
  left:        { flex:1, minWidth:0 },
  topRow:      { display:'flex', justifyContent:'space-between', alignItems:'flex-start', marginBottom:4 },
  load:        { padding:40, color:'var(--text-muted)', textAlign:'center' },
  filters:     { display:'flex', gap:6, marginBottom:14 },
  btn:         { padding:'5px 12px', borderRadius:6, border:'1px solid var(--border)', background:'transparent', color:'var(--text-muted)', fontSize:12, cursor:'pointer', fontWeight:500, display:'flex', alignItems:'center', gap:5 },
  btnOn:       { background:'var(--accent-dim)', color:'var(--accent)', borderColor:'rgba(59,130,246,0.3)' },
  cnt:         { background:'rgba(255,255,255,0.06)', borderRadius:10, padding:'0 5px', fontSize:10 },
  newBtn:      { padding:'7px 16px', borderRadius:6, background:'var(--accent)', color:'#fff', border:'none', fontSize:13, fontWeight:600, cursor:'pointer' },
  strong:      { fontWeight:600, color:'var(--text)' },
  muted:       { color:'var(--text-muted)' },
  err:         { color:'var(--error)', fontSize:12 },
  dim:         { color:'var(--text-dim)' },
  detailBtn:   { padding:'3px 8px', borderRadius:5, background:'var(--accent-dim)', color:'var(--accent)', border:'1px solid rgba(59,130,246,0.3)', fontSize:12, cursor:'pointer' },
  detailBtnOn: { background:'var(--accent)', color:'#fff' },
  delBtn:      { padding:'3px 8px', borderRadius:5, background:'rgba(244,63,94,0.08)', color:'var(--error)', border:'1px solid rgba(244,63,94,0.2)', fontSize:12, cursor:'pointer' },
  // Detail panel
  detail:      { width:320, minWidth:320, background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:10, marginLeft:16, position:'sticky' as const, top:28, overflow:'hidden' },
  detailHeader:{ display:'flex', justifyContent:'space-between', alignItems:'center', padding:'14px 16px', borderBottom:'1px solid var(--border)', background:'rgba(255,255,255,0.02)' },
  detailTitle: { fontSize:13, fontWeight:700, color:'var(--text)', textTransform:'uppercase' as const, letterSpacing:'0.06em' },
  closeBtn:    { background:'none', border:'none', color:'var(--text-muted)', fontSize:16, cursor:'pointer', padding:'2px 6px' },
  detailBody:  { padding:'16px' },
  detailRow:   { display:'flex', alignItems:'center', gap:8, marginBottom:10, flexWrap:'wrap' as const },
  dk:          { fontSize:11, fontWeight:700, color:'var(--text-dim)', textTransform:'uppercase' as const, letterSpacing:'0.06em', width:68, flexShrink:0 },
  dv:          { fontSize:13, color:'var(--text)' },
  mono:        { fontFamily:'var(--font-mono)', fontSize:11, color:'var(--accent)' },
  pulse:       { fontSize:11, color:'var(--running)', animation:'blink 1s infinite', marginLeft:6 },
  progressBox: { background:'rgba(96,165,250,0.06)', border:'1px solid rgba(96,165,250,0.2)', borderRadius:8, padding:'12px', marginTop:12 },
  progressBar: { height:3, background:'var(--border)', borderRadius:2, overflow:'hidden', marginBottom:8 },
  progressFill:{ height:'100%', width:'25%', background:'var(--running)', borderRadius:2 },
  progressText:{ fontSize:13, color:'var(--running)', fontWeight:600 },
  progressSub: { fontSize:11, color:'var(--text-dim)', marginTop:3 },
  successBox:  { background:'rgba(34,197,94,0.08)', border:'1px solid rgba(34,197,94,0.2)', borderRadius:8, padding:'10px 12px', fontSize:13, color:'var(--success)', fontWeight:600, marginTop:12 },
  errBox:      { background:'rgba(244,63,94,0.08)', border:'1px solid rgba(244,63,94,0.2)', borderRadius:8, padding:'10px 12px', marginTop:8 },
  errTitle:    { fontSize:11, fontWeight:700, color:'var(--error)', textTransform:'uppercase' as const, letterSpacing:'0.06em', marginBottom:4 },
  errMsg:      { fontSize:12, color:'var(--error)', lineHeight:1.5 },
  idRow:       { marginTop:12, paddingTop:10, borderTop:'1px solid var(--border)', display:'flex', alignItems:'center', gap:8 },
  // Modal
  field:       { marginBottom:16 },
  label:       { display:'block', fontSize:11, fontWeight:700, color:'var(--text-dim)', textTransform:'uppercase' as const, letterSpacing:'0.08em', marginBottom:6 },
  select:      { width:'100%', padding:'8px 11px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, cursor:'pointer' },
  errBox2:     { background:'rgba(244,63,94,0.1)', border:'1px solid rgba(244,63,94,0.25)', borderRadius:6, padding:'8px 12px', fontSize:13, color:'var(--error)', marginBottom:8 },
  actions:     { display:'flex', gap:8, justifyContent:'flex-end', paddingTop:16, borderTop:'1px solid var(--border)' },
  cancelBtn:   { padding:'7px 16px', borderRadius:6, background:'transparent', border:'1px solid var(--border)', color:'var(--text-muted)', fontSize:13, cursor:'pointer' },
  submitBtn:   { padding:'7px 20px', borderRadius:6, background:'var(--success)', color:'#000', border:'none', fontSize:13, fontWeight:700, cursor:'pointer' },
}
