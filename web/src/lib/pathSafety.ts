// Advisory restore-target path checks (B_RESTORE_REPO_SELECT guardrails).
//
// Pure, UI-facing heuristic — the authoritative check is the agent's
// validateRestorePath (sandbox root + no filesystem roots). This module only
// warns the operator *before* submitting so a dangerous path is caught early
// and an empty (auto-sandbox) target is encouraged.

export type PathRisk = 'empty' | 'safe' | 'caution' | 'danger'

export interface PathAssessment {
  risk:   PathRisk
  title:  string
  detail: string
}

const SANDBOX_HINTS = ['restore', 'sandbox', 'tmp', 'temp', 'scratch', 'test']

// Protected locations a restore must never target. Matched against a normalised
// (forward-slash, lower-case, no trailing slash) path.
const DANGEROUS_PREFIXES = [
  '/etc', '/usr', '/bin', '/sbin', '/boot', '/lib', '/var', '/root',
  '/sys', '/proc', '/dev', '/home',
  'c:/windows', 'c:/program files', 'c:/program files (x86)',
  'c:/users', 'c:/programdata',
]

function normalise(raw: string): string {
  const n = raw.trim().replace(/\\/g, '/').toLowerCase().replace(/\/+$/, '')
  return n === '' ? '' : n
}

export function assessRestorePath(raw: string): PathAssessment {
  if (raw.trim() === '') {
    return {
      risk: 'empty',
      title: 'Automatischer Sandbox-Pfad',
      detail: 'Leer lassen wird empfohlen — der Agent restauriert in sein isoliertes Sandbox-Verzeichnis.',
    }
  }

  const norm = normalise(raw)

  // Filesystem root or a bare drive letter (e.g. "/", "C:", "C:/")
  if (norm === '/' || /^[a-z]:$/.test(norm)) {
    return {
      risk: 'danger',
      title: 'Dateisystem-Wurzel',
      detail: 'Ein Restore in ein Wurzelverzeichnis ist nicht erlaubt und würde vom Agent abgelehnt.',
    }
  }

  // The agent's own sandbox lives under .../opensourcebackup/restore-tests —
  // that is explicitly safe even though it sits under c:/programdata.
  const isAgentSandbox = norm.includes('opensourcebackup/restore')

  if (!isAgentSandbox) {
    for (const p of DANGEROUS_PREFIXES) {
      if (norm === p || norm.startsWith(p + '/')) {
        return {
          risk: 'danger',
          title: 'Geschütztes System-Verzeichnis',
          detail: `„${raw.trim()}" liegt in einem System-Verzeichnis. Vorhandene Daten könnten überschrieben werden — der Agent akzeptiert nur Pfade in seiner Restore-Sandbox.`,
        }
      }
    }
  }

  if (SANDBOX_HINTS.some(h => norm.includes(h))) {
    return {
      risk: 'safe',
      title: 'Sieht nach Sandbox-Verzeichnis aus',
      detail: 'Stelle sicher, dass das Verzeichnis leer ist — vorhandene Dateien können überschrieben werden.',
    }
  }

  return {
    risk: 'caution',
    title: 'Eigener Zielpfad',
    detail: 'Verwende ein leeres Verzeichnis. Vorhandene Dateien können überschrieben werden, und der Agent akzeptiert nur Pfade unterhalb seines Restore-Sandbox-Roots.',
  }
}
