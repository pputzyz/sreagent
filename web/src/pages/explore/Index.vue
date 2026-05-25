<script setup lang="ts">
/**
 * Data Query Page — unified metrics + logs query interface.
 *
 * Multi-panel architecture (Nightingale Metric panels pattern):
 *  - Shared time range toolbar at top
 *  - Each panel is an independent QueryPanelContent instance
 *  - URL querystring sync for shareability
 *  - Per-panel add/remove
 */
import { ref, onMounted, onUnmounted, computed, watch, shallowRef, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  NButton, NDatePicker, NIcon,
  NDrawer, NDrawerContent, NDescriptions, NDescriptionsItem,
  NSelect,
} from 'naive-ui'
import {
  RefreshOutline, TimeOutline, AddOutline,
} from '@vicons/ionicons5'
import { datasourceApi } from '@/api'
import QueryPanelContent from '@/components/query/QueryPanelContent.vue'
import ViewSelect from '@/components/query/ViewSelect.vue'
import type { DataSource } from '@/types'
import type { SavedView } from '@/components/query/ViewSelect.vue'

const { t } = useI18n()
const router = useRouter()

// --- Datasources (shared) ---
const datasources = ref<DataSource[]>([])

async function loadDs() {
  try {
    const res = await datasourceApi.list({ page: 1, page_size: 100 })
    const list = res.data?.data?.list
    datasources.value = (Array.isArray(list) ? list : []).filter((d: DataSource) => d.is_enabled)
  } catch (e) { console.warn('[Explore] Failed to load datasources:', e) }
}

// --- Lazy ECharts (shared) ---
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

// --- Multi-panel state ---
interface Panel {
  id: number
}
const panels = ref<Panel[]>([{ id: 1 }])
let panelCounter = 1

function addPanel() {
  panelCounter++
  panels.value.push({ id: panelCounter })
}

function removePanel(panelId: number) {
  if (panels.value.length <= 1) return
  panels.value = panels.value.filter(p => p.id !== panelId)
}

// --- Time range (shared) ---
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

// --- Step (shared) ---
const stepValue = ref<string>('auto')

// --- Auto-refresh (shared) ---
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
      runAllPanels()
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

function runAllPanels() {
  if (rangeMin.value !== -1) now.value = Date.now()
  // Each panel's run is called via template ref
  panelRefs.value.forEach(ref => ref?.run?.())
}

// --- Panel refs ---
const panelRefs = ref<InstanceType<typeof QueryPanelContent>[]>([])

function setPanelRef(el: InstanceType<typeof QueryPanelContent> | null, idx: number) {
  if (el) panelRefs.value[idx] = el
}

// --- Label detail drawer (shared) ---
const labelDrawerVisible = ref(false)
const labelDrawerData = ref<Record<string, string>>({})
const labelDrawerSeriesName = ref('')

function openLabelDrawer(labels: Record<string, string>, seriesName: string) {
  labelDrawerData.value = labels
  labelDrawerSeriesName.value = seriesName
  labelDrawerVisible.value = true
}

function goToCreateAlertRule(expr: string) {
  // Save Explore state so we can restore when user returns
  const p = panelRefs.value[0] as any
  if (p) {
    sessionStorage.setItem('sre-explore-return', JSON.stringify({
      ds: p.selectedDsId,
      expr: p.expression,
      tab: p.activeTab,
    }))
  }
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

// --- View load handler ---
function onViewLoad(view: SavedView) {
  const p = panelRefs.value[0]
  if (p) {
    p.setState(view.dsId, view.expression, view.tab)
  }
}

// Current panel state for ViewSelect
// Exposed refs are auto-unwrapped via template ref, so access directly
const currentPanelTab = computed(() => {
  const p = panelRefs.value[0] as any
  const tab = p?.activeTab
  return ((tab === 'logs') ? 'logs' : 'metrics') as 'metrics' | 'logs'
})
const currentPanelDsId = computed(() => {
  const p = panelRefs.value[0] as any
  return (p?.selectedDsId ?? null) as number | null
})
const currentPanelDsName = computed(() => {
  const dsId = currentPanelDsId.value
  if (!dsId) return ''
  return datasources.value.find(d => d.id === dsId)?.name || ''
})
const currentPanelExpression = computed(() => {
  const p = panelRefs.value[0] as any
  return (p?.expression || '') as string
})

// --- URL sync ---
function syncToURL() {
  const url = new URL(window.location.href)
  // Time range
  if (rangeMin.value === -1 && customRange.value) {
    url.searchParams.set('start', String(Math.floor(customRange.value[0] / 1000)))
    url.searchParams.set('end', String(Math.floor(customRange.value[1] / 1000)))
    url.searchParams.delete('range')
  } else {
    url.searchParams.set('range', String(rangeMin.value))
    url.searchParams.delete('start')
    url.searchParams.delete('end')
  }
  // Panels (only sync single-panel state to URL for simplicity)
  url.searchParams.delete('ds')
  url.searchParams.delete('expr')
  url.searchParams.delete('tab')
  if (panels.value.length === 1) {
    const p = panelRefs.value[0] as any
    if (p) {
      const ds = p.selectedDsId
      const expr = p.expression
      const tab = p.activeTab
      if (ds) url.searchParams.set('ds', String(ds))
      if (expr) url.searchParams.set('expr', expr)
      if (tab) url.searchParams.set('tab', tab)
    }
  }
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

  if (range) {
    const v = Number(range)
    if (!isNaN(v) && presetOptions.some(p => p.value === v)) rangeMin.value = v
  }
  if (start && end) {
    rangeMin.value = -1
    customRange.value = [Number(start) * 1000, Number(end) * 1000]
    showCustomPicker.value = true
  }
  return { ds: ds ? Number(ds) : null, expr: expr || '', tab: tab || '' }
}

// --- Watch ---
watch(autoRefreshSec, () => startAutoTimer())

// Auto-sync URL when panel state changes (debounced)
let syncTimer: ReturnType<typeof setTimeout> | null = null
function debouncedSyncToURL() {
  if (syncTimer) clearTimeout(syncTimer)
  syncTimer = setTimeout(syncToURL, 500)
}

// Watch first panel's reactive state for URL sync
watch(() => {
  const p = panelRefs.value[0] as any
  if (!p) return null
  return `${p.selectedDsId}-${p.expression}-${p.activeTab}`
}, (val) => {
  if (val) debouncedSyncToURL()
})

// Handle openLabels from child panel
function onPanelOpenLabels(labels: Record<string, string>, seriesName: string) {
  openLabelDrawer(labels, seriesName)
}

// Handle histogram time range zoom from child panel
function onPanelTimeRangeChange(start: number, end: number) {
  // Update shared time range to the zoomed range
  rangeMin.value = -1
  customRange.value = [start * 1000, end * 1000]
  showCustomPicker.value = true
  now.value = Date.now()
}

onMounted(async () => {
  const urlState = syncFromURL()
  await loadDs()
  // Check if returning from alert rule creation
  const returnState = sessionStorage.getItem('sre-explore-return')
  if (returnState) {
    sessionStorage.removeItem('sre-explore-return')
    try {
      const saved = JSON.parse(returnState)
      if (saved.ds && datasources.value.some(d => d.id === saved.ds)) {
        setTimeout(() => {
          panelRefs.value[0]?.setState(saved.ds, saved.expr, saved.tab)
        }, 100)
      }
    } catch { /* ignore */ }
  } else if (urlState.ds && datasources.value.some(d => d.id === urlState.ds)) {
    // Apply URL state to first panel after datasources loaded
    setTimeout(() => {
      panelRefs.value[0]?.setState(urlState.ds, urlState.expr, urlState.tab)
    }, 100)
  }
  loadECharts()
})

onUnmounted(() => {
  stopAutoTimer()
})
</script>

<template>
  <div class="query-page">
    <!-- Header: title + time range + actions (Nightingale: compact inline) -->
    <div class="query-header">
      <div class="header-left">
        <h2 class="query-title">{{ t('query.title') }}</h2>
        <div class="time-range-bar">
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
          <span class="range-display-inline">
            <NIcon size="12"><TimeOutline /></NIcon>
            {{ rangeDisplay }}
          </span>
        </div>
        <div v-if="rangeMin === -1 && showCustomPicker" class="custom-range-inline">
          <NDatePicker
            v-model:value="customRange"
            type="datetimerange"
            size="small"
            clearable
            class="custom-date-picker"
          />
        </div>
      </div>
      <div class="header-actions">
        <NButton size="small" quaternary @click="runAllPanels">
          <template #icon><NIcon><RefreshOutline /></NIcon></template>
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
        <ViewSelect
          :current-tab="currentPanelTab"
          :current-ds-id="currentPanelDsId"
          :current-ds-name="currentPanelDsName"
          :current-expression="currentPanelExpression"
          @load="onViewLoad"
        />
        <NButton size="small" @click="addPanel">
          <template #icon><NIcon><AddOutline /></NIcon></template>
          {{ t('query.addPanel') }}
        </NButton>
      </div>
    </div>

    <!-- Panels -->
    <div class="panels-container">
      <div v-for="(panel, idx) in panels" :key="panel.id" class="panel-wrapper">
        <QueryPanelContent
          :ref="(el: any) => setPanelRef(el, idx)"
          :panelId="panel.id"
          :datasources="datasources"
          :timeStart="timeStart"
          :timeEnd="timeEnd"
          :stepValue="stepValue"
          :ChartReady="ChartReady"
          :VChart="VChart"
          :canClose="panels.length > 1"
          @remove="removePanel"
          @open-labels="onPanelOpenLabels"
          @time-range-change="onPanelTimeRangeChange"
        />
      </div>
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

/* Compact header (Nightingale: inline title + time range + actions) */
.query-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.header-left {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.query-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
  color: var(--sre-text-primary);
}
.time-range-bar {
  display: flex;
  gap: 4px;
  align-items: center;
  flex-wrap: wrap;
}
.range-display-inline {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: 4px;
}
.custom-range-inline {
  margin-top: 4px;
}
.custom-date-picker {
  width: 420px;
}

.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-shrink: 0;
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
.auto-refresh-select {
  width: 120px;
}

.panels-container {
  display: flex;
  flex-direction: column;
  gap: 0;
}

/* Label drawer */
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

@media (max-width: 768px) {
  .query-page {
    padding: 16px;
  }
  .query-header {
    flex-direction: column;
    gap: 12px;
  }
  .header-actions {
    flex-wrap: wrap;
  }
  .custom-date-picker {
    width: 100%;
  }
}
</style>
