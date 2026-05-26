<script setup lang="ts">
/**
 * MetricLabelSelector — Cascading label filter builder.
 * Inspired by Nightingale's label filter pattern:
 *  1. Select a label name (fetched from datasource)
 *  2. Select operator (=, !=, =~, !~)
 *  3. Select label values (fetched filtered by current selectors)
 *  4. Multiple filters combine into a PromQL selector
 */
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NSelect, NButton, NIcon, NTag, NSpace, NSpin,
} from 'naive-ui'
import { AddOutline, CloseOutline } from '@vicons/ionicons5'
import { datasourceApi } from '@/api'

export interface LabelFilter {
  id: string
  label: string
  operator: '=' | '!=' | '=~' | '!~'
  value: string
  values: string[]
  loadingValues: boolean
}

const props = defineProps<{
  datasourceId: number | null
  metricName?: string
}>()

const emit = defineEmits<{
  (e: 'update:selector', selector: string): void
}>()

const { t } = useI18n()

// Available label names
const labelNames = ref<string[]>([])
const loadingLabels = ref(false)

// Active filters
const filters = ref<LabelFilter[]>([])

// Operator options
const operatorOptions = [
  { label: '=', value: '=' },
  { label: '!=', value: '!=' },
  { label: '=~', value: '=~' },
  { label: '!~', value: '!~' },
]

// Load label names when datasource changes
watch(() => props.datasourceId, async (dsId) => {
  filters.value = []
  labelNames.value = []
  if (!dsId) return
  loadingLabels.value = true
  try {
    const res = await datasourceApi.labelKeys(dsId)
    labelNames.value = res.data?.data || []
  } catch {
    labelNames.value = []
  } finally {
    loadingLabels.value = false
  }
  emitSelector()
}, { immediate: true })

// Available label names (exclude already selected)
const availableLabelOptions = computed(() => {
  const used = new Set(filters.value.map(f => f.label))
  return labelNames.value
    .filter(l => !used.has(l))
    .map(l => ({ label: l, value: l }))
})

// Build the match expression for fetching label values
function buildMatchExpr(excludeFilterId?: string): string | undefined {
  const parts: string[] = []
  // Include metric name if selected
  if (props.metricName) parts.push(props.metricName)
  // Include existing filters (except the one being edited)
  for (const f of filters.value) {
    if (f.id === excludeFilterId) continue
    if (!f.label || !f.value) continue
    if (f.operator === '=') parts.push(`${f.label}="${f.value}"`)
    else if (f.operator === '!=') parts.push(`${f.label}!="${f.value}"`)
    else parts.push(`${f.label}${f.operator}"${f.value}"`)
  }
  if (parts.length === 0) return undefined
  // If only metric name, return it directly
  if (parts.length === 1 && !parts[0].includes('=')) return parts[0]
  // Build selector
  const name = props.metricName || ''
  const labelParts = parts.filter(p => p.includes('='))
  return name ? `${name}{${labelParts.join(',')}}` : `{${labelParts.join(',')}}`
}

// Fetch values for a specific filter
async function fetchLabelValues(filter: LabelFilter) {
  if (!props.datasourceId || !filter.label) return
  filter.loadingValues = true
  filter.values = []
  try {
    const matchExpr = buildMatchExpr(filter.id)
    const params: Record<string, string> = {}
    if (matchExpr) params['match[]'] = matchExpr
    const res = await datasourceApi.proxy(
      props.datasourceId,
      `/api/v1/label/${filter.label}/values`,
      params
    )
    const data = res.data?.data as string[] | undefined
    filter.values = data || []
  } catch {
    filter.values = []
  } finally {
    filter.loadingValues = false
  }
}

// Add a new empty filter
function addFilter() {
  const filter: LabelFilter = {
    id: Date.now().toString(36) + Math.random().toString(36).slice(2, 6),
    label: '',
    operator: '=',
    value: '',
    values: [],
    loadingValues: false,
  }
  filters.value.push(filter)
}

// Remove a filter
function removeFilter(id: string) {
  filters.value = filters.value.filter(f => f.id !== id)
  emitSelector()
}

// When label name is selected, fetch its values
function onLabelChange(filter: LabelFilter) {
  filter.value = ''
  fetchLabelValues(filter)
  emitSelector()
}

// When value is selected, emit the new selector
function onValueChange(filter: LabelFilter) {
  emitSelector()
}

// When operator changes, re-fetch values and emit
function onOperatorChange(filter: LabelFilter) {
  if (filter.label) fetchLabelValues(filter)
  emitSelector()
}

// Build and emit the PromQL selector string
function emitSelector() {
  const activeFilters = filters.value.filter(f => f.label && f.value)
  if (activeFilters.length === 0) {
    emit('update:selector', props.metricName || '')
    return
  }
  const parts: string[] = []
  for (const f of activeFilters) {
    if (f.operator === '=') parts.push(`${f.label}="${f.value}"`)
    else if (f.operator === '!=') parts.push(`${f.label}!="${f.value}"`)
    else if (f.operator === '=~') parts.push(`${f.label}=~"${f.value}"`)
    else parts.push(`${f.label}!~"${f.value}"`)
  }
  const name = props.metricName || ''
  emit('update:selector', name ? `${name}{${parts.join(',')}}` : `{${parts.join(',')}}`)
}

// Label value options for NSelect
function valueOptions(filter: LabelFilter) {
  return filter.values.map(v => ({ label: v, value: v }))
}

// Refresh all label values (called when metric name changes)
function refreshAll() {
  for (const f of filters.value) {
    if (f.label) fetchLabelValues(f)
  }
}

// Expose for parent
defineExpose({ refreshAll, filters })
</script>

<template>
  <div class="label-selector">
    <div class="label-selector-header">
      <span class="label-selector-title">{{ t('query.labelFilters') || 'Label Filters' }}</span>
      <NButton size="tiny" quaternary @click="addFilter" :disabled="!datasourceId || availableLabelOptions.length === 0">
        <template #icon><NIcon size="14"><AddOutline /></NIcon></template>
        {{ t('common.add') }}
      </NButton>
    </div>

    <div v-if="loadingLabels" class="label-loading">
      <NSpin size="small" />
    </div>

    <div v-else-if="!datasourceId" class="label-empty">
      {{ t('query.selectDatasource') }}
    </div>

    <div v-else-if="filters.length === 0" class="label-empty">
      {{ t('query.noLabelFilters') || 'No label filters. Click + to add.' }}
    </div>

    <div v-else class="label-filters">
      <div v-for="filter in filters" :key="filter.id" class="label-filter-row">
        <!-- Label name -->
        <NSelect
          v-model:value="filter.label"
          :options="availableLabelOptions"
          :placeholder="t('query.labelName') || 'Label'"
          filterable
          size="small"
          class="filter-label-select"
          @update:value="onLabelChange(filter)"
        />
        <!-- Operator -->
        <NSelect
          v-model:value="filter.operator"
          :options="operatorOptions"
          size="small"
          class="filter-op-select"
          @update:value="onOperatorChange(filter)"
        />
        <!-- Value -->
        <NSelect
          v-model:value="filter.value"
          :options="valueOptions(filter)"
          :placeholder="filter.loadingValues ? (t('common.loading') || 'Loading...') : (t('query.labelValue') || 'Value')"
          filterable
          :loading="filter.loadingValues"
          size="small"
          class="filter-value-select"
          @update:value="onValueChange(filter)"
        />
        <!-- Remove -->
        <NButton size="tiny" quaternary type="error" @click="removeFilter(filter.id)">
          <template #icon><NIcon size="14"><CloseOutline /></NIcon></template>
        </NButton>
      </div>
    </div>

    <!-- Active filters summary -->
    <div v-if="filters.some(f => f.label && f.value)" class="label-summary">
      <NTag
        v-for="f in filters.filter(f => f.label && f.value)"
        :key="f.id"
        size="small"
        :bordered="false"
        closable
        @close="removeFilter(f.id)"
      >
        {{ f.label }}{{ f.operator }}"{{ f.value }}"
      </NTag>
    </div>
  </div>
</template>

<style scoped>
.label-selector {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.label-selector-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.label-selector-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.label-loading {
  display: flex;
  justify-content: center;
  padding: 12px;
}
.label-empty {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  padding: 8px 0;
  text-align: center;
}
.label-filters {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.label-filter-row {
  display: flex;
  gap: 4px;
  align-items: center;
}
.filter-label-select {
  width: 140px;
  flex-shrink: 0;
}
.filter-op-select {
  width: 64px;
  flex-shrink: 0;
}
.filter-value-select {
  flex: 1;
  min-width: 0;
}
.label-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding-top: 4px;
  border-top: 1px solid var(--sre-border);
}
</style>
