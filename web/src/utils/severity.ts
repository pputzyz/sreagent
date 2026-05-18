export function severityLabel(sev: string): string {
  const map: Record<string, string> = {
    critical: '严重', warning: '警告', info: '提示',
    p0: 'P0', p1: 'P1', p2: 'P2', p3: 'P3', p4: 'P4',
  }
  return map[sev] || sev
}

export function severityType(sev: string): 'error' | 'warning' | 'info' | 'success' {
  if (sev === 'critical' || sev === 'p0' || sev === 'p1') return 'error'
  if (sev === 'warning' || sev === 'p2') return 'warning'
  if (sev === 'info' || sev === 'p4') return 'info'
  return 'info'
}
