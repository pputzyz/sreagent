<script setup lang="ts">
import { ref, onMounted, computed, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import {
  NButton, NIcon, NTag, NCard, NDataTable, NSpin,
  NDescriptions, NDescriptionsItem, NSpace, NCollapse, NCollapseItem,
  NEmpty, NTimeline, NTimelineItem,
} from 'naive-ui'
import {
  ArrowBackOutline, RefreshOutline, CheckmarkCircleOutline,
  CloseCircleOutline, TimeOutline, EllipseOutline,
} from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import { taskApi, type TaskRecord, type TaskHostRecord, getTaskStatusLabel, getTaskStatusType, TaskStatus } from '@/api/task'
import { getErrorMessage, formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const message = useMessage()

const loading = ref(true)
const record = ref<TaskRecord | null>(null)
const hostRecords = ref<TaskHostRecord[]>([])
const hostsLoading = ref(false)

// --- Computed ---
const taskId = computed(() => Number(route.params.id))

const statusType = computed(() => {
  if (!record.value) return 'info'
  return getTaskStatusType(record.value.status)
})

const statusLabel = computed(() => {
  if (!record.value) return ''
  const label = getTaskStatusLabel(record.value.status)
  return t(`task.status${label.charAt(0).toUpperCase() + label.slice(1)}`)
})

const parsedHosts = computed<string[]>(() => {
  if (!record.value) return []
  try {
    const arr = JSON.parse(record.value.hosts || '[]')
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
})

const statusCounts = computed(() => {
  const counts = { total: hostRecords.value.length, success: 0, failed: 0, running: 0, pending: 0 }
  for (const hr of hostRecords.value) {
    if (hr.status === TaskStatus.Success) counts.success++
    else if (hr.status === TaskStatus.Fail) counts.failed++
    else if (hr.status === TaskStatus.Running) counts.running++
    else counts.pending++
  }
  return counts
})

// --- API ---
async function fetchRecord() {
  loading.value = true
  try {
    const resp = await taskApi.get(taskId.value)
    record.value = resp.data.data || null
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

async function fetchHostRecords() {
  hostsLoading.value = true
  try {
    const resp = await taskApi.getHosts(taskId.value)
    hostRecords.value = resp.data.data || []
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    hostsLoading.value = false
  }
}

function handleRefresh() {
  fetchRecord()
  fetchHostRecords()
}

// --- Host status icon ---
function getStatusIcon(status: number) {
  switch (status) {
    case TaskStatus.Success: return CheckmarkCircleOutline
    case TaskStatus.Fail: return CloseCircleOutline
    case TaskStatus.Running: return TimeOutline
    default: return EllipseOutline
  }
}

function getStatusColor(status: number): string {
  switch (status) {
    case TaskStatus.Success: return 'var(--n-success-color, #18a058)'
    case TaskStatus.Fail: return 'var(--n-error-color, #d03050)'
    case TaskStatus.Running: return 'var(--n-info-color, #2080f0)'
    default: return 'var(--n-text-color-3, #999)'
  }
}

// --- Host columns ---
const hostColumns = computed<DataTableColumns<TaskHostRecord>>(() => [
  {
    title: t('task.host'),
    key: 'host',
    minWidth: 180,
    ellipsis: { tooltip: true },
    render: (row) =>
      h('div', { style: 'display: flex; align-items: center; gap: 6px;' }, [
        h(NIcon, { size: 16, color: getStatusColor(row.status) }, { default: () => h(getStatusIcon(row.status)) }),
        h('span', {}, row.host),
      ]),
  },
  {
    title: t('task.hostStatus'),
    key: 'status',
    width: 100,
    render: (row) =>
      h(NTag, {
        size: 'small',
        type: getTaskStatusType(row.status),
        bordered: false,
      }, () => {
        const label = getTaskStatusLabel(row.status)
        return t(`task.status${label.charAt(0).toUpperCase() + label.slice(1)}`)
      }),
  },
  {
    title: t('task.duration'),
    key: 'duration_ms',
    width: 100,
    render: (row) => {
      if (!row.duration_ms) return '-'
      const ms = row.duration_ms
      if (ms < 1000) return `${ms}ms`
      return `${(ms / 1000).toFixed(1)}s`
    },
  },
  {
    title: t('task.exitCode'),
    key: 'exit_code',
    width: 80,
    render: (row) => {
      const color = row.exit_code === 0 ? 'var(--n-success-color, #18a058)' : 'var(--n-error-color, #d03050)'
      return h('span', { style: `color: ${color}; font-family: monospace; font-weight: 600;` }, String(row.exit_code))
    },
  },
])

// --- Init ---
onMounted(() => {
  if (!taskId.value) {
    message.error(t('task.invalidId'))
    return
  }
  fetchRecord()
  fetchHostRecords()
})
</script>

<template>
  <div class="task-result-page">
    <!-- Header -->
    <div class="result-header">
      <NButton text @click="router.push('/platform/tasks')">
        <template #icon><NIcon><ArrowBackOutline /></NIcon></template>
      </NButton>
      <h2 style="margin: 0; font-size: 18px;">
        {{ t('task.resultTitle', { id: taskId }) }}
      </h2>
      <NTag v-if="record" :type="statusType" size="small">{{ statusLabel }}</NTag>
      <div style="margin-left: auto;">
        <NButton size="small" @click="handleRefresh">
          <template #icon><NIcon :component="RefreshOutline" /></template>
          {{ t('common.refresh') }}
        </NButton>
      </div>
    </div>

    <NSpin :show="loading">
      <template v-if="record">
        <!-- Task Metadata -->
        <NCard size="small" :title="t('task.metadata')" style="margin-bottom: 16px;">
          <NDescriptions :column="2" label-placement="left" bordered size="small">
            <NDescriptionsItem :label="t('task.taskTitle')">
              {{ record.title }}
            </NDescriptionsItem>
            <NDescriptionsItem :label="t('task.status')">
              <NTag :type="statusType" size="small" bordered>{{ statusLabel }}</NTag>
            </NDescriptionsItem>
            <NDescriptionsItem :label="t('task.createdBy')">
              {{ record.create_by || '-' }}
            </NDescriptionsItem>
            <NDescriptionsItem :label="t('task.createdAt')">
              {{ formatTime(record.created_at) }}
            </NDescriptionsItem>
            <NDescriptionsItem :label="t('task.account')">
              {{ record.account || '-' }}
            </NDescriptionsItem>
            <NDescriptionsItem :label="t('task.timeout')">
              {{ record.timeout }}s
            </NDescriptionsItem>
            <NDescriptionsItem :label="t('task.batch')">
              {{ record.batch === 0 ? t('taskTpl.allAtOnce') : record.batch }}
            </NDescriptionsItem>
            <NDescriptionsItem :label="t('task.tolerance')">
              {{ record.tolerance }}
            </NDescriptionsItem>
          </NDescriptions>
        </NCard>

        <!-- Host Summary -->
        <NCard size="small" :title="t('task.hostSummary')" style="margin-bottom: 16px;">
          <div style="display: flex; gap: 24px; align-items: center;">
            <div class="stat-item">
              <span class="stat-value">{{ statusCounts.total }}</span>
              <span class="stat-label">{{ t('task.totalHosts') }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-value" style="color: var(--n-success-color, #18a058);">{{ statusCounts.success }}</span>
              <span class="stat-label">{{ t('task.statusSuccess') }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-value" style="color: var(--n-error-color, #d03050);">{{ statusCounts.failed }}</span>
              <span class="stat-label">{{ t('task.statusFailed') }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-value" style="color: var(--n-info-color, #2080f0);">{{ statusCounts.running }}</span>
              <span class="stat-label">{{ t('task.statusRunning') }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-value" style="color: var(--n-text-color-3, #999);">{{ statusCounts.pending }}</span>
              <span class="stat-label">{{ t('task.statusPending') }}</span>
            </div>
          </div>
        </NCard>

        <!-- Script -->
        <NCard size="small" :title="t('task.script')" style="margin-bottom: 16px;" v-if="record.script">
          <pre class="script-block">{{ record.script }}</pre>
          <div v-if="record.args" style="margin-top: 8px;">
            <strong>{{ t('task.args') }}:</strong> <code>{{ record.args }}</code>
          </div>
        </NCard>

        <!-- Per-Host Results -->
        <NCard size="small" :title="t('task.hostResults')">
          <NSpin :show="hostsLoading">
            <NEmpty v-if="!hostsLoading && hostRecords.length === 0" :description="t('task.noHostResults')" />

            <template v-else>
              <NDataTable
                :columns="hostColumns"
                :data="hostRecords"
                :row-key="(row: TaskHostRecord) => row.id"
                size="small"
                :bordered="false"
                striped
              />

              <!-- Expandable stdout/stderr for each host -->
              <div v-if="hostRecords.length > 0" style="margin-top: 16px;">
                <NCollapse>
                  <NCollapseItem
                    v-for="hr in hostRecords"
                    :key="hr.id"
                    :name="hr.id"
                    :title="`${hr.host} — ${getTaskStatusLabel(hr.status)}`"
                  >
                    <div class="host-output">
                      <div v-if="hr.stdout" class="output-section">
                        <h4>stdout</h4>
                        <pre class="output-block stdout">{{ hr.stdout }}</pre>
                      </div>
                      <div v-if="hr.stderr" class="output-section">
                        <h4>stderr</h4>
                        <pre class="output-block stderr">{{ hr.stderr }}</pre>
                      </div>
                      <div v-if="!hr.stdout && !hr.stderr" class="output-section">
                        <NEmpty :description="t('task.noOutput')" size="small" />
                      </div>
                    </div>
                  </NCollapseItem>
                </NCollapse>
              </div>
            </template>
          </NSpin>
        </NCard>
      </template>

      <NEmpty v-else-if="!loading" :description="t('task.noData')" style="padding: 60px 0" />
    </NSpin>
  </div>
</template>

<style scoped>
.task-result-page {
  padding: 16px;
  max-width: 1400px;
}
.result-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}
.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}
.stat-value {
  font-size: 24px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}
.stat-label {
  font-size: 12px;
  color: var(--sre-text-secondary);
}
.script-block {
  background: var(--n-color);
  border: 1px solid var(--n-border-color);
  border-radius: 4px;
  padding: 12px;
  font-family: monospace;
  font-size: 13px;
  line-height: 1.5;
  overflow-x: auto;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}
.host-output {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.output-section h4 {
  margin: 0 0 4px 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-secondary);
}
.output-block {
  background: var(--n-color);
  border: 1px solid var(--n-border-color);
  border-radius: 4px;
  padding: 12px;
  font-family: monospace;
  font-size: 12px;
  line-height: 1.5;
  overflow-x: auto;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 400px;
  overflow-y: auto;
}
.output-block.stderr {
  border-left: 3px solid var(--n-error-color, #d03050);
}
.output-block.stdout {
  border-left: 3px solid var(--n-success-color, #18a058);
}
</style>
