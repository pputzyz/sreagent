<script setup lang="ts">
import { onMounted, computed, shallowRef, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { alertV2Api, channelV2Api } from '@/api'
import type { AlertV2, Channel } from '@/types'
import { usePaginatedList } from '@/composables'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import {
  RefreshOutline,
  SearchOutline,
  ChevronForwardOutline,
  ShieldCheckmarkOutline,
} from '@vicons/ionicons5'
import { relTime, getErrorMessage } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

const channels = shallowRef<Channel[]>([])
const statusFilter = ref<string>('')
const severityFilter = ref<string>('')
const channelFilter = ref<number | null>(null)
const searchQuery = ref('')
const firstLoaded = ref(false)

const {
  loading,
  items: alerts,
  total,
  page,
  pageSize,
  fetchList,
  refresh,
} = usePaginatedList<AlertV2>({
  apiFn: alertV2Api.list,
  extraParams: () => {
    const params: Record<string, unknown> = {}
    if (statusFilter.value) params.status = statusFilter.value
    if (severityFilter.value) params.severity = severityFilter.value
    if (channelFilter.value != null) params.channel_id = channelFilter.value
    if (searchQuery.value) params.query = searchQuery.value
    return params
  },
  onError: (err: unknown) => {
    message.error((err as Error)?.message ?? t('common.loadFailed'))
  },
})

watch(loading, (isLoading) => {
  if (!isLoading) firstLoaded.value = true
})

async function loadChannels() {
  try {
    const res = await channelV2Api.list({ status: 'active', page: 1, page_size: 100 })
    channels.value = res.data.data?.list ?? []
  } catch { /* silent */ }
}

const channelOptions = computed(() => [
  { label: t('common.all'), value: undefined as number | undefined },
  ...channels.value.map(c => ({ label: c.name, value: c.id })),
])

const severityOptions = [
  { label: () => t('alert.critical'), value: 'critical' },
  { label: () => t('alert.warning'), value: 'warning' },
  { label: () => t('alert.info'), value: 'info' },
]

function goDetail(a: AlertV2) {
  router.push(`/alerts-v2/${a.id}`)
}

const isEmpty = computed(() => firstLoaded.value && !loading.value && alerts.value.length === 0)
const hasFilters = computed(() =>
  statusFilter.value !== '' || severityFilter.value !== '' || channelFilter.value !== null || searchQuery.value.trim() !== ''
)

onMounted(() => { fetchList(); loadChannels() })
</script>

<template>
  <div class="alertsv2-page">
    <PageHeader
      :title="t('alertV2.title')"
      :subtitle="t('alertV2.subtitle')"
    >
      <template #actions>
        <n-button circle quaternary @click="fetchList" :aria-label="t('common.refresh')">
          <template #icon><n-icon :component="RefreshOutline" /></template>
        </n-button>
      </template>
    </PageHeader>

    <!-- Status segmented -->
    <div class="status-row">
      <span class="sre-label-eyebrow">{{ t('alertV2.status') }}</span>
      <n-radio-group
        v-model:value="statusFilter"
        size="medium"
        @update:value="fetchList"
      >
        <n-radio-button value="">{{ t('common.all') }}</n-radio-button>
        <n-radio-button value="firing">{{ t('alertV2.firing') }}</n-radio-button>
        <n-radio-button value="resolved">{{ t('alertV2.resolved') }}</n-radio-button>
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
        @update:value="fetchList"
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
        @update:value="fetchList"
      />
      <n-select
        v-model:value="channelFilter"
        :options="channelOptions"
        :placeholder="t('alertV2.channel')"
        clearable
        size="small"
        style="width: 200px"
        @update:value="fetchList"
      />
    </div>

    <!-- List -->
    <LoadingSkeleton v-if="loading && alerts.length === 0" :rows="6" variant="row" />
    <n-spin v-else :show="loading && alerts.length > 0">
      <EmptyState
        v-if="isEmpty"
        :icon="ShieldCheckmarkOutline"
        :title="t('alertV2.noActiveAlerts')"
        :description="hasFilters ? t('alertV2.emptyFiltered') : t('alertV2.noActiveAlertsDesc')"
      />

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
              <span class="ar-title">{{ alert.title }}</span>
              <span class="ar-status" :data-status="alert.status">{{ alert.status === 'firing' ? t('alertV2.firing') : t('alertV2.resolved') }}</span>
              <span class="ar-count tnum">{{ alert.event_count }}</span>
            </div>
            <div class="ar-footer">
              <code class="ar-keyval">{{ alert.alert_key }}</code>
              <template v-if="alert.channel">
                <span class="sre-meta-divider"></span>
                <span>{{ alert.channel.name }}</span>
              </template>
              <span class="sre-meta-divider"></span>
              <span>{{ relTime(alert.last_fired_at, t) }}</span>
              <template v-if="alert.incident_id">
                <span class="sre-meta-divider"></span>
                <span>{{ t('alertV2.linkedIncident') }} #{{ alert.incident_id }}</span>
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
          :page-size="pageSize"
          :item-count="total"
          :page-slot="7"
          @update:page="fetchList"
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
  border-radius: var(--sre-radius-lg);
  margin-bottom: 16px;
}
.search-box { width: 240px; }

.alert-list {
  display: flex;
  flex-direction: column;
  gap: var(--sre-row-gap);
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
.ar-title {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}
.ar-status {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 7px;
  border-radius: var(--sre-radius-pill);
  white-space: nowrap;
  flex-shrink: 0;
}
.ar-status[data-status="firing"] {
  background: var(--sre-critical-soft);
  color: var(--sre-critical);
}
.ar-status[data-status="resolved"] {
  background: var(--sre-success-soft, rgba(34,197,94,0.12));
  color: var(--sre-success, #22c55e);
}
.ar-count {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  flex-shrink: 0;
  min-width: 20px;
  text-align: right;
}
.ar-keyval {
  font-family: var(--sre-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 12px;
  background: var(--sre-bg-elevated);
  border-radius: var(--sre-radius-xs);
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
  border-radius: var(--sre-radius-md);
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
