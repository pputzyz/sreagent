<script setup lang="ts">
import { ref, shallowRef, computed, onMounted } from 'vue'
import { NRadioGroup, NRadioButton, NSelect, NDatePicker, NInput, NPagination, NSpin, NIcon } from 'naive-ui'
import { ListOutline, SearchOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { auditLogApi } from '@/api'
import type { AuditLog } from '@/types'
import { formatTime } from '@/utils/format'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()

const loading = ref(false)
const logs = shallowRef<AuditLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const firstLoad = ref(true)

type RangePreset = 'today' | '7d' | '30d' | 'custom'
const rangePreset = ref<RangePreset>('7d')
const customRange = ref<[number, number] | null>(null)

const filterAction = ref<string | null>(null)
const filterResourceType = ref<string | null>(null)
const filterUser = ref<string | null>(null)
const search = ref('')

const actionOptions = [
  { label: 'CREATE', value: 'create' },
  { label: 'UPDATE', value: 'update' },
  { label: 'DELETE', value: 'delete' },
  { label: 'TOGGLE', value: 'toggle' },
  { label: 'ACKNOWLEDGE', value: 'acknowledge' },
  { label: 'ASSIGN', value: 'assign' },
  { label: 'RESOLVE', value: 'resolve' },
  { label: 'CLOSE', value: 'close' },
  { label: 'SILENCE', value: 'silence' },
  { label: 'LOGIN', value: 'login' },
  { label: 'LOGOUT', value: 'logout' },
]

const resourceOptions = [
  { label: 'alert_rule', value: 'alert_rule' },
  { label: 'alert_event', value: 'alert_event' },
  { label: 'incident', value: 'incident' },
  { label: 'user', value: 'user' },
  { label: 'team', value: 'team' },
  { label: 'datasource', value: 'datasource' },
  { label: 'notify_rule', value: 'notify_rule' },
  { label: 'mute_rule', value: 'mute_rule' },
]

const userOptions = computed(() => {
  const seen = new Set<string>()
  const out: Array<{ label: string; value: string }> = []
  for (const log of logs.value) {
    const u = log.username || ''
    if (u && !seen.has(u)) {
      seen.add(u)
      out.push({ label: u, value: u })
    }
  }
  return out
})

function computeRange(): [Date, Date] | null {
  const now = new Date()
  if (rangePreset.value === 'today') {
    const start = new Date(now)
    start.setHours(0, 0, 0, 0)
    return [start, now]
  }
  if (rangePreset.value === '7d') {
    return [new Date(now.getTime() - 7 * 86400_000), now]
  }
  if (rangePreset.value === '30d') {
    return [new Date(now.getTime() - 30 * 86400_000), now]
  }
  if (rangePreset.value === 'custom' && customRange.value) {
    return [new Date(customRange.value[0]), new Date(customRange.value[1])]
  }
  return null
}

async function fetchLogs() {
  loading.value = true
  try {
    const params: Record<string, any> = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (filterAction.value) params.action = filterAction.value
    if (filterResourceType.value) params.resource_type = filterResourceType.value
    if (filterUser.value) params.username = filterUser.value
    if (search.value.trim()) params.q = search.value.trim()
    const range = computeRange()
    if (range) {
      params.start_time = range[0].toISOString()
      params.end_time = range[1].toISOString()
    }
    const { data } = await auditLogApi.list(params)
    logs.value = data.data.list || []
    total.value = data.data.total || 0
  } catch {
    logs.value = []
    total.value = 0
  } finally {
    loading.value = false
    firstLoad.value = false
  }
}

function reset() {
  page.value = 1
  fetchLogs()
}

function actionTone(action: string): 'success' | 'warning' | 'critical' | 'info' {
  const a = (action || '').toUpperCase()
  if (a.includes('CREATE')) return 'success'
  if (a.includes('DELETE')) return 'critical'
  if (a.includes('UPDATE') || a.includes('MODIFY') || a.includes('TOGGLE')) return 'warning'
  return 'info'
}

function truncateUA(ua: string): string {
  if (!ua) return ''
  if (ua.length <= 60) return ua
  return ua.slice(0, 60) + '…'
}

onMounted(fetchLogs)
</script>

<template>
  <div class="audit-page">
    <header class="page-header">
      <div>
        <h2 class="page-title">{{ t('settings.auditLog') || 'Audit Log' }}</h2>
        <p class="page-subtitle">{{ t('settings.auditSubtitle') }}</p>
      </div>
    </header>

    <div class="filter-bar">
      <div class="filter-row">
        <span class="sre-label-eyebrow">{{ t('settings.auditTimeLabel') }}</span>
        <NRadioGroup v-model:value="rangePreset" size="small" @update:value="reset">
          <NRadioButton value="today">{{ t('settings.auditToday') }}</NRadioButton>
          <NRadioButton value="7d">{{ t('settings.audit7d') }}</NRadioButton>
          <NRadioButton value="30d">{{ t('settings.audit30d') }}</NRadioButton>
          <NRadioButton value="custom">{{ t('common.custom') }}</NRadioButton>
        </NRadioGroup>
        <NDatePicker
          v-if="rangePreset === 'custom'"
          v-model:value="customRange"
          type="daterange"
          size="small"
          clearable
          class="filter-datepicker"
          @update:value="reset"
        />
      </div>

      <div class="filter-row">
        <NSelect
          v-model:value="filterAction"
          :options="actionOptions"
          :placeholder="t('settings.auditAction')"
          clearable
          size="small"
          class="filter-select-sm"
          @update:value="reset"
        />
        <NSelect
          v-model:value="filterResourceType"
          :options="resourceOptions"
          :placeholder="t('settings.auditResource')"
          clearable
          size="small"
          class="filter-select-md"
          @update:value="reset"
        />
        <NSelect
          v-model:value="filterUser"
          :options="userOptions"
          :placeholder="t('settings.auditUser')"
          clearable
          filterable
          size="small"
          class="filter-select-sm"
          @update:value="reset"
        />
        <NInput
          v-model:value="search"
          :placeholder="t('settings.auditSearchPlaceholder')"
          size="small"
          clearable
          class="filter-search"
          @keydown.enter="reset"
          @clear="reset"
        >
          <template #prefix>
            <NIcon :component="SearchOutline" />
          </template>
        </NInput>
      </div>
    </div>

    <div class="timeline-wrap">
      <NSpin :show="loading && !firstLoad">
        <div v-if="loading && firstLoad" class="state-pad">
          <NSpin size="medium" />
        </div>
        <EmptyState
          v-else-if="!logs.length"
          :icon="ListOutline"
          :title="t('settings.auditLog') || 'Audit Log'"
          :description="t('settings.auditNoRecordsInRange')"
        />
        <div v-else class="audit-list sre-stagger">
          <div
            v-for="log in logs"
            :key="log.id"
            class="audit-item"
          >
            <div class="audit-rail">
              <span class="audit-dot" :data-action="actionTone(log.action)"></span>
            </div>
            <div class="audit-main">
              <div class="audit-headline">
                <span class="audit-time tnum">{{ formatTime(log.created_at) }}</span>
                <span class="audit-actor">{{ log.username || t('settings.auditSystem') }}</span>
                <span class="audit-action-chip" :data-action="actionTone(log.action)">
                  {{ (log.action || '').toUpperCase() }}
                </span>
                <span class="audit-resource">{{ log.resource_type }}</span>
                <span v-if="log.resource_name" class="audit-name">"{{ log.resource_name }}"</span>
                <span v-else-if="log.resource_id" class="audit-id">#{{ log.resource_id }}</span>
                <span
                  v-if="log.status && log.status !== 'success'"
                  class="audit-status-fail"
                >{{ t('settings.auditFailed') }}</span>
              </div>
              <div v-if="log.detail" class="audit-detail">{{ log.detail }}</div>
              <div v-if="log.ip" class="audit-meta">
                <span class="mono">IP {{ log.ip }}</span>
              </div>
            </div>
          </div>
        </div>
      </NSpin>
    </div>

    <div v-if="total > pageSize" class="pager">
      <NPagination
        v-model:page="page"
        v-model:page-size="pageSize"
        :item-count="total"
        :page-sizes="[20, 50, 100]"
        size="small"
        show-size-picker
        @update:page="fetchLogs"
        @update:page-size="() => { page = 1; fetchLogs() }"
      />
    </div>
  </div>
</template>

<style scoped>
.audit-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  padding-bottom: 14px;
  border-bottom: var(--sre-hairline);
}
.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--sre-text-primary);
}
.page-subtitle {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--sre-text-tertiary);
}

.filter-bar {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.filter-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.filter-datepicker { width: 280px; }
.filter-select-sm { width: 160px; }
.filter-select-md { width: 180px; }
.filter-search { flex: 1; min-width: 200px; max-width: 320px; }

.timeline-wrap {
  position: relative;
  min-height: 200px;
}

.state-pad {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 80px 0;
}

.audit-list {
  display: flex;
  flex-direction: column;
}

.audit-item {
  display: grid;
  grid-template-columns: 24px 1fr;
  gap: 12px;
  padding: 12px 16px;
  border-radius: var(--sre-radius-sm);
  position: relative;
  transition: background 120ms ease;
}
.audit-item:hover {
  background: var(--sre-bg-hover);
}

.audit-rail {
  position: relative;
  display: flex;
  justify-content: center;
}
.audit-rail::before {
  content: '';
  position: absolute;
  left: 50%;
  top: -12px;
  bottom: -12px;
  width: 1px;
  background: var(--sre-hairline);
  transform: translateX(-50%);
}
.audit-item:first-child .audit-rail::before { top: 50%; }
.audit-item:last-child .audit-rail::before { bottom: 50%; }

.audit-dot {
  position: relative;
  z-index: 1;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--sre-text-tertiary);
  margin-top: 8px;
  border: 2px solid var(--sre-bg-page);
}
.audit-dot[data-action="success"]  { background: var(--sre-primary); }
.audit-dot[data-action="warning"]  { background: var(--sre-warning); }
.audit-dot[data-action="critical"] { background: var(--sre-critical); }
.audit-dot[data-action="info"]     { background: var(--sre-info); }

.audit-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.audit-headline {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 13px;
}
.audit-time {
  font-family: var(--sre-font-mono);
  font-size: 12px;
  color: var(--sre-text-tertiary);
  font-variant-numeric: tabular-nums;
}
.audit-actor {
  font-weight: 600;
  color: var(--sre-text-primary);
}
.audit-action-chip {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  font-family: var(--sre-font-mono);
  letter-spacing: 0.3px;
}
.audit-action-chip[data-action="success"]  { background: var(--sre-primary-soft); color: var(--sre-primary); }
.audit-action-chip[data-action="warning"]  { background: var(--sre-warning-soft); color: var(--sre-warning); }
.audit-action-chip[data-action="critical"] { background: var(--sre-critical-soft); color: var(--sre-critical); }
.audit-action-chip[data-action="info"]     { background: var(--sre-info-soft); color: var(--sre-info); }

.audit-resource {
  color: var(--sre-text-secondary);
  font-size: 13px;
}
.audit-name {
  color: var(--sre-text-primary);
  font-style: italic;
}
.audit-id {
  font-family: var(--sre-font-mono);
  font-size: 12px;
  color: var(--sre-text-tertiary);
}
.audit-status-fail {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--sre-critical-soft);
  color: var(--sre-critical);
  font-family: var(--sre-font-mono);
}

.audit-detail {
  font-size: 12px;
  color: var(--sre-text-secondary);
  font-family: var(--sre-font-mono);
  background: var(--sre-bg-elevated);
  padding: 4px 8px;
  border-radius: 4px;
  border-left: 2px solid var(--sre-hairline);
  word-break: break-word;
  white-space: pre-wrap;
}
.audit-meta {
  display: flex;
  align-items: center;
  gap: 0;
  font-size: 11px;
  color: var(--sre-text-tertiary);
}
.mono { font-family: var(--sre-font-mono); }

.pager {
  display: flex;
  justify-content: flex-end;
  padding-top: 8px;
}
</style>
