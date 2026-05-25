<script setup lang="ts">
/**
 * QueryPanelContent — per-panel query editor + results.
 * Used by the multi-panel Explore page.
 *
 * Each panel manages its own: datasource, expression, tab, result mode, data.
 * Shared state (time range, datasources list) comes from parent via props.
 */
import { ref, computed, watch, h, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NSelect, NButton, NSpace, NTag, NSpin,
  NDataTable, NTabs, NTabPane, NPopover, NIcon, NTooltip,
  NButtonGroup, NDrawer, NDrawerContent, NDescriptions, NDescriptionsItem,
  useMessage,
} from 'naive-ui'
import {
  TimeOutline, TrashOutline, DownloadOutline, AlertCircleOutline,
  AddOutline, CloseCircleOutline,
} from '@vicons/ionicons5'
import { datasourceApi } from '@/api'
import { formatTime } from '@/utils/format'
import PromQLEditor from './PromQLEditor.vue'
import LogsQLEditor from './LogsQLEditor.vue'
import LogHistogram from './LogHistogram.vue'
import MetricChartControls from './MetricChartControls.vue'
import type { ChartSettings } from './MetricChartControls.vue'
import LogDetailDrawer from './LogDetailDrawer.vue'
import LogFieldSidebar from './LogFieldSidebar.vue'
import LogViewSettings from './LogViewSettings.vue'
import FieldValueToken from './FieldValueToken.vue'
import FullscreenButton from './FullscreenButton.vue'
import type { DataSource, QueryResponse, LogEntry } from '@/types'

const props = defineProps<{
  panelId: number
  datasources: DataSource[]
  timeStart: number
  timeEnd: number
  stepValue: string
  ChartReady: boolean
  VChart: Component | null
  canClose?: boolean
}>()

const emit = defineEmits<{
  (e: 'remove', panelId: number): void
  (e: 'openLabels', labels: Record<string, string>, seriesName: string): void
  (e: 'timeRangeChange', start: number, end: number): void
}>()

// Local copy of stepValue so we can use v-model
const localStepValue = ref(props.stepValue)

const { t } = useI18n()
const message = useMessage()

// --- Panel-local state ---
type QueryMode = 'instant' | 'range'
type ResultMode = 'chart' | 'table'
type QueryTab = 'metrics' | 'logs'

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
const queryMode = ref<QueryMode>('range')

// Log histogram
interface HistogramBucket { timestamp: string; count: number }
const histogramBuckets = ref<HistogramBucket[]>([])
const histogramLoading = ref(false)
const showHistogram = ref(true)

// Query stats
interface QueryStats { executionTimeMs: number; resultCount: number; step?: string }
const queryStats = ref<QueryStats | null>(null)

// Chart settings (PromGraphCpt-style)
const chartSettings = ref<ChartSettings>({
  maxDataPoints: null,
  minStep: null,
  chartType: 'line',
  showLegend: true,
  sharedTooltip: false,
  tooltipSort: 'desc',
})

// Per-panel graph time range (Nightingale Graph.tsx: each panel has own TimeRangePicker)
const graphRangeMin = ref<number>(-1) // -1 means use parent range
const graphNow = ref(Date.now())

const graphPresetOptions = [
  { label: '5m', value: 5 },
  { label: '15m', value: 15 },
  { label: '30m', value: 30 },
  { label: '1h', value: 60 },
  { label: '3h', value: 180 },
  { label: '6h', value: 360 },
  { label: '12h', value: 720 },
  { label: '24h', value: 1440 },
]

const graphTimeStart = computed(() => {
  if (graphRangeMin.value === -1) return props.timeStart
  return Math.floor((graphNow.value - graphRangeMin.value * 60000) / 1000)
})
const graphTimeEnd = computed(() => {
  if (graphRangeMin.value === -1) return props.timeEnd
  return Math.floor(graphNow.value / 1000)
})

function selectGraphPreset(v: number) {
  graphRangeMin.value = v
  graphNow.value = Date.now()
}

function resetGraphRange() {
  graphRangeMin.value = -1
}

// Log mode (Nightingale: origin/table toggle)
type LogMode = 'origin' | 'table'
const logMode = ref<LogMode>('origin')
const logOptions = ref({
  lineBreak: true,
  showTime: true,
  showLabels: true,
  showLineNum: false,
  jsonExpandLevel: 2,
})

// Computed log fields from entries
const logFields = computed(() => {
  const fieldSet = new Set<string>()
  for (const entry of logEntries.value) {
    if (entry.labels) {
      for (const k of Object.keys(entry.labels)) fieldSet.add(k)
    }
  }
  return Array.from(fieldSet).sort()
})

// Limits
const metricLimit = ref(100)
const metricLimitOptions = [50, 100, 200, 500, 1000].map(v => ({ label: String(v), value: v }))
const logLimit = ref(200)
const logLimitOptions = [50, 100, 200, 500, 1000, 5000].map(v => ({ label: String(v), value: v }))

// Step options
const stepOptions = computed(() => [
  { label: t('query.stepAuto'), value: 'auto' },
  { label: '15s', value: '15s' },
  { label: '30s', value: '30s' },
  { label: '1m', value: '1m' },
  { label: '5m', value: '5m' },
  { label: '15m', value: '15m' },
  { label: '1h', value: '1h' },
])

// Log detail drawer (NavigableDrawer pattern)
const drawerVisible = ref(false)
const drawerCurrentIndex = ref(0)
const drawerLogEntry = computed(() => logEntries.value[drawerCurrentIndex.value] || null)

function openLogDrawer(index: number) {
  drawerCurrentIndex.value = index
  drawerVisible.value = true
}
function onDrawerPrev() { if (drawerCurrentIndex.value > 0) drawerCurrentIndex.value-- }
function onDrawerNext() { if (drawerCurrentIndex.value < logEntries.value.length - 1) drawerCurrentIndex.value++ }

// History — per-datasource (Nightingale HistoricalRecords pattern)
type HistoryItem = { tab: QueryTab; expression: string; ts: number }
const HISTORY_BASE_KEY = 'sre-query-history'
const history = ref<HistoryItem[]>([])
const historyVisible = ref(false)

function historyKey(): string {
  return selectedDsId.value ? `${HISTORY_BASE_KEY}-${selectedDsId.value}` : HISTORY_BASE_KEY
}

function loadHistory() {
  try {
    const raw = localStorage.getItem(historyKey())
    if (raw) history.value = JSON.parse(raw) || []
    else history.value = []
  } catch { history.value = [] }
}

function pushHistory(tab: QueryTab, expr: string) {
  if (!expr.trim()) return
  const key = historyKey()
  const list = history.value.filter(h => !(h.tab === tab && h.expression === expr))
  list.unshift({ tab, expression: expr, ts: Date.now() })
  history.value = list.slice(0, 50)
  try { localStorage.setItem(key, JSON.stringify(history.value)) } catch { /* ignore */ }
}

const filteredHistory = computed(() =>
  history.value.filter(h => h.tab === activeTab.value).slice(0, 15)
)

// --- Computed ---
const selectedDs = computed(() => props.datasources.find(d => d.id === selectedDsId.value))
const metricDatasources = computed(() => props.datasources.filter(d => d.supports_query && d.type !== 'victorialogs'))
const logDatasources = computed(() => props.datasources.filter(d => d.type === 'victorialogs'))
const isLogs = computed(() => activeTab.value === 'logs')

// Datasource selector options (filtered by active tab)
const datasourceOptions = computed(() => {
  const list = isLogs.value ? logDatasources.value : metricDatasources.value
  return list.map(d => ({ label: `${d.name} (${typeBadge(d.type)})`, value: d.id }))
})

// Ref for fullscreen target
const logResultsRef = ref<HTMLElement | null>(null)
const isMetricLimited = computed(() => {
  if (!metricData.value?.series) return false
  return metricData.value.series.length >= metricLimit.value
})

function dsLabel(ds: DataSource): string { return `${ds.name} (${typeBadge(ds.type)})` }
function typeBadge(tp: string): string {
  const m: Record<string, string> = { prometheus: 'Prometheus', victoriametrics: 'VictoriaMetrics', victorialogs: 'VictoriaLogs', zabbix: 'Zabbix' }
  return m[tp] || tp
}
function typeColor(tp: string): string {
  const tokenMap: Record<string, string> = {
    prometheus: '--sre-ds-prometheus', victoriametrics: '--sre-ds-victoriametrics',
    victorialogs: '--sre-ds-victorialogs', zabbix: '--sre-ds-zabbix',
  }
  const token = tokenMap[tp]
  if (token && typeof document !== 'undefined') {
    const val = getComputedStyle(document.documentElement).getPropertyValue(token).trim()
    return val || '#64748b'
  }
  return '#64748b'
}

// --- Log level detection ---
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

// --- Resolve step (Nightingale: maxDataPoints/minStep affect step calculation) ---
function resolveStep(): string {
  if (localStepValue.value !== 'auto') return localStepValue.value
  const start = graphTimeStart.value
  const end = graphTimeEnd.value
  const duration = end - start
  const maxPts = chartSettings.value.maxDataPoints || 240
  const computedStep = Math.ceil(duration / maxPts)
  const minStep = chartSettings.value.minStep || 15
  return `${Math.max(computedStep, minStep)}s`
}

// --- Actions ---
async function run() {
  if (!selectedDsId.value || !expression.value.trim()) return
  const startTime = Date.now()
  loading.value = true
  errorMsg.value = ''
  metricData.value = null
  logEntries.value = []
  try {
    if (isLogs.value) {
      const res = await datasourceApi.logQuery(selectedDsId.value, {
        expression: expression.value,
        start: props.timeStart,
        end: props.timeEnd,
        limit: logLimit.value,
      })
      const data = res.data?.data
      if (data) {
        logEntries.value = (data.entries || []).map((e: LogEntry, i: number) => ({ ...e, _key: i }))
        logTotal.value = data.total || 0
        logTruncated.value = data.truncated || false
      }
      // Fetch histogram
      if (showHistogram.value) {
        fetchHistogram()
      }
    } else if (queryMode.value === 'instant') {
      const res = await datasourceApi.query(selectedDsId.value, {
        expression: expression.value,
        time: graphTimeEnd.value,
      })
      const data = res.data?.data
      if (data?.series && data.series.length > metricLimit.value) data.series = data.series.slice(0, metricLimit.value)
      metricData.value = data
    } else {
      const res = await datasourceApi.rangeQuery(selectedDsId.value, {
        expression: expression.value,
        start: graphTimeStart.value,
        end: graphTimeEnd.value,
        step: resolveStep(),
      })
      const data = res.data?.data
      if (data?.series && data.series.length > metricLimit.value) data.series = data.series.slice(0, metricLimit.value)
      metricData.value = data
    }
    queryStats.value = { executionTimeMs: Date.now() - startTime, resultCount: isLogs.value ? logEntries.value.length : (metricData.value?.series?.length || 0), step: isLogs.value ? undefined : resolveStep() }
    pushHistory(activeTab.value, expression.value)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string; message?: string } }; message?: string }
    errorMsg.value = err?.response?.data?.error || err?.response?.data?.message || err?.message || t('query.queryFailed')
  } finally {
    loading.value = false
  }
}

async function fetchHistogram() {
  if (!selectedDsId.value || !expression.value.trim()) { histogramBuckets.value = []; return }
  histogramLoading.value = true
  try {
    const res = await datasourceApi.logHistogram(selectedDsId.value, { expression: expression.value, start: props.timeStart, end: props.timeEnd })
    histogramBuckets.value = res.data?.data?.buckets || []
  } catch { histogramBuckets.value = [] }
  finally { histogramLoading.value = false }
}

function onHistogramBarClick(start: number, end: number) {
  // Emit to parent to zoom time range to the clicked bar
  emit('timeRangeChange', start, end)
}

function onHistogramBrushSelect(start: number, end: number) {
  // Emit to parent to zoom time range to the brushed selection
  emit('timeRangeChange', start, end)
}

function onFieldFilterAdd(key: string, value: string) {
  // Add field filter to query expression (Nightingale: AND filter pattern)
  const filterExpr = `${key}="${value}"`
  if (expression.value.trim()) {
    expression.value = expression.value.trim() + ', ' + filterExpr
  } else {
    expression.value = filterExpr
  }
}

function onTokenFilter(key: string, value: string, operator: string) {
  if (operator === 'AND') {
    const filterExpr = `${key}="${value}"`
    if (expression.value.trim()) {
      expression.value = expression.value.trim() + ', ' + filterExpr
    } else {
      expression.value = filterExpr
    }
  } else if (operator === 'NOT') {
    const filterExpr = `${key}!="${value}"`
    if (expression.value.trim()) {
      expression.value = expression.value.trim() + ', ' + filterExpr
    } else {
      expression.value = filterExpr
    }
  }
  run()
}

// --- Chart option ---
const chartOption = computed(() => {
  if (!metricData.value?.series?.length) return null
  interface EChartsSeries { name: string; type: string; data: [number, number][]; smooth: boolean; showSymbol: boolean; connectNulls: boolean; areaStyle?: { opacity: number } }
  const seriesList: EChartsSeries[] = []
  const isArea = chartSettings.value.chartType === 'area'
  for (const s of metricData.value.series) {
    const name = formatLegend(s.labels)
    const data: [number, number][] = []
    for (const v of s.values || []) data.push([Number(v.ts) * 1000, v.value != null ? Number(v.value) : 0])
    const seriesItem: EChartsSeries = { name, type: 'line', data, smooth: true, showSymbol: false, connectNulls: true }
    if (isArea) seriesItem.areaStyle = { opacity: 0.5 }
    seriesList.push(seriesItem)
  }
  const tertiaryColor = typeof document !== 'undefined' ? getComputedStyle(document.documentElement).getPropertyValue('--sre-text-tertiary').trim() || '#64748b' : '#64748b'
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: chartSettings.value.sharedTooltip ? 'axis' : 'item',
      confine: true,
      ...(chartSettings.value.sharedTooltip ? { axisPointer: { type: 'cross' } } : {}),
    },
    legend: chartSettings.value.showLegend ? {
      type: 'scroll', bottom: 0,
      textStyle: { color: tertiaryColor, fontSize: 12 },
    } : { show: false },
    grid: { left: 80, right: 20, top: 20, bottom: chartSettings.value.showLegend ? 50 : 20 },
    xAxis: { type: 'time', axisLabel: { fontSize: 11 } },
    yAxis: { type: 'value', axisLabel: { fontSize: 11 }, splitLine: { lineStyle: { type: 'dashed' } } },
    series: seriesList,
    dataZoom: [{ type: 'inside', start: 0, end: 100 }],
  }
})

// --- Table columns ---
interface MetricTableRow { _key: number; name: string; value: string; labels: string; _rawLabels: Record<string, string>; _rawExpression: string }

const metricColumns = computed(() => [
  { title: t('query.metricName'), key: 'name', ellipsis: { tooltip: true }, width: 200 },
  { title: t('query.value'), key: 'value', width: 160 },
  { title: t('query.labelsHeader'), key: 'labels', ellipsis: { tooltip: true } },
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

const logColumnsEnhanced = computed(() => [
  {
    title: '',
    key: 'expand',
    width: 40,
    render: (_r: LogEntry, index: number) =>
      h(NTooltip, { trigger: 'hover' }, {
        trigger: () => h('div', {
          style: 'width:100%;height:100%;display:flex;align-items:center;justify-content:center;cursor:pointer;',
          onClick: () => openLogDrawer(index),
        }, [h('span', { style: 'font-size:14px;color:var(--sre-text-tertiary);' }, '\u{1F50D}')]),
        default: () => t('query.logDetailTip'),
      }),
  },
  { title: '', key: 'level', width: 6, render: (r: LogEntry) => h('div', { style: { width: '4px', height: '100%', minHeight: '20px', borderRadius: '2px', background: LEVEL_COLORS[detectLogLevel(r)] } }) },
  { title: t('query.logTime'), key: 'timestamp', width: 180, render: (r: LogEntry) => h('span', { style: { fontFamily: 'var(--sre-font-mono, monospace)', fontSize: '12px' } }, fmtTs(r.timestamp)) },
  { title: t('query.logMessage'), key: 'message', ellipsis: { tooltip: true }, render: (r: LogEntry) => h('span', { style: { fontFamily: 'var(--sre-font-mono, monospace)', fontSize: '12px', whiteSpace: 'pre-wrap', wordBreak: 'break-all', color: (detectLogLevel(r) === 'error' || detectLogLevel(r) === 'fatal') ? '#ef4444' : detectLogLevel(r) === 'warn' ? '#eab308' : undefined } }, r.message || '-') },
  { title: t('query.logLabels'), key: '_labels', width: 280, ellipsis: { tooltip: true }, render: (r: LogEntry) => {
    const labels = r.labels || {}
    const entries = Object.entries(labels)
    if (entries.length === 0) return '-'
    return h('div', { style: 'display:flex;flex-wrap:wrap;gap:2px;' }, entries.slice(0, 6).map(([k, v]) => h(NTag, { size: 'tiny', bordered: false, style: 'cursor:pointer;max-width:140px;', onClick: (e: MouseEvent) => { e.stopPropagation(); copyFieldValue(k, v) } }, { default: () => `${k}=${v}` })))
  } },
])

// --- Helpers ---
function copyFieldValue(k: string, v: unknown) {
  window.navigator?.clipboard?.writeText(`${k}=${v}`)
  message.success(`${t('query.copiedField')}: ${k}=${v}`)
}
function formatLegend(lbs: Record<string, string>): string {
  if (!lbs) return 'value'
  const parts: string[] = []
  for (const k of Object.keys(lbs)) { if (k !== '__name__') parts.push(`${k}="${lbs[k]}"`) }
  return parts.length ? parts.join(', ') : (lbs.__name__ || 'value')
}
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

// --- CSV Export ---
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
  return resultMode.value === 'table' && metricTableData.value.length > 0
})

// --- Watch ---
watch(selectedDsId, () => { expression.value = ''; metricData.value = null; logEntries.value = []; histogramBuckets.value = []; errorMsg.value = ''; loadHistory() })
watch(activeTab, () => { selectedDsId.value = null; expression.value = ''; metricData.value = null; logEntries.value = []; histogramBuckets.value = []; errorMsg.value = '' })

// Load history on mount
loadHistory()

// Methods for parent to set state (used for URL sync)
function setState(ds: number | null, expr: string, tab?: string) {
  if (ds != null) selectedDsId.value = ds
  if (expr) expression.value = expr
  if (tab === 'metrics' || tab === 'logs') activeTab.value = tab
}

// Expose for parent
defineExpose({ run, setState, activeTab, expression, selectedDsId })
</script>

<template>
  <div class="panel-content">
    <!-- Panel header: datasource + tabs + close (Nightingale: Row gutter=8) -->
    <div class="panel-top-row">
      <div class="panel-top-left">
        <NSelect
          v-model:value="selectedDsId"
          :options="datasourceOptions"
          :placeholder="t('query.selectDatasource')"
          filterable
          size="small"
          class="ds-select"
        />
        <NTabs v-model:value="activeTab" type="line" size="small" class="panel-tabs-inline">
          <NTabPane name="metrics" :tab="t('query.metricsTab')" />
          <NTabPane name="logs" :tab="t('query.logsTab')" />
        </NTabs>
      </div>
      <NButton v-if="canClose" size="tiny" quaternary @click="emit('remove', panelId)">
        <template #icon><NIcon><CloseCircleOutline /></NIcon></template>
      </NButton>
    </div>

    <div v-if="isLogs && !logDatasources.length" class="query-empty-inline">
      {{ t('query.noLogDatasources') }}
    </div>

    <!-- Editor + Execute (Nightingale: flex gap-[8px] side-by-side) -->
    <div v-if="selectedDsId != null" class="editor-row">
      <div class="editor-input-wrap">
        <PromQLEditor v-if="!isLogs" v-model="expression" :datasource-id="selectedDsId" :placeholder="t('query.promqlPlaceholder')" @execute="run" />
        <LogsQLEditor v-else v-model="expression" :datasource-id="selectedDsId" :placeholder="t('query.logQueryPlaceholder')" @execute="run" />
      </div>
      <div class="editor-actions">
        <NPopover v-model:show="historyVisible" trigger="click" placement="bottom-end">
          <template #trigger>
            <NTooltip><template #trigger><NButton size="small" quaternary><template #icon><NIcon><TimeOutline /></NIcon></template></NButton></template>{{ t('query.queryHistory') }}</NTooltip>
          </template>
          <div class="history-pop">
            <div class="history-title">{{ t('query.recentQueries') }}</div>
            <div v-if="!filteredHistory.length" class="history-empty">{{ t('query.noHistory') }}</div>
            <div v-for="item in filteredHistory" :key="item.ts" class="history-item" @click="expression = item.expression; historyVisible = false">
              <div class="history-expr">{{ item.expression }}</div>
              <div class="history-ts">{{ fmtTs(item.ts) }}</div>
            </div>
          </div>
        </NPopover>
        <NTooltip><template #trigger><NButton size="small" quaternary :disabled="!expression" @click="expression = ''"><template #icon><NIcon><TrashOutline /></NIcon></template></NButton></template>{{ t('query.clearBtn') }}</NTooltip>
        <NButton type="primary" size="small" :loading="loading" :disabled="!selectedDsId || !expression.trim()" @click="run">
          {{ t('query.runQuery') }}
        </NButton>
      </div>
    </div>

    <!-- Pre-query controls (instant/range, step, limit) -->
    <div v-if="selectedDsId != null" class="pre-query-controls">
      <NSpace :size="8" align="center">
        <template v-if="!isLogs">
          <NButtonGroup size="small">
            <NButton :type="queryMode === 'instant' ? 'primary' : 'default'" :secondary="queryMode !== 'instant'" @click="queryMode = 'instant'">{{ t('query.instant') }}</NButton>
            <NButton :type="queryMode === 'range' ? 'primary' : 'default'" :secondary="queryMode !== 'range'" @click="queryMode = 'range'">{{ t('query.range') }}</NButton>
          </NButtonGroup>
          <span class="field-label">{{ t('query.step') }}</span>
          <NSelect v-model:value="localStepValue" :options="stepOptions" size="small" class="control-select-sm" />
          <span class="field-label">{{ t('query.limit') }}</span>
          <NSelect v-model:value="metricLimit" :options="metricLimitOptions" size="small" class="control-select-sm" />
        </template>
        <template v-else>
          <span class="field-label">{{ t('query.limit') }}</span>
          <NSelect v-model:value="logLimit" :options="logLimitOptions" size="small" class="control-select-sm" />
        </template>
      </NSpace>
      <span class="shortcut-hint">{{ t('query.shortcutHint') }}</span>
    </div>

    <div v-if="selectedDsId == null" class="query-empty-inline" />

    <!-- Error -->
    <div v-if="errorMsg" class="error-card">
      <div class="error-icon-wrap"><NIcon size="18" color="#fafaf9"><AlertCircleOutline /></NIcon></div>
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
    <div v-if="loading" class="loading-container"><NSpin size="medium" /></div>

    <!-- ============================================ -->
    <!-- Metrics Results (Nightingale: card-style tabs) -->
    <!-- ============================================ -->
    <div v-if="!loading && !isLogs && metricData?.series?.length" class="metrics-results">
      <!-- Card tabs: Table / Graph (Nightingale PromGraphCpt type='card') -->
      <NTabs v-model:value="resultMode" type="card" size="small" class="metric-card-tabs"
        :tab-bar-style="{ marginBottom: 0 }"
      >
        <template #suffix>
          <div class="card-tabs-suffix">
            <span class="results-count">
              {{ metricData.series.length }} {{ t('query.seriesCount') }}
              <NTag v-if="isMetricLimited" type="warning" size="tiny" :bordered="false" class="tag-ml">{{ t('query.limitedTo', { n: metricLimit }) }}</NTag>
            </span>
            <span v-if="queryStats" class="query-stats">{{ queryStats.executionTimeMs }}ms<template v-if="queryStats.step"> · step {{ queryStats.step }}</template></span>
            <NButton v-if="canExport" size="tiny" quaternary @click="exportCsv"><template #icon><NIcon :size="14"><DownloadOutline /></NIcon></template></NButton>
          </div>
        </template>

        <!-- Table Tab -->
        <NTabPane name="table" tab="Table">
          <NDataTable
            :columns="metricColumns"
            :data="metricTableData"
            :row-key="(r: Record<string, unknown>) => String(r._key)"
            :row-props="(row: MetricTableRow) => ({ style: 'cursor: pointer', onClick: () => emit('openLabels', row._rawLabels, row.name) })"
            size="small"
            :single-line="false"
            striped
            max-height="500"
            virtual-scroll
          />
        </NTabPane>

        <!-- Graph Tab (Nightingale: controls row inside graph pane) -->
        <NTabPane name="graph" tab="Graph">
          <!-- Graph Controls Row (Nightingale Graph.tsx pattern) -->
          <div class="graph-controls-row">
            <div class="graph-time-presets">
              <NButton
                v-for="opt in graphPresetOptions"
                :key="opt.value"
                size="tiny"
                :type="graphRangeMin === opt.value ? 'primary' : 'default'"
                :secondary="graphRangeMin !== opt.value"
                @click="selectGraphPreset(opt.value)"
              >{{ opt.label }}</NButton>
              <NButton
                size="tiny"
                :type="graphRangeMin === -1 ? 'primary' : 'default'"
                :secondary="graphRangeMin !== -1"
                @click="resetGraphRange"
              >Global</NButton>
            </div>
            <MetricChartControls v-model="chartSettings" />
          </div>
          <div class="chart-container">
            <template v-if="ChartReady && VChart && chartOption">
              <component :is="VChart" :option="chartOption" :autoresize="true" class="chart-full" />
            </template>
            <div v-else class="chart-fallback">
              <p>{{ t('query.chartUnavailable') }}</p>
            </div>
          </div>
        </NTabPane>
      </NTabs>
    </div>

    <!-- ============================================ -->
    <!-- Log Results (Nightingale logExplorer pattern) -->
    <!-- ============================================ -->
    <div v-if="!loading && isLogs && logEntries.length" ref="logResultsRef" class="log-results">
      <!-- Histogram (Nightingale: 120px, always on top) -->
      <LogHistogram v-if="showHistogram" :buckets="histogramBuckets" :loading="histogramLoading" class="log-histogram-container" @bar-click="onHistogramBarClick" @brush-select="onHistogramBrushSelect" />

      <!-- Log Controls Row (Nightingale: mode + settings + fullscreen) -->
      <div class="log-controls-row">
        <NSpace :size="8" align="center">
          <NButtonGroup size="small">
            <NButton :type="logMode === 'origin' ? 'primary' : 'default'" :secondary="logMode !== 'origin'" @click="logMode = 'origin'">{{ t('query.rawMode') }}</NButton>
            <NButton :type="logMode === 'table' ? 'primary' : 'default'" :secondary="logMode !== 'table'" @click="logMode = 'table'">{{ t('query.logTableMode') }}</NButton>
          </NButtonGroup>
          <LogViewSettings v-model:options="logOptions" />
          <FullscreenButton :target-ref="logResultsRef" />
          <NButton size="small" quaternary @click="showHistogram = !showHistogram">
            {{ showHistogram ? t('query.hideHistogram') : t('query.showHistogram') }}
          </NButton>
        </NSpace>
        <NSpace :size="4" align="center">
          <span class="results-count">
            {{ t('query.showing') }} {{ logEntries.length }}
            <template v-if="logTotal > 0"> / {{ logTotal }}</template>
            {{ t('query.entries') }}
            <NTag v-if="logTruncated" type="warning" size="tiny" :bordered="false" class="tag-ml">{{ t('query.truncated') }}</NTag>
          </span>
          <span v-if="queryStats" class="query-stats">{{ queryStats.executionTimeMs }}ms</span>
          <NButton v-if="canExport" size="tiny" quaternary @click="exportCsv"><template #icon><NIcon :size="14"><DownloadOutline /></NIcon></template></NButton>
        </NSpace>
      </div>

      <!-- Log Content: Sidebar + Main -->
      <div class="log-content-area">
        <!-- Field Sidebar (Nightingale FieldsList pattern) -->
        <LogFieldSidebar
          :fields="logFields"
          :log-entries="logEntries"
          @add-field-filter="onFieldFilterAdd"
        />

        <!-- Log Main Area -->
        <div class="log-main-area">
          <!-- Level Legend -->
          <div class="log-level-legend">
            <span v-for="(color, level) in LEVEL_COLORS" :key="level" class="level-item" v-show="level !== 'unknown'">
              <span class="level-dot" :style="{ background: color }" />
              <span class="level-label">{{ level }}</span>
            </span>
          </div>

          <!-- Origin Mode (Nightingale Raw.tsx pattern) -->
          <div v-if="logMode === 'origin'" class="log-origin-view">
            <div
              v-for="(entry, idx) in logEntries"
              :key="(entry as any)._key ?? idx"
              class="log-origin-row"
              :class="logRowClassName(entry)"
              @click="openLogDrawer(idx)"
            >
              <span class="origin-level-dot" :style="{ background: LEVEL_COLORS[detectLogLevel(entry)] }" />
              <span v-if="logOptions.showTime !== false" class="origin-time">{{ fmtTs(entry.timestamp) }}</span>
              <span class="origin-message" :style="{ whiteSpace: logOptions.lineBreak ? 'pre-wrap' : 'nowrap' }">{{ entry.message || '-' }}</span>
              <div v-if="logOptions.showLabels !== false" class="origin-labels">
                <span
                  v-for="([k, v], i) in Object.entries(entry.labels || {}).slice(0, 6)"
                  :key="i"
                  class="origin-label-pair"
                >
                  <span class="origin-label-key">{{ k }}</span>=<FieldValueToken
                    :field-key="k"
                    :field-value="String(v ?? '')"
                    @filter="(key: string, value: string, op: string) => onTokenFilter(key, value, op)"
                  />
                </span>
              </div>
            </div>
          </div>

          <!-- Table Mode (Nightingale Table.tsx pattern) -->
          <NDataTable
            v-if="logMode === 'table'"
            :columns="logColumnsEnhanced"
            :data="logEntries"
            :row-key="(r: Record<string, unknown>) => String(r._key)"
            :row-class-name="logRowClassName"
            size="small"
            max-height="600"
            virtual-scroll
          />
        </div>
      </div>
    </div>

    <!-- No results -->
    <div v-if="!loading && !errorMsg && selectedDsId && expression.trim() && ((!isLogs && metricData !== null && !metricData?.series?.length) || (isLogs && !logEntries.length && metricData === null && logTotal === 0))" class="query-empty">
      {{ t('query.noResults') }}
    </div>

    <!-- Log Detail Drawer -->
    <LogDetailDrawer
      v-model:show="drawerVisible"
      :log-entry="drawerLogEntry"
      :log-entries="logEntries"
      :current-index="drawerCurrentIndex"
      @prev="onDrawerPrev"
      @next="onDrawerNext"
    />
  </div>
</template>

<style scoped>
/* Nightingale panel: bg-fc-100, border, rounded-lg, p-4 */
.panel-content {
  background: var(--sre-bg-sunken, #f8fafc);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
  position: relative;
}

/* Panel top row: ds select + tabs + close (Nightingale: Row gutter=8) */
.panel-top-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
  flex-shrink: 0;
}
.panel-top-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}
.ds-select { width: 260px; flex-shrink: 0; }
.panel-tabs-inline { flex: 1; min-width: 0; }
.panel-tabs-inline :deep(.n-tabs-tab) { padding: 4px 12px; }

/* Editor row (Nightingale: PromQL input + Execute button side-by-side) */
.editor-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
.editor-input-wrap {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}
.editor-actions {
  flex-shrink: 0;
  display: flex;
  align-items: flex-start;
  gap: 4px;
}

/* Pre-query controls (instant/range, step, limit) */
.pre-query-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.field-label { font-size: 12px; color: var(--sre-text-tertiary); }
.shortcut-hint { font-size: 11px; color: var(--sre-text-tertiary); white-space: nowrap; }
.control-select-sm { width: 100px; }

.query-empty-inline { padding: 16px 4px; color: var(--sre-text-tertiary); font-size: 13px; }
.query-empty { display: flex; align-items: center; justify-content: center; min-height: 200px; color: var(--sre-text-tertiary); font-size: 14px; }

/* Metrics Results (Nightingale: card-style tabs container) */
.metrics-results {
  overflow: visible;
}

/* Card tabs: Nightingale PromGraphCpt style.less pattern */
.metric-card-tabs {
  display: flex;
  flex-direction: column;
}
.metric-card-tabs :deep(.n-tabs-tab) {
  padding: 6px 16px;
  font-size: 13px;
  color: var(--sre-text-tertiary);
}
.metric-card-tabs :deep(.n-tabs-tab--active) {
  border-top: 2px solid var(--sre-primary) !important;
  color: var(--sre-text-primary);
}
.metric-card-tabs :deep(.n-tabs-tab-pad) {
  display: none;
}
.metric-card-tabs :deep(.n-tabs-content) {
  padding: 0;
}
.card-tabs-suffix {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-right: 8px;
}

/* Graph controls row (Nightingale Graph.tsx: Space wrap pattern) */
.graph-controls-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--sre-border);
  margin-bottom: 12px;
}
.graph-time-presets {
  display: flex;
  gap: 4px;
  align-items: center;
  flex-wrap: wrap;
}

.results-count { font-size: 13px; color: var(--sre-text-secondary); }
.query-stats { font-size: 11px; color: var(--sre-text-tertiary); font-family: var(--sre-font-mono, monospace); }
.tag-ml { margin-left: 4px; }

.chart-container { min-height: 300px; display: flex; align-items: center; justify-content: center; }
.chart-fallback { display: flex; flex-direction: column; align-items: center; gap: 12px; color: var(--sre-text-tertiary); font-size: 13px; }
.chart-full { width: 100%; height: 300px; }
.loading-container { display: flex; justify-content: center; padding: 40px; }

/* Log Results (Nightingale logExplorer pattern) */
.log-results {
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  overflow: visible;
  background: var(--sre-bg-card);
  padding: 12px;
}
.log-histogram-container { margin-bottom: 8px; }

/* Log controls row (Nightingale: mode + settings + stats) */
.log-controls-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--sre-border);
  margin-bottom: 8px;
  flex-wrap: wrap;
}

/* Log content area: sidebar + main (Nightingale: flex row) */
.log-content-area {
  display: flex;
  gap: 0;
  min-height: 400px;
}
.log-main-area {
  flex: 1;
  min-width: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
}

/* Level legend */
.log-level-legend { display: flex; gap: 12px; margin-bottom: 8px; padding: 4px 0; flex-shrink: 0; }
.level-item { display: flex; align-items: center; gap: 4px; font-size: 11px; color: var(--sre-text-tertiary); }
.level-dot { width: 8px; height: 8px; border-radius: 2px; flex-shrink: 0; }
.level-label { text-transform: uppercase; font-weight: 500; letter-spacing: 0.5px; }

/* Origin mode (Nightingale Raw.tsx pattern: inline field-value rows) */
.log-origin-view {
  flex: 1;
  overflow-y: auto;
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  line-height: 1.6;
}
.log-origin-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 4px 8px;
  cursor: pointer;
  border-bottom: 1px solid var(--sre-border-light, rgba(0,0,0,0.04));
  transition: background 0.15s;
}
.log-origin-row:hover {
  background: var(--sre-bg-hover);
}
.origin-level-dot {
  width: 4px;
  min-height: 16px;
  border-radius: 2px;
  flex-shrink: 0;
  margin-top: 3px;
}
.origin-time {
  flex-shrink: 0;
  color: var(--sre-text-tertiary);
  font-size: 11px;
  white-space: nowrap;
}
.origin-message {
  flex: 1;
  min-width: 0;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
}
.origin-labels {
  flex-shrink: 0;
  display: flex;
  gap: 2px;
  flex-wrap: wrap;
  max-width: 300px;
}
.origin-label-tag {
  max-width: 140px;
  cursor: pointer;
  font-size: 11px;
}
.origin-label-pair {
  font-size: 11px;
  font-family: var(--sre-font-mono, monospace);
  color: var(--sre-text-secondary);
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.origin-label-key {
  color: var(--sre-primary);
}

/* History popover */
.history-pop { min-width: 360px; max-width: 480px; }
.history-title { font-size: 12px; font-weight: 600; color: var(--sre-text-secondary); margin-bottom: 8px; }
.history-empty { font-size: 12px; color: var(--sre-text-tertiary); padding: 12px 0; text-align: center; }
.history-item { padding: 8px; border-radius: 6px; cursor: pointer; border: 1px solid transparent; }
.history-item:hover { background: var(--sre-bg-hover); border-color: var(--sre-border); }
.history-expr { font-family: var(--sre-font-mono, monospace); font-size: 12px; color: var(--sre-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.history-ts { font-size: 11px; color: var(--sre-text-tertiary); margin-top: 2px; }

:deep(.log-row-error) { background: rgba(239, 68, 68, 0.04) !important; }
:deep(.log-row-warn) { background: rgba(234, 179, 8, 0.04) !important; }

/* Error card */
.error-card { display: flex; align-items: flex-start; gap: 12px; padding: 16px; margin: 12px 0; background: var(--sre-critical-soft); border: 1px solid var(--sre-critical-soft); border-radius: var(--sre-radius-md); }
.error-icon-wrap { flex-shrink: 0; width: 32px; height: 32px; border-radius: 50%; background: var(--sre-critical); display: flex; align-items: center; justify-content: center; font-size: 18px; color: var(--sre-text-inverse); }
.error-body { flex: 1; min-width: 0; }
.error-title { font-size: var(--sre-fs-md); font-weight: var(--sre-fw-semibold); color: var(--sre-text-primary); margin: 0 0 4px; }
.error-message { font-size: var(--sre-fs-sm); color: var(--sre-text-secondary); line-height: var(--sre-lh-snug); word-break: break-all; }
.error-actions { display: flex; gap: 8px; flex-shrink: 0; margin-top: 2px; }
</style>
