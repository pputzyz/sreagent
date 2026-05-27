<script setup lang="ts">
import { ref, computed, onMounted, h, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import {
  NButton, NSpace, NDataTable, NInput, NSelect, NDrawer, NDrawerContent,
  NForm, NFormItem, NTag, NSwitch, NIcon, NInputNumber, NEmpty, NPagination,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { AddOutline, TrashOutline, CreateOutline, SearchOutline, PlayOutline } from '@vicons/ionicons5'
import { alertRuleTemplateApi } from '@/api/alert-rule-template'
import type { AlertRuleTemplate } from '@/api/alert-rule-template'
import { usePaginatedList, usePermissions } from '@/composables'
import { getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const { hasPerm } = usePermissions()

// Filters
const searchQuery = ref('')
const filterCategory = ref<string | null>(null)
const categories = ref<string[]>([])

// Pagination & data
const {
  loading,
  items: templates,
  total,
  page,
  pageSize,
  fetchList,
  refresh,
} = usePaginatedList<AlertRuleTemplate>({
  apiFn: alertRuleTemplateApi.list,
  pageSize: 20,
  extraParams: () => {
    const params: Record<string, unknown> = {}
    if (filterCategory.value) params.category = filterCategory.value
    if (searchQuery.value.trim()) params.search = searchQuery.value.trim()
    return params
  },
  onError: (err: unknown) => {
    message.error(getErrorMessage(err))
  },
})

// Debounced search
let searchTimer: ReturnType<typeof setTimeout> | null = null
watch(searchQuery, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { page.value = 1; fetchList() }, 300)
})
watch(filterCategory, () => {
  page.value = 1
  fetchList()
})

// Category options
const categoryOptions = computed(() => [
  { label: t('common.all'), value: '' },
  ...categories.value.map(c => ({ label: c, value: c })),
])

// Severity options
const severityOptions = [
  { label: 'Critical', value: 'critical' },
  { label: 'Warning', value: 'warning' },
  { label: 'Info', value: 'info' },
]

// Datasource type options
const datasourceTypeOptions = [
  { label: 'Prometheus', value: 'prometheus' },
  { label: 'VictoriaMetrics', value: 'victoriametrics' },
  { label: 'Elasticsearch', value: 'elasticsearch' },
  { label: 'Zabbix', value: 'zabbix' },
]

// Severity color map
function severityType(sev: string): 'error' | 'warning' | 'info' | 'default' {
  if (sev === 'critical') return 'error'
  if (sev === 'warning') return 'warning'
  if (sev === 'info') return 'info'
  return 'default'
}

// Drawer state
const showDrawer = ref(false)
const drawerMode = ref<'create' | 'edit'>('create')
const saving = ref(false)
const editingId = ref<number | null>(null)
const form = ref({
  name: '',
  category: '',
  description: '',
  datasource_type: 'prometheus',
  expression: '',
  for_duration: '0s',
  severity: 'warning',
  labels: {} as Record<string, string>,
  annotations: {} as Record<string, string>,
  group_name: '',
  eval_interval: 60,
  no_data_enabled: false,
  no_data_duration: '5m',
  ack_sla_minutes: 0,
})

// KV editor state for labels/annotations
const labelPairs = ref<Array<{ key: string; value: string }>>([])
const annotationPairs = ref<Array<{ key: string; value: string }>>([])

function pairsToRecord(pairs: Array<{ key: string; value: string }>): Record<string, string> {
  const result: Record<string, string> = {}
  for (const p of pairs) {
    if (p.key.trim()) result[p.key.trim()] = p.value
  }
  return result
}

function recordToPairs(record: Record<string, string>): Array<{ key: string; value: string }> {
  return Object.entries(record || {}).map(([key, value]) => ({ key, value }))
}

// Apply dialog state
const showApplyDialog = ref(false)
const applyTemplate = ref<AlertRuleTemplate | null>(null)
const applyLoading = ref(false)

// Table columns
const columns = computed<DataTableColumns<AlertRuleTemplate>>(() => [
  {
    title: t('common.name'),
    key: 'name',
    ellipsis: { tooltip: true },
    minWidth: 180,
  },
  {
    title: t('ruleTemplates.category'),
    key: 'category',
    width: 120,
    render: (row) => row.category
      ? h(NTag, { size: 'small', bordered: false, round: true }, { default: () => row.category })
      : h('span', { style: 'color: var(--sre-text-tertiary)' }, '—'),
  },
  {
    title: t('ruleTemplates.datasourceType'),
    key: 'datasource_type',
    width: 140,
    render: (row) => h(NTag, { size: 'small', bordered: false, round: true, type: 'default' }, { default: () => row.datasource_type }),
  },
  {
    title: t('ruleTemplates.severity'),
    key: 'severity',
    width: 100,
    render: (row) => h(NTag, {
      size: 'small',
      bordered: false,
      round: true,
      type: severityType(row.severity),
    }, { default: () => row.severity }),
  },
  {
    title: t('ruleTemplates.expression'),
    key: 'expression',
    ellipsis: { tooltip: true },
    minWidth: 200,
    render: (row) => h('span', { style: 'font-family: var(--sre-font-mono, monospace); font-size: 12px; color: var(--sre-text-secondary)' }, row.expression),
  },
  {
    title: t('ruleTemplates.isBuiltin'),
    key: 'is_builtin',
    width: 80,
    render: (row) => row.is_builtin
      ? h(NTag, { type: 'warning', size: 'small', bordered: false, round: true }, { default: () => t('common.yes') })
      : h('span', { style: 'color: var(--sre-text-tertiary)' }, t('common.no')),
  },
  {
    title: t('ruleTemplates.usageCount'),
    key: 'usage_count',
    width: 100,
    render: (row) => h('span', { class: 'tnum' }, row.usage_count),
  },
  {
    title: t('common.createdAt'),
    key: 'created_at',
    width: 160,
    render: (row) => h('span', { class: 'tnum', style: 'font-size: 12px; color: var(--sre-text-tertiary)' }, row.created_at ? new Date(row.created_at).toLocaleString() : '—'),
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 120,
    fixed: 'right',
    render: (row) => h(NSpace, { size: 'small', justify: 'center' }, {
      default: () => [
        h(NButton, {
          quaternary: true,
          circle: true,
          size: 'small',
          title: t('ruleTemplates.apply'),
          disabled: !hasPerm('rules.manage'),
          onClick: (e: MouseEvent) => { e.stopPropagation(); openApply(row) },
        }, { icon: () => h(NIcon, { component: PlayOutline }) }),
        h(NButton, {
          quaternary: true,
          circle: true,
          size: 'small',
          title: t('common.edit'),
          disabled: row.is_builtin || !hasPerm('rules.manage'),
          onClick: (e: MouseEvent) => { e.stopPropagation(); openEdit(row) },
        }, { icon: () => h(NIcon, { component: CreateOutline }) }),
        h(NButton, {
          quaternary: true,
          circle: true,
          size: 'small',
          title: t('common.delete'),
          type: 'error',
          disabled: row.is_builtin || !hasPerm('rules.manage'),
          onClick: (e: MouseEvent) => { e.stopPropagation(); handleDelete(row) },
        }, { icon: () => h(NIcon, { component: TrashOutline }) }),
      ],
    }),
  },
])

// Fetch categories
async function fetchCategories() {
  try {
    const resp = await alertRuleTemplateApi.listCategories()
    categories.value = resp.data.data || []
  } catch (e) {
    console.warn('[RuleTemplates] Failed to fetch categories:', e)
  }
}

// Drawer operations
function resetForm() {
  form.value = {
    name: '',
    category: '',
    description: '',
    datasource_type: 'prometheus',
    expression: '',
    for_duration: '0s',
    severity: 'warning',
    labels: {},
    annotations: {},
    group_name: '',
    eval_interval: 60,
    no_data_enabled: false,
    no_data_duration: '5m',
    ack_sla_minutes: 0,
  }
  labelPairs.value = []
  annotationPairs.value = []
}

function openCreate() {
  resetForm()
  drawerMode.value = 'create'
  showDrawer.value = true
}

function openEdit(row: AlertRuleTemplate) {
  form.value = {
    name: row.name,
    category: row.category || '',
    description: row.description || '',
    datasource_type: row.datasource_type || 'prometheus',
    expression: row.expression,
    for_duration: row.for_duration || '0s',
    severity: row.severity || 'warning',
    labels: row.labels || {},
    annotations: row.annotations || {},
    group_name: row.group_name || '',
    eval_interval: row.eval_interval || 60,
    no_data_enabled: row.no_data_enabled || false,
    no_data_duration: row.no_data_duration || '5m',
    ack_sla_minutes: row.ack_sla_minutes || 0,
  }
  labelPairs.value = recordToPairs(row.labels)
  annotationPairs.value = recordToPairs(row.annotations)
  drawerMode.value = 'edit'
  editingId.value = row.id
  showDrawer.value = true
}

async function handleSave() {
  if (!form.value.name.trim()) {
    message.warning(t('alert.nameRequired'))
    return
  }
  if (!form.value.expression.trim()) {
    message.warning(t('alert.expressionRequired'))
    return
  }
  saving.value = true
  try {
    const payload = {
      ...form.value,
      name: form.value.name.trim(),
      description: form.value.description.trim() || undefined,
      category: form.value.category.trim() || undefined,
      group_name: form.value.group_name.trim() || undefined,
      labels: pairsToRecord(labelPairs.value),
      annotations: pairsToRecord(annotationPairs.value),
    }
    if (drawerMode.value === 'create') {
      await alertRuleTemplateApi.create(payload)
      message.success(t('common.createSuccess'))
    } else {
      await alertRuleTemplateApi.update(editingId.value!, payload)
      message.success(t('common.updateSuccess'))
    }
    showDrawer.value = false
    fetchList()
    fetchCategories()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

function handleDelete(row: AlertRuleTemplate) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('common.confirmDeleteMsg'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await alertRuleTemplateApi.delete(row.id)
        message.success(t('common.deleteSuccess'))
        fetchList()
      } catch (err: unknown) {
        message.error(getErrorMessage(err))
      }
    },
  })
}

// Apply
function openApply(row: AlertRuleTemplate) {
  applyTemplate.value = row
  showApplyDialog.value = true
}

async function handleApply() {
  if (!applyTemplate.value) return
  applyLoading.value = true
  try {
    await alertRuleTemplateApi.apply(applyTemplate.value.id)
    message.success(t('alert.templateApplied'))
    showApplyDialog.value = false
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    applyLoading.value = false
  }
}

// Label/Annotation KV helpers
function addLabelPair() {
  labelPairs.value.push({ key: '', value: '' })
}
function removeLabelPair(index: number) {
  labelPairs.value.splice(index, 1)
}
function addAnnotationPair() {
  annotationPairs.value.push({ key: '', value: '' })
}
function removeAnnotationPair(index: number) {
  annotationPairs.value.splice(index, 1)
}

onMounted(() => {
  fetchList()
  fetchCategories()
})
</script>

<template>
  <div class="rule-templates-page">
    <PageHeader :title="t('ruleTemplates.title')" :subtitle="t('ruleTemplates.subtitle')">
      <template #actions>
        <n-button v-if="hasPerm('rules.manage')" size="small" type="primary" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('common.create') }}
        </n-button>
      </template>
    </PageHeader>

    <!-- Toolbar -->
    <div class="toolbar">
      <n-input
        v-model:value="searchQuery"
        size="small"
        :placeholder="t('common.search')"
        clearable
        class="toolbar-search"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select
        v-model:value="filterCategory"
        size="small"
        :options="categoryOptions"
        :placeholder="t('ruleTemplates.category')"
        clearable
        class="toolbar-select"
      />
    </div>

    <!-- Empty state -->
    <NEmpty
      v-if="!loading && templates.length === 0"
      :description="t('common.noData')"
      style="margin-top: 80px"
    />

    <!-- Data table -->
    <n-data-table
      v-else
      :columns="columns"
      :data="templates"
      :loading="loading"
      :bordered="false"
      :single-line="false"
      size="small"
      :row-key="(row: AlertRuleTemplate) => row.id"
      style="margin-top: 8px"
    />

    <!-- Pagination -->
    <div v-if="total > 0" class="pagination-wrap">
      <n-pagination
        v-model:page="page"
        :page-size="pageSize"
        :item-count="total"
        :page-slot="7"
        @update:page="fetchList"
      />
    </div>

    <!-- Create/Edit Drawer -->
    <n-drawer v-model:show="showDrawer" :width="520" placement="right">
      <n-drawer-content :title="drawerMode === 'create' ? t('common.create') : t('common.edit')">
        <n-form label-placement="top" size="small">
          <n-form-item :label="t('common.name')" required>
            <n-input
              v-model:value="form.name"
              :placeholder="t('common.name')"
              maxlength="200"
              show-count
            />
          </n-form-item>
          <n-form-item :label="t('ruleTemplates.category')">
            <n-input
              v-model:value="form.category"
              :placeholder="t('ruleTemplates.category')"
            />
          </n-form-item>
          <n-form-item :label="t('common.description')">
            <n-input
              v-model:value="form.description"
              type="textarea"
              :placeholder="t('common.description')"
              :rows="2"
            />
          </n-form-item>
          <n-form-item :label="t('ruleTemplates.datasourceType')" required>
            <n-select
              v-model:value="form.datasource_type"
              :options="datasourceTypeOptions"
            />
          </n-form-item>
          <n-form-item :label="t('ruleTemplates.expression')" required>
            <n-input
              v-model:value="form.expression"
              type="textarea"
              :placeholder="'avg(rate(http_requests_total[5m])) > 100'"
              :rows="4"
              style="font-family: var(--sre-font-mono, monospace)"
            />
          </n-form-item>
          <n-form-item :label="t('ruleTemplates.forDuration')">
            <n-input
              v-model:value="form.for_duration"
              :placeholder="'0s, 5m, 1h'"
            />
          </n-form-item>
          <n-form-item :label="t('ruleTemplates.severity')" required>
            <n-select
              v-model:value="form.severity"
              :options="severityOptions"
            />
          </n-form-item>
          <n-form-item :label="t('ruleTemplates.groupName')">
            <n-input
              v-model:value="form.group_name"
              :placeholder="t('alert.groupNamePlaceholder')"
            />
          </n-form-item>
          <n-form-item :label="t('ruleTemplates.evalInterval')">
            <n-input-number
              v-model:value="form.eval_interval"
              :min="1"
              :max="86400"
              style="width: 100%"
            />
          </n-form-item>

          <!-- Labels KV editor -->
          <n-form-item :label="t('alert.labels')">
            <div class="kv-editor">
              <div v-for="(pair, idx) in labelPairs" :key="idx" class="kv-row">
                <n-input v-model:value="pair.key" size="small" :placeholder="t('common.key')" class="kv-key" />
                <n-input v-model:value="pair.value" size="small" :placeholder="t('common.value')" class="kv-value" />
                <n-button quaternary circle size="small" type="error" @click="removeLabelPair(idx)">
                  <template #icon><n-icon :component="TrashOutline" /></template>
                </n-button>
              </div>
              <n-button size="tiny" dashed @click="addLabelPair">{{ t('alert.addLabel') }}</n-button>
            </div>
          </n-form-item>

          <!-- Annotations KV editor -->
          <n-form-item :label="t('alert.annotations')">
            <div class="kv-editor">
              <div v-for="(pair, idx) in annotationPairs" :key="idx" class="kv-row">
                <n-input v-model:value="pair.key" size="small" :placeholder="t('common.key')" class="kv-key" />
                <n-input v-model:value="pair.value" size="small" :placeholder="t('common.value')" class="kv-value" />
                <n-button quaternary circle size="small" type="error" @click="removeAnnotationPair(idx)">
                  <template #icon><n-icon :component="TrashOutline" /></template>
                </n-button>
              </div>
              <n-button size="tiny" dashed @click="addAnnotationPair">{{ t('alert.addAnnotation') }}</n-button>
            </div>
          </n-form-item>

          <!-- Advanced settings -->
          <n-form-item :label="t('ruleTemplates.noDataEnabled')">
            <n-switch v-model:value="form.no_data_enabled" />
          </n-form-item>
          <n-form-item v-if="form.no_data_enabled" :label="t('ruleTemplates.noDataDuration')">
            <n-input
              v-model:value="form.no_data_duration"
              :placeholder="'5m'"
            />
          </n-form-item>
          <n-form-item :label="t('ruleTemplates.ackSlaMinutes')">
            <n-input-number
              v-model:value="form.ack_sla_minutes"
              :min="0"
              style="width: 100%"
            />
          </n-form-item>
        </n-form>
        <template #footer>
          <n-space justify="end">
            <n-button size="small" @click="showDrawer = false">{{ t('common.cancel') }}</n-button>
            <n-button size="small" type="primary" :loading="saving" @click="handleSave">
              {{ t('common.save') }}
            </n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>

    <!-- Apply Dialog -->
    <n-drawer v-model:show="showApplyDialog" :width="480" placement="right">
      <n-drawer-content :title="t('ruleTemplates.apply')">
        <template v-if="applyTemplate">
          <div class="apply-detail">
            <div class="apply-field">
              <span class="apply-label">{{ t('common.name') }}</span>
              <span class="apply-value">{{ applyTemplate.name }}</span>
            </div>
            <div class="apply-field">
              <span class="apply-label">{{ t('ruleTemplates.category') }}</span>
              <span class="apply-value">{{ applyTemplate.category || '—' }}</span>
            </div>
            <div class="apply-field">
              <span class="apply-label">{{ t('ruleTemplates.datasourceType') }}</span>
              <span class="apply-value">{{ applyTemplate.datasource_type }}</span>
            </div>
            <div class="apply-field">
              <span class="apply-label">{{ t('ruleTemplates.severity') }}</span>
              <NTag :type="severityType(applyTemplate.severity)" size="small" bordered round>
                {{ applyTemplate.severity }}
              </NTag>
            </div>
            <div class="apply-field">
              <span class="apply-label">{{ t('ruleTemplates.expression') }}</span>
              <code class="apply-expr">{{ applyTemplate.expression }}</code>
            </div>
            <div v-if="applyTemplate.description" class="apply-field">
              <span class="apply-label">{{ t('common.description') }}</span>
              <span class="apply-value">{{ applyTemplate.description }}</span>
            </div>
          </div>
          <p style="margin-top: 16px; color: var(--sre-text-secondary); font-size: 13px;">
            {{ t('ruleTemplates.applyConfirm') }}
          </p>
        </template>
        <template #footer>
          <n-space justify="end">
            <n-button size="small" @click="showApplyDialog = false">{{ t('common.cancel') }}</n-button>
            <n-button size="small" type="primary" :loading="applyLoading" @click="handleApply">
              {{ t('common.confirm') }}
            </n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<style scoped>
.rule-templates-page {
  max-width: 1400px;
  font-family: var(--sre-font-sans);
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 0;
  margin-bottom: 4px;
}

.toolbar-search {
  width: 240px;
}

.toolbar-select {
  width: 180px;
}

.pagination-wrap {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.tnum {
  font-variant-numeric: tabular-nums;
}

/* KV editor */
.kv-editor {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.kv-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.kv-key {
  flex: 1;
}

.kv-value {
  flex: 2;
}

/* Apply detail */
.apply-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.apply-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.apply-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--sre-text-tertiary);
}

.apply-value {
  font-size: 13px;
  color: var(--sre-text-primary);
}

.apply-expr {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-secondary);
  background: var(--sre-bg-elevated, rgba(255,255,255,0.04));
  padding: 8px 12px;
  border-radius: 6px;
  word-break: break-all;
}
</style>
