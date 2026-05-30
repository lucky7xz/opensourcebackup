import type { ReactNode, CSSProperties } from 'react'

interface Col<T> { header: string; render: (r:T) => ReactNode; width?: string; style?: CSSProperties }

export function Table<T>({ cols, rows, keyFn, empty='No data' }: {
  cols: Col<T>[]; rows: T[]; keyFn: (r:T) => string; empty?: string
}) {
  return (
    <div style={{ overflowX:'auto' }}>
      <table style={{ width:'100%', borderCollapse:'collapse', fontSize:13 }}>
        <thead>
          <tr>
            {cols.map(c => (
              <th key={c.header} style={{ ...th, width:c.width }}>{c.header}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0
            ? <tr><td colSpan={cols.length} style={emptyCell}>{empty}</td></tr>
            : rows.map(r => (
              <tr key={keyFn(r)} style={{ borderBottom:'1px solid var(--border)' }}>
                {cols.map(c => <td key={c.header} style={{ ...td, ...c.style }}>{c.render(r)}</td>)}
              </tr>
            ))
          }
        </tbody>
      </table>
    </div>
  )
}

const th: CSSProperties = {
  background:'rgba(30,36,54,0.8)', color:'var(--text-dim)', padding:'9px 14px',
  textAlign:'left', fontWeight:600, fontSize:11, textTransform:'uppercase',
  letterSpacing:'0.08em', borderBottom:'1px solid var(--border)', whiteSpace:'nowrap',
}
const td: CSSProperties = { padding:'10px 14px', color:'var(--text-muted)', verticalAlign:'middle' }
const emptyCell: CSSProperties = { padding:'36px', textAlign:'center', color:'var(--text-dim)', fontSize:13 }
