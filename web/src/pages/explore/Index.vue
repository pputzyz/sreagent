<script setup lang="ts">
/**
 * Data Query Page — vmui-style layout.
 *
 * Architecture:
 *  - Query rows: Q1/Q2/... each with datasource + expression + enable/disable
 *  - Shared controls: time range buttons + step + refresh + Execute
 *  - Display mode: Graph / JSON / Table segmented control
 *  - Results panel: merged results from all enabled queries
 *  - Status bar: series count, points, query time, range info
 */
import { ref, onMounted, onUnmounted, computed, watch, shallowRef, h, type Component, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  NButton, NIcon, NSelect, NSpin, NTag, NTooltip,
  NDrawer, NDrawerContent, NDescriptions, NDescriptionsItem,
  NDataTable, NDatePicker, NInputNumber,
  useMessage,
} from 'naive-ui'
import {
  TimeOutline, AddOutline, RefreshOutline, DownloadOutline,
  AlertCircleOutline,
} from '@vicons/ionicons5'
import { datasourceApi } from '@/api'
import { formatTime } from '@/utils/format'
import ExploreQueryRow from '@/components/query/ExploreQueryRow.vue'
import MetricChartControls from '@/components/query/MetricChartControls.vue'
import type { ChartSettings } from '@/components/query/MetricChartControls.vue'
import LogHistogram from '@/components/query/LogHistogram.vue'
import LogDetailDrawer from '@/components/query/LogDetailDrawer.vue'
import LogFieldSidebar from '@/components/query/LogFieldSidebar.vue'
import LogViewSettings from '@/components/query/LogViewSettings.vue'
import FieldValueToken from '@/components/query/FieldValueToken.vue'
import FullscreenButton from '@/components/query/FullscreenButton.vue'
import ViewSelect from '@/components/query/ViewSelect.vue'
import type { DataSource, QueryResponse, LogEntry } from '@/types'
import type { SavedView } from '@/components/query/ViewSelect.vue'

const { t } = useI18n()
const router = useRouter()
const message = useMessage()

// ===== Datasources =====
const datasources = ref<DataSource[]>([])
async function loadDs() {
  try {
    const res = await datasourceApi.list({ page: 1, page_size: 100 })
    const list = res.data?.data?.list
    datasources.value = (Array.isArray(list) ? list : []).filter((d: DataSource) => d.is_enabled)
  } catch (e) { console.warn('[Explore] Failed to load datasources:', e) }
}

// ===== Lazy ECharts =====
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
    use([CanvasRenderer, LineChart, components.TooltipComponent, components.LegendComponent, components.GridComponent, components.DataZoomComponent])
    VChart.value = vc.default
    ChartReady.value = true
  } catch (e) { console.warn('[Explore] ECharts load failed:', e) }
}

// ===== Query Rows =====
interface QueryRowState {
  id: number
  dsId: number | null
  expression: string
  enabled: boolean
}
const queries = ref<QueryRowState[]>([
  { id: 1, dsId: null, expression: '', enabled: true },
])
let queryCounter = 1

function addQuery() {
  queryCounter++
  queries.value.push({ id: queryCounter, dsId: null, expression: '', enabled: true })
}
function removeQuery(id: number) {
  if (queries.value.length <= 1) return
  queries.value = queries.value.filter(q => q.id !== id)
}

// ===== Active Tab (metrics / logs) =====
const activeTab = ref<'metrics' | 'logs'>('metrics')
watch(activeTab, () => {
  // Reset datasource selections when switching tab
  queries.value.forEach(q => { q.dsId = null; q.expression = '' })
  resultData.value = null
  logEntries.value = []
  errorMsg.value = ''
})

// ===== Time Range =====
const rangeMin = ref<number>(60)
const customRange = ref<[number, number] | null>(null)
const now = ref(Date.now())
const showCustomPicker = ref(false)

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
const RANGE_SECONDS: Record<number, number> = {}
presetOptions.forEach(o => { RANGE_SECONDS[o.value] = o.value * 60 })
const RANGE_LABEL: Record<number, string> = {}
presetOptions.forEach(o => { RANGE_LABEL[o.value] = o.label })

const timeStart = computed(() => {
  if (rangeMin.value === -1 && customRange.value) return Math.floor(customRange.value[0] / 1000)
  return Math.floor((now.value - rangeMin.value * 60000) / 1000)
})
const timeEnd = computed(() => {
  if (rangeMin.value === -1 && customRange.value) return Math.floor(customRange.value[1] / 1000)
  return Math.floor(now.value / 1000)
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

// ===== Step =====
const stepValue = ref<string>('auto')
const stepOptions = computed(() => [
  { label: t('query.stepAuto'), value: 'auto' },
  { label: '15s', value: '15s' },
  { label: '30s', value: '30s' },
  { label: '1m', value: '1m' },
  { label: '5m', value: '5m' },
  { label: '15m', value: '15m' },
  { label: '1h', value: '1h' },
])

// ===== Auto-refresh =====
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
      runAll()
      autoCountdown.value = autoRefreshSec.value
    }
  }, 1000)
}
function stopAutoTimer() {
  if (autoTimer) { clearInterval(autoTimer); autoTimer = null }
  autoCountdown.value = 0
}
watch(autoRefreshSec, () => startAutoTimer())

// ===== Display Mode =====
type DisplayMode = 'graph' | 'json' | 'table'
const displayMode = ref<DisplayMode>('graph')
const stacked = ref(false)
const logScale = ref(false)

// ===== Results =====
const loading = ref(false)
const errorMsg = ref('')
const resultData = ref<QueryResponse | null>(null)
const instantData = ref<QueryResponse | null>(null)
const logEntries = ref<LogEntry[]>([])
const logTotal = ref(0)
const logTruncated = ref(false)
const queryDuration = ref(0)
const isLogs = computed(() => activeTab.value === 'logs')

// Log-specific state
const histogramBuckets = ref<{ timestamp: string; count: number }[]>([])
const histogramLoading = ref(false)
const showHistogram = ref(true)
const logLimit = ref(200)
const logLimitOptions = [50, 100, 200, 500, 1000, 5000].map(v => ({ label: String(v), value: v }))
const metricLimit = ref(100)
const metricLimitOptions = [50, 100, 200, 500, 1000].map(v => ({ label: String(v), value: v }))

type LogMode = 'origin' | 'table'
const logMode = ref<LogMode>('origin')
const logOptions = ref({ lineBreak: true, showTime: true, showLabels: true, showLineNum: false, jsonExpandLevel: 2 })

const logFields = computed(() => {
  const fieldSet = new Set<string>()
  for (const entry of logEntries.value) {
    if (entry.labels) for (const k of Object.keys(entry.labels)) fieldSet.add(k)
  }
  return Array.from(fieldSet).sort()
})

// Log level detection
type LogLevel = 'debug' | 'info' | 'warn' | 'error' | 'fatal' | 'unknown'
const LEVEL_COLORS: Record<LogLevel, string> = {
  debug: '#64748b', info: '#0d9488', warn: '#eab308', error: '#ef4444', fatal: '#dc2626', unknown: 'transparent',
}
function detectLogLevel(entry: LogEntry): LogLevel {
  const level = (entry.labels?.level || entry.labels?.severity || entry.labels?.lvl || '').toString().toLowerCase()
  if (level.includes('error') || level.includes('err')) return 'error'
  if (level.includes('warn') || level.includes('wrn')) return 'warn'
  if (level.includes('info') || level.includes('inf')) return 'info'
  if (level.includes('debug') || level.includes('dbg') || level.includes('trace')) return 'debug'
  if (level.includes('fatal') || level.includes('crit') || level.includes('panic')) return 'fatal'
  const msg = (entry.message || '').toLowerCase()
  if (msg.includes('error') || msg.includes('exception') || msg.includes('fail')) return 'error'
  if (msg.includes('warn')) return 'warn'
  return 'unknown'
}
function logRowClassName(row: LogEntry) {
  const level = detectLogLevel(row)
  if (level === 'error' || level === 'fatal') return 'log-row-error'
  if (level === 'warn') return 'log-row-warn'
  return ''
}

// Chart
const chartRef = ref<any>(null)
const chartSettings = ref<ChartSettings>({
  maxDataPoints: null, minStep: null, chartType: 'line', showLegend: true, sharedTooltip: false, tooltipSort: 'desc',
})
const maxDataPoints = ref<number | null>(null)
const minStep = ref<number | null>(null)
const isolatedSeries = ref<string | null>(null)
const legendColors = [
  '#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de',
  '#3ba272', '#fc8452', '#9a60b4', '#ea7ccc', '#48b8d0',
  '#c4ccd3', '#5ab1ef', '#d87c7c', '#8d98b3', '#e5cf0d',
  '#97b552', '#95706d', '#dc69aa', '#07a2a4', '#9a7fd1',
]

function formatLegend(lbs: Record<string, string>): string {
  if (!lbs) return 'value'
  const parts: string[] = []
  for (const k of Object.keys(lbs)) { if (k !== '__name__') parts.push(`${k}="${lbs[k]}"`) }
  return parts.length ? parts.join(', ') : (lbs.__name__ || 'value')
}
function formatLegendFull(lbs: Record<string, string>): string {
  if (!lbs) return 'value'
  const parts: string[] = []
  for (const [k, v] of Object.entries(lbs)) { if (k !== '__name__') parts.push(`${k}="${v}"`) }
  const name = lbs.__name__ || ''
  return parts.length > 0 ? `${name}{${parts.join(', ')}}` : name
}

const legendItems = computed(() => {
  if (!resultData.value?.series?.length) return []
  return resultData.value.series.map((s, i) => ({
    name: formatLegend(s.labels),
    color: legendColors[i % legendColors.length],
    fullName: formatLegendFull(s.labels),
  }))
})

function toggleLegend(name: string) {
  if (isolatedSeries.value === name) isolatedSeries.value = null
  else isolatedSeries.value = name
  const chart = chartRef.value?.chart || chartRef.value
  if (!chart || !resultData.value?.series) return
  const option = chart.getOption()
  if (!option?.series) return
  const selected: Record<string, boolean> = {}
  for (const s of option.series) selected[s.name] = isolatedSeries.value ? s.name === isolatedSeries.value : true
  chart.setOption({ legend: { selected } })
}

function resolveStep(): string {
  if (stepValue.value !== 'auto') return stepValue.value
  const duration = timeEnd.value - timeStart.value
  const maxPts = maxDataPoints.value || 240
  const computedStep = Math.ceil(duration / maxPts)
  const min = minStep.value || 15
  return `${Math.max(computedStep, min)}s`
}

const isMetricLimited = computed(() => {
  if (!resultData.value?.series) return false
  return resultData.value.series.length >= metricLimit.value
})

// Chart option
watch(resultData, () => { isolatedSeries.value = null })
const chartOption = computed(() => {
  if (!resultData.value?.series?.length) return null
  const seriesList: any[] = []
  const isArea = chartSettings.value.chartType === 'area'
  for (let i = 0; i < resultData.value.series.length; i++) {
    const s = resultData.value.series[i]
    const name = formatLegend(s.labels)
    const data: [number, number][] = []
    for (const v of s.values || []) data.push([Number(v.ts) * 1000, v.value != null ? Number(v.value) : 0])
    const seriesItem: any = { name, type: 'line', data, smooth: true, showSymbol: false, connectNulls: true, lineStyle: { width: 1.5, color: legendColors[i % legendColors.length] }, itemStyle: { color: legendColors[i % legendColors.length] } }
    if (isArea) { seriesItem.areaStyle = { color: legendColors[i % legendColors.length], opacity: 0.2 }; seriesItem.stack = stacked.value ? 'total' : undefined }
    seriesList.push(seriesItem)
  }
  const secondaryColor = typeof document !== 'undefined' ? getComputedStyle(document.documentElement).getPropertyValue('--sre-text-secondary').trim() || '#475569' : '#475569'
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis', confine: true,
      axisPointer: { type: 'cross', lineStyle: { type: 'dashed', opacity: 0.4 } },
      backgroundColor: 'var(--sre-bg-card, #fff)',
      borderColor: 'var(--sre-border, #e5e7eb)',
      textStyle: { color: 'var(--sre-text-primary, #1e293b)', fontSize: 12 },
      valueFormatter: (v: any) => (v == null ? '-' : Number(v).toFixed(4)),
    },
    legend: { show: false },
    grid: { left: 60, right: 24, top: 16, bottom: 20 },
    xAxis: { type: 'time', axisLine: { lineStyle: { color: 'var(--sre-border, #cbd5e1)' } }, axisLabel: { color: secondaryColor, fontSize: 11 }, splitLine: { show: false } },
    yAxis: { type: logScale.value ? 'log' : 'value', axisLine: { show: false }, axisLabel: { color: secondaryColor, fontSize: 11 }, splitLine: { lineStyle: { type: 'dashed' } } },
    series: seriesList,
    dataZoom: [{ type: 'inside' }],
  }
})

// ===== Execute =====
async function runAll() {
  const enabledQueries = queries.value.filter(q => q.enabled && q.dsId && q.expression.trim())
  if (!enabledQueries.length) return
  if (rangeMin.value !== -1) now.value = Date.now()

  loading.value = true
  errorMsg.value = ''
  resultData.value = null
  instantData.value = null
  logEntries.value = []
  const t0 = performance.now()

  try {
    if (isLogs.value) {
      // Log queries — run first enabled query (logs don't merge well)
      const q = enabledQueries[0]
      const res = await datasourceApi.logQuery(q.dsId!, {
        expression: q.expression,
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
      if (showHistogram.value) fetchHistogram()
    } else {
      // Metric queries — merge all enabled queries
      const allSeries: any[] = []
      let lastRangeData: any = null
      for (const q of enabledQueries) {
        try {
          const res = await datasourceApi.rangeQuery(q.dsId!, {
            expression: q.expression,
            start: timeStart.value,
            end: timeEnd.value,
            step: resolveStep(),
          })
          const data = res.data?.data
          if (data?.series) {
            lastRangeData = data
            allSeries.push(...data.series)
          }
        } catch (e: any) {
          console.warn(`[Explore] Query failed for Q${q.id}:`, e)
        }
      }
      if (allSeries.length > metricLimit.value) allSeries.length = metricLimit.value
      resultData.value = { result_type: 'matrix', series: allSeries, raw_count: allSeries.length }

      // Instant query for table
      try {
        const q0 = enabledQueries[0]
        const instantRes = await datasourceApi.query(q0.dsId!, { expression: q0.expression })
        instantData.value = instantRes.data?.data || null
      } catch { instantData.value = null }
    }
    queryDuration.value = Math.round(performance.now() - t0)
  } catch (e: any) {
    errorMsg.value = e?.response?.data?.error || e?.response?.data?.message || e?.message || t('query.queryFailed')
  } finally {
    loading.value = false
  }
}

async function fetchHistogram() {
  const q = queries.value.find(q => q.enabled && q.dsId && q.expression.trim())
  if (!q) { histogramBuckets.value = []; return }
  histogramLoading.value = true
  try {
    const res = await datasourceApi.logHistogram(q.dsId!, { expression: q.expression, start: timeStart.value, end: timeEnd.value })
    histogramBuckets.value = res.data?.data?.buckets || []
  } catch { histogramBuckets.value = [] }
  finally { histogramLoading.value = false }
}

function onHistogramRangeChange(start: number, end: number) {
  rangeMin.value = -1
  customRange.value = [start * 1000, end * 1000]
  showCustomPicker.value = true
}

// ===== Status bar =====
const statusSeries = computed(() => {
  if (isLogs.value) return logEntries.value.length
  return resultData.value?.series?.length || 0
})
const statusPoints = computed(() => {
  if (isLogs.value) return logEntries.value.length
  if (!resultData.value?.series) return 0
  return resultData.value.series.reduce((s, x) => s + (x.values?.length || 0), 0)
})
const statusRangeLabel = computed(() => {
  const label = RANGE_LABEL[rangeMin.value] || 'custom'
  return `Range: ${label} · Step: ${resolveStep()}`
})

// ===== Table data =====
interface MetricTableRow { _key: number; name: string; value: string; labels: string; _rawLabels: Record<string, string> }
const metricColumns = computed(() => [
  { title: '#', key: '_key', width: 40 },
  { title: t('query.metricName'), key: 'name', ellipsis: { tooltip: true }, width: 200 },
  { title: t('query.value'), key: 'value', width: 140 },
  { title: t('query.labelsHeader'), key: 'labels', ellipsis: { tooltip: true } },
])
const metricTableData = computed(() => {
  const source = instantData.value?.series?.length ? instantData.value : resultData.value
  if (!source?.series) return []
  const rows: MetricTableRow[] = []
  let idx = 0
  for (const s of source.series) {
    const values = s.values || []
    const displayValues = instantData.value?.series?.length ? values : values.slice(-1)
    for (const v of displayValues) {
      rows.push({
        _key: ++idx,
        name: s.labels?.__name__ || '-',
        value: typeof v.value === 'number' ? v.value.toFixed(4) : String(v.value ?? '-'),
        labels: formatLabelsStr(s.labels),
        _rawLabels: s.labels || {},
      })
    }
  }
  return rows
})
function formatLabelsStr(lbs: Record<string, unknown> | undefined): string {
  if (!lbs) return '-'
  const parts: string[] = []
  for (const k of Object.keys(lbs)) { if (k !== '__name__') parts.push(`${k}=${lbs[k]}`) }
  return parts.length ? parts.join(', ') : '-'
}
function buildExpression(labels: Record<string, string>): string {
  const name = labels.__name__ || ''
  const parts: string[] = []
  for (const [k, v] of Object.entries(labels)) { if (k !== '__name__') parts.push(`${k}="${v}"`) }
  return parts.length > 0 ? `${name}{${parts.join(',')}}` : name
}
function fmtTs(ts: string | number | undefined): string {
  if (!ts) return '-'
  return formatTime(String(ts))
}

// Log columns
const logColumnsEnhanced = computed(() => [
  { title: '', key: 'expand', width: 40, render: (_r: LogEntry, index: number) => h('div', { style: 'width:100%;height:100%;display:flex;align-items:center;justify-content:center;cursor:pointer;', onClick: () => openLogDrawer(index) }, [h('span', { style: 'font-size:14px;color:var(--sre-text-tertiary);' }, '\u{1F50D}')]) },
  { title: '', key: 'level', width: 6, render: (r: LogEntry) => h('div', { style: { width: '4px', height: '100%', minHeight: '20px', borderRadius: '2px', background: LEVEL_COLORS[detectLogLevel(r)] } }) },
  { title: t('query.logTime'), key: 'timestamp', width: 180, render: (r: LogEntry) => h('span', { style: { fontFamily: 'var(--sre-font-mono)', fontSize: '12px' } }, fmtTs(r.timestamp)) },
  { title: t('query.logMessage'), key: 'message', ellipsis: { tooltip: true }, render: (r: LogEntry) => h('span', { style: { fontFamily: 'var(--sre-font-mono)', fontSize: '12px', whiteSpace: 'pre-wrap', wordBreak: 'break-all', color: (detectLogLevel(r) === 'error' || detectLogLevel(r) === 'fatal') ? '#ef4444' : detectLogLevel(r) === 'warn' ? '#eab308' : undefined } }, r.message || '-') },
  { title: t('query.logLabels'), key: '_labels', width: 280, ellipsis: { tooltip: true }, render: (r: LogEntry) => {
    const labels = r.labels || {}
    const entries = Object.entries(labels)
    if (entries.length === 0) return '-'
    return h('div', { style: 'display:flex;flex-wrap:wrap;gap:2px;' }, entries.slice(0, 6).map(([k, v]) => h(NTag, { size: 'tiny', bordered: false, style: 'cursor:pointer;max-width:140px;', onClick: (e: MouseEvent) => { e.stopPropagation(); copyFieldValue(k, v) } }, { default: () => `${k}=${v}` })))
  } },
])
function copyFieldValue(k: string, v: unknown) {
  window.navigator?.clipboard?.writeText(`${k}=${v}`)
  message.success(`${t('query.copiedField')}: ${k}=${v}`)
}

// Log drawer
const drawerVisible = ref(false)
const drawerCurrentIndex = ref(0)
const drawerLogEntry = computed(() => logEntries.value[drawerCurrentIndex.value] || null)
function openLogDrawer(index: number) { drawerCurrentIndex.value = index; drawerVisible.value = true }
function onDrawerPrev() { if (drawerCurrentIndex.value > 0) drawerCurrentIndex.value-- }
function onDrawerNext() { if (drawerCurrentIndex.value < logEntries.value.length - 1) drawerCurrentIndex.value++ }

function onFieldFilterAdd(key: string, value: string) {
  const q = queries.value.find(q => q.enabled && q.dsId)
  if (q) {
    const filterExpr = `${key}="${value}"`
    q.expression = q.expression.trim() ? q.expression.trim() + ', ' + filterExpr : filterExpr
  }
}
function onTokenFilter(key: string, value: string, operator: string) {
  const q = queries.value.find(q => q.enabled && q.dsId)
  if (!q) return
  if (operator === 'AND') {
    const filterExpr = `${key}="${value}"`
    q.expression = q.expression.trim() ? q.expression.trim() + ', ' + filterExpr : filterExpr
  } else if (operator === 'NOT') {
    const filterExpr = `${key}!="${value}"`
    q.expression = q.expression.trim() ? q.expression.trim() + ', ' + filterExpr : filterExpr
  }
  runAll()
}

// ===== CSV Export =====
function csvEscape(v: unknown): string {
  if (v == null) return ''
  const s = String(v)
  if (s.includes(',') || s.includes('"') || s.includes('\n')) return `"${s.replace(/"/g, '""')}"`
  return s
}
function downloadCsv(rows: string[][], filename: string) {
  const csv = rows.map(r => r.map(csvEscape).join(',')).join('\n')
  const blob = new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
function exportCsv() {
  const ts = new Date().toISOString().replace(/[:.]/g, '-')
  if (isLogs.value && logEntries.value.length) {
    const rows = [[t('query.csvTimestamp'), t('query.csvMessage'), t('query.csvLabels')]]
    for (const e of logEntries.value) rows.push([fmtTs(e.timestamp), e.message || '', formatLabelsStr(e.labels)])
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
  return displayMode.value === 'table' && metricTableData.value.length > 0
})

// ===== Label Drawer =====
const labelDrawerVisible = ref(false)
const labelDrawerData = ref<Record<string, string>>({})
const labelDrawerSeriesName = ref('')
function openLabelDrawer(labels: Record<string, string>, seriesName: string) {
  labelDrawerData.value = labels
  labelDrawerSeriesName.value = seriesName
  labelDrawerVisible.value = true
}
function goToCreateAlertRule(expr: string) {
  router.push({ path: '/alert/rules', query: { from: 'explore', expr: encodeURIComponent(expr) } })
}

// ===== URL Sync =====
function syncToURL() {
  const url = new URL(window.location.href)
  if (rangeMin.value === -1 && customRange.value) {
    url.searchParams.set('start', String(Math.floor(customRange.value[0] / 1000)))
    url.searchParams.set('end', String(Math.floor(customRange.value[1] / 1000)))
    url.searchParams.delete('range')
  } else {
    url.searchParams.set('range', String(rangeMin.value))
    url.searchParams.delete('start')
    url.searchParams.delete('end')
  }
  url.searchParams.set('tab', activeTab.value)
  // Sync first query
  const q0 = queries.value[0]
  if (q0?.dsId) url.searchParams.set('ds', String(q0.dsId))
  else url.searchParams.delete('ds')
  if (q0?.expression) url.searchParams.set('expr', q0.expression)
  else url.searchParams.delete('expr')
  window.history.replaceState({}, '', url.toString())
}
function syncFromURL() {
  const params = new URLSearchParams(window.location.search)
  const ds = params.get('ds')
  const expr = params.get('expr')
  const tab = params.get('tab')
  const range = params.get('range')
  const start = params.get('start')
  const end = params.get('end')
  if (range) { const v = Number(range); if (!isNaN(v) && presetOptions.some(p => p.value === v)) rangeMin.value = v }
  if (start && end) { rangeMin.value = -1; customRange.value = [Number(start) * 1000, Number(end) * 1000]; showCustomPicker.value = true }
  if (tab === 'metrics' || tab === 'logs') activeTab.value = tab
  return { ds: ds ? Number(ds) : null, expr: expr || '' }
}
let syncTimer: ReturnType<typeof setTimeout> | null = null
function debouncedSyncToURL() {
  if (syncTimer) clearTimeout(syncTimer)
  syncTimer = setTimeout(syncToURL, 500)
}
watch(() => {
  const q0 = queries.value[0]
  return `${q0?.dsId}-${q0?.expression}-${activeTab.value}`
}, (val) => { if (val) debouncedSyncToURL() })

// ===== ViewSelect =====
const currentPanelTab = computed(() => activeTab.value)
const currentPanelDsId = computed(() => queries.value[0]?.dsId ?? null)
const currentPanelDsName = computed(() => {
  const dsId = currentPanelDsId.value
  if (!dsId) return ''
  return datasources.value.find(d => d.id === dsId)?.name || ''
})
const currentPanelExpression = computed(() => queries.value[0]?.expression || '')
function onViewLoad(view: SavedView) {
  const q0 = queries.value[0]
  if (q0) {
    if (view.dsId) q0.dsId = view.dsId
    if (view.expression) q0.expression = view.expression
    if (view.tab === 'metrics' || view.tab === 'logs') activeTab.value = view.tab
  }
}

// ===== Log results ref =====
const logResultsRef = ref<HTMLElement | null>(null)

// ===== Lifecycle =====
onMounted(async () => {
  const urlState = syncFromURL()
  await loadDs()
  if (urlState.ds && datasources.value.some(d => d.id === urlState.ds)) {
    queries.value[0].dsId = urlState.ds
    queries.value[0].expression = urlState.expr
  }
  loadECharts()
})
onUnmounted(() => { stopAutoTimer() })
</script>

<template>
  <div class="query-page">
    <!-- Query Rows -->
    <section class="card queries-section">
      <ExploreQueryRow
        v-for="(q, idx) in queries"
        :key="q.id"
        :index="idx"
        :ds-id="q.dsId"
        :expression="q.expression"
        :enabled="q.enabled"
        :datasources="datasources"
        :active-tab="activeTab"
        :can-remove="queries.length > 1"
        @update:ds-id="(v: number | null) => q.dsId = v"
        @update:expression="(v: string) => q.expression = v"
        @update:enabled="(v: boolean) => q.enabled = v"
        @remove="removeQuery(q.id)"
        @execute="runAll"
      />
      <button class="add-query-btn" @click="addQuery">+ Add Query</button>
    </section>

    <!-- Controls Row -->
    <section class="controls-row">
      <div class="range-group">
        <button
          v-for="opt in presetOptions"
          :key="opt.value"
          class="range-btn"
          :class="{ active: rangeMin === opt.value }"
          @click="selectPreset(opt.value)"
        >{{ opt.label }}</button>
        <button
          class="range-btn"
          :class="{ active: rangeMin === -1 }"
          @click="openCustomRange"
        >{{ t('query.timeCustom') }}</button>
      </div>
      <div v-if="rangeMin === -1 && showCustomPicker" class="custom-range-inline">
        <NDatePicker v-model:value="customRange" type="datetimerange" size="small" class="custom-date-picker" />
      </div>
      <div class="controls-spacer" />
      <div class="controls-right">
        <!-- Tab switch (Metrics / Logs) -->
        <div class="tab-group">
          <button class="tab-btn" :class="{ active: activeTab === 'metrics' }" @click="activeTab = 'metrics'">Metrics</button>
          <button class="tab-btn" :class="{ active: activeTab === 'logs' }" @click="activeTab = 'logs'">Logs</button>
        </div>
        <div class="ctrl-compact">
          <span class="ctrl-label">step</span>
          <NSelect v-model:value="stepValue" :options="stepOptions" size="small" class="ctrl-select-sm" />
        </div>
        <div class="ctrl-compact">
          <span class="ctrl-label">refresh</span>
          <NSelect v-model:value="autoRefreshSec" :options="autoRefreshOptions" size="small" class="ctrl-select-sm">
            <template #arrow>
              <span v-if="autoRefreshSec > 0 && autoCountdown > 0" class="countdown">{{ autoCountdown }}s</span>
            </template>
          </NSelect>
        </div>
        <ViewSelect
          :current-tab="currentPanelTab"
          :current-ds-id="currentPanelDsId"
          :current-ds-name="currentPanelDsName"
          :current-expression="currentPanelExpression"
          @load="onViewLoad"
        />
        <button class="execute-btn" :disabled="loading" @click="runAll">
          <span v-if="loading" class="execute-spinner">⟳</span>
          <span v-else>▶</span>
          {{ loading ? 'Running...' : 'Execute' }}
        </button>
      </div>
    </section>

    <!-- Display Mode -->
    <section class="display-row">
      <div class="segmented">
        <button :class="{ active: displayMode === 'graph' }" @click="displayMode = 'graph'">Graph</button>
        <button :class="{ active: displayMode === 'json' }" @click="displayMode = 'json'">JSON</button>
        <button :class="{ active: displayMode === 'table' }" @click="displayMode = 'table'">Table</button>
      </div>
      <div class="display-opts" :style="{ visibility: displayMode === 'graph' ? 'visible' : 'hidden' }">
        <label class="checkbox"><input type="checkbox" v-model="stacked" /> Stacked</label>
        <label class="checkbox"><input type="checkbox" v-model="logScale" /> Log scale</label>
      </div>
    </section>

    <!-- Error -->
    <div v-if="errorMsg" class="error-card">
      <div class="error-icon-wrap"><NIcon size="18" color="#fafaf9"><AlertCircleOutline /></NIcon></div>
      <div class="error-body">
        <div class="error-title">{{ t('query.queryFailed') }}</div>
        <div class="error-message">{{ errorMsg }}</div>
      </div>
      <div class="error-actions">
        <NButton size="small" @click="runAll">{{ t('common.retry') }}</NButton>
        <NButton size="small" quaternary @click="errorMsg = ''">{{ t('common.close') }}</NButton>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-container"><NSpin size="medium" /></div>

    <!-- Results Panel -->
    <section v-if="!loading" class="card results-panel">
      <!-- Graph mode -->
      <div v-if="displayMode === 'graph' && !isLogs" class="graph-area">
        <div class="graph-controls-row">
          <div class="graph-ctrl-item">
            <span class="ctrl-label">Max pts</span>
            <NInputNumber v-model:value="maxDataPoints" size="small" class="ctrl-input-sm" :min="10" :max="10000" :show-button="false" placeholder="240" />
          </div>
          <div class="graph-ctrl-item">
            <span class="ctrl-label">Min step</span>
            <NInputNumber v-model:value="minStep" size="small" class="ctrl-input-sm" :min="1" :max="3600" :show-button="false" placeholder="15" />
          </div>
          <MetricChartControls v-model="chartSettings" />
          <NButton v-if="canExport" size="tiny" quaternary @click="exportCsv">
            <template #icon><NIcon :size="14"><DownloadOutline /></NIcon></template>
          </NButton>
        </div>
        <div class="chart-container">
          <template v-if="ChartReady && VChart && chartOption">
            <component :is="VChart" ref="chartRef" :option="chartOption" :autoresize="true" class="chart-full" />
          </template>
          <div v-else class="chart-fallback">
            <p>{{ t('query.chartUnavailable') }}</p>
          </div>
        </div>
        <!-- Legend -->
        <div v-if="chartSettings.showLegend && legendItems.length" class="custom-legend-container">
          <div
            v-for="item in legendItems"
            :key="item.name"
            class="custom-legend-item"
            :class="{ 'legend-dimmed': isolatedSeries !== null && isolatedSeries !== item.name, 'legend-isolated': isolatedSeries === item.name }"
            :title="item.fullName"
            @click="toggleLegend(item.name)"
          >
            <span class="legend-color-dot" :style="{ background: item.color }" />
            <span class="legend-label">{{ item.name }}</span>
          </div>
        </div>
      </div>

      <!-- JSON mode -->
      <div v-if="displayMode === 'json'" class="json-area">
        <pre class="json-pre">{{ JSON.stringify({
          status: 'success',
          data: {
            resultType: 'matrix',
            result: (resultData?.series || []).slice(0, 8).map(s => ({
              metric: s.labels,
              values: (s.values || []).slice(0, 3).map(v => [Math.floor(Number(v.ts)), String(v.value)]),
            })),
          },
          stats: { seriesFetched: resultData?.series?.length || 0, executionTimeMsec: queryDuration },
        }, null, 2) }}</pre>
      </div>

      <!-- Table mode (metrics) -->
      <div v-if="displayMode === 'table' && !isLogs" class="table-area">
        <div class="table-controls">
          <span class="field-label">{{ t('query.limit') }}</span>
          <NSelect v-model:value="metricLimit" :options="metricLimitOptions" size="small" class="ctrl-select-sm" />
          <NButton v-if="canExport" size="tiny" quaternary @click="exportCsv">
            <template #icon><NIcon :size="14"><DownloadOutline /></NIcon></template>
            {{ t('query.exportCsv') }}
          </NButton>
        </div>
        <NDataTable
          :columns="metricColumns"
          :data="metricTableData"
          :row-key="(r: Record<string, unknown>) => String(r._key)"
          :row-props="(row: MetricTableRow) => ({ style: 'cursor: pointer', onClick: () => openLabelDrawer(row._rawLabels, row.name) })"
          size="small"
          :single-line="false"
          striped
          max-height="500"
          virtual-scroll
        />
      </div>

      <!-- Log results -->
      <div v-if="isLogs && logEntries.length" ref="logResultsRef" class="log-results">
        <LogHistogram v-if="showHistogram" :buckets="histogramBuckets" :loading="histogramLoading" class="log-histogram-container" @bar-click="onHistogramRangeChange" @brush-select="onHistogramRangeChange" />
        <div class="log-controls-row">
          <div class="log-ctrl-left">
            <div class="btn-group">
              <button :class="{ active: logMode === 'origin' }" @click="logMode = 'origin'">{{ t('query.rawMode') }}</button>
              <button :class="{ active: logMode === 'table' }" @click="logMode = 'table'">{{ t('query.logTableMode') }}</button>
            </div>
            <span class="field-label">{{ t('query.limit') }}</span>
            <NSelect v-model:value="logLimit" :options="logLimitOptions" size="small" class="ctrl-select-sm" />
            <LogViewSettings v-model:options="logOptions" />
            <FullscreenButton :target-ref="logResultsRef" />
            <NButton size="small" quaternary @click="showHistogram = !showHistogram">
              {{ showHistogram ? t('query.hideHistogram') : t('query.showHistogram') }}
            </NButton>
          </div>
          <div class="log-ctrl-right">
            <span class="results-count">
              {{ t('query.showing') }} {{ logEntries.length }}
              <template v-if="logTotal > 0"> / {{ logTotal }}</template>
              {{ t('query.entries') }}
              <NTag v-if="logTruncated" type="warning" size="tiny" :bordered="false" class="tag-ml">{{ t('query.truncated') }}</NTag>
            </span>
            <NButton v-if="canExport" size="tiny" quaternary @click="exportCsv">
              <template #icon><NIcon :size="14"><DownloadOutline /></NIcon></template>
            </NButton>
          </div>
        </div>
        <div class="log-content-area">
          <LogFieldSidebar :fields="logFields" :log-entries="logEntries" @add-field-filter="onFieldFilterAdd" />
          <div class="log-main-area">
            <div class="log-level-legend">
              <span v-for="(color, level) in LEVEL_COLORS" :key="level" class="level-item" v-show="level !== 'unknown'">
                <span class="level-dot" :style="{ background: color }" />
                <span class="level-label">{{ level }}</span>
              </span>
            </div>
            <div v-if="logMode === 'origin'" class="log-origin-view">
              <div v-for="(entry, idx) in logEntries" :key="(entry as any)._key ?? idx" class="log-origin-row" :class="logRowClassName(entry)" @click="openLogDrawer(idx)">
                <span v-if="logOptions.showLineNum" class="origin-line-num">{{ idx + 1 }}</span>
                <span class="origin-level-dot" :style="{ background: LEVEL_COLORS[detectLogLevel(entry)] }" />
                <span v-if="logOptions.showTime !== false" class="origin-time">{{ fmtTs(entry.timestamp) }}</span>
                <span class="origin-message" :style="{ whiteSpace: logOptions.lineBreak ? 'pre-wrap' : 'nowrap' }">{{ entry.message || '-' }}</span>
                <div v-if="logOptions.showLabels !== false" class="origin-labels">
                  <span v-for="([k, v], i) in Object.entries(entry.labels || {}).slice(0, 6)" :key="i" class="origin-label-pair">
                    <span class="origin-label-key">{{ k }}</span>=<FieldValueToken :field-key="k" :field-value="String(v ?? '')" @filter="(key: string, value: string, op: string) => onTokenFilter(key, value, op)" />
                  </span>
                </div>
              </div>
            </div>
            <NDataTable v-if="logMode === 'table'" :columns="logColumnsEnhanced" :data="logEntries" :row-key="(r: Record<string, unknown>) => String(r._key)" :row-class-name="logRowClassName" size="small" max-height="600" virtual-scroll />
          </div>
        </div>
      </div>

      <!-- No results -->
      <div v-if="!errorMsg && !isLogs && displayMode === 'graph' && resultData && !resultData?.series?.length" class="query-empty">{{ t('query.noResults') }}</div>
      <div v-if="!errorMsg && isLogs && !logEntries.length && resultData === null" class="query-empty">{{ t('query.noResults') }}</div>
    </section>

    <!-- Status Bar -->
    <footer class="status-bar">
      <span>{{ statusSeries }} series &middot; {{ statusPoints }} points</span>
      <span class="status-sep">&middot;</span>
      <span>Query took {{ queryDuration }}ms</span>
      <span class="status-sep">&middot;</span>
      <span>{{ statusRangeLabel }}</span>
    </footer>

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
          <NButton type="primary" size="small" @click="goToCreateAlertRule(buildExpression(labelDrawerData))">
            <template #icon><NIcon :component="AddOutline" /></template>
            {{ t('query.addToAlertRule') }}
          </NButton>
        </div>
      </NDrawerContent>
    </NDrawer>

    <!-- Log Detail Drawer -->
    <LogDetailDrawer v-model:show="drawerVisible" :log-entry="drawerLogEntry" :log-entries="logEntries" :current-index="drawerCurrentIndex" @prev="onDrawerPrev" @next="onDrawerNext" />
  </div>
</template>

<style scoped>
/* ===== Page Layout ===== */
.query-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 20px;
  max-width: 1600px;
  height: 100%;
}

.card {
  background: var(--sre-bg-card, #ffffff);
  border: 1px solid var(--sre-border, #e5e7eb);
  border-radius: 8px;
  box-shadow: 0 1px 2px rgba(0,0,0,0.04);
}

/* ===== Query Section ===== */
.queries-section {
  padding: 12px 16px;
}

/* ===== Add Query Button ===== */
.add-query-btn {
  margin-top: 8px;
  margin-left: 36px;
  background: transparent;
  border: 1px dashed var(--sre-border, #e5e7eb);
  border-radius: 6px;
  padding: 6px 12px;
  color: var(--sre-text-tertiary, #94a3b8);
  cursor: pointer;
  font-size: 13px;
  font-family: inherit;
  transition: all 0.15s;
}
.add-query-btn:hover {
  border-color: var(--sre-primary, #0D9488);
  color: var(--sre-primary, #0D9488);
}

/* ===== Controls Row ===== */
.controls-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.range-group {
  display: inline-flex;
  border: 1px solid var(--sre-border, #e5e7eb);
  border-radius: 6px;
  overflow: hidden;
  background: var(--sre-bg-card, #fff);
}
.range-btn {
  height: 30px;
  padding: 0 12px;
  background: var(--sre-bg-card, #fff);
  border: none;
  border-right: 1px solid var(--sre-border, #e5e7eb);
  color: var(--sre-text-tertiary, #94a3b8);
  font-size: 12px;
  cursor: pointer;
  font-family: inherit;
  transition: background 0.15s, color 0.15s;
}
.range-btn:last-child { border-right: none; }
.range-btn:hover { background: var(--sre-bg-hover, #f1f5f9); }
.range-btn.active { background: var(--sre-primary, #0D9488); color: #fff; }

.custom-range-inline { margin-left: 8px; }
.custom-date-picker { width: 420px; }

.controls-spacer { flex: 1; }
.controls-right {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

/* Tab group (Metrics / Logs) */
.tab-group {
  display: inline-flex;
  background: var(--sre-bg-hover, #f1f5f9);
  padding: 3px;
  border-radius: 8px;
}
.tab-btn {
  padding: 5px 16px;
  background: transparent;
  border: none;
  color: var(--sre-text-tertiary, #94a3b8);
  font-size: 13px;
  cursor: pointer;
  border-radius: 6px;
  font-family: inherit;
  transition: all 0.15s;
}
.tab-btn.active {
  background: var(--sre-bg-card, #fff);
  color: var(--sre-text-primary, #1e293b);
  box-shadow: 0 1px 2px rgba(0,0,0,0.04);
  font-weight: 500;
}

.ctrl-compact {
  display: inline-flex;
  align-items: center;
  height: 30px;
  border: 1px solid var(--sre-border, #e5e7eb);
  border-radius: 6px;
  background: var(--sre-bg-card, #fff);
  overflow: hidden;
}
.ctrl-label {
  padding: 0 10px;
  color: var(--sre-text-tertiary, #94a3b8);
  font-size: 12px;
  border-right: 1px solid var(--sre-border, #e5e7eb);
  height: 100%;
  display: inline-flex;
  align-items: center;
}
.ctrl-select-sm { width: 100px; }
.ctrl-input-sm { width: 70px; }

.countdown {
  color: var(--sre-primary, #0D9488);
  font-weight: 500;
  font-size: 12px;
}

.execute-btn {
  height: 30px;
  padding: 0 14px;
  border-radius: 6px;
  background: var(--sre-primary, #0D9488);
  color: #fff;
  border: none;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  font-family: inherit;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: filter 0.15s;
}
.execute-btn:hover { filter: brightness(1.08); }
.execute-btn:disabled { opacity: 0.6; cursor: wait; }
.execute-spinner { animation: spin 0.8s linear infinite; display: inline-block; }
@keyframes spin { to { transform: rotate(360deg); } }

/* ===== Display Mode ===== */
.display-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.segmented {
  display: inline-flex;
  background: var(--sre-bg-hover, #f1f5f9);
  padding: 3px;
  border-radius: 8px;
}
.segmented button {
  padding: 5px 16px;
  background: transparent;
  border: none;
  color: var(--sre-text-tertiary, #94a3b8);
  font-size: 13px;
  cursor: pointer;
  border-radius: 6px;
  font-family: inherit;
  transition: all 0.15s;
}
.segmented button.active {
  background: var(--sre-bg-card, #fff);
  color: var(--sre-text-primary, #1e293b);
  box-shadow: 0 1px 2px rgba(0,0,0,0.04);
  font-weight: 500;
}
.display-opts {
  display: flex;
  gap: 16px;
}
.checkbox {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--sre-text-tertiary, #94a3b8);
  cursor: pointer;
  user-select: none;
}
.checkbox input { margin: 0; cursor: pointer; }

/* ===== Results Panel ===== */
.results-panel {
  flex: 1;
  min-height: 440px;
  padding: 12px;
  display: flex;
  flex-direction: column;
}

/* Graph */
.graph-area { flex: 1; display: flex; flex-direction: column; }
.graph-controls-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--sre-border, #e5e7eb);
  margin-bottom: 12px;
}
.graph-ctrl-item {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.chart-container { min-height: 400px; display: flex; align-items: center; justify-content: center; position: relative; flex: 1; }
.chart-fallback { display: flex; flex-direction: column; align-items: center; gap: 12px; color: var(--sre-text-tertiary); font-size: 13px; }
.chart-full { width: 100%; height: 400px; }

/* Legend */
.custom-legend-container {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 12px;
  padding: 8px 4px;
  max-height: 120px;
  overflow-y: auto;
  overflow-x: hidden;
  border-top: 1px solid var(--sre-border, #e5e7eb);
  margin-top: 4px;
  cursor: pointer;
  user-select: none;
}
.custom-legend-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  color: var(--sre-text-secondary);
  transition: opacity 0.15s, background 0.15s;
  max-width: 280px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.custom-legend-item:hover { background: var(--sre-bg-hover); }
.legend-dimmed { opacity: 0.3; }
.legend-isolated { opacity: 1; background: var(--sre-bg-hover); font-weight: 600; }
.legend-color-dot { width: 12px; height: 4px; border-radius: 2px; flex-shrink: 0; }
.legend-label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* JSON */
.json-area { flex: 1; overflow: auto; }
.json-pre {
  margin: 0;
  padding: 12px;
  background: var(--sre-bg-sunken, #f8fafc);
  border-radius: 6px;
  font-size: 12px;
  font-family: var(--sre-font-mono, monospace);
  color: var(--sre-text-primary);
  line-height: 1.55;
  overflow: auto;
}

/* Table */
.table-area { flex: 1; overflow: auto; }
.table-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--sre-border, #e5e7eb);
  margin-bottom: 12px;
}
.field-label { font-size: 12px; color: var(--sre-text-tertiary, #94a3b8); }

/* Log Results */
.log-results { flex: 1; overflow: visible; display: flex; flex-direction: column; }
.log-histogram-container { margin-bottom: 8px; }
.log-controls-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--sre-border, #e5e7eb);
  margin-bottom: 8px;
  flex-wrap: wrap;
}
.log-ctrl-left { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.log-ctrl-right { display: flex; gap: 8px; align-items: center; }
.btn-group {
  display: inline-flex;
  border: 1px solid var(--sre-border, #e5e7eb);
  border-radius: 6px;
  overflow: hidden;
  background: var(--sre-bg-card, #fff);
}
.btn-group button {
  height: 30px;
  padding: 0 12px;
  background: var(--sre-bg-card, #fff);
  border: none;
  border-right: 1px solid var(--sre-border, #e5e7eb);
  color: var(--sre-text-tertiary, #94a3b8);
  font-size: 12px;
  cursor: pointer;
  font-family: inherit;
  transition: background 0.15s, color 0.15s;
}
.btn-group button:last-child { border-right: none; }
.btn-group button:hover { background: var(--sre-bg-hover); }
.btn-group button.active { background: var(--sre-primary, #0D9488); color: #fff; }

.results-count { font-size: 13px; color: var(--sre-text-secondary); }
.tag-ml { margin-left: 4px; }

/* Log content */
.log-content-area { display: flex; gap: 0; min-height: 400px; }
.log-main-area { flex: 1; min-width: 0; overflow: auto; display: flex; flex-direction: column; }
.log-level-legend { display: flex; gap: 12px; margin-bottom: 8px; padding: 4px 0; flex-shrink: 0; }
.level-item { display: flex; align-items: center; gap: 4px; font-size: 11px; color: var(--sre-text-tertiary); }
.level-dot { width: 8px; height: 8px; border-radius: 2px; flex-shrink: 0; }
.level-label { text-transform: uppercase; font-weight: 500; letter-spacing: 0.5px; }

/* Log origin view */
.log-origin-view { flex: 1; overflow-y: auto; font-family: var(--sre-font-mono, monospace); font-size: 12px; line-height: 1.6; }
.log-origin-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 4px 8px;
  cursor: pointer;
  border-bottom: 1px solid var(--sre-border-light, rgba(0,0,0,0.04));
  transition: background 0.15s;
}
.log-origin-row:hover { background: var(--sre-bg-hover); }
.origin-level-dot { width: 4px; min-height: 16px; border-radius: 2px; flex-shrink: 0; margin-top: 3px; }
.origin-line-num { flex-shrink: 0; min-width: 28px; text-align: right; padding-right: 6px; color: var(--sre-text-tertiary); font-size: 11px; user-select: none; opacity: 0.6; }
.origin-time { flex-shrink: 0; color: var(--sre-text-tertiary); font-size: 11px; white-space: nowrap; }
.origin-message { flex: 1; min-width: 0; color: var(--sre-text-primary); overflow: hidden; text-overflow: ellipsis; }
.origin-labels { flex-shrink: 0; display: flex; gap: 2px; flex-wrap: wrap; max-width: 300px; }
.origin-label-pair { font-size: 11px; font-family: var(--sre-font-mono, monospace); color: var(--sre-text-secondary); max-width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.origin-label-key { color: var(--sre-primary, #0D9488); }

/* Empty / Loading */
.query-empty { display: flex; align-items: center; justify-content: center; min-height: 200px; color: var(--sre-text-tertiary); font-size: 14px; }
.loading-container { display: flex; justify-content: center; padding: 40px; }

/* Error */
.error-card { display: flex; align-items: flex-start; gap: 12px; padding: 16px; background: var(--sre-critical-soft); border: 1px solid var(--sre-critical-soft); border-radius: var(--sre-radius-md, 8px); }
.error-icon-wrap { flex-shrink: 0; width: 32px; height: 32px; border-radius: 50%; background: var(--sre-critical); display: flex; align-items: center; justify-content: center; }
.error-body { flex: 1; min-width: 0; }
.error-title { font-size: 14px; font-weight: 600; color: var(--sre-text-primary); margin: 0 0 4px; }
.error-message { font-size: 12px; color: var(--sre-text-secondary); word-break: break-all; }
.error-actions { display: flex; gap: 8px; flex-shrink: 0; margin-top: 2px; }

/* Status Bar */
.status-bar {
  font-size: 12px;
  color: var(--sre-text-tertiary, #94a3b8);
  display: flex;
  gap: 8px;
  align-items: center;
  padding: 2px 4px 4px;
  flex-shrink: 0;
}
.status-sep { color: var(--sre-border, #e5e7eb); }

/* Label Drawer */
.label-drawer-header { margin-bottom: 16px; }
.label-drawer-series { font-size: 14px; font-weight: 600; color: var(--sre-text-primary); font-family: var(--sre-font-mono, monospace); }
.label-drawer-value { font-family: var(--sre-font-mono, monospace); font-size: 12px; word-break: break-all; }
.label-drawer-actions { margin-top: 20px; display: flex; justify-content: flex-end; }

:deep(.log-row-error) { background: rgba(239, 68, 68, 0.04) !important; }
:deep(.log-row-warn) { background: rgba(234, 179, 8, 0.04) !important; }

@media (max-width: 768px) {
  .query-page { padding: 12px; }
  .controls-row { flex-direction: column; align-items: flex-start; }
  .controls-spacer { display: none; }
  .custom-date-picker { width: 100%; }
}
</style>
