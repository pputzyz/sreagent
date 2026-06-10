<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  NButton, NIcon, NSwitch, NDataTable, NCard, NSpace, NTag,
  NModal, NForm, NFormItem, NInput, NSelect, NPopconfirm,
  useMessage, useDialog,
} from 'naive-ui'
import {
  AddOutline, PlayOutline, TrashOutline, CreateOutline,
  RefreshOutline, TimeOutline, CopyOutline,
} from '@vicons/ionicons5'
import { inspectionApi } from '@/api/inspection'
import type { InspectionTask, InspectionRun } from '@/api/inspection'
import { getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import CronInput from '@/components/common/CronInput.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const router = useRouter()

const loading = ref(false)
const tasks = ref<InspectionTask[]>([])
const showModal = ref(false)
const editingTask = ref<InspectionTask | null>(null)
const formLoading = ref(false)

const form = ref({
  name: '',
  description: '',
  cron_expr: '0 0 9 * * *',
  target_type: 'global',
  target_ids: '',
  allowed_tools: '',
  output_channels: '[{"type":"lark_bot"}]',
  enabled: true,
})

const targetTypeOptions = [
  { label: t('inspection.targetGlobal'), value: 'global' },
  { label: t('inspection.targetBizGroup'), value: 'biz_group' },
]

// FE7-9: Cron scheduling preview — show next 5 runs
const cronPreview = ref<string[]>([])
const cronPreviewLoading = ref(false)
let cronPreviewTimer: ReturnType<typeof setTimeout> | null = null

function debouncedCronPreview() {
  if (cronPreviewTimer) clearTimeout(cronPreviewTimer)
  cronPreviewTimer = setTimeout(fetchCronPreview, 500)
}

onBeforeUnmount(() => {
  if (cronPreviewTimer) clearTimeout(cronPreviewTimer)
})

async function fetchCronPreview() {
  const expr = form.value.cron_expr?.trim()
  if (!expr) { cronPreview.value = []; return }
  cronPreviewLoading.value = true
  try {
    const { data } = await inspectionApi.validateCron(expr)
    cronPreview.value = (data.data?.next_runs || []).slice(0, 5)
  } catch {
    cronPreview.value = []
  } finally {
    cronPreviewLoading.value = false
  }
}

watch(() => form.value.cron_expr, debouncedCronPreview)

// --- Table columns ---
const taskColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: t('inspection.taskName'), key: 'name', width: 180 },
  {
    title: 'Cron',
    key: 'cron_expr',
    width: 150,
    render: (row: InspectionTask) => h('div', { style: 'display:flex;align-items:center;gap:4px' }, [
      h(NIcon, { size: 14 }, { default: () => h(TimeOutline) }),
      h('code', { style: 'font-size:12px' }, row.cron_expr),
    ]),
  },
  {
    title: t('inspection.status'),
    key: 'enabled',
    width: 80,
    render: (row: InspectionTask) => h(NSwitch, {
      value: row.enabled,
      size: 'small',
      onUpdateValue: async (val: boolean) => {
        try {
          await inspectionApi.updateTask(row.id, { enabled: val })
          row.enabled = val
          message.success(val ? t('inspection.enabled') : t('inspection.disabled'))
        } catch (e) {
          message.error(getErrorMessage(e))
        }
      },
    }),
  },
  {
    title: t('inspection.actions'),
    key: 'actions',
    width: 200,
    render: (row: InspectionTask) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'small', type: 'primary', secondary: true, onClick: () => runTask(row) }, {
          icon: () => h(NIcon, null, { default: () => h(PlayOutline) }),
          default: () => t('inspection.run'),
        }),
        h(NButton, { size: 'small', secondary: true, onClick: () => openEdit(row) }, {
          icon: () => h(NIcon, null, { default: () => h(CreateOutline) }),
        }),
        h(NButton, { size: 'small', secondary: true, onClick: () => cloneTask(row), title: t('inspection.clone') || 'Clone' }, {
          icon: () => h(NIcon, null, { default: () => h(CopyOutline) }),
        }),
        h(NPopconfirm, { onPositiveClick: () => deleteTask(row) }, {
          trigger: () => h(NButton, { size: 'small', type: 'error', secondary: true }, {
            icon: () => h(NIcon, null, { default: () => h(TrashOutline) }),
          }),
          default: () => t('inspection.confirmDelete'),
        }),
      ],
    }),
  },
]

// --- API calls ---
async function fetchTasks() {
  loading.value = true
  try {
    const { data } = await inspectionApi.listTasks()
    tasks.value = data.data?.list || []
  } catch (e) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

async function runTask(task: InspectionTask) {
  try {
    await inspectionApi.runNow(task.id)
    message.success(t('inspection.submitted', { name: task.name }))
  } catch (e) {
    message.error(getErrorMessage(e))
  }
}

async function deleteTask(task: InspectionTask) {
  try {
    await inspectionApi.deleteTask(task.id)
    message.success(t('inspection.deleted'))
    await fetchTasks()
  } catch (e) {
    message.error(getErrorMessage(e))
  }
}

// FE7-7: Clone task
function cloneTask(task: InspectionTask) {
  editingTask.value = null
  form.value = {
    name: task.name + ' (copy)',
    description: task.description,
    cron_expr: task.cron_expr,
    target_type: task.target_type,
    target_ids: task.target_ids || '',
    allowed_tools: task.allowed_tools || '',
    output_channels: task.output_channels || '[{"type":"lark_bot"}]',
    enabled: false,
  }
  cronPreview.value = []
  showModal.value = true
  fetchCronPreview()
}

function openCreate() {
  editingTask.value = null
  form.value = {
    name: '',
    description: '',
    cron_expr: '0 0 9 * * *',
    target_type: 'global',
    target_ids: '',
    allowed_tools: '',
    output_channels: '[{"type":"lark_bot"}]',
    enabled: true,
  }
  cronPreview.value = []
  showModal.value = true
  fetchCronPreview()
}

function openEdit(task: InspectionTask) {
  editingTask.value = task
  form.value = {
    name: task.name,
    description: task.description,
    cron_expr: task.cron_expr,
    target_type: task.target_type,
    target_ids: task.target_ids || '',
    allowed_tools: task.allowed_tools || '',
    output_channels: task.output_channels || '[{"type":"lark_bot"}]',
    enabled: task.enabled,
  }
  cronPreview.value = []
  showModal.value = true
  fetchCronPreview()
}

async function handleSubmit() {
  if (!form.value.name.trim()) {
    message.warning(t('inspection.enterName'))
    return
  }
  if (!form.value.description.trim()) {
    message.warning(t('inspection.enterDesc'))
    return
  }
  formLoading.value = true
  try {
    if (editingTask.value) {
      await inspectionApi.updateTask(editingTask.value.id, form.value)
      message.success(t('inspection.updated'))
    } else {
      await inspectionApi.createTask(form.value)
      message.success(t('inspection.created'))
    }
    showModal.value = false
    await fetchTasks()
  } catch (e) {
    message.error(getErrorMessage(e))
  } finally {
    formLoading.value = false
  }
}

// --- Runs ---
const runs = ref<InspectionRun[]>([])
const runsLoading = ref(false)

async function fetchRuns() {
  runsLoading.value = true
  try {
    const { data } = await inspectionApi.listRuns({ page: 1, page_size: 50 })
    runs.value = data.data?.list || []
  } catch (e) {
    message.error(getErrorMessage(e))
  } finally {
    runsLoading.value = false
  }
}

function viewRun(run: InspectionRun) {
  router.push(`/platform/inspections/runs/${run.id}`)
}

const runColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: t('inspection.taskId'), key: 'task_id', width: 80 },
  {
    title: t('inspection.status'),
    key: 'status',
    width: 80,
    render: (row: InspectionRun) => h(NTag, {
      type: row.status === 'success' ? 'success' : row.status === 'failed' ? 'error' : 'info',
      size: 'small',
    }, { default: () => row.status }),
  },
  { title: t('inspection.summary'), key: 'report_summary', ellipsis: { tooltip: true } },
  {
    title: t('inspection.startTime'),
    key: 'started_at',
    width: 170,
    render: (row: InspectionRun) => new Date(row.started_at).toLocaleString(),
  },
  {
    title: t('inspection.actions'),
    key: 'actions',
    width: 80,
    render: (row: InspectionRun) => h(NButton, {
      size: 'small',
      text: true,
      type: 'primary',
      onClick: () => viewRun(row),
    }, { default: () => t('inspection.view') }),
  },
]

onMounted(() => {
  fetchTasks()
  fetchRuns()
})
</script>

<template>
  <div style="padding: 16px; display: flex; flex-direction: column; gap: 16px;">
    <PageHeader :title="t('inspection.title')" :description="t('inspection.subtitle')">
      <template #actions>
        <NButton @click="fetchTasks">
          <template #icon><NIcon><RefreshOutline /></NIcon></template>
          {{ t('inspection.refresh') }}
        </NButton>
        <NButton type="primary" @click="openCreate">
          <template #icon><NIcon><AddOutline /></NIcon></template>
          {{ t('inspection.createTask') }}
        </NButton>
      </template>
    </PageHeader>

    <!-- Tasks table -->
    <NCard :title="t('inspection.taskList')" size="small">
      <NDataTable
        v-if="tasks.length > 0"
        :columns="taskColumns"
        :data="tasks"
        :loading="loading"
        :bordered="false"
        size="small"
      />
      <EmptyState v-else-if="!loading" :title="t('inspection.noTasks')" :description="t('inspection.noTasksHint')" />
    </NCard>

    <!-- Recent runs -->
    <NCard :title="t('inspection.recentRuns')" size="small">
      <NDataTable
        :columns="runColumns"
        :data="runs"
        :loading="runsLoading"
        :bordered="false"
        size="small"
        :pagination="{ pageSize: 10 }"
      />
    </NCard>

    <!-- Create/Edit modal -->
    <NModal
      v-model:show="showModal"
      preset="card"
      :title="editingTask ? t('inspection.editTask') : t('inspection.newTask')"
      style="width: 600px"
      :bordered="false"
    >
      <NForm label-placement="left" label-width="100">
        <NFormItem :label="t('inspection.taskName')" required>
          <NInput v-model:value="form.name" :placeholder="t('inspection.taskNamePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('inspection.taskDesc')" required>
          <NInput
            v-model:value="form.description"
            type="textarea"
            :rows="4"
            :placeholder="t('inspection.taskDescPlaceholder')"
          />
        </NFormItem>
        <NFormItem :label="t('inspection.cronRule')" required>
          <CronInput v-model="form.cron_expr" />
        </NFormItem>
        <!-- FE7-9: Cron scheduling preview -->
        <NFormItem v-if="cronPreview.length > 0 || cronPreviewLoading" :label="t('inspection.nextRuns') || 'Next runs'">
          <div v-if="cronPreviewLoading" style="font-size: 12px; color: var(--sre-text-tertiary);">
            {{ t('common.loading') }}
          </div>
          <div v-else style="display: flex; flex-direction: column; gap: 4px;">
            <div
              v-for="(run, idx) in cronPreview"
              :key="idx"
              style="display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--sre-text-secondary);"
            >
              <NIcon :component="TimeOutline" :size="12" />
              <span>{{ new Date(run).toLocaleString() }}</span>
            </div>
          </div>
        </NFormItem>
        <NFormItem :label="t('inspection.targetType')">
          <NSelect v-model:value="form.target_type" :options="targetTypeOptions" />
        </NFormItem>
        <NFormItem v-if="form.target_type === 'biz_group'" :label="t('inspection.targetIds')">
          <NInput v-model:value="form.target_ids" :placeholder="t('inspection.targetIdsPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('inspection.toolWhitelist')">
          <NInput
            v-model:value="form.allowed_tools"
            :placeholder="t('inspection.toolWhitelistPlaceholder')"
          />
        </NFormItem>
        <NFormItem :label="t('inspection.outputChannel')">
          <NInput
            v-model:value="form.output_channels"
            type="textarea"
            :rows="2"
            placeholder='[{"type":"lark_bot"}]'
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="formLoading" @click="handleSubmit">
            {{ editingTask ? t('inspection.update') : t('inspection.create') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
