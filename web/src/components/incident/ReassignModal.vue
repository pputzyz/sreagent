<script setup lang="ts">
/**
 * ReassignModal — search users and reassign an incident.
 * Extracted from incident/Detail.vue (FlashCat Phase 6).
 */
import { ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { incidentApi, userApi } from '@/api'
import type { User } from '@/types'

const props = defineProps<{
  show: boolean
  incidentId: number
}>()

const emit = defineEmits<{
  (e: 'update:show', val: boolean): void
  (e: 'done'): void
}>()

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const search = ref('')
const searchLoading = ref(false)
const users = ref<User[]>([])
const userId = ref<number | null>(null)

const allUsers = ref<User[]>([])

async function fetchUsers() {
  searchLoading.value = true
  try {
    const res = await userApi.list({ page: 1, page_size: 50 })
    allUsers.value = res.data.data?.list ?? []
    filterUsers()
  } catch (e: any) { message.error(e?.message ?? t('incident.searchFailed')) } finally { searchLoading.value = false }
}

function filterUsers() {
  const q = search.value.toLowerCase()
  users.value = q
    ? allUsers.value.filter(u =>
        (u.username?.toLowerCase().includes(q)) ||
        (u.display_name?.toLowerCase().includes(q)))
    : allUsers.value
}

watch(() => props.show, (v) => {
  if (v) {
    search.value = ''
    userId.value = null
    if (!allUsers.value.length) fetchUsers()
  }
})

async function doReassign() {
  if (!userId.value) { message.warning(t('incident.selectAssignee')); return }
  loading.value = true
  try {
    await incidentApi.reassign(props.incidentId, userId.value)
    message.success(t('incident.reassignSuccess'))
    emit('update:show', false)
    userId.value = null
    emit('done')
  } catch (e: any) { message.error(e?.message ?? t('incident.opFailed')) } finally { loading.value = false }
}
</script>

<template>
  <n-modal
    :show="show"
    :title="t('incident.reassignLabel')"
    preset="card"
    class="reassign-modal"
    :bordered="false"
    @update:show="emit('update:show', $event)"
  >
    <n-input
      v-model:value="search"
      :placeholder="t('incident.searchUserHint')"
      clearable
      class="search-input"
      @update:value="filterUsers"
    />
    <n-spin :show="searchLoading">
      <div class="picker-list">
        <div
          v-for="u in users" :key="u.id"
          class="picker-row user-row"
          :class="{ selected: userId === u.id }"
          @click="userId = u.id"
        >
          <n-avatar size="small" round>
            {{ (u.display_name || u.username).charAt(0).toUpperCase() }}
          </n-avatar>
          <div class="user-meta">
            <div class="user-name">{{ u.display_name || u.username }}</div>
            <div class="user-handle">{{ u.username }}</div>
          </div>
        </div>
      </div>
    </n-spin>
    <template #footer>
      <n-space justify="end">
        <n-button @click="emit('update:show', false)">{{ t('incident.cancelBtn') }}</n-button>
        <n-button type="primary" :loading="loading" :disabled="!userId" @click="doReassign">
          {{ t('incident.confirmReassign') }}
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
.reassign-modal {
  width: 460px;
}

.search-input {
  margin-bottom: 12px;
}

.picker-list {
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

.user-row {
  padding: 10px 12px;
}

.user-meta {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.user-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-text-primary);
}

.user-handle {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  font-family: var(--sre-font-mono);
}
</style>
