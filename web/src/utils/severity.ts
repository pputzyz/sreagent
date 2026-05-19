import i18n from '@/i18n'

const t = i18n.global.t

export function severityLabel(sev: string): string {
  const key: Record<string, string> = {
    critical: 'severity.critical',
    warning: 'severity.warning',
    info: 'severity.info',
    p0: 'severity.p0',
    p1: 'severity.p1',
    p2: 'severity.p2',
    p3: 'severity.p3',
    p4: 'severity.p4',
  }
  return key[sev] ? t(key[sev]) : sev
}

export function severityType(sev: string): 'error' | 'warning' | 'info' | 'success' {
  if (sev === 'critical' || sev === 'p0' || sev === 'p1') return 'error'
  if (sev === 'warning' || sev === 'p2') return 'warning'
  if (sev === 'info' || sev === 'p4') return 'info'
  return 'info'
}
