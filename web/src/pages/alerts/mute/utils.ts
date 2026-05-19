import type { MuteRule } from '@/types'

/** Determine the mute rule type based on which time fields are populated. */
export function ruleType(r: MuteRule): 'once' | 'periodic' | 'unknown' {
  if (r.start_time && r.end_time) return 'once'
  if (r.periodic_start && r.periodic_end) return 'periodic'
  return 'unknown'
}

/** Safely parse an ISO timestamp string to milliseconds, returning null on failure. */
export function toMs(t: string | null | undefined): number | null {
  if (!t) return null
  const ms = new Date(t).getTime()
  return Number.isNaN(ms) ? null : ms
}

/** Check whether a mute rule is currently active (muting right now). */
export function isActiveNow(r: MuteRule): boolean {
  if (!r.is_enabled) return false
  const now = Date.now()
  if (ruleType(r) === 'once') {
    const s = toMs(r.start_time), e = toMs(r.end_time)
    return !!(s && e && now >= s && now <= e)
  }
  if (ruleType(r) === 'periodic') {
    const d = new Date()
    const cur = d.getHours() * 60 + d.getMinutes()
    const [sh, sm] = (r.periodic_start || '0:0').split(':').map(Number)
    const [eh, em] = (r.periodic_end || '0:0').split(':').map(Number)
    const s = sh * 60 + sm, e = eh * 60 + em
    const inWindow = s <= e ? (cur >= s && cur <= e) : (cur >= s || cur <= e)
    if (!inWindow) return false
    const days = (r.days_of_week || '').split(',').map(x => x.trim()).filter(Boolean)
    if (days.length === 0) return true
    return days.includes(String(d.getDay()))
  }
  return false
}

/** Check whether a once-type mute rule is scheduled for the future. */
export function isFuture(r: MuteRule): boolean {
  if (!r.is_enabled) return false
  if (ruleType(r) !== 'once') return false
  const s = toMs(r.start_time)
  return !!(s && s > Date.now())
}

/** Check whether a once-type mute rule has already expired. */
export function isExpired(r: MuteRule): boolean {
  if (ruleType(r) !== 'once') return false
  const e = toMs(r.end_time)
  return !!(e && e < Date.now())
}

/** Map mute rule status to a severity-like tag for the status dot. */
export function statusToSev(r: MuteRule): string {
  if (!r.is_enabled) return 'muted'
  if (isActiveNow(r)) return 'success'
  if (isFuture(r)) return 'info'
  if (isExpired(r)) return 'muted'
  return 'info'
}

/** Return a status text key for the given mute rule (caller must translate). */
export function statusKey(r: MuteRule): 'mute.statusDisabled' | 'mute.statusActive' | 'mute.statusScheduled' | 'mute.statusExpired' | 'mute.statusIdle' {
  if (!r.is_enabled) return 'mute.statusDisabled'
  if (isActiveNow(r)) return 'mute.statusActive'
  if (isFuture(r)) return 'mute.statusScheduled'
  if (isExpired(r)) return 'mute.statusExpired'
  return 'mute.statusIdle'
}

/** Get the hit count from a mute rule (may be absent on the type). */
export function getHitCount(r: MuteRule): number {
  return (r as MuteRule & { hit_count?: number }).hit_count || 0
}

/** Compute remaining minutes until a once-type mute rule expires. */
export function remainingMin(r: MuteRule): number {
  const e = toMs(r.end_time)
  if (!e) return 0
  return Math.max(0, Math.round((e - Date.now()) / 60000))
}

/** Format a future timestamp as a relative duration string (e.g. "5m", "2h", "3d"). */
export function relTimeFuture(t: string | null): string {
  const ms = toMs(t)
  if (!ms) return '-'
  const diff = ms - Date.now()
  const m = Math.round(diff / 60000)
  if (m < 60) return `${m}m`
  const h = Math.round(m / 60)
  if (h < 24) return `${h}h`
  return `${Math.round(h / 24)}d`
}

/** Build a human-readable description of a periodic mute schedule. */
export function describePeriodic(r: MuteRule, dayMap: Record<string, string>, dailyLabel: string): string {
  const days = (r.days_of_week || '').split(',').map(x => x.trim()).filter(Boolean)
  const dayLabel = days.length ? days.map(d => dayMap[d] || d).join('/') : dailyLabel
  return `${dayLabel} ${r.periodic_start} - ${r.periodic_end} (${r.timezone || 'UTC'})`
}
