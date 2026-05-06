<script setup lang="ts">
/**
 * Data Query Page — unified metrics + logs query interface.
 *
 * Supports:
 *  - Prometheus / VictoriaMetrics: PromQL / MetricsQL range queries → chart + table
 *  - VictoriaLogs: LogsQL queries → log entry table
 *  - Zabbix: item key queries → table
 *
 * All heavy dependencies (ECharts, CodeMirror) are lazily imported via
 * defineAsyncComponent / dynamic import so that a missing or broken dep
 * never white-screens the page — the fallback is a plain textarea + table.
 */
import { ref, onMounted, computed, watch, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NSelect, NButton, NSpace, NTag, NAlert, NSpin,
  NDataTable, NTabs, NTabPane, NInputNumber, NInput,
  useMessage,
} from 'naive-ui'
import { datasourceApi } from '@/api'
import type { DataSource, DataSourceType } from '@/types'

const { t } = useI18n()
const message = useMessage()

// --- Lazy ECharts (never blocks page render) ---
const ChartReady = ref(false)
const VChart = shallowRef<any>(null)
const echartsSetup = shallowRef<any>(null)

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
      CanvasRenderer,
      LineChart,
      components.TooltipComponent,
      components.LegendComponent,
      components.GridComponent,
      components.DataZoomComponent,
    ])
    VChart.value = vc.default
    ChartReady.value = true
  } catch (e) {
    console.warn('[DataQuery] ECharts load failed, chart mode unavailable:', e)
  }
}

// PromQLEditor removed — CodeMirror dependency chain was the root cause of
// repeated white-screen issues (v1.16.10-v1.16.18). Using stable NInput textarea
// for query input. If CodeMirror support is needed later, re-introduce it as
// a lazy async component with proper error boundary.

type ResultMode = 'chart' | 'table'
type QueryTab = 'metrics' | 'logs'

// --- state ---
const datasources = ref<DataSource[]>([])
const selectedDsId = ref<number | null>(null)
const expression = ref('')
const loading = ref(false)
const errorMsg = ref('')
const logEntries = ref<any[]>([])
const metricData = ref<any>(null)
const logTotal = ref(0)
const logTruncated = ref(false)
const logLimit = ref(200)
const resultMode = ref<ResultMode>('chart')
const activeTab = ref<QueryTab>('metrics')

// --- time ---
const now = ref(Date.now())
const rangeH = ref(1)
const timeStart = computed(() => Math.floor((now.value - rangeH.value * 3600000) / 1000))
const timeEnd = computed(() => Math.floor(now.value / 1000))

const timeOptions = [
  { label: t('query.last1h'), value: 1 },
  { label: t('query.last6h'), value: 6 },
  { label: t('query.last24h'), value: 24 },
  { label: t('query.last3d'), value: 72 },
  { label: t('query.last7d'), value: 168 },
]

// --- computed ---
const selectedDs = computed(() => datasources.value.find(d => d.id === selectedDsId.value))

const metricDatasources = computed(() =>
  datasources.value.filter(d => ['prometheus', 'victoriametrics', 'zabbix'].includes(d.type))
)
const logDatasources = computed(() =>
  datasources.value.filter(d => d.type === 'victorialogs')
)

const currentDsList = computed(() =>
  activeTab.value === 'logs' ? logDatasources.value : metricDatasources.value
)

const isLogs = computed(() => activeTab.value === 'logs')
const isMetrics = computed(() => activeTab.value === 'metrics')

function dsLabel(ds: DataSource): string {
  return `${ds.name} (${typeBadge(ds.type)})`
}

function typeBadge(t: DataSourceType): string {
  const m: Record<string, string> = {
    prometheus: 'Prometheus',
    victoriametrics: 'VictoriaMetrics',
    victorialogs: 'VictoriaLogs',
    zabbix: 'Zabbix',
  }
  return m[t] || t
}

function typeColor(t: DataSourceType): string {
  const m: Record<string, string> = {
    prometheus: '#e6522c',
    victoriametrics: '#1a7f37',
    victorialogs: '#0550ae',
    zabbix: '#d32f2f',
  }
  return m[t] || '#666'
}

// --- actions ---
async function loadDs() {
  try {
    const res = await datasourceApi.list({ page: 1, page_size: 100 })
    const list = res.data?.data?.list
    datasources.value = (Array.isArray(list) ? list : []).filter((d: any) => d.is_enabled)
  } catch { /* ignore */ }
}

function refreshTime() {
  now.value = Date.now()
}

async function run() {
  if (!selectedDsId.value || !expression.value.trim()) return
  refreshTime()
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
        logEntries.value = (data.entries || []).map((e: any, i: number) => ({ ...e, _key: i }))
        logTotal.value = data.total || 0
        logTruncated.value = data.truncated || false
      }
    } else {
      const diff = timeEnd.value - timeStart.value
      const step = diff <= 3600 ? '15s' : diff <= 21600 ? '1m' : diff <= 86400 ? '5m' : '15m'
      const res = await datasourceApi.rangeQuery(selectedDsId.value, {
        expression: expression.value,
        start: timeStart.value,
        end: timeEnd.value,
        step,
      })
      metricData.value = res.data?.data
    }
  } catch (e: any) {
    errorMsg.value = e?.response?.data?.error || e?.response?.data?.message || e?.message || t('query.queryFailed')
  } finally {
    loading.value = false
  }
}

// --- chart option ---
const chartOption = computed(() => {
  if (!metricData.value?.series?.length) return null
  const seriesList: any[] = []
  const allTimestamps = new Set<number>()
  for (const s of metricData.value.series) {
    const name = formatLegend(s.labels)
    const data: [number, number][] = []
    for (const v of s.values || []) {
      const ts = Number(v.ts) * 1000
      const val = v.value != null ? Number(v.value) : 0
      data.push([ts, val])
      allTimestamps.add(ts)
    }
    seriesList.push({ name, type: 'line', data, smooth: false, showSymbol: false, connectNulls: true })
  }
  return {
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis', confine: true },
    legend: { type: 'scroll', bottom: 0, textStyle: { color: '#888', fontSize: 12 } },
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

// --- metric table ---
const metricColumns = computed(() => [
  { title: t('query.metricName'), key: 'name', ellipsis: { tooltip: true }, width: 200 },
  { title: t('query.value'), key: 'value', width: 160 },
  { title: t('query.labelsHeader'), key: 'labels', ellipsis: { tooltip: true } },
])

const metricTableData = computed(() => {
  if (!metricData.value?.series) return []
  const rows: any[] = []
  let idx = 0
  for (const s of metricData.value.series) {
    for (const v of (s.values || [])) {
      rows.push({
        _key: idx++,
        name: s.labels?.__name__ || '-',
        value: typeof v.value === 'number' ? v.value.toFixed(4) : String(v.value ?? '-'),
        labels: formatLabelsStr(s.labels),
      })
    }
  }
  return rows
})

// --- log table ---
const logColumns = computed(() => [
  { title: t('query.logTime'), key: 'timestamp', width: 200, render: (r: any) => fmtTs(r.timestamp) },
  { title: t('query.logMessage'), key: 'message', ellipsis: { tooltip: true } },
  { title: t('query.logLabels'), key: '_labels', width: 300, ellipsis: { tooltip: true }, render: (r: any) => formatLabelsStr(r.labels) },
])

// --- helpers ---
function formatLegend(lbs: Record<string, string>): string {
  if (!lbs) return 'value'
  const parts: string[] = []
  for (const k of Object.keys(lbs)) {
    if (k !== '__name__') parts.push(`${k}="${lbs[k]}"`)
  }
  return parts.length ? parts.join(', ') : (lbs.__name__ || 'value')
}

function formatLabelsStr(lbs: any): string {
  if (!lbs) return '-'
  const parts: string[] = []
  for (const k of Object.keys(lbs)) {
    if (k !== '__name__') parts.push(`${k}=${lbs[k]}`)
  }
  return parts.length ? parts.join(', ') : '-'
}

function fmtTs(ts: any): string {
  if (!ts) return '-'
  try { return new Date(ts).toLocaleString() } catch { return String(ts) }
}

// --- watch ---
watch(selectedDsId, () => {
  expression.value = ''
  metricData.value = null
  logEntries.value = []
  errorMsg.value = ''
})

watch(activeTab, () => {
  selectedDsId.value = null
  expression.value = ''
  metricData.value = null
  logEntries.value = []
  errorMsg.value = ''
})

onMounted(() => {
  loadDs()
  loadECharts()
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
      <NSpace align="center" :size="12">
        <NSelect
          v-model:value="rangeH"
          :options="timeOptions"
          size="small"
          style="width:160px"
        />
      </NSpace>
    </div>

    <!-- Tabs: Metrics / Logs -->
    <NTabs v-model:value="activeTab" type="line" style="margin-bottom:16px">
      <NTabPane name="metrics" :tab="t('query.metricsTab')">
        <!-- Data Source Selector -->
        <div class="ds-selector">
          <NSelect
            v-model:value="selectedDsId"
            :options="metricDatasources.map(d => ({ label: dsLabel(d), value: d.id }))"
            :placeholder="t('query.selectDatasource')"
            filterable
            clearable
            style="max-width:480px;flex:1"
          />
          <div v-if="selectedDs" class="ds-info">
            <NTag :color="{ color: typeColor(selectedDs.type), textColor: '#fff' }" size="small" :bordered="false">
              {{ typeBadge(selectedDs.type) }}
            </NTag>
            <span class="ds-endpoint">{{ selectedDs.endpoint }}</span>
          </div>
        </div>

        <!-- Query Input -->
        <div v-if="selectedDsId != null" class="query-bar">
          <div class="query-editor-wrap">
            <NInput
              v-model:value="expression"
              type="textarea"
              :rows="3"
              :placeholder="t('query.promqlPlaceholder')"
              style="font-family: 'JetBrains Mono', 'Fira Code', monospace; font-size: 13px"
              @keyup.ctrl.enter="run"
              @keyup.meta.enter="run"
            />
          </div>
          <div class="query-actions">
            <NButton type="primary" :loading="loading" :disabled="!expression.trim()" @click="run">
              {{ t('query.runQuery') }}
            </NButton>
            <span class="shortcut-hint">Ctrl + Enter</span>
          </div>
        </div>

        <!-- Empty state -->
        <div v-if="selectedDsId == null" class="query-empty">
          {{ t('query.selectDatasource') }}
        </div>

        <!-- Error -->
        <NAlert v-if="errorMsg" type="error" :title="errorMsg" closable style="margin-bottom:12px" @close="errorMsg = ''" />

        <!-- Loading -->
        <div v-if="loading" style="display:flex;justify-content:center;padding:40px">
          <NSpin size="medium" />
        </div>

        <!-- Metrics Results -->
        <div v-if="!loading && metricData?.series?.length" class="results-panel">
          <div class="results-header">
            <span class="results-count">
              {{ metricData.series.length }} {{ t('query.seriesCount') }}
              <template v-if="metricData.resultType"> · {{ metricData.resultType }}</template>
            </span>
            <NSpace :size="4">
              <NButton size="small" :type="resultMode === 'chart' ? 'primary' : 'default'" @click="resultMode = 'chart'">
                {{ t('query.chart') }}
              </NButton>
              <NButton size="small" :type="resultMode === 'table' ? 'primary' : 'default'" @click="resultMode = 'table'">
                {{ t('query.table') }}
              </NButton>
            </NSpace>
          </div>

          <!-- Chart (lazy, with fallback) -->
          <div v-if="resultMode === 'chart'" class="chart-container">
            <template v-if="ChartReady && VChart && chartOption">
              <component :is="VChart" :option="chartOption" :autoresize="true" style="width:100%;height:400px" />
            </template>
            <div v-else class="chart-fallback">
              <p>{{ t('query.chartUnavailable') }}</p>
              <NButton size="small" @click="resultMode = 'table'">{{ t('query.switchToTable') }}</NButton>
            </div>
          </div>

          <!-- Table -->
          <NDataTable
            v-if="resultMode === 'table'"
            :columns="metricColumns"
            :data="metricTableData"
            :row-key="(r: any) => r._key"
            size="small"
            :single-line="false"
            striped
            max-height="500"
            virtual-scroll
          />
        </div>

        <!-- No results -->
        <div v-if="!loading && !errorMsg && selectedDsId && expression.trim() && !metricData?.series?.length && metricData !== null" class="query-empty">
          {{ t('query.noResults') }}
        </div>
      </NTabPane>

      <NTabPane name="logs" :tab="t('query.logsTab')">
        <!-- Data Source Selector for Logs -->
        <div class="ds-selector">
          <NSelect
            v-model:value="selectedDsId"
            :options="logDatasources.map(d => ({ label: dsLabel(d), value: d.id }))"
            :placeholder="t('query.selectLogDatasource')"
            filterable
            clearable
            style="max-width:480px;flex:1"
          />
          <div v-if="selectedDs" class="ds-info">
            <NTag :color="{ color: typeColor(selectedDs.type), textColor: '#fff' }" size="small" :bordered="false">
              {{ typeBadge(selectedDs.type) }}
            </NTag>
            <span class="ds-endpoint">{{ selectedDs.endpoint }}</span>
          </div>
        </div>

        <div v-if="!logDatasources.length" class="query-empty">
          {{ t('query.noLogDatasources') }}
        </div>

        <!-- Log Query Input -->
        <div v-if="selectedDsId != null" class="query-bar">
          <div class="query-editor-wrap">
            <NInput
              v-model:value="expression"
              type="textarea"
              :rows="3"
              :placeholder="t('query.logQueryPlaceholder')"
              style="font-family: 'JetBrains Mono', 'Fira Code', monospace"
              @keyup.ctrl.enter="run"
              @keyup.meta.enter="run"
            />
          </div>
          <div class="query-actions">
            <NInputNumber
              v-model:value="logLimit"
              :min="10"
              :max="10000"
              size="small"
              style="width:120px"
            >
              <template #prefix>Limit</template>
            </NInputNumber>
            <NButton type="primary" :loading="loading" :disabled="!expression.trim()" @click="run">
              {{ t('query.runQuery') }}
            </NButton>
            <span class="shortcut-hint">Ctrl + Enter</span>
          </div>
        </div>

        <!-- Error -->
        <NAlert v-if="errorMsg" type="error" :title="errorMsg" closable style="margin-bottom:12px" @close="errorMsg = ''" />

        <!-- Loading -->
        <div v-if="loading" style="display:flex;justify-content:center;padding:40px">
          <NSpin size="medium" />
        </div>

        <!-- Log Results -->
        <div v-if="!loading && logEntries.length" class="results-panel">
          <div class="results-header">
            <span class="results-count">
              {{ t('query.showing') }} {{ logEntries.length }}
              <template v-if="logTotal > 0"> / {{ logTotal }}</template>
              {{ t('query.entries') }}
            </span>
            <NTag v-if="logTruncated" type="warning" size="small" :bordered="false">
              {{ t('query.truncated') }}
            </NTag>
          </div>
          <NDataTable
            :columns="logColumns"
            :data="logEntries"
            :row-key="(r: any) => r._key"
            size="small"
            max-height="600"
            virtual-scroll
          />
        </div>

        <!-- No results -->
        <div v-if="!loading && !errorMsg && selectedDsId && expression.trim() && !logEntries.length && logTotal === 0 && metricData === null" class="query-empty">
          {{ t('query.noResults') }}
        </div>
      </NTabPane>
    </NTabs>
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
  margin-bottom: 20px;
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

.ds-selector {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
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
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
}

.query-editor-wrap {
  flex: 1;
  min-width: 0;
}

.query-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  flex-wrap: wrap;
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

.results-panel {
  background: var(--sre-bg-card, #fff);
  border-radius: 12px;
  padding: 16px;
  border: 1px solid var(--sre-border, #e0e0e0);
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
}

.chart-fallback {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--sre-text-tertiary);
  font-size: 13px;
}
</style>
