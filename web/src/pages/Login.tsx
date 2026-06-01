import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

const BASE = import.meta.env.VITE_API_URL || ''

function csrfToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)osb_csrf=([^;]+)/)
  return match ? decodeURIComponent(match[1]) : ''
}

export function Login() {
  const nav = useNavigate()
  const [email,    setEmail]    = useState('')
  const [password, setPassword] = useState('')
  const [error,    setError]    = useState('')
  const [loading,  setLoading]  = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true); setError('')
    try {
      const res = await fetch(`${BASE}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken() },
        body: JSON.stringify({ email, password }),
      })
      if (res.ok) {
        nav('/', { replace: true })
      } else {
        const d = await res.json().catch(() => ({}))
        setError(d.error ?? 'Invalid credentials')
      }
    } catch {
      setError('Cannot reach the control plane')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={s.page}>
      <div style={s.card}>
        {/* Logo */}
        <div style={s.logo}>
          <div style={s.logoIcon}>OSB</div>
          <div>
            <div style={s.logoName}>OpenSourceBackup</div>
            <div style={s.logoSub}>Restore Assured</div>
          </div>
        </div>

        <h2 style={s.title}>Sign in to your account</h2>
        <p style={s.sub}>Enter your credentials to access the dashboard</p>

        <form onSubmit={submit} style={s.form}>
          <div style={s.field}>
            <label style={s.label}>Email address</label>
            <input
              style={s.input}
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              placeholder="admin@example.com"
              autoFocus
              required
            />
          </div>

          <div style={s.field}>
            <label style={s.label}>Password</label>
            <input
              style={s.input}
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              placeholder="••••••••••"
              required
            />
          </div>

          {error && <div style={s.error}>{error}</div>}

          <button type="submit" disabled={loading} style={s.btn}>
            {loading ? 'Signing in…' : 'Sign in'}
          </button>
        </form>

        <div style={s.footer}>
          <span style={{ color:'var(--text-dim)', fontSize:11 }}>
            Creating backups is easy. Proving recoverability is the difference.
          </span>
        </div>
      </div>
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  page:     { minHeight:'100vh', display:'flex', alignItems:'center', justifyContent:'center', background:'var(--bg)', padding:24 },
  card:     { width:'100%', maxWidth:400, background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:16, padding:'36px 40px', boxShadow:'0 8px 40px rgba(0,0,0,0.4)' },
  logo:     { display:'flex', alignItems:'center', gap:12, marginBottom:28 },
  logoIcon: { width:40, height:40, borderRadius:10, background:'linear-gradient(135deg,var(--accent) 0%,#5a8f1a 100%)', display:'flex', alignItems:'center', justifyContent:'center', fontSize:13, fontWeight:800, color:'#fff', boxShadow:'0 0 16px rgba(137,189,40,0.3)' },
  logoName: { fontSize:14, fontWeight:700, color:'var(--text)' },
  logoSub:  { fontSize:10, color:'var(--text-dim)' },
  title:    { fontSize:20, fontWeight:700, color:'var(--text)', marginBottom:6 },
  sub:      { fontSize:13, color:'var(--text-muted)', marginBottom:24 },
  form:     { display:'flex', flexDirection:'column', gap:16 },
  field:    { display:'flex', flexDirection:'column', gap:6 },
  label:    { fontSize:12, fontWeight:600, color:'var(--text-muted)' },
  input:    { padding:'10px 14px', background:'var(--bg)', border:'1px solid var(--border)', borderRadius:8, color:'var(--text)', fontSize:14, outline:'none' },
  error:    { background:'rgba(239,68,68,0.1)', border:'1px solid rgba(239,68,68,0.3)', borderRadius:8, padding:'10px 14px', fontSize:13, color:'var(--error)' },
  btn:      { padding:'11px', borderRadius:8, background:'var(--accent)', color:'#fff', border:'none', fontSize:14, fontWeight:700, cursor:'pointer', marginTop:4 },
  footer:   { marginTop:24, textAlign:'center' as const },
}
