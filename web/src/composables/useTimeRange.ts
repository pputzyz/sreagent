import { ref, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TimeRange, RelativeTimeOption, AutoRefreshOption } from '@/types/query'

export function useRelativeTimeOptions(): RelativeTimeOption[] {
  const { t } = useI18n()
  return [
    { label: t('timeRangeOptions.last5m'), value: '5m', ms: 5 * 60 * 1000 },
    { label: t('timeRangeOptions.last15m'), value: '15m', ms: 15 * 60 * 1000 },
    { label: t('timeRangeOptions.last30m'), value: '30m', ms: 30 * 60 * 1000 },
    { label: t('timeRangeOptions.last1h'), value: '1h', ms: 60 * 60 * 1000 },
    { label: t('timeRangeOptions.last3h'), value: '3h', ms: 3 * 60 * 60 * 1000 },
    { label: t('timeRangeOptions.last6h'), value: '6h', ms: 6 * 60 * 60 * 1000 },
    { label: t('timeRangeOptions.last12h'), value: '12h', ms: 12 * 60 * 60 * 1000 },
    { label: t('timeRangeOptions.last24h'), value: '24h', ms: 24 * 60 * 60 * 1000 },
    { label: t('timeRangeOptions.last7d'), value: '7d', ms: 7 * 24 * 60 * 60 * 1000 },
    { label: t('timeRangeOptions.last30d'), value: '30d', ms: 30 * 24 * 60 * 60 * 1000 },
  ]
}

/**
 * @deprecated Hardcoded English labels — use `useRelativeTimeOptions()` composable
 * for i18n-aware reactive options. Kept only for non-reactive / non-i18n contexts.
 */
export const relativeTimeOptions: RelativeTimeOption[] = [
  { label: 'Last 5 minutes', value: '5m', ms: 5 * 60 * 1000 },
  { label: 'Last 15 minutes', value: '15m', ms: 15 * 60 * 1000 },
  { label: 'Last 30 minutes', value: '30m', ms: 30 * 60 * 1000 },
  { label: 'Last 1 hour', value: '1h', ms: 60 * 60 * 1000 },
  { label: 'Last 3 hours', value: '3h', ms: 3 * 60 * 60 * 1000 },
  { label: 'Last 6 hours', value: '6h', ms: 6 * 60 * 60 * 1000 },
  { label: 'Last 12 hours', value: '12h', ms: 12 * 60 * 60 * 1000 },
  { label: 'Last 24 hours', value: '24h', ms: 24 * 60 * 60 * 1000 },
  { label: 'Last 7 days', value: '7d', ms: 7 * 24 * 60 * 60 * 1000 },
  { label: 'Last 30 days', value: '30d', ms: 30 * 24 * 60 * 60 * 1000 },
]

/**
 * @deprecated Hardcoded English labels — use `useTimeRange()` composable
 * for i18n-aware auto-refresh options. Kept only for non-reactive / non-i18n contexts.
 */
export const autoRefreshOptions: AutoRefreshOption[] = [
  { label: 'Off', value: null },
  { label: '5s', value: 5000 },
  { label: '10s', value: 10000 },
  { label: '30s', value: 30000 },
  { label: '1m', value: 60000 },
  { label: '5m', value: 300000 },
]

function computeRange(ms: number): TimeRange {
  const end = Date.now()
  return { start: end - ms, end }
}

export function useTimeRange(defaultDuration = '1h') {
  const relativeOptions = useRelativeTimeOptions()
  const defaultOpt = relativeOptions.find(o => o.value === defaultDuration) || relativeOptions.find(o => o.value === '1h')!
  const timeRange = ref<TimeRange>(computeRange(defaultOpt.ms))
  const isRelative = ref(true)
  const relativeDuration = ref(defaultDuration)
  const autoRefreshInterval = ref<number | null>(null)

  function setRelative(duration: string) {
    const opt = relativeOptions.find(o => o.value === duration)
    if (!opt) return
    isRelative.value = true
    relativeDuration.value = duration
    timeRange.value = computeRange(opt.ms)
  }

  function setAbsolute(start: number, end: number) {
    isRelative.value = false
    timeRange.value = { start, end }
  }

  function refresh() {
    if (isRelative.value) {
      const opt = relativeOptions.find(o => o.value === relativeDuration.value)
      if (opt) timeRange.value = computeRange(opt.ms)
    }
  }

  let timer: ReturnType<typeof setInterval> | null = null
  watch(autoRefreshInterval, (interval) => {
    if (timer) clearInterval(timer)
    if (interval) {
      timer = setInterval(refresh, interval)
    } else {
      timer = null
    }
  })

  onUnmounted(() => {
    if (timer) clearInterval(timer)
  })

  return {
    timeRange,
    isRelative,
    relativeDuration,
    autoRefreshInterval,
    setRelative,
    setAbsolute,
    refresh,
  }
}
