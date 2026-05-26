<script setup lang="ts">
/**
 * MetricList — Searchable metric name list with prefix grouping.
 * Inspired by Nightingale's metric selector pattern.
 *
 * Fetches __name__ values from the datasource proxy API,
 * groups by prefix (e.g., go_*, node_*), and allows search/filter.
 */
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NInput, NIcon, NSpin, NCollapse, NCollapseItem, NTag, NEmpty,
} from 'naive-ui'
import { SearchOutline, ChevronForwardOutline } from '@vicons/ionicons5'
import { datasourceApi } from '@/api'

const props = defineProps<{
  datasourceId: number | null
}>()

const emit = defineEmits<{
  (e: 'select', metricName: string): void
}>()

const { t } = useI18n()

// State
const metricNames = ref<string[]>([])
const loading = ref(false)
const searchQuery = ref('')
const selectedMetric = ref('')

// Group metrics by prefix
interface MetricGroup {
  prefix: string
  metrics: string[]
}

const metricGroups = computed<MetricGroup[]>(() => {
  const filtered = searchQuery.value
    ? metricNames.value.filter(m => m.toLowerCase().includes(searchQuery.value.toLowerCase()))
    : metricNames.value

  if (filtered.length === 0) return []

  // Group by prefix (first segment before _)
  const groupMap = new Map<string, string[]>()
  for (const m of filtered) {
    const idx = m.indexOf('_')
    const prefix = idx > 0 ? m.substring(0, idx + 1) + '*' : m
    const arr = groupMap.get(prefix) || []
    arr.push(m)
    groupMap.set(prefix, arr)
  }

  // Convert to array, sort by prefix
  const groups: MetricGroup[] = []
  for (const [prefix, metrics] of groupMap) {
    groups.push({ prefix, metrics: metrics.sort() })
  }
  groups.sort((a, b) => a.prefix.localeCompare(b.prefix))
  return groups
})

// Flat filtered list for search
const flatFiltered = computed(() => {
  if (!searchQuery.value) return []
  return metricNames.value
    .filter(m => m.toLowerCase().includes(searchQuery.value.toLowerCase()))
    .sort()
    .slice(0, 200)
})

// Total metric count
const totalCount = computed(() => metricNames.value.length)
const filteredCount = computed(() => {
  if (!searchQuery.value) return totalCount.value
  return metricNames.value.filter(m => m.toLowerCase().includes(searchQuery.value.toLowerCase())).length
})

// Load metric names
async function loadMetrics() {
  if (!props.datasourceId) {
    metricNames.value = []
    return
  }
  loading.value = true
  try {
    const res = await datasourceApi.metricNames(props.datasourceId, undefined, 5000)
    metricNames.value = res.data?.data || []
  } catch {
    metricNames.value = []
  } finally {
    loading.value = false
  }
}

// Watch datasource changes
watch(() => props.datasourceId, () => {
  selectedMetric.value = ''
  searchQuery.value = ''
  loadMetrics()
}, { immediate: true })

// Select a metric
function selectMetric(name: string) {
  selectedMetric.value = name
  emit('select', name)
}

// Expose for parent
defineExpose({ refresh: loadMetrics, selectedMetric })
</script>

<template>
  <div class="metric-list">
    <!-- Search -->
    <div class="metric-search">
      <NInput
        v-model:value="searchQuery"
        :placeholder="t('query.searchMetrics') || 'Search metrics...'"
        size="small"
        clearable
      >
        <template #prefix><NIcon size="14"><SearchOutline /></NIcon></template>
      </NInput>
      <div class="metric-count">
        <span v-if="loading">{{ t('common.loading') }}</span>
        <span v-else>{{ filteredCount }} / {{ totalCount }}</span>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="metric-loading">
      <NSpin size="small" />
    </div>

    <!-- Empty -->
    <NEmpty v-else-if="!datasourceId" :description="t('query.selectDatasource')" size="small" class="metric-empty" />
    <NEmpty v-else-if="metricNames.length === 0" :description="t('common.noData')" size="small" class="metric-empty" />
    <NEmpty v-else-if="searchQuery && flatFiltered.length === 0" :description="t('common.noData')" size="small" class="metric-empty" />

    <!-- Search results (flat list) -->
    <div v-else-if="searchQuery && flatFiltered.length > 0" class="metric-flat-list">
      <div
        v-for="name in flatFiltered"
        :key="name"
        class="metric-item"
        :class="{ 'metric-item-selected': selectedMetric === name }"
        @click="selectMetric(name)"
      >
        <span class="metric-item-name">{{ name }}</span>
      </div>
    </div>

    <!-- Grouped list (no search) -->
    <div v-else-if="!searchQuery && metricGroups.length > 0" class="metric-grouped-list">
      <NCollapse :default-expanded-names="metricGroups.slice(0, 5).map(g => g.prefix)" accordion>
        <NCollapseItem
          v-for="group in metricGroups"
          :key="group.prefix"
          :name="group.prefix"
        >
          <template #header>
            <div class="group-header">
              <span class="group-prefix">{{ group.prefix }}</span>
              <NTag size="tiny" :bordered="false">{{ group.metrics.length }}</NTag>
            </div>
          </template>
          <div class="group-metrics">
            <div
              v-for="name in group.metrics"
              :key="name"
              class="metric-item"
              :class="{ 'metric-item-selected': selectedMetric === name }"
              @click="selectMetric(name)"
            >
              <span class="metric-item-name">{{ name }}</span>
            </div>
          </div>
        </NCollapseItem>
      </NCollapse>
    </div>
  </div>
</template>

<style scoped>
.metric-list {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
}
.metric-search {
  padding-bottom: 8px;
  border-bottom: 1px solid var(--sre-border);
  margin-bottom: 8px;
  flex-shrink: 0;
}
.metric-count {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  margin-top: 4px;
}
.metric-loading {
  display: flex;
  justify-content: center;
  padding: 24px;
}
.metric-empty {
  padding: 24px 0;
}
.metric-flat-list {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}
.metric-grouped-list {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}
.group-header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.group-prefix {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
}
.group-metrics {
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.metric-item {
  padding: 4px 8px;
  cursor: pointer;
  border-radius: 4px;
  transition: background 0.12s;
}
.metric-item:hover {
  background: var(--sre-bg-hover);
}
.metric-item-selected {
  background: var(--sre-primary-soft, rgba(59, 130, 246, 0.1));
}
.metric-item-selected .metric-item-name {
  color: var(--sre-primary);
  font-weight: 600;
}
.metric-item-name {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-primary);
}

/* Collapse overrides */
.metric-grouped-list :deep(.n-collapse-item) {
  margin-bottom: 2px;
}
.metric-grouped-list :deep(.n-collapse-item__header) {
  padding: 4px 8px;
  min-height: 28px;
}
.metric-grouped-list :deep(.n-collapse-item__content-wrapper) {
  padding-left: 8px;
}
</style>
