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
