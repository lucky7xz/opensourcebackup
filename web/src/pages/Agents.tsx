import { useState } from 'react'
import { SectionHeader } from '../components/Card'

const AGENT_TYPES = [
  { id:'server',    label:'Server Agent',     icon:'🖥',  desc:'Linux / Windows Server, file backup via Restic',     os:['linux-amd64','linux-arm64','windows-amd64'] },
  { id:'endpoint',  label:'Windows Endpoint', icon:'🪟',  desc:'Windows workstations, MSI installer',                os:['windows-amd64'] },
  { id:'firewall',  label:'Firewall Agent',   icon:'🔒',  desc:'OPNsense / pfSense — config and ruleset backup',      os:['linux-amd64'] },
  { id:'vmhost',    label:'VM Host Agent',    icon:'🧮',  desc:'Proxmox / VMware / Hyper-V — VM config backup',       os:['linux-amd64'] },
  { id:'database',  label:'Database Agent',   icon:'🗄',  desc:'PostgreSQL (pgBackRest), MySQL, MongoDB',             os:['linux-amd64','linux-arm64'] },
  { id:'kubernetes',label:'Kubernetes',       icon:'☸',  desc:'Cluster backup via Velero — Helm chart deployment',   os:['helm'] },
]

const VERSION = 'v0.1.0'
const CP_URL  = 'http://localhost:8080'

export function Agents() {
  const [selected, setSelected] = useState<string|null>(null)
  const [enrollCmd, setEnrollCmd] = useState<string|null>(null)

  function generateEnrollCmd(_agentType: string, os: string) {
    const token = 'YOUR_ENROLLMENT_TOKEN_HERE'
    if (os === 'windows-amd64') {
      setEnrollCmd(
`# 1. Download agent (PowerShell)
Invoke-WebRequest "${CP_URL}/downloads/agent/${VERSION}/windows-amd64" \`
  -OutFile opensourcebackup-agent.exe

# 2. Set environment variables and run
# The agent enrolls automatically on first start (token saved to data\\agent-token)
$env:CONTROL_PLANE_URL  = "${CP_URL}"
$env:ENROLLMENT_TOKEN   = "${token}"
$env:RESTIC_PASSWORD    = "YOUR_RESTIC_PASSWORD"
$env:RESTIC_REPO        = "s3:your-bucket/backups/hostname"
.\\opensourcebackup-agent.exe

# 3. On subsequent starts (token already saved):
$env:CONTROL_PLANE_URL  = "${CP_URL}"
$env:RESTIC_PASSWORD    = "YOUR_RESTIC_PASSWORD"
$env:RESTIC_REPO        = "s3:your-bucket/backups/hostname"
.\\opensourcebackup-agent.exe`
      )
    } else if (os === 'helm') {
      setEnrollCmd(
`helm repo add opensourcebackup ${CP_URL}/helm
helm install opensourcebackup-agent opensourcebackup/agent \\
  --set controlPlane.url=${CP_URL} \\
  --set enrollmentToken=${token}`
      )
    } else {
      setEnrollCmd(
`# 1. Download agent
curl -fsSL "${CP_URL}/downloads/agent/${VERSION}/${os}" \\
  -o /usr/local/bin/opensourcebackup-agent
chmod +x /usr/local/bin/opensourcebackup-agent

# 2. Create environment file
mkdir -p /etc/opensourcebackup
cat > /etc/opensourcebackup/agent.env << EOF
CONTROL_PLANE_URL=${CP_URL}
ENROLLMENT_TOKEN=${token}
RESTIC_PASSWORD=YOUR_RESTIC_PASSWORD
RESTIC_REPO=s3:your-bucket/backups/$(hostname)
EOF
chmod 600 /etc/opensourcebackup/agent.env

# 3. First start — enrolls automatically (token saved to data/agent-token)
opensourcebackup-agent

# 4. Install as systemd service
cat > /etc/systemd/system/opensourcebackup-agent.service << 'UNIT'
[Unit]
Description=OpenSourceBackup Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/opensourcebackup-agent
Restart=always
EnvironmentFile=/etc/opensourcebackup/agent.env

[Install]
WantedBy=multi-user.target
UNIT

systemctl enable --now opensourcebackup-agent`
      )
    }
  }

  return (
    <div style={s.page}>
      <SectionHeader title="Agent Downloads & Deployment" />
      <p style={s.intro}>
        Select the agent type for your target system, generate an enrollment token,
        and copy the install command.
      </p>

      <div style={s.notice}>
        <strong>One binary, multiple profiles.</strong> All agent types use the same binary
        configured via <code style={s.code}>--profile</code>. Build from source or download
        a pre-built release.
      </div>

      {/* Agent type grid */}
      <div style={s.grid}>
        {AGENT_TYPES.map(a => (
          <div key={a.id}
            onClick={() => { setSelected(selected===a.id?null:a.id); setEnrollCmd(null) }}
            style={{...s.card, ...(selected===a.id?s.cardSelected:{})}}>
            <div style={s.cardIcon}>{a.icon}</div>
            <div style={s.cardTitle}>{a.label}</div>
            <div style={s.cardDesc}>{a.desc}</div>
            <div style={s.tags}>
              {a.os.map(o => <span key={o} style={s.tag}>{o}</span>)}
            </div>
          </div>
        ))}
      </div>

      {/* Install commands panel */}
      {selected && (()=>{
        const agent = AGENT_TYPES.find(a=>a.id===selected)!
        return (
          <div style={s.panel}>
            <h3 style={s.panelTitle}>Deploy {agent.label}</h3>

            <div style={s.step}>
              <div style={s.stepNum}>1</div>
              <div>
                <div style={s.stepTitle}>Register system in control plane</div>
                <pre style={s.pre}>{`curl -X POST ${CP_URL}/v1/systems \\
  -H "Content-Type: application/json" \\
  -d '{"Hostname":"<your-hostname>","RiskClass":"standard"}'`}</pre>
              </div>
            </div>

            <div style={s.step}>
              <div style={s.stepNum}>2</div>
              <div>
                <div style={s.stepTitle}>Generate enrollment token (30 min TTL)</div>
                <pre style={s.pre}>{`curl -X POST ${CP_URL}/v1/systems/<system-id>/enrollment-token`}</pre>
              </div>
            </div>

            <div style={s.step}>
              <div style={s.stepNum}>3</div>
              <div>
                <div style={s.stepTitle}>Choose platform and get install command</div>
                <div style={s.osBtns}>
                  {agent.os.map(o => (
                    <button key={o} onClick={()=>generateEnrollCmd(selected,o)} style={s.osBtn}>
                      {o}
                    </button>
                  ))}
                </div>
                {enrollCmd && <pre style={s.pre}>{enrollCmd}</pre>}
              </div>
            </div>

            <div style={s.version}>
              Current version: <span style={{color:'var(--accent)'}}>{VERSION}</span>
              {' — '}Build with: <code style={s.code}>go build -o opensourcebackup-agent ./cmd/agent</code>
            </div>
          </div>
        )
      })()}

      {/* Agent Health placeholder */}
      <div style={{marginTop:32}}>
        <SectionHeader title="Agent Health" />
        <div style={s.healthCard}>
          <p style={{color:'var(--text-muted)', fontSize:13}}>
            Agent health monitoring (last seen, version, status) will be shown here
            once agents are enrolled and connected.
          </p>
        </div>
      </div>
    </div>
  )
}

const s: Record<string,React.CSSProperties> = {
  page:         { padding:'28px 36px', maxWidth:1000 },
  intro:        { color:'var(--text-muted)', fontSize:13, marginBottom:16 },
  notice:       { background:'rgba(59,130,246,0.07)', border:'1px solid rgba(59,130,246,0.2)', borderRadius:'var(--radius)', padding:'10px 16px', fontSize:13, color:'var(--text-muted)', marginBottom:20 },
  code:         { fontFamily:'var(--font-mono)', fontSize:12, background:'rgba(0,0,0,0.3)', padding:'1px 5px', borderRadius:3 },
  grid:         { display:'grid', gridTemplateColumns:'repeat(3,1fr)', gap:12, marginBottom:20 },
  card:         { background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:'var(--radius)', padding:'16px', cursor:'pointer', transition:'all 0.12s' },
  cardSelected: { borderColor:'var(--accent)', background:'var(--accent-dim)' },
  cardIcon:     { fontSize:22, marginBottom:8 },
  cardTitle:    { fontWeight:700, color:'var(--text)', fontSize:13, marginBottom:4 },
  cardDesc:     { fontSize:12, color:'var(--text-muted)', lineHeight:1.5, marginBottom:8 },
  tags:         { display:'flex', gap:5, flexWrap:'wrap' as const },
  tag:          { background:'rgba(0,0,0,0.3)', color:'var(--text-dim)', padding:'1px 7px', borderRadius:4, fontSize:10, fontFamily:'var(--font-mono)' },
  panel:        { background:'var(--bg-card)', border:'1px solid var(--accent)', borderRadius:'var(--radius)', padding:'20px 24px', marginTop:8 },
  panelTitle:   { fontSize:16, fontWeight:700, color:'var(--text)', marginBottom:20 },
  step:         { display:'flex', gap:14, marginBottom:20 },
  stepNum:      { width:26, height:26, borderRadius:'50%', background:'var(--accent)', color:'#fff', display:'flex', alignItems:'center', justifyContent:'center', fontSize:12, fontWeight:700, flexShrink:0 },
  stepTitle:    { fontSize:13, fontWeight:600, color:'var(--text)', marginBottom:8 },
  pre:          { background:'#0a0d14', border:'1px solid var(--border)', borderRadius:6, padding:'12px 14px', fontSize:12, fontFamily:'var(--font-mono)', color:'#a8d8ea', overflow:'auto', whiteSpace:'pre', marginTop:8 },
  osBtns:       { display:'flex', gap:8, marginBottom:8 },
  osBtn:        { padding:'5px 12px', borderRadius:6, border:'1px solid var(--border)', background:'transparent', color:'var(--text-muted)', fontSize:12, cursor:'pointer', fontFamily:'var(--font-mono)' },
  version:      { marginTop:16, fontSize:12, color:'var(--text-dim)', borderTop:'1px solid var(--border)', paddingTop:12 },
  healthCard:   { background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:'var(--radius)', padding:'20px 24px' },
}
