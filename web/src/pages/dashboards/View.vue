<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed, nextTick, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NSpace, NInput, NSelect, useMessage, NModal, NPopconfirm, NSpin, NDropdown } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { dashboardV2Api, datasourceApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import type { DashboardV2, DashboardConfig, PanelConfig, VariableConfig } from '@/types/dashboard'
import type { DataSource } from '@/types'
import { useTimeRange } from '@/composables/useTimeRange'
import { useQueryEngine, createDefaultTarget } from '@/composables/useQueryEngine'
import { useVariable } from '@/composables/useVariable'
import TimeRangePicker from '@/components/time/TimeRangePicker.vue'
import RefreshPicker from '@/components/time/RefreshPicker.vue'
import QueryPanel from '@/components/query/QueryPanel.vue'
import QueryResultChart from '@/components/query/QueryResultChart.vue'
import PanelCard from '@/components/query/PanelCard.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import PanelEditor from '@/components/dashboard/PanelEditor.vue'
import { ArrowBackOutline, AddOutline, SettingsOutline, CreateOutline, CopyOutline, ExpandOutline, EllipsisHorizontalOutline, DownloadOutline, TrashOutline } from '@vicons/ionicons5'
import VariableEditor from '@/components/dashboard/VariableEditor.vue'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const { t } = useI18n()

const isNew = computed(() => route.params.id === 'new')
const dashboard = ref<DashboardV2 | null>(null)
const loading = ref(false)
const saving = ref(false)
const config = ref<DashboardConfig>({
  panels: [],
  layout: { cols: 24, rowHeight: 100 },
  variables: [],
})

// --- Panel drag / resize ---
const GRID_COLS = 24
const MIN_W = 2
const MIN_H = 2
const GAP = 12

interface DragState {
  panelId: string
  mode: 'move' | 'resize'
  startX: number
  startY: number
  origX: number
  origY: number
  origW: number
  origH: number
}

const dragState = ref<DragState | null>(null)
const gridEl = ref<HTMLElement | null>(null)
const cellW = ref(100)

function recalcCellW() {
  if (gridEl.value) {
    const w = gridEl.value.clientWidth
    cellW.value = (w - (GRID_COLS - 1) * GAP) / GRID_COLS
  }
}

function onGridResize() {
  recalcCellW()
}

function startDrag(e: MouseEvent, panel: PanelConfig) {
  if ((e.target as HTMLElement).closest('button, input, .panel-drag-handle-resize')) return
  e.preventDefault()
  recalcCellW()
  dragState.value = {
    panelId: panel.id,
    mode: 'move',
    startX: e.clientX,
    startY: e.clientY,
    origX: panel.gridPos.x,
    origY: panel.gridPos.y,
    origW: panel.gridPos.w,
    origH: panel.gridPos.h,
  }
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}

function startResize(e: MouseEvent, panel: PanelConfig) {
  e.preventDefault()
  e.stopPropagation()
  recalcCellW()
  dragState.value = {
    panelId: panel.id,
    mode: 'resize',
    startX: e.clientX,
    startY: e.clientY,
    origX: panel.gridPos.x,
    origY: panel.gridPos.y,
    origW: panel.gridPos.w,
    origH: panel.gridPos.h,
  }
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}

function onDrag(e: MouseEvent) {
  const ds = dragState.value
  if (!ds) return
  const panel = config.value.panels.find(p => p.id === ds.panelId)
  if (!panel) return

  const cw = cellW.value || 1
  const rowH = config.value.layout?.rowHeight || 100
  const dx = Math.round((e.clientX - ds.startX) / (cw + GAP / GRID_COLS))
  const dy = Math.round((e.clientY - ds.startY) / (rowH + GAP))

  if (ds.mode === 'move') {
    const nx = Math.max(0, Math.min(GRID_COLS - panel.gridPos.w, ds.origX + dx))
    const ny = Math.max(0, ds.origY + dy)
    panel.gridPos.x = nx
    panel.gridPos.y = ny
  } else {
    const nw = Math.max(MIN_W, Math.min(GRID_COLS - panel.gridPos.x, ds.origW + dx))
    const nh = Math.max(MIN_H, ds.origH + dy)
    panel.gridPos.w = nw
    panel.gridPos.h = nh
  }
}

function stopDrag() {
  dragState.value = null
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
}

onBeforeUnmount(() => {
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
})

const datasources = ref<DataSource[]>([])

const {
  timeRange,
  isRelative,
  relativeDuration,
  autoRefreshInterval,
  setRelative,
  setAbsolute,
} = useTimeRange('1h')

const {
  targets,
  globalLoading,
  addTarget,
  removeTarget,
  toggleTarget,
  updateTarget,
  executeAll,
  executeQuery,
} = useQueryEngine(timeRange)

const variableConfig = ref<VariableConfig[]>(config.value.variables || [])
const {
  variableList,
  replaceVariables,
  setValue,
  setMultiValue,
  resolveAll,
  adhocFilters,
  addAdhocFilter,
  removeAdhocFilter,
} = useVariable(variableConfig, timeRange)

// Variable editor drawer
const showVariableEditor = ref(false)
const newAdhocKey = ref('')
const newAdhocOp = ref('=')
const newAdhocValue = ref('')

// --- Panel management ---
const panelToDelete = ref<PanelConfig | null>(null)

function addPanelFromQuery(type: PanelConfig['type'] = 'timeseries') {
  const activeTargets = targets.value.filter(t => t.enabled && t.datasourceId && t.expression?.trim())
  if (!activeTargets.length) {
    message.warning(t('dashboardV2.noQueryToAdd'))
    return
  }
  const panel: PanelConfig = {
    id: `panel-${Date.now()}`,
    title: `${t('tooltip.panelPrefix')} ${config.value.panels.length + 1}`,
    type,
    gridPos: { x: 0, y: config.value.panels.length * 6, w: 24, h: 6 },
    targets: activeTargets.map(t => ({
      datasourceId: t.datasourceId!,
      expression: t.expression,
      legendFormat: t.legendFormat || '',
    })),
    options: {},
  }
  config.value.panels.push(panel)
  message.success(t('dashboardV2.panelAdded'))
}

function removePanel(id: string) {
  config.value.panels = config.value.panels.filter(p => p.id !== id)
}

function updatePanelTitle(id: string, title: string) {
  const p = config.value.panels.find(p => p.id === id)
  if (p) p.title = title
}

// --- Panel editor ---
const editingPanel = ref<PanelConfig | null>(null)
const showPanelEditor = ref(false)

function editPanel(id: string) {
  const p = config.value.panels.find(p => p.id === id)
  if (p) {
    editingPanel.value = p
    showPanelEditor.value = true
  }
}

function duplicatePanel(panel: PanelConfig) {
  const clone: PanelConfig = JSON.parse(JSON.stringify(panel))
  clone.id = `panel-${Date.now()}`
  clone.title = `${panel.title} (copy)`
  clone.gridPos = { ...panel.gridPos, y: panel.gridPos.y + panel.gridPos.h }
  config.value.panels.push(clone)
  message.success('Panel duplicated')
}

function onPanelSave(updated: PanelConfig) {
  const idx = config.value.panels.findIndex(p => p.id === updated.id)
  if (idx >= 0) {
    config.value.panels[idx] = updated
  }
  showPanelEditor.value = false
  editingPanel.value = null
}

// --- Row panel collapse/expand ---
function toggleRowCollapse(panel: PanelConfig) {
  if (panel.type !== 'row') return
  if (!panel.options) panel.options = {}
  panel.options.collapsed = !panel.options.collapsed
}

function isRowCollapsed(panel: PanelConfig): boolean {
  return panel.type === 'row' && panel.options?.collapsed === true
}

const visiblePanels = computed(() => {
  const collapsedRowY = new Set<number>()
  for (const p of config.value.panels) {
    if (p.type === 'row' && p.options?.collapsed) {
      collapsedRowY.add(p.gridPos.y)
    }
  }
  if (collapsedRowY.size === 0) return config.value.panels
  return config.value.panels.filter(p => {
    if (p.type === 'row') return true
    // Hide panels whose gridPos.y is below a collapsed row (same visual section)
    return !Array.from(collapsedRowY).some(rowY => p.gridPos.y > rowY && p.gridPos.y < rowY + 100)
  })
})

// --- Data ---
async function fetchDatasources() {
  try {
    const res = await datasourceApi.list({ page: 1, page_size: 100 })
    datasources.value = (res.data.data.list || []).filter((ds: DataSource) => ds.is_enabled)
  } catch (e) { console.warn('[Dashboard] Failed to fetch datasources:', e) }
}

async function fetchDashboard() {
  if (isNew.value) return
  loading.value = true
  try {
    const res = await dashboardV2Api.get(Number(route.params.id))
    dashboard.value = res.data.data
    if (dashboard.value.config) {
      try {
        config.value = JSON.parse(dashboard.value.config)
        // Ensure panels array exists
        if (!config.value.panels) config.value.panels = []
        if (!config.value.layout) config.value.layout = { cols: 24, rowHeight: 100 }
        variableConfig.value = config.value.variables || []
      } catch (e) { console.warn('[Dashboard] Failed to parse config JSON:', e) }
    }
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('common.loadFailed'))
    router.back()
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    const cfg = { ...config.value, variables: variableConfig.value }
    const data = {
      name: dashboard.value?.name || t('tooltip.untitled'),
      description: dashboard.value?.description || '',
      tags: dashboard.value?.tags || {},
      config: JSON.stringify(cfg),
      is_public: dashboard.value?.is_public || false,
    }
    if (isNew.value) {
      const res = await dashboardV2Api.create(data)
      message.success(t('dashboardV2.created'))
      router.replace('/alert/dashboards/' + res.data.data.id)
    } else if (dashboard.value) {
      await dashboardV2Api.update(dashboard.value.id, data)
      message.success(t('dashboardV2.saved'))
    }
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

// --- More actions ---
const moreActionOptions = computed(() => [
  { label: t('dashboardV2.clone'), key: 'clone', icon: () => h(CopyOutline) },
  { label: t('dashboardV2.export'), key: 'export', icon: () => h(DownloadOutline) },
  { type: 'divider', key: 'd1' },
  { label: t('common.delete'), key: 'delete', icon: () => h(TrashOutline) },
])

async function handleMoreAction(key: string | number) {
  if (key === 'clone') {
    await handleClone()
  } else if (key === 'export') {
    handleExport()
  } else if (key === 'delete') {
    handleDeleteDashboard()
  }
}

async function handleClone() {
  if (!dashboard.value) return
  try {
    const panels = config.value.panels.map(p => ({
      ...p,
      id: `panel-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    }))
    const cfg = { ...config.value, panels }
    await dashboardV2Api.create({
      name: `${dashboard.value.name} (copy)`,
      description: dashboard.value.description,
      tags: dashboard.value.tags,
      config: JSON.stringify(cfg),
      is_public: dashboard.value.is_public,
    })
    message.success(t('dashboardV2.cloneSuccess'))
    router.push('/alert/dashboards')
  } catch (err: unknown) {
    message.error((err as Error)?.message || t('common.loadFailed'))
  }
}

function handleExport() {
  if (!dashboard.value) return
  const exportData = {
    ...dashboard.value,
    config: config.value,
  }
  const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${dashboard.value.name}.json`
  a.click()
  URL.revokeObjectURL(url)
  message.success(t('dashboardV2.exportSuccess'))
}

async function handleDeleteDashboard() {
  if (!dashboard.value) return
  try {
    await dashboardV2Api.delete(dashboard.value.id)
    message.success(t('dashboardV2.deleted'))
    router.push('/alert/dashboards')
  } catch (err: unknown) {
    message.error((err as Error)?.message || t('common.deleteFailed'))
  }
}

function handleExecuteSingle(id: string) {
  const target = targets.value.find(t => t.id === id)
  if (target) executeQuery(target)
}

const hasPanels = computed(() => config.value.panels.length > 0)
const hasResults = computed(() => targets.value.some(t => t.series && t.series.length > 0))
const isLoadingDashboard = computed(() => loading.value && !isNew.value)

onMounted(() => {
  fetchDatasources()
  fetchDashboard()
})
</script>

<template>
  <div class="dashboard-view">
    <!-- Header -->
    <div class="dashboard-header">
      <div class="header-left">
        <NButton quaternary size="small" @click="router.push('/alert/dashboards')">
          <template #icon><ArrowBackOutline /></template>
          {{ t('dashboardV2.back') }}
        </NButton>
        <NInput
          v-if="dashboard || isNew"
          :value="dashboard?.name || ''"
          :placeholder="t('dashboardV2.name')"
          size="small"
          class="dash-name-input"
          @update:value="(v: string) => { if (dashboard) dashboard.name = v; else dashboard = { name: v } as DashboardV2 }"
        />
      </div>
      <div class="header-right">
        <TimeRangePicker
          :time-range="timeRange"
          :is-relative="isRelative"
          :relative-duration="relativeDuration"
          @set-relative="setRelative"
          @set-absolute="setAbsolute"
        />
        <RefreshPicker
          :value="autoRefreshInterval"
          @update:value="(v) => autoRefreshInterval = v"
        />
        <NButton type="primary" size="small" :loading="saving" @click="handleSave">
          {{ t('dashboardV2.save') }}
        </NButton>
        <NDropdown
          v-if="!isNew"
          :options="moreActionOptions"
          trigger="click"
          @select="handleMoreAction"
        >
          <NButton quaternary size="small">
            <template #icon><EllipsisHorizontalOutline /></template>
          </NButton>
        </NDropdown>
      </div>
    </div>

    <!-- Loading state -->
    <LoadingSkeleton v-if="isLoadingDashboard" :rows="4" variant="card-grid" />

    <template v-else>
      <!-- Variable bar -->
      <div v-if="variableList.length > 0" class="variable-bar">
        <div class="variable-bar-items">
          <div v-for="v in variableList" :key="v.config.name" class="var-item">
            <label>{{ v.config.label || v.config.name }}</label>

            <!-- Multi-select mode -->
            <NSelect
              v-if="v.config.multi"
              :value="Array.isArray(v.value) ? v.value : v.value ? [v.value] : []"
              :options="v.options.map(o => ({ label: o === '__all' ? 'All' : o, value: o }))"
              :loading="v.loading"
              multiple
              size="small"
              class="var-select"
              @update:value="(vals: string[]) => setMultiValue(v.config.name, vals)"
            />

            <!-- Query / Custom / Interval / Datasource single-select -->
            <NSelect
              v-else-if="v.config.type === 'query' || v.config.type === 'custom' || v.config.type === 'interval' || v.config.type === 'datasource'"
              :value="v.value as string"
              :options="v.options.map(o => ({ label: o === '__all' ? 'All' : o, value: o }))"
              :loading="v.loading"
              size="small"
              class="var-select"
              @update:value="(val: string) => setValue(v.config.name, val)"
            />

            <!-- Textbox -->
            <NInput
              v-else-if="v.config.type === 'textbox'"
              :value="v.value as string"
              size="small"
              class="var-select"
              @update:value="(val: string) => setValue(v.config.name, val)"
            />

            <!-- Adhoc: show filters + add button -->
            <div v-else-if="v.config.type === 'adhoc'" class="adhoc-filters">
              <NSpace :size="4">
                <NButton
                  v-for="(f, fi) in (adhocFilters.get(v.config.name) || [])"
                  :key="fi"
                  size="tiny"
                  secondary
                  type="info"
                  @click="removeAdhocFilter(v.config.name, fi)"
                >
                  {{ f.key }} {{ f.op }} {{ f.value }} &times;
                </NButton>
                <NPopconfirm @positive-click="addAdhocFilter(v.config.name, { key: newAdhocKey, op: newAdhocOp, value: newAdhocValue }); newAdhocKey = ''; newAdhocValue = ''">
                  <template #trigger>
                    <NButton size="tiny" dashed>
                      <template #icon><AddOutline /></template>
                      Filter
                    </NButton>
                  </template>
                  <div style="display: flex; flex-direction: column; gap: 6px; min-width: 240px;">
                    <NInput v-model:value="newAdhocKey" size="small" placeholder="Label key" />
                    <NSelect v-model:value="newAdhocOp" size="small" :options="['=', '!=', '=~', '!~'].map(o => ({ label: o, value: o }))" />
                    <NInput v-model:value="newAdhocValue" size="small" placeholder="Value" />
                  </div>
                </NPopconfirm>
              </NSpace>
            </div>

            <!-- Constant / fallback -->
            <span v-else class="var-value">{{ Array.isArray(v.value) ? v.value.join(', ') : v.value }}</span>
          </div>
        </div>
        <NButton quaternary size="small" class="var-manage-btn" @click="showVariableEditor = true">
          <template #icon><SettingsOutline /></template>
        </NButton>
      </div>

      <!-- PANEL GRID -->
      <div v-if="hasPanels" ref="gridEl" class="panel-grid" @mousemove="onGridResize">
        <template v-for="panel in visiblePanels" :key="panel.id">
          <!-- Row panel -->
          <div
            v-if="panel.type === 'row'"
            class="panel-grid-item panel-row-item"
            :style="{
              gridColumn: `1 / span ${GRID_COLS}`,
              gridRow: `${(panel.gridPos?.y || 0) + 1} / span 1`,
            }"
          >
            <div class="panel-row-toolbar" @mousedown="(e: MouseEvent) => startDrag(e, panel)">
              <NButton quaternary size="tiny" @click.stop="toggleRowCollapse(panel)">
                <template #icon><ExpandOutline style="transition: transform 0.2s" :style="{ transform: isRowCollapsed(panel) ? 'rotate(-90deg)' : 'rotate(0deg)' }" /></template>
              </NButton>
              <span class="panel-row-title">{{ panel.title }}</span>
              <NSpace :size="4" @mousedown.stop>
                <NButton quaternary size="tiny" @click="editPanel(panel.id)">
                  <template #icon><CreateOutline /></template>
                </NButton>
                <NButton quaternary size="tiny" @click="duplicatePanel(panel)">
                  <template #icon><CopyOutline /></template>
                </NButton>
                <NPopconfirm @positive-click="removePanel(panel.id)">
                  <template #trigger>
                    <NButton quaternary size="tiny" type="error">&times;</NButton>
                  </template>
                  Delete this row?
                </NPopconfirm>
              </NSpace>
            </div>
          </div>

          <!-- Normal panel -->
          <div
            v-else
            class="panel-grid-item"
            :class="{ 'panel-dragging': dragState?.panelId === panel.id && dragState?.mode === 'move', 'panel-resizing': dragState?.panelId === panel.id && dragState?.mode === 'resize' }"
            :style="{
              gridColumn: `${(panel.gridPos?.x || 0) + 1} / span ${panel.gridPos?.w || 24}`,
              gridRow: `${(panel.gridPos?.y || 0) + 1} / span ${panel.gridPos?.h || 6}`,
            }"
          >
            <div class="panel-toolbar panel-drag-handle" @mousedown="(e: MouseEvent) => startDrag(e, panel)">
              <NInput
                :value="panel.title"
                size="tiny"
                class="panel-title-input"
                @update:value="(v: string) => updatePanelTitle(panel.id, v)"
              />
              <NSpace :size="4" @mousedown.stop>
                <NButton quaternary size="tiny" @click="editPanel(panel.id)">
                  <template #icon><CreateOutline /></template>
                </NButton>
                <NButton quaternary size="tiny" @click="duplicatePanel(panel)">
                  <template #icon><CopyOutline /></template>
                </NButton>
                <NPopconfirm @positive-click="removePanel(panel.id)">
                  <template #trigger>
                    <NButton quaternary size="tiny" type="error">&times;</NButton>
                  </template>
                  Delete this panel?
                </NPopconfirm>
              </NSpace>
            </div>
            <PanelCard :panel="panel" :time-range="timeRange" />
            <div class="panel-drag-handle-resize" @mousedown="(e: MouseEvent) => startResize(e, panel)">
              <svg width="10" height="10" viewBox="0 0 10 10"><path d="M0 10 L10 0 M4 10 L10 4 M8 10 L10 8" stroke="currentColor" fill="none" opacity="0.4"/></svg>
            </div>
          </div>
        </template>
      </div>

      <!-- Empty state -->
      <div v-if="!hasPanels && !hasResults" class="empty-dashboard">
        <div class="empty-text">{{ t('dashboardV2.emptyDashboardHint') }}</div>
      </div>

      <!-- Query editor (always visible) -->
      <details class="query-editor-section" :open="!hasPanels">
        <summary class="query-editor-toggle">{{ t('dashboardV2.queryEditor') }}</summary>
        <QueryPanel
          :targets="targets"
          :datasources="datasources"
          :loading="globalLoading"
          @add="addTarget"
          @remove="removeTarget"
          @toggle="toggleTarget"
          @update="updateTarget"
          @execute="handleExecuteSingle"
          @execute-all="executeAll"
        />

        <!-- Query results + add panel buttons -->
        <div v-if="hasResults" class="query-results-section">
          <div class="results-actions">
            <span class="results-label">{{ t('dashboardV2.addAsPanel') }}</span>
            <NSpace size="small">
              <NButton size="tiny" secondary @click="addPanelFromQuery('timeseries')">{{ t('dashboardV2.panelTimeseries') }}</NButton>
              <NButton size="tiny" secondary @click="addPanelFromQuery('stat')">{{ t('dashboardV2.panelStat') }}</NButton>
              <NButton size="tiny" secondary @click="addPanelFromQuery('gauge')">{{ t('dashboardV2.panelGauge') }}</NButton>
              <NButton size="tiny" secondary @click="addPanelFromQuery('bar')">{{ t('dashboardV2.panelBar') }}</NButton>
              <NButton size="tiny" secondary @click="addPanelFromQuery('pie')">{{ t('dashboardV2.panelPie') }}</NButton>
              <NButton size="tiny" secondary @click="addPanelFromQuery('table')">{{ t('dashboardV2.panelTable') }}</NButton>
            </NSpace>
          </div>
          <QueryResultChart :targets="targets" :time-range="timeRange" :height="300" />
        </div>
      </details>
    </template>

    <!-- Variable Editor Drawer -->
    <VariableEditor
      :show="showVariableEditor"
      :variables="variableConfig"
      :datasources="datasources"
      @update:variables="(v: VariableConfig[]) => { variableConfig = v; config.variables = v }"
      @close="showVariableEditor = false"
    />

    <!-- Panel Editor Drawer -->
    <PanelEditor
      v-if="editingPanel"
      v-model:show="showPanelEditor"
      :panel="editingPanel"
      :datasources="datasources"
      :time-range="timeRange"
      :variable-options="variableList.map(v => ({ label: v.config.label || v.config.name, value: v.config.name }))"
      @save="onPanelSave"
      @cancel="editingPanel = null"
    />
  </div>
</template>

<style scoped>
.dashboard-view {
  padding: 20px;
  max-width: 1600px;
}
.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dash-name-input {
  width: 280px;
}
.var-select {
  width: 160px;
}
.panel-title-input {
  width: 180px;
}

/* Variable bar */
.variable-bar {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
  padding: 12px;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
}
.variable-bar-items {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  flex: 1;
}
.var-manage-btn {
  flex-shrink: 0;
}
.var-item {
  display: flex;
  align-items: center;
  gap: 6px;
}
.var-item label {
  font-size: 12px;
  color: var(--sre-text-secondary);
  white-space: nowrap;
}
.var-value {
  font-size: 13px;
  padding: 4px 8px;
  background: var(--sre-bg-sunken);
  border-radius: var(--sre-radius-xs);
  color: var(--sre-text-primary);
}
.adhoc-filters {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* Panel grid */
.panel-grid {
  display: grid;
  grid-template-columns: repeat(24, 1fr);
  gap: 12px;
  margin-bottom: 20px;
  min-height: 0;
}
.panel-grid-item {
  display: flex;
  flex-direction: column;
  min-height: 200px;
  position: relative;
  transition: opacity 0.15s ease;
}
.panel-grid-item.panel-dragging {
  opacity: 0.7;
  z-index: 10;
  cursor: grabbing;
}
.panel-grid-item.panel-resizing {
  opacity: 0.85;
  z-index: 10;
}
.panel-drag-handle {
  cursor: grab;
  user-select: none;
}
.panel-drag-handle:active {
  cursor: grabbing;
}
.panel-drag-handle-resize {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 16px;
  height: 16px;
  cursor: nwse-resize;
  color: var(--sre-text-tertiary);
  opacity: 0;
  transition: opacity 0.15s ease;
  display: flex;
  align-items: flex-end;
  justify-content: flex-end;
}
.panel-grid-item:hover .panel-drag-handle-resize {
  opacity: 1;
}
.panel-drag-handle-resize:hover {
  color: var(--sre-text-primary);
}
.panel-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
  padding: 0 2px;
}
.panel-row-item {
  min-height: auto;
}
.panel-row-toolbar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: var(--sre-bg-sunken);
  border: 1px solid var(--sre-border);
  border-radius: 6px;
  cursor: grab;
  user-select: none;
}
.panel-row-toolbar:active {
  cursor: grabbing;
}
.panel-row-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  flex: 1;
}

/* Empty dashboard */
.empty-dashboard {
  padding: 60px 0;
  text-align: center;
}
.empty-text {
  font-size: 14px;
  color: var(--sre-text-tertiary);
}

/* Query editor */
.query-editor-section {
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
  padding: 12px 16px;
  background: var(--sre-bg-sunken);
}
.query-editor-toggle {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  cursor: pointer;
  user-select: none;
}
.query-editor-toggle:hover {
  color: var(--sre-text-primary);
}
.query-results-section {
  margin-top: 12px;
  border-top: var(--sre-hairline);
  padding-top: 12px;
}
.results-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.results-label {
  font-size: 12px;
  color: var(--sre-text-secondary);
}
</style>
