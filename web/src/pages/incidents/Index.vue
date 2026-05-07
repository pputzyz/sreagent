<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, NTag, NButton, NSpace, NPopconfirm } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { incidentApi } from '@/api'
import type { Incident } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import { RefreshOutline, AddOutline } from '@vicons/ionicons5'

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
const statusFilter = ref('')
const severityFilter = ref('')
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

const severityTagType: Record<string, 'error' | 'warning' | 'info' | 'default'> = {
  critical: 'error',
  warning: 'warning',
  info: 'info',
}

const statusTagType: Record<string, 'error' | 'warning' | 'success' | 'default'> = {
  triggered: 'error',
  processing: 'warning',
  closed: 'success',
}

const statusLabel: Record<string, string> = {
  triggered: 'incident.statusTriggered',
  processing: 'incident.statusProcessing',
  closed: 'incident.statusClosed',
}

const severityLabel: Record<string, string> = {
  critical: 'incident.severityCritical',
  warning: 'incident.severityWarning',
  info: 'incident.severityInfo',
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

async function acknowledgeIncident(id: number) {
  try {
    await incidentApi.acknowledge(id)
    message.success(t('common.success'))
    await loadIncidents()
  } catch (e: any) {
    message.error(e?.message ?? t('common.failed'))
  }
}

async function closeIncident(id: number) {
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

const columns = computed(() => [
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
    width: 100,
    render: (row: Incident) =>
      h(NTag, { type: statusTagType[row.status] ?? 'default', size: 'small' },
        { default: () => t(statusLabel[row.status] ?? row.status) }),
  },
  {
    title: t('incident.channel'),
    key: 'channel',
    render: (row: Incident) => h('span', {}, row.channel?.name ?? '—'),
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
    width: 90,
    render: (row: Incident) => h('span', {}, String(row.alert_count)),
  },
  {
    title: t('incident.triggeredAt'),
    key: 'triggered_at',
    render: (row: Incident) => h('span', { style: 'font-size:12px' }, formatTime(row.triggered_at)),
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 160,
    render: (row: Incident) =>
      h(NSpace, { size: 'small' }, {
        default: () => [
          row.status !== 'closed' && row.status !== 'processing'
            ? h(NButton, {
                size: 'tiny', type: 'primary',
                onClick: (e: Event) => { e.stopPropagation(); acknowledgeIncident(row.id) },
              }, { default: () => t('incident.acknowledge') })
            : null,
          row.status !== 'closed'
            ? h(NButton, {
                size: 'tiny', type: 'default',
                onClick: (e: Event) => { e.stopPropagation(); closeIncident(row.id) },
              }, { default: () => t('incident.close') })
            : null,
        ].filter(Boolean),
      }),
  },
])

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

    <!-- View toggle + filters -->
    <n-card :bordered="false" class="filter-card">
      <n-space wrap>
        <n-radio-group v-model:value="viewMode" @update:value="loadIncidents">
          <n-radio-button value="all">{{ t('incident.allIncidents') }}</n-radio-button>
          <n-radio-button value="mine">{{ t('incident.myIncidents') }}</n-radio-button>
        </n-radio-group>

        <n-select
          v-model:value="statusFilter"
          :options="[
            { label: t('incident.statusTriggered'), value: 'triggered' },
            { label: t('incident.statusProcessing'), value: 'processing' },
            { label: t('incident.statusClosed'), value: 'closed' },
          ]"
          :placeholder="t('common.status')"
          clearable
          style="width: 130px"
          @update:value="loadIncidents"
        />
        <n-select
          v-model:value="severityFilter"
          :options="[
            { label: t('incident.severityCritical'), value: 'critical' },
            { label: t('incident.severityWarning'), value: 'warning' },
            { label: t('incident.severityInfo'), value: 'info' },
          ]"
          :placeholder="t('incident.severity')"
          clearable
          style="width: 120px"
          @update:value="loadIncidents"
        />
        <n-input
          v-model:value="searchQuery"
          :placeholder="t('common.search')"
          clearable
          style="width: 220px"
          @update:value="loadIncidents"
        />
      </n-space>
    </n-card>

    <n-card :bordered="false" class="table-card">
      <n-data-table
        :loading="loading"
        :columns="columns"
        :data="incidents"
        :row-key="(row: Incident) => row.id"
        :row-props="(row: Incident) => ({ style: 'cursor:pointer', onClick: () => $router.push(`/incidents/${row.id}`) })"
      />
      <div v-if="total > pageSize" style="display:flex;justify-content:flex-end;margin-top:12px">
        <n-pagination
          v-model:page="page"
          :page-count="Math.ceil(total / pageSize)"
          @update:page="loadIncidents"
        />
      </div>
    </n-card>

    <!-- Create Modal -->
    <n-modal
      v-model:show="showCreateModal"
      :title="t('incident.create')"
      preset="card"
      style="width: 440px"
      :bordered="false"
    >
      <n-form label-placement="top" size="small">
        <n-form-item :label="t('incident.name')" required>
          <n-input v-model:value="createForm.title" :placeholder="t('incident.name')" />
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
.incidents-page { max-width: 1400px; }
.filter-card { border-radius: 12px; margin-bottom: 16px; }
.table-card { border-radius: 12px; }
</style>
