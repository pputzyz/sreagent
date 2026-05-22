/**
 * Format an ISO date string to a localized display string.
 * Uses the current document language to determine locale.
 */
export function formatTime(dateStr: string | null | undefined): string {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return '-'
  // Detect locale from <html lang="..."> or fallback to en-US
  const htmlLang = typeof document !== 'undefined' ? document.documentElement.lang : ''
  const locale = htmlLang === 'zh-CN' ? 'zh-CN' : 'en-US'
  return d.toLocaleString(locale, {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}

/**
 * Format a duration in seconds to a human-readable string.
 * e.g. 270 => "4m 30s", 3661 => "1h 1m 1s"
 */
export function formatDuration(seconds: number): string {
  if (seconds < 0) return '0s'
  if (seconds === 0) return '0s'

  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)

  const parts: string[] = []
  if (days > 0) parts.push(`${days}d`)
  if (hours > 0) parts.push(`${hours}h`)
  if (minutes > 0) parts.push(`${minutes}m`)
  if (secs > 0 || parts.length === 0) parts.push(`${secs}s`)

  return parts.join(' ')
}

/**
 * Format a timestamp as a relative time string (e.g. "5m ago", "2h ago", "3d ago").
 * Returns "—" for missing input.
 * Optionally accepts a vue-i18n `t` function for localized output.
 */
export function relTime(ts?: string | null, t?: (key: string, params?: Record<string, unknown>) => string): string {
  if (!ts) return '—'
  const diff = Math.max(0, Date.now() - new Date(ts).getTime())
  const s = Math.floor(diff / 1000)
  if (s < 60) return t ? t('alert.secsAgo', { n: s }) : `${s}s ago`
  const m = Math.floor(diff / 60000)
  if (m < 60) return t ? t('alert.minsAgo', { n: m }) : `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return t ? t('alert.hoursAgo', { n: h }) : `${h}h ago`
  const d = Math.floor(h / 24)
  return t ? t('alert.daysAgo', { n: d }) : `${d}d ago`
}

/**
 * Convert an array of { key, value } pairs to a Record<string, string>.
 * Empty keys are silently skipped; keys are trimmed.
 */
export function kvArrayToRecord(arr: { key: string; value: string }[]): Record<string, string> {
  const record: Record<string, string> = {}
  for (const item of arr) {
    if (item.key.trim()) {
      record[item.key.trim()] = item.value
    }
  }
  return record
}

/**
 * Convert a Record<string, string> to an array of { key, value } pairs.
 * Useful for populating KVEditor from API data.
 */
export function recordToKVArray(record: Record<string, string> | null | undefined): { key: string; value: string }[] {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({ key, value }))
}

/**
 * Safely extract an error message from an unknown thrown value.
 * Works with Error instances, Axios errors, and plain objects with a `message` property.
 */
export function getErrorMessage(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'object' && e !== null && 'message' in e) {
    return String((e as { message: unknown }).message)
  }
  return String(e)
}
