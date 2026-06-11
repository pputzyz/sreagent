<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import {
  NButton, NDataTable, NSpace, NTag, NSwitch, NIcon,
  NModal, NForm, NFormItem, NInput, NSelect, NSpin,
  NPopconfirm, useMessage, type DataTableColumns
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { AddOutline, PlayOutline, TrashOutline } from '@vicons/ionicons5'
import { reportApi, type ReportTask, type ReportRun } from '@/api/report'
import { getErrorMessage } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const loading = ref(false)
const tasks = ref<ReportTask[]>([])
const runs = ref<ReportRun[]>([])
const showModal = ref(false)
const editingTask = ref<Partial<ReportTask> | null>(null)
const selectedTaskId = ref<number | null>(null)

const reportTypeOptions = [
  { label: t('report.typeDaily'), value: 'daily' },
  { label: t('report.typeWeekly'), value: 'weekly' },
  { label: t('report.typeCustom'), value: 'custom' },
]

const taskColumns: DataTableColumns<ReportTask> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: t('report.name'), key: 'name', ellipsis: { tooltip: true } },
  { title: t('report.type'), key: 'report_type', width: 80, render: (row) => h(NTag, { size: 'small', bordered: false }, { default: () => row.report_type }) },
  { title: t('report.cron'), key: 'cron_expr', width: 120 },
  { title: t('common.enabled'), key: 'enabled', width: 80, render: (row) => h(NSwitch, { value: row.enabled, size: 'small', onUpdateValue: (v: boolean) => toggleEnabled(row, v) }) },
  {
    title: t('common.actions'), key: 'actions', width: 160,
    render: (row) => h(NSpace, { size: 4 }, {
      default: () => [
        h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => runNow(row) }, { icon: () => h(NIcon, { component: PlayOutline }), default: () => t('report.runNow') }),
        h(NButton, { size: 'tiny', quaternary: true, onClick: () => editTask(row) }, { default: () => t('common.edit') }),
        h(NPopconfirm, { onPositiveClick: () => deleteTask(row.id) }, {
          trigger: () => h(NButton, { size: 'tiny', quaternary: true, type: 'error' }, { icon: () => h(NIcon, { component: TrashOutline }) }),
          default: () => t('common.confirmDelete'),
        }),
      ]
    })
  }
]

const runColumns: DataTableColumns<ReportRun> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: t('report.status'), key: 'status', width: 80, render: (row) => h(NTag, { type: row.status === 'success' ? 'success' : row.status === 'failed' ? 'error' : 'warning', size: 'small', bordered: false }, { default: () => row.status }) },
  { title: t('report.startedAt'), key: 'started_at', width: 160 },
  { title: t('report.summary'), key: 'report_summary', ellipsis: { tooltip: true } },
  { title: t('report.error'), key: 'error_msg', ellipsis: { tooltip: true }, render: (row) => row.error_msg ? h('span', { style: 'color: var(--error-color)' }, row.error_msg) : '-' },
]

async function loadTasks() {
  loading.value = true
  try {
    const res = await reportApi.listTasks()
    tasks.value = res.data.data?.list ?? []
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function loadRuns(taskId?: number) {
  try {
    const res = await reportApi.listRuns(taskId ? { task_id: taskId } : undefined)
    runs.value = res.data.data?.list ?? []
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

async function toggleEnabled(task: ReportTask, enabled: boolean) {
  try {
    await reportApi.updateTask(task.id, { ...task, enabled })
    task.enabled = enabled
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

async function runNow(task: ReportTask) {
  try {
    await reportApi.runNow(task.id)
    message.success(t('report.runSubmitted'))
    setTimeout(() => loadRuns(task.id), 1000)
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

function createTask() {
  editingTask.value = { name: '', description: '', cron_expr: '0 9 * * *', report_type: 'daily', prompt_template: '', output_channels: '[]', enabled: true }
  showModal.value = true
}

function editTask(task: ReportTask) {
  editingTask.value = { ...task }
  showModal.value = true
}

async function saveTask() {
  if (!editingTask.value) return
  try {
    if (editingTask.value.id) {
      await reportApi.updateTask(editingTask.value.id, editingTask.value)
    } else {
      await reportApi.createTask(editingTask.value)
    }
    showModal.value = false
    message.success(t('common.savedSuccess'))
    loadTasks()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

async function deleteTask(id: number) {
  try {
    await reportApi.deleteTask(id)
    message.success(t('common.deletedSuccess'))
    loadTasks()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

function viewRuns(taskId: number) {
  selectedTaskId.value = taskId
  loadRuns(taskId)
}

onMounted(() => {
  loadTasks()
  loadRuns()
})
</script>

<template>
  <div class="sre-config-page">
    <header class="sre-config-header">
      <div>
        <h2 class="sre-config-header-title">{{ t('report.title') }}</h2>
        <p class="sre-config-header-sub">{{ t('report.subtitle') }}</p>
      </div>
      <NButton type="primary" size="small" @click="createTask">
        <template #icon><NIcon :component="AddOutline" /></template>
        {{ t('report.createTask') }}
      </NButton>
    </header>

    <NSpin :show="loading">
      <div class="sre-config-section">
        <h3 class="sre-config-section-title">{{ t('report.tasks') }}</h3>
        <NDataTable :columns="taskColumns" :data="tasks" :bordered="false" size="small" />
      </div>

      <div class="sre-config-section" style="margin-top: 24px">
        <h3 class="sre-config-section-title">
          {{ t('report.runs') }}
          <NTag v-if="selectedTaskId" size="small" closable @close="selectedTaskId = null; loadRuns()">
            Task #{{ selectedTaskId }}
          </NTag>
        </h3>
        <NDataTable :columns="runColumns" :data="runs" :bordered="false" size="small" />
      </div>
    </NSpin>

    <NModal v-model:show="showModal" preset="card" :title="editingTask?.id ? t('report.editTask') : t('report.createTask')" style="max-width: 600px">
      <NForm v-if="editingTask" label-placement="left" label-width="100">
        <NFormItem :label="t('report.name')">
          <NInput v-model:value="editingTask.name" :placeholder="t('report.namePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('report.description')">
          <NInput v-model:value="editingTask.description" type="textarea" :placeholder="t('report.descPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('report.cron')">
          <NInput v-model:value="editingTask.cron_expr" placeholder="0 9 * * *" />
        </NFormItem>
        <NFormItem :label="t('report.type')">
          <NSelect v-model:value="editingTask.report_type" :options="reportTypeOptions" />
        </NFormItem>
        <NFormItem :label="t('report.prompt')">
          <NInput v-model:value="editingTask.prompt_template" type="textarea" :rows="4" :placeholder="t('report.promptPlaceholder')" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" @click="saveTask">{{ t('common.save') }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
