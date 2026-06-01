import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  api, post, put,
  type System, type BackupJob, type BackupPolicy,
  type BackupRepository, type RepositoryHealth,
  type RestoreTest, type Snapshot,
} from '../api'
import { Topbar }              from '../components/Topbar'
import { Modal }               from '../components/Modal'
import { ConfirmDialog }       from '../components/ConfirmDialog'
import { SystemsTable }        from '../components/systems/SystemsTable'
import { SystemDetailPanel }   from '../components/systems/SystemDetailPanel'
import { RepositoriesTable }   from '../components/repositories/RepositoriesTable'

type ActiveTab = 'systems' | 'repositories'

// ── Risk-class filter options ──────────────────────────────────────────────────
const RISK_OPTIONS = ['All Risk Classes', 'Low', 'Medium', 'High', 'Critical']
const STATUS_OPTIONS = ['All Agent Status', 'Online', 'Idle', 'Offline']

function agentStatusOf(sys: System): 'online' | 'idle' | 'offline' {
  if (!sys.LastSeen) return 'offline'
  const mins = (Date.now() - new Date(sys.LastSeen).getTime()) / 60000
  return mins <= 2 ? 'online' : mins <= 15 ? 'idle' : 'offline'
}

function riskMatchesFilter(sys: System, filter: string): boolean {
  if (filter === 'All Risk Classes') return true
  const rc = sys.RiskClass?.toLowerCase() ?? 'standard'
  // map standard → low
  const normalized = rc === 'standard' ? 'low' : rc
  return normalized === filter.toLowerCase()
}

function statusMatchesFilter(sys: System, filter: string): boolean {
  if (filter === 'All Agent Status') return true
  return agentStatusOf(sys) === filter.toLowerCase()
}

// ── Component ─────────────────────────────────────────────────────────────────

export function Systems() {
  const nav = useNavigate()

  // ── Data ──
  const [systems,      setSystems]      = useState<System[]>([])
  const [jobs,         setJobs]         = useState<BackupJob[]>([])
  const [policies,     setPolicies]     = useState<BackupPolicy[]>([])
  const [restoreTests, setRestoreTests] = useState<RestoreTest[]>([])
  const [snapshots,    setSnapshots]    = useState<Snapshot[]>([])
  const [repos,        setRepos]        = useState<BackupRepository[]>([])
  const [repoHealth,   setRepoHealth]   = useState<RepositoryHealth[]>([])
  const [loading,      setLoading]      = useState(true)

  // ── UI state ──
  const [activeTab,   setActiveTab]   = useState<ActiveTab>('systems')
  const [selected,    setSelected]    = useState<System | null>(null)
  const [riskFilter,  setRiskFilter]  = useState('All Risk Classes')
  const [statFilter,  setStatFilter]  = useState('All Agent Status')
  const [search,      setSearch]      = useState('')
  const [msg,         setMsg]         = useState<string | null>(null)

  // ── Modal states (run, new, edit, delete) ──
  const [runFor,       setRunFor]       = useState<System | null>(null)
  const [editFor,      setEditFor]      = useState<System | null>(null)
  const [deleteFor,    setDeleteFor]    = useState<System | null>(null)
  const [showNewSys,   setShowNewSys]   = useState(false)

  // new system form
  const [newHostname,  setNewHostname]  = useState('')
  const [newOS,        setNewOS]        = useState('')
  const [newRisk,      setNewRisk]      = useState('standard')
  const [newTags,      setNewTags]      = useState('')
  const [saving,       setSaving]       = useState(false)
  const [saveErr,      setSaveErr]      = useState<string | null>(null)

  // edit form
  const [editHostname, setEditHostname] = useState('')
  const [editOS,       setEditOS]       = useState('')
  const [editRisk,     setEditRisk]     = useState('standard')
  const [editSaving,   setEditSaving]   = useState(false)

  // run backup
  const [selPolicy,    setSelPolicy]    = useState('')
  const [creating,     setCreating]     = useState(false)
  const [deleting,     setDeleting]     = useState(false)

  // ── Load all data ─────────────────────────────────────────────────────────
  const load = () =>
    Promise.all([
      api.systems(),
      api.jobs(),
      api.policies(),
      api.restoreTests().catch(() => [] as RestoreTest[]),
      api.snapshots().catch(() => [] as Snapshot[]),
      api.repositories().catch(() => [] as BackupRepository[]),
      api.repositoryHealth().catch(() => [] as RepositoryHealth[]),
    ]).then(([sy, j, p, rt, sn, r, rh]) => {
      setSystems(sy)
      setJobs(j)
      setPolicies(p)
      setRestoreTests(rt)
      setSnapshots(sn)
      setRepos(r)
      setRepoHealth(rh)
    }).finally(() => setLoading(false))

  useEffect(() => { load() }, [])

  // ── Actions ───────────────────────────────────────────────────────────────

  async function createSystem() {
    if (!newHostname.trim()) { setSaveErr('Hostname is required.'); return }
    setSaving(true); setSaveErr(null)
    try {
      const tags: Record<string, string> = {}
      newTags.split(',').forEach(t => {
        const [k, v] = t.split('=').map(s => s.trim())
        if (k && v) tags[k] = v
      })
      await post<System>('/v1/systems', {
        Hostname:  newHostname.trim(),
        OS:        newOS.trim() || undefined,
        RiskClass: newRisk,
        Tags:      Object.keys(tags).length ? tags : undefined,
      })
      setShowNewSys(false)
      setNewHostname(''); setNewOS(''); setNewRisk('standard'); setNewTags('')
      setMsg('System registered.')
      await load()
    } catch { setSaveErr('Could not register system.') }
    finally { setSaving(false) }
  }

  async function saveEdit() {
    if (!editFor) return
    setEditSaving(true)
    try {
      await put<System>(`/v1/systems/${editFor.ID}`, {
        ...editFor, Hostname: editHostname, OS: editOS || undefined, RiskClass: editRisk,
      })
      setEditFor(null)
      await load()
    } catch { /* ignore */ }
    finally { setEditSaving(false) }
  }

  async function deleteSystem() {
    if (!deleteFor) return
    setDeleting(true)
    try {
      await api.deleteSystem(deleteFor.ID)
      setMsg(`${deleteFor.Hostname} removed.`)
      if (selected?.ID === deleteFor.ID) setSelected(null)
      setDeleteFor(null)
      await load()
    } catch { setMsg('Failed to delete system.') }
    finally { setDeleting(false) }
  }

  async function runBackup() {
    if (!runFor || !selPolicy) return
    setCreating(true)
    try {
      await api.createJob(runFor.ID, selPolicy)
      setMsg(`Backup job created for ${runFor.Hostname}`)
      setRunFor(null); setSelPolicy('')
      await load()
    } catch { setMsg('Failed to create job.') }
    finally { setCreating(false) }
  }

  // ── Derived / filtered data ────────────────────────────────────────────────

  const filteredSystems = systems.filter(sys => {
    if (!riskMatchesFilter(sys, riskFilter)) return false
    if (!statusMatchesFilter(sys, statFilter)) return false
    if (search && !sys.Hostname.toLowerCase().includes(search.toLowerCase())) return false
    return true
  })

  const alertCount = 0 // would come from health alerts

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <div style={s.page}>
      <Topbar
        title="Systems & Repositories"
        sub="Manage your backup infrastructure, agents, and data repositories."
        alertCount={alertCount}
      />

      <div style={s.layout}>

        {/* ── Main content ── */}
        <div style={s.main}>

          {msg && (
            <div style={s.msgBox} onClick={() => setMsg(null)}>{msg} ✕</div>
          )}

          {/* Tabs */}
          <div style={s.tabBar}>
            {(['systems', 'repositories'] as ActiveTab[]).map(t => (
              <button key={t} style={{ ...s.tab, ...(activeTab === t ? s.tabOn : {}) }}
                onClick={() => setActiveTab(t)}>
                {t === 'systems' ? 'Systems' : 'Repositories'}
                <span style={{ ...s.tabCount, ...(activeTab === t ? s.tabCountOn : {}) }}>
                  {t === 'systems' ? systems.length : repos.length}
                </span>
              </button>
            ))}
          </div>

          {/* Toolbar (systems tab only) */}
          {activeTab === 'systems' && (
            <div style={s.toolbar}>
              <div style={s.toolLeft}>
                <select style={s.filterSel} value={riskFilter} onChange={e => setRiskFilter(e.target.value)}>
                  {RISK_OPTIONS.map(o => <option key={o}>{o}</option>)}
                </select>
                <select style={s.filterSel} value={statFilter} onChange={e => setStatFilter(e.target.value)}>
                  {STATUS_OPTIONS.map(o => <option key={o}>{o}</option>)}
                </select>
                <button style={s.filterBtn}>More Filters</button>
                <button style={s.filterBtn}>Save View</button>
              </div>
              <div style={s.toolRight}>
                <button style={s.filterBtn}>Bulk Actions ▾</button>
                <button style={s.outlineBtn} onClick={() => { setShowNewSys(true); setSaveErr(null) }}>
                  + New System
                </button>
                <button style={s.primaryBtn} onClick={() => nav('/agents')}>
                  Enroll Agent
                </button>
              </div>
            </div>
          )}

          {/* Systems table */}
          {activeTab === 'systems' && (
            <div style={s.card}>
              <div style={s.cardHead}>
                <span style={s.cardTitle}>Systems</span>
                <span style={s.countChip}>{filteredSystems.length} system{filteredSystems.length !== 1 ? 's' : ''}</span>
                <div style={s.searchBox}>
                  <span style={{ fontSize: 11, color: 'var(--text-dim)' }}>🔍</span>
                  <input
                    style={s.searchIn}
                    placeholder="Filter by hostname…"
                    value={search}
                    onChange={e => setSearch(e.target.value)}
                  />
                </div>
              </div>
              {loading
                ? <div style={s.loading}>Loading…</div>
                : <SystemsTable
                    systems={filteredSystems}
                    jobs={jobs}
                    policies={policies}
                    restoreTests={restoreTests}
                    selected={selected}
                    onSelect={sys => setSelected(s => s?.ID === sys.ID ? null : sys)}
                    onRun={sys => { setRunFor(sys); setSelPolicy('') }}
                    onEdit={sys => { setEditFor(sys); setEditHostname(sys.Hostname); setEditOS(sys.OS ?? ''); setEditRisk(sys.RiskClass || 'standard') }}
                    onDelete={sys => setDeleteFor(sys)}
                  />
              }
            </div>
          )}

          {/* Repositories — shown when tab = repositories, also under systems */}
          {(activeTab === 'repositories' || activeTab === 'systems') && (
            <RepositoriesTable
              repos={repos}
              health={repoHealth}
              onNew={() => nav('/repositories')}
            />
          )}

        </div>

        {/* ── Right detail panel ── */}
        {selected && (
          <SystemDetailPanel
            system={selected}
            jobs={jobs}
            policies={policies}
            restoreTests={restoreTests}
            snapshots={snapshots}
            onClose={() => setSelected(null)}
            onRunBackup={sys => { setRunFor(sys); setSelPolicy('') }}
          />
        )}
      </div>

      {/* ── Modals ─────────────────────────────────────────────────────────── */}

      {showNewSys && (
        <Modal title="Register New System" onClose={() => setShowNewSys(false)}>
          <div style={f.field}>
            <label style={f.label}>Hostname / Name <span style={{ color: 'var(--error)' }}>*</span></label>
            <input style={f.input} value={newHostname} onChange={e => setNewHostname(e.target.value)}
              placeholder="e.g. web-server-01, db-cluster-02" autoFocus />
          </div>
          <div style={f.row2}>
            <div style={f.field}>
              <label style={f.label}>Operating System</label>
              <input style={f.input} value={newOS} onChange={e => setNewOS(e.target.value)}
                placeholder="e.g. Ubuntu 22.04, Windows Server 2022" />
            </div>
            <div style={f.field}>
              <label style={f.label}>Risk Class</label>
              <select style={f.select} value={newRisk} onChange={e => setNewRisk(e.target.value)}>
                <option value="standard">Low (Standard)</option>
                <option value="critical">High (Critical)</option>
              </select>
            </div>
          </div>
          <div style={f.field}>
            <label style={f.label}>Tags <span style={{ fontWeight: 400, fontSize: 10, color: 'var(--text-dim)' }}>(optional — key=value, comma-separated)</span></label>
            <input style={f.input} value={newTags} onChange={e => setNewTags(e.target.value)}
              placeholder="env=prod, location=berlin" />
          </div>
          {saveErr && <div style={f.err}>{saveErr}</div>}
          <div style={f.actions}>
            <button onClick={() => setShowNewSys(false)} style={f.cancel}>Cancel</button>
            <button onClick={createSystem} disabled={saving || !newHostname.trim()} style={f.submit}>
              {saving ? 'Registering…' : '✓ Register System'}
            </button>
          </div>
        </Modal>
      )}

      {editFor && (
        <Modal title={`Edit — ${editFor.Hostname}`} onClose={() => setEditFor(null)}>
          <div style={f.field}>
            <label style={f.label}>Hostname</label>
            <input style={f.input} value={editHostname} onChange={e => setEditHostname(e.target.value)} />
          </div>
          <div style={f.row2}>
            <div style={f.field}>
              <label style={f.label}>OS</label>
              <input style={f.input} value={editOS} onChange={e => setEditOS(e.target.value)} placeholder="Ubuntu 22.04" />
            </div>
            <div style={f.field}>
              <label style={f.label}>Risk Class</label>
              <select style={f.select} value={editRisk} onChange={e => setEditRisk(e.target.value)}>
                <option value="standard">Low (Standard)</option>
                <option value="critical">High (Critical)</option>
              </select>
            </div>
          </div>
          <div style={f.actions}>
            <button onClick={() => setEditFor(null)} style={f.cancel}>Cancel</button>
            <button onClick={saveEdit} disabled={editSaving} style={f.submit}>{editSaving ? 'Saving…' : '✓ Save'}</button>
          </div>
        </Modal>
      )}

      {deleteFor && (
        <ConfirmDialog
          title={`Remove ${deleteFor.Hostname}?`}
          message={`This will delete the system record and revoke all agent tokens for ${deleteFor.Hostname}. The agent process will stop authenticating on the next poll. This cannot be undone.`}
          confirmLabel={deleting ? 'Removing…' : 'Remove System'}
          danger
          onConfirm={deleteSystem}
          onCancel={() => setDeleteFor(null)}
        />
      )}

      {runFor && (
        <Modal title={`Run Backup — ${runFor.Hostname}`} onClose={() => setRunFor(null)}>
          <div style={f.field}>
            <label style={f.label}>Select Policy</label>
            <select value={selPolicy} onChange={e => setSelPolicy(e.target.value)} style={f.select}>
              <option value="">— select policy —</option>
              {policies.map(p => (
                <option key={p.ID} value={p.ID}>
                  {p.Name} ({p.Engine}){!p.RepositoryID ? ' ⚠ no repository' : ''}
                </option>
              ))}
            </select>
          </div>
          <div style={f.actions}>
            <button onClick={() => setRunFor(null)} style={f.cancel}>Cancel</button>
            <button onClick={runBackup} disabled={creating || !selPolicy} style={f.submit}>
              {creating ? 'Creating…' : '▶ Run Backup'}
            </button>
          </div>
        </Modal>
      )}
    </div>
  )
}

// ── Styles ────────────────────────────────────────────────────────────────────

const s: Record<string, React.CSSProperties> = {
  page:       { display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 },
  layout:     { display: 'flex', flex: 1, minHeight: 0, overflow: 'hidden' },
  main:       { flex: 1, overflowY: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: 14, minWidth: 0 },

  msgBox:     { background: 'rgba(34,197,94,0.1)', border: '1px solid rgba(34,197,94,0.3)', borderRadius: 6, padding: '8px 14px', fontSize: 13, color: 'var(--success)', cursor: 'pointer' },

  tabBar:     { display: 'flex', gap: 0, borderBottom: '1px solid var(--border)' },
  tab:        { padding: '10px 18px', background: 'none', border: 'none', borderBottom: '2px solid transparent', color: 'var(--text-dim)', fontSize: 13, fontWeight: 600, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 7, transition: 'all 0.12s', marginBottom: -1 },
  tabOn:      { color: 'var(--accent)', borderBottomColor: 'var(--accent)', background: 'rgba(137,189,40,0.04)' },
  tabCount:   { fontSize: 10, padding: '1px 6px', borderRadius: 8, background: 'rgba(255,255,255,0.06)', color: 'var(--text-dim)', fontWeight: 700 },
  tabCountOn: { background: 'rgba(137,189,40,0.15)', color: 'var(--accent)' },

  toolbar:    { display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 8 },
  toolLeft:   { display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' },
  toolRight:  { display: 'flex', gap: 6, alignItems: 'center' },
  filterSel:  { padding: '6px 10px', borderRadius: 7, background: 'var(--bg-card)', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 12, cursor: 'pointer', outline: 'none' },
  filterBtn:  { padding: '6px 12px', borderRadius: 7, background: 'var(--bg-card)', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 12, cursor: 'pointer' },
  outlineBtn: { padding: '7px 14px', borderRadius: 8, background: 'rgba(137,189,40,0.06)', border: '1px solid rgba(137,189,40,0.3)', color: 'var(--accent)', fontSize: 12, fontWeight: 600, cursor: 'pointer' },
  primaryBtn: { padding: '7px 16px', borderRadius: 8, background: 'var(--accent)', border: 'none', color: '#000', fontSize: 12, fontWeight: 700, cursor: 'pointer' },

  card:       { background: 'linear-gradient(180deg, rgba(21,28,46,0.95), rgba(10,15,27,0.95))', border: '1px solid rgba(148,163,184,0.12)', borderRadius: 16, overflow: 'hidden' },
  cardHead:   { display: 'flex', alignItems: 'center', gap: 10, padding: '14px 18px', borderBottom: '1px solid var(--border)' },
  cardTitle:  { fontSize: 14, fontWeight: 700, color: 'var(--text)' },
  countChip:  { fontSize: 11, padding: '2px 8px', borderRadius: 8, background: 'rgba(255,255,255,0.05)', color: 'var(--text-dim)' },
  searchBox:  { marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 6, background: 'var(--bg)', border: '1px solid var(--border)', borderRadius: 7, padding: '5px 10px' },
  searchIn:   { background: 'none', border: 'none', color: 'var(--text)', fontSize: 12, outline: 'none', width: 180 },
  loading:    { padding: 40, color: 'var(--text-muted)', textAlign: 'center', fontSize: 13 },
}

// Form styles
const f: Record<string, React.CSSProperties> = {
  field:   { marginBottom: 14 },
  label:   { display: 'block', fontSize: 11, fontWeight: 700, color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: 6 },
  input:   { width: '100%', padding: '8px 11px', background: 'var(--bg)', border: '1px solid var(--border)', borderRadius: 6, color: 'var(--text)', fontSize: 13, outline: 'none', boxSizing: 'border-box' },
  select:  { width: '100%', padding: '8px 11px', background: 'var(--bg)', border: '1px solid var(--border)', borderRadius: 6, color: 'var(--text)', fontSize: 13, cursor: 'pointer' },
  row2:    { display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 0 },
  err:     { background: 'rgba(244,63,94,0.1)', border: '1px solid rgba(244,63,94,0.25)', borderRadius: 6, padding: '8px 12px', fontSize: 13, color: 'var(--error)', marginBottom: 8 },
  actions: { display: 'flex', gap: 8, justifyContent: 'flex-end', paddingTop: 16, borderTop: '1px solid var(--border)' },
  cancel:  { padding: '7px 16px', borderRadius: 6, background: 'transparent', border: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: 13, cursor: 'pointer' },
  submit:  { padding: '7px 20px', borderRadius: 6, background: 'var(--success)', color: '#000', border: 'none', fontSize: 13, fontWeight: 700, cursor: 'pointer' },
}
