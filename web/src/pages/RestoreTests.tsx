import { SectionHeader } from '../components/Card'

export function RestoreTests() {
  return (
    <div style={s.page}>
      <SectionHeader title="Restore Tests" />

      <div style={s.hero}>
        <div style={s.heroIcon}>✓</div>
        <h2 style={s.heroTitle}>Restore Verification</h2>
        <p style={s.heroSub}>
          This is where you prove your backups work.<br />
          A backup is only as good as the last successful restore.
        </p>
      </div>

      <div style={s.grid}>
        <div style={s.card}>
          <div style={s.cardIcon}>📦</div>
          <h3 style={s.cardTitle}>What restore tests do</h3>
          <ul style={s.list}>
            <li>Select a snapshot</li>
            <li>Restore to a sandbox target path</li>
            <li>Verify file count and checksums</li>
            <li>Report: verified, failed, or error</li>
          </ul>
        </div>
        <div style={s.card}>
          <div style={s.cardIcon}>🗓</div>
          <h3 style={s.cardTitle}>Planned: B13 Restore Test Model</h3>
          <ul style={s.list}>
            <li>restore_tests table in PostgreSQL</li>
            <li>API: POST /v1/restore-tests</li>
            <li>Agent: runs restore to temp dir</li>
            <li>Reports verified_files, verified_bytes</li>
          </ul>
        </div>
        <div style={s.card}>
          <div style={s.cardIcon}>⚙</div>
          <h3 style={s.cardTitle}>Planned: B14 Restore Runner</h3>
          <ul style={s.list}>
            <li>restic restore to sandbox path</li>
            <li>File count verification</li>
            <li>Checksum validation</li>
            <li>Automatic scheduling</li>
          </ul>
        </div>
      </div>

      <div style={s.notice}>
        <strong>Coming in B13 + B14.</strong> Until then, restore tests must be run manually
        and results recorded outside the system.
      </div>
    </div>
  )
}

const s: Record<string,React.CSSProperties> = {
  page:      { padding:'28px 36px', maxWidth:900 },
  hero:      { textAlign:'center', padding:'40px 20px', background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:'var(--radius)', marginBottom:24 },
  heroIcon:  { fontSize:40, color:'var(--success)', marginBottom:12 },
  heroTitle: { fontSize:22, fontWeight:700, color:'var(--text)', marginBottom:8 },
  heroSub:   { fontSize:14, color:'var(--text-muted)', lineHeight:1.7 },
  grid:      { display:'grid', gridTemplateColumns:'repeat(3,1fr)', gap:14, marginBottom:20 },
  card:      { background:'var(--bg-card)', border:'1px solid var(--border)', borderRadius:'var(--radius)', padding:'18px 20px' },
  cardIcon:  { fontSize:24, marginBottom:10 },
  cardTitle: { fontSize:13, fontWeight:700, color:'var(--text)', marginBottom:10 },
  list:      { paddingLeft:16, color:'var(--text-muted)', fontSize:13, lineHeight:2 },
  notice:    { background:'rgba(245,158,11,0.07)', border:'1px solid rgba(245,158,11,0.2)', borderRadius:'var(--radius)', padding:'12px 16px', fontSize:13, color:'var(--text-muted)' },
}
