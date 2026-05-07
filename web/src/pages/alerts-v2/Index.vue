<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, NTag, NButton } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { alertV2Api } from '@/api'
import type { AlertV2 } from '@/types'
import { formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import { RefreshOutline } from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

const loading = ref(false)
const alerts = ref<AlertV2[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const statusFilter = ref('')
const severityFilter = ref('')
const searchQuery = ref('')

const severityTagType: Record<string, 'error' | 'warning' | 'info' | 'default'> = {
  critical: 'error',
  warning: 'warning',
  info: 'info',
  p0: 'error', p1: 'error',
  p2: 'warning', p3: 'warning',
  p4: 'info',
}

async function loadAlerts() {
  loading.value = true
  try {
    const res = await alertV2Api.list({
      status: statusFilter.value,
      severity: severityFilter.value,
      query: searchQuery.value,
      page: page.value,
      page_size: pageSize.value,
    })
    alerts.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

const columns = computed(() => [
  {
    title: t('incident.severity'),
    key: 'severity',
    width: 90,
    render: (row: AlertV2) =>
      h(NTag, { type: severityTagType[row.severity] ?? 'default', size: 'small' },
        { default: () => row.severity.toUpperCase() }),
  },
    {
    title: t('common.name'),
    key: 'title',
    render: (row: AlertV2) =>
      h('a', {
        style: 'font-weight:500;cursor:pointer;color:var(--sre-primary)',
        onClick: () => router.push(`/alerts-v2/${row.id}`),
      }, row.title),
  },
  {
    title: t('common.status'),
    key: 'status',
    width: 100,
    render: (row: AlertV2) =>
      h(NTag, { type: row.status === 'firing' ? 'error' : 'success', size: 'small' },
        { default: () => row.status === 'firing' ? 'Firing' : 'Resolved' }),
  },
  {
    title: t('alertV2.linkedChannel'),
    key: 'channel',
    render: (row: AlertV2) =>
      row.channel
        ? h('a', {
            style: 'cursor:pointer;color:var(--sre-primary)',
            onClick: () => router.push(`/channels/${row.channel_id}`),
          }, row.channel.name)
        : h('span', {}, '—'),
  },
  {
    title: t('alertV2.linkedIncident'),
    key: 'incident',
    render: (row: AlertV2) =>
      row.incident
        ? h('a', {
            style: 'cursor:pointer;color:var(--sre-primary)',
            onClick: () => router.push(`/incidents/${row.incident_id}`),
          }, `#${row.incident_id} ${row.incident.title}`)
        : h('span', {}, '—'),
  },
  {
    title: t('alertV2.fireCount'),
    key: 'fire_count',
    width: 90,
    render: (row: AlertV2) => h('span', {}, String(row.fire_count)),
  },
  {
    title: t('alertV2.firstFiredAt'),
    key: 'first_fired_at',
    render: (row: AlertV2) => h('span', { style: 'font-size:12px' }, formatTime(row.first_fired_at)),
  },
  {
    title: t('alertV2.lastFiredAt'),
    key: 'last_fired_at',
    render: (row: AlertV2) => h('span', { style: 'font-size:12px' }, formatTime(row.last_fired_at)),
  },
])

onMounted(loadAlerts)
</script>

<template>
  <div class="alertsv2-page">
    <PageHeader :title="t('alertV2.title')" :subtitle="t('alertV2.subtitle')">
      <template #actions>
        <n-button circle quaternary @click="loadAlerts">
          <template #icon><n-icon :component="RefreshOutline" /></template>
        </n-button>
      </template>
    </PageHeader>

    <!-- Filters -->
    <n-card :bordered="false" class="filter-card">
      <n-space wrap>
        <n-select
          v-model:value="statusFilter"
          :options="[
            { label: 'Firing', value: 'firing' },
            { label: 'Resolved', value: 'resolved' },
          ]"
          :placeholder="t('common.status')"
          clearable
          style="width: 130px"
          @update:value="loadAlerts"
        />
        <n-select
          v-model:value="severityFilter"
          :options="[
            { label: 'Critical', value: 'critical' },
            { label: 'Warning', value: 'warning' },
            { label: 'Info', value: 'info' },
          ]"
          :placeholder="t('incident.severity')"
          clearable
          style="width: 120px"
          @update:value="loadAlerts"
        />
        <n-input
          v-model:value="searchQuery"
          :placeholder="t('common.search')"
          clearable
          style="width: 220px"
          @update:value="loadAlerts"
        />
      </n-space>
    </n-card>

    <n-card :bordered="false" class="table-card">
      <n-data-table
        :loading="loading"
        :columns="columns"
        :data="alerts"
        :row-key="(row: AlertV2) => row.id"
      />
      <div v-if="total > pageSize" style="display:flex;justify-content:flex-end;margin-top:12px">
        <n-pagination
          v-model:page="page"
          :page-count="Math.ceil(total / pageSize)"
          @update:page="loadAlerts"
        />
      </div>
    </n-card>
  </div>
</template>

<style scoped>
.alertsv2-page { max-width: 1400px; }
.filter-card { border-radius: 12px; margin-bottom: 16px; }
.table-card { border-radius: 12px; }
</style>
