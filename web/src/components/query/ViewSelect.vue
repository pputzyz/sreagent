<script setup lang="ts">
/**
 * ViewSelect — save/restore query state.
 * Inspired by Nightingale ViewSelect pattern.
 * localStorage-backed for now (no backend API needed).
 */
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton, NIcon, NPopover, NInput, NTag, NTooltip,
  useMessage, useDialog,
} from 'naive-ui'
import {
  BookmarkOutline, TrashOutline, SearchOutline,
} from '@vicons/ionicons5'

export interface SavedView {
  id: string
  name: string
  tab: 'metrics' | 'logs'
  dsId: number
  dsName: string
  expression: string
  createdAt: number
}

const props = defineProps<{
  currentTab: 'metrics' | 'logs'
  currentDsId: number | null
  currentDsName: string
  currentExpression: string
}>()

const emit = defineEmits<{
  (e: 'load', view: SavedView): void
}>()

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const VIEWS_KEY = 'sre-saved-views'
const views = ref<SavedView[]>([])
const popoverVisible = ref(false)
const searchQuery = ref('')
const viewName = ref('')

function loadViews() {
  try {
    const raw = localStorage.getItem(VIEWS_KEY)
    if (raw) views.value = JSON.parse(raw) || []
  } catch { views.value = [] }
}

function saveViews() {
  try { localStorage.setItem(VIEWS_KEY, JSON.stringify(views.value)) } catch { /* ignore */ }
}

const filteredViews = computed(() => {
  const q = searchQuery.value.toLowerCase()
  if (!q) return views.value
  return views.value.filter(v =>
    v.name.toLowerCase().includes(q) ||
    v.expression.toLowerCase().includes(q) ||
    v.dsName.toLowerCase().includes(q)
  )
})

function canSave(): boolean {
  return !!(props.currentDsId && props.currentExpression.trim())
}

function saveCurrentView() {
  if (!canSave()) return
  const name = viewName.value.trim() || `${props.currentDsName}: ${props.currentExpression.slice(0, 40)}`
  const view: SavedView = {
    id: Date.now().toString(36) + Math.random().toString(36).slice(2, 6),
    name,
    tab: props.currentTab,
    dsId: props.currentDsId!,
    dsName: props.currentDsName,
    expression: props.currentExpression,
    createdAt: Date.now(),
  }
  views.value.unshift(view)
  saveViews()
  viewName.value = ''
  message.success(t('query.viewSaved'))
}

function loadView(view: SavedView) {
  emit('load', view)
  popoverVisible.value = false
}

function deleteView(id: string) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('common.confirmDeleteMsg'),
    onPositiveClick: () => {
      views.value = views.value.filter(v => v.id !== id)
      saveViews()
      message.success(t('common.deleteSuccess'))
    },
  })
}

function formatTime(ts: number): string {
  return new Date(ts).toLocaleString()
}

// Load on component creation
loadViews()
</script>

<template>
  <NPopover
    v-model:show="popoverVisible"
    trigger="click"
    placement="bottom-start"
    :style="{ width: '380px' }"
  >
    <template #trigger>
      <NTooltip>
        <template #trigger>
          <NButton size="small" quaternary>
            <template #icon><NIcon><BookmarkOutline /></NIcon></template>
          </NButton>
        </template>
        {{ t('query.savedViews') }}
      </NTooltip>
    </template>

    <div class="view-select">
      <!-- Save current -->
      <div class="view-save-row">
        <NInput
          v-model:value="viewName"
          :placeholder="t('query.viewNamePlaceholder')"
          size="small"
          class="view-name-input"
          @keydown.enter="saveCurrentView"
        />
        <NButton
          size="small"
          type="primary"
          :disabled="!canSave()"
          @click="saveCurrentView"
        >
          {{ t('query.saveView') }}
        </NButton>
      </div>

      <!-- Search -->
      <div class="view-search">
        <NInput
          v-model:value="searchQuery"
          :placeholder="t('common.search')"
          size="small"
          clearable
        >
          <template #prefix><NIcon size="14"><SearchOutline /></NIcon></template>
        </NInput>
      </div>

      <!-- Views list -->
      <div class="view-list">
        <div v-if="!filteredViews.length" class="view-empty">
          {{ searchQuery ? t('common.noData') : t('query.noSavedViews') }}
        </div>
        <div
          v-for="view in filteredViews"
          :key="view.id"
          class="view-item"
          @click="loadView(view)"
        >
          <div class="view-item-header">
            <span class="view-item-name">{{ view.name }}</span>
            <NButton
              size="tiny"
              quaternary
              type="error"
              @click.stop="deleteView(view.id)"
            >
              <template #icon><NIcon size="14"><TrashOutline /></NIcon></template>
            </NButton>
          </div>
          <div class="view-item-meta">
            <NTag size="tiny" :bordered="false" :type="view.tab === 'logs' ? 'warning' : 'info'">
              {{ view.tab === 'logs' ? 'Logs' : 'Metrics' }}
            </NTag>
            <span class="view-item-ds">{{ view.dsName }}</span>
          </div>
          <div class="view-item-expr">{{ view.expression }}</div>
          <div class="view-item-time">{{ formatTime(view.createdAt) }}</div>
        </div>
      </div>
    </div>
  </NPopover>
</template>

<style scoped>
.view-select {
  max-height: 420px;
  display: flex;
  flex-direction: column;
}
.view-save-row {
  display: flex;
  gap: 6px;
  margin-bottom: 8px;
}
.view-name-input {
  flex: 1;
}
.view-search {
  margin-bottom: 8px;
}
.view-list {
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}
.view-empty {
  text-align: center;
  padding: 16px;
  color: var(--sre-text-tertiary);
  font-size: 12px;
}
.view-item {
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
  border: 1px solid transparent;
  margin-bottom: 4px;
}
.view-item:hover {
  background: var(--sre-bg-hover);
  border-color: var(--sre-border);
}
.view-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.view-item-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-text-primary);
}
.view-item-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 4px 0;
}
.view-item-ds {
  font-size: 11px;
  color: var(--sre-text-tertiary);
}
.view-item-expr {
  font-family: var(--sre-font-mono, monospace);
  font-size: 11px;
  color: var(--sre-text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.view-item-time {
  font-size: 10px;
  color: var(--sre-text-tertiary);
  margin-top: 2px;
}
</style>
