<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import {
  NButton, NIcon, NInput, NSelect, NTag, NCard, NScrollbar,
  NDrawer, NDrawerContent, NForm, NFormItem, NEmpty,
} from 'naive-ui'
import { AddOutline, TrashOutline, RefreshOutline } from '@vicons/ionicons5'
import { metricViewApi, type MetricView, type MetricViewConfig, type MetricViewFilter } from '@/api/metric-view'
import { datasourceApi } from '@/api'
import PageHeader from '@/components/common/PageHeader.vue'
import { getErrorMessage } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

// --- State ---
const views = ref<MetricView[]>([])
const loading = ref(false)
const selectedView = ref<MetricView | null>(null)

// Datasource for Prometheus queries
const datasourceId = ref<number | null>(null)
const datasourceOptions = ref<{ label: string; value: number }[]>([])

// Labels & Values from the selected view's filters
const dynamicLabels = ref<Record<string, string>>({})
const dimensionLabels = ref<Record<string, string[]>>({})
const labelValuesCache = ref<Record<string, string[]>>({})

// Metrics list (fetched from Prometheus)
const metricNames = ref<string[]>([])
const metricSearch = ref('')
const selectedMetrics = ref<string[]>([])
const metricLoading = ref(false)

// Graph data
const graphData = ref<Record<string, Array<{ timestamp: number; value: number }>>>({})
const graphLoading = ref<Record<string, boolean>>({})

// Edit drawer
const showEditDrawer = ref(false)
const editMode = ref<'create' | 'edit'>('create')
const editView = ref<Partial<MetricView>>({})
const editConfigs = ref<MetricViewConfig>({
  filters: [],
  dynamicLabels: [],
  dimensionLabels: [],
  ignorePrefix: '',
})

// --- Datasource ---
async function loadDatasources() {
  try {
    const resp = await datasourceApi.list({ page: 1, page_size: 100 })
    const list = (resp.data.data?.list || []).filter((d: any) =>
      d.type === 'prometheus' || d.type === 'victoriametrics'
    )
    datasourceOptions.value = list.map((d: any) => ({ label: d.name, value: d.id }))
    if (list.length > 0 && !datasourceId.value) {
      datasourceId.value = list[0].id
    }
  } catch {}
}

// --- Views CRUD ---
async function fetchViews() {
  loading.value = true
  try {
    const resp = await metricViewApi.list({ page: 1, page_size: 200 })
    views.value = resp.data.data?.list || []
    // Auto-select first view
    if (views.value.length > 0 && !selectedView.value) {
      selectView(views.value[0])
    }
  } catch (e: any) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

function selectView(view: MetricView) {
  selectedView.value = view
  // Parse configs
  const cfg = view.configs_json
  if (cfg) {
    // Init dynamic labels
    const dl: Record<string, string> = {}
    cfg.dynamicLabels?.forEach((d) => { dl[d.label] = d.value || '' })
    dynamicLabels.value = dl
    // Init dimension labels
    const dim: Record<string, string[]> = {}
    cfg.dimensionLabels?.forEach((d) => {
      if (d.length > 0) dim[d[0]] = []
    })
    dimensionLabels.value = dim
    // Fetch label values for each label
    cfg.filters?.forEach((f) => fetchLabelValues(f.label))
    cfg.dynamicLabels?.forEach((d) => fetchLabelValues(d.label))
    cfg.dimensionLabels?.forEach((d) => { if (d[0]) fetchLabelValues(d[0]) }
    )
  }
  // Fetch metrics
  fetchMetricNames()
}

function openCreate() {
  editMode.value = 'create'
  editView.value = {}
  editConfigs.value = { filters: [], dynamicLabels: [], dimensionLabels: [], ignorePrefix: '' }
  showEditDrawer.value = true
}

function openEdit(view: MetricView) {
  editMode.value = 'edit'
  editView.value = { ...view }
  editConfigs.value = view.configs_json ? { ...view.configs_json } : { filters: [], dynamicLabels: [], dimensionLabels: [], ignorePrefix: '' }
  showEditDrawer.value = true
}

async function handleSaveView() {
  if (!editView.value.name) {
    message.warning(t('metricViews.nameRequired'))
    return
  }
  try {
    const data = { name: editView.value.name!, configs: editConfigs.value }
    if (editMode.value === 'edit' && editView.value.id) {
      await metricViewApi.update(editView.value.id, data)
      message.success(t('common.savedSuccess'))
    } else {
      await metricViewApi.create(data)
      message.success(t('common.createSuccess'))
    }
    showEditDrawer.value = false
    await fetchViews()
  } catch (e: any) {
    message.error(getErrorMessage(e))
  }
}

async function handleDeleteView(view: MetricView) {
  dialog.warning({
    title: t('common.confirm'),
    content: t('metricViews.confirmDelete', { name: view.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await metricViewApi.delete(view.id)
        message.success(t('common.deleteSuccess'))
        if (selectedView.value?.id === view.id) selectedView.value = null
        await fetchViews()
      } catch (e: any) {
        message.error(getErrorMessage(e))
      }
    },
  })
}

// --- Label values from Prometheus ---
async function fetchLabelValues(label: string) {
  if (!datasourceId.value || labelValuesCache.value[label]) return
  try {
    const resp = await datasourceApi.labelValues(datasourceId.value, label)
    labelValuesCache.value[label] = resp.data.data || []
  } catch {
    labelValuesCache.value[label] = []
  }
}

// --- Build match string from filters + dynamic labels ---
const matchString = computed(() => {
  if (!selectedView.value?.configs_json) return ''
  const cfg = selectedView.value.configs_json
  const parts: string[] = []
  // Static filters
  cfg.filters?.forEach((f) => {
    if (f.label && f.value) {
      parts.push(`${f.label}${f.oper}"${f.value}"`)
    }
  })
  // Dynamic labels (selected values)
  Object.entries(dynamicLabels.value).forEach(([k, v]) => {
    if (v) parts.push(`${k}="${v}"`)
  })
  return parts.length > 0 ? `{${parts.join(',')}}` : ''
})

// --- Fetch metric names from Prometheus ---
async function fetchMetricNames() {
  if (!datasourceId.value) return
  metricLoading.value = true
  try {
    const params: Record<string, string> = {}
    if (matchString.value) params['match[]'] = matchString.value
    const resp = await datasourceApi.proxy(datasourceId.value, '/api/v1/label/__name__/values', params)
    metricNames.value = (resp.data?.data as string[]) || []
  } catch {
    metricNames.value = []
  } finally {
    metricLoading.value = false
  }
}

// Filtered metrics by search
const filteredMetrics = computed(() => {
  const q = metricSearch.value.toLowerCase()
  if (!q) return metricNames.value
  return metricNames.value.filter((m) => m.toLowerCase().includes(q))
})

// Group metrics by prefix (respecting ignorePrefix)
const metricGroups = computed(() => {
  const prefix = selectedView.value?.configs_json?.ignorePrefix || ''
  const groups: Record<string, string[]> = {}
  filteredMetrics.value.forEach((m) => {
    let name = m
    if (prefix && name.startsWith(prefix)) name = name.slice(prefix.length)
    const idx = name.indexOf('_')
    const group = idx > 0 ? name.slice(0, idx) : 'other'
    if (!groups[group]) groups[group] = []
    groups[group].push(m)
  })
  return groups
})

const activeGroup = ref('')

// --- Graph ---
async function fetchGraph(metric: string) {
  if (!datasourceId.value) return
  graphLoading.value[metric] = true
  try {
    const end = Math.floor(Date.now() / 1000)
    const start = end - 3600 // 1h
    const expr = matchString.value ? `${metric}${matchString.value}` : metric
    const resp = await datasourceApi.rangeQuery(datasourceId.value, {
      expression: expr,
      start,
      end,
      step: '15s',
    })
    const results = (resp.data?.data as any)?.result
    if (results && results.length > 0) {
      graphData.value[metric] = results[0].values?.map((v: [number, string]) => ({
        timestamp: v[0],
        value: parseFloat(v[1]),
      })) || []
    }
  } catch {
    graphData.value[metric] = []
  } finally {
    graphLoading.value[metric] = false
  }
}

function toggleMetric(metric: string) {
  const idx = selectedMetrics.value.indexOf(metric)
  if (idx >= 0) {
    selectedMetrics.value.splice(idx, 1)
  } else {
    selectedMetrics.value.push(metric)
    fetchGraph(metric)
  }
}

// --- Edit config helpers ---
function addFilter() {
  editConfigs.value.filters.push({ label: '', oper: '=', value: '' })
}
function removeFilter(idx: number) {
  editConfigs.value.filters.splice(idx, 1)
}
function addDynamicLabel() {
  editConfigs.value.dynamicLabels.push({ label: '', value: '' })
}
function removeDynamicLabel(idx: number) {
  editConfigs.value.dynamicLabels.splice(idx, 1)
}
function addDimensionLabel() {
  editConfigs.value.dimensionLabels.push([''])
}
function removeDimensionLabel(idx: number) {
  editConfigs.value.dimensionLabels.splice(idx, 1)
}

// --- Init ---
onMounted(() => {
  loadDatasources()
  fetchViews()
})
</script>

<template>
  <div class="metric-views-page">
    <PageHeader :title="t('menu.metricViews')" />

    <div class="three-column-layout">
      <!-- Left: View List -->
      <div class="left-panel">
        <div class="panel-header">
          <span class="panel-title">{{ t('metricViews.views') }}</span>
          <NButton size="tiny" quaternary @click="openCreate">
            <template #icon><NIcon><AddOutline /></NIcon></template>
          </NButton>
        </div>
        <NScrollbar style="flex: 1;">
          <div
            v-for="view in views"
            :key="view.id"
            class="view-item"
            :class="{ active: selectedView?.id === view.id }"
            @click="selectView(view)"
          >
            <div class="view-item-header">
              <span class="view-item-name">{{ view.name }}</span>
              <div class="view-item-actions">
                <NButton size="tiny" quaternary @click.stop="openEdit(view)">
                  <template #icon><NIcon size="12"><RefreshOutline /></NIcon></template>
                </NButton>
                <NButton size="tiny" quaternary type="error" @click.stop="handleDeleteView(view)">
                  <template #icon><NIcon size="12"><TrashOutline /></NIcon></template>
                </NButton>
              </div>
            </div>
            <div class="view-item-meta">
              <NTag v-if="view.configs_json?.filters?.length" size="tiny" :bordered="false" type="info">
                {{ view.configs_json.filters.length }} {{ t('metricViews.filters') }}
              </NTag>
            </div>
          </div>
          <NEmpty v-if="!views.length && !loading" :description="t('metricViews.noViews')" style="padding: 24px;" />
        </NScrollbar>
      </div>

      <!-- Middle: Labels & Values Filter -->
      <div class="middle-panel">
        <div class="panel-header">
          <span class="panel-title">{{ t('metricViews.labelFilter') }}</span>
          <NSelect
            v-model:value="datasourceId"
            :options="datasourceOptions"
            size="tiny"
            style="width: 140px;"
            @update:value="fetchMetricNames"
          />
        </div>
        <NScrollbar style="flex: 1;">
          <div v-if="!selectedView" class="empty-hint">
            {{ t('metricViews.selectViewHint') }}
          </div>
          <template v-else>
            <!-- Static Filters (read-only display) -->
            <div v-if="selectedView.configs_json?.filters?.length" class="filter-section">
              <div class="section-label">{{ t('metricViews.staticFilters') }}</div>
              <div
                v-for="(f, idx) in selectedView.configs_json.filters"
                :key="idx"
                class="filter-row"
              >
                <NTag size="small" :bordered="false">{{ f.label }} {{ f.oper }} {{ f.value }}</NTag>
              </div>
            </div>

            <!-- Dynamic Labels (single-select dropdowns) -->
            <div v-if="selectedView.configs_json?.dynamicLabels?.length" class="filter-section">
              <div class="section-label">{{ t('metricViews.dynamicLabels') }}</div>
              <div
                v-for="dl in selectedView.configs_json.dynamicLabels"
                :key="dl.label"
                class="label-select-row"
              >
                <span class="label-name">{{ dl.label }}</span>
                <NSelect
                  v-model:value="dynamicLabels[dl.label]"
                  :options="(labelValuesCache[dl.label] || []).map(v => ({ label: v, value: v }))"
                  size="small"
                  filterable
                  clearable
                  :placeholder="t('metricViews.selectValue')"
                  style="flex: 1;"
                  @update:value="fetchMetricNames"
                />
              </div>
            </div>

            <!-- Dimension Labels (multi-select checkboxes) -->
            <div v-if="selectedView.configs_json?.dimensionLabels?.length" class="filter-section">
              <div class="section-label">{{ t('metricViews.dimensionLabels') }}</div>
              <div
                v-for="dim in selectedView.configs_json.dimensionLabels"
                :key="dim[0]"
                class="dimension-section"
              >
                <span class="label-name">{{ dim[0] }}</span>
                <NSelect
                  v-model:value="dimensionLabels[dim[0]]"
                  :options="(labelValuesCache[dim[0]] || []).map(v => ({ label: v, value: v }))"
                  multiple
                  size="small"
                  filterable
                  :placeholder="t('metricViews.selectValues')"
                />
              </div>
            </div>
          </template>
        </NScrollbar>
      </div>

      <!-- Right: Metrics + Graphs -->
      <div class="right-panel">
        <div class="panel-header">
          <NInput
            v-model:value="metricSearch"
            :placeholder="t('metricViews.searchMetrics')"
            size="small"
            clearable
            style="width: 260px;"
          />
          <NButton size="small" @click="fetchMetricNames" :loading="metricLoading">
            <template #icon><NIcon><RefreshOutline /></NIcon></template>
          </NButton>
        </div>

        <div class="metrics-content">
          <!-- Metric Groups (tabs) -->
          <div class="metric-tabs">
            <NTag
              v-for="(_, group) in metricGroups"
              :key="group"
              size="small"
              :type="activeGroup === group ? 'info' : 'default'"
              checkable
              :checked="activeGroup === group"
              @update:checked="(v: boolean) => activeGroup = v ? group : ''"
            >
              {{ group }}
            </NTag>
          </div>

          <!-- Metric List -->
          <div class="metric-list">
            <div
              v-for="metric in (activeGroup ? metricGroups[activeGroup] || [] : filteredMetrics.slice(0, 100))"
              :key="metric"
              class="metric-item"
              :class="{ selected: selectedMetrics.includes(metric) }"
              @click="toggleMetric(metric)"
            >
              {{ metric }}
            </div>
            <NEmpty v-if="!filteredMetrics.length && !metricLoading" :description="t('metricViews.noMetrics')" />
          </div>

          <!-- Graph Area -->
          <div v-if="selectedMetrics.length" class="graph-area">
            <NCard
              v-for="metric in selectedMetrics"
              :key="metric"
              size="small"
              :title="metric"
              class="graph-card"
            >
              <div v-if="graphLoading[metric]" class="graph-loading">{{ t('common.loading') }}...</div>
              <div v-else-if="graphData[metric]?.length" class="graph-placeholder">
                {{ graphData[metric].length }} {{ t('metricViews.dataPoints') }}
              </div>
              <div v-else class="graph-placeholder">{{ t('metricViews.noData') }}</div>
            </NCard>
          </div>
        </div>
      </div>
    </div>

    <!-- Edit/Create Drawer -->
    <NDrawer v-model:show="showEditDrawer" :width="600">
      <NDrawerContent :title="editMode === 'edit' ? t('metricViews.editView') : t('metricViews.createView')">
        <NForm label-placement="left" label-width="100px">
          <NFormItem :label="t('metricViews.viewName')" required>
            <NInput v-model:value="editView.name" :placeholder="t('metricViews.viewNamePlaceholder')" />
          </NFormItem>

          <!-- Filters -->
          <NFormItem :label="t('metricViews.filters')">
            <div style="width: 100%;">
              <div v-for="(f, idx) in editConfigs.filters" :key="idx" style="display: flex; gap: 6px; margin-bottom: 6px;">
                <NInput v-model:value="f.label" size="small" placeholder="label" style="width: 120px;" />
                <NSelect v-model:value="f.oper" size="small" style="width: 70px;"
                  :options="[{ label: '=', value: '=' }, { label: '!=', value: '!=' }, { label: '=~', value: '=~' }, { label: '!~', value: '!~' }]" />
                <NInput v-model:value="f.value" size="small" placeholder="value" style="flex: 1;" />
                <NButton size="tiny" quaternary type="error" @click="removeFilter(idx)">-</NButton>
              </div>
              <NButton size="small" dashed @click="addFilter">+ {{ t('metricViews.addFilter') }}</NButton>
            </div>
          </NFormItem>

          <!-- Dynamic Labels -->
          <NFormItem :label="t('metricViews.dynamicLabels')">
            <div style="width: 100%;">
              <div v-for="(dl, idx) in editConfigs.dynamicLabels" :key="idx" style="display: flex; gap: 6px; margin-bottom: 6px;">
                <NInput v-model:value="dl.label" size="small" placeholder="label name" style="flex: 1;" />
                <NButton size="tiny" quaternary type="error" @click="removeDynamicLabel(idx)">-</NButton>
              </div>
              <NButton size="small" dashed @click="addDynamicLabel">+ {{ t('metricViews.addDynamicLabel') }}</NButton>
            </div>
          </NFormItem>

          <!-- Dimension Labels -->
          <NFormItem :label="t('metricViews.dimensionLabels')">
            <div style="width: 100%;">
              <div v-for="(dim, idx) in editConfigs.dimensionLabels" :key="idx" style="display: flex; gap: 6px; margin-bottom: 6px;">
                <NInput v-model:value="dim[0]" size="small" placeholder="label name" style="flex: 1;" />
                <NButton size="tiny" quaternary type="error" @click="removeDimensionLabel(idx)">-</NButton>
              </div>
              <NButton size="small" dashed @click="addDimensionLabel">+ {{ t('metricViews.addDimensionLabel') }}</NButton>
            </div>
          </NFormItem>

          <!-- Ignore Prefix -->
          <NFormItem :label="t('metricViews.ignorePrefix')">
            <NInput v-model:value="editConfigs.ignorePrefix" :placeholder="t('metricViews.ignorePrefixPlaceholder')" />
          </NFormItem>
        </NForm>
        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 8px;">
            <NButton @click="showEditDrawer = false">{{ t('common.cancel') }}</NButton>
            <NButton type="primary" @click="handleSaveView">{{ t('common.save') }}</NButton>
          </div>
        </template>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped>
.metric-views-page {
  padding: 16px;
  height: calc(100vh - 60px);
  display: flex;
  flex-direction: column;
}
.three-column-layout {
  display: flex;
  gap: 12px;
  flex: 1;
  min-height: 0;
}
.left-panel {
  width: 220px;
  border: 1px solid var(--n-border-color);
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.middle-panel {
  width: 280px;
  border: 1px solid var(--n-border-color);
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.right-panel {
  flex: 1;
  border: 1px solid var(--n-border-color);
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--n-border-color);
  gap: 8px;
}
.panel-title {
  font-size: 13px;
  font-weight: 600;
}
.view-item {
  padding: 8px 12px;
  cursor: pointer;
  border-bottom: 1px solid var(--n-border-color);
}
.view-item:hover {
  background: var(--n-color-hover);
}
.view-item.active {
  background: var(--n-primary-color-suppl);
}
.view-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.view-item-name {
  font-size: 13px;
  font-weight: 500;
}
.view-item-actions {
  display: flex;
  gap: 2px;
}
.view-item-meta {
  margin-top: 4px;
}
.filter-section {
  padding: 8px 12px;
  border-bottom: 1px solid var(--n-border-color);
}
.section-label {
  font-size: 11px;
  color: var(--n-text-color-3);
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.filter-row {
  margin-bottom: 4px;
}
.label-select-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.label-name {
  font-size: 12px;
  min-width: 80px;
  color: var(--n-text-color-2);
}
.dimension-section {
  margin-bottom: 8px;
}
.empty-hint {
  padding: 24px;
  text-align: center;
  font-size: 13px;
  color: var(--n-text-color-3);
}
.metrics-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.metric-tabs {
  display: flex;
  gap: 4px;
  padding: 8px 12px;
  flex-wrap: wrap;
  border-bottom: 1px solid var(--n-border-color);
}
.metric-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}
.metric-item {
  padding: 4px 12px;
  font-size: 12px;
  font-family: var(--sre-font-mono, monospace);
  cursor: pointer;
}
.metric-item:hover {
  background: var(--n-color-hover);
}
.metric-item.selected {
  background: var(--n-primary-color-suppl);
  color: var(--n-primary-color);
}
.graph-area {
  border-top: 1px solid var(--n-border-color);
  max-height: 50%;
  overflow-y: auto;
  padding: 8px;
}
.graph-card {
  margin-bottom: 8px;
}
.graph-loading, .graph-placeholder {
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--n-text-color-3);
}
</style>
