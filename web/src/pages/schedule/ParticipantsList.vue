<script setup lang="ts">
import { ref, computed } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { scheduleApi } from '@/api'
import type { ScheduleParticipant, User } from '@/types'
import { AlertCircleOutline } from '@vicons/ionicons5'
import EmptyState from '@/components/common/EmptyState.vue'

const props = defineProps<{
  scheduleId: number | null
  users: User[]
  getUserColor: (userId: number) => string
  getUserName: (userId: number) => string
}>()

const message = useMessage()
const { t } = useI18n()

const participants = ref<ScheduleParticipant[]>([])
const loading = ref(false)
const fetchError = ref(false)
const selectedUserId = ref<number | null>(null)
const saving = ref(false)

const userOptions = computed(() =>
  props.users.map(u => {
    const type = u.user_type && u.user_type !== 'human' ? ` [${u.user_type === 'bot' ? '\u{1F916}' : '\u{1F4E2}'}]` : ''
    return {
      label: (u.display_name || u.username) + type,
      value: u.id,
    }
  })
)

async function fetchParticipants() {
  if (!props.scheduleId) return
  loading.value = true
  fetchError.value = false
  try {
    const { data } = await scheduleApi.getParticipants(props.scheduleId)
    participants.value = data.data || []
  } catch {
    fetchError.value = true
    participants.value = []
  } finally {
    loading.value = false
  }
}

async function addParticipant() {
  if (!selectedUserId.value || !props.scheduleId) return
  if (participants.value.find(p => p.user_id === selectedUserId.value)) {
    message.warning(t('schedule.participantExists'))
    return
  }
  const newPosition = participants.value.length
  const updatedList = [
    ...participants.value.map(p => ({ user_id: p.user_id, position: p.position })),
    { user_id: selectedUserId.value, position: newPosition },
  ]
  saving.value = true
  try {
    await scheduleApi.setParticipants(props.scheduleId, updatedList)
    message.success(t('schedule.participantAdded'))
    selectedUserId.value = null
    await fetchParticipants()
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

async function removeParticipant(userId: number) {
  if (!props.scheduleId) return
  const updatedList = participants.value
    .filter(p => p.user_id !== userId)
    .map((p, idx) => ({ user_id: p.user_id, position: idx }))
  saving.value = true
  try {
    await scheduleApi.setParticipants(props.scheduleId, updatedList)
    message.success(t('schedule.participantRemoved'))
    await fetchParticipants()
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

async function moveParticipant(index: number, direction: 'up' | 'down') {
  if (!props.scheduleId) return
  const arr = [...participants.value]
  const targetIdx = direction === 'up' ? index - 1 : index + 1
  if (targetIdx < 0 || targetIdx >= arr.length) return
  ;[arr[index], arr[targetIdx]] = [arr[targetIdx], arr[index]]
  const updatedList = arr.map((p, idx) => ({ user_id: p.user_id, position: idx }))
  saving.value = true
  try {
    await scheduleApi.setParticipants(props.scheduleId, updatedList)
    await fetchParticipants()
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

defineExpose({ fetchParticipants })
</script>

<template>
  <!-- Error state -->
  <div v-if="fetchError && !loading" class="participant-error">
    <EmptyState
      :icon="AlertCircleOutline"
      :title="t('schedule.loadParticipantsError')"
      :description="t('schedule.loadParticipantsErrorHint')"
      :primary-text="t('common.retry')"
      @primary="fetchParticipants"
      size="sm"
      variant="warning"
    />
  </div>

  <n-spin v-else :show="loading">
    <div class="participants-section">
      <div class="participants-list">
        <div
          v-for="(p, index) in participants"
          :key="p.id"
          class="participant-item"
        >
          <div
            class="participant-position"
            :style="{ '--pos-color': getUserColor(p.user_id) }"
          >
            {{ index + 1 }}
          </div>
          <div
            class="participant-color-dot"
            :style="{ '--dot-color': getUserColor(p.user_id) }"
          />
          <span class="participant-name">{{ p.user?.display_name || p.user?.username || getUserName(p.user_id) }}</span>
          <div class="participant-actions">
            <n-button class="arrow-btn" size="tiny" quaternary :disabled="index === 0" @click="moveParticipant(index, 'up')">&#8593;</n-button>
            <n-button class="arrow-btn" size="tiny" quaternary :disabled="index === participants.length - 1" @click="moveParticipant(index, 'down')">&#8595;</n-button>
            <n-popconfirm @positive-click="removeParticipant(p.user_id)">
              <template #trigger>
                <n-button size="tiny" quaternary type="error">{{ t('common.remove') }}</n-button>
              </template>
              {{ t('schedule.removeParticipantConfirm') }}
            </n-popconfirm>
          </div>
        </div>

        <EmptyState
          v-if="!loading && participants.length === 0"
          :title="t('schedule.noParticipants')"
          size="sm"
        />
      </div>

      <div class="add-participant">
        <n-select
          v-model:value="selectedUserId"
          :options="userOptions"
          :placeholder="t('schedule.selectUserToAdd')"
          filterable
          size="small"
          style="flex: 1"
        />
        <n-button size="small" type="primary" :loading="saving" :disabled="!selectedUserId" @click="addParticipant">
          {{ t('common.add') }}
        </n-button>
      </div>
    </div>
  </n-spin>
</template>

<style scoped>
.participants-section {
  padding: 8px 0;
}

.participants-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
}

.participant-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 10px;
  background: var(--sre-bg-hover);
  border-radius: var(--sre-radius-sm);
}

.participant-position {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  font-size: 11px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: color-mix(in srgb, var(--pos-color) 13%, transparent);
  color: var(--pos-color);
}

.participant-color-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--dot-color);
}

.participant-name {
  font-size: 13px;
  color: var(--sre-text-primary);
}

.participant-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-left: auto;
}

.arrow-btn {
  min-width: 36px;
  min-height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
}

.add-participant {
  display: flex;
  gap: 8px;
  align-items: center;
}

.participant-error {
  padding: 24px 0;
}
</style>
