import { useEffect, useState } from 'react'
import { api, type BackupPolicy } from '../api'
import { Card, SectionHeader } from '../components/Card'
import { Table } from '../components/Table'

export function Policies() {
  const [policies, setPolicies] = useState<BackupPolicy[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => { api.policies().then(setPolicies).finally(()=>setLoading(false)) }, [])

  return (
    <div style={s.page}>
      <SectionHeader title="Policies" count={policies.length} />
      <Card>
        {loading ? <div style={s.load}>Loading…</div> : (
          <Table
            cols={[
              { header:'Name',       render:p=><span style={s.name}>{p.Name}</span> },
              { header:'Engine',     render:p=><span style={s.badge}>{p.Engine}</span>, width:'90px' },
              { header:'Schedule',   render:p=>p.Schedule?<span style={s.mono}>{p.Schedule}</span>:<span style={s.dim}>manual</span> },
              { header:'Includes',   render:p=><div>{(p.Includes??[]).map(i=><div key={i} style={s.path}>{i}</div>)}</div> },
              { header:'Repository', render:p=>p.RepositoryID
                  ? <span style={s.mono}>{p.RepositoryID.slice(0,8)}…</span>
                  : <span style={s.warn}>Policy has no repository configured</span> },
              { header:'Created',    render:p=>new Date(p.CreatedAt).toLocaleDateString() },
            ]}
            rows={policies} keyFn={p=>p.ID}
            empty="No policies yet"
          />
        )}
      </Card>
    </div>
  )
}

const s: Record<string,React.CSSProperties> = {
  page:  { padding:'28px 36px', maxWidth:1200 },
  load:  { padding:40, color:'var(--text-muted)', textAlign:'center' },
  name:  { fontWeight:600, color:'var(--text)' },
  mono:  { fontFamily:'var(--font-mono)', fontSize:12, color:'var(--accent)' },
  badge: { display:'inline-block', padding:'2px 8px', borderRadius:4, background:'rgba(59,130,246,0.1)', color:'var(--accent)', fontSize:11, fontWeight:600 },
  path:  { fontFamily:'var(--font-mono)', fontSize:11, color:'var(--text-muted)' },
  dim:   { color:'var(--text-dim)', fontSize:12 },
  warn:  { color:'var(--warning)', fontSize:12 },
}
