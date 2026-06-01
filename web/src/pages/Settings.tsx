import { useState, useEffect } from 'react'
import { SectionHeader } from '../components/Card'

const STORAGE_KEY = 'osb-settings'

interface AppSettings {
  controlPlaneUrl: string
  pollIntervalSec: number
  compactSidebar: boolean
}

const defaults: AppSettings = {
  controlPlaneUrl: 'http://localhost:8080',
  pollIntervalSec: 30,
  compactSidebar: false,
}

function loadSettings(): AppSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? { ...defaults, ...JSON.parse(raw) } : defaults
  } catch {
    return defaults
  }
}

export function Settings() {
  const [settings, setSettings] = useState<AppSettings>(loadSettings)
  const [saved, setSaved] = useState(false)

  function save() {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  function reset() {
    setSettings(defaults)
    localStorage.removeItem(STORAGE_KEY)
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  const set = (k: keyof AppSettings, v: AppSettings[keyof AppSettings]) =>
    setSettings(prev => ({ ...prev, [k]: v }))

  return (
    <div style={s.page}>
      <SectionHeader title="Settings" />

      {/* Control Plane */}
      <div style={s.section}>
        <h2 style={s.sectionTitle}>Control Plane</h2>
        <div style={s.field}>
          <label style={s.label}>API URL</label>
          <input style={s.input} value={settings.controlPlaneUrl}
            onChange={e => set('controlPlaneUrl', e.target.value)}
            placeholder="http://localhost:8080" />
          <div style={s.hint}>
            The URL of your OpensourceBackup Control Plane API.
            Reload the page after changing.
          </div>
        </div>

        <div style={s.infoBox}>
          <strong>Current connection:</strong>{' '}
          <code style={s.code}>{import.meta.env.VITE_API_URL ?? 'http://localhost:8080'}</code>
          <div style={{marginTop:6, fontSize:11, color:'var(--text-dim)'}}>
            Set <code style={s.code}>VITE_API_URL=https://your-server:8443</code> in your
            Vite environment to connect to a remote control plane.
          </div>
        </div>
      </div>

      {/* Backup Agent */}
      <div style={s.section}>
        <h2 style={s.sectionTitle}>Agent</h2>
        <div style={s.field}>
          <label style={s.label}>Default Poll Interval (seconds)</label>
          <input style={{...s.input, width:100}} type="number" min="5" max="300"
            value={settings.pollIntervalSec}
            onChange={e => set('pollIntervalSec', Number(e.target.value))} />
          <div style={s.hint}>
            How often the agent checks for new jobs. Set via <code style={s.code}>AGENT_POLL_INTERVAL=30s</code> on the agent.
          </div>
        </div>
      </div>

      {/* Notifications */}
      <NotificationChannels />

      {/* About */}
      <div style={s.section}>
        <h2 style={s.sectionTitle}>About</h2>
        <div style={s.aboutCard}>
          <div style={s.aboutRow}>
            <span style={s.aboutKey}>Product</span>
            <span style={s.aboutVal}>OpenSourceBackup</span>
          </div>
          <div style={s.aboutRow}>
            <span style={s.aboutKey}>Version</span>
            <span style={s.aboutVal}>v0.1.0</span>
          </div>
          <div style={s.aboutRow}>
            <span style={s.aboutKey}>License</span>
            <span style={s.aboutVal}>Apache 2.0</span>
          </div>
          <div style={s.aboutRow}>
            <span style={s.aboutKey}>Repository</span>
            <a style={{color:'var(--accent)', fontSize:13}}
               href="https://github.com/cerberus8484/opensourcebackup"
               target="_blank" rel="noreferrer">
              github.com/cerberus8484/opensourcebackup
            </a>
          </div>
          <div style={s.aboutRow}>
            <span style={s.aboutKey}>Positionierung</span>
            <span style={{...s.aboutVal, fontStyle:'italic', color:'var(--text-muted)'}}>
              Creating backups is easy. Proving recoverability is the difference.
            </span>
          </div>
        </div>
      </div>

      {/* API-Status */}
      <div style={s.section}>
        <h2 style={s.sectionTitle}>API Status</h2>
        <ApiStatus />
      </div>

      {/* Actions */}
      <div style={s.actions}>
        <button onClick={reset} style={s.resetBtn}>Reset to defaults</button>
        <button onClick={save} style={s.saveBtn}>
          {saved ? '✓ Saved' : 'Save Settings'}
        </button>
      </div>
    </div>
  )
}

const BASE = import.meta.env.VITE_API_URL || ''
function csrfToken() { const m = document.cookie.match(/osb_csrf=([^;]+)/); return m ? decodeURIComponent(m[1]) : '' }

interface NotifyChannel { id: string; name: string; type: string; target: string; enabled: boolean; min_severity: string }

function NotificationChannels() {
  const [channels, setChannels] = useState<NotifyChannel[]>([])
  const [showAdd, setShowAdd]   = useState(false)
  const [name,    setName]      = useState('')
  const [type,    setType]      = useState('webhook')
  const [target,  setTarget]    = useState('')
  const [minSev,  setMinSev]    = useState('warning')
  const [saving,  setSaving]    = useState(false)
  const [tested,  setTested]    = useState<string|null>(null)

  const load = () => fetch(`${BASE}/v1/notifications`).then(r => r.ok ? r.json() : []).then(setChannels).catch(()=>{})
  useEffect(() => { load() }, [])

  async function add() {
    if (!name || !target) return
    setSaving(true)
    try {
      await fetch(`${BASE}/v1/notifications`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken() },
        body: JSON.stringify({ name, type, target, min_severity: minSev, enabled: true }),
      })
      setShowAdd(false); setName(''); setTarget(''); load()
    } finally { setSaving(false) }
  }

  async function test(ch: NotifyChannel) {
    setTested(null)
    const r = await fetch(`${BASE}/v1/notifications/${ch.id}/test`, {
      method: 'POST', headers: { 'X-CSRF-Token': csrfToken() }
    })
    setTested(r.ok ? '✓ Test sent' : '✗ Failed')
    setTimeout(() => setTested(null), 3000)
  }

  async function del(id: string) {
    await fetch(`${BASE}/v1/notifications/${id}`, { method: 'DELETE', headers: { 'X-CSRF-Token': csrfToken() } })
    load()
  }

  return (
    <div style={s.section}>
      <div style={{ display:'flex', justifyContent:'space-between', alignItems:'center', marginBottom:16 }}>
        <h2 style={s.sectionTitle}>Notifications</h2>
        <button onClick={() => setShowAdd(v=>!v)} style={s.resetBtn}>+ Add Channel</button>
      </div>
      <div style={{ fontSize:12, color:'var(--text-muted)', marginBottom:12 }}>
        Send alerts via Webhook (Slack, Teams, Discord, custom) or Email when backup health drops.
      </div>

      {channels.length === 0 && !showAdd && (
        <div style={{ fontSize:12, color:'var(--text-dim)', fontStyle:'italic' }}>No notification channels configured.</div>
      )}

      {channels.map(ch => (
        <div key={ch.id} style={{ display:'flex', alignItems:'center', gap:10, padding:'8px 0', borderBottom:'1px solid var(--border)' }}>
          <span style={{ fontSize:16 }}>{ch.type === 'webhook' ? '🔗' : '✉️'}</span>
          <span style={{ flex:1, fontSize:13, color:'var(--text)', fontWeight:600 }}>{ch.name}</span>
          <span style={{ fontSize:11, color:'var(--text-dim)' }}>{ch.min_severity}+</span>
          <button onClick={() => test(ch)} style={{ ...s.resetBtn, fontSize:11, padding:'3px 10px' }}>Test</button>
          <button onClick={() => del(ch.id)} style={{ ...s.resetBtn, fontSize:11, padding:'3px 8px', color:'var(--error)' }}>✕</button>
        </div>
      ))}

      {tested && <div style={{ fontSize:12, color:'var(--success)', marginTop:8 }}>{tested}</div>}

      {showAdd && (
        <div style={{ marginTop:16, display:'flex', flexDirection:'column', gap:10 }}>
          <div style={{ display:'grid', gridTemplateColumns:'1fr 1fr', gap:10 }}>
            <div>
              <label style={s.label}>Name</label>
              <input style={s.input} value={name} onChange={e=>setName(e.target.value)} placeholder="Slack Backup Alerts" />
            </div>
            <div>
              <label style={s.label}>Type</label>
              <select style={s.input} value={type} onChange={e=>setType(e.target.value)}>
                <option value="webhook">Webhook (Slack / Teams / Discord)</option>
                <option value="email">Email (coming soon)</option>
              </select>
            </div>
          </div>
          <div>
            <label style={s.label}>Webhook URL</label>
            <input style={s.input} value={target} onChange={e=>setTarget(e.target.value)} placeholder="https://hooks.slack.com/services/..." />
          </div>
          <div>
            <label style={s.label}>Minimum Severity</label>
            <select style={{...s.input, width:200}} value={minSev} onChange={e=>setMinSev(e.target.value)}>
              <option value="info">Info (all alerts)</option>
              <option value="warning">Warning + Critical</option>
              <option value="critical">Critical only</option>
            </select>
          </div>
          <div style={{ display:'flex', gap:8, justifyContent:'flex-end' }}>
            <button onClick={() => setShowAdd(false)} style={s.resetBtn}>Cancel</button>
            <button onClick={add} disabled={saving} style={s.saveBtn}>{saving ? 'Saving…' : 'Add Channel'}</button>
          </div>
        </div>
      )}
    </div>
  )
}

function ApiStatus() {
  const [status, setStatus] = useState<'checking'|'ok'|'error'>('checking')
  const base = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

  useEffect(() => {
    fetch(`${base}/health`)
      .then(r => r.ok ? setStatus('ok') : setStatus('error'))
      .catch(() => setStatus('error'))
  }, [base])

  return (
    <div style={s.statusCard}>
      <div style={{display:'flex', alignItems:'center', gap:10}}>
        <div style={{
          width:10, height:10, borderRadius:'50%',
          background: status==='ok' ? 'var(--success)' : status==='error' ? 'var(--error)' : 'var(--warning)',
          boxShadow: status==='ok' ? '0 0 8px var(--success)' : 'none',
        }}/>
        <span style={{fontSize:13, color:'var(--text)', fontWeight:600}}>
          {status==='ok' ? 'Control Plane reachable' : status==='error' ? 'Control Plane unreachable' : 'Checking…'}
        </span>
      </div>
      <div style={{fontSize:12, color:'var(--text-dim)', marginTop:6}}>
        {base}/health
      </div>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  page:       { padding:'28px 36px', maxWidth:720 },
  section:    { background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:10, padding:'20px 24px', marginBottom:16 },
  sectionTitle:{ fontSize:13, fontWeight:700, color:'var(--text-muted)', textTransform:'uppercase' as const, letterSpacing:'0.08em', marginBottom:16 },
  field:      { marginBottom:16 },
  label:      { display:'block', fontSize:11, fontWeight:700, color:'var(--text-dim)', textTransform:'uppercase' as const, letterSpacing:'0.08em', marginBottom:6 },
  input:      { width:'100%', padding:'8px 11px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, outline:'none' },
  hint:       { fontSize:11, color:'var(--text-dim)', marginTop:4, lineHeight:1.5 },
  code:       { fontFamily:'var(--font-mono)', fontSize:11, background:'rgba(0,0,0,0.3)', padding:'1px 5px', borderRadius:3 },
  infoBox:    { background:'rgba(59,130,246,0.06)', border:'1px solid rgba(59,130,246,0.15)', borderRadius:6, padding:'10px 14px', fontSize:13, color:'var(--text-muted)' },
  aboutCard:  { display:'flex', flexDirection:'column' as const, gap:8 },
  aboutRow:   { display:'flex', alignItems:'baseline', gap:12, fontSize:13, borderBottom:'1px solid var(--border)', paddingBottom:8 },
  aboutKey:   { width:120, flexShrink:0, color:'var(--text-dim)', fontSize:11, textTransform:'uppercase' as const, letterSpacing:'0.06em' },
  aboutVal:   { color:'var(--text)' },
  statusCard: { background:'var(--bg)', border:'1px solid var(--border)', borderRadius:8, padding:'12px 16px' },
  actions:    { display:'flex', gap:8, justifyContent:'flex-end' },
  resetBtn:   { padding:'8px 16px', borderRadius:6, background:'transparent', border:'1px solid var(--border)', color:'var(--text-muted)', fontSize:13, cursor:'pointer' },
  saveBtn:    { padding:'8px 20px', borderRadius:6, background:'var(--accent)', color:'#fff', border:'none', fontSize:13, fontWeight:600, cursor:'pointer' },
}
