<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { incidentApi } from '@/api'
import type { Incident } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import {
  RefreshOutline,
  AddOutline,
  SearchOutline,
  ChevronForwardOutline,
  AlertCircleOutline,
  PersonOutline,
  TimeOutline,
  NotificationsOutline,
  ShieldCheckmarkOutline,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()
const authStore = useAuthStore()

const loading = ref(false)
const incidents = ref<Incident[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const viewMode = ref<'all' | 'mine'>('all')
const statusFilter = ref<string>('')
const severityFilter = ref<string>('')
const searchQuery = ref('')

// Create modal
const showCreateModal = ref(false)
const saving = ref(false)
const createForm = ref({
  title: '',
  description: '',
  severity: 'warning',
  channel_id: 0,
})

const SEVERITY_COLORS: Record<string, string> = {
  critical: 'var(--sre-critical)',
  warning: 'var(--sre-warning)',
  info: 'var(--sre-info)',
}

const STATUS_COLORS: Record<string, string> = {
  triggered: 'var(--sre-critical)',
  processing: 'var(--sre-warning)',
  closed: 'var(--sre-success)',
}

const STATUS_BG: Record<string, string> = {
  triggered: 'var(--sre-critical-soft)',
  processing: 'var(--sre-warning-soft)',
  closed: 'var(--sre-success-soft)',
}

const severityLabel: Record<string, string> = {
  critical: 'incident.severityCritical',
  warning: 'incident.severityWarning',
  info: 'incident.severityInfo',
}

const statusLabel: Record<string, string> = {
  triggered: 'incident.statusTriggered',
  processing: 'incident.statusProcessing',
  closed: 'incident.statusClosed',
}

async function loadIncidents() {
  loading.value = true
  try {
    const params: any = {
      page: page.value,
      page_size: pageSize.value,
      status: statusFilter.value,
      severity: severityFilter.value,
      query: searchQuery.value,
    }
    if (viewMode.value === 'mine' && authStore.user?.id) {
      params.assigned_to = authStore.user.id
    }
    const res = await incidentApi.list(params)
    incidents.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function acknowledgeIncident(id: number, e?: Event) {
  e?.stopPropagation()
  try {
    await incidentApi.acknowledge(id)
    message.success(t('common.success'))
    await loadIncidents()
  } catch (e: any) {
    message.error(e?.message ?? t('common.failed'))
  }
}

async function closeIncident(id: number, e?: Event) {
  e?.stopPropagation()
  try {
    await incidentApi.close(id)
    message.success(t('common.success'))
    await loadIncidents()
  } catch (e: any) {
    message.error(e?.message ?? t('common.failed'))
  }
}

async function createIncident() {
  if (!createForm.value.title.trim()) return
  saving.value = true
  try {
    const res = await incidentApi.create(createForm.value)
    message.success(t('common.createSuccess'))
    showCreateModal.value = false
    router.push(`/incidents/${res.data.data?.id}`)
  } catch (e: any) {
    message.error(e?.message ?? t('common.failed'))
  } finally {
    saving.value = false
  }
}

function gotoDetail(id: number) {
  router.push(`/incidents/${id}`)
}

function durationText(triggeredAt: string | undefined, closedAt?: string | null): string {
  if (!triggeredAt) return '—'
  const start = new Date(triggeredAt).getTime()
  const end = closedAt ? new Date(closedAt).getTime() : Date.now()
  let diff = Math.max(0, Math.floor((end - start) / 1000))
  const d = Math.floor(diff / 86400); diff -= d * 86400
  const h = Math.floor(diff / 3600); diff -= h * 3600
  const m = Math.floor(diff / 60)
  if (d > 0) return t('incident.durationDHM', { d, h })
  if (h > 0) return t('incident.durationHM', { h, m })
  return t('incident.durationM', { m })
}

function userInitial(u?: { display_name?: string; username?: string } | null): string {
  const name = u?.display_name || u?.username
  return name ? name.charAt(0).toUpperCase() : '?'
}

const isEmpty = computed(() => !loading.value && incidents.value.length === 0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

onMounted(loadIncidents)
</script>

<template>
  <div class="incidents-page">
    <PageHeader :title="t('incident.title')" :subtitle="t('incident.subtitle')">
      <template #actions>
        <n-button circle quaternary @click="loadIncidents">
          <template #icon><n-icon :component="RefreshOutline" /></template>
        </n-button>
        <n-button type="primary" @click="showCreateModal = true">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('incident.create') }}
        </n-button>
      </template>
    </PageHeader>

    <!-- Primary tabs: All / Mine -->
    <div class="primary-tabs">
      <n-radio-group v-model:value="viewMode" size="medium" @update:value="loadIncidents">
        <n-radio-button value="all">{{ t('incident.allIncidents') }}</n-radio-button>
        <n-radio-button value="mine">{{ t('incident.myIncidents') }}</n-radio-button>
      </n-radio-group>
    </div>

    <!-- Compact filter bar -->
    <div class="filter-bar">
      <div class="filter-group">
        <span class="filter-label">{{ t('common.status') }}</span>
        <n-radio-group v-model:value="statusFilter" size="small" @update:value="loadIncidents">
          <n-radio-button value="">{{ t('common.all') }}</n-radio-button>
          <n-radio-button value="triggered">{{ t('incident.statusTriggered') }}</n-radio-button>
          <n-radio-button value="processing">{{ t('incident.statusProcessing') }}</n-radio-button>
          <n-radio-button value="closed">{{ t('incident.statusClosed') }}</n-radio-button>
        </n-radio-group>
      </div>

      <div class="filter-group">
        <span class="filter-label">{{ t('incident.severity') }}</span>
        <n-radio-group v-model:value="severityFilter" size="small" @update:value="loadIncidents">
          <n-radio-button value="">{{ t('common.all') }}</n-radio-button>
          <n-radio-button value="critical">
            <span class="dot" data-severity="critical" />
            {{ t('incident.severityCritical') }}
          </n-radio-button>
          <n-radio-button value="warning">
            <span class="dot" data-severity="warning" />
            {{ t('incident.severityWarning') }}
          </n-radio-button>
          <n-radio-button value="info">
            <span class="dot" data-severity="info" />
            {{ t('incident.severityInfo') }}
          </n-radio-button>
        </n-radio-group>
      </div>

      <div class="filter-spacer" />

      <n-input
        v-model:value="searchQuery"
        :placeholder="t('common.search')"
        clearable
        size="small"
        class="search-box"
        @update:value="loadIncidents"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
    </div>

    <!-- Incident list -->
    <n-spin :show="loading && incidents.length > 0">
      <LoadingSkeleton v-if="loading && incidents.length === 0" :rows="6" variant="row" />
      <EmptyState
        v-else-if="isEmpty"
        :icon="ShieldCheckmarkOutline"
        :title="t('incident.empty')"
        :description="t('incident.emptyDesc')"
      />

      <div v-else class="incident-list">
        <div
          v-for="row in incidents"
          :key="row.id"
          class="incident-row"
          :class="{ 'is-closed': row.status === 'closed' }"
          @click="gotoDetail(row.id)"
        >
          <span class="status-bar" :data-severity="row.severity" />

          <div class="row-body">
            <div class="row-line-1">
              <span class="dot dot-lg" :data-severity="row.severity" />
              <span class="severity-text" :data-severity="row.severity">
                {{ t(severityLabel[row.severity] ?? row.severity) }}
              </span>
              <span class="incident-id">#{{ row.id }}</span>
              <span class="incident-title">{{ row.title }}</span>
            </div>

            <div class="row-line-2">
              <span v-if="row.channel?.name" class="meta-item">
                <n-icon :component="NotificationsOutline" size="14" />
                {{ row.channel.name }}
              </span>
              <span class="meta-item">
                <n-icon :component="TimeOutline" size="14" />
                {{ t('incident.duration') }}: {{ durationText(row.triggered_at, row.closed_at) }}
              </span>
              <span class="meta-item">
                <n-icon :component="AlertCircleOutline" size="14" />
                {{ t('incident.alertCount') }}: {{ row.alert_count ?? 0 }}
              </span>
            </div>

            <div class="row-line-3">
              <span class="status-pill" :data-status="row.status">
                {{ t(statusLabel[row.status] ?? row.status) }}
              </span>

              <span class="assignee">
                <span class="avatar" v-if="row.assigned_user">
                  {{ userInitial(row.assigned_user) }}
                </span>
                <n-icon v-else :component="PersonOutline" size="14" />
                <span class="assignee-name">
                  {{ row.assigned_user?.display_name ?? row.assigned_user?.username ?? t('incident.unassigned') }}
                </span>
              </span>

              <span class="trigger-time">
                {{ t('incident.triggeredAt') }}: {{ formatTime(row.triggered_at) }}
              </span>

              <span class="row-actions" @click.stop>
                <n-button
                  v-if="row.status !== 'closed' && row.status !== 'processing'"
                  size="tiny" type="primary" tertiary
                  @click="acknowledgeIncident(row.id, $event)"
                >{{ t('incident.acknowledge') }}</n-button>
                <n-button
                  v-if="row.status !== 'closed'"
                  size="tiny" tertiary
                  @click="closeIncident(row.id, $event)"
                >{{ t('incident.close') }}</n-button>
              </span>
            </div>
          </div>

          <n-icon :component="ChevronForwardOutline" class="chevron" size="18" />
        </div>
      </div>

      <div v-if="total > pageSize" class="pagination">
        <n-pagination
          v-model:page="page"
          :page-count="totalPages"
          @update:page="loadIncidents"
        />
      </div>
    </n-spin>

    <!-- Create Modal -->
    <n-modal
      v-model:show="showCreateModal"
      :title="t('incident.create')"
      preset="card"
      :bordered="false"
      class="create-modal"
    >
      <n-form label-placement="top" size="small">
        <n-form-item :label="t('incident.name')" required>
          <n-input v-model:value="createForm.title" :placeholder="t('incident.namePlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('incident.severity')">
          <n-select
            v-model:value="createForm.severity"
            :options="[
              { label: t('incident.severityCritical'), value: 'critical' },
              { label: t('incident.severityWarning'), value: 'warning' },
              { label: t('incident.severityInfo'), value: 'info' },
            ]"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreateModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="createIncident">{{ t('common.create') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.incidents-page { max-width: 1400px; font-family: var(--sre-font-sans); }
.create-modal { width: 440px; }

.primary-tabs {
  margin-bottom: 12px;
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  padding: 10px 14px;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: 10px;
  margin-bottom: 16px;
}
.filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}
.filter-label {
  font-size: 12px;
  color: var(--sre-text-tertiary);
}
.filter-spacer { flex: 1; }
.search-box { width: 240px; min-width: 180px; }

.dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
  vertical-align: middle;
}
.dot-lg { width: 10px; height: 10px; margin-right: 8px; }

/* Severity / status data-attribute colors */
.dot[data-severity="critical"],
.dot-lg[data-severity="critical"],
.status-bar[data-severity="critical"] { background: var(--sre-critical); }
.dot[data-severity="warning"],
.dot-lg[data-severity="warning"],
.status-bar[data-severity="warning"] { background: var(--sre-warning); }
.dot[data-severity="info"],
.dot-lg[data-severity="info"],
.status-bar[data-severity="info"] { background: var(--sre-info); }

.severity-text[data-severity="critical"] { color: var(--sre-critical); }
.severity-text[data-severity="warning"]  { color: var(--sre-warning); }
.severity-text[data-severity="info"]     { color: var(--sre-info); }

.status-pill[data-status="triggered"]  { color: var(--sre-critical); background: var(--sre-critical-soft); }
.status-pill[data-status="processing"] { color: var(--sre-warning); background: var(--sre-warning-soft); }
.status-pill[data-status="closed"]     { color: var(--sre-success); background: var(--sre-success-soft); }

/* Incident list */
.incident-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.incident-row {
  position: relative;
  display: flex;
  align-items: center;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: 10px;
  padding: 14px 18px 14px 22px;
  cursor: pointer;
  transition: background-color 0.15s ease, border-color 0.15s ease, transform 0.15s ease;
  overflow: hidden;
}
.incident-row:hover {
  background: var(--sre-bg-hover);
  border-color: var(--sre-primary);
}
.incident-row:hover .chevron {
  opacity: 1;
  transform: translateX(2px);
}
.incident-row.is-closed { opacity: 0.72; }

.status-bar {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
}

.row-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.row-line-1 {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  flex-wrap: wrap;
}
.severity-text {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.2px;
}
.incident-id {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  font-family: var(--sre-font-mono);
}
.incident-title {
  font-weight: 600;
  color: var(--sre-text-primary);
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}

.row-line-2 {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--sre-text-secondary);
}
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.row-line-3 {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--sre-text-tertiary);
}
.status-pill {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
}
.assignee {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--sre-text-secondary);
}
.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--sre-primary);
  color: white;
  font-size: 11px;
  font-weight: 600;
}
.assignee-name { max-width: 160px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.trigger-time { color: var(--sre-text-tertiary); }
.row-actions {
  margin-left: auto;
  display: inline-flex;
  gap: 6px;
}

.chevron {
  flex-shrink: 0;
  margin-left: 12px;
  color: var(--sre-text-tertiary);
  opacity: 0;
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
