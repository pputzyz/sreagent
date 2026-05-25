<script setup lang="ts">
import { ref, onMounted, watch, computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import { NButton, NSpace, NIcon } from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import { builtinMetricApi, type BuiltinMetric } from '@/api/builtin-metric'
import { useFilterMemory, usePermissions } from '@/composables'
import PageHeader from '@/components/common/PageHeader.vue'

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
filterMemory.bindRefs({ query: searchQuery, collector: filterCollector, typ: filterTyp })

// Metadata
const collectorOptions = ref<string[]>([])
const typeOptions = ref<string[]>([])

// Edit drawer
const showDrawer = ref(false)
const drawerMode = ref<'create' | 'edit'>('create')
const drawerMetric = ref<Partial<BuiltinMetric>>({})

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
    metrics.value = resp.data.data?.list || []
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
        onClick: () => openEdit(row),
      }, row.name),
  },
  {
    title: t('builtin.collector'),
    key: 'collector',
    width: 140,
  },
  {
    title: t('builtin.type'),
    key: 'typ',
    width: 120,
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
  },
  {
    title: t('builtin.metricType'),
    key: 'metric_type',
    width: 100,
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

watch([searchQuery, filterCollector, filterTyp], () => {
  page.value = 1
  fetchMetrics()
  fetchMetadata()
})

onMounted(() => {
  fetchMetrics()
  fetchMetadata()
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
          style="width: 260px;"
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
          v-model:value="filterTyp"
          :placeholder="t('builtin.type')"
          :options="typeOptions.map(t => ({ label: t, value: t }))"
          clearable
          filterable
          size="small"
          style="width: 140px;"
        />
      </div>
      <div class="toolbar-right" v-if="canWrite">
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
      </div>
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
  margin-bottom: 16px;
  gap: 12px;
}
.toolbar-left {
  display: flex;
  gap: 8px;
  align-items: center;
}
.toolbar-right {
  display: flex;
  gap: 8px;
  align-items: center;
}
.page-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
