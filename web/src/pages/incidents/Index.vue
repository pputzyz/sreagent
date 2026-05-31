<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage, useDialog } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { incidentApi } from '@/api'
import type { Incident } from '@/types'
import { usePaginatedList } from '@/composables'
import { useAuthStore } from '@/stores/auth'
import { formatTime, getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import {
  RefreshOutline,
  AddOutline,
  SearchOutline,
  ChevronForwardOutline,
  AlertCircleOutline,
  TimeOutline,
  NotificationsOutline,
  ShieldCheckmarkOutline,
  EllipsisHorizontal,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const viewMode = ref<'all' | 'mine'>('all')
const statusFilter = ref<string>('')
const severityFilter = ref<string>('')
const searchQuery = ref('')
let searchTimer: ReturnType<typeof setTimeout> | null = null
function onSearchUpdate() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => fetchList(), 300)
}

// FE1-13: Bulk actions state
const selectedIncidents = ref<Set<number>>(new Set())
const bulkLoading = ref(false)

function toggleSelect(id: number) {
  const s = new Set(selectedIncidents.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  selectedIncidents.value = s
}

function toggleSelectAll() {
  if (selectedIncidents.value.size === incidents.value.length) {
    selectedIncidents.value = new Set()
  } else {
    selectedIncidents.value = new Set(incidents.value.map(i => i.id))
  }
}

function clearSelection() {
  selectedIncidents.value = new Set()
}

const allSelected = computed(() =>
  incidents.value.length > 0 && selectedIncidents.value.size === incidents.value.length
)

function confirmBulkAction(action: 'close' | 'acknowledge') {
  const count = selectedIncidents.value.size
  if (count === 0) return
  const label = action === 'close' ? t('incident.bulkClose') : t('incident.bulkAcknowledge')
  dialog.warning({
    title: label,
    content: t('incident.bulkConfirmMsg', { count }),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => executeBulkAction(action),
  })
}

async function executeBulkAction(action: 'close' | 'acknowledge') {
  const ids = Array.from(selectedIncidents.value)
  bulkLoading.value = true
  try {
    if (action === 'close') {
      await incidentApi.bulkClose(ids)
    } else {
      await incidentApi.bulkAcknowledge(ids)
    }
    message.success(t('common.success'))
    clearSelection()
    await fetchList()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.failed'))
  } finally {
    bulkLoading.value = false
  }
}

// FE1-5: Apply query params from overview drill-down
onMounted(() => {
  if (route.query.status) statusFilter.value = String(route.query.status)
  if (route.query.severity) severityFilter.value = String(route.query.severity)
  if (route.query.search) searchQuery.value = String(route.query.search)
  fetchList()
})

const {
  loading,
  items: incidents,
  total,
  page,
  pageSize,
  fetchList,
  refresh,
} = usePaginatedList<Incident>({
  apiFn: incidentApi.list,
  extraParams: () => {
    const params: Record<string, unknown> = {}
    if (statusFilter.value) params.status = statusFilter.value
    if (severityFilter.value) params.severity = severityFilter.value
    if (searchQuery.value) params.query = searchQuery.value
    if (viewMode.value === 'mine' && authStore.user?.id) params.assigned_to = authStore.user.id
    return params
  },
  onError: (err: unknown) => {
    message.error((err as Error)?.message ?? t('common.loadFailed'))
  },
})

// Create modal
const showCreateModal = ref(false)
const saving = ref(false)
const createForm = ref({
  title: '',
  description: '',
  severity: 'warning',
  channel_id: 0,
})

function resetCreateForm() {
  createForm.value = { title: '', description: '', severity: 'warning', channel_id: 0 }
}

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

async function acknowledgeIncident(id: number, e?: Event) {
  e?.stopPropagation()
  try {
    await incidentApi.acknowledge(id)
    message.success(t('common.success'))
    await fetchList()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.failed'))
  }
}

async function closeIncident(id: number, e?: Event) {
  e?.stopPropagation()
  try {
    await incidentApi.close(id)
    message.success(t('common.success'))
    await fetchList()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.failed'))
  }
}

async function createIncident() {
  if (!createForm.value.title.trim()) return
  saving.value = true
  try {
    const res = await incidentApi.create(createForm.value)
    message.success(t('common.createSuccess'))
    showCreateModal.value = false
    router.push(`/oncall/incidents/${res.data.data?.id}`)
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.failed'))
  } finally {
    saving.value = false
  }
}

function actionOptions(row: Incident) {
  const opts: { label: string; key: string }[] = []
  if (row.status !== 'closed' && row.status !== 'processing') {
    opts.push({ label: t('incident.acknowledge'), key: 'acknowledge' })
  }
  if (row.status !== 'closed') {
    opts.push({ label: t('incident.close'), key: 'close' })
  }
  return opts
}

function handleAction(key: string, row: Incident) {
  if (key === 'acknowledge') acknowledgeIncident(row.id)
  else if (key === 'close') closeIncident(row.id)
}

function gotoDetail(id: number) {
  router.push(`/oncall/incidents/${id}`)
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
const hasFilters = computed(() =>
  statusFilter.value !== '' || severityFilter.value !== '' || searchQuery.value.trim() !== '' || viewMode.value === 'mine'
)

</script>

<template>
  <div class="incidents-page">
    <PageHeader :title="t('incident.title')" :subtitle="t('incident.subtitle')">
      <template #actions>
        <n-button circle quaternary @click="fetchList" :aria-label="t('common.refresh')">
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
      <n-radio-group v-model:value="viewMode" size="medium" @update:value="fetchList">
        <n-radio-button value="all">{{ t('incident.allIncidents') }}</n-radio-button>
        <n-radio-button value="mine">{{ t('incident.myIncidents') }}</n-radio-button>
      </n-radio-group>
    </div>

    <!-- Compact filter bar -->
    <div class="filter-bar">
      <div class="filter-group">
        <span class="filter-label">{{ t('common.status') }}</span>
        <n-radio-group v-model:value="statusFilter" size="small" @update:value="fetchList">
          <n-radio-button value="">{{ t('common.all') }}</n-radio-button>
          <n-radio-button value="triggered">{{ t('incident.statusTriggered') }}</n-radio-button>
          <n-radio-button value="processing">{{ t('incident.statusProcessing') }}</n-radio-button>
          <n-radio-button value="closed">{{ t('incident.statusClosed') }}</n-radio-button>
        </n-radio-group>
      </div>

      <div class="filter-group">
        <span class="filter-label">{{ t('incident.severity') }}</span>
        <n-radio-group v-model:value="severityFilter" size="small" @update:value="fetchList">
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

      <n-checkbox
        v-if="incidents.length > 0"
        :checked="allSelected"
        :indeterminate="selectedIncidents.size > 0 && !allSelected"
        @update:checked="toggleSelectAll"
        class="select-all-checkbox"
      >
        {{ t('common.selectAll') || 'Select all' }}
      </n-checkbox>

      <n-input
        v-model:value="searchQuery"
        :placeholder="t('common.search')"
        clearable
        size="small"
        class="search-box"
        @update:value="onSearchUpdate"
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
        :description="hasFilters ? t('incident.emptyFiltered') : t('incident.emptyDesc')"
      />

      <div v-else class="incident-list">
        <div
          v-for="row in incidents"
          :key="row.id"
          class="incident-row"
          :class="{ 'is-closed': row.status === 'closed', 'is-selected': selectedIncidents.has(row.id) }"
        >
          <n-checkbox
            :checked="selectedIncidents.has(row.id)"
            class="row-checkbox"
            @update:checked="toggleSelect(row.id)"
            @click.stop
          />
          <span class="status-bar" :data-severity="row.severity" @click="gotoDetail(row.id)" />

          <div class="row-body" @click="gotoDetail(row.id)">
            <div class="row-line-1">
              <span class="dot dot-lg" :data-severity="row.severity" />
              <span class="incident-title">{{ row.title }}</span>
              <span class="status-pill" :data-status="row.status">
                {{ t(statusLabel[row.status] ?? row.status) }}
              </span>
              <span class="assignee" v-if="row.assigned_user">
                <span class="avatar">{{ userInitial(row.assigned_user) }}</span>
              </span>
            </div>

            <div class="row-line-2">
              <span v-if="row.channel?.name" class="meta-item">
                <n-icon :component="NotificationsOutline" size="12" />
                {{ row.channel.name }}
              </span>
              <span class="meta-item">
                <n-icon :component="TimeOutline" size="12" />
                {{ durationText(row.triggered_at, row.closed_at) }}
              </span>
              <span class="meta-item">
                <n-icon :component="AlertCircleOutline" size="12" />
                {{ row.alert_count ?? 0 }}
              </span>
              <span class="meta-item">
                {{ formatTime(row.triggered_at) }}
              </span>
            </div>
          </div>

          <n-dropdown
            v-if="actionOptions(row).length > 0"
            :options="actionOptions(row)"
            trigger="click"
            placement="bottom-end"
            @select="(key: string) => handleAction(key, row)"
          >
            <n-icon
              :component="EllipsisHorizontal"
              class="action-trigger"
              size="18"
              @click.stop
            />
          </n-dropdown>

          <n-icon :component="ChevronForwardOutline" class="chevron" size="18" />
        </div>
      </div>

      <div v-if="total > pageSize" class="pagination">
        <n-pagination
          v-model:page="page"
          :page-size="pageSize"
          :item-count="total"
          :page-slot="7"
          @update:page="fetchList"
        />
      </div>
    </n-spin>

    <!-- FE1-13: Floating bulk action bar -->
    <Transition name="bulk-bar">
      <div v-if="selectedIncidents.size > 0" class="bulk-action-bar">
        <span class="bulk-count">
          {{ t('incident.selectedCount', { count: selectedIncidents.size }) || `${selectedIncidents.size} selected` }}
        </span>
        <n-button
          size="small"
          :loading="bulkLoading"
          @click="confirmBulkAction('acknowledge')"
        >
          {{ t('incident.bulkAcknowledge') || 'Bulk Acknowledge' }}
        </n-button>
        <n-button
          size="small"
          type="warning"
          :loading="bulkLoading"
          @click="confirmBulkAction('close')"
        >
          {{ t('incident.bulkClose') || 'Bulk Close' }}
        </n-button>
        <n-button size="small" quaternary @click="clearSelection">
          {{ t('common.cancel') }}
        </n-button>
      </div>
    </Transition>

    <!-- Create Modal -->
    <n-modal
      v-model:show="showCreateModal"
      :title="t('incident.create')"
      preset="card"
      :bordered="false"
      class="create-modal"
      @after-leave="resetCreateForm"
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
  border-radius: 8px;
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

.status-pill[data-status="triggered"]  { color: var(--sre-critical); background: var(--sre-critical-soft); }
.status-pill[data-status="processing"] { color: var(--sre-warning); background: var(--sre-warning-soft); }
.status-pill[data-status="closed"]     { color: var(--sre-success); background: var(--sre-success-soft); }

/* Incident list */
.incident-list {
  display: flex;
  flex-direction: column;
  gap: var(--sre-row-gap);
}
.incident-row {
  position: relative;
  display: flex;
  align-items: center;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: 8px;
  padding: 10px 14px 10px 20px;
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
}
.incident-row:hover .action-trigger {
  opacity: 1;
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
  gap: 4px;
  min-width: 0;
}

.row-line-1 {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  min-width: 0;
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
  gap: 14px;
  font-size: var(--sre-fs-xs, 11px);
  color: var(--sre-text-tertiary);
}
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  white-space: nowrap;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}
.assignee {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
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

.action-trigger {
  flex-shrink: 0;
  margin-left: 4px;
  padding: 4px;
  border-radius: 4px;
  color: var(--sre-text-tertiary);
  opacity: 0;
  cursor: pointer;
  transition: opacity 0.15s ease, background-color 0.15s ease;
}
.action-trigger:hover {
  background: var(--sre-bg-active, rgba(0,0,0,0.06));
  color: var(--sre-text-secondary);
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

/* FE1-13: Bulk actions */
.row-checkbox {
  flex-shrink: 0;
  margin-right: 8px;
  z-index: 1;
}
.incident-row.is-selected {
  border-color: var(--sre-primary);
  background: var(--sre-primary-soft, rgba(var(--sre-primary-rgb, 13,148,136), 0.06));
}
.select-all-checkbox {
  flex-shrink: 0;
}
.bulk-action-bar {
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 20px;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.18);
  z-index: 100;
}
.bulk-count {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
  white-space: nowrap;
}
.bulk-bar-enter-active,
.bulk-bar-leave-active {
  transition: all 0.2s ease;
}
.bulk-bar-enter-from,
.bulk-bar-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(12px);
}
</style>
