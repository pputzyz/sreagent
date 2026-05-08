<script setup lang="ts">
import { onMounted, computed, shallowRef, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { alertV2Api, channelV2Api } from '@/api'
import type { AlertV2, Channel } from '@/types'
import PageHeader from '@/components/common/PageHeader.vue'
import {
  RefreshOutline,
  SearchOutline,
  ChevronForwardOutline,
  ShieldCheckmarkOutline,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

const loading = ref(false)
const alerts = shallowRef<AlertV2[]>([])
const channels = shallowRef<Channel[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const statusFilter = ref<string>('')
const severityFilter = ref<string>('')
const channelFilter = ref<number | null>(null)
const searchQuery = ref('')
const firstLoaded = ref(false)

const SEVERITY_LABEL: Record<string, string> = {
  critical: 'Critical',
  warning: 'Warning',
  info: 'Info',
  p0: 'P0', p1: 'P1', p2: 'P2', p3: 'P3', p4: 'P4',
}

function severityLabel(s: string) {
  return SEVERITY_LABEL[s] ?? s.toUpperCase()
}

function relTime(ts?: string) {
  if (!ts) return '—'
  const diff = Math.max(0, Date.now() - new Date(ts).getTime())
  const m = Math.floor(diff / 60000)
  if (m < 1) return t('common.justNow') || 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  const d = Math.floor(h / 24)
  return `${d}d ago`
}

async function loadAlerts() {
  loading.value = true
  try {
    const res = await alertV2Api.list({
      status: statusFilter.value,
      severity: severityFilter.value,
      channel_id: channelFilter.value ?? undefined,
      query: searchQuery.value,
      page: page.value,
      page_size: pageSize.value,
    } as any)
    alerts.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
    firstLoaded.value = true
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadChannels() {
  try {
    const res = await channelV2Api.list({ status: 'active', page: 1, page_size: 100 } as any)
    channels.value = res.data.data?.list ?? []
  } catch { /* silent */ }
}

const channelOptions = computed(() => [
  { label: t('common.all'), value: null as any },
  ...channels.value.map(c => ({ label: c.name, value: c.id })),
])

const severityOptions = [
  { label: 'Critical', value: 'critical' },
  { label: 'Warning', value: 'warning' },
  { label: 'Info', value: 'info' },
]

function goDetail(a: AlertV2) {
  router.push(`/alerts-v2/${a.id}`)
}

const isEmpty = computed(() => firstLoaded.value && !loading.value && alerts.value.length === 0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

onMounted(() => { loadAlerts(); loadChannels() })
</script>

<template>
  <div class="alertsv2-page">
    <PageHeader
      title="Alerts"
      subtitle="Aggregated alert series across all integrations"
    >
      <template #actions>
        <n-button circle quaternary @click="loadAlerts">
          <template #icon><n-icon :component="RefreshOutline" /></template>
        </n-button>
      </template>
    </PageHeader>

    <!-- Status segmented -->
    <div class="status-row">
      <span class="sre-label-eyebrow">Status</span>
      <n-radio-group
        v-model:value="statusFilter"
        size="medium"
        @update:value="loadAlerts"
      >
        <n-radio-button value="">{{ t('common.all') }}</n-radio-button>
        <n-radio-button value="firing">Firing</n-radio-button>
        <n-radio-button value="resolved">Resolved</n-radio-button>
      </n-radio-group>
    </div>

    <!-- Filters -->
    <div class="filter-row">
      <n-input
        v-model:value="searchQuery"
        :placeholder="t('common.search')"
        clearable
        size="small"
        class="search-box"
        @update:value="loadAlerts"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select
        v-model:value="severityFilter"
        :options="severityOptions"
        :placeholder="t('incident.severity')"
        clearable
        size="small"
        style="width: 140px"
        @update:value="loadAlerts"
      />
      <n-select
        v-model:value="channelFilter"
        :options="channelOptions"
        placeholder="Channel"
        clearable
        size="small"
        style="width: 200px"
        @update:value="loadAlerts"
      />
    </div>

    <!-- List -->
    <n-spin :show="loading">
      <div v-if="isEmpty" class="empty-state">
        <n-icon :component="ShieldCheckmarkOutline" size="48" class="empty-icon" />
        <div class="empty-title">No active alerts</div>
        <div class="empty-sub">All series quiet across your integrations.</div>
      </div>

      <div v-else class="alert-list sre-stagger">
        <div
          v-for="alert in alerts"
          :key="alert.id"
          class="sre-row-card"
          :data-severity="alert.severity"
          :data-dim="alert.status === 'resolved' || undefined"
          @click="goDetail(alert)"
        >
          <div class="ar-main">
            <div class="ar-headline">
              <span class="sre-dot" :data-severity="alert.severity"></span>
              <span class="ar-sev-label">{{ severityLabel(alert.severity) }}</span>
              <span class="ar-title">{{ alert.title }}</span>
            </div>
            <div class="ar-key">
              <span class="ar-label">alert_key:</span>
              <code class="ar-keyval">{{ alert.alert_key }}</code>
            </div>
            <div class="ar-footer">
              <span class="tnum">{{ alert.event_count }} {{ t('alertV2.events') || 'events' }}</span>
              <span class="sre-meta-divider"></span>
              <span>{{ alert.status === 'firing' ? 'firing' : 'resolved' }}</span>
              <span class="sre-meta-divider"></span>
              <span>{{ relTime(alert.last_fired_at) }}</span>
              <template v-if="alert.channel">
                <span class="sre-meta-divider"></span>
                <span>channel: {{ alert.channel.name }}</span>
              </template>
              <template v-if="alert.incident_id">
                <span class="sre-meta-divider"></span>
                <span>incident #{{ alert.incident_id }}</span>
              </template>
            </div>
          </div>
          <div class="ar-arrow">
            <n-icon :component="ChevronForwardOutline" :size="16" />
          </div>
        </div>
      </div>

      <div v-if="total > pageSize" class="pagination">
        <n-pagination
          v-model:page="page"
          :page-count="totalPages"
          @update:page="loadAlerts"
        />
      </div>
    </n-spin>
  </div>
</template>

<style scoped>
.alertsv2-page { max-width: 1400px; }

.status-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 10px 14px;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-hairline, var(--sre-border));
  border-radius: 10px;
  margin-bottom: 16px;
}
.search-box { width: 240px; }

.alert-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ar-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ar-headline {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
}
.ar-sev-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.6px;
}
.ar-title {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}
.ar-key {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  display: flex;
  align-items: center;
  gap: 6px;
}
.ar-label { color: var(--sre-text-tertiary); }
.ar-keyval {
  font-family: var(--sre-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 12px;
  background: var(--sre-bg-elevated);
  border-radius: 4px;
  padding: 2px 6px;
  color: var(--sre-text-secondary);
}
.ar-footer {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}
.ar-arrow {
  color: var(--sre-text-tertiary);
  flex-shrink: 0;
  align-self: center;
  opacity: 0.5;
  transition: opacity var(--sre-duration-fast, 0.15s) ease;
}
.sre-row-card:hover .ar-arrow { opacity: 1; }

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  background: var(--sre-bg-card);
  border: 1px dashed var(--sre-hairline, var(--sre-border));
  border-radius: 12px;
}
.empty-icon { color: var(--sre-text-tertiary); margin-bottom: 12px; opacity: 0.6; }
.empty-title { font-size: 15px; font-weight: 600; color: var(--sre-text-primary); margin-bottom: 6px; }
.empty-sub { font-size: 13px; color: var(--sre-text-tertiary); }

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
