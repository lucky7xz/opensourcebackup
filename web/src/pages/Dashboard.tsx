import { useEffect, useState } from 'react'
import { api, fmt, timeAgo, duration, type BackupJob, type RestoreTest, type Snapshot, type System } from '../api'
import { HealthCard, SectionHeader, Card } from '../components/Card'
import { StatusBadge } from '../components/StatusBadge'
import { Table } from '../components/Table'

export function Dashboard() {
  const [systems,      setSystems]      = useState<System[]>([])
  const [jobs,         setJobs]         = useState<BackupJob[]>([])
  const [snapshots,    setSnapshots]    = useState<Snapshot[]>([])
  const [restoreTests, setRestoreTests] = useState<RestoreTest[]>([])
  const [loading,      setLoading]      = useState(true)

  useEffect(() => {
    Promise.all([api.systems(), api.jobs(), api.snapshots(), api.restoreTests()])
      .then(([s,j,sn,rt]) => { setSystems(s); setJobs(j); setSnapshots(sn); setRestoreTests(rt) })
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div style={s.loading}>Loading…</div>

  const success   = jobs.filter(j => j.Status==='success').length
  const failed    = jobs.filter(j => j.Status==='failed').length
  const running   = jobs.filter(j => j.Status==='running'||j.Status==='pending').length
  const totalBytes = jobs.reduce((a,j) => a+(j.BytesUploaded??0), 0)
  // Restore Tested: % of snapshots with at least one successful restore test
  const testedCount  = snapshots.filter(sn =>
    restoreTests.some(rt => rt.SnapshotID===sn.ID && rt.Status==='success')
  ).length
  const restoreTested = snapshots.length > 0 ? Math.round((testedCount/snapshots.length)*100) : 0

  const recentFailed = jobs
    .filter(j => j.Status==='failed')
    .sort((a,b) => new Date(b.CreatedAt).getTime()-new Date(a.CreatedAt).getTime())
    .slice(0,5)

  const recentJobs = [...jobs]
    .sort((a,b) => new Date(b.CreatedAt).getTime()-new Date(a.CreatedAt).getTime())
    .slice(0,8)

  const recentSnapshots = [...snapshots]
    .sort((a,b) => new Date(b.CreatedAt).getTime()-new Date(a.CreatedAt).getTime())
    .slice(0,5)

  return (
    <div style={s.page}>
      {/* Header */}
      <div style={s.header}>
        <div>
          <h1 style={s.h1}>Backup Health</h1>
          <p style={s.sub}>Are your systems protected — and has a restore been successfully tested?</p>
        </div>
      </div>

      {/* Health Cards */}
      <div style={s.grid6}>
        <HealthCard label="Protected Systems"   value={systems.length}  color="var(--text)" />
        <HealthCard label="Successful Backups"  value={success}         color="var(--success)" sub={`of ${jobs.length} total`} />
        <HealthCard label="Failed Jobs"         value={failed}          color={failed>0?'var(--error)':'var(--text-muted)'} warn={failed>0} />
        <HealthCard label="Active"              value={running}         color={running>0?'var(--running)':'var(--text-muted)'} sub="running or pending" />
        <HealthCard label="Restore Tested"      value={`${restoreTested}%`} color={restoreTested>0?'var(--success)':'var(--warning)'} sub="snapshots verified" warn={restoreTested===0} />
        <HealthCard label="Storage Used"        value={fmt(totalBytes)} color="var(--text-muted)" />
      </div>

      {/* Restore notice if 0% */}
      {restoreTested === 0 && (
        <div style={s.notice}>
          <span style={{color:'var(--warning)', fontWeight:600}}>Restore not tested</span>
          {' — '}No snapshots have been verified through a restore test. Configure restore tests in B13.
        </div>
      )}

      <div style={s.two}>
        {/* Recent Failures */}
        <div>
          <SectionHeader title="Recent Failures" count={recentFailed.length} />
          <Card>
            <Table
              cols={[
                { header:'System',  render:j=><span style={s.mono}>{j.SystemID.slice(0,8)}…</span> },
                { header:'Status',  render:j=><StatusBadge status={j.Status} />, width:'100px' },
                { header:'Error',   render:j=><span style={{color:'var(--error)',fontSize:12}}>{j.ErrorSummary||'—'}</span> },
                { header:'When',    render:j=>timeAgo(j.CreatedAt) },
              ]}
              rows={recentFailed} keyFn={j=>j.ID}
              empty="No failures — all recent jobs succeeded"
            />
          </Card>
        </div>

        {/* Recent Snapshots */}
        <div>
          <SectionHeader title="Recent Snapshots" count={snapshots.length} />
          <Card>
            <Table
              cols={[
                { header:'Snapshot',   render:sn=><span style={s.mono}>{sn.EngineSnapshotID.slice(0,10)}…</span> },
                { header:'Restore',    render:_=><StatusBadge status="not tested" />, width:'110px' },
                { header:'Size',       render:s=>{ const j=jobs.find(j=>j.ID===s.JobID); return fmt(j?.BytesUploaded) }},
                { header:'Created',    render:s=>timeAgo(s.CreatedAt) },
              ]}
              rows={recentSnapshots} keyFn={s=>s.ID}
              empty="No snapshots yet"
            />
          </Card>
        </div>
      </div>

      {/* All recent jobs */}
      <div style={{marginTop:28}}>
        <SectionHeader title="Recent Jobs" count={jobs.length} />
        <Card>
          <Table
            cols={[
              { header:'Status',   render:j=><StatusBadge status={j.Status} />, width:'100px' },
              { header:'System',   render:j=><span style={s.mono}>{j.SystemID.slice(0,8)}…</span> },
              { header:'Policy',   render:j=><span style={s.mono}>{j.PolicyID.slice(0,8)}…</span> },
              { header:'Size',     render:j=>fmt(j.BytesUploaded) },
              { header:'Duration', render:j=>duration(j.StartedAt,j.FinishedAt) },
              { header:'When',     render:j=>timeAgo(j.CreatedAt) },
            ]}
            rows={recentJobs} keyFn={j=>j.ID}
            empty="No jobs yet"
          />
        </Card>
      </div>
    </div>
  )
}

const s: Record<string,React.CSSProperties> = {
  page:    { padding:'28px 36px', maxWidth:1200 },
  loading: { padding:40, color:'var(--text-muted)' },
  header:  { marginBottom:24 },
  h1:      { fontSize:22, fontWeight:700, color:'var(--text)' },
  sub:     { fontSize:13, color:'var(--text-muted)', marginTop:4, fontStyle:'italic' },
  grid6:   { display:'grid', gridTemplateColumns:'repeat(6,1fr)', gap:12, marginBottom:16 },
  notice:  { background:'rgba(245,158,11,0.08)', border:'1px solid rgba(245,158,11,0.25)', borderRadius:'var(--radius)', padding:'10px 16px', fontSize:13, color:'var(--text-muted)', marginBottom:24 },
  two:     { display:'grid', gridTemplateColumns:'1fr 1fr', gap:20, marginTop:28 },
  mono:    { fontFamily:'var(--font-mono)', fontSize:12, color:'var(--accent)' },
}
