import type { ReactNode } from 'react'

interface Column<T> {
  header: string
  render: (row: T) => ReactNode
  width?: string
}

interface TableProps<T> {
  columns: Column<T>[]
  rows: T[]
  keyFn: (row: T) => string
  emptyMsg?: string
}

export function Table<T>({ columns, rows, keyFn, emptyMsg = 'No data' }: TableProps<T>) {
  return (
    <div style={{ overflowX: 'auto' }}>
      <table style={styles.table}>
        <thead>
          <tr>
            {columns.map(c => (
              <th key={c.header} style={{ ...styles.th, width: c.width }}>{c.header}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr><td colSpan={columns.length} style={styles.empty}>{emptyMsg}</td></tr>
          ) : rows.map(row => (
            <tr key={keyFn(row)} style={styles.tr}>
              {columns.map(c => (
                <td key={c.header} style={styles.td}>{c.render(row)}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  table: {
    width: '100%',
    borderCollapse: 'collapse',
    fontSize: 13,
  },
  th: {
    background: 'var(--bg-card)',
    color: 'var(--text-secondary)',
    padding: '10px 16px',
    textAlign: 'left',
    fontWeight: 600,
    fontSize: 11,
    textTransform: 'uppercase',
    letterSpacing: '0.06em',
    borderBottom: '1px solid var(--border)',
    whiteSpace: 'nowrap',
  },
  tr: { borderBottom: '1px solid var(--border)' },
  td: {
    padding: '11px 16px',
    color: 'var(--text-secondary)',
    verticalAlign: 'middle',
  },
  empty: {
    padding: '32px 16px',
    textAlign: 'center',
    color: 'var(--text-secondary)',
    fontSize: 13,
  },
}
