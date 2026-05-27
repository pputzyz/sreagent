<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import {
  teamNotifyChannelApi,
  teamApi,
  notifyMediaApi,
} from '@/api'
import type {
  TeamNotifyChannel,
  Team,
  NotifyMedia,
} from '@/types'
import { getErrorMessage } from '@/utils/format'
import { AddOutline, SearchOutline, TrashOutline, StarOutline, Star } from '@vicons/ionicons5'

const message = useMessage()
const { t } = useI18n()

// --- State ---
const selectedTeamId = ref<number | null>(null)
const channels = ref<TeamNotifyChannel[]>([])
const loading = ref(false)
const saving = ref(false)

// --- Reference data ---
const teams = shallowRef<Team[]>([])
const mediaList = shallowRef<NotifyMedia[]>([])

const teamOptions = computed(() =>
  teams.value.map(tm => ({ label: tm.name, value: tm.id })),
)

const availableMediaOptions = computed(() => {
  const existing = new Set(channels.value.map(c => c.media_id))
  return mediaList.value
    .filter(m => !existing.has(m.id))
    .map(m => ({ label: `${m.name} (${m.type})`, value: m.id }))
})

const selectedAddMediaId = ref<number | null>(null)

// --- Fetch helpers ---
async function fetchTeams() {
  try {
    const { data } = await teamApi.list({ page: 1, page_size: 100 })
    teams.value = data.data?.list || []
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
}

async function fetchMedia() {
  try {
    const { data } = await notifyMediaApi.list({ page: 1, page_size: 200 })
    mediaList.value = data.data?.list || []
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
}

async function fetchChannels() {
  if (!selectedTeamId.value) { channels.value = []; return }
  loading.value = true
  try {
    const { data } = await teamNotifyChannelApi.list(selectedTeamId.value)
    channels.value = data.data || []
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
  finally { loading.value = false }
}

// --- Actions ---
async function handleAdd() {
  if (!selectedTeamId.value || !selectedAddMediaId.value) return
  saving.value = true
  try {
    await teamNotifyChannelApi.create({
      team_id: selectedTeamId.value,
      media_id: selectedAddMediaId.value,
    })
    selectedAddMediaId.value = null
    message.success(t('teamChannel.added'))
    await fetchChannels()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
  finally { saving.value = false }
}

async function handleSetDefault(id: number) {
  try {
    await teamNotifyChannelApi.setDefault(id)
    message.success(t('teamChannel.defaultSet'))
    await fetchChannels()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
}

async function handleDelete(id: number) {
  try {
    await teamNotifyChannelApi.delete(id)
    message.success(t('teamChannel.removed'))
    await fetchChannels()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
}

function getMediaName(mediaId: number): string {
  return mediaList.value.find(m => m.id === mediaId)?.name || `#${mediaId}`
}

function getMediaType(mediaId: number): string {
  return mediaList.value.find(m => m.id === mediaId)?.type || '—'
}

// --- Lifecycle ---
watch(selectedTeamId, () => { fetchChannels() })
onMounted(() => { fetchTeams(); fetchMedia() })
</script>

<template>
  <div class="team-channel-config">
    <div class="tc-header">
      <div>
        <div class="tc-title">{{ t('teamChannel.title') }}</div>
        <div class="tc-subtitle">{{ t('teamChannel.subtitle') }}</div>
      </div>
    </div>

    <div class="tc-toolbar">
      <n-select
        v-model:value="selectedTeamId"
        :options="teamOptions"
        :placeholder="t('teamChannel.selectTeam')"
        filterable
        clearable
        size="small"
        style="width: 240px"
      />
    </div>

    <template v-if="selectedTeamId">
      <div class="tc-add-row">
        <n-select
          v-model:value="selectedAddMediaId"
          :options="availableMediaOptions"
          :placeholder="t('teamChannel.addChannel')"
          filterable
          size="small"
          style="flex: 1"
        />
        <n-button
          type="primary"
          size="small"
          :disabled="!selectedAddMediaId"
          :loading="saving"
          @click="handleAdd"
        >
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('teamChannel.addChannel') }}
        </n-button>
      </div>

      <div v-if="loading" class="tc-loading">
        <n-spin size="small" />
      </div>

      <div v-else-if="channels.length === 0" class="tc-empty">
        <n-empty :description="t('teamChannel.noChannels')" size="small" />
      </div>

      <div v-else class="tc-list">
        <div
          v-for="ch in channels"
          :key="ch.id"
          class="tc-row"
          :class="{ 'tc-row--default': ch.is_default }"
        >
          <div class="tc-row-main">
            <span class="tc-media-name">{{ getMediaName(ch.media_id) }}</span>
            <span class="tc-media-type">{{ getMediaType(ch.media_id) }}</span>
            <span v-if="ch.is_default" class="tc-default-badge">
              <n-icon :component="Star" size="14" />
              {{ t('teamChannel.defaultLabel') }}
            </span>
          </div>
          <div class="tc-row-actions">
            <n-button
              quaternary
              size="tiny"
              :type="ch.is_default ? 'warning' : 'default'"
              :disabled="ch.is_default"
              :title="t('teamChannel.setDefault')"
              @click="handleSetDefault(ch.id)"
            >
              <template #icon><n-icon :component="ch.is_default ? Star : StarOutline" /></template>
            </n-button>
            <n-button
              quaternary
              size="tiny"
              type="error"
              :title="t('teamChannel.removeChannel')"
              @click="handleDelete(ch.id)"
            >
              <template #icon><n-icon :component="TrashOutline" /></template>
            </n-button>
          </div>
        </div>
      </div>
    </template>

    <div v-else class="tc-no-team">
      <n-empty :description="t('teamChannel.selectTeam')" size="small" />
    </div>
  </div>
</template>

<style scoped>
.team-channel-config {
  font-family: var(--sre-font-sans);
}

.tc-header {
  margin-bottom: 16px;
}

.tc-title {
  font: 600 18px/1.2 var(--sre-font-sans), sans-serif;
  letter-spacing: -0.01em;
}

.tc-subtitle {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin-top: 4px;
}

.tc-toolbar {
  margin-bottom: 12px;
}

.tc-add-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}

.tc-loading {
  display: flex;
  justify-content: center;
  padding: 40px 0;
}

.tc-empty,
.tc-no-team {
  padding: 40px 20px;
  text-align: center;
}

.tc-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tc-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-radius: 6px;
  background: var(--sre-bg-elevated, rgba(255, 255, 255, 0.04));
  transition: background 0.15s;
}

.tc-row:hover {
  background: var(--sre-bg-hover, rgba(255, 255, 255, 0.08));
}

.tc-row--default {
  border-left: 3px solid var(--sre-warning, #f0a020);
}

.tc-row-main {
  display: flex;
  align-items: center;
  gap: 10px;
}

.tc-media-name {
  font: 600 13px/1.3 var(--sre-font-sans), sans-serif;
}

.tc-media-type {
  font: 500 10px/1 var(--sre-font-mono), monospace;
  text-transform: uppercase;
  padding: 3px 6px;
  border-radius: 4px;
  letter-spacing: 0.04em;
  background: var(--sre-bg-elevated, rgba(255, 255, 255, 0.06));
  color: var(--sre-text-secondary);
}

.tc-default-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font: 500 11px/1 var(--sre-font-sans), sans-serif;
  color: var(--sre-warning, #f0a020);
}

.tc-row-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}
</style>
