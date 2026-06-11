<script setup lang="ts">
/**
 * Metric Views — Dedicated metric exploration page.
 * Inspired by Nightingale's Metric Explorer architecture:
 *  - Left sidebar: saved views
 *  - Center: datasource selector + label filters + metric list
 *  - Bottom: chart area
 *
 * All Prometheus API calls go through the existing datasource proxy:
 *   /api/v1/datasources/:id/proxy/api/v1/...
 */
import { ref, computed, watch, onMounted, onBeforeUnmount, shallowRef, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  NSelect, NButton, NIcon, NSpin, NEmpty, NDrawer, NDrawerContent,
  NDescriptions, NDescriptionsItem, NTag, NDivider,
} from 'naive-ui'
import {
  RefreshOutline, TimeOutline, BookmarkOutline,
  TrashOutline, AddOutline, SearchOutline, ArrowBackOutline,
  CopyOutline,
} from '@vicons/ionicons5'
import { datasourceApi } from '@/api'
import MetricLabelSelector from '@/components/query/MetricLabelSelector.vue'
import MetricList from '@/components/query/MetricList.vue'
import type { DataSource, QueryResponse } from '@/types'

const { t } = useI18n()
const router = useRouter()
const message = useMessage()

// ===== Datasources =====
const datasources = ref<DataSource[]>([])
const selectedDsId = ref<number | null>(null)

const metricDatasources = computed(() =>
  datasources.value.filter(d => d.is_enabled && d.supports_query && d.type !== 'victorialogs')
)
const dsOptions = computed(() =>
  metricDatasources.value.map(d => ({
    label: `${d.name} (${typeBadge(d.type)})`,
    value: d.id,
  }))
)

function typeBadge(tp: string): string {
  const m: Record<string, string> = {
    prometheus: 'Prometheus',
    victoriametrics: 'VM',
    zabbix: 'Zabbix',
  }
  return m[tp] || tp
}

async function loadDs() {
  try {
    const res = await datasourceApi.list({ page: 1, page_size: 100 })
    const list = res.data?.data?.list
    datasources.value = (Array.isArray(list) ? list : []).filter((d: DataSource) => d.is_enabled)
  } catch (e) {
    console.warn('[MetricViews] Failed to load datasources:', e)
  }
}

// ===== Selected metric + label selector =====
const selectedMetric = ref('')
const labelSelector = ref('')  // Full PromQL selector from MetricLabelSelector
const metricListRef = ref<InstanceType<typeof MetricList> | null>(null)
const labelSelectorRef = ref<InstanceType<typeof MetricLabelSelector> | null>(null)

// Build the full expression for querying
const queryExpression = computed(() => {
  return labelSelector.value || selectedMetric.value
})

function onMetricSelect(name: string) {
  selectedMetric.value = name
  // Refresh label values when metric changes
  labelSelectorRef.value?.refreshAll()
}

function onSelectorUpdate(selector: string) {
  labelSelector.value = selector
}

// ===== Time range =====
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
  // Auto-fetch if metric is selected
  if (selectedMetric.value) fetchChartData()
}

function openCustomRange() {
  rangeMin.value = -1
  showCustomPicker.value = true
  if (!customRange.value) {
    const n = Date.now()
    customRange.value = [n - 3600000, n]
  }
}

// ===== Step calculation (Nightingale pattern) =====
function getStep(start: number, end: number, maxDataPoints = 240, minStep = 15): number {
  return Math.max(Math.floor((end - start) / maxDataPoints), minStep)
}

// ===== Chart data =====
const chartData = ref<QueryResponse | null>(null)
const chartLoading = ref(false)
const chartError = ref('')

// ECharts lazy loading
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
    console.warn('[MetricViews] ECharts load failed:', e)
  }
}

// Fetch chart data via range query
async function fetchChartData() {
  if (!selectedDsId.value || !queryExpression.value.trim()) return
  chartLoading.value = true
  chartError.value = ''
  chartData.value = null
  try {
    const step = getStep(timeStart.value, timeEnd.value)
    const res = await datasourceApi.rangeQuery(selectedDsId.value, {
      expression: queryExpression.value,
      start: timeStart.value,
      end: timeEnd.value,
      step: `${step}s`,
    })
    chartData.value = res.data?.data || null
    // Push to history
    pushHistory(queryExpression.value)
  } catch (e: unknown) {
    const err = e as { message?: string }
    chartError.value = err?.message || t('query.queryFailed')
  } finally {
    chartLoading.value = false
  }
}

// Auto-fetch when expression changes (debounced)
let fetchTimer: ReturnType<typeof setTimeout> | null = null
watch(queryExpression, (expr) => {
  if (fetchTimer) clearTimeout(fetchTimer)
  if (!expr.trim() || !selectedDsId.value) {
    chartData.value = null
    return
  }
  fetchTimer = setTimeout(fetchChartData, 500)
})

onBeforeUnmount(() => {
  if (fetchTimer) clearTimeout(fetchTimer)
})

// ===== Chart option =====
const chartOption = computed(() => {
  if (!chartData.value?.series?.length) return null
  interface EChartsSeries {
    name: string; type: string; data: [number, number][];
    smooth: boolean; showSymbol: boolean; connectNulls: boolean;
    lineStyle: { width: number }; areaStyle?: { opacity: number }
  }
  const seriesList: EChartsSeries[] = []
  for (const s of chartData.value.series) {
    const name = formatLegend(s.labels)
    const data: [number, number][] = []
    for (const v of s.values || []) {
      data.push([Number(v.ts) * 1000, v.value != null ? Number(v.value) : 0])
    }
    seriesList.push({
      name, type: 'line', data,
      smooth: true, showSymbol: false, connectNulls: true,
      lineStyle: { width: 1.5 },
    })
  }
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      confine: true,
      axisPointer: { type: 'cross', lineStyle: { type: 'dashed', opacity: 0.4 } },
    },
    legend: { show: false },
    grid: { left: 80, right: 20, top: 20, bottom: 40 },
    xAxis: { type: 'time', axisLabel: { fontSize: 11 } },
    yAxis: { type: 'value', axisLabel: { fontSize: 11 }, splitLine: { lineStyle: { type: 'dashed' } } },
    series: seriesList,
    dataZoom: [{ type: 'inside', start: 0, end: 100 }],
  }
})

function formatLegend(lbs: Record<string, string>): string {
  if (!lbs) return 'value'
  const parts: string[] = []
  for (const [k, v] of Object.entries(lbs)) {
    if (k !== '__name__') parts.push(`${k}="${v}"`)
  }
  return parts.length ? parts.join(', ') : (lbs.__name__ || 'value')
}

// ===== Label detail drawer =====
const labelDrawerVisible = ref(false)
const labelDrawerData = ref<Record<string, string>>({})
const labelDrawerSeriesName = ref('')

function openLabelDrawer(labels: Record<string, string>, seriesName: string) {
  labelDrawerData.value = labels
  labelDrawerSeriesName.value = seriesName
  labelDrawerVisible.value = true
}

// ===== Query History (localStorage) =====
interface HistoryItem { query: string; timestamp: number }
const HISTORY_KEY_PREFIX = 'sre-metric-history'
const historyVisible = ref(false)

function historyKey(): string {
  return selectedDsId.value ? `${HISTORY_KEY_PREFIX}-${selectedDsId.value}` : HISTORY_KEY_PREFIX
}

const history = ref<HistoryItem[]>([])

function loadHistory() {
  try {
    const raw = localStorage.getItem(historyKey())
    if (raw) history.value = JSON.parse(raw) || []
    else history.value = []
  } catch { history.value = [] }
}

function pushHistory(expr: string) {
  if (!expr.trim()) return
  const key = historyKey()
  const list = history.value.filter(h => h.query !== expr)
  list.unshift({ query: expr, timestamp: Date.now() })
  history.value = list.slice(0, 100)
  try { localStorage.setItem(key, JSON.stringify(history.value)) } catch { /* ignore */ }
}

function loadHistoryItem(item: HistoryItem) {
  // Navigate to explore with the expression
  router.push({ path: '/alert/explore', query: { ds: String(selectedDsId.value), expr: item.query } })
  historyVisible.value = false
}

function clearHistory() {
  history.value = []
  try { localStorage.removeItem(historyKey()) } catch { /* ignore */ }
}

// ===== Saved Views (localStorage) =====
interface SavedView {
  id: string
  name: string
  dsId: number
  metric: string
  selector: string
  createdAt: number
}

const VIEWS_KEY = 'sre-metric-views'
const savedViews = ref<SavedView[]>([])
const viewName = ref('')

function loadSavedViews() {
  try {
    const raw = localStorage.getItem(VIEWS_KEY)
    if (raw) savedViews.value = JSON.parse(raw) || []
  } catch { savedViews.value = [] }
}

function saveSavedViews() {
  try { localStorage.setItem(VIEWS_KEY, JSON.stringify(savedViews.value)) } catch { /* ignore */ }
}

function saveCurrentView() {
  if (!selectedDsId.value || !selectedMetric.value) return
  const ds = metricDatasources.value.find(d => d.id === selectedDsId.value)
  const name = viewName.value.trim() || `${ds?.name || ''}: ${selectedMetric.value}`
  const view: SavedView = {
    id: Date.now().toString(36) + Math.random().toString(36).slice(2, 6),
    name,
    dsId: selectedDsId.value,
    metric: selectedMetric.value,
    selector: queryExpression.value,
    createdAt: Date.now(),
  }
  savedViews.value.unshift(view)
  saveSavedViews()
  viewName.value = ''
}

function loadSavedView(view: SavedView) {
  selectedDsId.value = view.dsId
  selectedMetric.value = view.metric
  labelSelector.value = view.selector
}

function deleteSavedView(id: string) {
  savedViews.value = savedViews.value.filter(v => v.id !== id)
  saveSavedViews()
}

// ===== Navigate to Explore =====
function goToExplore() {
  if (selectedDsId.value && queryExpression.value) {
    router.push({
      path: '/alert/explore',
      query: { ds: String(selectedDsId.value), expr: queryExpression.value },
    })
  }
}

// ===== Share URL (FE5-4) =====
function copyShareUrl() {
  if (!selectedDsId.value || !queryExpression.value) return
  const url = new URL(window.location.origin + '/alert/explore')
  url.searchParams.set('ds', String(selectedDsId.value))
  url.searchParams.set('expr', queryExpression.value)
  navigator.clipboard.writeText(url.toString()).then(() => {
    message.success(t('common.copied'))
  }).catch(() => {
    message.error(t('common.failed'))
  })
}

// ===== Init =====
watch(selectedDsId, () => {
  selectedMetric.value = ''
  labelSelector.value = ''
  chartData.value = null
  loadHistory()
})

onMounted(async () => {
  await loadDs()
  loadSavedViews()
  loadHistory()
  loadECharts()
})
</script>

<template>
  <div class="metric-views-page">
    <!-- Header -->
    <div class="mv-header">
      <div class="mv-header-left">
        <h2 class="mv-title">{{ t('menu.metricViews') }}</h2>
        <!-- Time range -->
        <div class="mv-time-bar">
          <NButton
            v-for="opt in presetOptions"
            :key="opt.value"
            size="tiny"
            :type="rangeMin === opt.value ? 'primary' : 'default'"
            :secondary="rangeMin !== opt.value"
            @click="selectPreset(opt.value)"
          >
            {{ opt.label }}
          </NButton>
          <NButton
            size="tiny"
            :type="rangeMin === -1 ? 'primary' : 'default'"
            :secondary="rangeMin !== -1"
            @click="openCustomRange"
          >
            {{ t('query.timeCustom') }}
          </NButton>
          <span class="mv-range-display">
            <NIcon size="12"><TimeOutline /></NIcon>
            {{ rangeDisplay }}
          </span>
        </div>
      </div>
      <div class="mv-header-actions">
        <NButton size="small" quaternary @click="fetchChartData" :disabled="!queryExpression.trim()">
          <template #icon><NIcon><RefreshOutline /></NIcon></template>
        </NButton>
        <NButton size="small" quaternary @click="copyShareUrl" :disabled="!queryExpression.trim()">
          <template #icon><NIcon><CopyOutline /></NIcon></template>
          {{ t('common.share') }}
        </NButton>
        <NButton size="small" quaternary @click="goToExplore" :disabled="!queryExpression.trim()">
          <template #icon><NIcon><SearchOutline /></NIcon></template>
          {{ t('query.openInExplore') }}
        </NButton>
      </div>
    </div>

    <!-- Main content: sidebar + center -->
    <div class="mv-body">
      <!-- Left sidebar: saved views -->
      <div class="mv-sidebar">
        <div class="mv-sidebar-header">
          <NIcon size="14"><BookmarkOutline /></NIcon>
          <span>{{ t('query.savedViews') }}</span>
        </div>
        <!-- Save current view -->
        <div class="mv-save-row">
          <input
            v-model="viewName"
            :placeholder="t('query.viewNamePlaceholder')"
            class="mv-view-input"
            @keydown.enter="saveCurrentView"
          />
          <NButton size="tiny" type="primary" :disabled="!selectedDsId || !selectedMetric" @click="saveCurrentView">
            {{ t('common.save') }}
          </NButton>
        </div>
        <!-- Views list -->
        <div class="mv-views-list">
          <div v-if="!savedViews.length" class="mv-views-empty">
            {{ t('query.noSavedViews') }}
          </div>
          <div
            v-for="view in savedViews"
            :key="view.id"
            class="mv-view-item"
            @click="loadSavedView(view)"
          >
            <div class="mv-view-item-header">
              <span class="mv-view-item-name">{{ view.name }}</span>
              <NButton size="tiny" quaternary type="error" @click.stop="deleteSavedView(view.id)">
                <template #icon><NIcon size="12"><TrashOutline /></NIcon></template>
              </NButton>
            </div>
            <div class="mv-view-item-metric">{{ view.metric }}</div>
          </div>
        </div>

        <NDivider style="margin: 8px 0;" />

        <!-- Query History -->
        <div class="mv-sidebar-header">
          <NIcon size="14"><TimeOutline /></NIcon>
          <span>{{ t('query.queryHistory') }}</span>
          <NButton size="tiny" quaternary @click="clearHistory" style="margin-left: auto;">
            <template #icon><NIcon size="12"><TrashOutline /></NIcon></template>
          </NButton>
        </div>
        <div class="mv-views-list">
          <div v-if="!history.length" class="mv-views-empty">
            {{ t('query.noHistory') }}
          </div>
          <div
            v-for="item in history.slice(0, 20)"
            :key="item.timestamp"
            class="mv-history-item"
            @click="loadHistoryItem(item)"
          >
            <div class="mv-history-expr">{{ item.query }}</div>
            <div class="mv-history-time">{{ new Date(item.timestamp).toLocaleString() }}</div>
          </div>
        </div>
      </div>

      <!-- Center: label filters + metric list + chart -->
      <div class="mv-center">
        <!-- Datasource selector -->
        <div class="mv-ds-row">
          <NSelect
            v-model:value="selectedDsId"
            :options="dsOptions"
            :placeholder="t('query.selectDatasource')"
            filterable
            size="small"
            class="mv-ds-select"
          />
          <div v-if="selectedMetric" class="mv-selected-metric">
            <NTag size="small" :bordered="false" type="info">{{ selectedMetric }}</NTag>
          </div>
        </div>

        <!-- Label filters + Metric list (side by side) -->
        <div class="mv-selector-row">
          <!-- Label filters -->
          <div class="mv-label-panel">
            <MetricLabelSelector
              ref="labelSelectorRef"
              :datasource-id="selectedDsId"
              :metric-name="selectedMetric"
              @update:selector="onSelectorUpdate"
            />
          </div>

          <!-- Metric list -->
          <div class="mv-metric-panel">
            <MetricList
              ref="metricListRef"
              :datasource-id="selectedDsId"
              @select="onMetricSelect"
            />
          </div>
        </div>

        <!-- Chart area -->
        <div class="mv-chart-area">
          <!-- Loading -->
          <div v-if="chartLoading" class="mv-chart-loading">
            <NSpin size="medium" />
          </div>

          <!-- Error -->
          <div v-else-if="chartError" class="mv-chart-error">
            <span>{{ chartError }}</span>
            <NButton size="small" @click="fetchChartData">{{ t('common.retry') }}</NButton>
          </div>

          <!-- Empty -->
          <div v-else-if="!selectedMetric" class="mv-chart-empty">
            <NEmpty :description="t('query.selectMetric')" size="small" />
          </div>

          <div v-else-if="chartData !== null && !chartData.series?.length" class="mv-chart-empty">
            <NEmpty :description="t('query.noResults')" size="small" />
          </div>

          <!-- Chart -->
          <template v-else-if="ChartReady && VChart && chartOption">
            <div class="mv-chart-info">
              <span class="mv-chart-expr">{{ queryExpression }}</span>
              <span v-if="chartData?.series" class="mv-chart-count">
                {{ chartData.series.length }} {{ t('query.seriesCount') }}
              </span>
            </div>
            <component
              :is="VChart"
              :option="chartOption"
              :autoresize="true"
              class="mv-chart"
            />
            <!-- Legend -->
            <div v-if="chartData?.series?.length" class="mv-chart-legend">
              <div
                v-for="(s, i) in chartData.series"
                :key="i"
                class="mv-legend-item"
                @click="openLabelDrawer(s.labels, formatLegend(s.labels))"
              >
                <span class="mv-legend-dot" :style="{ background: ['#5470c6','#91cc75','#fac858','#ee6666','#73c0de','#3ba272','#fc8452','#9a60b4','#ea7ccc','#48b8d0'][i % 10] }" />
                <span class="mv-legend-label">{{ formatLegend(s.labels) }}</span>
              </div>
            </div>
          </template>

          <!-- Fallback: chart not loaded yet -->
          <div v-else-if="selectedMetric && !chartLoading && !chartData" class="mv-chart-empty">
            <NButton type="primary" size="small" @click="fetchChartData">
              <template #icon><NIcon><RefreshOutline /></NIcon></template>
              {{ t('query.runQuery') }}
            </NButton>
          </div>
        </div>
      </div>
    </div>

    <!-- Label Detail Drawer -->
    <NDrawer v-model:show="labelDrawerVisible" :width="400" placement="right">
      <NDrawerContent :title="t('query.labelDetails')">
        <div class="mv-drawer-header">
          <span class="mv-drawer-series">{{ labelDrawerSeriesName }}</span>
        </div>
        <NDescriptions :column="1" bordered size="small" label-placement="left">
          <NDescriptionsItem v-for="(v, k) in labelDrawerData" :key="k" :label="String(k)">
            <span class="mv-drawer-value">{{ v }}</span>
          </NDescriptionsItem>
        </NDescriptions>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped>
.metric-views-page {
  max-width: 1600px;
  padding: 24px;
  height: calc(100vh - 64px);
  display: flex;
  flex-direction: column;
}

/* Header */
.mv-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
  flex-shrink: 0;
  flex-wrap: wrap;
}
.mv-header-left {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.mv-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
  color: var(--sre-text-primary);
}
.mv-time-bar {
  display: flex;
  gap: 4px;
  align-items: center;
  flex-wrap: wrap;
}
.mv-range-display {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: 4px;
}
.mv-header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-shrink: 0;
}

/* Body */
.mv-body {
  display: flex;
  gap: 16px;
  flex: 1;
  min-height: 0;
}

/* Sidebar */
.mv-sidebar {
  width: 260px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  padding: 12px;
  overflow: hidden;
}
.mv-sidebar-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 8px;
}
.mv-save-row {
  display: flex;
  gap: 4px;
  margin-bottom: 8px;
}
.mv-view-input {
  flex: 1;
  min-width: 0;
  border: 1px solid var(--sre-border);
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 12px;
  background: var(--sre-bg-card);
  color: var(--sre-text-primary);
  outline: none;
  transition: border-color 0.15s;
}
.mv-view-input:focus {
  border-color: var(--sre-primary);
}
.mv-views-list {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}
.mv-views-empty {
  text-align: center;
  padding: 12px;
  color: var(--sre-text-tertiary);
  font-size: 12px;
}
.mv-view-item {
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
  margin-bottom: 4px;
  border: 1px solid transparent;
  transition: background 0.12s, border-color 0.12s;
}
.mv-view-item:hover {
  background: var(--sre-bg-hover);
  border-color: var(--sre-border);
}
.mv-view-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.mv-view-item-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mv-view-item-metric {
  font-family: var(--sre-font-mono, monospace);
  font-size: 11px;
  color: var(--sre-text-tertiary);
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* History items */
.mv-history-item {
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
  margin-bottom: 4px;
  border: 1px solid transparent;
  transition: background 0.12s, border-color 0.12s;
}
.mv-history-item:hover {
  background: var(--sre-bg-hover);
  border-color: var(--sre-border);
}
.mv-history-expr {
  font-family: var(--sre-font-mono, monospace);
  font-size: 11px;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mv-history-time {
  font-size: 10px;
  color: var(--sre-text-tertiary);
  margin-top: 2px;
}

/* Center */
.mv-center {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Datasource row */
.mv-ds-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.mv-ds-select {
  width: 280px;
  flex-shrink: 0;
}
.mv-selected-metric {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* Selector row: label filters + metric list side by side */
.mv-selector-row {
  display: flex;
  gap: 12px;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.mv-label-panel {
  width: 380px;
  flex-shrink: 0;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  padding: 12px;
  overflow-y: auto;
}
.mv-metric-panel {
  flex: 1;
  min-width: 0;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  padding: 12px;
  overflow: hidden;
}

/* Chart area */
.mv-chart-area {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  padding: 16px;
  min-height: 320px;
  display: flex;
  flex-direction: column;
}
.mv-chart-loading {
  display: flex;
  justify-content: center;
  align-items: center;
  flex: 1;
  min-height: 200px;
}
.mv-chart-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 40px;
  color: var(--sre-critical);
  font-size: 13px;
}
.mv-chart-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 200px;
}
.mv-chart-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.mv-chart-expr {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mv-chart-count {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  flex-shrink: 0;
}
.mv-chart {
  width: 100%;
  height: 300px;
  flex-shrink: 0;
}

/* Legend */
.mv-chart-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 12px;
  padding: 8px 4px;
  max-height: 80px;
  overflow-y: auto;
  border-top: 1px solid var(--sre-border);
  margin-top: 4px;
  cursor: pointer;
  user-select: none;
}
.mv-legend-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  color: var(--sre-text-secondary);
  transition: background 0.12s;
  max-width: 280px;
}
.mv-legend-item:hover {
  background: var(--sre-bg-hover);
}
.mv-legend-dot {
  width: 12px;
  height: 4px;
  border-radius: 2px;
  flex-shrink: 0;
}
.mv-legend-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--sre-font-mono, monospace);
  font-size: 11px;
}

/* Drawer */
.mv-drawer-header {
  margin-bottom: 16px;
}
.mv-drawer-series {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  font-family: var(--sre-font-mono, monospace);
}
.mv-drawer-value {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  word-break: break-all;
}

/* Responsive */
@media (max-width: 1200px) {
  .mv-sidebar {
    width: 220px;
  }
  .mv-label-panel {
    width: 300px;
  }
}
@media (max-width: 900px) {
  .mv-body {
    flex-direction: column;
  }
  .mv-sidebar {
    width: 100%;
    max-height: 200px;
  }
  .mv-selector-row {
    flex-direction: column;
  }
  .mv-label-panel {
    width: 100%;
  }
}
</style>
