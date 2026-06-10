<script setup lang="ts">
/**
 * LogFieldSidebar — Nightingale FieldsList-style sidebar for log field exploration.
 * Displays available fields from log entries with type icons, counts,
 * and a popover showing top 10 values with AND filter capability.
 */
import { ref, computed } from 'vue'
import {
  NInput, NPopover, NIcon, NProgress, NButton, NSpin, NBadge, NTooltip,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
import { AddCircleOutline, SearchOutline } from '@vicons/ionicons5'

const props = withDefaults(defineProps<{
  fields: string[]
  logEntries: Array<Record<string, any>>
  loading?: boolean
}>(), {
  loading: false,
})

const emit = defineEmits<{
  (e: 'addFieldFilter', key: string, value: string): void
}>()

// --- Search ---
const searchQuery = ref('')

// --- Field type inference (matches LogDetailDrawer) ---
type FieldType = 'string' | 'number' | 'date' | 'boolean' | 'unknown'

function inferFieldType(value: unknown): FieldType {
  if (value == null) return 'unknown'
  if (typeof value === 'boolean') return 'boolean'
  if (typeof value === 'number') return 'number'
  if (typeof value === 'string') {
    if (/^\d{4}-\d{2}-\d{2}/.test(value) || /^\d{13}$/.test(value)) return 'date'
    if (/^-?\d+(\.\d+)?$/.test(value)) return 'number'
    if (value === 'true' || value === 'false') return 'boolean'
  }
  return 'string'
}

const typeIconMap: Record<FieldType, { text: string; color: string }> = {
  string: { text: 't', color: '#4a7194' },
  number: { text: '#', color: '#387765' },
  date: { text: '\u{1F4C5}', color: '#7b705a' },
  boolean: { text: 'B', color: '#996130' },
  unknown: { text: '?', color: '#94a3b8' },
}

// --- Field stats: count per field + dominant type ---
interface FieldStat {
  name: string
  count: number
  type: FieldType
}

const fieldStats = computed<FieldStat[]>(() => {
  const total = props.logEntries.length
  if (!total || !props.fields.length) return []

  return props.fields.map(name => {
    let count = 0
    let dominantType: FieldType = 'unknown'
    const typeCounts: Record<FieldType, number> = {
      string: 0, number: 0, date: 0, boolean: 0, unknown: 0,
    }

    for (const entry of props.logEntries) {
      const value = entry[name]
      if (value != null) {
        count++
        const ft = inferFieldType(value)
        typeCounts[ft]++
      }
    }

    // Pick dominant type
    let maxCount = 0
    for (const [tp, cnt] of Object.entries(typeCounts)) {
      if (cnt > maxCount) { maxCount = cnt; dominantType = tp as FieldType }
    }

    return { name, count, type: dominantType }
  })
})

// --- Filtered fields ---
const filteredFields = computed<FieldStat[]>(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return fieldStats.value
  return fieldStats.value.filter(f => f.name.toLowerCase().includes(q))
})

// --- Top N values aggregation ---
interface TopValue {
  value: string
  count: number
  percent: number
}

function computeTopValues(fieldName: string): TopValue[] {
  const counter = new Map<string, number>()
  let total = 0

  for (const entry of props.logEntries) {
    const raw = entry[fieldName]
    if (raw == null) continue
    const key = String(raw)
    counter.set(key, (counter.get(key) || 0) + 1)
    total++
  }

  if (total === 0) return []

  // Sort by count descending, take top 10
  const sorted = [...counter.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, 10)

  return sorted.map(([value, count]) => ({
    value,
    count,
    percent: Math.round((count / total) * 100),
  }))
}
</script>

<template>
  <div class="field-sidebar">
    <!-- Search input -->
    <div class="field-search">
      <NInput
        v-model:value="searchQuery"
        :placeholder="t('explore.searchFields')"
        size="small"
        clearable
      >
        <template #prefix>
          <NIcon :size="14" color="var(--sre-text-tertiary)">
            <SearchOutline />
          </NIcon>
        </template>
      </NInput>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="field-loading">
      <NSpin size="small" />
    </div>

    <!-- Field list -->
    <div v-else class="field-list">
      <div v-if="!filteredFields.length" class="field-empty">
        {{ searchQuery ? t('explore.noMatchingFields') : t('explore.noFieldsAvailable') }}
      </div>

      <NPopover
        v-for="field in filteredFields"
        :key="field.name"
        placement="right"
        trigger="click"
        :style="{ width: '340px' }"
        :show-arrow="false"
      >
        <template #trigger>
          <div class="field-item">
            <!-- Type icon -->
            <span
              class="field-type-icon"
              :style="{ color: typeIconMap[field.type].color }"
            >
              {{ typeIconMap[field.type].text }}
            </span>

            <!-- Field name -->
            <NTooltip placement="top" :delay="500">
              <template #trigger>
                <span class="field-name">{{ field.name }}</span>
              </template>
              {{ field.name }}
            </NTooltip>

            <!-- Count badge -->
            <NBadge
              :value="field.count"
              :max="999"
              class="field-count"
            />
          </div>
        </template>

        <!-- Popover content: Top 10 values -->
        <div class="popover-content">
          <div class="popover-title">{{ field.name }}</div>
          <div class="popover-subtitle">Top 10 Values</div>

          <div class="popover-values">
            <div
              v-for="(item, idx) in computeTopValues(field.name)"
              :key="idx"
              class="value-row"
            >
              <div class="value-info">
                <NTooltip placement="top" :delay="300">
                  <template #trigger>
                    <span class="value-text">{{ item.value || '(empty)' }}</span>
                  </template>
                  {{ item.value || '(empty)' }}
                </NTooltip>
                <span class="value-count">{{ item.count }}</span>
              </div>
              <div class="value-bar-wrap">
                <NProgress
                  type="line"
                  :percentage="item.percent"
                  :show-indicator="false"
                  :height="4"
                  :border-radius="2"
                  color="var(--sre-primary)"
                  rail-color="var(--sre-bg-sunken)"
                />
              </div>
              <div class="value-actions">
                <NTooltip placement="top">
                  <template #trigger>
                    <NButton
                      size="tiny"
                      quaternary
                      @click="emit('addFieldFilter', field.name, item.value)"
                    >
                      <template #icon>
                        <NIcon :size="14">
                          <AddCircleOutline />
                        </NIcon>
                      </template>
                    </NButton>
                  </template>
                  AND filter
                </NTooltip>
              </div>
            </div>

            <div
              v-if="!computeTopValues(field.name).length"
              class="popover-empty"
            >
              No values found
            </div>
          </div>
        </div>
      </NPopover>
    </div>
  </div>
</template>

<style scoped>
.field-sidebar {
  width: 240px;
  height: 100%;
  border-right: 1px solid var(--sre-border);
  display: flex;
  flex-direction: column;
  background: var(--sre-bg-card);
}

.field-search {
  flex-shrink: 0;
}

.field-search :deep(.n-input) {
  border-radius: 0;
  border-top: none;
  border-left: none;
  border-right: none;
}

.field-loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.field-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.field-empty {
  padding: 16px 12px;
  text-align: center;
  font-size: 12px;
  color: var(--sre-text-tertiary);
}

.field-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  cursor: pointer;
  min-height: 28px;
  transition: background 0.15s;
}

.field-item:hover {
  background: var(--sre-bg-hover);
}

.field-type-icon {
  width: 16px;
  height: 16px;
  border-radius: 3px;
  background: var(--sre-bg-sunken, #f1f5f9);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 600;
  flex-shrink: 0;
  line-height: 1;
}

.field-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--sre-text-primary);
  font-family: var(--sre-font-mono, monospace);
}

.field-count {
  flex-shrink: 0;
}

.field-count :deep(.n-badge-sup) {
  font-size: 10px;
}

/* Popover content */
.popover-content {
  padding: 4px 0;
}

.popover-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
  padding: 0 4px 4px;
  font-family: var(--sre-font-mono, monospace);
  word-break: break-all;
}

.popover-subtitle {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  padding: 0 4px 8px;
  border-bottom: 1px solid var(--sre-border);
  margin-bottom: 6px;
}

.popover-values {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.value-row {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 4px;
  border-radius: 4px;
  transition: background 0.15s;
}

.value-row:hover {
  background: var(--sre-bg-hover);
}

.value-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.value-text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--sre-text-primary);
  font-family: var(--sre-font-mono, monospace);
}

.value-count {
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  font-variant-numeric: tabular-nums;
}

.value-bar-wrap {
  padding-right: 28px;
}

.value-bar-wrap :deep(.n-progress) {
  --n-rail-height: 4px !important;
}

.value-actions {
  position: absolute;
  right: 4px;
  top: 50%;
  transform: translateY(-50%);
}

.popover-empty {
  text-align: center;
  padding: 12px 4px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
}
</style>
