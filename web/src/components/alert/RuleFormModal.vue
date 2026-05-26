<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import {
  useMessage, NModal, NButton, NIcon, NForm, NFormItem, NGrid, NGi,
  NInput, NInputNumber, NSelect, NCollapseTransition, NSwitch, NCollapse, NCollapseItem,
  NCard, NDrawer, NDrawerContent, NSpin, NTabs, NTabPane, NDataTable,
} from 'naive-ui'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import {
  TooltipComponent, LegendComponent, GridComponent, DataZoomComponent,
} from 'echarts/components'
import VChart from 'vue-echarts'
import { useI18n } from 'vue-i18n'
import { alertRuleApi, datasourceApi, templateApi, labelRegistryApi } from '@/api'
import type { AlertRule, AlertRuleType, DataSource, AlertSeverity, DataSourceType, QueryResponse } from '@/types'
import { kvArrayToRecord, getErrorMessage } from '@/utils/format'
import { formatValue } from '@/utils/valueFormatter'
import KVEditor from '@/components/common/KVEditor.vue'
import { PlayOutline, StatsChartOutline } from '@vicons/ionicons5'

use([CanvasRenderer, LineChart, TooltipComponent, LegendComponent, GridComponent, DataZoomComponent])

const props = defineProps<{
  show: boolean
  rule: AlertRule | null
  datasources: DataSource[]
  duplicateFrom?: AlertRule | null
  initialExpr?: string
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const message = useMessage()
const { t } = useI18n()

const modalTitle = ref('')
const editingId = ref<number | null>(null)
const saving = ref(false)

// Collapsible section state
const sectionBasicOpen = ref(true)
const sectionQueryOpen = ref(true)
const sectionLabelsOpen = ref(false)
const sectionAdvancedOpen = ref(false)

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
  // Advanced fields
  rule_type: 'threshold' as string,
  eval_interval: 60,
  recovery_hold: '0s',
  nodata_enabled: false,
  nodata_duration: '5m',
  suppress_enabled: false,
  heartbeat_token: '',
  heartbeat_interval: 300,
  ack_sla_minutes: 0,
}
const form = reactive({ ...defaultForm })

// Label registry autocomplete
const labelKeys = ref<string[]>([])
const labelValues = ref<string[]>([])

async function fetchLabelKeys() {
  try {
    const res = await labelRegistryApi.getKeys(form.datasource_id ?? undefined)
    labelKeys.value = res.data?.data || []
  } catch { labelKeys.value = [] }
}

async function fetchLabelValues(key: string) {
  if (!key) { labelValues.value = []; return }
  try {
    const res = await labelRegistryApi.getValues(key, form.datasource_id ?? undefined)
    labelValues.value = res.data?.data || []
  } catch { labelValues.value = [] }
}

function onLabelKeyChange(_idx: number, key: string) {
  fetchLabelValues(key)
}

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

const categoryOptions = [
  { label: () => t('alert.categoryNode'), value: 'node' },
  { label: () => t('alert.categoryDatabase'), value: 'database' },
  { label: () => t('alert.categoryMiddleware'), value: 'middleware' },
  { label: () => t('alert.categoryNetwork'), value: 'network' },
  { label: () => t('alert.categoryApplication'), value: 'application' },
  { label: () => t('alert.categoryCustom'), value: 'custom' },
]

const datasourceTypeOptions = [
  { label: 'Prometheus', value: 'prometheus' },
  { label: 'VictoriaMetrics', value: 'victoriametrics' },
  { label: 'Zabbix', value: 'zabbix' },
  { label: 'VictoriaLogs', value: 'victorialogs' },
]

const datasourceOptions = computed(() =>
  props.datasources.map(ds => ({ label: `${ds.name} (${ds.type})`, value: ds.id })),
)

const selectedDatasource = computed(() =>
  props.datasources.find(ds => ds.id === form.datasource_id),
)

watch(() => form.datasource_id, (newId) => {
  if (newId != null) {
    const ds = props.datasources.find(d => d.id === newId)
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
  if (tp === 'victorialogs') return t('alert.expressionPlaceholderLog')
  if (tp === 'zabbix') return t('alert.expressionPlaceholderZabbix')
  return t('alert.expressionPlaceholderDefault')
})

// Expression test
const queryTesting = ref(false)
const queryResult = ref<QueryResponse | null>(null)

async function handleTestExpression() {
  if (!form.datasource_id || !form.expression.trim()) return
  queryTesting.value = true
  queryResult.value = null
  try {
    const { data } = await datasourceApi.query(form.datasource_id, { expression: form.expression })
    queryResult.value = data.data
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('common.failed'))
  } finally { queryTesting.value = false }
}

// Graph preview drawer
const showGraphDrawer = ref(false)
const graphLoading = ref(false)
const graphResult = ref<QueryResponse | null>(null)
const graphTimeRange = ref(3600) // seconds, default 1h

async function openGraphPreview() {
  if (!form.datasource_id || !form.expression.trim()) return
  showGraphDrawer.value = true
  graphLoading.value = true
  graphResult.value = null
  try {
    const end = Math.floor(Date.now() / 1000)
    const start = end - graphTimeRange.value
    const step = graphTimeRange.value <= 3600 ? '15s' : graphTimeRange.value <= 21600 ? '60s' : '300s'
    const { data } = await datasourceApi.rangeQuery(form.datasource_id, {
      expression: form.expression,
      start,
      end,
      step,
    })
    graphResult.value = data.data
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('common.failed'))
  } finally { graphLoading.value = false }
}

const graphChartOption = computed(() => {
  if (!graphResult.value?.series || graphResult.value.series.length === 0) return null
  const now = Math.floor(Date.now() / 1000)
  const start = now - graphTimeRange.value
  const allSeries = graphResult.value.series.map(s => ({
    name: Object.entries(s.labels || {}).filter(([k]) => k !== '__name__').map(([k, v]) => `${k}="${v}"`).join(', ') || 'value',
    type: 'line' as const,
    data: (s.values || []).map(v => [v.ts * 1000, v.value]),
    smooth: true,
    symbol: 'none',
    lineStyle: { width: 1.5 },
    emphasis: { lineStyle: { width: 2.5 } },
  }))
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' },
      valueFormatter: (val: number) => formatValue(val, 'short'),
    },
    legend: {
      type: 'scroll',
      bottom: 0,
      textStyle: { fontSize: 11 },
    },
    grid: { left: 60, right: 20, top: 30, bottom: 60 },
    xAxis: {
      type: 'time',
      min: start * 1000,
      max: now * 1000,
      axisLabel: {
        fontSize: 11,
        formatter: (val: number) => {
          const d = new Date(val)
          return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
        },
      },
    },
    yAxis: {
      type: 'value',
      axisLabel: { fontSize: 11, formatter: (val: number) => formatValue(val, 'short') },
      splitLine: { lineStyle: { type: 'dashed', color: 'var(--sre-border, #1e293b)' } },
    },
    series: allSeries,
    dataZoom: [
      { type: 'inside', xAxisIndex: 0 },
      { type: 'slider', xAxisIndex: 0, bottom: 25, height: 20 },
    ],
    animation: false,
  }
})

const graphLegendData = computed(() => {
  if (!graphResult.value?.series) return []
  return graphResult.value.series.map((s, i) => {
    const name = Object.entries(s.labels || {}).filter(([k]) => k !== '__name__').map(([k, v]) => `${k}="${v}"`).join(', ') || 'value'
    const values = (s.values || []).map(v => v.value)
    const min = values.length ? Math.min(...values) : 0
    const max = values.length ? Math.max(...values) : 0
    const avg = values.length ? values.reduce((a, b) => a + b, 0) / values.length : 0
    const last = values.length ? values[values.length - 1] : 0
    return { key: String(i), name, min, max, avg, last }
  })
})

const graphLegendColumns = computed(() => [
  { title: t('query.series'), key: 'name', ellipsis: { tooltip: true } },
  { title: t('query.min'), key: 'min', width: 80, align: 'right' as const, render: (row: { min: number }) => formatValue(row.min, 'short') },
  { title: t('query.max'), key: 'max', width: 80, align: 'right' as const, render: (row: { max: number }) => formatValue(row.max, 'short') },
  { title: t('query.avg'), key: 'avg', width: 80, align: 'right' as const, render: (row: { avg: number }) => formatValue(row.avg, 'short') },
  { title: t('query.last'), key: 'last', width: 80, align: 'right' as const, render: (row: { last: number }) => formatValue(row.last, 'short') },
])

const graphTimeRangeOptions = [
  { label: '15m', value: 900 },
  { label: '1h', value: 3600 },
  { label: '6h', value: 21600 },
  { label: '24h', value: 86400 },
]

// Templates
interface RuleTemplate {
  id: number
  name: string
  description: string
  datasource_type: string
  expression: string
  for_duration: string
  severity: string
  labels: Record<string, string>
  annotations: Record<string, string>
  group_name: string
  category: string
}

const showTemplatePicker = ref(false)
const templateLoading = ref(false)
const templates = ref<RuleTemplate[]>([])
const templateCategories = ref<string[]>([])
const templateSearch = ref('')
const templateCategory = ref('')
// Tracks which template was applied (used in template for "template applied" badge; value unused)
const appliedTemplateId = ref<number | null>(null)

function severitySlot(sev: string): 'critical' | 'warning' | 'info' | 'success' {
  if (sev === 'critical' || sev === 'p0' || sev === 'p1') return 'critical'
  if (sev === 'warning' || sev === 'p2') return 'warning'
  if (sev === 'info' || sev === 'p4') return 'info'
  return 'info'
}

function severityLabel(sev: string) {
  const map: Record<string, string> = {
    critical: t('alert.critical'),
    warning: t('alert.warning'),
    info: t('alert.info'),
    p0: t('alert.p0'), p1: t('alert.p1'), p2: t('alert.p2'), p3: t('alert.p3'), p4: t('alert.p4'),
  }
  return map[sev] || sev
}

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

async function loadTemplate(tpl: RuleTemplate) {
  appliedTemplateId.value = tpl.id
  Object.assign(form, {
    name: tpl.name || '',
    display_name: '',
    description: tpl.description || '',
    datasource_type: tpl.datasource_type || '',
    expression: tpl.expression || '',
    for_duration: tpl.for_duration || '5m',
    severity: tpl.severity || 'warning',
    labels: tpl.labels ? Object.entries(tpl.labels).map(([k, v]) => ({ key: k, value: v })) : [],
    annotations: tpl.annotations ? Object.entries(tpl.annotations).map(([k, v]) => ({ key: k, value: v })) : [],
    group_name: tpl.group_name || '',
    category: tpl.category || '',
  })
  showTemplatePicker.value = false
  message.success(t('alert.templateLoaded'))
}

async function saveAsTemplate() {
  const payload = {
    name: form.name,
    description: form.description,
    datasource_type: form.datasource_type || undefined,
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
    message.success(t('alert.templateSaved'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('common.saveFailed'))
  }
}

function openTemplatePicker() {
  fetchTemplates()
  showTemplatePicker.value = true
}

// Init form based on create/edit/duplicate mode
function formDataFromRule(r: AlertRule) {
  return {
    name: r.name,
    display_name: r.display_name,
    description: r.description,
    datasource_id: r.datasource_id,
    datasource_type: r.datasource_type || '',
    expression: r.expression,
    for_duration: r.for_duration,
    severity: r.severity,
    labels: Object.entries(r.labels || {}).map(([key, value]) => ({ key, value })),
    annotations: Object.entries(r.annotations || {}).map(([key, value]) => ({ key, value })),
    group_name: r.group_name,
    category: r.category || '',
    group_wait_seconds: r.group_wait_seconds || 0,
    group_interval_seconds: r.group_interval_seconds || 0,
    // Advanced fields
    rule_type: r.rule_type || 'threshold',
    eval_interval: r.eval_interval || 60,
    recovery_hold: r.recovery_hold || '0s',
    nodata_enabled: r.nodata_enabled || false,
    nodata_duration: r.nodata_duration || '5m',
    suppress_enabled: r.suppress_enabled || false,
    heartbeat_token: r.heartbeat_token || '',
    heartbeat_interval: r.heartbeat_interval || 300,
    ack_sla_minutes: r.ack_sla_minutes || 0,
  }
}

function initForm() {
  const dup = props.duplicateFrom
  if (props.rule) {
    editingId.value = props.rule.id
    modalTitle.value = t('alert.editRule')
    Object.assign(form, formDataFromRule(props.rule))
  } else if (dup) {
    editingId.value = null
    modalTitle.value = t('alert.createRule')
    Object.assign(form, {
      ...formDataFromRule(dup),
      name: dup.name + '_copy',
    })
  } else {
    editingId.value = null
    modalTitle.value = t('alert.createRule')
    Object.assign(form, { ...defaultForm, labels: [], annotations: [], expression: props.initialExpr || '' })
  }
  queryResult.value = null
  appliedTemplateId.value = null
}

watch(() => props.show, (val) => {
  if (val) { initForm(); fetchLabelKeys() }
})

async function handleSave() {
  if (!form.name.trim()) { message.warning(t('alert.nameRequired')); return }
  if (!form.expression.trim()) { message.warning(t('alert.expressionRequired')); return }
  if (form.datasource_id == null && !form.datasource_type) {
    message.warning(t('alert.datasourceRequired'))
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
      // Advanced fields
      rule_type: form.rule_type as AlertRuleType,
      eval_interval: form.eval_interval,
      recovery_hold: form.recovery_hold,
      nodata_enabled: form.nodata_enabled,
      nodata_duration: form.nodata_enabled ? form.nodata_duration : '',
      suppress_enabled: form.suppress_enabled,
      heartbeat_token: form.rule_type === 'heartbeat' ? form.heartbeat_token : '',
      heartbeat_interval: form.rule_type === 'heartbeat' ? form.heartbeat_interval : 0,
      ack_sla_minutes: form.ack_sla_minutes,
    }
    if (editingId.value) {
      await alertRuleApi.update(editingId.value, payload)
      message.success(t('alert.ruleUpdated'))
    } else {
      await alertRuleApi.create(payload)
      message.success(t('alert.ruleCreated'))
    }
    emit('saved')
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="modalTitle"
    class="rfm-modal"
    :bordered="false"
    @update:show="(v: boolean) => { if (!v) emit('close') }"
  >
    <div v-if="!editingId" class="rfm-template-bar">
      <n-button size="small" secondary @click="openTemplatePicker">
        {{ t('alert.loadFromTemplate') }}
      </n-button>
      <span v-if="appliedTemplateId" class="rfm-template-applied">
        {{ t('alert.templateApplied') }}
      </span>
    </div>

    <!-- Nested template picker -->
    <n-modal
      v-model:show="showTemplatePicker"
      preset="card"
      :title="t('alert.selectTemplate')"
      class="rfm-tpl-modal"
    >
      <div class="rfm-tpl-search">
        <n-input
          v-model:value="templateSearch"
          :placeholder="t('common.search')"
          size="small"
          clearable
          class="rfm-tpl-search-input"
          @update:value="fetchTemplates"
        />
        <n-select
          v-model:value="templateCategory"
          :options="templateCategories.map(c => ({ label: c, value: c }))"
          :placeholder="t('alert.category')"
          size="small"
          clearable
          class="rfm-tpl-cat-select"
          @update:value="fetchTemplates"
        />
      </div>
      <div class="tpl-list">
        <div v-if="templateLoading" class="tpl-loading">{{ t('common.loading') }}</div>
        <div v-else-if="templates.length === 0" class="tpl-loading">{{ t('common.noData') }}</div>
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

    <!-- Form: collapsible card sections -->
    <div class="rfm-sections">
      <n-form label-placement="top">
        <!-- Section 1: Basic Info -->
        <n-card size="small" class="rfm-section-card">
          <template #header>
            <div class="rfm-section-header" @click="sectionBasicOpen = !sectionBasicOpen">
              <span class="rfm-section-title">{{ t('alert.sectionBasic') }}</span>
              <span class="rfm-section-chevron" :class="{ open: sectionBasicOpen }">&#9662;</span>
            </div>
          </template>
          <n-collapse-transition :show="sectionBasicOpen">
            <n-grid :x-gap="12" :cols="2">
              <n-gi>
                <n-form-item :label="t('common.name')" required>
                  <n-input v-model:value="form.name" :placeholder="t('alert.namePlaceholder')" />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item :label="t('alert.displayName')">
                  <n-input v-model:value="form.display_name" :placeholder="t('alert.displayNamePlaceholder')" />
                </n-form-item>
              </n-gi>
            </n-grid>
            <n-form-item :label="t('common.description')">
              <n-input v-model:value="form.description" type="textarea" :placeholder="t('common.description')" :rows="2" />
            </n-form-item>
            <n-grid :x-gap="12" :cols="2">
              <n-gi>
                <n-form-item :label="t('alert.groupName')">
                  <n-input v-model:value="form.group_name" :placeholder="t('alert.groupNamePlaceholder')" />
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
                  <n-input-number v-model:value="form.group_wait_seconds" :min="0" :max="3600" :placeholder="t('alert.groupWaitPlaceholder')" class="rfm-input-full">
                    <template #suffix>{{ t('common.seconds') }}</template>
                  </n-input-number>
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item :label="t('alert.groupInterval')">
                  <n-input-number v-model:value="form.group_interval_seconds" :min="0" :max="86400" :placeholder="t('alert.groupIntervalPlaceholder')" class="rfm-input-full">
                    <template #suffix>{{ t('common.seconds') }}</template>
                  </n-input-number>
                </n-form-item>
              </n-gi>
            </n-grid>
          </n-collapse-transition>
        </n-card>

        <!-- Section 2: Query & Condition -->
        <n-card size="small" class="rfm-section-card">
          <template #header>
            <div class="rfm-section-header" @click="sectionQueryOpen = !sectionQueryOpen">
              <span class="rfm-section-title">{{ t('alert.sectionQuery') }}</span>
              <span class="rfm-section-chevron" :class="{ open: sectionQueryOpen }">&#9662;</span>
            </div>
          </template>
          <n-collapse-transition :show="sectionQueryOpen">
            <n-grid :x-gap="12" :cols="2">
              <n-gi>
                <n-form-item :label="t('alert.dataSource')">
                  <n-select v-model:value="form.datasource_id" :options="datasourceOptions" :placeholder="t('alert.selectDataSource')" clearable />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item :label="t('alert.datasourceType')">
                  <n-select
                    v-model:value="form.datasource_type"
                    :options="datasourceTypeOptions"
                    :placeholder="t('alert.selectDatasourceType')"
                    :disabled="form.datasource_id != null"
                    clearable
                  />
                </n-form-item>
              </n-gi>
            </n-grid>

            <n-form-item required>
              <template #label>
                <span class="rfm-expr-label">
                  {{ t('alert.expression') }} <span class="lang-pill">{{ expressionLang }}</span>
                </span>
              </template>
              <div class="rfm-expr-wrap">
                <n-input
                  v-model:value="form.expression"
                  type="textarea"
                  :placeholder="expressionPlaceholder"
                  :rows="3"
                  class="rfm-expr-input"
                />
                <div class="rfm-expr-actions">
                  <n-button
                    size="small"
                    :loading="queryTesting"
                    :disabled="!form.datasource_id || !form.expression.trim()"
                    @click="handleTestExpression"
                  >
                    <template #icon><n-icon :component="PlayOutline" /></template>
                    {{ queryTesting ? t('alert.testing') : t('alert.testExpression') }}
                  </n-button>
                  <n-button
                    size="small"
                    secondary
                    :disabled="!form.datasource_id || !form.expression.trim()"
                    @click="openGraphPreview"
                  >
                    <template #icon><n-icon :component="StatsChartOutline" /></template>
                    {{ t('alert.previewGraph') }}
                  </n-button>
                </div>
                <n-collapse-transition :show="queryResult !== null">
                  <div class="query-result">
                    <div class="sre-label-eyebrow qr-eyebrow">{{ t('alert.testResult') }}</div>
                    <div v-if="queryResult?.result_type === 'logs'" class="qr-logs">
                      {{ t('alert.matchedLogs') }}: <span class="tnum">{{ queryResult.raw_count }}</span>
                    </div>
                    <div v-else-if="queryResult?.series && queryResult.series.length > 0" class="series-list">
                      <div v-for="(s, i) in queryResult.series" :key="i" class="series-row">
                        <code class="series-labels">{{ Object.entries(s.labels || {}).map(([k, v]) => `${k}=${v}`).join(', ') }}</code>
                        <span class="series-value tnum">{{ s.values?.[0]?.value ?? '-' }}</span>
                      </div>
                    </div>
                    <div v-else class="qr-empty">
                      {{ t('alert.noResults') }}
                    </div>
                  </div>
                </n-collapse-transition>
              </div>
            </n-form-item>

            <n-grid :x-gap="12" :cols="2">
              <n-gi>
                <n-form-item :label="t('alert.forDuration')">
                  <n-input v-model:value="form.for_duration" :placeholder="t('alert.forDurationPlaceholder')" />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item :label="t('alert.severity')">
                  <n-select v-model:value="form.severity" :options="severityOptions" />
                </n-form-item>
              </n-gi>
            </n-grid>
          </n-collapse-transition>
        </n-card>

        <!-- Section 3: Labels & Annotations -->
        <n-card size="small" class="rfm-section-card">
          <template #header>
            <div class="rfm-section-header" @click="sectionLabelsOpen = !sectionLabelsOpen">
              <span class="rfm-section-title">{{ t('alert.sectionLabels') }}</span>
              <span class="rfm-section-chevron" :class="{ open: sectionLabelsOpen }">&#9662;</span>
            </div>
          </template>
          <n-collapse-transition :show="sectionLabelsOpen">
            <n-form-item :label="t('alert.labels')">
              <KVEditor v-model:modelValue="form.labels" :add-label="t('alert.addLabel')" :key-options="labelKeys" :value-options="labelValues" @key-change="onLabelKeyChange" />
            </n-form-item>
            <n-form-item :label="t('alert.annotations')">
              <KVEditor v-model:modelValue="form.annotations" :add-label="t('alert.addAnnotation')" :key-placeholder="t('alert.annotationKeyPlaceholder')" />
            </n-form-item>
          </n-collapse-transition>
        </n-card>

        <!-- Section 4: Advanced Settings -->
        <n-card size="small" class="rfm-section-card">
          <template #header>
            <div class="rfm-section-header" @click="sectionAdvancedOpen = !sectionAdvancedOpen">
              <span class="rfm-section-title">{{ t('alert.advancedSettings') }}</span>
              <span class="rfm-section-chevron" :class="{ open: sectionAdvancedOpen }">&#9662;</span>
            </div>
          </template>
          <n-collapse-transition :show="sectionAdvancedOpen">
            <n-grid :x-gap="12" :cols="2">
              <n-gi>
                <n-form-item :label="t('alert.ruleType')">
                  <n-select v-model:value="form.rule_type" :options="[
                    { label: t('alert.ruleTypeThreshold'), value: 'threshold' },
                    { label: t('alert.ruleTypeHeartbeat'), value: 'heartbeat' },
                  ]" />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item :label="t('alert.evalInterval')">
                  <n-input-number v-model:value="form.eval_interval" :min="10" :max="86400" class="rfm-input-full">
                    <template #suffix>{{ t('common.seconds') }}</template>
                  </n-input-number>
                </n-form-item>
              </n-gi>
            </n-grid>

            <n-grid :x-gap="12" :cols="2">
              <n-gi>
                <n-form-item :label="t('alert.recoveryHold')">
                  <n-input v-model:value="form.recovery_hold" placeholder="0s" />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item :label="t('alert.ackSla')">
                  <n-input-number v-model:value="form.ack_sla_minutes" :min="0" :max="1440" class="rfm-input-full">
                    <template #suffix>{{ t('common.minutes') }}</template>
                  </n-input-number>
                </n-form-item>
              </n-gi>
            </n-grid>

            <n-grid :x-gap="12" :cols="2">
              <n-gi>
                <n-form-item :label="t('alert.nodataEnabled')">
                  <n-switch v-model:value="form.nodata_enabled" />
                </n-form-item>
              </n-gi>
              <n-gi v-if="form.nodata_enabled">
                <n-form-item :label="t('alert.nodataDuration')">
                  <n-input v-model:value="form.nodata_duration" placeholder="5m" />
                </n-form-item>
              </n-gi>
            </n-grid>

            <n-form-item :label="t('alert.suppressEnabled')">
              <n-switch v-model:value="form.suppress_enabled" />
            </n-form-item>

            <!-- Heartbeat fields (only for heartbeat type) -->
            <template v-if="form.rule_type === 'heartbeat'">
              <n-grid :x-gap="12" :cols="2">
                <n-gi>
                  <n-form-item :label="t('alert.heartbeatToken')">
                    <n-input v-model:value="form.heartbeat_token" :placeholder="t('alert.heartbeatTokenPlaceholder')" />
                  </n-form-item>
                </n-gi>
                <n-gi>
                  <n-form-item :label="t('alert.heartbeatInterval')">
                    <n-input-number v-model:value="form.heartbeat_interval" :min="30" :max="86400" class="rfm-input-full">
                      <template #suffix>{{ t('common.seconds') }}</template>
                    </n-input-number>
                  </n-form-item>
                </n-gi>
              </n-grid>
            </template>
          </n-collapse-transition>
        </n-card>
      </n-form>
    </div>

    <template #action>
      <div class="rfm-footer">
        <n-button size="small" secondary @click="saveAsTemplate">
          {{ t('alert.saveAsTemplate') }}
        </n-button>
        <div class="rfm-footer-right">
          <n-button size="small" @click="emit('close')">{{ t('common.cancel') }}</n-button>
          <n-button size="small" type="primary" :loading="saving" @click="handleSave">
            {{ editingId ? t('common.update') : t('common.create') }}
          </n-button>
        </div>
      </div>
    </template>
  </n-modal>

  <!-- Graph Preview Drawer -->
  <n-drawer
    :show="showGraphDrawer"
    :width="560"
    placement="right"
    @update:show="(v: boolean) => { if (!v) showGraphDrawer = false }"
  >
    <n-drawer-content :title="t('alert.graphPreviewTitle')" closable>
      <div class="graph-drawer-body">
        <div class="graph-toolbar">
          <n-select
            v-model:value="graphTimeRange"
            :options="graphTimeRangeOptions"
            size="small"
            class="graph-time-select"
            @update:value="openGraphPreview"
          />
          <n-button
            size="small"
            secondary
            :loading="graphLoading"
            @click="openGraphPreview"
          >
            <template #icon><n-icon :component="PlayOutline" /></template>
            {{ t('alert.testExpression') }}
          </n-button>
        </div>

        <n-spin :show="graphLoading">
          <div v-if="graphChartOption" class="graph-chart-container">
            <VChart
              :option="graphChartOption"
              style="width: 100%; height: 320px"
              autoresize
            />
            <div class="graph-legend-table">
              <NDataTable
                :columns="graphLegendColumns"
                :data="graphLegendData"
                :max-height="200"
                size="small"
                striped
                :pagination="false"
                :scroll-x="480"
              />
            </div>
          </div>
          <div v-else class="graph-empty">
            <span v-if="graphLoading">{{ t('common.loading') }}</span>
            <span v-else>{{ t('common.noData') }}</span>
          </div>
        </n-spin>
      </div>
    </n-drawer-content>
  </n-drawer>
</template>

<style scoped>
.rfm-modal {
  width: 900px;
}

.rfm-template-bar {
  margin-bottom: 16px;
  display: flex;
  gap: 8px;
  align-items: center;
}

.rfm-template-applied {
  font-size: 12px;
  color: var(--sre-text-tertiary);
}

.rfm-tpl-modal {
  width: 600px;
}

.rfm-tpl-search {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.rfm-tpl-search-input {
  flex: 1;
}

.rfm-tpl-cat-select {
  width: 140px;
}

.rfm-input-full {
  width: 100%;
}

/* Collapsible card sections */
.rfm-sections {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.rfm-section-card {
  border-radius: 8px;
}

.rfm-section-card :deep(.n-card-header) {
  padding: 10px 16px;
}

.rfm-section-card :deep(.n-card__content) {
  padding: 12px 16px 16px;
}

.rfm-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  user-select: none;
}

.rfm-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
}

.rfm-section-chevron {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  transition: transform 200ms ease;
  line-height: 1;
}

.rfm-section-chevron.open {
  transform: rotate(0deg);
}

.rfm-section-chevron:not(.open) {
  transform: rotate(-90deg);
}

/* Expression area */
.rfm-expr-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.rfm-expr-wrap {
  width: 100%;
}

.rfm-expr-input {
  font-family: var(--sre-font-mono, monospace);
}

.rfm-expr-actions {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Footer */
.rfm-footer {
  display: flex;
  justify-content: space-between;
  width: 100%;
}

.rfm-footer-right {
  display: flex;
  gap: 8px;
}

/* Template list */
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
.qr-eyebrow {
  margin-bottom: 8px;
}
.qr-logs {
  font-size: 13px;
  color: var(--sre-text-secondary);
}
.qr-empty {
  font-size: 13px;
  color: var(--sre-text-secondary);
  padding: 8px 0;
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

/* Graph preview drawer */
.graph-drawer-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.graph-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.graph-time-select {
  width: 100px;
}

.graph-chart-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.graph-legend-table {
  border-top: var(--sre-hairline);
  padding-top: 8px;
}

.graph-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 320px;
  color: var(--sre-text-tertiary);
  font-size: 14px;
}
</style>
