<script setup lang="ts">
/**
 * LogDetailDrawer — Nightingale NavigableDrawer-style log detail panel.
 * Right-side drawer with Table/JSON tabs, prev/next navigation,
 * keyboard shortcuts (ArrowUp/Down), and resizable width.
 */
import { ref, computed, watch, onMounted, onUnmounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NDrawer, NDrawerContent, NTabs, NTabPane, NDataTable,
  NButton, NIcon, NTooltip, NButtonGroup,
} from 'naive-ui'
import { CopyOutline } from '@vicons/ionicons5'
import type { LogEntry } from '@/types'

const props = defineProps<{
  show: boolean
  logEntry: LogEntry | null
  logEntries: LogEntry[]
  currentIndex: number
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'prev'): void
  (e: 'next'): void
  (e: 'update:currentIndex', value: number): void
}>()

const { t } = useI18n()

// Size selector: Small=35%, Medium=55%, Large=75%
type SizeType = 'small' | 'medium' | 'large'
const currentSize = ref<SizeType>('small')
const sizeWidthMap: Record<SizeType, string> = { small: '35%', medium: '55%', large: '75%' }
const drawerWidth = computed(() => sizeWidthMap[currentSize.value])

// Navigation
const hasPrev = computed(() => props.currentIndex > 0)
const hasNext = computed(() => props.currentIndex < props.logEntries.length - 1)

function onPrev() { if (hasPrev.value) emit('prev') }
function onNext() { if (hasNext.value) emit('next') }

// Keyboard shortcuts
function handleKeydown(e: KeyboardEvent) {
  if (!props.show) return
  if (e.key === 'ArrowUp' && hasPrev.value) { e.preventDefault(); onPrev() }
  else if (e.key === 'ArrowDown' && hasNext.value) { e.preventDefault(); onNext() }
  else if (e.key === 'Escape') { e.preventDefault(); emit('update:show', false) }
}

onMounted(() => document.addEventListener('keydown', handleKeydown))
onUnmounted(() => document.removeEventListener('keydown', handleKeydown))

// Drawer title: formatted timestamp
const drawerTitle = computed(() => {
  if (!props.logEntry) return t('query.logDetail')
  const ts = props.logEntry.timestamp
  if (!ts) return t('query.logDetail')
  const d = new Date(typeof ts === 'number' ? (ts < 1e12 ? ts * 1000 : ts) : ts)
  const pad = (n: number) => String(n).padStart(2, '0')
  const ms = String(d.getMilliseconds()).padStart(3, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}.${ms}`
})

// Active tab
const activeDetailTab = ref('table')

// --- Type icon for fields ---
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

// Table tab data
const detailTableData = computed(() => {
  if (!props.logEntry) return []
  const labels = props.logEntry.labels || {}
  const entries: Array<{ field: string; value: string; type: FieldType; _rawValue: unknown }> = []
  // Add timestamp
  if (props.logEntry.timestamp) {
    entries.push({ field: 'timestamp', value: drawerTitle.value, type: 'date', _rawValue: props.logEntry.timestamp })
  }
  // Add message
  if (props.logEntry.message) {
    entries.push({ field: 'message', value: props.logEntry.message, type: 'string', _rawValue: props.logEntry.message })
  }
  // Add all labels
  for (const [k, v] of Object.entries(labels)) {
    entries.push({ field: k, value: String(v ?? ''), type: inferFieldType(v), _rawValue: v })
  }
  return entries
})

const detailColumns = [
  {
    title: '',
    key: 'type',
    width: 32,
    render: (row: { type: FieldType }) => {
      const icon = typeIconMap[row.type]
      return h('span', {
        style: `display:inline-flex;align-items:center;justify-content:center;width:16px;height:16px;border-radius:3px;background:var(--sre-bg-sunken,#f1f5f9);font-size:11px;font-weight:600;color:${icon.color};`,
      }, icon.text)
    },
  },
  {
    title: '',
    key: 'field',
    width: 180,
    render: (row: { field: string; type: FieldType }) =>
      h('span', { style: 'color:var(--sre-text-tertiary);font-size:12px;font-family:var(--sre-font-mono,monospace);' }, row.field),
  },
  {
    title: '',
    key: 'value',
    ellipsis: { tooltip: true },
    render: (row: { value: string }) =>
      h('span', { style: 'font-size:12px;font-family:var(--sre-font-mono,monospace);white-space:pre-wrap;word-break:break-all;color:var(--sre-text-primary);' }, row.value),
  },
]

// JSON tab
const jsonContent = computed(() => {
  if (!props.logEntry) return ''
  const obj: Record<string, unknown> = {
    timestamp: props.logEntry.timestamp,
    message: props.logEntry.message,
    ...props.logEntry.labels,
  }
  return JSON.stringify(obj, null, 2)
})

// Copy to clipboard
function copyToClipboard() {
  if (!props.logEntry) return
  const obj = {
    timestamp: props.logEntry.timestamp,
    message: props.logEntry.message,
    ...props.logEntry.labels,
  }
  navigator.clipboard?.writeText(JSON.stringify(obj, null, 4))
}
</script>

<template>
  <NDrawer
    :show="show"
    :width="drawerWidth"
    placement="right"
    :mask="false"
    :style="{ position: 'absolute' }"
    @update:show="(v: boolean) => emit('update:show', v)"
  >
    <NDrawerContent :title="drawerTitle" closable>
      <template #header>
        <div class="drawer-header">
          <span class="drawer-title">{{ drawerTitle }}</span>
          <div class="drawer-header-actions">
            <NButtonGroup size="tiny">
              <NButton :type="currentSize === 'small' ? 'primary' : 'default'" :secondary="currentSize !== 'small'" @click="currentSize = 'small'">{{ t('query.small') }}</NButton>
              <NButton :type="currentSize === 'medium' ? 'primary' : 'default'" :secondary="currentSize !== 'medium'" @click="currentSize = 'medium'">{{ t('query.medium') }}</NButton>
              <NButton :type="currentSize === 'large' ? 'primary' : 'default'" :secondary="currentSize !== 'large'" @click="currentSize = 'large'">{{ t('query.large') }}</NButton>
            </NButtonGroup>
          </div>
        </div>
      </template>
      <template #default>
        <div v-if="logEntry" class="drawer-body">
          <!-- Nav buttons -->
          <div class="drawer-nav">
            <NButton size="tiny" quaternary :disabled="!hasPrev" @click="onPrev">
              &uarr; {{ t('query.prevLog') }}
            </NButton>
            <span class="nav-position">{{ currentIndex + 1 }} / {{ logEntries.length }}</span>
            <NButton size="tiny" quaternary :disabled="!hasNext" @click="onNext">
              {{ t('query.nextLog') }} &darr;
            </NButton>
          </div>

          <!-- Tabs: Table + JSON -->
          <NTabs v-model:value="activeDetailTab" type="line" size="small" class="detail-tabs">
            <template #suffix>
              <NButton size="tiny" quaternary @click="copyToClipboard">
                <template #icon><NIcon><CopyOutline /></NIcon></template>
                {{ t('query.copyField') }}
              </NButton>
            </template>
            <NTabPane name="table" :tab="t('query.table')">
              <NDataTable
                :columns="detailColumns"
                :data="detailTableData"
                :row-key="(r: any) => r.field"
                :show-header="false"
                size="small"
                :bordered="false"
                max-height="calc(100vh - 200px)"
                virtual-scroll
              />
            </NTabPane>
            <NTabPane name="json" tab="JSON">
              <pre class="json-content">{{ jsonContent }}</pre>
            </NTabPane>
          </NTabs>
        </div>

        <!-- Keyboard hint -->
        <div class="drawer-hint">
          {{ t('query.drawerHint') }}
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped>
.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 12px;
}
.drawer-title {
  font-family: var(--sre-font-mono, monospace);
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
}
.drawer-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.drawer-body {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.drawer-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid var(--sre-border);
  margin-bottom: 8px;
}
.nav-position {
  font-size: 12px;
  color: var(--sre-text-tertiary);
}
.detail-tabs {
  flex: 1;
  min-height: 0;
}
.json-content {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  line-height: 1.5;
  color: var(--sre-text-primary);
  background: var(--sre-bg-sunken, #f8fafc);
  border-radius: 6px;
  padding: 12px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: calc(100vh - 240px);
}
.drawer-hint {
  position: absolute;
  bottom: 8px;
  left: 12px;
  font-size: 10px;
  color: var(--sre-text-tertiary);
  pointer-events: none;
}
</style>
