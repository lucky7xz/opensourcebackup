import { useEffect, useState } from 'react'
import { api, type BackupPolicy, type BackupRepository } from '../api'
import { Card, SectionHeader } from '../components/Card'
import { Table } from '../components/Table'
import { Modal } from '../components/Modal'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { ScheduleBuilder, EMPTY_SCHEDULE, type ScheduleConfig } from '../components/ScheduleBuilder'

// ── Engine info ────────────────────────────────────────────────────────────────

const ENGINES = [
  {
    value: 'restic',
    label: 'Restic',
    desc: 'Files & folders — Windows, Linux, NAS, S3',
    backupMode: 'Incremental deduplicated snapshots — only changed data blocks are stored after the initial backup.',
    supports: ['windows', 'linux', 'freebsd', 'nas', 's3', 'sftp'],
  },
  {
    value: 'borg',
    label: 'Borg',
    desc: 'Linux servers via SSH — very efficient deduplication',
    backupMode: 'Incremental deduplicated archives — Borg uses content-defined chunking for high efficiency.',
    supports: ['linux', 'ssh'],
  },
  {
    value: 'pgbackrest',
    label: 'pgBackRest',
    desc: 'PostgreSQL databases — WAL archiving & Point-in-Time Recovery',
    backupMode: 'Full, Differential, or Incremental — pgBackRest manages the backup type automatically based on retention.',
    supports: ['postgresql'],
  },
  {
    value: 'velero',
    label: 'Velero',
    desc: 'Kubernetes clusters — Deployments, Volumes & ConfigMaps',
    backupMode: 'Snapshot-based or file-copy depending on the storage provider configuration.',
    supports: ['kubernetes'],
  },
]

export function Policies() {
  const [policies,  setPolicies]  = useState<BackupPolicy[]>([])
  const [repos,     setRepos]     = useState<BackupRepository[]>([])
  const [loading,   setLoading]   = useState(true)
  const [showForm,  setShowForm]  = useState(false)
  const [editingId, setEditingId] = useState<string|null>(null)
  const [deleteFor, setDeleteFor] = useState<BackupPolicy|null>(null)
  const [saving,    setSaving]    = useState(false)
  const [err,       setErr]       = useState<string|null>(null)
  const [tab,       setTab]       = useState<'general'|'schedule'|'retention'>('general')

  // form state
  const [name,     setName]     = useState('')
  const [engine,   setEngine]   = useState('restic')
  const [repoID,   setRepoID]   = useState('')
  const [includes, setIncludes] = useState<string[]>([''])
  const [excludes, setExcludes] = useState<string[]>([])
  const [schedule, setSchedule] = useState<ScheduleConfig>({ ...EMPTY_SCHEDULE })
  // retention
  const [keepLast,    setKeepLast]    = useState('7')
  const [keepDaily,   setKeepDaily]   = useState('14')
  const [keepWeekly,  setKeepWeekly]  = useState('8')
  const [keepMonthly, setKeepMonthly] = useState('12')
  const [keepYearly,  setKeepYearly]  = useState('3')

  const load = () => Promise.all([api.policies(), api.repositories()])
    .then(([p, r]) => { setPolicies(p); setRepos(r) })
    .finally(() => setLoading(false))

  useEffect(() => { load() }, [])

  function resetForm() {
    setName(''); setEngine('restic'); setRepoID('')
    setIncludes(['']); setExcludes([])
    setSchedule({ ...EMPTY_SCHEDULE })
    setKeepLast('7'); setKeepDaily('14'); setKeepWeekly('8')
    setKeepMonthly('12'); setKeepYearly('3')
    setErr(null); setTab('general')
  }

  function openEdit(p: BackupPolicy) {
    setEditingId(p.ID)
    setName(p.Name)
    setEngine(p.Engine)
    setRepoID(p.RepositoryID ?? '')
    setIncludes(p.Includes?.length ? p.Includes : [''])
    setExcludes(p.Excludes ?? [])
    const sc = p.ScheduleConfig ?? {}
    setSchedule({
      cron:              sc.cron              ?? p.Schedule ?? '',
      timezone:          sc.timezone          ?? 'Europe/Berlin',
      window_start:      sc.window_start      ?? '',
      window_end:        sc.window_end        ?? '',
      if_missed:         sc.if_missed         ?? 'run_asap',
      restore_test_cron: sc.restore_test_cron ?? '',
      retention_cron:    sc.retention_cron    ?? '',
    })
    setKeepLast(String(p.RetentionPlan?.KeepLast    ?? 7))
    setKeepDaily(String(p.RetentionPlan?.KeepDaily   ?? 14))
    setKeepWeekly(String(p.RetentionPlan?.KeepWeekly ?? 8))
    setKeepMonthly(String(p.RetentionPlan?.KeepMonthly ?? 12))
    setKeepYearly(String(p.RetentionPlan?.KeepYearly  ?? 3))
    setShowForm(true); setTab('general')
  }

  async function save() {
    if (!name.trim()) { setErr('Policy name is required.'); setTab('general'); return }
    const cleanIncludes = includes.map(p => p.trim()).filter(Boolean)
    if (cleanIncludes.length === 0) { setErr('At least one include path is required.'); setTab('general'); return }
    setSaving(true); setErr(null)
    try {
      const data = {
        Name:         name.trim(),
        Engine:       engine,
        RepositoryID: repoID || undefined,
        Schedule:     schedule.cron || undefined,
        ScheduleConfig: schedule,
        Includes:     cleanIncludes,
        Excludes:     excludes.map(p => p.trim()).filter(Boolean),
        RetentionPlan: {
          KeepLast:    Number(keepLast)    || 0,
          KeepDaily:   Number(keepDaily)   || 0,
          KeepWeekly:  Number(keepWeekly)  || 0,
          KeepMonthly: Number(keepMonthly) || 0,
          KeepYearly:  Number(keepYearly)  || 0,
        },
      }
      if (editingId) {
        await api.updatePolicy(editingId, data)
      } else {
        await api.createPolicy(data)
      }
      setShowForm(false); setEditingId(null); resetForm(); await load()
    } catch { setErr('Failed to save policy. Check all values.') }
    finally { setSaving(false) }
  }

  const engineInfo = ENGINES.find(e => e.value === engine)

  return (
    <div style={s.page}>
      <div style={s.topRow}>
        <SectionHeader title="Policies" count={policies.length} />
        <button onClick={() => { resetForm(); setShowForm(true) }} style={s.newBtn}>
          + New Policy
        </button>
      </div>

      <p style={s.sub}>
        Policies define <strong>what</strong> to back up, <strong>when</strong> to run,
        how long to <strong>retain</strong> snapshots, and when to run <strong>restore tests</strong>.
      </p>

      <Card>
        {loading ? <div style={s.load}>Loading…</div> : (
          <Table
            cols={[
              { header: 'Name',     render: p => <span style={s.name}>{p.Name}</span> },
              { header: 'Engine',   render: p => <span style={s.badge}>{p.Engine}</span>, width: '90px' },
              { header: 'Schedule', render: p => {
                const cron = p.ScheduleConfig?.cron ?? p.Schedule
                return cron
                  ? <span style={s.mono}>{cron}</span>
                  : <span style={s.dim}>manual</span>
              }},
              { header: 'Includes', render: p => (
                <div>{(p.Includes ?? []).slice(0,2).map(i => (
                  <div key={i} style={s.path}>{i}</div>
                ))}
                {(p.Includes?.length ?? 0) > 2 && <div style={s.dim}>+{(p.Includes?.length ?? 0) - 2} more</div>}
                </div>
              )},
              { header: 'Retention', render: p => {
                const r = p.RetentionPlan
                if (!r || (!r.KeepLast && !r.KeepDaily)) return <span style={s.dim}>—</span>
                const parts = []
                if (r.KeepLast)    parts.push(`last ${r.KeepLast}`)
                if (r.KeepDaily)   parts.push(`${r.KeepDaily}d`)
                if (r.KeepWeekly)  parts.push(`${r.KeepWeekly}w`)
                if (r.KeepMonthly) parts.push(`${r.KeepMonthly}m`)
                return <span style={s.mono}>{parts.join(' · ')}</span>
              }},
              { header: 'Repository', render: p => p.RepositoryID
                ? <span style={s.mono}>{repos.find(r => r.ID === p.RepositoryID)?.Location?.slice(-30) ?? p.RepositoryID.slice(0,8)+'…'}</span>
                : <span style={s.warn}>⚠ not set</span>,
              },
              { header: '', render: p => (
                <div style={{ display: 'flex', gap: 6 }}>
                  <button onClick={() => openEdit(p)} style={s.editBtn}>✏</button>
                  <button onClick={() => setDeleteFor(p)} style={s.delBtn}>🗑</button>
                </div>
              ), width: '70px' },
            ]}
            rows={policies} keyFn={p => p.ID}
            empty="No policies yet. Click '+ New Policy' to create one."
          />
        )}
      </Card>

      {/* ── Policy Form Modal ────────────────────────────────────────── */}
      {showForm && (
        <Modal
          title={editingId ? `Edit: ${name}` : 'New Backup Policy'}
          onClose={() => { setShowForm(false); setEditingId(null); resetForm() }}
        >
          <div>
            {/* Tabs */}
            <div style={s.tabs}>
              {(['general', 'schedule', 'retention'] as const).map(t => (
                <button key={t} onClick={() => setTab(t)}
                  style={{ ...s.tab, ...(tab === t ? s.tabOn : {}) }}>
                  {t === 'general' ? '⚙ General' : t === 'schedule' ? '🕐 Schedule' : '🗄 Retention'}
                </button>
              ))}
            </div>

            {/* ── General Tab ── */}
            {tab === 'general' && (
              <div>
                <div style={s.row2}>
                  <div style={s.field}>
                    <label style={s.label}>Policy Name <span style={s.req}>*</span></label>
                    <input style={s.input} value={name} onChange={e => setName(e.target.value)}
                      placeholder="e.g. nightly-documents" autoFocus />
                  </div>
                  <div style={s.field}>
                    <label style={s.label}>Repository <span style={s.req}>*</span></label>
                    <select style={s.select} value={repoID} onChange={e => setRepoID(e.target.value)}>
                      <option value="">— select repository —</option>
                      {repos.map(r => <option key={r.ID} value={r.ID}>{r.Location} ({r.Type})</option>)}
                    </select>
                    {!repoID && <div style={s.warnText}>⚠ Agent cannot run this policy without a repository.</div>}
                  </div>
                </div>

                {/* Engine selector */}
                <div style={s.field}>
                  <label style={s.label}>Backup Engine</label>
                  <div style={s.engineGrid}>
                    {ENGINES.map(e => (
                      <div key={e.value} onClick={() => setEngine(e.value)}
                        style={{ ...s.engineCard, ...(engine === e.value ? s.engineCardOn : {}) }}>
                        <div style={s.engineName}>{e.label}</div>
                        <div style={s.engineDesc}>{e.desc}</div>
                      </div>
                    ))}
                  </div>
                  {engineInfo && (
                    <div style={s.engineInfo}>
                      <span style={{ color: 'var(--success)', marginRight: 6 }}>▲</span>
                      <strong>Backup mode:</strong> {engineInfo.backupMode}
                    </div>
                  )}
                </div>

                {/* Include paths */}
                <div style={s.field}>
                  <label style={s.label}>Include Paths <span style={s.req}>*</span></label>
                  {includes.map((p, i) => (
                    <div key={i} style={s.pathRow}>
                      <input style={s.input} value={p}
                        onChange={e => setIncludes(prev => prev.map((v, j) => j === i ? e.target.value : v))}
                        placeholder={i === 0 ? 'e.g. C:/Users/Admin/Documents' : 'Another path…'} />
                      {includes.length > 1 && (
                        <button onClick={() => setIncludes(prev => prev.filter((_, j) => j !== i))} style={s.removeBtn}>✕</button>
                      )}
                    </div>
                  ))}
                  <button onClick={() => setIncludes(prev => [...prev, ''])} style={s.addBtn}>+ Add path</button>
                </div>

                {/* Exclude paths */}
                <div style={s.field}>
                  <label style={s.label}>Exclude Paths <span style={s.opt}>(optional)</span></label>
                  {excludes.map((p, i) => (
                    <div key={i} style={s.pathRow}>
                      <input style={s.input} value={p}
                        onChange={e => setExcludes(prev => prev.map((v, j) => j === i ? e.target.value : v))}
                        placeholder="e.g. C:/Users/Admin/AppData" />
                      <button onClick={() => setExcludes(prev => prev.filter((_, j) => j !== i))} style={s.removeBtn}>✕</button>
                    </div>
                  ))}
                  <button onClick={() => setExcludes(prev => [...prev, ''])} style={s.addBtn}>+ Add exclusion</button>
                </div>
              </div>
            )}

            {/* ── Schedule Tab ── */}
            {tab === 'schedule' && (
              <ScheduleBuilder value={schedule} onChange={setSchedule} />
            )}

            {/* ── Retention Tab ── */}
            {tab === 'retention' && (
              <div>
                <div style={s.retInfo}>
                  <strong>Retention rules</strong> define how many snapshots to keep.
                  All rules apply simultaneously — a snapshot is kept if it matches <em>any</em> rule.
                  The prune schedule determines when old snapshots are actually deleted.
                </div>

                <div style={s.retGrid}>
                  {[
                    { label: 'Keep Last',    hint: 'N most recent snapshots',         val: keepLast,    set: setKeepLast },
                    { label: 'Keep Daily',   hint: 'One per day for N days',           val: keepDaily,   set: setKeepDaily },
                    { label: 'Keep Weekly',  hint: 'One per week for N weeks',         val: keepWeekly,  set: setKeepWeekly },
                    { label: 'Keep Monthly', hint: 'One per month for N months',       val: keepMonthly, set: setKeepMonthly },
                    { label: 'Keep Yearly',  hint: 'One per year for N years',         val: keepYearly,  set: setKeepYearly },
                  ].map(({ label, hint, val, set }) => (
                    <div key={label} style={s.retCard}>
                      <div style={s.retCardLabel}>{label}</div>
                      <input type="number" min="0" value={val}
                        onChange={e => set(e.target.value)}
                        style={s.retInput} />
                      <div style={s.retHint}>{hint}</div>
                      {Number(val) === 0 && <div style={s.retDim}>disabled</div>}
                    </div>
                  ))}
                </div>

                <div style={s.retNote}>
                  <span style={{ color: 'var(--success)' }}>✓</span>{' '}
                  Safety rule: the last restore-tested snapshot is never deleted,
                  even if all retention rules would remove it.
                </div>

                <div style={{ marginTop: 16 }}>
                  <label style={s.label}>Prune Schedule</label>
                  <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 8 }}>
                    Configure when pruning runs in the <strong>Schedule</strong> tab → Retention / Prune Schedule.
                  </div>
                  {schedule.retention_cron ? (
                    <div style={{ fontSize: 12, color: 'var(--success)', fontFamily: 'var(--font-mono)' }}>
                      ✓ {schedule.retention_cron}
                    </div>
                  ) : (
                    <div style={{ fontSize: 12, color: 'var(--text-dim)' }}>
                      Not scheduled — prune runs manually only.{' '}
                      <button onClick={() => setTab('schedule')} style={s.inlineLink}>
                        Configure →
                      </button>
                    </div>
                  )}
                </div>
              </div>
            )}

            {err && <div style={{ ...s.errBox, marginTop: 12 }}>{err}</div>}

            <div style={s.actions}>
              <button onClick={() => { setShowForm(false); setEditingId(null); resetForm() }} style={s.cancelBtn}>
                Cancel
              </button>
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
          onConfirm={async () => { await api.deletePolicy(deleteFor.ID); setDeleteFor(null); await load() }}
          onCancel={() => setDeleteFor(null)}
        />
      )}
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  page:        { padding: '28px 36px', maxWidth: 1300 },
  topRow:      { display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 4 },
  sub:         { fontSize: 13, color: 'var(--text-muted)', marginBottom: 16 },
  load:        { padding: 40, color: 'var(--text-muted)', textAlign: 'center' },
  name:        { fontWeight: 600, color: 'var(--text)' },
  mono:        { fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--accent)' },
  badge:       { display: 'inline-block', padding: '2px 8px', borderRadius: 4, background: 'rgba(59,130,246,0.1)', color: 'var(--accent)', fontSize: 11, fontWeight: 600 },
  path:        { fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' },
  dim:         { color: 'var(--text-dim)', fontSize: 11 },
  warn:        { color: 'var(--warning)', fontSize: 12 },
  editBtn:     { padding: '3px 8px', borderRadius: 5, background: 'rgba(245,158,11,0.08)', color: 'var(--warning)', border: '1px solid rgba(245,158,11,0.2)', fontSize: 12, cursor: 'pointer' },
  delBtn:      { padding: '3px 8px', borderRadius: 5, background: 'rgba(244,63,94,0.08)', color: 'var(--error)', border: '1px solid rgba(244,63,94,0.2)', fontSize: 12, cursor: 'pointer' },
  newBtn:      { padding: '7px 16px', borderRadius: 6, background: 'var(--accent)', color: '#fff', border: 'none', fontSize: 13, fontWeight: 600, cursor: 'pointer' },
  // form
  tabs:        { display: 'flex', gap: 4, marginBottom: 20, borderBottom: '1px solid var(--border)', paddingBottom: 8 },
  tab:         { padding: '6px 16px', borderRadius: 6, background: 'none', border: '1px solid transparent', fontSize: 13, cursor: 'pointer', color: 'var(--text-muted)' },
  tabOn:       { background: 'var(--accent-dim)', borderColor: 'var(--accent)', color: 'var(--text)', fontWeight: 600 },
  row2:        { display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 4 },
  field:       { marginBottom: 16 },
  label:       { display: 'block', fontSize: 11, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase' as const, letterSpacing: '0.08em', marginBottom: 6 },
  req:         { color: 'var(--error)' },
  opt:         { fontWeight: 400, textTransform: 'none' as const, color: 'var(--text-dim)', letterSpacing: 0, fontSize: 10 },
  input:       { width: '100%', padding: '8px 11px', background: 'var(--bg)', border: '1px solid var(--border)', borderRadius: 6, color: 'var(--text)', fontSize: 13, outline: 'none' },
  select:      { width: '100%', padding: '8px 11px', background: 'var(--bg)', border: '1px solid var(--border)', borderRadius: 6, color: 'var(--text)', fontSize: 13, cursor: 'pointer' },
  warnText:    { fontSize: 11, color: 'var(--warning)', marginTop: 4 },
  pathRow:     { display: 'flex', gap: 6, marginBottom: 6, alignItems: 'center' },
  removeBtn:   { padding: '6px 10px', borderRadius: 5, background: 'rgba(244,63,94,0.08)', color: 'var(--error)', border: '1px solid rgba(244,63,94,0.2)', fontSize: 12, cursor: 'pointer', flexShrink: 0 },
  addBtn:      { padding: '5px 12px', borderRadius: 5, background: 'transparent', color: 'var(--accent)', border: '1px dashed rgba(59,130,246,0.4)', fontSize: 12, cursor: 'pointer', marginTop: 2 },
  // engine
  engineGrid:  { display: 'grid', gridTemplateColumns: 'repeat(4,1fr)', gap: 8, marginBottom: 8 },
  engineCard:  { padding: '10px 12px', borderRadius: 7, border: '1px solid var(--border)', cursor: 'pointer', transition: 'all 0.12s' },
  engineCardOn:{ borderColor: 'var(--accent)', background: 'var(--accent-dim)' },
  engineName:  { fontWeight: 700, fontSize: 13, color: 'var(--text)', marginBottom: 3 },
  engineDesc:  { fontSize: 10, color: 'var(--text-dim)' },
  engineInfo:  { fontSize: 12, color: 'var(--text-muted)', background: 'rgba(0,212,255,0.05)', border: '1px solid rgba(0,212,255,0.15)', borderRadius: 6, padding: '8px 12px', marginTop: 4 },
  // retention
  retInfo:     { fontSize: 13, color: 'var(--text-muted)', background: 'rgba(255,255,255,0.03)', borderRadius: 6, padding: '10px 14px', marginBottom: 16 },
  retGrid:     { display: 'grid', gridTemplateColumns: 'repeat(5,1fr)', gap: 10, marginBottom: 16 },
  retCard:     { background: 'var(--bg)', border: '1px solid var(--border)', borderRadius: 8, padding: '12px', textAlign: 'center' as const },
  retCardLabel:{ fontSize: 11, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase' as const, letterSpacing: '0.06em', marginBottom: 8 },
  retInput:    { width: 64, padding: '8px 0', background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 6, color: 'var(--text)', fontSize: 20, fontWeight: 700, textAlign: 'center' as const, outline: 'none' },
  retHint:     { fontSize: 10, color: 'var(--text-dim)', marginTop: 6 },
  retDim:      { fontSize: 10, color: 'var(--error)', marginTop: 2 },
  retNote:     { fontSize: 12, color: 'var(--text-muted)', background: 'rgba(0,255,136,0.05)', border: '1px solid rgba(0,255,136,0.15)', borderRadius: 6, padding: '8px 12px' },
  inlineLink:  { background: 'none', border: 'none', color: 'var(--accent)', fontSize: 12, cursor: 'pointer', padding: 0 },
  // actions
  errBox:      { background: 'rgba(244,63,94,0.1)', border: '1px solid rgba(244,63,94,0.25)', borderRadius: 6, padding: '8px 12px', fontSize: 13, color: 'var(--error)' },
  actions:     { display: 'flex', gap: 8, justifyContent: 'flex-end', paddingTop: 16, borderTop: '1px solid var(--border)', marginTop: 16 },
  cancelBtn:   { padding: '7px 16px', borderRadius: 6, background: 'transparent', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 13, cursor: 'pointer' },
  submitBtn:   { padding: '7px 20px', borderRadius: 6, background: 'var(--success)', color: '#000', border: 'none', fontSize: 13, fontWeight: 700, cursor: 'pointer' },
}
