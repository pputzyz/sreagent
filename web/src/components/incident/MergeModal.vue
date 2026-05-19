<script setup lang="ts">
/**
 * MergeModal — search and merge current incident into another.
 * Extracted from incident/Detail.vue (FlashCat Phase 6).
 */
import { ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { incidentApi } from '@/api'
import type { Incident } from '@/types'
import { getErrorMessage } from '@/utils/format'

const props = defineProps<{
  show: boolean
  incidentId: number
}>()

const emit = defineEmits<{
  (e: 'update:show', val: boolean): void
  (e: 'done', targetId: number): void
}>()

const { t } = useI18n()
const message = useMessage()

const triggerEl = ref<HTMLElement | null>(null)

watch(() => props.show, (v) => {
  if (v) triggerEl.value = document.activeElement as HTMLElement
})

function handleAfterLeave() {
  triggerEl.value?.focus()
}

const loading = ref(false)
const search = ref('')
const searchLoading = ref(false)
const results = ref<Incident[]>([])
const targetId = ref<number | null>(null)

async function searchIncidents() {
  if (!search.value.trim()) return
  searchLoading.value = true
  try {
    const res = await incidentApi.list({ query: search.value, page: 1, page_size: 10 })
    results.value = (res.data.data?.list ?? []).filter((i: Incident) => i.id !== props.incidentId)
  } catch (e: unknown) { message.error(getErrorMessage(e) || t('incident.searchFailed')) } finally { searchLoading.value = false }
}

async function doMerge() {
  if (!targetId.value) { message.warning(t('incident.selectTargetIncident')); return }
  loading.value = true
  try {
    await incidentApi.merge(props.incidentId, targetId.value)
    message.success(t('incident.mergeSuccess'))
    emit('update:show', false)
    emit('done', targetId.value)
  } catch (e: unknown) { message.error(getErrorMessage(e) || t('incident.opFailed')) } finally { loading.value = false }
}
</script>

<template>
  <n-modal
    :show="show"
    :title="t('incident.mergeToTarget')"
    preset="card"
    class="merge-modal"
    :bordered="false"
    @update:show="emit('update:show', $event)"
    @after-leave="handleAfterLeave"
  >
    <p class="modal-hint">
      {{ t('incident.mergeDescription') }}
    </p>
    <n-input-group>
      <n-input
        v-model:value="search"
        :placeholder="t('incident.searchIncidentHint')"
        @keydown.enter="searchIncidents"
      />
      <n-button :loading="searchLoading" @click="searchIncidents">{{ t('incident.searchBtn') }}</n-button>
    </n-input-group>
    <div v-if="results.length" class="picker-list">
      <div
        v-for="inc in results" :key="inc.id"
        class="picker-row sre-row-card"
        :class="{ selected: targetId === inc.id }"
        :data-severity="inc.severity"
        @click="targetId = inc.id"
      >
        <span class="sre-dot" :data-severity="inc.severity" />
        <span class="picker-id tnum">#{{ inc.id }}</span>
        <span class="picker-title">{{ inc.title }}</span>
      </div>
    </div>
    <n-empty v-else-if="search && !searchLoading" :description="t('incident.noMatchingIncident')" class="merge-empty" />
    <template #footer>
      <n-space justify="end">
        <n-button @click="emit('update:show', false)">{{ t('incident.cancelBtn') }}</n-button>
        <n-popconfirm @positive-click="doMerge">
          <template #trigger>
            <n-button type="error" :loading="loading" :disabled="!targetId">
              {{ t('incident.confirmMerge') }}
            </n-button>
          </template>
          {{ t('incident.confirmMergeMsg') }}
        </n-popconfirm>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
.merge-modal {
  width: 540px;
}

.modal-hint {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin: 0 0 12px;
  line-height: 1.5;
}

.picker-list {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 280px;
  overflow-y: auto;
}

.picker-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: var(--sre-radius-sm);
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  cursor: pointer;
  transition: background 120ms ease;
}

.picker-row:hover { background: var(--sre-bg-hover); }

.picker-row.selected {
  background: var(--sre-primary-soft);
  border-color: var(--sre-primary);
}

.picker-id {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  font-weight: 500;
}

.picker-title {
  font-size: 13px;
  color: var(--sre-text-primary);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.merge-empty {
  padding: 16px 0;
}
</style>
