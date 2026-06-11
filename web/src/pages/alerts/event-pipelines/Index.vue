<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import {
  NButton, NSpace, NDataTable, NInput, NSelect, NDrawer, NDrawerContent,
  NForm, NFormItem, NTag, NSwitch, NPopconfirm, NIcon, NDivider,
  NInputNumber, NCheckbox, NEmpty, NScrollbar, NPagination,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { AddOutline, TrashOutline, PlayOutline, TimeOutline } from '@vicons/ionicons5'
import { eventPipelineApi } from '@/api/event-pipeline'
import type { EventPipeline, ProcessorConfig, TagFilter, EventPipelineExecution, NodeResult } from '@/api/event-pipeline'
import { useFilterMemory, usePermissions } from '@/composables'
import { getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const { hasPerm } = usePermissions()

// Filter memory
const filterMemory = useFilterMemory('event-pipelines')
const searchQuery = ref(filterMemory.restore('query', ''))
const filterDisabled = ref<string | null>(filterMemory.restore('disabled', null))
filterMemory.bindRefs({ query: searchQuery, disabled: filterDisabled })

// Data
const list = ref<EventPipeline[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)

// Drawer
const showDrawer = ref(false)
const drawerMode = ref<'create' | 'edit'>('create')
const editingId = ref<number | null>(null)
const saving = ref(false)
const form = ref({
  name: '',
  description: '',
  disabled: false,
  filter_enable: false,
  label_filters: [] as TagFilter[],
  processors: [] as ProcessorConfig[],
})

// Execution history
const showExecDrawer = ref(false)
const execList = ref<EventPipelineExecution[]>([])
const execTotal = ref(0)
const execPage = ref(1)
const execPageSize = ref(20)
const execLoading = ref(false)
const execPipelineName = ref('')

// Processor types available
const processorTypes = ref<string[]>([])

const canWrite = computed(() => hasPerm('rules.write'))

let _nextRowId = 0
const _rowIdMap = new WeakMap<object, number>()
function rowId(row: object): number {
  let id = _rowIdMap.get(row)
  if (id === undefined) { id = ++_nextRowId; _rowIdMap.set(row, id) }
  return id
}

const filterOptions = computed(() => [
  { label: t('common.all'), value: '' },
  { label: t('common.enabled'), value: 'false' },
  { label: t('common.disabled'), value: 'true' },
])

const processorTypeOptions = computed(() =>
  processorTypes.value.map((pt) => ({
    label: t(`eventPipeline.${pt}`, pt),
    value: pt,
  }))
)

const filterFuncOptions = [
  { label: '==', value: '==' },
  { label: '=~', value: '=~' },
  { label: 'in', value: 'in' },
  { label: '!=', value: '!=' },
  { label: '!~', value: '!~' },
  { label: 'not in', value: 'not in' },
]

const relabelActionOptions = [
  { label: 'replace', value: 'replace' },
  { label: 'keep', value: 'keep' },
  { label: 'drop', value: 'drop' },
  { label: 'labelmap', value: 'labelmap' },
  { label: 'hashmod', value: 'hashmod' },
]

const methodOptions = [
  { label: 'POST', value: 'POST' },
  { label: 'GET', value: 'GET' },
  { label: 'PUT', value: 'PUT' },
]

// Fetch list
async function fetchList() {
  loading.value = true
  try {
    const resp = await eventPipelineApi.list({
      page: page.value,
      page_size: pageSize.value,
      query: searchQuery.value || undefined,
      disabled: filterDisabled.value || undefined,
    })
    list.value = resp.data.data?.list || []
    total.value = resp.data.data?.total || 0
  } catch (e: any) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

async function fetchProcessorTypes() {
  try {
    const resp = await eventPipelineApi.listProcessorTypes()
    processorTypes.value = resp.data.data || []
  } catch {
    // fallback defaults
    processorTypes.value = ['relabel', 'callback', 'event_drop', 'ai_summary']
  }
}

function handlePageChange(p: number) {
  page.value = p
  fetchList()
}

function handlePageSizeChange(ps: number) {
  pageSize.value = ps
  page.value = 1
  fetchList()
}

function openCreate() {
  drawerMode.value = 'create'
  editingId.value = null
  resetForm()
  showDrawer.value = true
}

function openEdit(row: EventPipeline) {
  drawerMode.value = 'edit'
  editingId.value = row.id
  form.value = {
    name: row.name,
    description: row.description || '',
    disabled: row.disabled,
    filter_enable: row.filter_enable,
    label_filters: row.label_filters ? JSON.parse(JSON.stringify(row.label_filters)) : [],
    processors: row.processors ? JSON.parse(JSON.stringify(row.processors)) : [],
  }
  showDrawer.value = true
}

function resetForm() {
  form.value = {
    name: '',
    description: '',
    disabled: false,
    filter_enable: false,
    label_filters: [],
    processors: [],
  }
}

async function handleSave() {
  if (!form.value.name.trim()) {
    message.warning(t('eventPipeline.nameRequired'))
    return
  }
  saving.value = true
  try {
    const payload = { ...form.value }
    if (drawerMode.value === 'edit' && editingId.value) {
      await eventPipelineApi.update(editingId.value, payload)
      message.success(t('common.savedSuccess'))
    } else {
      await eventPipelineApi.create(payload)
      message.success(t('common.createSuccess'))
    }
    showDrawer.value = false
    fetchList()
  } catch (e: any) {
    message.error(getErrorMessage(e))
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  dialog.warning({
    title: t('common.confirm'),
    content: t('eventPipeline.confirmDelete'),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await eventPipelineApi.delete(id)
        message.success(t('common.deleteSuccess'))
        fetchList()
      } catch (e: any) {
        message.error(getErrorMessage(e))
      }
    },
  })
}

async function handleTryRun(id: number) {
  try {
    const resp = await eventPipelineApi.tryRun(id)
    const result = resp.data.data
    if (result) {
      message.success(t('eventPipeline.tryRunSuccess'))
    } else {
      message.info(t('eventPipeline.tryRunNoEvent'))
    }
  } catch (e: any) {
    message.error(getErrorMessage(e))
  }
}

async function openExecHistory(row: EventPipeline) {
  execPipelineName.value = row.name
  execList.value = []
  execTotal.value = 0
  execPage.value = 1
  showExecDrawer.value = true
  await fetchExecutions(row.id)
}

async function fetchExecutions(pipelineId: number) {
  execLoading.value = true
  try {
    const resp = await eventPipelineApi.listExecutions(pipelineId, {
      page: execPage.value,
      page_size: execPageSize.value,
    })
    execList.value = resp.data.data?.list || []
    execTotal.value = resp.data.data?.total || 0
  } catch (e: any) {
    message.error(getErrorMessage(e))
  } finally {
    execLoading.value = false
  }
}

function handleExecPageChange(p: number) {
  execPage.value = p
  // We need to know which pipeline to fetch for; store it
  const row = list.value.find((r) => r.name === execPipelineName.value)
  if (row) fetchExecutions(row.id)
}

// Processor helpers
function addProcessor() {
  form.value.processors.push({ typ: 'relabel', config: {} })
}

function removeProcessor(index: number) {
  form.value.processors.splice(index, 1)
}

function getProcessorDefaultConfig(typ: string): Record<string, any> {
  switch (typ) {
    case 'relabel':
      return { source_labels: [], separator: '', regex: '', target_label: '', replacement: '', action: 'replace' }
    case 'callback':
      return { url: '', method: 'POST', headers: {}, timeout: 10, skip_ssl_verify: false }
    case 'event_drop':
      return { condition: '' }
    case 'ai_summary':
      return { only_critical: false }
    default:
      return {}
  }
}

function onProcessorTypeChange(index: number, typ: string) {
  form.value.processors[index].typ = typ
  form.value.processors[index].config = getProcessorDefaultConfig(typ)
}

// Label filter helpers
function addLabelFilter() {
  form.value.label_filters.push({ key: '', func: '==', value: '' })
}

function removeLabelFilter(index: number) {
  form.value.label_filters.splice(index, 1)
}

// Parse node_results JSON for display
function parseNodeResults(jsonStr: string): NodeResult[] {
  try {
    return JSON.parse(jsonStr)
  } catch {
    return []
  }
}

// Columns using h() function
const columns = computed<DataTableColumns<EventPipeline>>(() => [
  {
    title: t('eventPipeline.name'),
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
    title: t('eventPipeline.description'),
    key: 'description',
    minWidth: 200,
    ellipsis: { tooltip: true },
    render: (row) => row.description || '-',
  },
  {
    title: t('eventPipeline.disabled'),
    key: 'disabled',
    width: 90,
    render: (row) =>
      h(NTag, {
        type: row.disabled ? 'warning' : 'success',
        size: 'small',
        bordered: false,
      }, () => row.disabled ? t('common.disabled') : t('common.enabled')),
  },
  {
    title: t('eventPipeline.processorCount'),
    key: 'processors',
    width: 110,
    render: (row) => row.processors?.length || 0,
  },
  {
    title: t('common.createdAt'),
    key: 'created_at',
    width: 170,
    render: (row) => row.created_at ? new Date(row.created_at).toLocaleString() : '-',
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 260,
    render: (row) =>
      h(NSpace, { size: 'small' }, () => [
        h(NButton, {
          size: 'tiny',
          quaternary: true,
          type: 'primary',
          onClick: () => openEdit(row),
        }, () => t('common.edit')),
        h(NButton, {
          size: 'tiny',
          quaternary: true,
          onClick: () => openExecHistory(row),
        }, {
          default: () => t('eventPipeline.history'),
          icon: () => h(NIcon, null, () => h(TimeOutline)),
        }),
        h(NButton, {
          size: 'tiny',
          quaternary: true,
          type: 'info',
          onClick: () => handleTryRun(row.id),
        }, {
          default: () => t('eventPipeline.tryRun'),
          icon: () => h(NIcon, null, () => h(PlayOutline)),
        }),
        h(NPopconfirm, {
          onPositiveClick: () => handleDelete(row.id),
        }, {
          trigger: () => h(NButton, {
            size: 'tiny',
            quaternary: true,
            type: 'error',
          }, () => t('common.delete')),
          default: () => t('eventPipeline.confirmDelete'),
        }),
      ]),
  },
])

// Execution table columns
const execColumns = computed<DataTableColumns<EventPipelineExecution>>(() => [
  {
    title: t('eventPipeline.executionId'),
    key: 'id',
    width: 200,
    ellipsis: { tooltip: true },
  },
  {
    title: t('common.status'),
    key: 'status',
    width: 100,
    render: (row) => {
      const typeMap: Record<string, string> = {
        success: 'success',
        failed: 'error',
        running: 'info',
      }
      return h(NTag, {
        type: (typeMap[row.status] || 'default') as any,
        size: 'small',
        bordered: false,
      }, () => row.status)
    },
  },
  {
    title: t('eventPipeline.duration'),
    key: 'duration_ms',
    width: 100,
    render: (row) => row.duration_ms ? `${row.duration_ms}ms` : '-',
  },
  {
    title: t('eventPipeline.triggerBy'),
    key: 'trigger_by',
    width: 120,
  },
  {
    title: t('common.createdAt'),
    key: 'created_at',
    width: 170,
    render: (row) => row.created_at ? new Date(row.created_at).toLocaleString() : '-',
  },
])

// Watch filters
import { watch } from 'vue'
watch([searchQuery, filterDisabled], () => {
  page.value = 1
  fetchList()
})

onMounted(() => {
  fetchList()
  fetchProcessorTypes()
})
</script>

<template>
  <div class="event-pipelines-page">
    <PageHeader :title="t('menu.eventPipelines')" />

    <div class="page-toolbar">
      <div class="toolbar-left">
        <NInput
          v-model:value="searchQuery"
          :placeholder="t('common.search')"
          clearable
          size="small"
          style="width: 260px;"
        />
        <NSelect
          v-model:value="filterDisabled"
          :options="filterOptions"
          clearable
          size="small"
          style="width: 140px;"
          :placeholder="t('common.status')"
        />
      </div>
      <div class="toolbar-right" v-if="canWrite">
        <NButton size="small" type="primary" @click="openCreate">
          <template #icon><NIcon><AddOutline /></NIcon></template>
          {{ t('common.create') }}
        </NButton>
      </div>
    </div>

    <NDataTable
      :columns="columns"
      :data="list"
      :loading="loading"
      :row-key="(row: EventPipeline) => row.id"
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
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </div>

    <!-- Create/Edit Drawer -->
    <NDrawer v-model:show="showDrawer" :width="680" placement="right">
      <NDrawerContent :title="drawerMode === 'edit' ? t('common.edit') : t('common.create')">
        <NForm label-placement="left" label-width="120px">
          <NFormItem :label="t('eventPipeline.name')" required>
            <NInput v-model:value="form.name" :placeholder="t('eventPipeline.namePlaceholder')" />
          </NFormItem>
          <NFormItem :label="t('eventPipeline.description')">
            <NInput v-model:value="form.description" type="textarea" :placeholder="t('eventPipeline.descriptionPlaceholder')" :autosize="{ minRows: 2, maxRows: 4 }" />
          </NFormItem>
          <NFormItem :label="t('eventPipeline.disabled')">
            <NSwitch v-model:value="form.disabled" />
          </NFormItem>
          <NFormItem :label="t('eventPipeline.filterEnable')">
            <NSwitch v-model:value="form.filter_enable" />
          </NFormItem>

          <!-- Label Filters -->
          <NDivider />
          <div class="section-header">
            <span>{{ t('eventPipeline.labelFilters') }}</span>
            <NButton size="tiny" quaternary type="primary" @click="addLabelFilter">
              <template #icon><NIcon><AddOutline /></NIcon></template>
              {{ t('eventPipeline.addFilter') }}
            </NButton>
          </div>
          <div v-if="form.label_filters.length === 0" class="empty-hint">
            {{ t('common.noData') }}
          </div>
          <div v-for="(filter, idx) in form.label_filters" :key="rowId(filter)" class="filter-row">
            <NInput v-model:value="filter.key" :placeholder="t('eventPipeline.filterKey')" size="small" style="width: 140px;" />
            <NSelect v-model:value="filter.func" :options="filterFuncOptions" size="small" style="width: 100px;" />
            <NInput v-model:value="filter.value" :placeholder="t('eventPipeline.filterValue')" size="small" style="flex: 1;" />
            <NButton size="tiny" quaternary type="error" @click="removeLabelFilter(idx)">
              <template #icon><NIcon><TrashOutline /></NIcon></template>
            </NButton>
          </div>

          <!-- Processors -->
          <NDivider />
          <div class="section-header">
            <span>{{ t('eventPipeline.processorConfigs') }}</span>
            <NButton size="tiny" quaternary type="primary" @click="addProcessor">
              <template #icon><NIcon><AddOutline /></NIcon></template>
              {{ t('eventPipeline.addProcessor') }}
            </NButton>
          </div>
          <div v-if="form.processors.length === 0" class="empty-hint">
            {{ t('common.noData') }}
          </div>
          <div v-for="(proc, idx) in form.processors" :key="rowId(proc)" class="processor-card">
            <div class="processor-header">
              <NSelect
                :value="proc.typ"
                :options="processorTypeOptions"
                size="small"
                style="width: 180px;"
                @update:value="(v: string) => onProcessorTypeChange(idx, v)"
              />
              <NButton size="tiny" quaternary type="error" @click="removeProcessor(idx)">
                <template #icon><NIcon><TrashOutline /></NIcon></template>
              </NButton>
            </div>

            <!-- Relabel config -->
            <div v-if="proc.typ === 'relabel'" class="processor-fields">
              <NFormItem :label="t('eventPipeline.sourceLabels')" label-width="100px" size="small">
                <NInput
                  :value="(proc.config.source_labels || []).join(',')"
                  size="small"
                  placeholder="label1,label2"
                  @update:value="(v: string) => proc.config.source_labels = v.split(',').map((s: string) => s.trim()).filter(Boolean)"
                />
              </NFormItem>
              <NFormItem :label="t('eventPipeline.separator')" label-width="100px" size="small">
                <NInput v-model:value="proc.config.separator" size="small" placeholder=";" />
              </NFormItem>
              <NFormItem :label="t('eventPipeline.regex')" label-width="100px" size="small">
                <NInput v-model:value="proc.config.regex" size="small" placeholder="(.*)" />
              </NFormItem>
              <NFormItem :label="t('eventPipeline.targetLabel')" label-width="100px" size="small">
                <NInput v-model:value="proc.config.target_label" size="small" placeholder="new_label" />
              </NFormItem>
              <NFormItem :label="t('eventPipeline.replacement')" label-width="100px" size="small">
                <NInput v-model:value="proc.config.replacement" size="small" placeholder="$1" />
              </NFormItem>
              <NFormItem :label="t('eventPipeline.action')" label-width="100px" size="small">
                <NSelect v-model:value="proc.config.action" :options="relabelActionOptions" size="small" />
              </NFormItem>
            </div>

            <!-- Callback config -->
            <div v-if="proc.typ === 'callback'" class="processor-fields">
              <NFormItem :label="t('eventPipeline.url')" label-width="100px" size="small">
                <NInput v-model:value="proc.config.url" size="small" placeholder="https://example.com/webhook" />
              </NFormItem>
              <NFormItem :label="t('eventPipeline.method')" label-width="100px" size="small">
                <NSelect v-model:value="proc.config.method" :options="methodOptions" size="small" />
              </NFormItem>
              <NFormItem :label="t('eventPipeline.timeout')" label-width="100px" size="small">
                <NInputNumber v-model:value="proc.config.timeout" size="small" :min="1" :max="120" style="width: 120px;" />
              </NFormItem>
              <NFormItem :label="t('eventPipeline.skipSSLVerify')" label-width="100px" size="small">
                <NCheckbox v-model:checked="proc.config.skip_ssl_verify" />
              </NFormItem>
            </div>

            <!-- Event Drop config -->
            <div v-if="proc.typ === 'event_drop'" class="processor-fields">
              <NFormItem :label="t('eventPipeline.condition')" label-width="100px" size="small">
                <NInput
                  v-model:value="proc.config.condition"
                  type="textarea"
                  size="small"
                  :autosize="{ minRows: 2, maxRows: 6 }"
                  placeholder="severity == 'info'"
                />
              </NFormItem>
            </div>

            <!-- AI Summary config -->
            <div v-if="proc.typ === 'ai_summary'" class="processor-fields">
              <NFormItem :label="t('eventPipeline.onlyCritical')" label-width="100px" size="small">
                <NCheckbox v-model:checked="proc.config.only_critical" />
              </NFormItem>
            </div>
          </div>
        </NForm>

        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 8px;">
            <NButton @click="showDrawer = false">{{ t('common.cancel') }}</NButton>
            <NButton type="primary" :loading="saving" @click="handleSave">{{ t('common.save') }}</NButton>
          </div>
        </template>
      </NDrawerContent>
    </NDrawer>

    <!-- Execution History Drawer -->
    <NDrawer v-model:show="showExecDrawer" :width="720" placement="right">
      <NDrawerContent :title="t('eventPipeline.history') + ' - ' + execPipelineName">
        <NDataTable
          :columns="execColumns"
          :data="execList"
          :loading="execLoading"
          :row-key="(row: EventPipelineExecution) => row.id"
          size="small"
          :bordered="false"
          striped
        />
        <div class="page-pagination" v-if="execTotal > 0">
          <NPagination
            v-model:page="execPage"
            v-model:page-size="execPageSize"
            :item-count="execTotal"
            :page-sizes="[20, 50]"
            show-size-picker
            @update:page="handleExecPageChange"
          />
        </div>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped>
.event-pipelines-page {
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
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-weight: 500;
}
.empty-hint {
  color: var(--n-text-color-3);
  font-size: 13px;
  padding: 8px 0;
}
.filter-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
.processor-card {
  border: 1px solid var(--n-border-color);
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 12px;
}
.processor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.processor-fields {
  margin-top: 8px;
}
</style>
