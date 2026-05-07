<script setup lang="ts">
import { ref, onMounted, computed, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage, NTag, NButton, NSpace } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { channelV2Api, incidentApi } from '@/api'
import NoiseConfig from './NoiseConfig.vue'
import type { Channel, Incident, ChannelStatus, ChannelAccessLevel } from '@/types'
import { formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import { ArrowBackOutline, SettingsOutline, RefreshOutline } from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const route = useRoute()
const router = useRouter()

const channelId = computed(() => Number(route.params.id))
const channel = ref<Channel | null>(null)
const incidents = ref<Incident[]>([])
const incidentTotal = ref(0)
const incidentPage = ref(1)
const incidentPageSize = ref(20)
const incidentStatus = ref('')
const incidentLoading = ref(false)
const channelLoading = ref(false)
const activeTab = ref('incidents')

// Edit modal
const showEditModal = ref(false)
const saving = ref(false)
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

const statusTagType: Record<string, 'error' | 'warning' | 'success' | 'default'> = {
  triggered: 'error', processing: 'warning', closed: 'success',
}
const statusLabel: Record<string, string> = {
  triggered: 'incident.statusTriggered',
  processing: 'incident.statusProcessing',
  closed: 'incident.statusClosed',
}
const severityTagType: Record<string, 'error' | 'warning' | 'info' | 'default'> = {
  critical: 'error', warning: 'warning', info: 'info',
}
const severityLabel: Record<string, string> = {
  critical: 'incident.severityCritical',
  warning: 'incident.severityWarning',
  info: 'incident.severityInfo',
}

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
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
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
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
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
  } catch (e: any) {
    message.error(e?.message ?? t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

// Stats derived from incidents
const stats = computed(() => {
  const triggeredCount = incidents.value.filter(i => i.status === 'triggered').length
  const processingCount = incidents.value.filter(i => i.status === 'processing').length
  const closedCount = incidents.value.filter(i => i.status === 'closed').length
  const criticalCount = incidents.value.filter(i => i.severity === 'critical').length
  return { triggeredCount, processingCount, closedCount, criticalCount }
})

const incidentColumns = computed(() => [
  {
    title: 'ID',
    key: 'id',
    width: 70,
    render: (row: Incident) => h('span', { style: 'font-size:12px;color:var(--sre-text-secondary)' }, `#${row.id}`),
  },
  {
    title: t('incident.severity'),
    key: 'severity',
    width: 90,
    render: (row: Incident) =>
      h(NTag, { type: severityTagType[row.severity] ?? 'default', size: 'small' },
        { default: () => t(severityLabel[row.severity] ?? row.severity) }),
  },
  {
    title: t('incident.name'),
    key: 'title',
    render: (row: Incident) =>
      h('a', {
        style: 'cursor:pointer;color:var(--sre-primary)',
        onClick: () => router.push(`/incidents/${row.id}`),
      }, row.title),
  },
  {
    title: t('incident.status'),
    key: 'status',
    width: 110,
    render: (row: Incident) =>
      h(NTag, { type: statusTagType[row.status] ?? 'default', size: 'small' },
        { default: () => t(statusLabel[row.status] ?? row.status) }),
  },
  {
    title: t('incident.assignee'),
    key: 'assigned_user',
    render: (row: Incident) =>
      h('span', {}, row.assigned_user?.display_name ?? row.assigned_user?.username ?? '—'),
  },
  {
    title: t('incident.alertCount'),
    key: 'alert_count',
    width: 80,
    render: (row: Incident) => h('span', {}, String(row.alert_count)),
  },
  {
    title: t('incident.triggeredAt'),
    key: 'triggered_at',
    render: (row: Incident) => h('span', { style: 'font-size:12px' }, formatTime(row.triggered_at)),
  },
])

onMounted(async () => {
  await loadChannel()
  await loadIncidents()
})
</script>

<template>
  <div class="channel-detail">
    <PageHeader
      :title="channel?.name ?? t('channel.title')"
      :subtitle="channel?.description ?? ''"
    >
      <template #actions>
        <n-button quaternary @click="router.back()">
          <template #icon><n-icon :component="ArrowBackOutline" /></template>
          {{ t('common.back') }}
        </n-button>
        <n-button @click="showEditModal = true">
          <template #icon><n-icon :component="SettingsOutline" /></template>
          {{ t('common.edit') }}
        </n-button>
      </template>
    </PageHeader>

    <n-spin :show="channelLoading">
      <div v-if="channel">

        <!-- Tabs -->
        <n-card :bordered="false" class="main-card">
          <n-tabs v-model:value="activeTab" type="line" animated>

            <!-- Tab 1: Incident list -->
            <n-tab-pane name="incidents" :tab="t('incident.title')">
              <div class="tab-toolbar">
                <n-select
                  v-model:value="incidentStatus"
                  :options="[
                    { label: t('incident.statusTriggered'), value: 'triggered' },
                    { label: t('incident.statusProcessing'), value: 'processing' },
                    { label: t('incident.statusClosed'),    value: 'closed' },
                  ]"
                  :placeholder="t('common.status')"
                  clearable
                  style="width:140px"
                  @update:value="loadIncidents"
                />
                <n-button circle quaternary size="small" @click="loadIncidents">
                  <template #icon><n-icon :component="RefreshOutline" /></template>
                </n-button>
              </div>

              <n-data-table
                :loading="incidentLoading"
                :columns="incidentColumns"
                :data="incidents"
                :row-key="(row: Incident) => row.id"
                size="small"
                :row-props="(row: Incident) => ({
                  style: 'cursor:pointer',
                  onClick: () => router.push(`/incidents/${row.id}`)
                })"
              />
              <div v-if="incidentTotal > incidentPageSize" class="pagination-row">
                <n-pagination
                  v-model:page="incidentPage"
                  :page-count="Math.ceil(incidentTotal / incidentPageSize)"
                  @update:page="loadIncidents"
                />
              </div>
            </n-tab-pane>

            <!-- Tab 2: Stats overview -->
            <n-tab-pane name="stats" :tab="'Overview'">
              <div class="stats-grid">
                <div class="stat-card">
                  <div class="stat-label">{{ t('incident.statusTriggered') }}</div>
                  <div class="stat-value triggered">{{ stats.triggeredCount }}</div>
                </div>
                <div class="stat-card">
                  <div class="stat-label">{{ t('incident.statusProcessing') }}</div>
                  <div class="stat-value processing">{{ stats.processingCount }}</div>
                </div>
                <div class="stat-card">
                  <div class="stat-label">{{ t('incident.statusClosed') }}</div>
                  <div class="stat-value closed">{{ stats.closedCount }}</div>
                </div>
                <div class="stat-card">
                  <div class="stat-label">{{ t('incident.severityCritical') }}</div>
                  <div class="stat-value critical">{{ stats.criticalCount }}</div>
                </div>
              </div>

              <!-- Channel meta info -->
              <n-descriptions :columns="2" label-placement="left" bordered size="small" style="margin-top:24px">
                <n-descriptions-item :label="t('channel.status')">
                  <n-tag :type="channel.status === 'active' ? 'success' : 'warning'" size="small">
                    {{ channel.status === 'active' ? t('common.active') : t('common.disabled') }}
                  </n-tag>
                </n-descriptions-item>
                <n-descriptions-item :label="t('channel.accessLevel')">
                  {{ channel.access_level === 'public' ? t('channel.accessPublic') : t('channel.accessPrivate') }}
                </n-descriptions-item>
                <n-descriptions-item :label="t('channel.autoClose')">
                  {{ channel.auto_close_enabled ? `${channel.auto_close_minutes} min` : t('common.off') }}
                </n-descriptions-item>
                <n-descriptions-item :label="t('channel.followAlertClose')">
                  {{ channel.follow_alert_close ? t('common.yes') : t('common.no') }}
                </n-descriptions-item>
                <n-descriptions-item :label="t('channel.activeIncidents')">
                  {{ channel.active_incident_count }}
                </n-descriptions-item>
                <n-descriptions-item v-if="channel.team" :label="t('channel.team')">
                  {{ channel.team.name }}
                </n-descriptions-item>
              </n-descriptions>
            </n-tab-pane>

            <!-- Tab 3: Noise reduction config -->
            <n-tab-pane name="noise" :tab="t('channel.noiseTab')">
              <NoiseConfig :channel-id="channelId" />
            </n-tab-pane>

            <!-- Tab 4: Basic config (auto-close, access) -->
            <n-tab-pane name="config" :tab="t('common.actions')">
              <n-form label-placement="top" size="small" style="max-width:520px">
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
                <n-button type="primary" :loading="saving" @click="saveChannel">
                  {{ t('common.save') }}
                </n-button>
              </n-form>
            </n-tab-pane>

          </n-tabs>
        </n-card>
      </div>
    </n-spin>

    <!-- Quick-edit modal -->
    <n-modal
      v-model:show="showEditModal"
      :title="t('common.edit') + ' — ' + (channel?.name ?? '')"
      preset="card"
      style="width:440px"
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
.channel-detail { max-width: 1400px; }
.main-card { border-radius: 12px; }

.tab-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.pagination-row {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 8px;
}

.stat-card {
  background: var(--sre-bg-page);
  border: 1px solid var(--sre-border);
  border-radius: 10px;
  padding: 20px 16px;
  text-align: center;
}

.stat-label {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin-bottom: 8px;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  line-height: 1;
}

.stat-value.triggered { color: #e03131; }
.stat-value.processing { color: #f08c00; }
.stat-value.closed     { color: #2f9e44; }
.stat-value.critical   { color: #e03131; }
</style>
