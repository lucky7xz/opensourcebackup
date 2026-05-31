import { useEffect, useState } from 'react'
import { api, post, type System } from '../api'
import { ConfirmDialog } from '../components/ConfirmDialog'

const VERSION = 'v0.1.0'

const PLATFORMS = [
  { id: 'windows-amd64', label: 'Windows',      icon: '🪟', sub: 'Windows Server / Workstation (64-bit)' },
  { id: 'linux-amd64',   label: 'Linux x64',    icon: '🐧', sub: 'Debian, Ubuntu, RHEL, CentOS (64-bit)' },
  { id: 'linux-arm64',   label: 'Linux ARM64',  icon: '🐧', sub: 'Raspberry Pi, ARM servers' },
]

type Step = 'system' | 'platform' | 'config' | 'install'

export function Agents() {
  const [step,       setStep]       = useState<Step>('system')
  const [systems,    setSystems]    = useState<System[]>([])
  const [selSystem,    setSelSystem]    = useState<System|null>(null)
  const [showNewSys,   setShowNewSys]   = useState(false)
  const [newHostname,  setNewHostname]  = useState('')
  const [creatingSys,  setCreatingSys]  = useState(false)
  const [platform,   setPlatform]   = useState('')
  const [resticRepo, setResticRepo] = useState('C:/tmp/backup-repo')
  const [resticPass, setResticPass] = useState('')
  const [pollSec,    setPollSec]    = useState('30')
  const [token,      setToken]      = useState<string|null>(null)
  const [loading,    setLoading]    = useState(false)
  const [err,        setErr]        = useState<string|null>(null)
  const [deleteFor,  setDeleteFor]  = useState<System|null>(null)

  useEffect(() => { api.systems().then(setSystems) }, [])

  // ── Step helpers ───────────────────────────────────────────────────────────

  async function registerAndSelect() {
    if (!newHostname.trim()) return
    setCreatingSys(true); setErr(null)
    try {
      const sys = await post<System>('/v1/systems', { Hostname: newHostname.trim(), RiskClass: 'standard' })
      setSystems(prev => [...prev, sys])
      setSelSystem(sys)
      setShowNewSys(false)
      setNewHostname('')
    } catch { setErr('Could not register system. Is the control plane running?') }
    finally { setCreatingSys(false) }
  }

  function goToPlatform() {
    if (!selSystem) { setErr('Please select a system.'); return }
    setErr(null)
    setStep('platform')
  }

  function goToConfig() {
    if (!platform) { setErr('Select a platform.'); return }
    setErr(null)
    setResticRepo(platform === 'windows-amd64' ? 'C:/tmp/backup-repo' : '/tmp/backup-repo')
    setStep('config')
  }

  async function goToInstall() {
    if (!resticRepo.trim() || !resticPass.trim()) { setErr('Fill in repository path and password.'); return }
    setErr(null); setLoading(true)
    try {
      const et = await api.createEnrollmentToken(selSystem!.ID)
      setToken(et.token)
      setStep('install')
    } catch { setErr('Could not generate enrollment token.') }
    finally { setLoading(false) }
  }

  // ── Install command ────────────────────────────────────────────────────────

  const cpUrl = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

  function installCmd() {
    if (platform === 'windows-amd64') return (
`# Paste these commands into PowerShell — one block:

Invoke-WebRequest "${cpUrl}/downloads/agent/${VERSION}/windows-amd64" \`
  -OutFile opensourcebackup-agent.exe

$env:CONTROL_PLANE_URL  = "${cpUrl}"
$env:ENROLLMENT_TOKEN   = "${token}"
$env:RESTIC_PASSWORD    = "${resticPass}"
$env:RESTIC_REPO        = "${resticRepo}"
$env:AGENT_POLL_INTERVAL= "${pollSec}s"
.\\opensourcebackup-agent.exe`)
    return (
`# Paste into your terminal:

curl -fsSL "${cpUrl}/downloads/agent/${VERSION}/${platform}" \\
  -o /usr/local/bin/opensourcebackup-agent
chmod +x /usr/local/bin/opensourcebackup-agent

CONTROL_PLANE_URL="${cpUrl}" \\
ENROLLMENT_TOKEN="${token}" \\
RESTIC_PASSWORD="${resticPass}" \\
RESTIC_REPO="${resticRepo}" \\
AGENT_POLL_INTERVAL="${pollSec}s" \\
/usr/local/bin/opensourcebackup-agent`)
  }

  function reset() {
    setStep('system'); setSelSystem(null); setNewHostname('')
    setPlatform(''); setResticRepo(''); setResticPass('')
    setToken(null); setErr(null)
  }

  // ── Render ─────────────────────────────────────────────────────────────────

  const steps: Step[] = ['system', 'platform', 'config', 'install']
  const stepLabels = ['System', 'Platform', 'Configure', 'Install']

  return (
    <div style={s.page}>
      <h1 style={s.h1}>Install Agent</h1>
      <p style={s.sub}>Follow the steps to deploy the agent on a new system.</p>

      {/* Progress bar */}
      <div style={s.progress}>
        {steps.map((st, i) => {
          const done    = steps.indexOf(step) > i
          const current = step === st
          return (
            <div key={st} style={s.progressStep}>
              <div style={{
                ...s.dot,
                background: done ? 'var(--success)' : current ? 'var(--accent)' : 'var(--border)',
                color: done || current ? '#fff' : 'var(--text-dim)',
              }}>
                {done ? '✓' : i+1}
              </div>
              <span style={{ fontSize:12, color: current ? 'var(--text)' : 'var(--text-dim)', fontWeight: current ? 600 : 400 }}>
                {stepLabels[i]}
              </span>
              {i < steps.length-1 && <div style={s.line} />}
            </div>
          )
        })}
      </div>

      {/* Step content */}
      <div style={s.card}>

        {/* ── Step 1: System ── */}
        {step === 'system' && (
          <>
            <h2 style={s.stepTitle}>Which system should the agent run on?</h2>
            <p style={s.stepSub}>
              Select a registered system. To add a new system, go to{' '}
              <a href="/systems" style={{color:'var(--accent)'}}>Systems</a> first.
            </p>

            {systems.length === 0 ? (
              <div style={s.emptyHint}>
                No systems registered yet.{' '}
                <a href="/systems" style={{color:'var(--accent)'}}>Go to Systems →</a>{' '}
                to register your first system.
              </div>
            ) : (
              <div style={s.section}>
                <div style={s.systemList}>
                  {systems.map(sys => (
                    <div key={sys.ID} onClick={() => setSelSystem(sys)}
                      style={{...s.systemItem, ...(selSystem?.ID===sys.ID ? s.systemItemOn : {})}}>
                      <span style={s.sysIcon}>🖥</span>
                      <div>
                        <div style={{fontWeight:600, color:'var(--text)'}}>{sys.Hostname}</div>
                        <div style={{fontSize:11, color:'var(--text-dim)'}}>{sys.OS ?? 'unknown OS'} · {sys.RiskClass}</div>
                      </div>
                      {selSystem?.ID===sys.ID && <span style={s.checkmark}>✓</span>}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Register new system inline */}
            <div style={s.newSysToggle}>
              <button onClick={() => { setShowNewSys(v => !v); setErr(null) }} style={s.addBtn}>
                {showNewSys ? '▲ Cancel' : '+ New Agent'}
              </button>
            </div>

            {showNewSys && (
              <div style={s.newSysBox}>
                <div style={s.label}>Agent Name / Hostname</div>
                <div style={{ display:'flex', gap:8 }}>
                  <input
                    style={{ ...s.input, flex:1 }}
                    placeholder="z.B. web-server-01 oder 192.168.1.10"
                    value={newHostname}
                    onChange={e => setNewHostname(e.target.value)}
                    onKeyDown={e => e.key==='Enter' && registerAndSelect()}
                    autoFocus
                  />
                  <button
                    onClick={registerAndSelect}
                    disabled={creatingSys || !newHostname.trim()}
                    style={s.primary}
                  >
                    {creatingSys ? '…' : 'Add'}
                  </button>
                </div>
              </div>
            )}

            {err && <div style={s.err}>{err}</div>}
            <div style={s.actions}>
              <button onClick={goToPlatform} disabled={!selSystem} style={s.primary}>
                Continue →
              </button>
            </div>
          </>
        )}

        {/* ── Step 2: Platform ── */}
        {step === 'platform' && (
          <>
            <h2 style={s.stepTitle}>What type of system is <em style={{color:'var(--accent)'}}>{selSystem?.Hostname}</em>?</h2>
            <p style={s.stepSub}>Choose the operating system of the target machine.</p>

            <div style={s.platformGrid}>
              {PLATFORMS.map(p => (
                <div key={p.id} onClick={() => setPlatform(p.id)}
                  style={{...s.platformCard, ...(platform===p.id ? s.platformCardOn : {})}}>
                  <div style={{fontSize:28, marginBottom:8}}>{p.icon}</div>
                  <div style={{fontWeight:700, color:'var(--text)', marginBottom:4}}>{p.label}</div>
                  <div style={{fontSize:12, color:'var(--text-muted)'}}>{p.sub}</div>
                </div>
              ))}
            </div>

            {err && <div style={s.err}>{err}</div>}
            <div style={s.actions}>
              <button onClick={() => setStep('system')} style={s.back}>← Back</button>
              <button onClick={goToConfig} disabled={!platform} style={s.primary}>Continue →</button>
            </div>
          </>
        )}

        {/* ── Step 3: Config ── */}
        {step === 'config' && (
          <>
            <h2 style={s.stepTitle}>Where should the backup be stored?</h2>
            <p style={s.stepSub}>Configure the backup repository and encryption password.</p>

            <div style={s.section}>
              <label style={s.label}>Backup repository path or URL</label>
              <input style={s.input} value={resticRepo} onChange={e => setResticRepo(e.target.value)}
                placeholder={platform==='windows-amd64' ? 'C:/backups or s3:bucket/path' : '/var/backups or s3:bucket/path'} />
              <div style={s.hint}>Restic supports: local path, s3:, sftp:, b2:, azure:, gs:</div>
            </div>

            <div style={s.section}>
              <label style={s.label}>Encryption password</label>
              <input style={s.input} type="password" value={resticPass}
                onChange={e => setResticPass(e.target.value)}
                placeholder="Strong password — store it safely, you need it for restores!" />
              <div style={s.hint}>⚠ This password encrypts all backups. Keep it safe — without it, data cannot be restored.</div>
            </div>

            <div style={s.section}>
              <label style={s.label}>Poll interval (seconds)</label>
              <input style={{...s.input, width:100}} value={pollSec} onChange={e => setPollSec(e.target.value)} />
              <div style={s.hint}>How often the agent checks for new backup jobs (default: 30s)</div>
            </div>

            {err && <div style={s.err}>{err}</div>}
            <div style={s.actions}>
              <button onClick={() => setStep('platform')} style={s.back}>← Back</button>
              <button onClick={goToInstall} disabled={loading || !resticRepo.trim() || !resticPass.trim()} style={s.primary}>
                {loading ? 'Generating token…' : 'Generate install command →'}
              </button>
            </div>
          </>
        )}

        {/* ── Step 4: Install ── */}
        {step === 'install' && token && (
          <>
            <h2 style={s.stepTitle}>Ready to install on <em style={{color:'var(--accent)'}}>{selSystem?.Hostname}</em></h2>
            <p style={s.stepSub}>Copy the command below and run it on your target system. The token expires in 30 minutes.</p>

            <div style={s.checklist}>
              <div style={s.checkItem}><span style={s.checkIcon}>✓</span> System registered: <strong>{selSystem?.Hostname}</strong></div>
              <div style={s.checkItem}><span style={s.checkIcon}>✓</span> Platform: <strong>{PLATFORMS.find(p=>p.id===platform)?.label}</strong></div>
              <div style={s.checkItem}><span style={s.checkIcon}>✓</span> Repository: <strong style={{fontFamily:'var(--font-mono)', fontSize:12}}>{resticRepo}</strong></div>
              <div style={s.checkItem}><span style={s.checkIcon}>✓</span> Enrollment token generated (30 min TTL)</div>
            </div>

            <div style={s.cmdBox}>
              <div style={s.cmdHeader}>
                <span style={{fontSize:12, color:'var(--text-muted)'}}>
                  {platform === 'windows-amd64' ? 'PowerShell' : 'Terminal / bash'}
                </span>
                <button onClick={() => navigator.clipboard.writeText(installCmd())} style={s.copyBtn}>
                  📋 Copy
                </button>
              </div>
              <pre style={s.pre}>{installCmd()}</pre>
            </div>

            <div style={s.infoBox}>
              The agent will enroll automatically on first run and save the token to <code style={s.code}>data/agent-token</code>.
              On subsequent starts, only <code style={s.code}>CONTROL_PLANE_URL</code>, <code style={s.code}>RESTIC_PASSWORD</code> and <code style={s.code}>RESTIC_REPO</code> are needed.
            </div>

            {/* Stop / Start / Restart commands */}
            <h3 style={s.cmdSectionTitle}>Stop / Start / Restart</h3>

            {platform === 'windows-amd64' ? (
              <>
                <div style={s.cmdRow}>
                  <span style={s.cmdLabel}>Stop</span>
                  <pre style={s.cmdLine}>Stop-Process -Name "opensourcebackup-agent" -Force</pre>
                  <button onClick={() => navigator.clipboard.writeText('Stop-Process -Name "opensourcebackup-agent" -Force')} style={s.copySmall}>📋</button>
                </div>
                <div style={s.cmdRow}>
                  <span style={s.cmdLabel}>Start</span>
                  <pre style={s.cmdLine}>{`$env:CONTROL_PLANE_URL="${cpUrl}"; $env:RESTIC_PASSWORD="${resticPass}"; $env:RESTIC_REPO="${resticRepo}"; .\\opensourcebackup-agent.exe`}</pre>
                  <button onClick={() => navigator.clipboard.writeText(`$env:CONTROL_PLANE_URL="${cpUrl}"; $env:RESTIC_PASSWORD="${resticPass}"; $env:RESTIC_REPO="${resticRepo}"; .\\opensourcebackup-agent.exe`)} style={s.copySmall}>📋</button>
                </div>
                <div style={s.cmdRow}>
                  <span style={s.cmdLabel}>Restart</span>
                  <pre style={s.cmdLine}>{`Stop-Process -Name "opensourcebackup-agent" -Force -ErrorAction SilentlyContinue; Start-Sleep 1; $env:CONTROL_PLANE_URL="${cpUrl}"; $env:RESTIC_PASSWORD="${resticPass}"; $env:RESTIC_REPO="${resticRepo}"; .\\opensourcebackup-agent.exe`}</pre>
                  <button onClick={() => navigator.clipboard.writeText(`Stop-Process -Name "opensourcebackup-agent" -Force -ErrorAction SilentlyContinue; Start-Sleep 1; $env:CONTROL_PLANE_URL="${cpUrl}"; $env:RESTIC_PASSWORD="${resticPass}"; $env:RESTIC_REPO="${resticRepo}"; .\\opensourcebackup-agent.exe`)} style={s.copySmall}>📋</button>
                </div>
                <div style={s.cmdRow}>
                  <span style={s.cmdLabel}>Status</span>
                  <pre style={s.cmdLine}>Get-Process -Name "opensourcebackup-agent" -ErrorAction SilentlyContinue</pre>
                  <button onClick={() => navigator.clipboard.writeText('Get-Process -Name "opensourcebackup-agent" -ErrorAction SilentlyContinue')} style={s.copySmall}>📋</button>
                </div>
              </>
            ) : (
              <>
                <div style={s.cmdRow}>
                  <span style={s.cmdLabel}>Stop</span>
                  <pre style={s.cmdLine}>systemctl stop opensourcebackup-agent</pre>
                  <button onClick={() => navigator.clipboard.writeText('systemctl stop opensourcebackup-agent')} style={s.copySmall}>📋</button>
                </div>
                <div style={s.cmdRow}>
                  <span style={s.cmdLabel}>Start</span>
                  <pre style={s.cmdLine}>systemctl start opensourcebackup-agent</pre>
                  <button onClick={() => navigator.clipboard.writeText('systemctl start opensourcebackup-agent')} style={s.copySmall}>📋</button>
                </div>
                <div style={s.cmdRow}>
                  <span style={s.cmdLabel}>Restart</span>
                  <pre style={s.cmdLine}>systemctl restart opensourcebackup-agent</pre>
                  <button onClick={() => navigator.clipboard.writeText('systemctl restart opensourcebackup-agent')} style={s.copySmall}>📋</button>
                </div>
                <div style={s.cmdRow}>
                  <span style={s.cmdLabel}>Status</span>
                  <pre style={s.cmdLine}>systemctl status opensourcebackup-agent</pre>
                  <button onClick={() => navigator.clipboard.writeText('systemctl status opensourcebackup-agent')} style={s.copySmall}>📋</button>
                </div>
                <div style={s.cmdRow}>
                  <span style={s.cmdLabel}>Logs</span>
                  <pre style={s.cmdLine}>journalctl -u opensourcebackup-agent -f</pre>
                  <button onClick={() => navigator.clipboard.writeText('journalctl -u opensourcebackup-agent -f')} style={s.copySmall}>📋</button>
                </div>
              </>
            )}

            <div style={s.actions}>
              <button onClick={reset} style={s.back}>Install another agent</button>
            </div>
          </>
        )}

      </div>

      {/* Connected agents */}
      <div style={{marginTop:32}}>
        <h2 style={s.sectionTitle}>Connected Systems ({systems.length})</h2>
        <div style={s.agentGrid}>
          {systems.map(sys => (
            <div key={sys.ID} style={s.agentCard}>
              <div style={{fontSize:20, marginBottom:8}}>🖥</div>
              <div style={{fontWeight:600, color:'var(--text)', fontSize:13}}>{sys.Hostname}</div>
              <div style={{fontSize:11, color:'var(--text-dim)', marginTop:2}}>{sys.OS ?? 'unknown OS'}</div>
              <div style={{fontSize:11, color:'var(--text-dim)', marginBottom:10}}>{sys.RiskClass}</div>
              <button onClick={() => setDeleteFor(sys)} style={s.agentDelBtn}>🗑 Remove</button>
            </div>
          ))}
          {systems.length === 0 && (
            <div style={{color:'var(--text-dim)', fontSize:13}}>No systems yet. Use the wizard above to install your first agent.</div>
          )}
        </div>
      </div>

      {deleteFor && (
        <ConfirmDialog
          title={`Remove ${deleteFor.Hostname}?`}
          message={`This will delete the system record and revoke all agent tokens for ${deleteFor.Hostname}. The running agent will stop authenticating on the next poll (within 30s). This cannot be undone.`}
          confirmLabel="Remove Agent"
          danger
          onConfirm={async () => {
            await api.deleteSystem(deleteFor.ID)
            setDeleteFor(null)
            setSystems(prev => prev.filter(s => s.ID !== deleteFor.ID))
          }}
          onCancel={() => setDeleteFor(null)}
        />
      )}
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  page:         { padding:'28px 36px', maxWidth:860 },
  h1:           { fontSize:22, fontWeight:700, color:'var(--text)', marginBottom:4 },
  sub:          { fontSize:13, color:'var(--text-muted)', marginBottom:28 },
  progress:     { display:'flex', alignItems:'center', marginBottom:28 },
  progressStep: { display:'flex', alignItems:'center', gap:8 },
  dot:          { width:28, height:28, borderRadius:'50%', display:'flex', alignItems:'center', justifyContent:'center', fontSize:12, fontWeight:700, flexShrink:0 },
  line:         { width:40, height:1, background:'var(--border)', margin:'0 4px' },
  card:         { background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:10, padding:'28px 32px' },
  stepTitle:    { fontSize:18, fontWeight:700, color:'var(--text)', marginBottom:6 },
  stepSub:      { fontSize:13, color:'var(--text-muted)', marginBottom:24 },
  section:      { marginBottom:20 },
  label:        { display:'block', fontSize:11, fontWeight:700, color:'var(--text-dim)', textTransform:'uppercase' as const, letterSpacing:'0.08em', marginBottom:8 },
  input:        { width:'100%', padding:'9px 12px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, outline:'none' },
  hint:         { fontSize:11, color:'var(--text-dim)', marginTop:5 },
  divider:      { textAlign:'center' as const, color:'var(--text-dim)', fontSize:12, margin:'16px 0', position:'relative' as const },
  systemList:   { display:'flex', flexDirection:'column' as const, gap:8 },
  systemItem:   { display:'flex', alignItems:'center', gap:12, padding:'12px 16px', borderRadius:8, border:'1px solid var(--border)', cursor:'pointer', transition:'all 0.12s' },
  systemItemOn: { borderColor:'var(--accent)', background:'var(--accent-dim)' },
  sysIcon:      { fontSize:20 },
  checkmark:    { marginLeft:'auto', color:'var(--accent)', fontWeight:700, fontSize:16 },
  emptyHint:    { background:'rgba(245,158,11,0.07)', border:'1px solid rgba(245,158,11,0.2)', borderRadius:8, padding:'14px 16px', fontSize:13, color:'var(--text-muted)' },
  newSysToggle: { marginTop:12 },
  newSysBox:    { marginTop:8, background:'var(--bg)', border:'1px solid var(--border)', borderRadius:8, padding:'14px 16px' },
  input:        { padding:'8px 11px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:6, color:'var(--text)', fontSize:13, outline:'none' },
  platformGrid: { display:'grid', gridTemplateColumns:'repeat(3,1fr)', gap:12, marginBottom:8 },
  platformCard: { padding:'20px 16px', borderRadius:8, border:'1px solid var(--border)', cursor:'pointer', textAlign:'center' as const, transition:'all 0.12s' },
  platformCardOn:{ borderColor:'var(--accent)', background:'var(--accent-dim)' },
  err:          { background:'rgba(244,63,94,0.1)', border:'1px solid rgba(244,63,94,0.25)', borderRadius:6, padding:'8px 12px', fontSize:13, color:'var(--error)', marginBottom:12 },
  actions:      { display:'flex', gap:8, justifyContent:'flex-end', marginTop:24, paddingTop:20, borderTop:'1px solid var(--border)' },
  primary:      { padding:'9px 22px', borderRadius:6, background:'var(--accent)', color:'#fff', border:'none', fontSize:13, fontWeight:600, cursor:'pointer' },
  back:         { padding:'9px 16px', borderRadius:6, background:'transparent', border:'1px solid var(--border)', color:'var(--text-muted)', fontSize:13, cursor:'pointer' },
  checklist:    { background:'var(--bg)', border:'1px solid var(--border)', borderRadius:8, padding:'14px 16px', marginBottom:16 },
  checkItem:    { display:'flex', alignItems:'center', gap:8, fontSize:13, color:'var(--text-muted)', marginBottom:6 },
  checkIcon:    { color:'var(--success)', fontWeight:700 },
  cmdBox:       { background:'#0a0d14', border:'1px solid var(--border)', borderRadius:8, overflow:'hidden', marginBottom:14 },
  cmdHeader:    { display:'flex', justifyContent:'space-between', alignItems:'center', padding:'8px 14px', borderBottom:'1px solid var(--border)', background:'rgba(255,255,255,0.03)' },
  copyBtn:      { padding:'3px 10px', borderRadius:4, background:'var(--accent-dim)', color:'var(--accent)', border:'1px solid rgba(59,130,246,0.3)', fontSize:11, cursor:'pointer' },
  pre:          { padding:'16px', fontSize:12, fontFamily:'var(--font-mono)', color:'#a8d8ea', overflow:'auto', whiteSpace:'pre', margin:0, lineHeight:1.7 },
  infoBox:      { background:'rgba(59,130,246,0.07)', border:'1px solid rgba(59,130,246,0.15)', borderRadius:8, padding:'10px 14px', fontSize:12, color:'var(--text-muted)', lineHeight:1.7 },
  cmdSectionTitle: { fontSize:12, fontWeight:700, color:'var(--text-dim)', textTransform:'uppercase' as const, letterSpacing:'0.08em', margin:'20px 0 10px' },
  cmdRow:       { display:'flex', alignItems:'center', gap:8, marginBottom:6, background:'#0a0d14', borderRadius:6, padding:'6px 10px' },
  cmdLabel:     { fontSize:10, fontWeight:700, color:'var(--text-dim)', textTransform:'uppercase' as const, letterSpacing:'0.06em', width:52, flexShrink:0 },
  cmdLine:      { flex:1, fontFamily:'var(--font-mono)', fontSize:11, color:'#a8d8ea', overflow:'auto', whiteSpace:'nowrap' as const, margin:0, padding:0, background:'none' },
  copySmall:    { padding:'2px 8px', borderRadius:4, background:'var(--accent-dim)', color:'var(--accent)', border:'none', fontSize:11, cursor:'pointer', flexShrink:0 },
  code:         { fontFamily:'var(--font-mono)', fontSize:11, background:'rgba(0,0,0,0.3)', padding:'1px 5px', borderRadius:3 },
  sectionTitle: { fontSize:14, fontWeight:700, color:'var(--text-muted)', textTransform:'uppercase' as const, letterSpacing:'0.08em', marginBottom:14 },
  agentGrid:    { display:'grid', gridTemplateColumns:'repeat(4,1fr)', gap:12 },
  agentCard:    { background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:8, padding:'16px', textAlign:'center' as const, display:'flex', flexDirection:'column' as const, alignItems:'center' },
  agentDelBtn:  { padding:'4px 12px', borderRadius:5, background:'rgba(244,63,94,0.08)', color:'var(--error)', border:'1px solid rgba(244,63,94,0.2)', fontSize:11, cursor:'pointer', width:'100%' },
}
