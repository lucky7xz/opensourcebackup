import { describe, it, expect } from 'vitest'
import { assessRestorePath } from './pathSafety'

describe('assessRestorePath', () => {
  it('treats an empty path as the recommended auto-sandbox', () => {
    expect(assessRestorePath('').risk).toBe('empty')
    expect(assessRestorePath('   ').risk).toBe('empty')
  })

  it('flags filesystem roots and bare drives as danger', () => {
    expect(assessRestorePath('/').risk).toBe('danger')
    expect(assessRestorePath('C:').risk).toBe('danger')
    expect(assessRestorePath('C:\\').risk).toBe('danger')
    expect(assessRestorePath('D:/').risk).toBe('danger')
  })

  it('flags protected system directories as danger', () => {
    const dangerous = [
      'C:/Windows',
      'C:\\Windows\\System32',
      'C:/Program Files/app',
      'C:/Users',
      'C:/ProgramData',
      '/etc',
      '/etc/ssh',
      '/usr/bin',
      '/var/log',
      '/home',
    ]
    for (const p of dangerous) {
      expect(assessRestorePath(p).risk, p).toBe('danger')
    }
  })

  it("allows the agent's own restore sandbox even under ProgramData", () => {
    expect(assessRestorePath('C:/ProgramData/opensourcebackup/restore-tests/abc').risk).not.toBe('danger')
    expect(assessRestorePath('C:\\ProgramData\\opensourcebackup\\restore-tests\\abc').risk).toBe('safe')
  })

  it('marks sandbox-looking paths as safe', () => {
    expect(assessRestorePath('C:/tmp/restore-test').risk).toBe('safe')
    expect(assessRestorePath('/tmp/restore-sandbox').risk).toBe('safe')
    expect(assessRestorePath('D:/scratch/verify').risk).toBe('safe')
  })

  it('marks an arbitrary custom path as caution', () => {
    expect(assessRestorePath('D:/Projekte/foo').risk).toBe('caution')
    expect(assessRestorePath('E:/data/customer').risk).toBe('caution')
  })

  it('is case-insensitive and trims input', () => {
    expect(assessRestorePath('  c:/WINDOWS  ').risk).toBe('danger')
    expect(assessRestorePath('C:/TMP/Restore').risk).toBe('safe')
  })

  it('always returns a non-empty title and detail', () => {
    for (const p of ['', '/', 'C:/Windows', 'C:/tmp/restore', 'D:/x']) {
      const a = assessRestorePath(p)
      expect(a.title.length).toBeGreaterThan(0)
      expect(a.detail.length).toBeGreaterThan(0)
    }
  })
})
