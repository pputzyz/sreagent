<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  useMessage, useDialog,
  NButton, NIcon, NTabs, NTabPane, NSpin, NDropdown,
  NSelect, NPagination, NEmpty, NDescriptions, NDescriptionsItem, NTag,
  NForm, NFormItem, NInput, NInputNumber, NSwitch, NCheckbox, NRadioGroup, NRadio,
  NModal, NSpace,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { channelV2Api, incidentApi } from '@/api'
import NoiseConfig from './NoiseConfig.vue'
import DispatchConfig from './DispatchConfig.vue'
import type { Channel, Incident, ChannelStatus, ChannelAccessLevel } from '@/types'
import { formatTime, getErrorMessage } from '@/utils/format'
import {
  ArrowBackOutline, StarOutline, Star, EllipsisHorizontal,
  RefreshOutline, TrashOutline, CreateOutline,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const route = useRoute()
const router = useRouter()

const channelId = computed(() => Number(route.params.id))
const channel = shallowRef<Channel | null>(null)
const channelLoading = ref(false)

const incidents = shallowRef<Incident[]>([])
const incidentTotal = ref(0)
const incidentPage = ref(1)
const incidentPageSize = ref(20)
const incidentStatus = ref('')
const incidentLoading = ref(false)

const activeTab = ref('incidents')
const showEditModal = ref(false)
const saving = ref(false)
const starring = ref(false)

const editForm = ref<{
  name: string
  description: string
  status: ChannelStatus
  access_level: ChannelAccessLevel
  auto_close_enabled: boolean
  auto_close_minutes: number
  follow_alert_close: boolean
}>({
  name: '',
  description: '',
  status: 'active',
  access_level: 'public',
  auto_close_enabled: false,
  auto_close_minutes: 60,
  follow_alert_close: true,
})

// ─────────────────────────────────────────────────────────
// Loaders
// ─────────────────────────────────────────────────────────
async function loadChannel() {
  channelLoading.value = true
  try {
    const res = await channelV2Api.get(channelId.value)
    channel.value = res.data.data ?? null
    if (channel.value) {
      editForm.value = {
        name: channel.value.name,
        description: channel.value.description ?? '',
        status: channel.value.status,
        access_level: channel.value.access_level,
        auto_close_enabled: channel.value.auto_close_enabled,
        auto_close_minutes: channel.value.auto_close_minutes || 60,
        follow_alert_close: channel.value.follow_alert_close,
      }
    }
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.loadFailed'))
  } finally {
    channelLoading.value = false
  }
}

async function loadIncidents() {
  incidentLoading.value = true
  try {
    const res = await incidentApi.list({
      channel_id: channelId.value,
      status: incidentStatus.value,
      page: incidentPage.value,
      page_size: incidentPageSize.value,
    })
    incidents.value = res.data.data?.list ?? []
    incidentTotal.value = res.data.data?.total ?? 0
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.loadFailed'))
  } finally {
    incidentLoading.value = false
  }
}

async function saveChannel() {
  saving.value = true
  try {
    await channelV2Api.update(channelId.value, editForm.value)
    message.success(t('common.savedSuccess'))
    showEditModal.value = false
    await loadChannel()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function toggleStar() {
  if (!channel.value || starring.value) return
  starring.value = true
  try {
    if (channel.value.is_starred) {
      await channelV2Api.unstar(channelId.value)
    } else {
      await channelV2Api.star(channelId.value)
    }
    await loadChannel()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.saveFailed'))
  } finally {
    starring.value = false
  }
}

function confirmDelete() {
  if (!channel.value) return
  dialog.warning({
    title: t('common.confirmDelete'),
    content: channel.value.name,
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await channelV2Api.delete(channelId.value)
        message.success(t('common.deletedSuccess'))
        router.replace('/oncall/spaces')
      } catch (e: unknown) {
        message.error(getErrorMessage(e) || t('common.deleteFailed'))
      }
    },
  })
}

// ─────────────────────────────────────────────────────────
// Derived data — KPI
// ─────────────────────────────────────────────────────────
const kpi = computed(() => {
  const list = incidents.value
  const active = list.filter(i => i.status !== 'closed').length
  const today = list.length
  // MTTA / MTTR: derive simple averages over closed items, in seconds
  const closed = list.filter(i => i.status === 'closed' && i.triggered_at)
  let mtta = 0, mttr = 0
  if (closed.length) {
    let acks = 0, ackSum = 0, resSum = 0, resN = 0
    closed.forEach(i => {
      const trig = new Date(i.triggered_at).getTime()
      if (i.acknowledged_at) {
        acks++
        ackSum += (new Date(i.acknowledged_at).getTime() - trig)
      }
      if (i.closed_at) {
        resN++
        resSum += (new Date(i.closed_at).getTime() - trig)
      }
    })
    mtta = acks ? Math.round(ackSum / acks / 1000) : 0
    mttr = resN ? Math.round(resSum / resN / 1000) : 0
  }
  return {
    active,
    today,
    mtta: fmtDuration(mtta),
    mttr: fmtDuration(mttr),
  }
})

function fmtDuration(seconds: number): string {
  if (!seconds || seconds < 0) return '—'
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

// ─────────────────────────────────────────────────────────
// Helpers — severity / status labels
// ─────────────────────────────────────────────────────────
const severityToneMap: Record<string, string> = {
  critical: 'critical', warning: 'warning', info: 'info',
}
const statusLabelMap: Record<string, string> = {
  triggered: 'incident.statusTriggered',
  processing: 'incident.statusProcessing',
  closed: 'incident.statusClosed',
}

function relTime(ts?: string): string {
  if (!ts) return '—'
  const diff = (Date.now() - new Date(ts).getTime()) / 1000
  if (diff < 60) return t('common.secsAgo', { n: Math.floor(diff) })
  if (diff < 3600) return t('common.minsAgo', { n: Math.floor(diff / 60) })
  if (diff < 86400) return t('common.hoursAgo', { n: Math.floor(diff / 3600) })
  return t('common.daysAgo', { n: Math.floor(diff / 86400) })
}

const moreOptions = computed(() => [
  { label: t('common.delete'), key: 'delete', icon: () => h(NIcon, { component: TrashOutline }) },
])

function onMoreSelect(key: string) {
  if (key === 'delete') confirmDelete()
}

const statusFilterOptions = computed(() => [
  { label: t('incident.statusTriggered'), value: 'triggered' },
  { label: t('incident.statusProcessing'), value: 'processing' },
  { label: t('incident.statusClosed'), value: 'closed' },
])

onMounted(async () => {
  await loadChannel()
  await loadIncidents()
})
</script>

<template>
  <div class="channel-detail">
    <n-spin :show="channelLoading">
      <!-- Header -->
      <header class="cd-header sre-stagger">
        <div class="cd-header-left">
          <n-button quaternary circle size="small" @click="router.back()">
            <template #icon><n-icon :component="ArrowBackOutline" /></template>
          </n-button>
          <div class="cd-title-block">
            <h1 class="cd-title">{{ channel?.name ?? t('channel.title') }}</h1>
            <div class="cd-subtitle">
              <span v-if="channel?.description">{{ channel.description }}</span>
              <span v-if="channel?.description && channel?.team" class="sre-meta-divider">·</span>
              <span v-if="channel?.team">{{ channel.team.name }}</span>
            </div>
          </div>
        </div>
        <div class="cd-header-right">
          <n-button
            quaternary circle size="small" :loading="starring"
            :type="channel?.is_starred ? 'warning' : 'default'"
            @click="toggleStar"
          >
            <template #icon>
              <n-icon :component="channel?.is_starred ? Star : StarOutline" />
            </template>
          </n-button>
          <n-button size="small" @click="showEditModal = true">
            <template #icon><n-icon :component="CreateOutline" /></template>
            {{ t('common.edit') }}
          </n-button>
          <n-dropdown trigger="click" :options="moreOptions" @select="onMoreSelect">
            <n-button quaternary circle size="small">
              <template #icon><n-icon :component="EllipsisHorizontal" /></template>
            </n-button>
          </n-dropdown>
        </div>
      </header>

      <!-- KPI row -->
      <section class="kpi-row sre-stagger" v-if="channel">
        <div class="kpi-card sre-lift">
          <div class="kpi-value tnum">{{ kpi.active }}</div>
          <div class="sre-label-eyebrow">{{ t('incident.statusTriggered') }}</div>
          <div class="kpi-stripe" data-tone="critical"></div>
        </div>
        <div class="kpi-card sre-lift">
          <div class="kpi-value tnum">{{ kpi.today }}</div>
          <div class="sre-label-eyebrow">{{ t('incident.title') }}</div>
          <div class="kpi-stripe" data-tone="info"></div>
        </div>
        <div class="kpi-card sre-lift">
          <div class="kpi-value tnum">{{ kpi.mtta }}</div>
          <div class="sre-label-eyebrow">{{ t('dashboard.mtta') }}</div>
          <div class="kpi-stripe" data-tone="success"></div>
        </div>
        <div class="kpi-card sre-lift">
          <div class="kpi-value tnum">{{ kpi.mttr }}</div>
          <div class="sre-label-eyebrow">{{ t('dashboard.mttr') }}</div>
          <div class="kpi-stripe" data-tone="success"></div>
        </div>
      </section>

      <!-- Tabs -->
      <section class="cd-tabs" v-if="channel">
        <n-tabs v-model:value="activeTab" type="line" animated size="medium">
          <!-- Incidents -->
          <n-tab-pane name="incidents" :tab="t('incident.title')">
            <div class="tab-toolbar">
              <n-select
                v-model:value="incidentStatus"
                :options="statusFilterOptions"
                :placeholder="t('common.status')"
                clearable
                size="small"
                class="ch-select-150"
                @update:value="loadIncidents"
              />
              <n-button quaternary circle size="small" @click="loadIncidents">
                <template #icon><n-icon :component="RefreshOutline" /></template>
              </n-button>
            </div>

            <div class="incident-list">
              <div v-if="!incidentLoading && incidents.length === 0" class="empty-wrap">
                <n-empty :description="t('common.noData')" />
              </div>

              <div
                v-for="row in incidents"
                :key="row.id"
                class="sre-row-card incident-row"
                @click="router.push(`/oncall/incidents/${row.id}`)"
              >
                <span class="sre-dot" :data-tone="severityToneMap[row.severity] ?? 'default'"></span>
                <span class="i-id tnum">#{{ row.id }}</span>
                <span class="i-title">{{ row.title }}</span>
                <span class="i-status" :data-status="row.status">
                  {{ t(statusLabelMap[row.status] ?? row.status) }}
                </span>
                <span class="i-meta tnum">
                  {{ row.alert_count }} {{ t('incident.alertCount') }}
                </span>
                <span class="i-meta">
                  {{ row.assigned_user?.display_name ?? row.assigned_user?.username ?? '—' }}
                </span>
                <span class="i-time tnum">{{ relTime(row.triggered_at) }}</span>
              </div>
            </div>

            <div v-if="incidentTotal > incidentPageSize" class="pagination-row">
              <n-pagination
                v-model:page="incidentPage"
                :page-count="Math.ceil(incidentTotal / incidentPageSize)"
                @update:page="loadIncidents"
              />
            </div>
          </n-tab-pane>

          <!-- Overview -->
          <n-tab-pane name="overview" :tab="t('common.overview')">
            <n-descriptions :columns="2" label-placement="left" bordered size="small">
              <n-descriptions-item :label="t('channel.status')">
                <n-tag :type="channel.status === 'active' ? 'success' : 'warning'" size="small" round>
                  {{ channel.status === 'active' ? t('common.active') : t('common.disabled') }}
                </n-tag>
              </n-descriptions-item>
              <n-descriptions-item :label="t('channel.accessLevel')">
                {{ channel.access_level === 'public' ? t('channel.accessPublic') : t('channel.accessPrivate') }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('channel.autoClose')">
                {{ channel.auto_close_enabled ? t('channel.autoCloseMinutesUnit', { n: channel.auto_close_minutes }) : t('common.off') }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('channel.followAlertClose')">
                {{ channel.follow_alert_close ? t('common.yes') : t('common.no') }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('channel.activeIncidents')">
                <span class="tnum">{{ channel.active_incident_count }}</span>
              </n-descriptions-item>
              <n-descriptions-item v-if="channel.team" :label="t('channel.team')">
                {{ channel.team.name }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('common.createdAt')">
                <span class="tnum">{{ formatTime(channel.created_at) }}</span>
              </n-descriptions-item>
              <n-descriptions-item :label="t('common.updatedAt')">
                <span class="tnum">{{ formatTime(channel.updated_at) }}</span>
              </n-descriptions-item>
            </n-descriptions>
          </n-tab-pane>

          <!-- Noise -->
          <n-tab-pane name="noise" :tab="t('channel.noiseTab')">
            <NoiseConfig :channel-id="channelId" />
          </n-tab-pane>

          <!-- Dispatch -->
          <n-tab-pane name="dispatch" :tab="t('channel.dispatchTab')">
            <DispatchConfig :channel-id="channelId" />
          </n-tab-pane>

          <!-- Settings -->
          <n-tab-pane name="settings" :tab="t('common.settings')">
            <div class="settings-grid">
              <n-form label-placement="top" size="small" class="settings-form">
                <n-form-item :label="t('channel.name')">
                  <n-input v-model:value="editForm.name" />
                </n-form-item>
                <n-form-item :label="t('channel.description')">
                  <n-input v-model:value="editForm.description" type="textarea" :rows="2" />
                </n-form-item>
                <n-form-item :label="t('channel.status')">
                  <n-radio-group v-model:value="editForm.status">
                    <n-radio value="active">{{ t('common.active') }}</n-radio>
                    <n-radio value="disabled">{{ t('common.disabled') }}</n-radio>
                  </n-radio-group>
                </n-form-item>
                <n-form-item :label="t('channel.accessLevel')">
                  <n-radio-group v-model:value="editForm.access_level">
                    <n-radio value="public">{{ t('channel.accessPublic') }}</n-radio>
                    <n-radio value="private">{{ t('channel.accessPrivate') }}</n-radio>
                  </n-radio-group>
                </n-form-item>
                <n-form-item :label="t('channel.autoClose')">
                  <n-switch v-model:value="editForm.auto_close_enabled" />
                </n-form-item>
                <n-form-item v-if="editForm.auto_close_enabled" :label="t('channel.autoCloseMinutes')">
                  <n-input-number v-model:value="editForm.auto_close_minutes" :min="1" :max="10080" />
                </n-form-item>
                <n-form-item>
                  <n-checkbox v-model:checked="editForm.follow_alert_close">
                    {{ t('channel.followAlertClose') }}
                  </n-checkbox>
                </n-form-item>
                <div class="form-actions">
                  <n-button type="primary" :loading="saving" @click="saveChannel">
                    {{ t('common.save') }}
                  </n-button>
                </div>
              </n-form>

              <div class="danger-zone">
                <div class="danger-eyebrow sre-label-eyebrow">{{ t('common.dangerZone') }}</div>
                <div class="danger-body">
                  <div class="danger-text">
                    <div class="danger-title">{{ t('common.delete') }} {{ channel.name }}</div>
                    <div class="danger-desc">{{ t('channel.deleteDesc') }}</div>
                  </div>
                  <n-button type="error" ghost size="small" @click="confirmDelete">
                    <template #icon><n-icon :component="TrashOutline" /></template>
                    {{ t('common.delete') }}
                  </n-button>
                </div>
              </div>
            </div>
          </n-tab-pane>
        </n-tabs>
      </section>
    </n-spin>

    <!-- Quick edit modal -->
    <n-modal
      v-model:show="showEditModal"
      :title="t('common.edit') + ' — ' + (channel?.name ?? '')"
      preset="card"
      class="ch-modal-edit"
      :bordered="false"
    >
      <n-form label-placement="top" size="small">
        <n-form-item :label="t('channel.name')" required>
          <n-input v-model:value="editForm.name" />
        </n-form-item>
        <n-form-item :label="t('channel.description')">
          <n-input v-model:value="editForm.description" type="textarea" :rows="2" />
        </n-form-item>
        <n-form-item :label="t('channel.status')">
          <n-select
            v-model:value="editForm.status"
            :options="[
              { label: t('common.active'), value: 'active' },
              { label: t('common.disabled'), value: 'disabled' },
            ]"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="saveChannel">{{ t('common.save') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.channel-detail {
  max-width: 1400px;
  font-family: var(--sre-font-sans);
}

/* ───────── Header ───────── */
.cd-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 4px 4px 20px;
  border-bottom: var(--sre-hairline);
  margin-bottom: 24px;
}
.cd-header-left {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}
.cd-title-block { min-width: 0; }
.cd-title {
  font-size: 24px;
  font-weight: 700;
  line-height: 1.2;
  letter-spacing: -0.01em;
  margin: 0 0 4px;
  color: var(--sre-text-primary);
}
.cd-subtitle {
  font-size: 13px;
  color: var(--sre-text-secondary);
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.cd-header-right {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

/* ───────── KPI ───────── */
.kpi-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 24px;
}
.kpi-card {
  position: relative;
  padding: 20px;
  background: var(--sre-bg-elev, var(--sre-bg-page));
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md, 10px);
  overflow: hidden;
}
.kpi-value {
  font-size: 28px;
  font-weight: 700;
  line-height: 1.1;
  letter-spacing: -0.02em;
  color: var(--sre-text-primary);
  margin-bottom: 8px;
}
.kpi-stripe {
  position: absolute;
  left: 0; right: 0; bottom: 0;
  height: 3px;
  background: var(--sre-text-tertiary);
}
.kpi-stripe[data-tone="critical"] { background: var(--sre-danger); }
.kpi-stripe[data-tone="warning"]  { background: var(--sre-warning); }
.kpi-stripe[data-tone="success"]  { background: var(--sre-success); }
.kpi-stripe[data-tone="info"]     { background: var(--sre-primary); }

/* ───────── Tabs ───────── */
.cd-tabs {
  background: transparent;
}
:deep(.n-tabs .n-tabs-tab) {
  font-family: var(--sre-font-sans);
  font-weight: 500;
}
:deep(.n-tabs .n-tabs-tab--active) {
  color: var(--sre-primary);
}
:deep(.n-tabs-bar) {
  background-color: var(--sre-primary);
}

.tab-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

/* ───────── Incident rows ───────── */
.incident-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.incident-row {
  display: grid;
  grid-template-columns: 14px 60px 1fr 100px 110px 130px 60px;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  cursor: pointer;
  border-left: 4px solid transparent;
  transition: border-color .15s ease;
}
.incident-row:hover {
  border-left-color: var(--sre-primary);
}
.i-id {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  font-variant-numeric: tabular-nums;
}
.i-title {
  font-size: 13.5px;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.i-status {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--sre-text-secondary);
  font-weight: 500;
}
.i-status[data-status="triggered"] { color: var(--sre-danger); }
.i-status[data-status="processing"] { color: var(--sre-warning); }
.i-status[data-status="closed"] { color: var(--sre-text-tertiary); }
.i-meta {
  font-size: 12px;
  color: var(--sre-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.i-time {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  text-align: right;
}
.empty-wrap {
  padding: 48px 0;
}
.pagination-row {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

/* ───────── Settings ───────── */
.settings-grid {
  display: flex;
  flex-direction: column;
  gap: 32px;
  max-width: 600px;
}
.settings-form { width: 100%; }
.form-actions {
  padding-top: 8px;
  border-top: var(--sre-hairline);
  margin-top: 8px;
}
.danger-zone {
  border: 1px solid var(--sre-danger);
  border-radius: var(--sre-radius-md);
  padding: 16px 20px;
  background: color-mix(in srgb, var(--sre-danger) 4%, transparent);
}
.danger-eyebrow {
  color: var(--sre-danger);
  margin-bottom: 12px;
}
.danger-body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.danger-title {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--sre-text-primary);
  margin-bottom: 2px;
}
.danger-desc {
  font-size: 12px;
  color: var(--sre-text-secondary);
}

@media (max-width: 768px) {
  .kpi-row { grid-template-columns: repeat(2, 1fr); }
  .incident-row {
    grid-template-columns: 14px 1fr 60px;
    grid-template-areas:
      "dot title time"
      "dot meta meta";
    row-gap: 4px;
  }
  .i-id, .i-status { display: none; }
}
</style>

<style>
@import '@/styles/channels.css';
</style>
