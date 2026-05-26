<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  NButton, NIcon, NTag, NDataTable, NPagination,
  NEmpty, NSelect, NSpace,
} from 'naive-ui'
import { SearchOutline, RefreshOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import { taskApi, type TaskRecord, getTaskStatusLabel, getTaskStatusType, TaskStatus } from '@/api/task'
import { getErrorMessage, formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

// --- State ---
const records = ref<TaskRecord[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const statusFilter = ref<number>(-1)

// --- Options ---
const statusOptions = computed(() => [
  { label: t('common.all'), value: -1 },
  { label: t('task.statusPending'), value: TaskStatus.Pending },
  { label: t('task.statusRunning'), value: TaskStatus.Running },
  { label: t('task.statusSuccess'), value: TaskStatus.Success },
  { label: t('task.statusFailed'), value: TaskStatus.Fail },
])

// --- API ---
async function fetchRecords() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    if (statusFilter.value >= 0) {
      params.status = statusFilter.value
    }
    const resp = await taskApi.list(params)
    records.value = resp.data.data?.list || []
    total.value = resp.data.data?.total || 0
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

function handlePageChange(p: number) {
  page.value = p
  fetchRecords()
}

function handlePageSizeChange(ps: number) {
  pageSize.value = ps
  page.value = 1
  fetchRecords()
}

function handleRefresh() {
  fetchRecords()
}

// --- Helpers ---
function parseHosts(hostsStr: string): string[] {
  try {
    const arr = JSON.parse(hostsStr || '[]')
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
}

// --- Columns ---
const columns = computed<DataTableColumns<TaskRecord>>(() => [
  {
    title: 'ID',
    key: 'id',
    width: 70,
    render: (row) =>
      h('a', {
        style: 'color: var(--sre-primary); cursor: pointer; text-decoration: none;',
        onClick: () => router.push(`/platform/tasks/${row.id}`),
      }, `#${row.id}`),
  },
  {
    title: t('task.taskTitle'),
    key: 'title',
    minWidth: 200,
    ellipsis: { tooltip: true },
    render: (row) =>
      h('a', {
        style: 'color: var(--sre-primary); cursor: pointer; text-decoration: none;',
        onClick: () => router.push(`/platform/tasks/${row.id}`),
      }, row.title || '-'),
  },
  {
    title: t('task.status'),
    key: 'status',
    width: 100,
    render: (row) =>
      h(NTag, {
        size: 'small',
        type: getTaskStatusType(row.status),
        bordered: false,
      }, () => t(`task.status${getTaskStatusLabel(row.status).charAt(0).toUpperCase() + getTaskStatusLabel(row.status).slice(1)}`)),
  },
  {
    title: t('task.hostCount'),
    key: 'hosts',
    width: 100,
    render: (row) => {
      const hosts = parseHosts(row.hosts)
      return h('span', {}, `${hosts.length} ${t('task.hosts')}`)
    },
  },
  {
    title: t('task.createdBy'),
    key: 'create_by',
    width: 120,
    ellipsis: { tooltip: true },
    render: (row) => row.create_by || '-',
  },
  {
    title: t('task.createdAt'),
    key: 'created_at',
    width: 170,
    render: (row) => formatTime(row.created_at),
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 100,
    fixed: 'right',
    render: (row) =>
      h(NButton, {
        size: 'tiny',
        quaternary: true,
        type: 'primary',
        onClick: () => router.push(`/platform/tasks/${row.id}`),
      }, () => t('task.viewDetail')),
  },
])

// --- Init ---
onMounted(fetchRecords)
</script>

<template>
  <div class="task-index-page">
    <PageHeader :title="t('task.title')" :subtitle="t('task.subtitle')">
      <template #actions>
        <n-button size="small" @click="handleRefresh">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          {{ t('common.refresh') }}
        </n-button>
      </template>
    </PageHeader>

    <div class="toolbar">
      <n-select
        v-model:value="statusFilter"
        :options="statusOptions"
        :placeholder="t('task.statusFilter')"
        clearable
        style="width: 180px"
        size="small"
        @update:value="fetchRecords"
      />
      <span class="count tnum">{{ total }} {{ t('task.records') }}</span>
    </div>

    <n-empty v-if="!loading && records.length === 0" :description="t('task.noData')" style="padding: 60px 0" />

    <template v-else>
      <n-data-table
        :columns="columns"
        :data="records"
        :loading="loading"
        :row-key="(row: TaskRecord) => row.id"
        size="small"
        :bordered="false"
        striped
        :scroll-x="800"
      />

      <div class="page-pagination" v-if="total > 0">
        <n-pagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :item-count="total"
          :page-sizes="[20, 50, 100]"
          show-size-picker
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </template>
  </div>
</template>

<style scoped>
.task-index-page {
  padding: 16px;
  max-width: 1400px;
}
.toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}
.count {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin-left: auto;
  font-variant-numeric: tabular-nums;
}
.page-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
