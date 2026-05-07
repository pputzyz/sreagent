<script setup lang="ts">
import { h, ref, reactive, onMounted, computed, watch } from 'vue'
import { useMessage, NTag, NButton, NSpace, NPopconfirm } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { alertRuleApi, datasourceApi, templateApi } from '@/api'
import type { AlertRule, DataSource, AlertSeverity, DataSourceType, QueryResponse } from '@/types'
import { formatTime, kvArrayToRecord } from '@/utils/format'
import { getSeverityType, getRuleStatusType } from '@/utils/alert'
import KVEditor from '@/components/common/KVEditor.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import { AddOutline, RefreshOutline, CloudUploadOutline, CloudDownloadOutline, PlayOutline, FunnelOutline, CheckmarkDoneOutline, BanOutline, TrashOutline } from '@vicons/ionicons5'

const message = useMessage()
const { t } = useI18n()
const loading = ref(false)
const rules = ref<AlertRule[]>([])
const total = ref(0)
const page = ref(1)
const datasources = ref<DataSource[]>([])

// Batch selection
const selectedKeys = ref<number[]>([])
const batchLoading = ref(false)

async function handleBatchEnable() {
  if (selectedKeys.value.length === 0) return
  batchLoading.value = true
  try {
    await alertRuleApi.batchEnable(selectedKeys.value)
    message.success(t('alert.batchEnabled', { count: selectedKeys.value.length }))
    selectedKeys.value = []
    fetchRules()
  } catch (err: any) {
    message.error(err.message)
  } finally {
    batchLoading.value = false
  }
}

async function handleBatchDisable() {
  if (selectedKeys.value.length === 0) return
  batchLoading.value = true
  try {
    await alertRuleApi.batchDisable(selectedKeys.value)
    message.success(t('alert.batchDisabled', { count: selectedKeys.value.length }))
    selectedKeys.value = []
    fetchRules()
  } catch (err: any) {
    message.error(err.message)
  } finally {
    batchLoading.value = false
  }
}

async function handleBatchDelete() {
  if (selectedKeys.value.length === 0) return
  batchLoading.value = true
  try {
    await alertRuleApi.batchDelete(selectedKeys.value)
    message.success(t('alert.batchDeleted', { count: selectedKeys.value.length }))
    selectedKeys.value = []
    fetchRules()
  } catch (err: any) {
    message.error(err.message)
  } finally {
    batchLoading.value = false
  }
}

// Category state
const activeCategory = ref('')
const categories = ref<string[]>([])

// Expression test state
const queryTesting = ref(false)
const queryResult = ref<QueryResponse | null>(null)

// Template state
const showTemplatePicker = ref(false)
const templateLoading = ref(false)
const templates = ref<any[]>([])
const templateCategories = ref<string[]>([])
const templateSearch = ref('')
const templateCategory = ref('')
const appliedTemplateId = ref<number | null>(null)

async function fetchTemplates() {
  templateLoading.value = true
  try {
    const res = await templateApi.list({
      category: templateCategory.value || undefined,
      search: templateSearch.value || undefined,
      page: 1,
      page_size: 50,
    })
    templates.value = res.data.data.list || []
    templateCategories.value = (await templateApi.listCategories()).data.data || []
  } catch { /* ignore */ }
  finally { templateLoading.value = false }
}

async function loadTemplate(tpl: any) {
  appliedTemplateId.value = tpl.id
  Object.assign(form, {
    name: tpl.name || '',
    display_name: '',
    description: tpl.description || '',
    datasource_type: tpl.datasource_type || '',
    expression: tpl.expression || '',
    for_duration: tpl.for_duration || '5m',
    severity: tpl.severity || 'warning',
    labels: tpl.labels ? Object.entries(tpl.labels).map(([k, v]: any) => ({ key: k, value: v })) : [],
    annotations: tpl.annotations ? Object.entries(tpl.annotations).map(([k, v]: any) => ({ key: k, value: v })) : [],
    group_name: tpl.group_name || '',
    category: tpl.category || '',
  })
  showTemplatePicker.value = false
  message.success(t('alert.templateLoaded') || 'Template loaded')
}

async function saveAsTemplate() {
  const payload = {
    name: form.name,
    description: form.description,
    datasource_type: form.datasource_type,
    expression: form.expression,
    for_duration: form.for_duration,
    severity: form.severity,
    labels: kvArrayToRecord(form.labels),
    annotations: kvArrayToRecord(form.annotations),
    group_name: form.group_name,
    category: form.category,
  }
  try {
    await templateApi.create(payload)
    message.success(t('alert.templateSaved') || 'Template saved')
  } catch (err: any) {
    message.error(err.message || t('common.saveFailed'))
  }
}

function openTemplatePicker() {
  fetchTemplates()
  showTemplatePicker.value = true
}

// Import/Export state
const showImportExport = ref(false)
const importFile = ref<File | null>(null)
const importDatasourceId = ref<number | null>(null)
const importing = ref(false)
const exportFormat = ref('yaml')
const exportCategory = ref('')

const categoryOptions = [
  { label: () => t('alert.categoryNode'), value: 'node' },
  { label: () => t('alert.categoryDatabase'), value: 'database' },
  { label: () => t('alert.categoryMiddleware'), value: 'middleware' },
  { label: () => t('alert.categoryNetwork'), value: 'network' },
  { label: () => t('alert.categoryApplication'), value: 'application' },
  { label: () => t('alert.categoryCustom'), value: 'custom' },
]

// Modal state
const showModal = ref(false)
const modalTitle = ref('')
const editingId = ref<number | null>(null)
const saving = ref(false)

const defaultForm = {
  name: '',
  display_name: '',
  description: '',
  datasource_id: null as number | null,
  datasource_type: '' as DataSourceType | '',
  expression: '',
  for_duration: '5m',
  severity: 'warning' as AlertSeverity,
  labels: [] as { key: string; value: string }[],
  annotations: [] as { key: string; value: string }[],
  group_name: '',
  category: '',
  group_wait_seconds: 0,
  group_interval_seconds: 0,
}

const form = reactive({ ...defaultForm })

const severityOptions = [
  { label: () => t('alert.p0'), value: 'p0' },
  { label: () => t('alert.p1'), value: 'p1' },
  { label: () => t('alert.p2'), value: 'p2' },
  { label: () => t('alert.p3'), value: 'p3' },
  { label: () => t('alert.p4'), value: 'p4' },
  { label: () => t('alert.critical'), value: 'critical' },
  { label: () => t('alert.warning'), value: 'warning' },
  { label: () => t('alert.info'), value: 'info' },
]

const datasourceOptions = computed(() =>
  datasources.value.map(ds => ({ label: `${ds.name} (${ds.type})`, value: ds.id }))
)

const datasourceTypeOptions = [
  { label: 'Prometheus', value: 'prometheus' },
  { label: 'VictoriaMetrics', value: 'victoriametrics' },
  { label: 'Zabbix', value: 'zabbix' },
  { label: 'VictoriaLogs', value: 'victorialogs' },
]

const selectedDatasource = computed(() =>
  datasources.value.find(ds => ds.id === form.datasource_id)
)

// When a specific datasource is selected, auto-fill datasource_type
watch(() => form.datasource_id, (newId) => {
  if (newId != null) {
    const ds = datasources.value.find(d => d.id === newId)
    if (ds) form.datasource_type = ds.type as DataSourceType
  }
})

const expressionLang = computed(() => {
  const t = selectedDatasource.value?.type
  if (t === 'victorialogs') return 'LogsQL'
  if (t === 'zabbix') return 'Zabbix'
  return 'PromQL'
})

const expressionPlaceholder = computed(() => {
  const t = selectedDatasource.value?.type
  if (t === 'victorialogs') return 'e.g. error level:error _time:5m'
  if (t === 'zabbix') return 'e.g. system.cpu.util[,user]'
  return 'e.g. avg(rate(cpu_usage_total[5m])) > 0.9'
})

const columns = [
  { type: 'selection' as const },
  {
    title: () => t('common.name'),
    key: 'name',
    width: 160,
    ellipsis: { tooltip: true },
    render: (row: AlertRule) =>
      h('div', [
        h('div', { style: 'font-weight: 500' }, row.display_name || row.name),
        h('div', { style: 'font-size: 11px; color: var(--sre-text-secondary)' }, row.name),
      ]),
  },
  {
    title: () => t('alert.groupName'),
    key: 'group_name',
    width: 120,
    ellipsis: { tooltip: true },
  },
  {
    title: () => t('alert.category'),
    key: 'category',
    width: 110,
    render: (row: AlertRule) =>
      row.category
        ? h(NTag, { size: 'small', round: true, bordered: false, type: 'info' }, { default: () => row.category })
        : h('span', { style: 'color: var(--sre-text-secondary); font-size: 12px' }, '-'),
  },
  {
    title: () => t('alert.severity'),
    key: 'severity',
    width: 100,
    render: (row: AlertRule) =>
      h(NTag, { type: getSeverityType(row.severity), size: 'small', round: true }, { default: () => t(`alert.${row.severity}` as any) || row.severity }),
  },
  {
    title: () => t('alert.expression'),
    key: 'expression',
    ellipsis: { tooltip: true },
    render: (row: AlertRule) =>
      h('code', { style: 'font-size: 12px; color: var(--sre-text-secondary)' }, row.expression),
  },
  {
    title: () => t('alert.forDuration'),
    key: 'for_duration',
    width: 90,
  },
  {
    title: () => t('common.status'),
    key: 'status',
    width: 100,
    render: (row: AlertRule) =>
      h(NTag, { type: getRuleStatusType(row.status), size: 'small' }, { default: () => row.status }),
  },
  {
    title: () => t('common.actions'),
    key: 'actions',
    width: 220,
    render: (row: AlertRule) =>
      h(NSpace, { size: 4 }, {
        default: () => [
          h(NButton, {
            size: 'small',
            quaternary: true,
            type: 'info',
            onClick: () => openEdit(row),
          }, { default: () => t('common.edit') }),
          h(NButton, {
            size: 'small',
            quaternary: true,
            type: row.status === 'enabled' ? 'warning' : 'success',
            onClick: () => handleToggleStatus(row),
          }, { default: () => row.status === 'enabled' ? t('common.disabled') : t('common.enabled') }),
          h(NPopconfirm, {
            onPositiveClick: () => handleDelete(row.id),
          }, {
            trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error' }, { default: () => t('common.delete') }),
            default: () => t('alert.deleteRuleConfirm'),
          }),
        ],
      }),
  },
]

async function fetchRules() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: page.value, page_size: 50 }
    if (activeCategory.value) params.category = activeCategory.value
    const { data } = await alertRuleApi.list(params)
    rules.value = data.data.list || []
    total.value = data.data.total
  } catch (err: any) {
    message.error(err.message)
  } finally {
    loading.value = false
  }
}

async function fetchCategories() {
  try {
    const { data } = await alertRuleApi.listCategories()
    categories.value = data.data || []
  } catch {
    // silently fail
  }
}

function handleCategoryChange(cat: string) {
  activeCategory.value = cat
  page.value = 1
  fetchRules()
}

async function fetchDatasources() {
  try {
    const { data } = await datasourceApi.list({ page: 1, page_size: 100 })
    datasources.value = data.data.list || []
  } catch (_err) {
    // silently fail
  }
}

function openCreate() {
  editingId.value = null
  modalTitle.value = t('alert.createRule')
  Object.assign(form, {
    name: '',
    display_name: '',
    description: '',
    datasource_id: null,
    datasource_type: '',
    expression: '',
    for_duration: '5m',
    severity: 'warning',
    labels: [],
    annotations: [],
    group_name: '',
    category: '',
    group_wait_seconds: 0,
    group_interval_seconds: 0,
  })
  queryResult.value = null
  showModal.value = true
}

function openEdit(rule: AlertRule) {
  editingId.value = rule.id
  modalTitle.value = t('alert.editRule')
  Object.assign(form, {
    name: rule.name,
    display_name: rule.display_name,
    description: rule.description,
    datasource_id: rule.datasource_id,
    datasource_type: rule.datasource_type || '',
    expression: rule.expression,
    for_duration: rule.for_duration,
    severity: rule.severity,
    labels: Object.entries(rule.labels || {}).map(([key, value]) => ({ key, value })),
    annotations: Object.entries(rule.annotations || {}).map(([key, value]) => ({ key, value })),
    group_name: rule.group_name,
    category: rule.category || '',
    group_wait_seconds: rule.group_wait_seconds || 0,
    group_interval_seconds: rule.group_interval_seconds || 0,
  })
  queryResult.value = null
  showModal.value = true
}


async function handleSave() {
  if (!form.name.trim()) {
    message.warning(t('alert.nameRequired'))
    return
  }
  if (!form.expression.trim()) {
    message.warning(t('alert.expressionRequired'))
    return
  }
  if (form.datasource_id == null && !form.datasource_type) {
    message.warning(t('alert.datasourceRequired', 'A datasource or datasource type is required'))
    return
  }

  saving.value = true
  try {
    const payload = {
      name: form.name,
      display_name: form.display_name,
      description: form.description,
      datasource_id: form.datasource_id,
      datasource_type: form.datasource_type,
      expression: form.expression,
      for_duration: form.for_duration,
      severity: form.severity,
      labels: kvArrayToRecord(form.labels),
      annotations: kvArrayToRecord(form.annotations),
      group_name: form.group_name,
      category: form.category,
      group_wait_seconds: form.group_wait_seconds,
      group_interval_seconds: form.group_interval_seconds,
    }

    if (editingId.value) {
      await alertRuleApi.update(editingId.value, payload)
      message.success(t('alert.ruleUpdated'))
    } else {
      await alertRuleApi.create(payload)
      message.success(t('alert.ruleCreated'))
    }
    showModal.value = false
    fetchRules()
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

async function handleToggleStatus(rule: AlertRule) {
  const newStatus = rule.status === 'enabled' ? 'disabled' : 'enabled'
  try {
    await alertRuleApi.toggleStatus(rule.id, newStatus)
    message.success(newStatus === 'enabled' ? t('alert.ruleEnabled') : t('alert.ruleDisabled'))
    fetchRules()
  } catch (err: any) {
    message.error(err.message)
  }
}

async function handleDelete(id: number) {
  try {
    await alertRuleApi.delete(id)
    message.success(t('alert.ruleDeleted'))
    fetchRules()
  } catch (err: any) {
    message.error(err.message)
  }
}

async function handleTestExpression() {
  if (!form.datasource_id || !form.expression.trim()) return
  queryTesting.value = true
  queryResult.value = null
  try {
    const { data } = await datasourceApi.query(form.datasource_id, { expression: form.expression })
    queryResult.value = data.data
  } catch (err: any) {
    message.error(err.message || 'Query failed')
  } finally {
    queryTesting.value = false
  }
}

async function handleImport() {
  if (!importFile.value) return
  importing.value = true
  try {
    const { data } = await alertRuleApi.importRules(importFile.value, importDatasourceId.value || undefined)
    const result = data.data
    message.success(t('alert.rulesImported', { success: result.success, total: result.total }))
    if (result.errors && result.errors.length > 0) {
      message.warning(result.errors.join('\n'))
    }
    showImportExport.value = false
    importFile.value = null
    fetchRules()
    fetchCategories()
  } catch (err: any) {
    message.error(err.message)
  } finally {
    importing.value = false
  }
}

async function handleExport() {
  try {
    const params: Record<string, string> = { format: exportFormat.value }
    if (exportCategory.value) params.category = exportCategory.value
    const response = await alertRuleApi.exportRules(params)
    const blob = new Blob([response.data as any])
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `alert-rules.${exportFormat.value}`
    a.click()
    URL.revokeObjectURL(url)
  } catch (err: any) {
    message.error(err.message)
  }
}

onMounted(() => {
  fetchRules()
  fetchDatasources()
  fetchCategories()
})
</script>

<template>
  <div class="rules-page">
    <PageHeader :title="t('alert.rules')" :subtitle="t('alert.rulesSubtitle')">
      <template #actions>
        <n-button @click="showImportExport = true">
          <template #icon><n-icon :component="FunnelOutline" /></template>
          {{ t('alert.importExport') }}
        </n-button>
        <n-button @click="fetchRules" :loading="loading">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          {{ t('common.refresh') }}
        </n-button>
        <n-button type="primary" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('alert.createRule') }}
        </n-button>
      </template>
    </PageHeader>

    <!-- Category Tabs -->
    <div class="category-tabs">
      <n-button
        :type="activeCategory === '' ? 'primary' : 'default'"
        size="small"
        @click="handleCategoryChange('')"
        :quaternary="activeCategory !== ''"
      >
        {{ t('alert.allCategories') }}
        <template #icon v-if="activeCategory === ''"><n-icon :component="FunnelOutline" /></template>
      </n-button>
      <n-button
        v-for="cat in categories"
        :key="cat"
        :type="activeCategory === cat ? 'primary' : 'default'"
        size="small"
        @click="handleCategoryChange(cat)"
        :quaternary="activeCategory !== cat"
      >
        {{ cat }}
      </n-button>
    </div>

    <!-- Batch toolbar -->
    <div v-if="selectedKeys.length > 0" style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px; padding: 8px 12px; background: var(--sre-bg-card); border-radius: 8px; border: 1px solid var(--sre-border)">
      <span style="font-size: 13px; color: var(--sre-text-secondary)">
        {{ t('common.selected', { count: selectedKeys.length }) }}
      </span>
      <n-button size="small" type="success" :loading="batchLoading" @click="handleBatchEnable">
        <template #icon><n-icon :component="CheckmarkDoneOutline" /></template>
        {{ t('common.enabled') }}
      </n-button>
      <n-button size="small" type="warning" :loading="batchLoading" @click="handleBatchDisable">
        <template #icon><n-icon :component="BanOutline" /></template>
        {{ t('common.disabled') }}
      </n-button>
      <n-popconfirm @positive-click="handleBatchDelete">
        <template #trigger>
          <n-button size="small" type="error" :loading="batchLoading">
            <template #icon><n-icon :component="TrashOutline" /></template>
            {{ t('common.delete') }}
          </n-button>
        </template>
        {{ t('alert.batchDeleteConfirm', { count: selectedKeys.length }) }}
      </n-popconfirm>
      <n-button size="small" quaternary @click="selectedKeys = []">{{ t('common.cancel') }}</n-button>
    </div>

    <n-card :bordered="false" style="background: var(--sre-bg-card); border-radius: 12px">
      <n-data-table
        :loading="loading"
        :columns="columns"
        :data="rules"
        :row-key="(row: AlertRule) => row.id"
        :bordered="false"
        :checked-row-keys="selectedKeys"
        @update:checked-row-keys="(keys) => selectedKeys = keys as number[]"
        :pagination="{
          page: page,
          pageSize: 50,
          itemCount: total,
          onChange: (p: number) => { page = p; fetchRules() },
        }"
      />

      <n-empty v-if="!loading && rules.length === 0" :description="t('alert.noRules')" style="padding: 60px 0">
        <template #extra>
          <n-button type="primary" @click="openCreate">{{ t('alert.createFirstRule') }}</n-button>
        </template>
      </n-empty>
    </n-card>

    <!-- Create/Edit Modal -->
    <n-modal v-model:show="showModal" preset="card" :title="modalTitle" style="width: 680px" :bordered="false">
      <!-- Template bar (create mode only) -->
      <div v-if="!editingId" style="margin-bottom: 16px; display: flex; gap: 8px; align-items: center">
        <n-button size="small" secondary @click="openTemplatePicker">
          {{ t('alert.loadFromTemplate', 'Load from template') }}
        </n-button>
        <span v-if="appliedTemplateId" style="font-size: 12px; color: var(--sre-text-tertiary)">
          {{ t('alert.templateApplied', 'Template applied') }}
        </span>
      </div>

      <!-- Template picker sub-modal -->
      <n-modal v-model:show="showTemplatePicker" preset="card" :title="t('alert.selectTemplate', 'Select Template')" style="width: 600px">
        <div style="display: flex; gap: 8px; margin-bottom: 12px">
          <n-input v-model:value="templateSearch" :placeholder="t('common.search')" size="small" clearable style="flex: 1" @update:value="fetchTemplates" />
          <n-select
            v-model:value="templateCategory"
            :options="templateCategories.map(c => ({ label: c, value: c }))"
            :placeholder="t('alert.category')"
            size="small"
            clearable
            style="width: 140px"
            @update:value="fetchTemplates"
          />
        </div>
        <n-data-table
          :columns="[
            { title: () => t('common.name'), key: 'name', ellipsis: { tooltip: true } },
            { title: () => t('alert.category'), key: 'category', width: 100 },
            { title: () => t('alert.severity'), key: 'severity', width: 80 },
            {
              title: () => '',
              key: 'action',
              width: 80,
              render: (row: any) => h(NButton, { size: 'tiny', secondary: true, onClick: () => loadTemplate(row) }, { default: () => t('common.apply') }),
            },
          ]"
          :data="templates"
          :loading="templateLoading"
          :bordered="false"
          size="small"
          :pagination="false"
          :max-height="320"
          :row-key="(row: any) => row.id"
        />
      </n-modal>

      <n-form label-placement="top">
        <n-grid :x-gap="12" :cols="2">
          <n-gi>
            <n-form-item :label="t('common.name')" required>
              <n-input v-model:value="form.name" placeholder="e.g. high_cpu_usage" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('alert.displayName')">
              <n-input v-model:value="form.display_name" placeholder="e.g. High CPU Usage" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-form-item :label="t('common.description')">
          <n-input v-model:value="form.description" type="textarea" :placeholder="t('common.description')" :rows="2" />
        </n-form-item>

        <n-grid :x-gap="12" :cols="2">
          <n-gi>
            <n-form-item :label="t('alert.dataSource')">
              <n-select v-model:value="form.datasource_id" :options="datasourceOptions" :placeholder="t('alert.selectDataSource')" clearable />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('alert.datasourceType', 'Datasource Type')">
              <n-select
                v-model:value="form.datasource_type"
                :options="datasourceTypeOptions"
                :placeholder="t('alert.selectDatasourceType', 'Auto or select type')"
                :disabled="form.datasource_id != null"
                clearable
              />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-grid :x-gap="12" :cols="2">
          <n-gi>
            <n-form-item :label="t('alert.groupName')">
              <n-input v-model:value="form.group_name" placeholder="e.g. infrastructure" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-grid :x-gap="12" :cols="2">
          <n-gi>
            <n-form-item :label="t('alert.groupWait')">
              <n-input-number v-model:value="form.group_wait_seconds" :min="0" :max="3600" :placeholder="t('alert.groupWaitPlaceholder')" style="width: 100%">
                <template #suffix>{{ t('common.seconds', '秒') }}</template>
              </n-input-number>
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('alert.groupInterval')">
              <n-input-number v-model:value="form.group_interval_seconds" :min="0" :max="86400" :placeholder="t('alert.groupIntervalPlaceholder')" style="width: 100%">
                <template #suffix>{{ t('common.seconds', '秒') }}</template>
              </n-input-number>
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-form-item :label="t('alert.category')">
          <n-select
            v-model:value="form.category"
            :options="categoryOptions"
            :placeholder="t('alert.selectCategory')"
            clearable
            tag
            filterable
          />
        </n-form-item>

        <n-form-item required>
          <template #label>
            <n-space size="small" align="center" style="gap:6px">
              <span>{{ t('alert.expression') }}</span>
              <n-tag size="tiny" :type="expressionLang === 'LogsQL' ? 'info' : expressionLang === 'Zabbix' ? 'warning' : 'success'" round>
                {{ expressionLang }}
              </n-tag>
            </n-space>
          </template>
          <div style="width: 100%">
            <n-input
              v-model:value="form.expression"
              type="textarea"
              :placeholder="expressionPlaceholder"
              :rows="3"
              style="font-family: monospace"
            />
            <div style="margin-top: 8px; display: flex; align-items: center; gap: 8px">
              <n-button
                size="small"
                :loading="queryTesting"
                :disabled="!form.datasource_id || !form.expression.trim()"
                @click="handleTestExpression"
              >
                <template #icon><n-icon :component="PlayOutline" /></template>
                {{ queryTesting ? t('alert.testing') : t('alert.testExpression') }}
              </n-button>
            </div>
            <n-collapse-transition :show="queryResult !== null">
              <div class="query-result">
                <div class="query-result__header">{{ t('alert.testResult') }}</div>
                <div v-if="queryResult?.result_type === 'logs'" style="font-size: 13px; color: var(--sre-text-secondary)">
                  {{ t('alert.matchedLogs') }}: {{ queryResult.raw_count }}
                </div>
                <div v-else-if="queryResult?.series && queryResult.series.length > 0">
                  <n-data-table
                    :columns="[
                      { title: 'Labels', key: 'labels', render: (row: any) => Object.entries(row.labels || {}).map(([k, v]: any) => `${k}=${v}`).join(', ') },
                      { title: 'Value', key: 'value', width: 120, render: (row: any) => row.values?.[0]?.value ?? '-' },
                    ]"
                    :data="queryResult.series"
                    :bordered="false"
                    size="small"
                    :pagination="false"
                    :max-height="200"
                  />
                </div>
                <div v-else style="font-size: 13px; color: var(--sre-text-secondary); padding: 8px 0">
                  {{ t('alert.noResults') }}
                </div>
              </div>
            </n-collapse-transition>
          </div>
        </n-form-item>

        <n-grid :x-gap="12" :cols="2">
          <n-gi>
            <n-form-item :label="t('alert.forDuration')">
              <n-input v-model:value="form.for_duration" placeholder="e.g. 5m, 10m, 1h" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('alert.severity')">
              <n-select v-model:value="form.severity" :options="severityOptions" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <!-- Labels -->
        <n-form-item :label="t('alert.labels')">
          <KVEditor v-model:modelValue="form.labels" :add-label="t('alert.addLabel')" />
        </n-form-item>

        <!-- Annotations -->
        <n-form-item :label="t('alert.annotations')">
          <KVEditor v-model:modelValue="form.annotations" :add-label="t('alert.addAnnotation')" key-placeholder="Key (e.g. summary)" />
        </n-form-item>
      </n-form>

      <template #action>
        <n-space justify="space-between" style="width: 100%">
          <n-button size="small" secondary @click="saveAsTemplate">
            {{ t('alert.saveAsTemplate', 'Save as template') }}
          </n-button>
          <n-space>
            <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
            <n-button type="primary" :loading="saving" @click="handleSave">
              {{ editingId ? t('common.update') : t('common.create') }}
            </n-button>
          </n-space>
        </n-space>
      </template>
    </n-modal>

    <!-- Import/Export Drawer -->
    <n-drawer v-model:show="showImportExport" :width="480" placement="right">
      <n-drawer-content :title="t('alert.importExport')">
        <n-tabs type="line">
          <n-tab-pane name="import" :tab="t('alert.importFile')">
            <n-space vertical size="large">
              <n-upload
                :max="1"
                accept=".yaml,.yml,.json"
                :default-upload="false"
                @change="({ file }: any) => { importFile = file?.file || null }"
              >
                <n-upload-dragger>
                  <div style="padding: 20px; text-align: center">
                    <n-icon :component="CloudUploadOutline" :size="36" style="color: var(--sre-text-secondary)" />
                    <div style="margin-top: 8px; color: var(--sre-text-secondary); font-size: 13px">
                      {{ t('alert.dragOrClick') }}
                    </div>
                  </div>
                </n-upload-dragger>
              </n-upload>
              <n-form-item :label="t('alert.dataSource')">
                <n-select
                  v-model:value="importDatasourceId"
                  :options="datasourceOptions"
                  :placeholder="t('alert.selectDataSource')"
                  clearable
                />
              </n-form-item>
              <n-button
                type="primary"
                block
                :loading="importing"
                :disabled="!importFile"
                @click="handleImport"
              >
                {{ t('alert.importFile') }}
              </n-button>
            </n-space>
          </n-tab-pane>
          <n-tab-pane name="export" :tab="t('alert.exportRules')">
            <n-space vertical size="large">
              <n-form-item :label="t('alert.exportFormat')">
                <n-radio-group v-model:value="exportFormat">
                  <n-radio-button value="yaml">YAML</n-radio-button>
                  <n-radio-button value="json">JSON</n-radio-button>
                </n-radio-group>
              </n-form-item>
              <n-form-item :label="t('alert.category')">
                <n-select
                  v-model:value="exportCategory"
                  :options="[{ label: t('alert.allCategories'), value: '' }, ...categories.map(c => ({ label: c, value: c }))]"
                  :placeholder="t('alert.selectCategory')"
                />
              </n-form-item>
              <n-button
                type="primary"
                block
                @click="handleExport"
              >
                <template #icon><n-icon :component="CloudDownloadOutline" /></template>
                {{ t('alert.exportRules') }}
              </n-button>
            </n-space>
          </n-tab-pane>
        </n-tabs>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<style scoped>
.rules-page {
  max-width: 1400px;
}

.category-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}

.query-result {
  margin-top: 12px;
  padding: 12px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.query-result__header {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}
</style>
