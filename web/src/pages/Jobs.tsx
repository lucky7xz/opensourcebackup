import { useEffect, useState } from 'react'
import { api, fmt, duration, timeAgo, type BackupJob } from '../api'
import { Card, SectionHeader } from '../components/Card'
import { StatusBadge } from '../components/StatusBadge'
import { Table } from '../components/Table'

const FILTERS = ['all','success','running','pending','failed']

export function Jobs() {
  const [jobs,   setJobs]   = useState<BackupJob[]>([])
  const [filter, setFilter] = useState('all')
  const [loading, setLoading] = useState(true)

  useEffect(() => { api.jobs().then(setJobs).finally(()=>setLoading(false)) }, [])

  const sorted   = [...jobs].sort((a,b)=>new Date(b.CreatedAt).getTime()-new Date(a.CreatedAt).getTime())
  const filtered = filter==='all' ? sorted : sorted.filter(j=>j.Status===filter)

  const counts = FILTERS.reduce((acc,f) => ({
    ...acc, [f]: f==='all' ? jobs.length : jobs.filter(j=>j.Status===f).length
  }), {} as Record<string,number>)

  return (
    <div style={s.page}>
      <SectionHeader title="Jobs" count={jobs.length} />

      <div style={s.filters}>
        {FILTERS.map(f => (
          <button key={f} onClick={()=>setFilter(f)}
            style={{...s.btn, ...(filter===f ? s.btnOn : {})}}>
            {f} <span style={s.cnt}>{counts[f]}</span>
          </button>
        ))}
      </div>

      <Card>
        {loading ? <div style={s.load}>Loading…</div> : (
          <Table
            cols={[
              { header:'Status',    render:j=><StatusBadge status={j.Status} />, width:'110px' },
              { header:'System',    render:j=><span style={s.mono}>{j.SystemID.slice(0,8)}…</span> },
              { header:'Policy',    render:j=><span style={s.mono}>{j.PolicyID.slice(0,8)}…</span> },
              { header:'Size',      render:j=>fmt(j.BytesUploaded) },
              { header:'Duration',  render:j=>duration(j.StartedAt,j.FinishedAt) },
              { header:'Started',   render:j=>j.StartedAt ? new Date(j.StartedAt).toLocaleString() : timeAgo(j.CreatedAt) },
              { header:'Error',     render:j=>j.ErrorSummary
                  ? <span style={{color:'var(--error)',fontSize:12}}>{j.ErrorSummary}</span>
                  : <span style={{color:'var(--text-dim)'}}>—</span> },
            ]}
            rows={filtered} keyFn={j=>j.ID}
            empty={`No ${filter==='all'?'':''+filter+' '}jobs found`}
          />
        )}
      </Card>
    </div>
  )
}

const s: Record<string,React.CSSProperties> = {
  page:    { padding:'28px 36px', maxWidth:1200 },
  load:    { padding:40, color:'var(--text-muted)', textAlign:'center' },
  filters: { display:'flex', gap:6, marginBottom:14 },
  btn:     { padding:'5px 12px', borderRadius:6, border:'1px solid var(--border)', background:'transparent', color:'var(--text-muted)', fontSize:12, cursor:'pointer', fontWeight:500, display:'flex', alignItems:'center', gap:5 },
  btnOn:   { background:'var(--accent-dim)', color:'var(--accent)', borderColor:'rgba(59,130,246,0.3)' },
  cnt:     { background:'rgba(255,255,255,0.06)', borderRadius:10, padding:'0 5px', fontSize:10 },
  mono:    { fontFamily:'var(--font-mono)', fontSize:12, color:'var(--accent)' },
}
