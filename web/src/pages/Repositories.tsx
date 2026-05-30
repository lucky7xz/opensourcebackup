import { useEffect, useState } from 'react'
import { api, type BackupRepository } from '../api'
import { Card, SectionHeader } from '../components/Card'
import { Table } from '../components/Table'

export function Repositories() {
  const [repos, setRepos] = useState<BackupRepository[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => { api.repositories().then(setRepos).finally(()=>setLoading(false)) }, [])

  return (
    <div style={s.page}>
      <SectionHeader title="Repositories" count={repos.length} />
      <Card>
        {loading ? <div style={s.load}>Loading…</div> : (
          <Table
            cols={[
              { header:'Type',       render:r=><span style={s.badge}>{r.Type}</span>, width:'80px' },
              { header:'Location',   render:r=><span style={s.mono}>{r.Location}</span> },
              { header:'Encryption', render:r=>r.EncryptionMode??'—', width:'100px' },
              { header:'WORM Lock',  render:r=>r.ObjectLockEnabled
                  ? <span style={{color:'var(--success)',fontSize:12}}>✓ enabled</span>
                  : <span style={{color:'var(--text-dim)',fontSize:12}}>—</span>, width:'90px' },
              { header:'ID',         render:r=><span style={s.mono}>{r.ID.slice(0,8)}…</span> },
              { header:'Created',    render:r=>new Date(r.CreatedAt).toLocaleDateString() },
            ]}
            rows={repos} keyFn={r=>r.ID}
            empty="No repositories configured"
          />
        )}
      </Card>
    </div>
  )
}

const s: Record<string,React.CSSProperties> = {
  page:  { padding:'28px 36px', maxWidth:1100 },
  load:  { padding:40, color:'var(--text-muted)', textAlign:'center' },
  mono:  { fontFamily:'var(--font-mono)', fontSize:12, color:'var(--accent)' },
  badge: { display:'inline-block', padding:'2px 8px', borderRadius:4, background:'rgba(59,130,246,0.1)', color:'var(--accent)', fontSize:11, fontWeight:600 },
}
