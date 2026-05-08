<script setup lang="ts">
import { ref, shallowRef, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, NRadioGroup, NRadioButton, NSelect, NInput, NDatePicker, NButton, NPagination, NSpin } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { alertEventApi, alertRuleApi, alertExportApi } from '@/api'
import type { AlertEvent, AlertRule } from '@/types'
import { ArchiveOutline, DownloadOutline } from '@vicons/ionicons5'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'

const router = useRouter()
const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const events = shallowRef<AlertEvent[]>([])
const rules = shallowRef<AlertRule[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const firstLoad = ref(true)

const range = ref<'7d' | '30d' | '90d' | 'custom'>('30d')
const customRange = ref<[number, number] | null>(null)
const severityFilter = ref<string | null>(null)
const ruleFilter = ref<number | null>(null)
const search = ref('')

const severityOptions = computed(() => [
  { label: t('alert.critical'), value: 'critical' },
  { label: t('alert.warning'), value: 'warning' },
  { label: t('alert.info'), value: 'info' },
])

const ruleOptions = computed(() =>
  rules.value.map(r => ({ label: r.name, value: r.id }))
)

function getTimeRange(): { start_time?: string; end_time?: string } {
  if (range.value === 'custom' && customRange.value) {
    return {
      start_time: new Date(customRange.value[0]).toISOString(),
      end_time: new Date(customRange.value[1]).toISOString(),
    }
  }
  const map: Record<string, number> = { '7d': 7, '30d': 30, '90d': 90 }
  const days = map[range.value]
  if (days) {
    return { start_time: new Date(Date.now() - days * 86400000).toISOString() }
  }
  return {}
}

async function fetchEvents() {
  loading.value = true
  try {
    const tr = getTimeRange()
    const { data } = await alertEventApi.list({
      page: page.value,
      page_size: pageSize.value,
      status: ['resolved', 'closed'],
      severity: severityFilter.value ? [severityFilter.value] : undefined,
      alert_name: search.value || undefined,
      rule_id: ruleFilter.value || undefined,
      ...tr,
    } as any)
    events.value = data.data.list || []
    total.value = data.data.total
  } catch (err: any) {
    message.error(err.message)
  } finally {
    loading.value = false
    firstLoad.value = false
  }
}

async function fetchRules() {
  try {
    const { data } = await alertRuleApi.list({ page: 1, page_size: 200 })
    rules.value = data.data.list || []
  } catch { /* silent */ }
}

function onFilterChange() {
  page.value = 1
  fetchEvents()
}

function onRangeChange(v: string) {
  range.value = v as any
  if (v !== 'custom') customRange.value = null
  onFilterChange()
}

function onCustomRange(v: [number, number] | null) {
  customRange.value = v
  if (v) {
    range.value = 'custom'
    onFilterChange()
  }
}

function exportCsv() {
  const tr = getTimeRange()
  const url = alertExportApi.exportCSV({
    status: 'resolved,closed',
    severity: severityFilter.value || undefined,
    start: tr.start_time,
    end: tr.end_time,
  })
  window.open(url, '_blank')
}

function severityLabel(s: string) {
  return t(`alert.${s}`) || s.toUpperCase()
}

function statusLabel(s: string) {
  return t(`alert.${s}`) || s
}

function goDetail(ev: AlertEvent) {
  router.push(`/alerts/events/${ev.id}`)
}

function duration(start?: string, end?: string): string {
  if (!start) return '—'
  const a = new Date(start).getTime()
  const b = end ? new Date(end).getTime() : Date.now()
  const s = Math.max(0, Math.floor((b - a) / 1000))
  if (s < 60) return `${s}s`
  if (s < 3600) return `${Math.floor(s / 60)}m`
  if (s < 86400) return `${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`
  return `${Math.floor(s / 86400)}d ${Math.floor((s % 86400) / 3600)}h`
}

function relTime(iso?: string): string {
  if (!iso) return '—'
  const diff = Date.now() - new Date(iso).getTime()
  const s = Math.floor(diff / 1000)
  if (s < 60) return `${s}s ago`
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`
  return `${Math.floor(s / 86400)}d ago`
}

onMounted(() => {
  fetchRules()
  fetchEvents()
})
</script>

<template>
  <div class="hist-page">
    <!-- Header -->
    <header class="hist-header">
      <div>
        <h1 class="hist-title-main">{{ t('alert.history') }}</h1>
        <p class="hist-subtitle">{{ t('alert.historyDesc') }}</p>
      </div>
      <NButton size="small" @click="exportCsv">
        <template #icon><NIcon :component="DownloadOutline" /></template>
        {{ t('alert.exportCSV') }}
      </NButton>
    </header>

    <!-- Time range -->
    <div class="hist-toolbar">
      <div class="hist-toolbar-row">
        <span class="sre-label-eyebrow">{{ t('alert.time') }}</span>
        <NRadioGroup :value="range" size="small" @update:value="onRangeChange">
          <NRadioButton value="7d">{{ t('alert.last7d') }}</NRadioButton>
          <NRadioButton value="30d">{{ t('alert.last30d') }}</NRadioButton>
          <NRadioButton value="90d">{{ t('alert.last90d') }}</NRadioButton>
          <NRadioButton value="custom">{{ t('common.custom') }}</NRadioButton>
        </NRadioGroup>
        <NDatePicker
          v-if="range === 'custom'"
          type="datetimerange"
          :value="customRange"
          size="small"
          clearable
          class="hist-date-picker"
          @update:value="onCustomRange"
        />
      </div>

      <div class="hist-toolbar-row">
        <NSelect
          v-model:value="severityFilter"
          :options="severityOptions"
          :placeholder="t('alert.severity')"
          clearable
          size="small"
          class="hist-filter-sev"
          @update:value="onFilterChange"
        />
        <NSelect
          v-model:value="ruleFilter"
          :options="ruleOptions"
          :placeholder="t('alert.rule') || 'Rule'"
          filterable
          clearable
          size="small"
          class="hist-filter-rule"
          @update:value="onFilterChange"
        />
        <NInput
          v-model:value="search"
          :placeholder="t('alert.alertNameSearch')"
          clearable
          size="small"
          class="hist-filter-search"
          @update:value="onFilterChange"
        />
      </div>
    </div>

    <!-- List -->
    <LoadingSkeleton v-if="loading && firstLoad && events.length === 0" :rows="6" variant="row" />
    <NSpin v-else :show="loading && !firstLoad">
      <div v-if="events.length" class="hist-list" :class="{ 'sre-stagger': firstLoad }">
        <div
          v-for="ev in events"
          :key="ev.id"
          class="sre-row-card sre-lift hist-row"
          :data-severity="ev.severity"
          data-dim="true"
          @click="goDetail(ev)"
        >
          <div class="hist-main">
            <div class="hist-headline">
              <span class="sre-dot" :data-severity="ev.severity"></span>
              <span class="hist-sev-label">{{ severityLabel(ev.severity) }}</span>
              <span class="hist-title">{{ (ev as any).title || ev.alert_name }}</span>
            </div>
            <div class="hist-context">
              <span>{{ t('alert.ruleLabel') }} {{ (ev as any).rule?.name || '—' }}</span>
              <span class="sre-meta-divider"></span>
              <span>{{ t('alert.datasourceLabel') }} {{ (ev as any).datasource?.name || ev.source || '—' }}</span>
            </div>
            <div class="hist-footer">
              <span class="tnum">{{ t('alert.firedCount', { n: ev.fire_count || 0 }) }}</span>
              <span class="sre-meta-divider"></span>
              <span class="tnum">{{ t('alert.duration') }} {{ duration(ev.fired_at, ev.resolved_at || ev.closed_at || undefined) }}</span>
              <span class="sre-meta-divider"></span>
              <span class="tnum">{{ t('alert.recoveredFor', { n: relTime(ev.resolved_at || ev.closed_at || undefined) }) }}</span>
            </div>
          </div>
          <div class="hist-status">
            <span class="sre-dot" :data-severity="ev.status === 'resolved' ? 'success' : null"></span>
            <span class="hist-status-text">{{ statusLabel(ev.status) }}</span>
          </div>
        </div>
      </div>

      <EmptyState
        v-else-if="!loading"
        :icon="ArchiveOutline"
        :title="t('alert.noHistoryDesc')"
        size="md"
      />
    </NSpin>

    <!-- Pagination -->
    <div v-if="total > 0" class="hist-pagination">
      <NPagination
        v-model:page="page"
        v-model:page-size="pageSize"
        :item-count="total"
        :page-sizes="[20, 50, 100]"
        size="small"
        show-size-picker
        @update:page="fetchEvents"
        @update:page-size="() => { page = 1; fetchEvents() }"
      />
    </div>
  </div>
</template>

<style scoped>
.hist-page {
  max-width: 1280px;
  font-family: var(--sre-font-sans);
}

.hist-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--sre-hairline);
  margin-bottom: 16px;
}
.hist-title-main {
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.01em;
  margin: 0;
  color: var(--sre-text-primary);
}
.hist-subtitle {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin: 4px 0 0;
}

.hist-toolbar {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}
.hist-toolbar-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.hist-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.hist-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  cursor: pointer;
}

.hist-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.hist-headline {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
}
.hist-sev-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.6px;
}
.hist-title {
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.hist-context,
.hist-footer {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0;
}

.hist-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--sre-text-secondary);
  flex-shrink: 0;
  padding-right: 4px;
}

.hist-date-picker { width: 320px; }
.hist-filter-sev { width: 120px; }
.hist-filter-rule { width: 200px; }
.hist-filter-search { width: 240px; }

.hist-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
