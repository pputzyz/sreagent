import type { AlertSeverity, AlertEventStatus, AlertRuleStatus, DataSourceStatus, TimelineAction } from '@/types'

// ===== Severity Helpers =====

/**
 * Map alert severity to Naive UI NTag `type` prop.
 */
export function getSeverityType(severity: string): 'error' | 'warning' | 'info' | 'default' {
  switch (severity) {
    case 'p0':
    case 'critical': return 'error'
    case 'p1':
    case 'p2':
    case 'warning':  return 'warning'
    case 'p3':
    case 'info':     return 'info'
    case 'p4':       return 'default'
    default:         return 'default'
  }
}

/**
 * Map alert severity to a hex color string.
 */
export function getSeverityColor(severity: string): string {
  switch (severity) {
    case 'p0':
    case 'critical': return '#ef4444'
    case 'p1':       return '#f97316'
    case 'p2':
    case 'warning':  return '#f59e0b'
    case 'p3':
    case 'info':     return '#3b82f6'
    case 'p4':       return '#6b7280'
    default:         return '#999'
  }
}

/**
 * Return a CSS class name for table row highlighting by severity.
 */
export function severityRowClass(row: { severity?: string }): string {
  const s = row.severity
  if (s === 'p0' || s === 'critical') return 'row-critical'
  if (s === 'p1' || s === 'p2' || s === 'warning') return 'row-warning'
  return ''
}

// ===== Alert Event Status Helpers =====

/**
 * Map alert event status to Naive UI NTag `type` prop.
 */
export function getEventStatusType(status: string): 'error' | 'warning' | 'info' | 'success' | 'default' {
  switch (status) {
    case 'firing':       return 'error'
    case 'acknowledged': return 'warning'
    case 'assigned':     return 'info'
    case 'resolved':     return 'success'
    case 'closed':       return 'default'
    case 'silenced':     return 'default'
    default:             return 'default'
  }
}

/**
 * Map alert event status to a hex color string.
 */
export function getStatusColor(status: string): string {
  switch (status) {
    case 'firing':       return '#ef4444'
    case 'acknowledged': return '#f59e0b'
    case 'assigned':     return '#3b82f6'
    case 'resolved':     return '#10b981'
    case 'closed':       return '#666666'
    case 'silenced':     return '#a78bfa'
    default:             return '#999'
  }
}

/**
 * Build an NTag `color` prop object from a hex color for a subtle transparent look.
 * Returns `{ color: hex + '18', textColor: hex, borderColor: 'transparent' }`
 */
export function statusTagColor(status: string) {
  const hex = getStatusColor(status)
  return { color: hex + '18', textColor: hex, borderColor: 'transparent' }
}

/**
 * i18n key map for alert event statuses.
 */
const statusLabelKeys: Record<string, string> = {
  firing:       'alert.firing',
  acknowledged: 'alert.acknowledged',
  assigned:     'alert.assigned',
  resolved:     'alert.resolved',
  closed:       'alert.closed',
  silenced:     'alert.silenced',
}

/**
 * Return the i18n key for a given alert event status, or the status string itself.
 */
export function getStatusLabelKey(status: string): string {
  return statusLabelKeys[status] || status
}

// ===== Alert Rule Status Helpers =====

/**
 * Map alert rule status to Naive UI NTag `type` prop.
 */
export function getRuleStatusType(status: string): 'success' | 'default' | 'warning' {
  switch (status) {
    case 'enabled':  return 'success'
    case 'disabled': return 'default'
    case 'muted':    return 'warning'
    default:         return 'default'
  }
}

// ===== Datasource Status Helpers =====

/**
 * Map datasource health status to Naive UI NTag `type` prop.
 */
export function getDatasourceStatusType(status: string): 'success' | 'error' | 'warning' {
  switch (status) {
    case 'healthy':   return 'success'
    case 'unhealthy': return 'error'
    default:          return 'warning'
  }
}

// ===== Timeline Helpers =====

/**
 * Map timeline action to Naive UI NTag/Timeline `type` prop.
 */
export function getTimelineType(action: string): 'error' | 'warning' | 'info' | 'success' | 'default' {
  switch (action) {
    case 'created':      return 'error'
    case 'resolved':     return 'success'
    case 'closed':       return 'default'
    case 'acknowledged': return 'warning'
    case 'assigned':     return 'info'
    case 'escalated':    return 'error'
    case 'commented':    return 'info'
    default:             return 'info'
  }
}

// ===== Row highlight CSS (global injection via <style> tag) =====

const ROW_HIGHLIGHT_ID = 'sre-row-highlight-styles'

/**
 * Inject shared row-highlight CSS classes into the document head once.
 * Safe to call multiple times — only injects once.
 */
export function injectRowHighlightCSS(): void {
  if (typeof document === 'undefined') return
  if (document.getElementById(ROW_HIGHLIGHT_ID)) return
  const style = document.createElement('style')
  style.id = ROW_HIGHLIGHT_ID
  style.textContent = `
.row-critical { background-color: rgba(239, 68, 68, 0.04); }
.row-warning  { background-color: rgba(245, 158, 11, 0.04); }
`
  document.head.appendChild(style)
}

/**
 * @deprecated Use `injectRowHighlightCSS()` instead for global CSS injection.
 */
export const ROW_HIGHLIGHT_CSS = `
.row-critical { background-color: rgba(239, 68, 68, 0.04); }
.row-warning  { background-color: rgba(245, 158, 11, 0.04); }
`
