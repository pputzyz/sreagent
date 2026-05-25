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

// --- Resolve step ---
function resolveStep(): string {
  if (localStepValue.value !== 'auto') return localStepValue.value
  const diff = props.timeEnd - props.timeStart
  return diff <= 3600 ? '15s' : diff <= 21600 ? '1m' : diff <= 86400 ? '5m' : '15m'
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
        time: props.timeEnd,
      })
      const data = res.data?.data
      if (data?.series && data.series.length > metricLimit.value) data.series = data.series.slice(0, metricLimit.value)
      metricData.value = data
    } else {
      const res = await datasourceApi.rangeQuery(selectedDsId.value, {
        expression: expression.value,
        start: props.timeStart,
        end: props.timeEnd,
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
    if (isArea) seriesItem.areaStyle = { opacity: 0.3 }
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
    <!-- Panel header with close button -->
    <div v-if="canClose" class="panel-header">
      <span class="panel-label">{{ t('query.panel') }} {{ panelId }}</span>
      <NButton size="tiny" quaternary @click="emit('remove', panelId)">
        <template #icon><NIcon><CloseCircleOutline /></NIcon></template>
      </NButton>
    </div>
    <!-- Tab selector -->
    <NTabs v-model:value="activeTab" type="line" size="small" class="panel-tabs">
      <NTabPane name="metrics" :tab="t('query.metricsTab')" />
      <NTabPane name="logs" :tab="t('query.logsTab')" />
    </NTabs>

    <!-- Datasource selector -->
    <div class="ds-selector">
      <NSelect
        v-model:value="selectedDsId"
        :options="(isLogs ? logDatasources : metricDatasources).map(d => ({ label: dsLabel(d), value: d.id }))"
        :placeholder="isLogs ? t('query.selectLogDatasource') : t('query.selectDatasource')"
        filterable clearable size="small" class="ds-select"
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

    <!-- Editor -->
    <div v-if="selectedDsId != null" class="query-bar">
      <div class="query-editor-wrap">
        <PromQLEditor v-if="!isLogs" v-model="expression" :datasource-id="selectedDsId" :placeholder="t('query.promqlPlaceholder')" @execute="run" />
        <LogsQLEditor v-else v-model="expression" :datasource-id="selectedDsId" :placeholder="t('query.logQueryPlaceholder')" @execute="run" />
        <div class="editor-tools">
          <NPopover v-model:show="historyVisible" trigger="click" placement="bottom-end">
            <template #trigger>
              <NTooltip><template #trigger><NButton size="tiny" quaternary><template #icon><NIcon><TimeOutline /></NIcon></template></NButton></template>{{ t('query.queryHistory') }}</NTooltip>
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
          <NTooltip><template #trigger><NButton size="tiny" quaternary :disabled="!expression" @click="expression = ''"><template #icon><NIcon><TrashOutline /></NIcon></template></NButton></template>{{ t('query.clearBtn') }}</NTooltip>
        </div>
      </div>
    </div>

    <!-- Controls -->
    <div v-if="selectedDsId != null" class="query-actions-row">
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

    <!-- Metrics Results -->
    <div v-if="!loading && !isLogs && metricData?.series?.length" class="results-panel">
      <div class="results-header">
        <div class="results-header-left">
          <span class="results-count">
            {{ metricData.series.length }} {{ t('query.seriesCount') }}
            <template v-if="metricData.result_type"> · {{ metricData.result_type }}</template>
            <NTag v-if="isMetricLimited" type="warning" size="small" :bordered="false" class="tag-ml">{{ t('query.limitedTo', { n: metricLimit }) }}</NTag>
          </span>
          <span v-if="queryStats" class="query-stats">{{ queryStats.executionTimeMs }}ms<template v-if="queryStats.step"> · step {{ queryStats.step }}</template></span>
        </div>
        <NSpace :size="4" align="center">
          <MetricChartControls v-model="chartSettings" />
          <NButton size="small" :type="resultMode === 'chart' ? 'primary' : 'default'" :secondary="resultMode !== 'chart'" @click="resultMode = 'chart'">{{ t('query.chart') }}</NButton>
          <NButton size="small" :type="resultMode === 'table' ? 'primary' : 'default'" :secondary="resultMode !== 'table'" @click="resultMode = 'table'">{{ t('query.table') }}</NButton>
          <NButton v-if="canExport" size="small" tertiary @click="exportCsv"><template #icon><NIcon><DownloadOutline /></NIcon></template>{{ t('query.exportCsv') }}</NButton>
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
        :row-props="(row: MetricTableRow) => ({ style: 'cursor: pointer', onClick: () => emit('openLabels', row._rawLabels, row.name) })"
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
            <NTag v-if="logTruncated" type="warning" size="small" :bordered="false" class="tag-ml">{{ t('query.truncated') }}</NTag>
          </span>
          <span v-if="queryStats" class="query-stats">{{ queryStats.executionTimeMs }}ms</span>
        </div>
        <NSpace :size="4" align="center">
          <NButton size="tiny" quaternary @click="showHistogram = !showHistogram">{{ showHistogram ? t('query.hideHistogram') : t('query.showHistogram') }}</NButton>
          <NButton v-if="canExport" size="small" tertiary @click="exportCsv"><template #icon><NIcon><DownloadOutline /></NIcon></template>{{ t('query.exportCsv') }}</NButton>
        </NSpace>
      </div>
      <LogHistogram v-if="showHistogram" :buckets="histogramBuckets" :loading="histogramLoading" class="log-histogram-container" @bar-click="onHistogramBarClick" @brush-select="onHistogramBrushSelect" />
      <div class="log-level-legend">
        <span v-for="(color, level) in LEVEL_COLORS" :key="level" class="level-item" v-show="level !== 'unknown'">
          <span class="level-dot" :style="{ background: color }" />
          <span class="level-label">{{ level }}</span>
        </span>
      </div>
      <NDataTable :columns="logColumnsEnhanced" :data="logEntries" :row-key="(r: Record<string, unknown>) => String(r._key)" :row-class-name="logRowClassName" size="small" max-height="600" virtual-scroll />
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
.panel-content {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.panel-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.panel-tabs { margin-bottom: 12px; }
.ds-selector { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.ds-info { display: flex; align-items: center; gap: 8px; }
.ds-endpoint { font-size: 12px; color: var(--sre-text-tertiary); }
.ds-select { max-width: 420px; flex: 1; }
.query-bar { margin-bottom: 12px; }
.query-editor-wrap { position: relative; }
.editor-tools { position: absolute; top: 6px; right: 6px; display: flex; gap: 2px; }
.query-actions-row { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.field-label { font-size: 12px; color: var(--sre-text-tertiary); }
.shortcut-hint { font-size: 11px; color: var(--sre-text-tertiary); white-space: nowrap; }
.control-select-sm { width: 100px; }
.query-empty-inline { padding: 16px 4px; color: var(--sre-text-tertiary); font-size: 13px; }
.query-empty { display: flex; align-items: center; justify-content: center; min-height: 200px; color: var(--sre-text-tertiary); font-size: 14px; }
.results-panel { background: var(--sre-bg-sunken, #f8fafc); border-radius: 8px; padding: 16px; border: 1px solid var(--sre-border); overflow: hidden; }
.results-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.results-header-left { display: flex; align-items: center; gap: 4px; }
.results-count { font-size: 13px; color: var(--sre-text-secondary); }
.query-stats { font-size: 11px; color: var(--sre-text-tertiary); font-family: var(--sre-font-mono, monospace); margin-left: 8px; }
.tag-ml { margin-left: 8px; }
.chart-container { min-height: 300px; display: flex; align-items: center; justify-content: center; overflow: hidden; }
.chart-fallback { display: flex; flex-direction: column; align-items: center; gap: 12px; color: var(--sre-text-tertiary); font-size: 13px; }
.chart-full { width: 100%; height: 300px; }
.loading-container { display: flex; justify-content: center; padding: 40px; }
.log-histogram-container { margin-bottom: 12px; }
.log-level-legend { display: flex; gap: 12px; margin-bottom: 8px; padding: 4px 0; }
.level-item { display: flex; align-items: center; gap: 4px; font-size: 11px; color: var(--sre-text-tertiary); }
.level-dot { width: 8px; height: 8px; border-radius: 2px; flex-shrink: 0; }
.level-label { text-transform: uppercase; font-weight: 500; letter-spacing: 0.5px; }
.log-expanded-row { padding: 12px 16px; background: var(--sre-bg-sunken, #f8fafc); border-radius: 6px; }
.log-expanded-title { font-weight: 600; font-size: 13px; color: var(--sre-text-primary); margin-bottom: 8px; }
.log-expanded-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 6px; }
.log-field-item { display: flex; gap: 6px; padding: 4px 8px; border-radius: 4px; background: var(--sre-bg-card, #fff); border: 1px solid var(--sre-border); cursor: pointer; font-size: 12px; transition: border-color 0.15s; }
.log-field-item:hover { border-color: var(--sre-primary); }
.log-field-key { color: var(--sre-primary); font-weight: 500; min-width: 80px; font-family: var(--sre-font-mono, monospace); }
.log-field-value { color: var(--sre-text-secondary); word-break: break-all; font-family: var(--sre-font-mono, monospace); }
.log-expanded-level { margin-top: 8px; font-size: 12px; color: var(--sre-text-tertiary); }
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
