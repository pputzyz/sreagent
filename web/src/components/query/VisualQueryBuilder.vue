<script setup lang="ts">
/**
 * VisualQueryBuilder — Visual PromQL construction UI.
 *
 * Allows building PromQL expressions without writing raw text:
 *  - Metric selection (autocomplete from datasource)
 *  - Label filters (key=value, !=, =~, !~)
 *  - Range-vector functions (rate, increase, etc.)
 *  - Aggregation (sum, avg, etc.) with by/without grouping
 *  - Binary operations (+, -, *, /)
 *
 * Bidirectionally syncs with a parent-provided expression string.
 */
import { ref, watch, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NSelect, NButton, NIcon, NInput,
  NCollapse, NCollapseItem,
} from 'naive-ui'
import {
  AddOutline, CloseOutline,
} from '@vicons/ionicons5'
import { datasourceApi } from '@/api'
import {
  useQueryBuilder,
  RANGE_FUNCTIONS, AGGREGATION_OPS, BINARY_OPS, DURATION_PRESETS,
} from '@/composables/useQueryBuilder'

const props = defineProps<{
  datasourceId: number | null
  modelValue: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'execute'): void
}>()

const { t } = useI18n()

// ---- Composable ----
const {
  state,
  generatedPromQL,
  parseExpression,
  addLabelFilter,
  removeLabelFilter,
  addBinaryOperand,
  removeBinaryOperand,
  reset,
} = useQueryBuilder()

// ---- Metric name autocomplete ----
const metricNames = ref<string[]>([])
const metricLoading = ref(false)

async function loadMetricNames() {
  if (!props.datasourceId) { metricNames.value = []; return }
  metricLoading.value = true
  try {
    const res = await datasourceApi.metricNames(props.datasourceId, undefined, 5000)
    metricNames.value = res.data?.data || []
  } catch {
    metricNames.value = []
  } finally {
    metricLoading.value = false
  }
}

const metricOptions = computed(() => {
  return metricNames.value.map(m => ({ label: m, value: m }))
})

onMounted(() => { loadMetricNames() })
watch(() => props.datasourceId, () => { loadMetricNames() })

// ---- Label name autocomplete ----
const labelNames = ref<string[]>([])
const labelLoading = ref(false)

async function loadLabelNames() {
  if (!props.datasourceId) { labelNames.value = []; return }
  labelLoading.value = true
  try {
    const res = await datasourceApi.labelKeys(props.datasourceId)
    labelNames.value = res.data?.data || []
  } catch {
    labelNames.value = []
  } finally {
    labelLoading.value = false
  }
}

const labelNameOptions = computed(() => {
  return labelNames.value.map(l => ({ label: l, value: l }))
})

onMounted(() => { loadLabelNames() })
watch(() => props.datasourceId, () => { loadLabelNames() })

// ---- Label value autocomplete (per filter) ----
const labelValueCache = ref<Record<string, string[]>>({})
const labelValueLoading = ref<Record<string, boolean>>({})

async function fetchLabelValues(filterId: string, labelKey: string) {
  if (!props.datasourceId || !labelKey) return
  labelValueLoading.value[filterId] = true
  try {
    const res = await datasourceApi.labelValues(props.datasourceId, labelKey)
    labelValueCache.value[filterId] = res.data?.data || []
  } catch {
    labelValueCache.value[filterId] = []
  } finally {
    labelValueLoading.value[filterId] = false
  }
}

function onFilterKeyChange(filterId: string, key: string) {
  const f = state.value.labelFilters.find(l => l.id === filterId)
  if (f) {
    f.key = key
    f.value = ''
    fetchLabelValues(filterId, key)
  }
}

function labelValueOptions(filterId: string) {
  return (labelValueCache.value[filterId] || []).map(v => ({ label: v, value: v }))
}

// ---- Operator options ----
const operatorOptions = [
  { label: '=', value: '=' },
  { label: '!=', value: '!=' },
  { label: '=~', value: '=~' },
  { label: '!~', value: '!~' },
]

// ---- Range function options ----
const rangeFnOptions = RANGE_FUNCTIONS.map(f => ({ label: f.label, value: f.value }))
const durationOptions = DURATION_PRESETS.map(d => ({ label: d, value: d }))

// ---- Aggregation options ----
const aggOpOptions = AGGREGATION_OPS.map(a => ({ label: a.label, value: a.value }))
const groupModifierOptions = [
  { label: 'none', value: '' },
  { label: 'by', value: 'by' },
  { label: 'without', value: 'without' },
]

// ---- Binary operation options ----
const binaryOpOptions = BINARY_OPS.map(b => ({ label: b.label, value: b.value }))
const binaryTypeOptions = [
  { label: 'Scalar', value: 'scalar' },
  { label: 'Metric', value: 'metric' },
]

// ---- Collapse state ----
const collapseValue = ref<string[]>(['metric', 'labels'])

// ---- Bidirectional sync ----
let ignoreNextEmit = false

// External expression (from code mode) -> parse into builder state
watch(() => props.modelValue, (expr) => {
  // Skip if this change was triggered by the builder itself
  if (ignoreNextEmit) { ignoreNextEmit = false; return }
  parseExpression(expr)
}, { immediate: true })

// Builder state changes -> emit updated expression to parent
watch(generatedPromQL, (newExpr) => {
  ignoreNextEmit = true
  emit('update:modelValue', newExpr)
})

// ---- Reset all ----
function resetAll() {
  reset()
  labelValueCache.value = {}
}

// ---- Expose ----
defineExpose({ reset: resetAll, state })
</script>

<template>
  <div class="visual-query-builder">
    <NCollapse v-model:value="collapseValue" :accordion="false" class="builder-collapse">
      <!-- Metric Selection -->
      <NCollapseItem name="metric" :title="t('query.vqbMetric') || 'Metric'">
        <div class="builder-section">
          <NSelect
            v-model:value="state.metricName"
            :options="metricOptions"
            :placeholder="t('query.vqbSelectMetric') || 'Select a metric...'"
            filterable
            clearable
            :loading="metricLoading"
            size="small"
            class="metric-select"
            @update:value="() => { /* refresh label values on metric change */ }"
          />
        </div>
      </NCollapseItem>

      <!-- Label Filters -->
      <NCollapseItem name="labels" :title="t('query.vqbLabelFilters') || 'Label Filters'">
        <div class="builder-section">
          <div v-if="state.labelFilters.length === 0" class="builder-empty">
            {{ t('query.vqbNoFilters') || 'No label filters. Click + to add.' }}
          </div>
          <div v-for="filter in state.labelFilters" :key="filter.id" class="filter-row">
            <!-- Label key -->
            <NSelect
              :value="filter.key"
              :options="labelNameOptions"
              :placeholder="t('query.vqbLabelKey') || 'Label'"
              filterable
              :loading="labelLoading"
              size="small"
              class="filter-key"
              @update:value="(v: string) => onFilterKeyChange(filter.id, v)"
            />
            <!-- Operator -->
            <NSelect
              v-model:value="filter.operator"
              :options="operatorOptions"
              size="small"
              class="filter-op"
            />
            <!-- Value -->
            <NSelect
              v-model:value="filter.value"
              :options="labelValueOptions(filter.id)"
              :placeholder="labelValueLoading[filter.id] ? (t('common.loading') || 'Loading...') : (t('query.vqbLabelValue') || 'Value')"
              filterable
              tag
              :loading="labelValueLoading[filter.id]"
              size="small"
              class="filter-value"
            />
            <!-- Remove -->
            <NButton size="tiny" quaternary type="error" @click="removeLabelFilter(filter.id)">
              <template #icon><NIcon size="14"><CloseOutline /></NIcon></template>
            </NButton>
          </div>
          <NButton size="tiny" quaternary @click="addLabelFilter" :disabled="!datasourceId">
            <template #icon><NIcon size="14"><AddOutline /></NIcon></template>
            {{ t('common.add') || 'Add' }}
          </NButton>
        </div>
      </NCollapseItem>

      <!-- Range Function -->
      <NCollapseItem name="function" :title="t('query.vqbFunction') || 'Function'">
        <div class="builder-section">
          <div class="toggle-row">
            <span class="toggle-label">{{ t('query.vqbEnableFunction') || 'Wrap with function' }}</span>
            <NButton
              size="tiny"
              :type="state.rangeFunction.enabled ? 'primary' : 'default'"
              :secondary="!state.rangeFunction.enabled"
              @click="state.rangeFunction.enabled = !state.rangeFunction.enabled"
            >
              {{ state.rangeFunction.enabled ? 'ON' : 'OFF' }}
            </NButton>
          </div>
          <div v-if="state.rangeFunction.enabled" class="function-config">
            <NSelect
              v-model:value="state.rangeFunction.fn"
              :options="rangeFnOptions"
              size="small"
              class="fn-select"
            />
            <NSelect
              v-model:value="state.rangeFunction.range"
              :options="durationOptions"
              :placeholder="t('query.vqbDuration') || 'Duration'"
              filterable
              tag
              size="small"
              class="fn-duration"
            />
          </div>
        </div>
      </NCollapseItem>

      <!-- Aggregation -->
      <NCollapseItem name="aggregation" :title="t('query.vqbAggregation') || 'Aggregation'">
        <div class="builder-section">
          <div class="toggle-row">
            <span class="toggle-label">{{ t('query.vqbEnableAggregation') || 'Apply aggregation' }}</span>
            <NButton
              size="tiny"
              :type="state.aggregation.enabled ? 'primary' : 'default'"
              :secondary="!state.aggregation.enabled"
              @click="state.aggregation.enabled = !state.aggregation.enabled"
            >
              {{ state.aggregation.enabled ? 'ON' : 'OFF' }}
            </NButton>
          </div>
          <div v-if="state.aggregation.enabled" class="aggregation-config">
            <NSelect
              v-model:value="state.aggregation.op"
              :options="aggOpOptions"
              size="small"
              class="agg-op-select"
            />
            <NSelect
              v-model:value="state.aggregation.groupModifier"
              :options="groupModifierOptions"
              size="small"
              class="agg-mod-select"
            />
            <div v-if="state.aggregation.groupModifier" class="agg-group-labels">
              <NSelect
                v-model:value="state.aggregation.groupLabels"
                :options="labelNameOptions"
                :placeholder="t('query.vqbGroupLabels') || 'Group by labels...'"
                filterable
                multiple
                :loading="labelLoading"
                size="small"
                class="agg-labels-select"
              />
            </div>
          </div>
        </div>
      </NCollapseItem>

      <!-- Binary Operations -->
      <NCollapseItem name="binary" :title="t('query.vqbBinaryOps') || 'Binary Operations'">
        <div class="builder-section">
          <div v-if="state.binaryOperands.length === 0" class="builder-empty">
            {{ t('query.vqbNoBinary') || 'No binary operations. Click + to add.' }}
          </div>
          <div v-for="operand in state.binaryOperands" :key="operand.id" class="binary-row">
            <NSelect
              v-model:value="operand.op"
              :options="binaryOpOptions"
              size="small"
              class="binary-op"
            />
            <NSelect
              v-model:value="operand.type"
              :options="binaryTypeOptions"
              size="small"
              class="binary-type"
            />
            <NInput
              v-if="operand.type === 'scalar'"
              v-model:value="operand.scalarValue"
              :placeholder="t('query.vqbScalarValue') || 'e.g. 1024'"
              size="small"
              class="binary-input"
            />
            <NInput
              v-else
              v-model:value="operand.metricExpression"
              :placeholder="t('query.vqbMetricExpr') || 'e.g. metric_name{...}'"
              size="small"
              class="binary-input"
            />
            <NButton size="tiny" quaternary type="error" @click="removeBinaryOperand(operand.id)">
              <template #icon><NIcon size="14"><CloseOutline /></NIcon></template>
            </NButton>
          </div>
          <NButton size="tiny" quaternary @click="addBinaryOperand">
            <template #icon><NIcon size="14"><AddOutline /></NIcon></template>
            {{ t('common.add') || 'Add' }}
          </NButton>
        </div>
      </NCollapseItem>
    </NCollapse>

    <!-- Preview -->
    <div class="builder-preview">
      <span class="preview-label">{{ t('query.vqbPreview') || 'Preview' }}</span>
      <code class="preview-expr">{{ generatedPromQL || '(empty)' }}</code>
    </div>
  </div>
</template>

<style scoped>
.visual-query-builder {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.builder-collapse {
  border: 1px solid var(--sre-border);
  border-radius: 6px;
  overflow: hidden;
}

.builder-collapse :deep(.n-collapse-item) {
  --n-title-font-size: 13px;
}

.builder-collapse :deep(.n-collapse-item__header) {
  padding: 8px 12px;
  min-height: 36px;
  font-weight: 600;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--sre-text-secondary);
}

.builder-collapse :deep(.n-collapse-item__content-wrapper) {
  padding: 0;
}

.builder-collapse :deep(.n-collapse-item__content) {
  padding: 0;
}

.builder-section {
  padding: 8px 12px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.builder-empty {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  padding: 4px 0;
  text-align: center;
}

/* Metric select */
.metric-select {
  width: 100%;
}

/* Label filter rows */
.filter-row {
  display: flex;
  gap: 4px;
  align-items: center;
}
.filter-key {
  width: 160px;
  flex-shrink: 0;
}
.filter-op {
  width: 72px;
  flex-shrink: 0;
}
.filter-value {
  flex: 1;
  min-width: 0;
}

/* Toggle rows */
.toggle-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.toggle-label {
  font-size: 12px;
  color: var(--sre-text-secondary);
}

/* Function config */
.function-config {
  display: flex;
  gap: 8px;
  align-items: center;
}
.fn-select {
  width: 200px;
  flex-shrink: 0;
}
.fn-duration {
  width: 120px;
  flex-shrink: 0;
}

/* Aggregation config */
.aggregation-config {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.agg-op-select {
  width: 180px;
}
.agg-mod-select {
  width: 140px;
}
.agg-group-labels {
  width: 100%;
}
.agg-labels-select {
  width: 100%;
}

/* Binary operations */
.binary-row {
  display: flex;
  gap: 4px;
  align-items: center;
}
.binary-op {
  width: 72px;
  flex-shrink: 0;
}
.binary-type {
  width: 100px;
  flex-shrink: 0;
}
.binary-input {
  flex: 1;
  min-width: 0;
}

/* Preview */
.builder-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--sre-bg-sunken, #f8fafc);
  border: 1px solid var(--sre-border);
  border-radius: 6px;
}
.preview-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  flex-shrink: 0;
}
.preview-expr {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-primary);
  word-break: break-all;
  white-space: pre-wrap;
  line-height: 1.5;
}
</style>
