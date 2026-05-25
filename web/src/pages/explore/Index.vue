<script setup lang="ts">
/**
 * Data Query Page — unified metrics + logs query interface.
 *
 * Inspired by Nightingale Explorer + Grafana Explore:
 *  - Log histogram with click-to-zoom (Nightingale HistogramChart pattern)
 *  - Log level coloring with left border (Nightingale Loki LogRow pattern)
 *  - Row expansion for full log detail (Nightingale LogsViewer pattern)
 *  - Field value click-to-filter (Nightingale FieldValueWithFilter pattern)
 *  - URL querystring sync (Nightingale Explorer pattern)
 *  - Query stats display (Nightingale PromGraph QueryStatsView pattern)
 *  - Multi-panel support (Nightingale Metric panels pattern)
 *  - Compact horizontal preset time buttons + custom range
 *  - Query history per datasource (Nightingale HistoricalRecords pattern)
 */
import { ref, onMounted, onUnmounted, computed, watch, shallowRef, h, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  NSelect, NButton, NSpace, NTag, NAlert, NSpin,
  NDataTable, NTabs, NTabPane, NDatePicker,
  NPopover, NIcon, NTooltip, NButtonGroup, NDrawer, NDrawerContent,
  NDescriptions, NDescriptionsItem,
  useMessage,
} from 'naive-ui'
import {
  RefreshOutline, TimeOutline, TrashOutline, DownloadOutline,
  AlertCircleOutline, AddOutline,
} from '@vicons/ionicons5'
import { datasourceApi } from '@/api'
import { formatTime } from '@/utils/format'
import PromQLEditor from '@/components/query/PromQLEditor.vue'
import LogsQLEditor from '@/components/query/LogsQLEditor.vue'
import LogHistogram from '@/components/query/LogHistogram.vue'
import type { DataSource, DataSourceType, QueryResponse, LogEntry } from '@/types'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

// --- Query mode (instant / range) ---
type QueryMode = 'instant' | 'range'
const queryMode = ref<QueryMode>('range')

// --- Label detail drawer ---
const labelDrawerVisible = ref(false)
const labelDrawerData = ref<Record<string, string>>({})
const labelDrawerSeriesName = ref('')

// --- Lazy ECharts ---
const ChartReady = ref(false)
const VChart = shallowRef<Component | null>(null)

async function loadECharts() {
  try {
    const [{ use }, { CanvasRenderer }, { LineChart }, components, vc] = await Promise.all([
      import('echarts/core'),
      import('echarts/renderers'),
      import('echarts/charts'),
      import('echarts/components'),
      import('vue-echarts'),
    ])
    use([
      CanvasRenderer, LineChart,
      components.TooltipComponent, components.LegendComponent,
      components.GridComponent, components.DataZoomComponent,
    ])
    VChart.value = vc.default
    ChartReady.value = true
  } catch (e) {
    console.warn('[DataQuery] ECharts load failed:', e)
  }
}

type ResultMode = 'chart' | 'table'
type QueryTab = 'metrics' | 'logs'

// --- state ---
const datasources = ref<DataSource[]>([])
const selectedDsId = ref<number | null>(null)
const expression = ref('')
const loading = ref(false)
const errorMsg = ref('')
const logEntries = ref<LogEntry[]>([])
const metricData = ref<QueryResponse | null>(null)
const logTotal = ref(0)
const logTruncated = ref(false)
const resultMode = ref<ResultMode>('chart')
const activeTab = ref<QueryTab>('metrics')

// --- Log histogram ---
interface HistogramBucket { timestamp: string; count: number }
const histogramBuckets = ref<HistogramBucket[]>([])
const histogramLoading = ref(false)
const showHistogram = ref(true)

// --- Log level detection ---
type LogLevel = 'debug' | 'info' | 'warn' | 'error' | 'fatal' | 'unknown'
const LEVEL_COLORS: Record<LogLevel, string> = {
  debug: '#64748b',   // slate
  info: '#0d9488',    // teal
  warn: '#eab308',    // yellow
  error: '#ef4444',   // red
  fatal: '#dc2626',   // dark red
  unknown: 'transparent',
}

function detectLogLevel(entry: LogEntry): LogLevel {
  // Check explicit level field
  const level = (entry.labels?.level || entry.labels?.severity || entry.labels?.lvl || '').toString().toLowerCase()
  if (level.includes('error') || level.includes('err')) return 'error'
  if (level.includes('warn') || level.includes('wrn')) return 'warn'
  if (level.includes('info') || level.includes('inf')) return 'info'
  if (level.includes('debug') || level.includes('dbg') || level.includes('trace')) return 'debug'
  if (level.includes('fatal') || level.includes('crit') || level.includes('panic')) return 'fatal'
  // Fallback: check message content
  const msg = (entry.message || '').toLowerCase()
  if (msg.includes('error') || msg.includes('exception') || msg.includes('fail')) return 'error'
  if (msg.includes('warn')) return 'warn'
  return 'unknown'
}

// --- Row expansion ---
const expandedRowKeys = ref<number[]>([])

// --- Query stats ---
interface QueryStats { executionTimeMs: number; resultCount: number; step?: string }
const queryStats = ref<QueryStats | null>(null)


// --- time range ---
// rangeMin: minutes; -1 = custom
const rangeMin = ref<number>(60)
const customRange = ref<[number, number] | null>(null)
const now = ref(Date.now())

const presetOptions = [
  { label: '5m', value: 5 },
  { label: '15m', value: 15 },
  { label: '30m', value: 30 },
  { label: '1h', value: 60 },
  { label: '3h', value: 180 },
  { label: '6h', value: 360 },
  { label: '12h', value: 720 },
  { label: '24h', value: 1440 },
  { label: '2d', value: 2880 },
  { label: '7d', value: 10080 },
  { label: '30d', value: 43200 },
]

const showCustomPicker = ref(false)

const timeStart = computed(() => {
  if (rangeMin.value === -1 && customRange.value) {
    return Math.floor(customRange.value[0] / 1000)
  }
  return Math.floor((now.value - rangeMin.value * 60000) / 1000)
})
const timeEnd = computed(() => {
  if (rangeMin.value === -1 && customRange.value) {
    return Math.floor(customRange.value[1] / 1000)
  }
  return Math.floor(now.value / 1000)
})

const rangeDisplay = computed(() => {
  const s = new Date(timeStart.value * 1000).toLocaleString()
  const e = new Date(timeEnd.value * 1000).toLocaleString()
  return `${s} → ${e}`
})

function selectPreset(v: number) {
  rangeMin.value = v
  showCustomPicker.value = false
  now.value = Date.now()
}

function openCustomRange() {
  rangeMin.value = -1
  showCustomPicker.value = true
  if (!customRange.value) {
    const n = Date.now()
    customRange.value = [n - 3600000, n]
  }
}

// --- step ---
const stepOptions = computed(() => [
  { label: t('query.stepAuto'), value: 'auto' },
  { label: '15s', value: '15s' },
  { label: '30s', value: '30s' },
  { label: '1m', value: '1m' },
  { label: '5m', value: '5m' },
  { label: '15m', value: '15m' },
  { label: '1h', value: '1h' },
])
const stepValue = ref<string>('auto')

function resolveStep(): string {
  if (stepValue.value !== 'auto') return stepValue.value
  const diff = timeEnd.value - timeStart.value
  return diff <= 3600 ? '15s' : diff <= 21600 ? '1m' : diff <= 86400 ? '5m' : '15m'
}

// --- limit ---
const metricLimit = ref<number>(100)
const metricLimitOptions = [50, 100, 200, 500, 1000].map(v => ({ label: String(v), value: v }))

const logLimit = ref<number>(200)
const logLimitOptions = [50, 100, 200, 500, 1000, 5000].map(v => ({ label: String(v), value: v }))

// --- auto-refresh ---
const autoRefreshOptions = computed(() => [
  { label: t('query.refreshOff'), value: 0 },
  { label: '5s', value: 5 },
  { label: '10s', value: 10 },
  { label: '30s', value: 30 },
  { label: '1min', value: 60 },
  { label: '5min', value: 300 },
])
const autoRefreshSec = ref<number>(0)
const autoCountdown = ref<number>(0)
let autoTimer: ReturnType<typeof setInterval> | null = null

function startAutoTimer() {
  stopAutoTimer()
  if (autoRefreshSec.value <= 0) return
  autoCountdown.value = autoRefreshSec.value
  autoTimer = setInterval(() => {
    autoCountdown.value -= 1
    if (autoCountdown.value <= 0) {
      run()
      autoCountdown.value = autoRefreshSec.value
    }
  }, 1000)
}

function stopAutoTimer() {
  if (autoTimer) {
    clearInterval(autoTimer)
    autoTimer = null
  }
  autoCountdown.value = 0
}

watch(autoRefreshSec, () => startAutoTimer())

// --- query history ---
type HistoryItem = { tab: QueryTab; expression: string; ts: number }
const HISTORY_KEY = 'sre-query-history'
const history = ref<HistoryItem[]>([])
const historyVisible = ref(false)

function loadHistory() {
  try {
    const raw = localStorage.getItem(HISTORY_KEY)
    if (raw) history.value = JSON.parse(raw) || []
  } catch { /* ignore */ }
}

function pushHistory(tab: QueryTab, expr: string) {
  if (!expr.trim()) return
  const list = history.value.filter(h => !(h.tab === tab && h.expression === expr))
  list.unshift({ tab, expression: expr, ts: Date.now() })
  history.value = list.slice(0, 20)
  try { localStorage.setItem(HISTORY_KEY, JSON.stringify(history.value)) } catch { /* ignore */ }
}

const filteredHistory = computed(() =>
  history.value.filter(h => h.tab === activeTab.value).slice(0, 10)
)

function applyHistory(item: HistoryItem) {
  expression.value = item.expression
  historyVisible.value = false
}

function clearExpression() {
  expression.value = ''
}

// --- computed ---
const selectedDs = computed(() => datasources.value.find(d => d.id === selectedDsId.value))
const metricDatasources = computed(() =>
  datasources.value.filter(d => d.supports_query && d.type !== 'victorialogs')
)
const logDatasources = computed(() =>
  datasources.value.filter(d => d.type === 'victorialogs')
)
const isLogs = computed(() => activeTab.value === 'logs')
const isMetricLimited = computed(() => {
  if (!metricData.value?.series) return false
  return metricData.value.series.length >= metricLimit.value
})

function dsLabel(ds: DataSource): string {
  return `${ds.name} (${typeBadge(ds.type)})`
}
function typeBadge(tp: DataSourceType): string {
  const m: Record<string, string> = {
    prometheus: 'Prometheus', victoriametrics: 'VictoriaMetrics',
    victorialogs: 'VictoriaLogs', zabbix: 'Zabbix',
  }
  return m[tp] || tp
}
const _dsColorCache: Record<string, string> = {}
let _themeObserver: MutationObserver | null = null

function typeColor(tp: DataSourceType): string {
  if (_dsColorCache[tp]) return _dsColorCache[tp]
  const tokenMap: Record<string, string> = {
    prometheus: '--sre-ds-prometheus', victoriametrics: '--sre-ds-victoriametrics',
    victorialogs: '--sre-ds-victorialogs', zabbix: '--sre-ds-zabbix',
  }
  const token = tokenMap[tp]
  if (token && typeof document !== 'undefined') {
    const val = getComputedStyle(document.documentElement).getPropertyValue(token).trim()
    _dsColorCache[tp] = val || '#64748b'
  } else {
    _dsColorCache[tp] = '#64748b'
  }
  return _dsColorCache[tp]
}

function setupThemeObserver() {
  if (typeof document === 'undefined') return
  _themeObserver = new MutationObserver(() => {
    for (const key in _dsColorCache) delete _dsColorCache[key]
  })
  _themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class', 'data-theme'] })
}

// --- actions ---
async function loadDs() {
  try {
    const res = await datasourceApi.list({ page: 1, page_size: 100 })
    const list = res.data?.data?.list
    datasources.value = (Array.isArray(list) ? list : []).filter((d: DataSource) => d.is_enabled)
  } catch (e) { console.warn('[Explore] Failed to load datasources:', e) }
}

async function run() {
  if (!selectedDsId.value || !expression.value.trim()) return
  if (rangeMin.value !== -1) now.value = Date.now()
  const startTime = Date.now()
  loading.value = true
  errorMsg.value = ''
  metricData.value = null
  logEntries.value = []
  try {
    if (isLogs.value) {
      const res = await datasourceApi.logQuery(selectedDsId.value, {
        expression: expression.value,
        start: timeStart.value,
        end: timeEnd.value,
        limit: logLimit.value,
      })
      const data = res.data?.data
      if (data) {
        logEntries.value = (data.entries || []).map((e: LogEntry, i: number) => ({ ...e, _key: i }))
        logTotal.value = data.total || 0
        logTruncated.value = data.truncated || false
      }
    } else if (queryMode.value === 'instant') {
      const res = await datasourceApi.query(selectedDsId.value, {
        expression: expression.value,
        time: timeEnd.value,
      })
      const data = res.data?.data
      if (data?.series && data.series.length > metricLimit.value) {
        data.series = data.series.slice(0, metricLimit.value)
      }
      metricData.value = data
    } else {
      const res = await datasourceApi.rangeQuery(selectedDsId.value, {
        expression: expression.value,
        start: timeStart.value,
        end: timeEnd.value,
        step: resolveStep(),
      })
      const data = res.data?.data
      if (data?.series && data.series.length > metricLimit.value) {
        data.series = data.series.slice(0, metricLimit.value)
        Object.defineProperty(data, '_limited', { value: true, enumerable: false })
      }
      metricData.value = data
    }
    queryStats.value = { executionTimeMs: Date.now() - startTime, resultCount: isLogs.value ? logEntries.value.length : (metricData.value?.series?.length || 0), step: isLogs.value ? undefined : resolveStep() }
    pushHistory(activeTab.value, expression.value)
    syncToURL()
    // Fetch histogram for log queries
    if (isLogs.value && showHistogram.value) {
      fetchHistogram()
    }
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string; message?: string } }; message?: string }
    errorMsg.value = err?.response?.data?.error || err?.response?.data?.message || err?.message || t('query.queryFailed')
  } finally {
    loading.value = false
  }
}

// --- chart option ---
const chartOption = computed(() => {
  if (!metricData.value?.series?.length) return null
  interface EChartsSeries {
    name: string
    type: string
    data: [number, number][]
    smooth: boolean
    showSymbol: boolean
    connectNulls: boolean
  }
  const seriesList: EChartsSeries[] = []
  for (const s of metricData.value.series) {
    const name = formatLegend(s.labels)
    const data: [number, number][] = []
    for (const v of s.values || []) {
      data.push([Number(v.ts) * 1000, v.value != null ? Number(v.value) : 0])
    }
    seriesList.push({ name, type: 'line', data, smooth: false, showSymbol: false, connectNulls: true })
  }
  return {
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis', confine: true },
    legend: { type: 'scroll', bottom: 0, textStyle: { color: (typeof document !== 'undefined' ? getComputedStyle(document.documentElement).getPropertyValue('--sre-text-tertiary').trim() || '#64748b' : '#64748b'), fontSize: 12 } },
    grid: { left: 80, right: 20, top: 20, bottom: 50 },
    xAxis: { type: 'time', axisLabel: { fontSize: 11 } },
    yAxis: { type: 'value', axisLabel: { fontSize: 11 }, splitLine: { lineStyle: { type: 'dashed' } } },
    series: seriesList,
    dataZoom: [
      { type: 'inside', start: 0, end: 100 },
      { type: 'slider', start: 0, end: 100, height: 24, bottom: 32 },
    ],
  }
})

interface MetricTableRow { _key: number; name: string; value: string; labels: string; _rawLabels: Record<string, string>; _rawExpression: string }

function openLabelDrawer(labels: Record<string, string>, seriesName: string) {
  labelDrawerData.value = labels
  labelDrawerSeriesName.value = seriesName
  labelDrawerVisible.value = true
}

function goToCreateAlertRule(expr: string) {
  router.push({ path: '/alert/rules', query: { from: 'explore', expr: encodeURIComponent(expr) } })
}

function buildExpression(labels: Record<string, string>): string {
  const name = labels.__name__ || ''
  const parts: string[] = []
  for (const [k, v] of Object.entries(labels)) {
    if (k !== '__name__') parts.push(`${k}="${v}"`)
  }
  return parts.length > 0 ? `${name}{${parts.join(',')}}` : name
}

const metricColumns = computed(() => [
  { title: t('query.metricName'), key: 'name', ellipsis: { tooltip: true }, width: 200 },
  { title: t('query.value'), key: 'value', width: 160 },
  {
    title: t('query.labelsHeader'), key: 'labels', ellipsis: { tooltip: true },
    render: (row: MetricTableRow) => {
      const labels = row._rawLabels || {}
      const entries = Object.entries(labels).filter(([k]) => k !== '__name__')
      if (entries.length === 0) return '-'
      return h('span', {
        style: 'cursor: pointer; text-decoration: underline dotted; color: var(--sre-primary)',
        onClick: (e: MouseEvent) => {
          e.stopPropagation()
          openLabelDrawer(labels, row.name)
        },
      }, formatLabelsStr(labels))
    },
  },
  {
    title: '',
    key: 'actions',
    width: 140,
    render: (row: MetricTableRow) => {
      const expr = row._rawExpression || ''
      if (!expr) return null
      return h(NButton, {
        size: 'tiny',
        quaternary: true,
        type: 'primary',
        onClick: (e: MouseEvent) => {
          e.stopPropagation()
          goToCreateAlertRule(expr)
        },
      }, { default: () => t('query.addToAlertRule'), icon: () => h(NIcon, { component: AddOutline }) })
    },
  },
])

const metricTableData = computed(() => {
  if (!metricData.value?.series) return []
  const rows: MetricTableRow[] = []
  let idx = 0
  for (const s of metricData.value.series) {
    for (const v of (s.values || [])) {
      rows.push({
        _key: idx++,
        name: s.labels?.__name__ || '-',
        value: typeof v.value === 'number' ? v.value.toFixed(4) : String(v.value ?? '-'),
        labels: formatLabelsStr(s.labels),
        _rawLabels: s.labels || {},
        _rawExpression: buildExpression(s.labels || {}),
      })
    }
  }
  return rows
})

const logColumns = computed(() => [
  { title: t('query.logTime'), key: 'timestamp', width: 200, render: (r: LogEntry) => fmtTs(r.timestamp) },
  { title: t('query.logMessage'), key: 'message', ellipsis: { tooltip: true } },
  { title: t('query.logLabels'), key: '_labels', width: 300, ellipsis: { tooltip: true }, render: (r: LogEntry) => formatLabelsStr(r.labels) },
])

// Enhanced log columns with level indicator and expand support
const logColumnsEnhanced = computed(() => [
  {
    title: '',
    key: 'level',
    width: 6,
    render: (r: LogEntry) => {
      const level = detectLogLevel(r)
      return h('div', {
        style: {
          width: '4px',
          height: '100%',
          minHeight: '20px',
          borderRadius: '2px',
          background: LEVEL_COLORS[level],
        },
      })
    },
  },
  {
    title: t('query.logTime'),
    key: 'timestamp',
    width: 180,
    render: (r: LogEntry) => {
      const level = detectLogLevel(r)
      return h('span', {
        style: { fontFamily: 'var(--sre-font-mono, monospace)', fontSize: '12px', color: level === 'error' || level === 'fatal' ? '#ef4444' : undefined },
      }, fmtTs(r.timestamp))
    },
  },
  {
    title: t('query.logMessage'),
    key: 'message',
    ellipsis: { tooltip: true },
    render: (r: LogEntry) => {
      const level = detectLogLevel(r)
      const color = level === 'error' || level === 'fatal' ? '#ef4444' : level === 'warn' ? '#eab308' : undefined
      return h('span', {
        style: { fontFamily: 'var(--sre-font-mono, monospace)', fontSize: '12px', color, whiteSpace: 'pre-wrap', wordBreak: 'break-all' },
      }, r.message || '-')
    },
  },
  {
    title: t('query.logLabels'),
    key: '_labels',
    width: 280,
    ellipsis: { tooltip: true },
    render: (r: LogEntry) => {
      const labels = r.labels || {}
      const entries = Object.entries(labels)
      if (entries.length === 0) return '-'
      return h('div', { style: 'display:flex;flex-wrap:wrap;gap:2px;' },
        entries.slice(0, 6).map(([k, v]) =>
          h(NTag, {
            size: 'tiny',
            bordered: false,
            style: 'cursor:pointer;max-width:140px;',
            onClick: (e: MouseEvent) => {
              e.stopPropagation()
              // Copy field value to clipboard
              navigator.clipboard?.writeText(`${k}=${v}`)
              message.success(`Copied: ${k}=${v}`)
            },
          }, { default: () => `${k}=${v}` })
        )
      )
    },
  },
])

function logRowClassName(row: LogEntry) {
  const level = detectLogLevel(row)
  if (level === 'error' || level === 'fatal') return 'log-row-error'
  if (level === 'warn') return 'log-row-warn'
  return ''
}

// --- helpers ---
function formatLegend(lbs: Record<string, string>): string {
  if (!lbs) return 'value'
  const parts: string[] = []
  for (const k of Object.keys(lbs)) {
    if (k !== '__name__') parts.push(`${k}="${lbs[k]}"`)
  }
  return parts.length ? parts.join(', ') : (lbs.__name__ || 'value')
}
function formatLabelsStr(lbs: Record<string, unknown> | undefined): string {
  if (!lbs) return '-'
  const parts: string[] = []
  for (const k of Object.keys(lbs)) {
    if (k !== '__name__') parts.push(`${k}=${lbs[k]}`)
  }
  return parts.length ? parts.join(', ') : '-'
}
function fmtTs(ts: string | number | undefined): string {
  if (!ts) return '-'
  return formatTime(String(ts))
}

// --- CSV export ---
function csvEscape(v: unknown): string {
  if (v == null) return ''
  const s = String(v)
  if (s.includes(',') || s.includes('"') || s.includes('\n')) {
    return `"${s.replace(/"/g, '""')}"`
  }
  return s
}

function downloadCsv(rows: string[][], filename: string) {
  const csv = rows.map(r => r.map(csvEscape).join(',')).join('\n')
  const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

function exportCsv() {
  const ts = new Date().toISOString().replace(/[:.]/g, '-')
  if (isLogs.value && logEntries.value.length) {
    const rows = [[t('query.csvTimestamp'), t('query.csvMessage'), t('query.csvLabels')]]
    for (const e of logEntries.value) {
      rows.push([fmtTs(e.timestamp), e.message || '', formatLabelsStr(e.labels)])
    }
    downloadCsv(rows, `query-result-${ts}.csv`)
    message.success(t('query.csvExported'))
  } else if (metricTableData.value.length) {
    const rows = [[t('query.csvName'), t('query.csvValue'), t('query.csvLabels')]]
    for (const r of metricTableData.value) rows.push([r.name, r.value, r.labels])
    downloadCsv(rows, `query-result-${ts}.csv`)
    message.success(t('query.csvExported'))
  }
}

const canExport = computed(() => {
  if (isLogs.value) return logEntries.value.length > 0
  return resultMode.value === 'table' && metricTableData.value.length > 0
})

// --- histogram fetch ---
async function fetchHistogram() {
  if (!selectedDsId.value || !expression.value.trim() || !isLogs.value) {
    histogramBuckets.value = []
    return
  }
  histogramLoading.value = true
  try {
    const res = await datasourceApi.logHistogram(selectedDsId.value, {
      expression: expression.value,
      start: timeStart.value,
      end: timeEnd.value,
    })
    const data = res.data?.data
    histogramBuckets.value = data?.buckets || []
  } catch {
    histogramBuckets.value = []
  } finally {
    histogramLoading.value = false
  }
}

function onHistogramBarClick(start: number, end: number) {
  // Zoom time range to the clicked bucket
  rangeMin.value = -1
  customRange.value = [start * 1000, end * 1000]
  showCustomPicker.value = true
  run()
}

// --- URL sync (Nightingale Explorer pattern) ---
function syncToURL() {
  const url = new URL(window.location.href)
  if (selectedDsId.value) url.searchParams.set('ds', String(selectedDsId.value))
  else url.searchParams.delete('ds')
  if (expression.value) url.searchParams.set('expr', expression.value)
  else url.searchParams.delete('expr')
  url.searchParams.set('tab', activeTab.value)
  url.searchParams.set('mode', queryMode.value)
  if (rangeMin.value === -1 && customRange.value) {
    url.searchParams.set('start', String(Math.floor(customRange.value[0] / 1000)))
    url.searchParams.set('end', String(Math.floor(customRange.value[1] / 1000)))
  } else {
    url.searchParams.set('range', String(rangeMin.value))
    url.searchParams.delete('start')
    url.searchParams.delete('end')
  }
  window.history.replaceState({}, '', url.toString())
}

function syncFromURL() {
  const params = new URLSearchParams(window.location.search)
  const ds = params.get('ds')
  const expr = params.get('expr')
  const tab = params.get('tab')
  const mode = params.get('mode')
  const range = params.get('range')
  const start = params.get('start')
  const end = params.get('end')

  if (tab === 'metrics' || tab === 'logs') activeTab.value = tab
  if (mode === 'instant' || mode === 'range') queryMode.value = mode as QueryMode
  if (range) {
    const v = Number(range)
    if (!isNaN(v) && presetOptions.some(p => p.value === v)) rangeMin.value = v
  }
  if (start && end) {
    rangeMin.value = -1
    customRange.value = [Number(start) * 1000, Number(end) * 1000]
    showCustomPicker.value = true
  }
  // ds and expr will be applied after datasources load
  return { ds: ds ? Number(ds) : null, expr: expr || '' }
}

// --- watch ---
watch(selectedDsId, () => {
  expression.value = ''
  metricData.value = null
  logEntries.value = []
  histogramBuckets.value = []
  errorMsg.value = ''
})

watch(activeTab, () => {
  selectedDsId.value = null
  expression.value = ''
  metricData.value = null
  logEntries.value = []
  errorMsg.value = ''
})

// --- URL sync for query mode ---
function syncModeFromURL() {
  const params = new URLSearchParams(window.location.search)
  const type = params.get('type')
  if (type === 'instant' || type === 'range') {
    queryMode.value = type
  }
}

function syncModeToURL() {
  const url = new URL(window.location.href)
  url.searchParams.set('type', queryMode.value)
  window.history.replaceState({}, '', url.toString())
}

watch(queryMode, () => {
  syncModeToURL()
})

onMounted(async () => {
  const urlState = syncFromURL()
  await loadDs()
  // Apply URL state after datasources loaded
  if (urlState.ds && datasources.value.some(d => d.id === urlState.ds)) {
    selectedDsId.value = urlState.ds
    if (urlState.expr) expression.value = urlState.expr
  }
  loadECharts()
  loadHistory()
  setupThemeObserver()
})

onUnmounted(() => {
  stopAutoTimer()
  _themeObserver?.disconnect()
})
</script>

<template>
  <div class="query-page">
    <!-- Header -->
    <div class="query-header">
      <div>
        <h2 class="query-title">{{ t('query.title') }}</h2>
        <p class="query-subtitle">{{ t('query.subtitle') }}</p>
      </div>
    </div>

    <!-- Toolbar Card: time range + refresh -->
    <div class="toolbar-card">
      <div class="toolbar-row">
        <div class="preset-group">
          <NButton
            v-for="opt in presetOptions"
            :key="opt.value"
            size="small"
            :type="rangeMin === opt.value ? 'primary' : 'default'"
            :secondary="rangeMin !== opt.value"
            @click="selectPreset(opt.value)"
          >
            {{ opt.label }}
          </NButton>
          <NButton
            size="small"
            :type="rangeMin === -1 ? 'primary' : 'default'"
            :secondary="rangeMin !== -1"
            @click="openCustomRange"
          >
            {{ t('query.timeCustom') }}
          </NButton>
        </div>

        <div class="toolbar-right">
          <NButton size="small" @click="run" :loading="loading">
            <template #icon><NIcon><RefreshOutline /></NIcon></template>
            {{ t('query.refreshBtn') }}
          </NButton>
          <NSelect
            v-model:value="autoRefreshSec"
            :options="autoRefreshOptions"
            size="small"
            class="auto-refresh-select"
          >
            <template #arrow>
              <span class="select-prefix">
                <span v-if="autoRefreshSec > 0 && autoCountdown > 0" class="countdown">{{ autoCountdown }}s</span>
              </span>
            </template>
          </NSelect>
        </div>
      </div>

      <div v-if="rangeMin === -1 && showCustomPicker" class="custom-range-row">
        <NDatePicker
          v-model:value="customRange"
          type="datetimerange"
          size="small"
          clearable
          class="custom-date-picker"
        />
      </div>

      <div class="range-display">
        <NIcon size="12"><TimeOutline /></NIcon>
        <span>{{ rangeDisplay }}</span>
      </div>
    </div>

    <!-- Tabs -->
    <NTabs v-model:value="activeTab" type="line" class="tabs-margin">
      <NTabPane name="metrics" :tab="t('query.metricsTab')" />
      <NTabPane name="logs" :tab="t('query.logsTab')" />
    </NTabs>

    <!-- Editor Card -->
    <div class="editor-card">
      <div class="ds-selector">
        <NSelect
          v-model:value="selectedDsId"
          :options="(isLogs ? logDatasources : metricDatasources).map(d => ({ label: dsLabel(d), value: d.id }))"
          :placeholder="isLogs ? t('query.selectLogDatasource') : t('query.selectDatasource')"
          filterable
          clearable
          size="small"
          class="ds-select"
        />
        <div v-if="selectedDs" class="ds-info">
          <NTag :color="{ color: typeColor(selectedDs.type), textColor: '#f1f5f9' }" size="small" :bordered="false">
            {{ typeBadge(selectedDs.type) }}
          </NTag>
          <span class="ds-endpoint">{{ selectedDs.endpoint }}</span>
        </div>
      </div>

      <div v-if="isLogs && !logDatasources.length" class="query-empty-inline">
        {{ t('query.noLogDatasources') }}
      </div>

      <div v-if="selectedDsId != null" class="query-bar">
        <div class="query-editor-wrap">
          <PromQLEditor
            v-if="!isLogs"
            v-model="expression"
            :datasource-id="selectedDsId"
            :placeholder="t('query.promqlPlaceholder')"
            @execute="run"
          />
          <LogsQLEditor
            v-else
            v-model="expression"
            :datasource-id="selectedDsId"
            :placeholder="t('query.logQueryPlaceholder')"
            @execute="run"
          />
          <div class="editor-tools">
            <NPopover v-model:show="historyVisible" trigger="click" placement="bottom-end" class="history-popover">
              <template #trigger>
                <NTooltip>
                  <template #trigger>
                    <NButton size="tiny" quaternary>
                      <template #icon><NIcon><TimeOutline /></NIcon></template>
                    </NButton>
                  </template>
                  {{ t('query.queryHistory') }}
                </NTooltip>
              </template>
              <div class="history-pop">
                <div class="history-title">{{ t('query.recentQueries') }}</div>
                <div v-if="!filteredHistory.length" class="history-empty">{{ t('query.noHistory') }}</div>
                <div
                  v-for="h in filteredHistory"
                  :key="h.ts"
                  class="history-item"
                  @click="applyHistory(h)"
                >
                  <div class="history-expr">{{ h.expression }}</div>
                  <div class="history-ts">{{ fmtTs(h.ts) }}</div>
                </div>
              </div>
            </NPopover>
            <NTooltip>
              <template #trigger>
                <NButton size="tiny" quaternary :disabled="!expression" @click="clearExpression">
                  <template #icon><NIcon><TrashOutline /></NIcon></template>
                </NButton>
              </template>
              {{ t('query.clearBtn') }}
            </NTooltip>
          </div>
        </div>
      </div>

      <div v-if="selectedDsId != null" class="query-actions-row">
        <NSpace :size="8" align="center">
          <template v-if="!isLogs">
            <NButtonGroup size="small">
              <NButton :type="queryMode === 'instant' ? 'primary' : 'default'" :secondary="queryMode !== 'instant'" @click="queryMode = 'instant'">
                {{ t('query.instant') }}
              </NButton>
              <NButton :type="queryMode === 'range' ? 'primary' : 'default'" :secondary="queryMode !== 'range'" @click="queryMode = 'range'">
                {{ t('query.range') }}
              </NButton>
            </NButtonGroup>
            <span class="field-label">{{ t('query.step') }}</span>
            <NSelect v-model:value="stepValue" :options="stepOptions" size="small" class="control-select-sm" />
            <span class="field-label">{{ t('query.limit') }}</span>
            <NSelect v-model:value="metricLimit" :options="metricLimitOptions" size="small" class="control-select-sm" />
          </template>
          <template v-else>
            <span class="field-label">{{ t('query.limit') }}</span>
            <NSelect v-model:value="logLimit" :options="logLimitOptions" size="small" class="control-select-sm" />
          </template>
        </NSpace>
        <NSpace :size="8" align="center">
          <span class="shortcut-hint">{{ t('query.shortcutHint') }}</span>
          <NButton type="primary" size="small" :loading="loading" :disabled="!expression.trim()" @click="run">
            {{ t('query.runQuery') }}
          </NButton>
        </NSpace>
      </div>

      <div v-if="selectedDsId == null && !(isLogs && !logDatasources.length)" class="query-empty-inline">
        {{ isLogs ? t('query.selectLogDatasource') : t('query.selectDatasource') }}
      </div>
    </div>

    <!-- Error -->
    <div v-if="errorMsg" class="error-card">
      <div class="error-icon-wrap">
        <NIcon size="18" color="#fafaf9"><AlertCircleOutline /></NIcon>
      </div>
      <div class="error-body">
        <div class="error-title">{{ t('query.queryFailed') }}</div>
        <div class="error-message">{{ errorMsg }}</div>
      </div>
      <div class="error-actions">
        <NButton size="small" @click="run">{{ t('common.retry') }}</NButton>
        <NButton size="small" quaternary @click="errorMsg = ''">{{ t('common.close') }}</NButton>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-container">
      <NSpin size="medium" />
    </div>

    <!-- Metrics Results -->
    <div v-if="!loading && !isLogs && metricData?.series?.length" class="results-panel">
      <div class="results-header">
        <div class="results-header-left">
          <span class="results-count">
            {{ metricData.series.length }} {{ t('query.seriesCount') }}
            <template v-if="metricData.result_type"> · {{ metricData.result_type }}</template>
            <NTag v-if="isMetricLimited" type="warning" size="small" :bordered="false" class="tag-ml">
              {{ t('query.limitedTo', { n: metricLimit }) }}
            </NTag>
          </span>
          <span v-if="queryStats" class="query-stats">
            {{ queryStats.executionTimeMs }}ms
            <template v-if="queryStats.step"> · step {{ queryStats.step }}</template>
          </span>
        </div>
        <NSpace :size="4">
          <NButton size="small" :type="resultMode === 'chart' ? 'primary' : 'default'" :secondary="resultMode !== 'chart'" @click="resultMode = 'chart'">
            {{ t('query.chart') }}
          </NButton>
          <NButton size="small" :type="resultMode === 'table' ? 'primary' : 'default'" :secondary="resultMode !== 'table'" @click="resultMode = 'table'">
            {{ t('query.table') }}
          </NButton>
          <NButton v-if="canExport" size="small" tertiary @click="exportCsv">
            <template #icon><NIcon><DownloadOutline /></NIcon></template>
            {{ t('query.exportCsv') }}
          </NButton>
        </NSpace>
      </div>

      <div v-if="resultMode === 'chart'" class="chart-container">
        <template v-if="ChartReady && VChart && chartOption">
          <component :is="VChart" :option="chartOption" :autoresize="true" class="chart-full" />
        </template>
        <div v-else class="chart-fallback">
          <p>{{ t('query.chartUnavailable') }}</p>
          <NButton size="small" @click="resultMode = 'table'">{{ t('query.switchToTable') }}</NButton>
        </div>
      </div>

      <NDataTable
        v-if="resultMode === 'table'"
        :columns="metricColumns"
        :data="metricTableData"
        :row-key="(r: Record<string, unknown>) => String(r._key)"
        size="small"
        :single-line="false"
        striped
        max-height="500"
        virtual-scroll
      />
    </div>

    <!-- Log Results -->
    <div v-if="!loading && isLogs && logEntries.length" class="results-panel">
      <div class="results-header">
        <div class="results-header-left">
          <span class="results-count">
            {{ t('query.showing') }} {{ logEntries.length }}
            <template v-if="logTotal > 0"> / {{ logTotal }}</template>
            {{ t('query.entries') }}
            <NTag v-if="logTruncated" type="warning" size="small" :bordered="false" class="tag-ml">
              {{ t('query.truncated') }}
            </NTag>
          </span>
          <span v-if="queryStats" class="query-stats">
            {{ queryStats.executionTimeMs }}ms
          </span>
        </div>
        <NSpace :size="4" align="center">
          <NButton size="tiny" quaternary @click="showHistogram = !showHistogram">
            {{ showHistogram ? t('query.hideHistogram') : t('query.showHistogram') }}
          </NButton>
          <NButton v-if="canExport" size="small" tertiary @click="exportCsv">
            <template #icon><NIcon><DownloadOutline /></NIcon></template>
            {{ t('query.exportCsv') }}
          </NButton>
        </NSpace>
      </div>

      <!-- Histogram -->
      <LogHistogram
        v-if="showHistogram"
        :buckets="histogramBuckets"
        :loading="histogramLoading"
        class="log-histogram-container"
        @bar-click="onHistogramBarClick"
      />

      <!-- Log Level Legend -->
      <div class="log-level-legend">
        <span v-for="(color, level) in LEVEL_COLORS" :key="level" class="level-item" v-show="level !== 'unknown'">
          <span class="level-dot" :style="{ background: color }" />
          <span class="level-label">{{ level }}</span>
        </span>
      </div>

      <!-- Log Table -->
      <NDataTable
        :columns="logColumnsEnhanced"
        :data="logEntries"
        :row-key="(r: Record<string, unknown>) => String(r._key)"
        :expanded-row-keys="expandedRowKeys"
        :row-class-name="logRowClassName"
        size="small"
        max-height="600"
        virtual-scroll
        @update:expanded-row-keys="(keys: number[]) => expandedRowKeys = keys"
      >
        <template #expand="{ row }">
          <div class="log-expanded-row">
            <div class="log-expanded-title">{{ t('query.logFields') }}</div>
            <div class="log-expanded-grid">
              <div
                v-for="(v, k) in row.labels"
                :key="k"
                class="log-field-item"
                @click="() => { navigator.clipboard?.writeText(`${k}=${v}`); message.success(`${t('query.copiedField')}: ${k}=${v}`) }"
              >
                <span class="log-field-key">{{ k }}</span>
                <span class="log-field-value">{{ v }}</span>
              </div>
            </div>
            <div class="log-expanded-level">
              Level: <strong>{{ detectLogLevel(row).toUpperCase() }}</strong>
            </div>
          </div>
        </template>
      </NDataTable>
    </div>

    <!-- No results -->
    <div
      v-if="!loading && !errorMsg && selectedDsId && expression.trim()
        && ((!isLogs && metricData !== null && !metricData?.series?.length)
          || (isLogs && !logEntries.length && metricData === null && logTotal === 0))"
      class="query-empty"
    >
      {{ t('query.noResults') }}
    </div>

    <!-- Label Detail Drawer -->
    <NDrawer v-model:show="labelDrawerVisible" :width="400" placement="right">
      <NDrawerContent :title="t('query.labelDetails')">
        <div class="label-drawer-header">
          <span class="label-drawer-series">{{ labelDrawerSeriesName }}</span>
        </div>
        <NDescriptions :column="1" bordered size="small" label-placement="left">
          <NDescriptionsItem v-for="(v, k) in labelDrawerData" :key="k" :label="String(k)">
            <span class="label-drawer-value">{{ v }}</span>
          </NDescriptionsItem>
        </NDescriptions>
        <div class="label-drawer-actions">
          <NButton
            type="primary"
            size="small"
            @click="goToCreateAlertRule(buildExpression(labelDrawerData))"
          >
            <template #icon><NIcon :component="AddOutline" /></template>
            {{ t('query.addToAlertRule') }}
          </NButton>
        </div>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped>
.query-page {
  max-width: 1600px;
  padding: 24px;
}

.query-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.query-title {
  font-size: 22px;
  font-weight: 600;
  margin: 0 0 4px 0;
  color: var(--sre-text-primary);
}

.query-subtitle {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin: 0;
}

.toolbar-card {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 12px;
}

.toolbar-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.preset-group {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.toolbar-right {
  display: flex;
  gap: 8px;
  align-items: center;
}

.select-prefix {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.countdown {
  color: var(--sre-primary);
  font-weight: 500;
}

.custom-range-row {
  margin-top: 8px;
}

.range-display {
  margin-top: 8px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  display: flex;
  align-items: center;
  gap: 4px;
}

.editor-card {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
}

.ds-selector {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.ds-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ds-endpoint {
  font-size: 12px;
  color: var(--sre-text-tertiary);
}

.query-bar {
  margin-bottom: 12px;
}

.query-editor-wrap {
  position: relative;
}

.expr-textarea {
  width: 100%;
  min-height: 56px;
  font-family: var(--sre-font-mono);
  font-size: 13px;
  padding: 8px 12px;
  border: 1px solid var(--n-border-color, #e0e0e6);
  border-radius: 4px;
  resize: vertical;
  background: var(--n-color, #fff);
  color: var(--n-text-color, #333);
  outline: none;
}
.expr-textarea:focus {
  border-color: var(--n-primary-color, #18a058);
}

.editor-tools {
  position: absolute;
  top: 6px;
  right: 6px;
  display: flex;
  gap: 2px;
}

.query-actions-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.field-label {
  font-size: 12px;
  color: var(--sre-text-tertiary);
}

.shortcut-hint {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  white-space: nowrap;
}

.query-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  color: var(--sre-text-tertiary);
  font-size: 14px;
}

.query-empty-inline {
  padding: 16px 4px;
  color: var(--sre-text-tertiary);
  font-size: 13px;
}

.results-panel {
  background: var(--sre-bg-card);
  border-radius: 8px;
  padding: 16px;
  border: 1px solid var(--sre-border);
  overflow: hidden;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.results-count {
  font-size: 13px;
  color: var(--sre-text-secondary);
}

.chart-container {
  min-height: 400px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  position: relative;
  z-index: 1;
}

.chart-fallback {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--sre-text-tertiary);
  font-size: 13px;
}

.history-pop {
  min-width: 360px;
  max-width: 480px;
}

.history-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  margin-bottom: 8px;
}

.history-empty {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  padding: 12px 0;
  text-align: center;
}

.history-item {
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
  border: 1px solid transparent;
}

.history-item:hover {
  background: var(--sre-bg-hover);
  border-color: var(--sre-border);
}

.history-expr {
  font-family: var(--sre-font-mono);
  font-size: 12px;
  color: var(--sre-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.history-ts {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  margin-top: 2px;
}

/* ---- Inline-style replacements ---- */
.custom-date-picker {
  width: 420px;
}
.ds-select {
  max-width: 420px;
  flex: 1;
}
.control-select-sm {
  width: 100px;
}
.auto-refresh-select {
  width: 140px;
}
.chart-full {
  width: 100%;
  height: 400px;
}
.loading-container {
  display: flex;
  justify-content: center;
  padding: 40px;
}
.tabs-margin {
  margin-bottom: 12px;
}
.tag-ml {
  margin-left: 8px;
}

/* Log histogram */
.log-histogram-container {
  margin-bottom: 12px;
}

/* Log level legend */
.log-level-legend {
  display: flex;
  gap: 12px;
  margin-bottom: 8px;
  padding: 4px 0;
}
.level-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--sre-text-tertiary);
}
.level-dot {
  width: 8px;
  height: 8px;
  border-radius: 2px;
  flex-shrink: 0;
}
.level-label {
  text-transform: uppercase;
  font-weight: 500;
  letter-spacing: 0.5px;
}

/* Query stats */
.query-stats {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  font-family: var(--sre-font-mono, monospace);
  margin-left: 8px;
}
.results-header-left {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* Log row level coloring */
:deep(.log-row-error) {
  background: rgba(239, 68, 68, 0.04) !important;
}
:deep(.log-row-warn) {
  background: rgba(234, 179, 8, 0.04) !important;
}

/* Log expanded row */
.log-expanded-row {
  padding: 12px 16px;
  background: var(--sre-bg-sunken, #f8fafc);
  border-radius: 6px;
}
.log-expanded-title {
  font-weight: 600;
  font-size: 13px;
  color: var(--sre-text-primary);
  margin-bottom: 8px;
}
.log-expanded-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 6px;
}
.log-field-item {
  display: flex;
  gap: 6px;
  padding: 4px 8px;
  border-radius: 4px;
  background: var(--sre-bg-card, #fff);
  border: 1px solid var(--sre-border);
  cursor: pointer;
  font-size: 12px;
  transition: border-color 0.15s;
}
.log-field-item:hover {
  border-color: var(--sre-primary);
}
.log-field-key {
  color: var(--sre-primary);
  font-weight: 500;
  min-width: 80px;
  font-family: var(--sre-font-mono, monospace);
}
.log-field-value {
  color: var(--sre-text-secondary);
  word-break: break-all;
  font-family: var(--sre-font-mono, monospace);
}
.log-expanded-level {
  margin-top: 8px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
}

/* ---- Error card ---- */
.error-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  margin: 12px 0;
  background: var(--sre-critical-soft);
  border: 1px solid var(--sre-critical-soft);
  border-radius: var(--sre-radius-md);
}
.error-icon-wrap {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--sre-critical);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  color: var(--sre-text-inverse);
}
.error-body {
  flex: 1;
  min-width: 0;
}
.error-title {
  font-size: var(--sre-fs-md);
  font-weight: var(--sre-fw-semibold);
  color: var(--sre-text-primary);
  margin: 0 0 4px;
}
.error-message {
  font-size: var(--sre-fs-sm);
  color: var(--sre-text-secondary);
  line-height: var(--sre-lh-snug);
  word-break: break-all;
}
.error-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
  margin-top: 2px;
}

/* ---- Label drawer ---- */
.label-drawer-header {
  margin-bottom: 16px;
}
.label-drawer-series {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  font-family: var(--sre-font-mono, monospace);
}
.label-drawer-value {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  word-break: break-all;
}
.label-drawer-actions {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

/* ---- Mobile responsive ---- */
@media (max-width: 768px) {
  .query-page {
    padding: 16px;
  }
  .toolbar-row {
    flex-direction: column;
    align-items: stretch;
  }
  .preset-group {
    justify-content: center;
  }
  .toolbar-right {
    justify-content: center;
  }
  .ds-selector {
    flex-direction: column;
    align-items: stretch;
  }
  .ds-select {
    max-width: 100%;
  }
  .query-actions-row {
    flex-direction: column;
    align-items: stretch;
  }
  .custom-date-picker {
    width: 100%;
  }
  .results-header {
    flex-direction: column;
    gap: 8px;
  }
  .query-header {
    flex-direction: column;
    gap: 12px;
  }
}
</style>
