import { useEffect, useState } from 'react'
import { api, timeAgo, type BackupJob, type System } from '../api'
import { Card, SectionHeader } from '../components/Card'
import { StatusBadge } from '../components/StatusBadge'
import { Table } from '../components/Table'

function systemStatus(s: System, jobs: BackupJob[]): string {
  const last = jobs.filter(j => j.SystemID===s.ID).sort((a,b) => new Date(b.CreatedAt).getTime()-new Date(a.CreatedAt).getTime())[0]
  if (!last) return 'unknown'
  return last.Status==='success' ? 'healthy' : last.Status==='failed' ? 'failed' : last.Status
}

export function Systems() {
  const [systems, setSystems] = useState<System[]>([])
  const [jobs,    setJobs]    = useState<BackupJob[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([api.systems(), api.jobs()])
      .then(([s,j]) => { setSystems(s); setJobs(j) })
      .finally(() => setLoading(false))
  }, [])

  return (
    <div style={s.page}>
      <SectionHeader title="Systems" count={systems.length} />
      <Card>
        {loading ? <div style={s.load}>Loading…</div> : (
          <Table
            cols={[
              { header:'Hostname',       render:s=><span style={s.name}>{s.Hostname}</span> },
              { header:'Status',         render:s=><StatusBadge status={systemStatus(s, jobs)} />, width:'110px' },
              { header:'OS',             render:s=>s.OS??'—' },
              { header:'Risk',           render:s=><StatusBadge status={s.RiskClass||'standard'} />, width:'100px' },
              { header:'Last Backup',    render:s=>{
                const last = jobs.filter(j=>j.SystemID===s.ID).sort((a,b)=>new Date(b.CreatedAt).getTime()-new Date(a.CreatedAt).getTime())[0]
                return last ? <><StatusBadge status={last.Status} /> <span style={{fontSize:11, color:'var(--text-dim)', marginLeft:6}}>{timeAgo(last.CreatedAt)}</span></> : <span style={{color:'var(--text-dim)'}}>never</span>
              }},
              { header:'Restore Tested', render:_=><span style={{color:'var(--text-dim)', fontSize:12}}>not tested</span> },
              { header:'Agent Version',  render:s=>s.AgentVersion??<span style={{color:'var(--text-dim)'}}>—</span> },
              { header:'Tags',           render:s=>s.Tags ? Object.entries(s.Tags).map(([k,v])=>(
                <span key={k} style={s.tag}>{k}={v}</span>
              )) : '—' },
            ]}
            rows={systems} keyFn={s=>s.ID}
            empty="No systems registered. Add a system and install the agent to get started."
          />
        )}
      </Card>
    </div>
  )
}

const s: Record<string,React.CSSProperties> = {
  page: { padding:'28px 36px', maxWidth:1200 },
  load: { padding:40, color:'var(--text-muted)', textAlign:'center' },
  name: { fontWeight:600, color:'var(--text)' },
  tag:  { display:'inline-block', background:'rgba(59,130,246,0.08)', color:'var(--text-muted)', padding:'1px 6px', borderRadius:4, fontSize:11, marginRight:4, fontFamily:'var(--font-mono)' },
}
