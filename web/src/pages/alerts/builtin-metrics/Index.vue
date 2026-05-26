<script setup lang="ts">
import { ref, onMounted, watch, computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import { NButton, NSpace, NIcon, NTag } from 'naive-ui'
import { AddOutline, SearchOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import { builtinMetricApi, metricFilterApi, type BuiltinMetric, type MetricFilter, type FilterConfig } from '@/api/builtin-metric'
import { useFilterMemory, usePermissions } from '@/composables'
import PageHeader from '@/components/common/PageHeader.vue'
import PromQLEditor from '@/components/query/PromQLEditor.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const { hasPerm } = usePermissions()

// State
const metrics = ref<BuiltinMetric[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const selectedIds = ref<(string | number)[]>([])

// Filters
const filterMemory = useFilterMemory('builtin-metrics')
const searchQuery = ref(filterMemory.restore('query', ''))
const filterCollector = ref<string | null>(filterMemory.restore('collector', null))
const filterTyp = ref<string | null>(filterMemory.restore('typ', null))
const filterUnit = ref<string[]>(filterMemory.restore('unit', []))
filterMemory.bindRefs({ query: searchQuery, collector: filterCollector, typ: filterTyp, unit: filterUnit })

// Metadata
const collectorOptions = ref<string[]>([])
const typeOptions = ref<string[]>([])
const unitOptions = ref<string[]>([])

// Edit drawer
const showDrawer = ref(false)
const drawerMode = ref<'create' | 'edit'>('create')
const drawerMetric = ref<Partial<BuiltinMetric>>({})

// Explorer drawer (Nightingale pattern: click metric name to explore)
const showExplorerDrawer = ref(false)
const explorerMetric = ref<BuiltinMetric | null>(null)
const explorerPromql = ref('')

// MetricFilter management
const showFilterModal = ref(false)
const savedFilters = ref<MetricFilter[]>([])
const editingFilter = ref<Partial<MetricFilter> & { configs: FilterConfig[] }>({ configs: [] })
const filterModalMode = ref<'create' | 'edit'>('create')

// Active filter (applied to metric expressions)
const activeFilter = ref<MetricFilter | null>(null)

const canWrite = computed(() => hasPerm('metrics.write'))

async function fetchMetrics() {
  loading.value = true
  try {
    const resp = await builtinMetricApi.list({
      page: page.value,
      page_size: pageSize.value,
      collector: filterCollector.value || undefined,
      typ: filterTyp.value || undefined,
      query: searchQuery.value || undefined,
    })
    let list = resp.data.data?.list || []
    // Client-side unit filter
    if (filterUnit.value.length > 0) {
      list = list.filter((m) => filterUnit.value.includes(m.unit))
    }
    metrics.value = list
    total.value = resp.data.data?.total || 0
  } catch (e: any) {
    message.error(e.message || 'Failed to load metrics')
  } finally {
    loading.value = false
  }
}

async function fetchMetadata() {
  try {
    const [typesResp, collectorsResp] = await Promise.all([
      builtinMetricApi.getTypes({ collector: filterCollector.value || undefined }),
      builtinMetricApi.getCollectors({ typ: filterTyp.value || undefined }),
    ])
    typeOptions.value = typesResp.data.data || []
    collectorOptions.value = collectorsResp.data.data || []

    // Build unit options from current metrics
    const units = new Set<string>()
    metrics.value.forEach((m) => { if (m.unit) units.add(m.unit) })
    unitOptions.value = Array.from(units).sort()
  } catch {}
}

async function fetchFilters() {
  try {
    const resp = await metricFilterApi.list()
    savedFilters.value = resp.data.data || []
  } catch {}
}

function openCreate() {
  drawerMode.value = 'create'
  drawerMetric.value = {
    expression_type: 'metric_name',
    lang: 'zh',
  }
  showDrawer.value = true
}

function openEdit(metric: BuiltinMetric) {
  drawerMode.value = 'edit'
  drawerMetric.value = { ...metric }
  showDrawer.value = true
}

// Nightingale pattern: click metric name to open explorer drawer
function openExplorer(metric: BuiltinMetric) {
  explorerMetric.value = metric
  let promql = metric.expression || metric.name
  // If expression_type is metric_name, wrap as simple metric query
  if (metric.expression_type === 'metric_name') {
    promql = metric.name
    // If there's an active filter, inject label matchers
    if (activeFilter.value?.configs && activeFilter.value.configs.length > 0) {
      const matchers = activeFilter.value.configs
        .filter((c) => c.label && c.value)
        .map((c) => `${c.label}${c.operator}"${c.value}"`)
        .join(', ')
      if (matchers) {
        promql = `${metric.name}{${matchers}}`
      }
    }
  }
  explorerPromql.value = promql
  showExplorerDrawer.value = true
}

async function handleSave() {
  const m = drawerMetric.value
  if (!m.name || !m.expression) {
    message.warning(t('builtin.nameAndExprRequired'))
    return
  }
  try {
    if (drawerMode.value === 'edit' && m.id) {
      await builtinMetricApi.update(m)
      message.success(t('common.savedSuccess'))
    } else {
      await builtinMetricApi.create(m)
      message.success(t('common.createSuccess'))
    }
    showDrawer.value = false
    fetchMetrics()
  } catch (e: any) {
    message.error(e.message || t('common.saveFailed'))
  }
}

async function handleDelete(metric: BuiltinMetric) {
  dialog.warning({
    title: t('common.confirm'),
    content: t('builtin.confirmDelete', { name: metric.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await builtinMetricApi.delete([metric.id])
        message.success(t('common.deleteSuccess'))
        fetchMetrics()
      } catch (e: any) {
        message.error(e.message || t('common.deleteFailed'))
      }
    },
  })
}

async function handleBatchDelete() {
  if (!selectedIds.value.length) return
  const ids = selectedIds.value.map((id) => Number(id))
  dialog.warning({
    title: t('common.confirm'),
    content: t('builtin.confirmBatchDelete', { count: ids.length }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await builtinMetricApi.delete(ids)
        message.success(t('common.deleteSuccess'))
        selectedIds.value = []
        fetchMetrics()
      } catch (e: any) {
        message.error(e.message || t('common.deleteFailed'))
      }
    },
  })
}

function handleExport() {
  const data = selectedIds.value.length
    ? metrics.value.filter((m) => selectedIds.value.includes(m.id))
    : metrics.value
  const exportData = data.map((m) => ({
    collector: m.collector,
    typ: m.typ,
    name: m.name,
    unit: m.unit,
    note: m.note,
    expression: m.expression,
    expression_type: m.expression_type,
    metric_type: m.metric_type,
    extra_fields: m.extra_fields,
  }))
  const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'builtin-metrics.json'
  a.click()
  URL.revokeObjectURL(url)
}

// MetricFilter CRUD
function openCreateFilter() {
  filterModalMode.value = 'create'
  editingFilter.value = { configs: [] }
  showFilterModal.value = true
}

function openEditFilter(filter: MetricFilter) {
  filterModalMode.value = 'edit'
  editingFilter.value = { ...filter, configs: [...(filter.configs || [])] }
  showFilterModal.value = true
}

async function handleSaveFilter() {
  const f = editingFilter.value
  if (!f.name) {
    message.warning(t('builtin.filterNameRequired'))
    return
  }
  try {
    if (filterModalMode.value === 'edit' && f.id) {
      await metricFilterApi.update(f as MetricFilter)
      message.success(t('common.savedSuccess'))
    } else {
      await metricFilterApi.create(f)
      message.success(t('common.createSuccess'))
    }
    showFilterModal.value = false
    fetchFilters()
  } catch (e: any) {
    message.error(e.message || t('common.saveFailed'))
  }
}

async function handleDeleteFilter(filter: MetricFilter) {
  dialog.warning({
    title: t('common.confirm'),
    content: t('builtin.confirmDeleteFilter', { name: filter.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await metricFilterApi.delete([filter.id])
        message.success(t('common.deleteSuccess'))
        if (activeFilter.value?.id === filter.id) activeFilter.value = null
        fetchFilters()
      } catch (e: any) {
        message.error(e.message || t('common.deleteFailed'))
      }
    },
  })
}

function applyFilter(filter: MetricFilter | null) {
  activeFilter.value = filter
}

// Get filter configs (already parsed by API)
function getFilterConfigs(): FilterConfig[] {
  return editingFilter.value.configs || []
}

const columns = computed<DataTableColumns<BuiltinMetric>>(() => [
  { type: 'selection' },
  {
    title: t('builtin.name'),
    key: 'name',
    minWidth: 180,
    ellipsis: { tooltip: true },
    render: (row) =>
      h('a', {
        style: 'color: var(--sre-primary); cursor: pointer; text-decoration: none;',
        onClick: () => openExplorer(row),
      }, row.name),
  },
  {
    title: t('builtin.collector'),
    key: 'collector',
    width: 140,
    render: (row) => row.collector
      ? h(NTag, { size: 'small', bordered: false, type: 'info' }, () => row.collector)
      : '-',
  },
  {
    title: t('builtin.type'),
    key: 'typ',
    width: 120,
    render: (row) => row.typ
      ? h(NTag, { size: 'small', bordered: false, type: 'warning' }, () => row.typ)
      : '-',
  },
  {
    title: t('builtin.unit'),
    key: 'unit',
    width: 80,
  },
  {
    title: t('builtin.expression'),
    key: 'expression',
    minWidth: 200,
    ellipsis: { tooltip: true },
    render: (row) =>
      h('code', {
        style: 'font-size: 12px; background: var(--n-code-color, rgba(0,0,0,0.05)); padding: 1px 4px; border-radius: 2px;',
      }, row.expression),
  },
  {
    title: t('builtin.metricType'),
    key: 'metric_type',
    width: 100,
    render: (row) => row.metric_type
      ? h(NTag, { size: 'small', bordered: false }, () => row.metric_type)
      : '-',
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 140,
    render: (row) =>
      h(NSpace, { size: 'small' }, () => [
        h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => openEdit(row) }, () => t('common.edit')),
        h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleDelete(row) }, () => t('common.delete')),
      ]),
  },
])

watch([searchQuery, filterCollector, filterTyp, filterUnit], () => {
  page.value = 1
  fetchMetrics()
  fetchMetadata()
})

onMounted(() => {
  fetchMetrics()
  fetchMetadata()
  fetchFilters()
})
</script>

<template>
  <div class="builtin-metrics-page">
    <PageHeader :title="t('menu.builtinMetrics')" />

    <div class="page-toolbar">
      <div class="toolbar-left">
        <NInput
          v-model:value="searchQuery"
          :placeholder="t('builtin.searchPlaceholder')"
          clearable
          size="small"
          style="width: 240px;"
        />
        <NSelect
          v-model:value="filterTyp"
          :placeholder="t('builtin.type')"
          :options="typeOptions.map(t => ({ label: t, value: t }))"
          clearable
          filterable
          size="small"
          style="width: 140px;"
        />
        <NSelect
          v-model:value="filterCollector"
          :placeholder="t('builtin.collector')"
          :options="collectorOptions.map(c => ({ label: c, value: c }))"
          clearable
          filterable
          size="small"
          style="width: 160px;"
        />
        <NSelect
          v-model:value="filterUnit"
          :placeholder="t('builtin.unit')"
          :options="unitOptions.map(u => ({ label: u, value: u }))"
          multiple
          clearable
          filterable
          size="small"
          style="width: 160px;"
        />
        <!-- Active filter indicator -->
        <NTag
          v-if="activeFilter"
          closable
          size="small"
          type="success"
          @close="applyFilter(null)"
        >
          {{ activeFilter.name }}
        </NTag>
      </div>
      <div class="toolbar-right">
        <NButton size="small" @click="openCreateFilter">
          <template #icon><NIcon><SearchOutline /></NIcon></template>
          {{ t('builtin.manageFilters') }}
        </NButton>
        <template v-if="canWrite">
          <NButton size="small" type="primary" @click="openCreate">
            <template #icon><NIcon><AddOutline /></NIcon></template>
            {{ t('builtin.create') }}
          </NButton>
          <NButton size="small" @click="handleExport">
            {{ t('builtin.export') }}
          </NButton>
          <NButton
            v-if="selectedIds.length"
            size="small"
            type="error"
            @click="handleBatchDelete"
          >
            {{ t('common.delete') }} ({{ selectedIds.length }})
          </NButton>
        </template>
      </div>
    </div>

    <!-- Saved Filters Bar -->
    <div v-if="savedFilters.length" class="filter-bar">
      <span class="filter-bar-label">{{ t('builtin.filters') }}:</span>
      <NTag
        v-for="f in savedFilters"
        :key="f.id"
        :type="activeFilter?.id === f.id ? 'success' : 'default'"
        size="small"
        checkable
        :checked="activeFilter?.id === f.id"
        @update:checked="(v: boolean) => applyFilter(v ? f : null)"
        closable
        @close="handleDeleteFilter(f)"
      >
        {{ f.name }}
      </NTag>
      <NButton size="tiny" quaternary @click="openCreateFilter">+</NButton>
    </div>

    <NDataTable
      :columns="columns"
      :data="metrics"
      :loading="loading"
      :row-key="(row: BuiltinMetric) => row.id"
      :checked-row-keys="selectedIds"
      @update:checked-row-keys="(keys: (string | number)[]) => selectedIds = keys"
      size="small"
      :bordered="false"
      striped
    />

    <div class="page-pagination" v-if="total > 0">
      <NPagination
        v-model:page="page"
        v-model:page-size="pageSize"
        :item-count="total"
        :page-sizes="[20, 50, 100]"
        show-size-picker
      />
    </div>

    <!-- Edit Drawer -->
    <NDrawer v-model:show="showDrawer" :width="600">
      <NDrawerContent :title="drawerMode === 'edit' ? t('builtin.editMetric') : t('builtin.createMetric')">
        <NForm label-placement="left" label-width="100px">
          <NFormItem :label="t('builtin.name')" required>
            <NInput v-model:value="drawerMetric.name" :placeholder="t('builtin.namePlaceholder')" />
          </NFormItem>
          <NFormItem :label="t('builtin.collector')">
            <NInput v-model:value="drawerMetric.collector" placeholder="node_exporter" />
          </NFormItem>
          <NFormItem :label="t('builtin.type')">
            <NInput v-model:value="drawerMetric.typ" placeholder="Linux" />
          </NFormItem>
          <NFormItem :label="t('builtin.expression')" required>
            <NInput v-model:value="drawerMetric.expression" placeholder="cpu_usage_percent" />
          </NFormItem>
          <NFormItem :label="t('builtin.expressionType')">
            <NRadioGroup v-model:value="drawerMetric.expression_type">
              <NRadioButton value="metric_name">Metric Name</NRadioButton>
              <NRadioButton value="promql">PromQL</NRadioButton>
            </NRadioGroup>
          </NFormItem>
          <NFormItem :label="t('builtin.metricType')">
            <NSelect
              v-model:value="drawerMetric.metric_type"
              :options="[
                { label: 'Gauge', value: 'gauge' },
                { label: 'Counter', value: 'counter' },
                { label: 'Histogram', value: 'histogram' },
              ]"
              clearable
            />
          </NFormItem>
          <NFormItem :label="t('builtin.unit')">
            <NInput v-model:value="drawerMetric.unit" placeholder="%, req/s, bytes" />
          </NFormItem>
          <NFormItem :label="t('builtin.note')">
            <NInput v-model:value="drawerMetric.note" type="textarea" :autosize="{ minRows: 2, maxRows: 6 }" />
          </NFormItem>
        </NForm>
        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 8px;">
            <NButton @click="showDrawer = false">{{ t('common.cancel') }}</NButton>
            <NButton type="primary" @click="handleSave">{{ t('common.save') }}</NButton>
          </div>
        </template>
      </NDrawerContent>
    </NDrawer>

    <!-- Explorer Drawer (Nightingale pattern: click metric to explore) -->
    <NDrawer v-model:show="showExplorerDrawer" :width="800">
      <NDrawerContent :title="explorerMetric ? `${t('builtin.explore')}: ${explorerMetric.name}` : t('builtin.explore')">
        <div v-if="explorerMetric" class="explorer-drawer">
          <div class="explorer-info">
            <NTag size="small" type="info" bordered>{{ explorerMetric.collector }}</NTag>
            <NTag size="small" type="warning" bordered>{{ explorerMetric.typ }}</NTag>
            <span v-if="explorerMetric.unit" style="font-size: 12px; color: var(--n-text-color-3);">{{ explorerMetric.unit }}</span>
          </div>
          <div v-if="explorerMetric.note" class="explorer-note">{{ explorerMetric.note }}</div>
          <PromQLEditor
            :model-value="explorerPromql"
            :datasource-id="null"
            placeholder="PromQL expression..."
            style="width: 100%; min-height: 100px; border: 1px solid var(--n-border-color); border-radius: 3px;"
            @update:model-value="explorerPromql = $event"
          />
        </div>
        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 8px;">
            <NButton @click="showExplorerDrawer = false">{{ t('common.close') }}</NButton>
            <NButton type="primary" @click="explorerMetric && openEdit(explorerMetric); showExplorerDrawer = false">
              {{ t('common.edit') }}
            </NButton>
          </div>
        </template>
      </NDrawerContent>
    </NDrawer>

    <!-- MetricFilter Modal -->
    <NModal
      v-model:show="showFilterModal"
      preset="card"
      :title="filterModalMode === 'edit' ? t('builtin.editFilter') : t('builtin.createFilter')"
      style="width: 600px;"
    >
      <NForm label-placement="left" label-width="100px">
        <NFormItem :label="t('builtin.filterName')" required>
          <NInput v-model:value="editingFilter.name" :placeholder="t('builtin.filterNamePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('builtin.filterConditions')">
          <div style="width: 100%;">
            <div
              v-for="(cfg, idx) in editingFilter.configs"
              :key="idx"
              style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center;"
            >
              <NInput v-model:value="cfg.label" size="small" placeholder="label" style="width: 120px;" />
              <NSelect
                v-model:value="cfg.operator"
                size="small"
                style="width: 70px;"
                :options="[
                  { label: '=', value: '=' },
                  { label: '!=', value: '!=' },
                  { label: '=~', value: '=~' },
                  { label: '!~', value: '!~' },
                ]"
              />
              <NInput v-model:value="cfg.value" size="small" placeholder="value" style="flex: 1;" />
              <NButton size="tiny" quaternary type="error" @click="editingFilter.configs!.splice(idx, 1)">-</NButton>
            </div>
            <NButton size="small" dashed @click="editingFilter.configs!.push({ label: '', operator: '=', value: '' })">+ {{ t('builtin.addCondition') }}</NButton>
          </div>
        </NFormItem>
      </NForm>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 8px;">
          <NButton @click="showFilterModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" @click="handleSaveFilter">{{ t('common.save') }}</NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.builtin-metrics-page {
  padding: 16px;
}
.page-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  gap: 12px;
}
.toolbar-left {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.toolbar-right {
  display: flex;
  gap: 8px;
  align-items: center;
}
.filter-bar {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.filter-bar-label {
  font-size: 12px;
  color: var(--n-text-color-3);
  margin-right: 4px;
}
.page-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.explorer-drawer {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.explorer-info {
  display: flex;
  gap: 8px;
  align-items: center;
}
.explorer-note {
  font-size: 13px;
  color: var(--n-text-color-3);
  padding: 8px 12px;
  background: var(--n-code-color, rgba(0,0,0,0.03));
  border-radius: 4px;
}
</style>
