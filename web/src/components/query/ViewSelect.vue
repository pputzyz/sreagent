<script setup lang="ts">
/**
 * ViewSelect -- save/restore query state.
 * Backend API primary, localStorage fallback.
 */
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton, NIcon, NPopover, NInput, NTag, NTooltip,
  useMessage, useDialog,
} from 'naive-ui'
import {
  BookmarkOutline, TrashOutline, SearchOutline, CopyOutline,
} from '@vicons/ionicons5'
import { savedViewApi } from '@/api/saved-views'
import type { SavedViewApiItem } from '@/api/saved-views'

/** Local view shape used by the parent component. */
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
const apiAvailable = ref(true)

// ---- mapping helpers ----

function apiToLocal(item: SavedViewApiItem): SavedView {
  return {
    id: String(item.id),
    name: item.name,
    tab: item.tab,
    dsId: item.datasource_id,
    dsName: '', // backend does not return ds name; shown as fallback id
    expression: item.expression,
    createdAt: new Date(item.created_at).getTime(),
  }
}

function loadLocal(): SavedView[] {
  try {
    const raw = localStorage.getItem(VIEWS_KEY)
    return raw ? JSON.parse(raw) || [] : []
  } catch {
    return []
  }
}

function saveLocal(list: SavedView[]) {
  try { localStorage.setItem(VIEWS_KEY, JSON.stringify(list)) } catch { /* ignore */ }
}

// ---- load ----

async function loadViews() {
  if (!apiAvailable.value) {
    views.value = loadLocal()
    return
  }
  try {
    const res = await savedViewApi.list({ tab: props.currentTab, page: 1, page_size: 200 })
    const items: SavedViewApiItem[] = res.data.data?.list ?? []
    views.value = items.map(apiToLocal)
    // keep localStorage in sync
    saveLocal(views.value)
  } catch {
    apiAvailable.value = false
    views.value = loadLocal()
  }
}

// ---- filtered list ----

const filteredViews = computed(() => {
  const q = searchQuery.value.toLowerCase()
  if (!q) return views.value
  return views.value.filter(v =>
    v.name.toLowerCase().includes(q) ||
    v.expression.toLowerCase().includes(q) ||
    v.dsName.toLowerCase().includes(q)
  )
})

// ---- save current ----

function canSave(): boolean {
  return !!(props.currentDsId && props.currentExpression.trim())
}

async function saveCurrentView() {
  if (!canSave()) return
  const name = viewName.value.trim() || `${props.currentDsName}: ${props.currentExpression.slice(0, 40)}`

  if (apiAvailable.value) {
    try {
      await savedViewApi.create({
        name,
        tab: props.currentTab,
        datasource_id: props.currentDsId!,
        expression: props.currentExpression,
      })
      await loadViews()
      viewName.value = ''
      message.success(t('query.viewSaved'))
      return
    } catch {
      // fall through to localStorage
    }
  }

  // localStorage fallback
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
  saveLocal(views.value)
  viewName.value = ''
  message.success(t('query.viewSaved'))
}

// ---- load a view ----

function loadView(view: SavedView) {
  emit('load', view)
  popoverVisible.value = false
}

// ---- delete ----

async function deleteView(id: string) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('common.confirmDeleteMsg'),
    onPositiveClick: async () => {
      if (apiAvailable.value) {
        try {
          await savedViewApi.delete(Number(id))
          await loadViews()
          message.success(t('common.deleteSuccess'))
          return
        } catch {
          // fall through
        }
      }
      views.value = views.value.filter(v => v.id !== id)
      saveLocal(views.value)
      message.success(t('common.deleteSuccess'))
    },
  })
}

// ---- copy ----

async function copyView(id: string) {
  if (apiAvailable.value) {
    try {
      await savedViewApi.copy(Number(id))
      await loadViews()
      message.success(t('query.viewCopied'))
      return
    } catch {
      // fall through
    }
  }
  // localStorage fallback: duplicate in-memory
  const src = views.value.find(v => v.id === id)
  if (!src) return
  const copy: SavedView = {
    ...src,
    id: Date.now().toString(36) + Math.random().toString(36).slice(2, 6),
    name: src.name + ' (copy)',
    createdAt: Date.now(),
  }
  views.value.unshift(copy)
  saveLocal(views.value)
  message.success(t('query.viewCopied'))
}

// ---- helpers ----

function formatTime(ts: number): string {
  return new Date(ts).toLocaleString()
}

// init
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
            <div class="view-item-actions">
              <NTooltip>
                <template #trigger>
                  <NButton
                    size="tiny"
                    quaternary
                    @click.stop="copyView(view.id)"
                  >
                    <template #icon><NIcon size="14"><CopyOutline /></NIcon></template>
                  </NButton>
                </template>
                {{ t('query.copyView') }}
              </NTooltip>
              <NButton
                size="tiny"
                quaternary
                type="error"
                @click.stop="deleteView(view.id)"
              >
                <template #icon><NIcon size="14"><TrashOutline /></NIcon></template>
              </NButton>
            </div>
          </div>
          <div class="view-item-meta">
            <NTag size="tiny" :bordered="false" :type="view.tab === 'logs' ? 'warning' : 'info'">
              {{ view.tab === 'logs' ? 'Logs' : 'Metrics' }}
            </NTag>
            <span v-if="view.dsName" class="view-item-ds">{{ view.dsName }}</span>
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
.view-item-actions {
  display: flex;
  gap: 2px;
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
