<script setup lang="ts">
import { h, ref, shallowRef, reactive, onMounted, computed, watch } from 'vue'
import { useMessage, NButton, NIcon, NDropdown } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { alertRuleApi, datasourceApi, templateApi } from '@/api'
import type { AlertRule, DataSource, AlertSeverity, DataSourceType, QueryResponse } from '@/types'
import { kvArrayToRecord } from '@/utils/format'
import KVEditor from '@/components/common/KVEditor.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import {
  AddOutline,
  CloudUploadOutline,
  CloudDownloadOutline,
  PlayOutline,
  SearchOutline,
  EllipsisHorizontalOutline,
  FileTrayOutline,
  CloseOutline,
  CreateOutline,
  CopyOutline,
  TrashOutline,
  PowerOutline,
  DocumentTextOutline,
} from '@vicons/ionicons5'

const message = useMessage()
const { t } = useI18n()
const router = useRouter()

const loading = ref(false)
const rules = shallowRef<AlertRule[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 50
const datasources = ref<DataSource[]>([])
const isFirstLoad = ref(true)

// Filters
const searchKeyword = ref('')
const filterDatasource = ref<number | null>(null)
const filterSeverity = ref<string | null>(null)
const filterStatus = ref<string | null>(null)

// Batch
const selectedKeys = ref<number[]>([])
const batchLoading = ref(false)

// Category
const activeCategory = ref('')
const categories = ref<string[]>([])
const categoryCounts = ref<Record<string, number>>({})

// Expression test
const queryTesting = ref(false)
const queryResult = ref<QueryResponse | null>(null)

// Templates
const showTemplatePicker = ref(false)
const templateLoading = ref(false)
const templates = ref<any[]>([])
const templateCategories = ref<string[]>([])
const templateSearch = ref('')
const templateCategory = ref('')
const appliedTemplateId = ref<number | null>(null)

// Import/Export
const showImportExport = ref(false)
const importFile = ref<File | null>(null)
const importDatasourceId = ref<number | null>(null)
const importing = ref(false)
const exportFormat = ref('yaml')
const exportCategory = ref('')

// Modal
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

const severityFilterOptions = computed(() => [
  { label: t('alert.critical'), value: 'critical' },
  { label: t('alert.warning'), value: 'warning' },
  { label: t('alert.info'), value: 'info' },
])

const statusFilterOptions = computed(() => [
  { label: t('common.enabled'), value: 'enabled' },
  { label: t('common.disabled'), value: 'disabled' },
])

const categoryOptions = [
  { label: () => t('alert.categoryNode'), value: 'node' },
  { label: () => t('alert.categoryDatabase'), value: 'database' },
  { label: () => t('alert.categoryMiddleware'), value: 'middleware' },
  { label: () => t('alert.categoryNetwork'), value: 'network' },
  { label: () => t('alert.categoryApplication'), value: 'application' },
  { label: () => t('alert.categoryCustom'), value: 'custom' },
]

const datasourceOptions = computed(() =>
  datasources.value.map(ds => ({ label: `${ds.name} (${ds.type})`, value: ds.id })),
)

const datasourceTypeOptions = [
  { label: 'Prometheus', value: 'prometheus' },
  { label: 'VictoriaMetrics', value: 'victoriametrics' },
  { label: 'Zabbix', value: 'zabbix' },
  { label: 'VictoriaLogs', value: 'victorialogs' },
]

const selectedDatasource = computed(() => datasources.value.find(ds => ds.id === form.datasource_id))

watch(() => form.datasource_id, (newId) => {
  if (newId != null) {
    const ds = datasources.value.find(d => d.id === newId)
    if (ds) form.datasource_type = ds.type as DataSourceType
  }
})

const expressionLang = computed(() => {
  const tp = selectedDatasource.value?.type
  if (tp === 'victorialogs') return 'LogsQL'
  if (tp === 'zabbix') return 'Zabbix'
  return 'PromQL'
})

const expressionPlaceholder = computed(() => {
  const tp = selectedDatasource.value?.type
  if (tp === 'victorialogs') return 'e.g. error level:error _time:5m'
  if (tp === 'zabbix') return 'e.g. system.cpu.util[,user]'
  return 'e.g. avg(rate(cpu_usage_total[5m])) > 0.9'
})

// Filtered list (client-side filter on top of paginated data)
const filteredRules = computed(() => {
  let arr = rules.value
  if (searchKeyword.value.trim()) {
    const kw = searchKeyword.value.trim().toLowerCase()
    arr = arr.filter(r =>
      r.name?.toLowerCase().includes(kw) ||
      r.display_name?.toLowerCase().includes(kw) ||
      r.expression?.toLowerCase().includes(kw),
    )
  }
  if (filterDatasource.value != null) {
    arr = arr.filter(r => r.datasource_id === filterDatasource.value)
  }
  if (filterSeverity.value) {
    arr = arr.filter(r => r.severity === filterSeverity.value)
  }
  if (filterStatus.value) {
    arr = arr.filter(r => r.status === filterStatus.value)
  }
  return arr
})

const allCount = computed(() => total.value)

function severityLabel(sev: string) {
  const map: Record<string, string> = {
    critical: t('alert.critical'),
    warning: t('alert.warning'),
    info: t('alert.info'),
    p0: t('alert.p0'), p1: t('alert.p1'), p2: t('alert.p2'), p3: t('alert.p3'), p4: t('alert.p4'),
  }
  return map[sev] || sev
}

function severitySlot(sev: string): 'critical' | 'warning' | 'info' | 'success' {
  if (sev === 'critical' || sev === 'p0' || sev === 'p1') return 'critical'
  if (sev === 'warning' || sev === 'p2') return 'warning'
  if (sev === 'info' || sev === 'p4') return 'info'
  return 'info'
}

async function fetchRules() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: page.value, page_size: pageSize }
    if (activeCategory.value) params.category = activeCategory.value
    const { data } = await alertRuleApi.list(params)
    rules.value = data.data.list || []
    total.value = data.data.total
  } catch (err: any) {
    message.error(err.message)
  } finally {
    loading.value = false
    if (isFirstLoad.value) {
      // disable stagger after first load
      setTimeout(() => { isFirstLoad.value = false }, 800)
    }
  }
}

async function fetchCategories() {
  try {
    const { data } = await alertRuleApi.listCategories()
    categories.value = data.data || []
  } catch { /* ignore */ }
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
  } catch { /* ignore */ }
}

// Batch
async function handleBatchEnable() {
  if (selectedKeys.value.length === 0) return
  batchLoading.value = true
  try {
    await alertRuleApi.batchEnable(selectedKeys.value)
    message.success(t('alert.batchEnabled', { count: selectedKeys.value.length }))
    selectedKeys.value = []
    fetchRules()
  } catch (err: any) { message.error(err.message) } finally { batchLoading.value = false }
}

async function handleBatchDisable() {
  if (selectedKeys.value.length === 0) return
  batchLoading.value = true
  try {
    await alertRuleApi.batchDisable(selectedKeys.value)
    message.success(t('alert.batchDisabled', { count: selectedKeys.value.length }))
    selectedKeys.value = []
    fetchRules()
  } catch (err: any) { message.error(err.message) } finally { batchLoading.value = false }
}

async function handleBatchDelete() {
  if (selectedKeys.value.length === 0) return
  batchLoading.value = true
  try {
    await alertRuleApi.batchDelete(selectedKeys.value)
    message.success(t('alert.batchDeleted', { count: selectedKeys.value.length }))
    selectedKeys.value = []
    fetchRules()
  } catch (err: any) { message.error(err.message) } finally { batchLoading.value = false }
}

function toggleSelect(id: number, checked: boolean) {
  if (checked) {
    if (!selectedKeys.value.includes(id)) selectedKeys.value = [...selectedKeys.value, id]
  } else {
    selectedKeys.value = selectedKeys.value.filter(k => k !== id)
  }
}

function isSelected(id: number) {
  return selectedKeys.value.includes(id)
}

const allSelected = computed(() =>
  filteredRules.value.length > 0 && filteredRules.value.every(r => selectedKeys.value.includes(r.id)),
)

function toggleSelectAll(checked: boolean) {
  if (checked) {
    selectedKeys.value = filteredRules.value.map(r => r.id)
  } else {
    selectedKeys.value = []
  }
}

// Templates
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

// CRUD
function openCreate() {
  editingId.value = null
  modalTitle.value = t('alert.createRule')
  Object.assign(form, { ...defaultForm, labels: [], annotations: [] })
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
  if (!form.name.trim()) { message.warning(t('alert.nameRequired')); return }
  if (!form.expression.trim()) { message.warning(t('alert.expressionRequired')); return }
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

async function toggleEnabled(rule: AlertRule) {
  const newStatus = rule.status === 'enabled' ? 'disabled' : 'enabled'
  try {
    await alertRuleApi.toggleStatus(rule.id, newStatus)
    message.success(newStatus === 'enabled' ? t('alert.ruleEnabled') : t('alert.ruleDisabled'))
    fetchRules()
  } catch (err: any) { message.error(err.message) }
}

async function handleDelete(id: number) {
  try {
    await alertRuleApi.delete(id)
    message.success(t('alert.ruleDeleted'))
    fetchRules()
  } catch (err: any) { message.error(err.message) }
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
  } finally { queryTesting.value = false }
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
  } catch (err: any) { message.error(err.message) } finally { importing.value = false }
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
  } catch (err: any) { message.error(err.message) }
}

// Row actions menu
function rowActions(rule: AlertRule) {
  return [
    { label: t('common.edit'), key: 'edit', icon: () => h(NIcon, { component: CreateOutline }) },
    { label: rule.status === 'enabled' ? t('common.disabled') : t('common.enabled'), key: 'toggle', icon: () => h(NIcon, { component: PowerOutline }) },
    { label: t('common.duplicate', '复制'), key: 'duplicate', icon: () => h(NIcon, { component: CopyOutline }) },
    { type: 'divider', key: 'd1' },
    { label: t('common.delete'), key: 'delete', icon: () => h(NIcon, { component: TrashOutline }) },
  ]
}

function onRowAction(key: string, rule: AlertRule) {
  if (key === 'edit') openEdit(rule)
  else if (key === 'toggle') toggleEnabled(rule)
  else if (key === 'duplicate') {
    openCreate()
    Object.assign(form, {
      name: rule.name + '_copy',
      display_name: rule.display_name,
      description: rule.description,
      datasource_id: rule.datasource_id,
      datasource_type: rule.datasource_type || '',
      expression: rule.expression,
      for_duration: rule.for_duration,
      severity: rule.severity,
      labels: Object.entries(rule.labels || {}).map(([k, v]) => ({ key: k, value: v })),
      annotations: Object.entries(rule.annotations || {}).map(([k, v]) => ({ key: k, value: v })),
      group_name: rule.group_name,
      category: rule.category || '',
    })
  } else if (key === 'delete') {
    if (window.confirm(t('alert.deleteRuleConfirm'))) handleDelete(rule.id)
  }
}

function goDetail(rule: AlertRule) {
  router.push(`/alerts/rules/${rule.id}`).catch(() => { /* fallback no-op */ })
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
        <n-button size="small" secondary @click="showImportExport = true">
          <template #icon><n-icon :component="CloudUploadOutline" /></template>
          {{ t('alert.importExport') }}
        </n-button>
        <n-button size="small" type="primary" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('alert.createRule') }}
        </n-button>
      </template>
    </PageHeader>

    <div class="rules-layout">
      <!-- Sidebar: categories -->
      <aside class="cat-aside">
        <div class="sre-label-eyebrow cat-eyebrow">{{ t('alert.category') }}</div>
        <a
          class="cat-item"
          :class="{ active: activeCategory === '' }"
          @click="handleCategoryChange('')"
        >
          <span class="cat-name">{{ t('alert.allCategories') }}</span>
          <span class="cat-count tnum">{{ allCount }}</span>
        </a>
        <a
          v-for="cat in categories"
          :key="cat"
          class="cat-item"
          :class="{ active: activeCategory === cat }"
          @click="handleCategoryChange(cat)"
        >
          <span class="cat-name">{{ cat }}</span>
          <span class="cat-count tnum">{{ categoryCounts[cat] ?? '' }}</span>
        </a>
      </aside>

      <!-- Main column -->
      <section class="rules-main">
        <!-- Toolbar -->
        <div class="toolbar">
          <n-input
            v-model:value="searchKeyword"
            size="small"
            :placeholder="t('common.search')"
            clearable
            class="toolbar-search"
          >
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-select
            v-model:value="filterDatasource"
            size="small"
            :options="datasourceOptions"
            :placeholder="t('alert.dataSource')"
            clearable
            class="toolbar-select"
          />
          <n-select
            v-model:value="filterSeverity"
            size="small"
            :options="severityFilterOptions"
            :placeholder="t('alert.severity')"
            clearable
            class="toolbar-select"
          />
          <n-select
            v-model:value="filterStatus"
            size="small"
            :options="statusFilterOptions"
            :placeholder="t('common.status')"
            clearable
            class="toolbar-select"
          />
          <div style="flex: 1"></div>
          <label class="select-all-label">
            <input
              type="checkbox"
              :checked="allSelected"
              @change="toggleSelectAll(($event.target as HTMLInputElement).checked)"
            />
            <span>{{ t('common.selectAll', '全选') }}</span>
          </label>
        </div>

        <!-- Selection bar -->
        <div v-if="selectedKeys.length > 0" class="selection-bar">
          <span class="sel-text tnum">{{ selectedKeys.length }} {{ t('common.selected', { count: selectedKeys.length }) }}</span>
          <n-button size="small" secondary :loading="batchLoading" @click="handleBatchEnable">
            {{ t('common.enabled') }}
          </n-button>
          <n-button size="small" secondary :loading="batchLoading" @click="handleBatchDisable">
            {{ t('common.disabled') }}
          </n-button>
          <n-popconfirm @positive-click="handleBatchDelete">
            <template #trigger>
              <n-button size="small" tertiary type="error" :loading="batchLoading">
                {{ t('common.delete') }}
              </n-button>
            </template>
            {{ t('alert.batchDeleteConfirm', { count: selectedKeys.length }) }}
          </n-popconfirm>
          <div style="flex: 1"></div>
          <n-button size="small" quaternary circle @click="selectedKeys = []">
            <template #icon><n-icon :component="CloseOutline" /></template>
          </n-button>
        </div>

        <!-- Loading skeleton -->
        <LoadingSkeleton v-if="loading && filteredRules.length === 0" :rows="6" variant="row" />

        <!-- Empty state -->
        <EmptyState
          v-else-if="!loading && filteredRules.length === 0"
          :icon="DocumentTextOutline"
          title="No alert rules"
          description="Create your first rule to start monitoring"
          :primary-text="t('alert.createFirstRule')"
          :secondary-text="t('alert.importFile')"
          @primary="openCreate"
          @secondary="showImportExport = true"
        />

        <!-- Rule list -->
        <div v-else class="rule-list" :class="{ 'sre-stagger': isFirstLoad }">
          <div
            v-for="rule in filteredRules"
            :key="rule.id"
            class="sre-row-card rule-row"
            :data-severity="severitySlot(rule.severity)"
            :data-dim="rule.status !== 'enabled' || undefined"
            @click="goDetail(rule)"
          >
            <input
              type="checkbox"
              class="rc-check"
              :checked="isSelected(rule.id)"
              @click.stop
              @change="toggleSelect(rule.id, ($event.target as HTMLInputElement).checked)"
            />
            <div class="rc-main">
              <div class="rc-title">
                <span class="rc-name">{{ rule.display_name || rule.name }}</span>
                <span class="rc-id tnum">#{{ rule.id }}</span>
              </div>
              <div class="rc-expr">{{ rule.expression }}</div>
              <div class="rc-meta">
                <span class="rc-meta-item">
                  <span class="sre-dot" :data-severity="severitySlot(rule.severity)"></span>
                  {{ severityLabel(rule.severity) }}
                </span>
                <span class="sre-meta-divider"></span>
                <span class="rc-meta-item">{{ rule.datasource?.name || '—' }}</span>
                <template v-if="rule.category">
                  <span class="sre-meta-divider"></span>
                  <span class="rc-meta-item">{{ rule.category }}</span>
                </template>
                <template v-if="rule.for_duration">
                  <span class="sre-meta-divider"></span>
                  <span class="rc-meta-item tnum">for {{ rule.for_duration }}</span>
                </template>
              </div>
            </div>
            <div class="rc-toggle" @click.stop>
              <n-switch :value="rule.status === 'enabled'" size="small" @update:value="toggleEnabled(rule)" />
            </div>
            <div class="rc-actions" @click.stop>
              <n-dropdown :options="rowActions(rule)" trigger="click" @select="(k: string) => onRowAction(k, rule)">
                <n-button quaternary circle size="small">
                  <template #icon><n-icon :component="EllipsisHorizontalOutline" /></template>
                </n-button>
              </n-dropdown>
            </div>
          </div>
        </div>

        <!-- Pagination -->
        <div v-if="filteredRules.length > 0" class="pagination-wrap">
          <n-pagination
            v-model:page="page"
            :page-size="pageSize"
            :item-count="total"
            :page-slot="7"
            @update:page="fetchRules"
          />
        </div>
      </section>
    </div>

    <!-- Create/Edit Modal -->
    <n-modal v-model:show="showModal" preset="card" :title="modalTitle" style="width: 680px" :bordered="false">
      <div v-if="!editingId" style="margin-bottom: 16px; display: flex; gap: 8px; align-items: center">
        <n-button size="small" secondary @click="openTemplatePicker">
          {{ t('alert.loadFromTemplate', 'Load from template') }}
        </n-button>
        <span v-if="appliedTemplateId" style="font-size: 12px; color: var(--sre-text-tertiary)">
          {{ t('alert.templateApplied', 'Template applied') }}
        </span>
      </div>

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
        <div class="tpl-list">
          <div v-if="templateLoading" class="tpl-loading">{{ t('common.loading', 'Loading…') }}</div>
          <div v-else-if="templates.length === 0" class="tpl-loading">{{ t('common.noData', '暂无数据') }}</div>
          <div
            v-for="tpl in templates"
            :key="tpl.id"
            class="sre-row-card tpl-row"
            :data-severity="severitySlot(tpl.severity || 'info')"
          >
            <div class="rc-main">
              <div class="rc-title">
                <span class="rc-name">{{ tpl.name }}</span>
              </div>
              <div class="rc-meta">
                <span class="rc-meta-item">
                  <span class="sre-dot" :data-severity="severitySlot(tpl.severity || 'info')"></span>
                  {{ severityLabel(tpl.severity || 'info') }}
                </span>
                <template v-if="tpl.category">
                  <span class="sre-meta-divider"></span>
                  <span class="rc-meta-item">{{ tpl.category }}</span>
                </template>
              </div>
            </div>
            <n-button size="tiny" secondary @click="loadTemplate(tpl)">{{ t('common.apply') }}</n-button>
          </div>
        </div>
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
          <n-gi>
            <n-form-item :label="t('alert.category')">
              <n-select v-model:value="form.category" :options="categoryOptions" :placeholder="t('alert.selectCategory')" clearable tag filterable />
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

        <n-form-item required>
          <template #label>
            <span class="sre-label-eyebrow" style="display:inline-flex; align-items:center; gap:6px">
              {{ t('alert.expression') }} <span class="lang-pill">{{ expressionLang }}</span>
            </span>
          </template>
          <div style="width: 100%">
            <n-input
              v-model:value="form.expression"
              type="textarea"
              :placeholder="expressionPlaceholder"
              :rows="3"
              style="font-family: var(--sre-font-mono, monospace)"
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
                <div class="sre-label-eyebrow" style="margin-bottom: 8px">{{ t('alert.testResult') }}</div>
                <div v-if="queryResult?.result_type === 'logs'" style="font-size: 13px; color: var(--sre-text-secondary)">
                  {{ t('alert.matchedLogs') }}: <span class="tnum">{{ queryResult.raw_count }}</span>
                </div>
                <div v-else-if="queryResult?.series && queryResult.series.length > 0" class="series-list">
                  <div v-for="(s, i) in queryResult.series" :key="i" class="series-row">
                    <code class="series-labels">{{ Object.entries(s.labels || {}).map(([k, v]) => `${k}=${v}`).join(', ') }}</code>
                    <span class="series-value tnum">{{ s.values?.[0]?.value ?? '-' }}</span>
                  </div>
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

        <n-form-item :label="t('alert.labels')">
          <KVEditor v-model:modelValue="form.labels" :add-label="t('alert.addLabel')" />
        </n-form-item>

        <n-form-item :label="t('alert.annotations')">
          <KVEditor v-model:modelValue="form.annotations" :add-label="t('alert.addAnnotation')" key-placeholder="Key (e.g. summary)" />
        </n-form-item>
      </n-form>

      <template #action>
        <div style="display:flex; justify-content:space-between; width:100%">
          <n-button size="small" secondary @click="saveAsTemplate">
            {{ t('alert.saveAsTemplate', 'Save as template') }}
          </n-button>
          <div style="display:flex; gap:8px">
            <n-button size="small" @click="showModal = false">{{ t('common.cancel') }}</n-button>
            <n-button size="small" type="primary" :loading="saving" @click="handleSave">
              {{ editingId ? t('common.update') : t('common.create') }}
            </n-button>
          </div>
        </div>
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
              <n-button type="primary" block :loading="importing" :disabled="!importFile" @click="handleImport">
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
              <n-button type="primary" block @click="handleExport">
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
  font-family: var(--sre-font-sans);
}

.rules-layout {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 24px;
  margin-top: 16px;
  align-items: start;
}

/* Sidebar */
.cat-aside {
  background: var(--sre-bg-card);
  border-right: var(--sre-hairline);
  border-radius: 8px 0 0 8px;
  padding: 16px 0;
  position: sticky;
  top: 16px;
}
.cat-eyebrow {
  padding: 0 16px 8px;
  color: var(--sre-text-tertiary);
}
.cat-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  font-size: 13px;
  color: var(--sre-text-secondary);
  cursor: pointer;
  position: relative;
  transition: background 120ms ease, color 120ms ease;
  border-left: 2px solid transparent;
}
.cat-item:hover {
  color: var(--sre-text-primary);
  background: var(--sre-bg-hover, rgba(255,255,255,0.03));
}
.cat-item.active {
  color: var(--sre-primary);
  background: var(--sre-primary-soft);
  border-left-color: var(--sre-primary);
  font-weight: 500;
}
.cat-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cat-count {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  font-family: var(--sre-font-mono, monospace);
}
.cat-item.active .cat-count {
  color: var(--sre-primary);
}

/* Main */
.rules-main {
  min-width: 0;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 0;
  margin-bottom: 4px;
}
.toolbar-search { width: 240px; }
.toolbar-select { width: 160px; }

.select-all-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  user-select: none;
}
.select-all-label input {
  accent-color: var(--sre-primary);
}

/* Selection bar */
.selection-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--sre-primary-soft);
  border-radius: 8px;
  margin-bottom: 16px;
}
.sel-text {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-primary);
}

/* Empty */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 24px;
  text-align: center;
  background: var(--sre-bg-card);
  border-radius: 8px;
  border: var(--sre-hairline);
}
.empty-icon { color: var(--sre-text-tertiary); opacity: 0.5; }
.empty-title {
  margin-top: 16px;
  font-size: 14px;
  color: var(--sre-text-secondary);
}
.empty-actions {
  margin-top: 20px;
  display: flex;
  gap: 8px;
}

/* List */
.rule-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.rule-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 16px 14px 20px;
  cursor: pointer;
}
.rc-check {
  width: 14px;
  height: 14px;
  cursor: pointer;
  flex-shrink: 0;
  align-self: center;
  accent-color: var(--sre-primary);
}
.rc-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.rc-title {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.rc-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rc-id {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-tertiary);
}
.rc-expr {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rc-meta {
  display: flex;
  align-items: center;
  gap: 0;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  flex-wrap: wrap;
  row-gap: 4px;
}
.rc-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.rc-toggle, .rc-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

/* Dimmed (disabled) rows */
.sre-row-card[data-dim] {
  opacity: 0.55;
}
.sre-row-card[data-dim] .rc-name {
  color: var(--sre-text-secondary);
}

/* Pagination */
.pagination-wrap {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}

/* Template picker */
.tpl-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 360px;
  overflow-y: auto;
}
.tpl-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px 12px 20px;
}
.tpl-loading {
  padding: 24px;
  text-align: center;
  font-size: 13px;
  color: var(--sre-text-tertiary);
}

/* Expression result */
.lang-pill {
  display: inline-flex;
  align-items: center;
  padding: 1px 6px;
  border-radius: 3px;
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.3px;
}
.query-result {
  margin-top: 12px;
  padding: 12px 14px;
  background: var(--sre-bg-card);
  border-radius: 6px;
  border: var(--sre-hairline);
}
.series-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 200px;
  overflow-y: auto;
}
.series-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 4px 0;
  border-bottom: var(--sre-hairline);
}
.series-row:last-child { border-bottom: none; }
.series-labels {
  font-family: var(--sre-font-mono, monospace);
  font-size: 11px;
  color: var(--sre-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}
.series-value {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-primary);
  font-weight: 500;
}
</style>
