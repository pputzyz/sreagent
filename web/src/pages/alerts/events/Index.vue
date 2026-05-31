<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  useMessage,
  NButton,
  NIcon,
  NInput,
  NSelect,
  NDatePicker,
  NRadioGroup,
  NRadioButton,
  NDropdown,
  NPagination,
  NSpin,
  NEmpty,
  NText,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import {
  RefreshOutline,
  DownloadOutline,
  EllipsisHorizontalOutline,
  ShieldCheckmarkOutline,
  CloseOutline,
} from '@vicons/ionicons5'
import { alertEventApi, alertRuleApi } from '@/api'
import type { AlertEvent, AlertRule, AlertViewMode } from '@/types'
import { usePaginatedList, useFilterMemory, usePermissions } from '@/composables'
import { getErrorMessage } from '@/utils/format'
import { useAuthStore } from '@/stores/auth'
import { DynamicScroller, DynamicScrollerItem } from 'vue-virtual-scroller'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import PageHeader from '@/components/common/PageHeader.vue'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const { t } = useI18n()
const authStore = useAuthStore()
const { hasPerm } = usePermissions()

// ===== State =====
const firstLoad = ref(true)
const selected = ref<Set<number>>(new Set())
const crossPageActive = ref(false)
const crossPageLoading = ref(false)

// ===== Filters =====
type StatusTab = 'all' | 'firing' | 'acked' | 'resolved'
const statusTab = ref<StatusTab>('all')
const search = ref('')
const severityFilter = ref<string | null>(null)
const ruleFilter = ref<number | null>(null)
const tagFilter = ref('')
const customRange = ref<[number, number] | null>(null)
const timePreset = ref<string>('24h')

const viewMode = ref<AlertViewMode>('mine')

// Persist filter state to localStorage
const filterMemory = useFilterMemory('alert-events')
filterMemory.bindRefs({ statusTab, search, severityFilter, ruleFilter, tagFilter, timePreset, customRange, viewMode })

// Apply drill-down query params (override saved filters)
const queryPreset = route.query.time_preset as string | undefined
const queryStatus = route.query.status_tab as string | undefined
if (queryPreset && ['1h', '6h', '24h', '7d', '30d'].includes(queryPreset)) {
  timePreset.value = queryPreset
}
if (queryStatus && ['all', 'firing', 'acked', 'resolved'].includes(queryStatus)) {
  statusTab.value = queryStatus as StatusTab
}

const canViewAll = computed(
  () => authStore.user?.role === 'admin' || authStore.user?.role === 'team_lead',
)

// ===== Paginated list =====
const {
  loading,
  items: events,
  total,
  page,
  pageSize,
  fetchList,
  refresh,
} = usePaginatedList<AlertEvent>({
  apiFn: alertEventApi.list,
  pageSize: 20,
  extraParams: () => {
    const tr = getTimeRange()
    return {
      status: statusFilterArray(),
      severity: severityFilter.value ? [severityFilter.value] : undefined,
      alert_name: search.value || undefined,
      rule_id: ruleFilter.value ?? undefined, // FE4-4: server-side filter
      view_mode: viewMode.value,
      ...tr,
    }
  },
  onError: (err: unknown) => {
    message.error(getErrorMessage(err))
  },
})

// FE4-6 KNOWN LIMITATION: tagFilter is applied client-side on the current page
// only.  This means pagination total does not reflect this filter.
// To fix properly, label-key filtering should be sent as server-side query params.
const filteredEvents = computed(() => {
  let list = events.value
  // FE4-4: rule_id is now a server-side filter — no client-side filtering needed
  if (tagFilter.value.trim()) {
    const [k, v] = tagFilter.value.split('=').map((s) => s.trim())
    if (k) {
      list = list.filter((e) => {
        const lv = e.labels?.[k]
        return v ? lv === v : lv != null
      })
    }
  }
  return list
})

watch(loading, (isLoading) => {
  if (!isLoading) firstLoad.value = false
})

const severityOptions = [
  { label: () => t('alert.p0'), value: 'p0' },
  { label: () => t('alert.p1'), value: 'p1' },
  { label: () => t('alert.p2'), value: 'p2' },
  { label: () => t('alert.p3'), value: 'p3' },
  { label: () => t('alert.p4'), value: 'p4' },
  { label: () => t('alert.critical'), value: 'critical' },
  { label: () => t('alert.warning'), value: 'warning' },
  { label: () => t('alert.info'), value: 'info' },
]

const timePresetOptions = [
  { label: () => t('alert.last1h'), value: '1h' },
  { label: () => t('alert.last6h'), value: '6h' },
  { label: () => t('alert.last24h'), value: '24h' },
  { label: () => t('alert.last7d'), value: '7d' },
  { label: () => t('alert.last30d'), value: '30d' },
  { label: () => t('alert.custom'), value: 'custom' },
]

const refreshOptions = [
  { label: () => t('common.off'), value: 0 },
  { label: '30s', value: 30 },
  { label: '60s', value: 60 },
  { label: '5min', value: 300 },
]
const REFRESH_KEY = 'sre.alerts.refreshInterval'
const refreshInterval = ref<number>(
  Number(localStorage.getItem(REFRESH_KEY) ?? 30),
)

// ===== Rule list (for filter) =====
const ruleOptions = shallowRef<{ label: string; value: number }[]>([])
async function loadRules() {
  try {
    const { data } = await alertRuleApi.list({ page: 1, page_size: 200 })
    const list = (data.data?.list || []) as AlertRule[]
    ruleOptions.value = list.map((r) => ({ label: r.name, value: r.id }))
  } catch {
    /* silent */
  }
}

// ===== Time range =====
function getTimeRange(): { start_time?: string; end_time?: string } {
  if (timePreset.value === 'custom' && customRange.value) {
    return {
      start_time: new Date(customRange.value[0]).toISOString(),
      end_time: new Date(customRange.value[1]).toISOString(),
    }
  }
  const now = Date.now()
  const map: Record<string, number> = {
    '1h': 3600000,
    '6h': 21600000,
    '24h': 86400000,
    '7d': 604800000,
    '30d': 2592000000,
  }
  const ms = map[timePreset.value]
  if (ms) return { start_time: new Date(now - ms).toISOString() }
  return {}
}

function statusFilterArray(): string[] | undefined {
  switch (statusTab.value) {
    case 'firing':
      return ['firing']
    case 'acked':
      return ['acknowledged', 'assigned']
    case 'resolved':
      return ['resolved', 'closed']
    default:
      return undefined
  }
}

// ===== Refilter =====
let searchTimer: ReturnType<typeof setTimeout> | null = null
function refilter() {
  refresh()
}
function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => refilter(), 300)
}

// ===== Actions =====
async function onAck(ev: AlertEvent) {
  try {
    await alertEventApi.acknowledge(ev.id)
    message.success(t('alert.alertAcknowledged'))
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}
async function onResolve(ev: AlertEvent) {
  try {
    await alertEventApi.resolve(ev.id, { resolution: t('alert.manuallyResolved') })
    message.success(t('alert.alertResolved'))
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}
async function onClose(ev: AlertEvent) {
  try {
    await alertEventApi.close(ev.id)
    message.success(t('alert.alertClosed'))
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}
async function onSilence(ev: AlertEvent) {
  try {
    await alertEventApi.silence(ev.id, { duration_minutes: 60, reason: 'manual' })
    message.success(t('alert.silenced'))
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}
const batchLoading = ref(false)
async function batchAck() {
  const ids = Array.from(selected.value)
  if (!ids.length) return
  batchLoading.value = true
  try {
    await alertEventApi.batchAcknowledge(ids)
    message.success(t('alert.batchAckSuccess'))
    selected.value = new Set()
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally { batchLoading.value = false }
}
async function batchCloseAction() {
  const ids = Array.from(selected.value)
  if (!ids.length) return
  batchLoading.value = true
  try {
    await alertEventApi.batchClose(ids)
    message.success(t('alert.batchCloseSuccess'))
    selected.value = new Set()
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally { batchLoading.value = false }
}
async function batchSilence() {
  const ids = Array.from(selected.value)
  if (!ids.length) return
  try {
    await Promise.all(
      ids.map((id) =>
        alertEventApi.silence(id, { duration_minutes: 60, reason: 'manual batch' }),
      ),
    )
    message.success(t('alert.silenced'))
    selected.value = new Set()
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

function rowActions(ev: AlertEvent) {
  const opts = [
    { label: t('alert.detail'), key: 'detail' },
    { label: t('alert.silence'), key: 'silence' },
  ]
  if (ev.status === 'firing' || ev.status === 'acknowledged') {
    opts.unshift({ label: t('alert.resolve'), key: 'resolve' })
  }
  return opts
}
function handleAction(key: string, ev: AlertEvent) {
  if (key === 'detail') router.push(`/alert/events/${ev.id}`)
  else if (key === 'silence') onSilence(ev)
  else if (key === 'resolve') onResolve(ev)
}

// ===== Selection =====
function toggleSelect(id: number) {
  const next = new Set(selected.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selected.value = next
}
function clearSelection() {
  selected.value = new Set()
  crossPageActive.value = false
}

// ===== Navigation =====
function goDetail(ev: AlertEvent) {
  router.push(`/alert/events/${ev.id}`)
}

// ===== Helpers =====
function relTime(input: string | number | null | undefined): string {
  if (!input) return '—'
  const ts = typeof input === 'number' ? input : Date.parse(input)
  if (Number.isNaN(ts)) return '—'
  const diff = Math.max(0, Math.floor((Date.now() - ts) / 1000))
  if (diff < 60) return t('alert.secsAgo', { n: diff })
  if (diff < 3600) return t('alert.minsAgo', { n: Math.floor(diff / 60) })
  if (diff < 86400) return t('alert.hoursAgo', { n: Math.floor(diff / 3600) })
  return t('alert.daysAgo', { n: Math.floor(diff / 86400) })
}

function severityLabel(sev: string): string {
  const k = `alert.${sev}`
  const v = t(k)
  return v === k ? sev.toUpperCase() : v
}

function statusLabel(status: string): string {
  const map: Record<string, string> = {
    firing: 'alert.firing',
    acknowledged: 'alert.acknowledged',
    assigned: 'alert.assigned',
    resolved: 'alert.resolved',
    closed: 'alert.closed',
    silenced: 'alert.silenced',
  }
  return map[status] ? t(map[status]) : status
}

function severityDotKey(sev: string): string {
  if (['p0', 'critical'].includes(sev)) return 'critical'
  if (['p1', 'p2', 'warning'].includes(sev)) return 'warning'
  if (['p3', 'info'].includes(sev)) return 'info'
  if (['p4', 'success'].includes(sev)) return 'success'
  return 'info'
}

function hasLabels(ev: AlertEvent): boolean {
  return !!ev.labels && Object.keys(ev.labels).length > 0
}

function assigneeInitial(u: { display_name?: string; username?: string }): string {
  const n = u.display_name || u.username || '?'
  return n.charAt(0).toUpperCase()
}

function isDim(ev: AlertEvent): boolean {
  return ev.status === 'resolved' || ev.status === 'closed'
}

const allOnPageSelected = computed(
  () =>
    filteredEvents.value.length > 0 &&
    filteredEvents.value.every((e) => selected.value.has(e.id)),
)

function toggleSelectAll() {
  if (allOnPageSelected.value) {
    const next = new Set(selected.value)
    filteredEvents.value.forEach((e) => next.delete(e.id))
    selected.value = next
    crossPageActive.value = false
  } else {
    const next = new Set(selected.value)
    filteredEvents.value.forEach((e) => next.add(e.id))
    selected.value = next
  }
}

async function selectAllAcrossPages() {
  crossPageLoading.value = true
  try {
    const tr = getTimeRange()
    const params: Record<string, unknown> = {
      page: 1,
      page_size: total.value || 10000,
      status: statusFilterArray(),
      severity: severityFilter.value ? [severityFilter.value] : undefined,
      alert_name: search.value || undefined,
      view_mode: viewMode.value,
      ...tr,
    }
    const { data } = await alertEventApi.list(params as never)
    const allItems = data.data?.list || []
    const next = new Set<number>()
    allItems.forEach((e: AlertEvent) => next.add(e.id))
    selected.value = next
    crossPageActive.value = true
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    crossPageLoading.value = false
  }
}

function clearCrossPageSelection() {
  crossPageActive.value = false
  selected.value = new Set()
}

// ===== Export =====
function handleExportCSV() {
  const tr = getTimeRange()
  const params = new URLSearchParams()
  const sf = statusFilterArray()
  if (sf) params.set('status', sf.join(','))
  if (severityFilter.value) params.set('severity', severityFilter.value)
  if (search.value) params.set('alert_name', search.value)
  if (tr.start_time) params.set('start_time', tr.start_time)
  if (tr.end_time) params.set('end_time', tr.end_time)
  params.set('view_mode', viewMode.value)
  const url = `/api/v1/alert-events/export?${params.toString()}`
  const a = document.createElement('a')
  a.href = url
  a.download = `alert-events-${new Date().toISOString().slice(0, 10)}.csv`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

// ===== Auto-refresh (防重入锁：前一次请求未完成时跳过本轮) =====
let timer: ReturnType<typeof setInterval> | null = null
let refreshLock = false
function applyAutoRefresh() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
  refreshLock = false
  if (refreshInterval.value > 0) {
    timer = setInterval(() => {
      if (refreshLock) return
      refreshLock = true
      fetchList().finally(() => { refreshLock = false })
    }, refreshInterval.value * 1000)
  }
  localStorage.setItem(REFRESH_KEY, String(refreshInterval.value))
}

onMounted(() => {
  loadRules()
  fetchList()
  applyAutoRefresh()
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})

// ===== Render helpers (icons via h to avoid template noise) =====
const EllipsisIcon = () => h(NIcon, { component: EllipsisHorizontalOutline })
</script>

<template>
  <div class="ae-page sre-stagger">
    <!-- Header -->
    <PageHeader :title="t('alert.events')" :subtitle="t('alert.eventsSubtitle')">
      <template #actions>
        <NSelect
          :value="refreshInterval"
          :options="refreshOptions"
          size="small"
          class="ae-filter-sm"
          @update:value="(v: number) => { refreshInterval = v; applyAutoRefresh() }"
        />
        <NButton circle quaternary size="small" :loading="loading" @click="fetchList">
          <template #icon><NIcon :component="RefreshOutline" /></template>
        </NButton>
        <NButton size="small" @click="handleExportCSV">
          <template #icon><NIcon :component="DownloadOutline" /></template>
          {{ t('alert.exportCSV') }}
        </NButton>
      </template>
    </PageHeader>

    <!-- Status tabs + filters -->
    <section class="ae-filters">
      <div class="ae-filter-row">
        <NRadioGroup
          :value="statusTab"
          size="small"
          @update:value="(v: StatusTab) => { statusTab = v; refilter() }"
        >
          <NRadioButton value="all">{{ t('common.all') }}</NRadioButton>
          <NRadioButton value="firing">{{ t('alert.firing') }}</NRadioButton>
          <NRadioButton value="acked">{{ t('alert.acknowledged') }}</NRadioButton>
          <NRadioButton value="resolved">{{ t('alert.resolved') }}</NRadioButton>
        </NRadioGroup>

        <div v-if="canViewAll" class="ae-view-mode">
          <NRadioGroup
            :value="viewMode"
            size="small"
            @update:value="(v: AlertViewMode) => { viewMode = v; refilter() }"
          >
            <NRadioButton value="mine">{{ t('alert.myAlerts') }}</NRadioButton>
            <NRadioButton value="unassigned">{{ t('alert.unassigned') }}</NRadioButton>
            <NRadioButton value="all">{{ t('alert.allAlerts') }}</NRadioButton>
          </NRadioGroup>
        </div>
      </div>

      <div class="ae-filter-row ae-filter-row--inputs">
        <NInput
          v-model:value="search"
          size="small"
          clearable
          :placeholder="t('alert.alertNameSearch')"
          class="ae-filter-search"
          @update:value="onSearchInput"
        />
        <NSelect
          v-model:value="severityFilter"
          :options="severityOptions"
          size="small"
          clearable
          :placeholder="t('alert.severity')"
          class="ae-filter-sev"
          @update:value="refilter"
        />
        <NSelect
          v-model:value="ruleFilter"
          :options="ruleOptions"
          size="small"
          clearable
          filterable
          :placeholder="t('alert.rule')"
          class="ae-filter-rule"
          @update:value="refilter"
        />
        <NInput
          v-model:value="tagFilter"
          size="small"
          clearable
          :placeholder="t('alert.filterPlaceholder')"
          class="ae-filter-tag"
          @update:value="refilter"
        />
        <NSelect
          v-model:value="timePreset"
          :options="timePresetOptions"
          size="small"
          class="ae-filter-sm"
          @update:value="refilter"
        />
        <NDatePicker
          v-if="timePreset === 'custom'"
          v-model:value="customRange"
          type="daterange"
          size="small"
          clearable
          @update:value="refilter"
        />
      </div>
    </section>

    <!-- Selection bar -->
    <transition name="ae-fade">
      <div v-if="selected.size > 0" class="ae-selection-bar" role="toolbar" :aria-label="t('alert.batchActions')">
        <span class="ae-selection-count tnum">{{ selected.size }} {{ t('alert.selected') }}</span>
        <span v-if="crossPageActive" class="ae-crosspage-badge">{{ t('alert.crossPageActive') }}</span>
        <NButton size="small" type="primary" :loading="batchLoading" @click="batchAck">
          {{ t('alert.batchAck') }}
        </NButton>
        <NButton size="small" :loading="batchLoading" @click="batchCloseAction">
          {{ t('alert.batchClose') }}
        </NButton>
        <NButton size="small" type="warning" ghost @click="batchSilence">
          {{ t('alert.silence') }}
        </NButton>
        <div class="ae-spacer" />
        <NButton circle quaternary size="small" @click="clearSelection">
          <template #icon><NIcon :component="CloseOutline" /></template>
        </NButton>
      </div>
    </transition>

    <!-- Select-all hairline -->
    <div v-if="filteredEvents.length > 0" class="ae-selectall">
      <input
        type="checkbox"
        class="ec-check"
        :checked="allOnPageSelected"
        :aria-label="allOnPageSelected ? t('alert.deselectAll') : t('alert.selectAllPage')"
        @change="toggleSelectAll"
      />
      <span class="ae-selectall-label">{{ allOnPageSelected ? t('alert.deselectAll') : t('alert.selectAllPage') }}</span>
      <span v-if="!crossPageActive && total > filteredEvents.length" class="ae-selectall-cross">
        <span class="sre-meta-divider"></span>
        <NButton
          text
          size="tiny"
          :loading="crossPageLoading"
          @click="selectAllAcrossPages"
        >{{ t('alert.selectAllN', { n: total }) }}</NButton>
      </span>
      <span class="sre-meta-divider"></span>
      <span class="tnum">{{ total }} {{ t('alert.items') }}</span>
    </div>

    <!-- Event list -->
    <LoadingSkeleton v-if="loading && firstLoad && filteredEvents.length === 0" :rows="6" variant="row" />
    <NSpin v-else :show="loading && !firstLoad">
      <DynamicScroller
        v-if="filteredEvents.length > 0"
        class="event-list"
        :class="{ 'sre-stagger': firstLoad }"
        :items="filteredEvents"
        key-field="id"
        :min-item-size="80"
      >
        <template #default="{ item: ev }">
          <DynamicScrollerItem
            :item="ev"
            :active="true"
            :size-dependencies="[ev.labels, ev.acked_by_user, ev.oncall_user]"
          >
            <div
              class="sre-row-card event-row"
              :data-severity="severityDotKey(ev.severity)"
              :data-dim="isDim(ev) || undefined"
              @click="goDetail(ev)"
            >
          <input
            type="checkbox"
            class="ec-check"
            :checked="selected.has(ev.id)"
            :aria-label="`${t('alert.selectAlert')} ${ev.alert_name}`"
            @click.stop
            @change="toggleSelect(ev.id)"
          />
          <div class="ec-main">
            <div class="ec-headline">
              <span class="sre-dot" :data-severity="severityDotKey(ev.severity)"></span>
              <span class="ec-sev-label">{{ severityLabel(ev.severity) }}</span>
              <span class="ec-title">{{ ev.alert_name }}</span>
            </div>
            <div class="ec-context">
              <span>{{ t('alert.ruleLabel') }} {{ ev.rule?.name || '—' }}</span>
              <span class="sre-meta-divider"></span>
              <span>{{ t('alert.datasourceLabel') }} {{ ev.source || '—' }}</span>
            </div>
            <div v-if="hasLabels(ev)" class="ec-labels">
              <span
                v-for="(v, k) in ev.labels"
                :key="k"
                class="ec-chip"
              >{{ k }}={{ v }}</span>
            </div>
            <div class="ec-footer">
              <span class="tnum">{{ ev.fire_count }} {{ t('alert.fireCount') }}</span>
              <span class="sre-meta-divider"></span>
              <span>{{ t('alert.firstTrigger') }} {{ relTime(ev.fired_at) }}</span>
              <span class="sre-meta-divider"></span>
              <span>{{ t('alert.lastTrigger') }} {{ relTime(ev.acked_at || ev.fired_at) }}</span>
              <template v-if="ev.acked_by_user">
                <span class="sre-meta-divider"></span>
                <span class="ec-assignee">
                  <span class="ec-avatar">{{ assigneeInitial(ev.acked_by_user) }}</span>
                  {{ ev.acked_by_user.display_name || ev.acked_by_user.username }}
                </span>
              </template>
              <template v-else-if="ev.oncall_user">
                <span class="sre-meta-divider"></span>
                <span class="ec-assignee">
                  <span class="ec-avatar">{{ assigneeInitial(ev.oncall_user) }}</span>
                  {{ ev.oncall_user.display_name || ev.oncall_user.username }}
                </span>
              </template>
            </div>
          </div>
          <div class="ec-status">
            <span class="sre-dot" :data-status="ev.status"></span>
            <span class="ec-status-text">{{ statusLabel(ev.status) }}</span>
          </div>
          <div class="ec-actions" @click.stop>
            <NButton
              v-if="ev.status === 'firing' && hasPerm('events.ack')"
              size="tiny"
              type="primary"
              @click="onAck(ev)"
            >{{ t('alert.claim') }}</NButton>
            <NButton
              v-if="ev.status !== 'closed' && ev.status !== 'resolved' && hasPerm('events.manage')"
              size="tiny"
              quaternary
              @click="onClose(ev)"
            >{{ t('alert.closeAlert') }}</NButton>
            <NDropdown
              :options="rowActions(ev)"
              trigger="click"
              @select="(k: string) => handleAction(k, ev)"
            >
              <NButton quaternary circle size="small">
                <template #icon>
                  <NIcon :component="EllipsisHorizontalOutline" />
                </template>
              </NButton>
            </NDropdown>
          </div>
            </div>
          </DynamicScrollerItem>
        </template>
      </DynamicScroller>

      <!-- Empty state -->
      <EmptyState
        v-else-if="!loading"
        :icon="ShieldCheckmarkOutline"
        :title="t('alert.allQuiet')"
        :description="t('alert.noActiveAlerts')"
      />
    </NSpin>

    <!-- Pagination -->
    <div v-if="total > pageSize" class="ae-pagination">
      <NPagination
        :page="page"
        :page-size="pageSize"
        :item-count="total"
        :page-sizes="[20, 50, 100]"
        show-size-picker
        @update:page="(p: number) => { page = p; clearSelection(); fetchList() }"
        @update:page-size="(s: number) => { pageSize = s; page = 1; fetchList() }"
      />
    </div>
  </div>
</template>

<style scoped>
.ae-page {
  max-width: 1440px;
  font-family: var(--sre-font-sans);
}

/* ===== Header ===== */
.ae-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 4px 0 20px;
  border-bottom: var(--sre-hairline);
  margin-bottom: 20px;
}
.ae-title {
  font-family: var(--sre-font-sans);
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--sre-text-primary);
  margin: 0 0 4px;
  line-height: 1.2;
}
.ae-subtitle {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin: 0;
}
.ae-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* ===== Filters ===== */
.ae-filters {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}
.ae-filter-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.ae-filter-row--inputs {
  padding-top: 4px;
}
.ae-filter-sm { width: 110px; }
.ae-filter-sev { width: 130px; }
.ae-filter-rule { width: 200px; }
.ae-filter-tag { width: 180px; }
.ae-filter-search { width: 240px; }
.ae-spacer { flex: 1; }
.ae-view-mode {
  margin-left: auto;
}

/* ===== Selection bar ===== */
.ae-selection-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--sre-primary-soft);
  border-radius: 8px;
  padding: 10px 16px;
  margin-bottom: 12px;
  border: var(--sre-hairline);
}
.ae-selection-count {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-primary);
  margin-right: 4px;
}
.ae-fade-enter-active,
.ae-fade-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}
.ae-fade-enter-from,
.ae-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

/* ===== Select-all ===== */
.ae-selectall {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  padding: 4px 14px 8px;
}
.ae-selectall-label {
  cursor: pointer;
}
.ae-selectall-cross {
  display: inline-flex;
  align-items: center;
}
.ae-crosspage-badge {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-primary);
  background: var(--sre-primary-soft);
  border-radius: 4px;
  padding: 2px 6px;
  margin-right: 4px;
}

/* ===== Event list ===== */
.event-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: calc(100vh - 320px);
  overflow-y: auto;
}
.event-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 0.12s ease, border-color 0.12s ease, transform 0.12s ease;
}
.event-row:hover {
  background: var(--sre-bg-elevated);
}
.ec-check {
  width: 14px;
  height: 14px;
  align-self: center;
  cursor: pointer;
  flex-shrink: 0;
  accent-color: var(--sre-primary);
}
.ec-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ec-headline {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  font-family: var(--sre-font-sans);
}
.ec-sev-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.6px;
}
.ec-title {
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ec-context,
.ec-footer {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0;
}
.ec-context > span,
.ec-footer > span {
  display: inline-flex;
  align-items: center;
}
.ec-labels {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.ec-chip {
  font-size: 11px;
  font-family: var(--sre-font-mono, 'Geist Mono', ui-monospace, monospace);
  background: var(--sre-bg-elevated);
  border-radius: 4px;
  padding: 2px 6px;
  color: var(--sre-text-secondary);
  border: 1px solid var(--sre-border);
}
.ec-assignee {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.ec-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
  font-size: 10px;
  font-weight: 600;
}
.ec-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--sre-text-secondary);
  flex-shrink: 0;
  padding-right: 4px;
  min-width: 80px;
}
.ec-status-text {
  font-variant-numeric: tabular-nums;
  text-transform: capitalize;
}
.ec-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

/* Status dot fallbacks (in case not in global) */
:deep(.sre-dot[data-status='firing']) {
  background: var(--sre-critical);
}
:deep(.sre-dot[data-status='acked']),
:deep(.sre-dot[data-status='acknowledged']),
:deep(.sre-dot[data-status='assigned']) {
  background: var(--sre-warning);
}
:deep(.sre-dot[data-status='resolved']) {
  background: var(--sre-primary);
}
:deep(.sre-dot[data-status='closed']) {
  background: var(--sre-text-tertiary);
}
:deep(.sre-dot[data-status='silenced']) {
  background: var(--sre-info);
}

/* ===== Pagination ===== */
.ae-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
  padding-top: 12px;
  border-top: var(--sre-hairline);
}

/* ===== Responsive: narrow screens ===== */
@media (max-width: 768px) {
  .ae-filter-row {
    flex-direction: column;
    align-items: stretch;
  }
  .ae-filter-row--inputs {
    gap: 6px;
  }
  .ae-filter-sm,
  .ae-filter-sev,
  .ae-filter-rule,
  .ae-filter-tag,
  .ae-filter-search {
    width: 100%;
  }
  .ae-view-mode {
    margin-left: 0;
  }
  .event-row {
    flex-wrap: wrap;
    gap: 8px;
  }
  .ec-status {
    min-width: unset;
  }
  .ec-actions {
    width: 100%;
    justify-content: flex-end;
  }
  .ae-selection-bar {
    flex-wrap: wrap;
  }
}
</style>
