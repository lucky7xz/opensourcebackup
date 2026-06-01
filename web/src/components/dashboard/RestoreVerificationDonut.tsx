import { useNavigate } from 'react-router-dom'
import type { Snapshot, RestoreTest } from '../../api'

interface Props { snapshots: Snapshot[]; restoreTests: RestoreTest[] }

export function RestoreVerificationDonut({ snapshots, restoreTests }: Props) {
  const nav = useNavigate()

  const verified = snapshots.filter(sn =>
    restoreTests.some(rt => rt.SnapshotID === sn.ID && rt.Status === 'success')
  ).length
  const failed = snapshots.filter(sn =>
    !restoreTests.some(rt => rt.SnapshotID === sn.ID && rt.Status === 'success') &&
     restoreTests.some(rt => rt.SnapshotID === sn.ID && rt.Status === 'failed')
  ).length
  const notTested = snapshots.length - verified - failed
  const total = snapshots.length

  // SVG donut
  const R = 44, C = 2 * Math.PI * R
  const segs = [
    { v: verified,  c: 'var(--success)', l: 'Verified',   pct: total > 0 ? Math.round(verified/total*100) : 0 },
    { v: notTested, c: 'var(--text-dim)', l: 'Not Tested', pct: total > 0 ? Math.round(notTested/total*100) : 0 },
    { v: failed,    c: 'var(--error)',    l: 'Failed',     pct: total > 0 ? Math.round(failed/total*100) : 0 },
  ]
  const gap = total > 1 ? 2 : 0
  let offset = C / 4
  const arcs = segs.map(seg => {
    const len = total > 0 ? (seg.v / total) * C : 0
    const el = { ...seg, len, offset }
    offset -= (len + (seg.v > 0 ? gap : 0))
    return el
  })

  return (
    <div className="dash-card" style={s.card}>
      <div style={s.header}>
        <span style={s.title}>Restore Verification</span>
        <button style={s.link} onClick={() => nav('/restore-tests')}>View all →</button>
      </div>

      {total === 0 ? (
        <div style={s.empty}>
          <div style={{ fontSize: 28, marginBottom: 8 }}>🔄</div>
          <div style={{ fontSize: 12, color: 'var(--text-dim)', textAlign: 'center' }}>
            Run a backup first,<br />then configure restore tests.
          </div>
        </div>
      ) : (
        <div style={s.body}>
          <div style={s.donutWrap}>
            <svg width={110} height={110} viewBox="0 0 110 110">
              <circle cx={55} cy={55} r={R} fill="none" stroke="var(--border)" strokeWidth={10} />
              {arcs.map((a, i) => a.len > 0 && (
                <circle key={i} cx={55} cy={55} r={R} fill="none"
                  stroke={a.c} strokeWidth={10}
                  strokeDasharray={`${Math.max(0,a.len-gap)} ${C}`}
                  strokeDashoffset={a.offset}
                  strokeLinecap="butt"
                />
              ))}
              <text x={55} y={50} textAnchor="middle" fontSize={20} fontWeight={800} fill="var(--text)">{total}</text>
              <text x={55} y={64} textAnchor="middle" fontSize={9} fill="var(--text-muted)">Snapshots</text>
            </svg>
          </div>

          <div style={s.legend}>
            {segs.map(seg => (
              <div key={seg.l} style={s.legendRow}>
                <span style={{ width:9, height:9, borderRadius:'50%', background:seg.c, flexShrink:0, display:'block' }} />
                <span style={{ fontSize:12, color:'var(--text-muted)', flex:1 }}>{seg.l}</span>
                <span style={{ fontSize:12, fontWeight:700, color:'var(--text)' }}>{seg.v}</span>
                <span style={{ fontSize:10, color:'var(--text-dim)', minWidth:34, textAlign:'right' as const }}>({seg.pct}%)</span>
              </div>
            ))}
          </div>

          {(notTested > 0 || failed > 0) && (
            <div style={s.notice}>
              {notTested > 0 && <span style={{ color:'var(--warning)', fontWeight:600 }}>{notTested} not tested</span>}
              {notTested > 0 && failed > 0 && ' · '}
              {failed > 0 && <span style={{ color:'var(--error)', fontWeight:600 }}>{failed} failed</span>}
              {' — '}
              <button style={s.noticeLink} onClick={() => nav('/restore-tests')}>schedule tests →</button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

const s: Record<string, React.CSSProperties> = {
  card:       { display: 'flex', flexDirection: 'column' },
  header:     { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '14px 18px 10px', borderBottom: '1px solid var(--border)' },
  title:      { fontSize: 13, fontWeight: 700 },
  link:       { background: 'none', border: 'none', color: 'var(--accent)', fontSize: 11, cursor: 'pointer', padding: 0 },
  empty:      { display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: '28px 20px' },
  body:       { padding: '16px 18px', display: 'flex', flexDirection: 'column', gap: 14 },
  donutWrap:  { display: 'flex', justifyContent: 'center' },
  legend:     { display: 'flex', flexDirection: 'column', gap: 8 },
  legendRow:  { display: 'flex', alignItems: 'center', gap: 8 },
  notice:     { fontSize: 11, color: 'var(--text-muted)', background: 'rgba(245,158,11,0.07)', border: '1px solid rgba(245,158,11,0.2)', borderRadius: 6, padding: '7px 10px' },
  noticeLink: { background: 'none', border: 'none', color: 'var(--accent)', fontSize: 11, cursor: 'pointer', padding: 0 },
}
