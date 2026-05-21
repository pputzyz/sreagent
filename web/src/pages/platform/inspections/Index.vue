<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  NButton, NIcon, NSwitch, NDataTable, NCard, NSpace, NTag,
  NModal, NForm, NFormItem, NInput, NSelect, NPopconfirm,
  useMessage, useDialog,
} from 'naive-ui'
import {
  AddOutline, PlayOutline, TrashOutline, CreateOutline,
  RefreshOutline, TimeOutline,
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
  { label: '全局', value: 'global' },
  { label: '业务组', value: 'biz_group' },
]

// --- Table columns ---
const taskColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '任务名称', key: 'name', width: 180 },
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
    title: '状态',
    key: 'enabled',
    width: 80,
    render: (row: InspectionTask) => h(NSwitch, {
      value: row.enabled,
      size: 'small',
      onUpdateValue: async (val: boolean) => {
        try {
          await inspectionApi.updateTask(row.id, { enabled: val })
          row.enabled = val
          message.success(val ? '已启用' : '已禁用')
        } catch (e) {
          message.error(getErrorMessage(e))
        }
      },
    }),
  },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    render: (row: InspectionTask) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'small', type: 'primary', secondary: true, onClick: () => runTask(row) }, {
          icon: () => h(NIcon, null, { default: () => h(PlayOutline) }),
          default: () => '执行',
        }),
        h(NButton, { size: 'small', secondary: true, onClick: () => openEdit(row) }, {
          icon: () => h(NIcon, null, { default: () => h(CreateOutline) }),
        }),
        h(NPopconfirm, { onPositiveClick: () => deleteTask(row) }, {
          trigger: () => h(NButton, { size: 'small', type: 'error', secondary: true }, {
            icon: () => h(NIcon, null, { default: () => h(TrashOutline) }),
          }),
          default: () => '确认删除该巡检任务？',
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
    message.success(`巡检任务 "${task.name}" 已提交`)
  } catch (e) {
    message.error(getErrorMessage(e))
  }
}

async function deleteTask(task: InspectionTask) {
  try {
    await inspectionApi.deleteTask(task.id)
    message.success('已删除')
    await fetchTasks()
  } catch (e) {
    message.error(getErrorMessage(e))
  }
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
  showModal.value = true
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
  showModal.value = true
}

async function handleSubmit() {
  if (!form.value.name.trim()) {
    message.warning('请输入任务名称')
    return
  }
  if (!form.value.description.trim()) {
    message.warning('请输入任务描述')
    return
  }
  formLoading.value = true
  try {
    if (editingTask.value) {
      await inspectionApi.updateTask(editingTask.value.id, form.value)
      message.success('已更新')
    } else {
      await inspectionApi.createTask(form.value)
      message.success('已创建')
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
  } catch {
    // ignore
  } finally {
    runsLoading.value = false
  }
}

function viewRun(run: InspectionRun) {
  router.push(`/platform/inspections/runs/${run.id}`)
}

const runColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '任务ID', key: 'task_id', width: 80 },
  {
    title: '状态',
    key: 'status',
    width: 80,
    render: (row: InspectionRun) => h(NTag, {
      type: row.status === 'success' ? 'success' : row.status === 'failed' ? 'error' : 'info',
      size: 'small',
    }, { default: () => row.status }),
  },
  { title: '摘要', key: 'report_summary', ellipsis: { tooltip: true } },
  {
    title: '开始时间',
    key: 'started_at',
    width: 170,
    render: (row: InspectionRun) => new Date(row.started_at).toLocaleString(),
  },
  {
    title: '操作',
    key: 'actions',
    width: 80,
    render: (row: InspectionRun) => h(NButton, {
      size: 'small',
      text: true,
      type: 'primary',
      onClick: () => viewRun(row),
    }, { default: () => '查看' }),
  },
]

onMounted(() => {
  fetchTasks()
  fetchRuns()
})
</script>

<template>
  <div style="padding: 16px; display: flex; flex-direction: column; gap: 16px;">
    <PageHeader title="定时巡检" description="配置和管理 AI 自动巡检任务">
      <template #actions>
        <NButton @click="fetchTasks">
          <template #icon><NIcon><RefreshOutline /></NIcon></template>
          刷新
        </NButton>
        <NButton type="primary" @click="openCreate">
          <template #icon><NIcon><AddOutline /></NIcon></template>
          新建任务
        </NButton>
      </template>
    </PageHeader>

    <!-- Tasks table -->
    <NCard title="巡检任务" size="small">
      <NDataTable
        v-if="tasks.length > 0"
        :columns="taskColumns"
        :data="tasks"
        :loading="loading"
        :bordered="false"
        size="small"
      />
      <EmptyState v-else-if="!loading" title="暂无巡检任务" description="点击右上角新建按钮创建巡检任务" />
    </NCard>

    <!-- Recent runs -->
    <NCard title="最近执行记录" size="small">
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
      :title="editingTask ? '编辑巡检任务' : '新建巡检任务'"
      style="width: 600px"
      :bordered="false"
    >
      <NForm label-placement="left" label-width="100">
        <NFormItem label="任务名称" required>
          <NInput v-model:value="form.name" placeholder="如：每日集群健康检查" />
        </NFormItem>
        <NFormItem label="任务描述" required>
          <NInput
            v-model:value="form.description"
            type="textarea"
            :rows="4"
            placeholder="自然语言描述巡检目标，将直接喂给 AI Agent"
          />
        </NFormItem>
        <NFormItem label="定时规则" required>
          <CronInput v-model="form.cron_expr" />
        </NFormItem>
        <NFormItem label="目标类型">
          <NSelect v-model:value="form.target_type" :options="targetTypeOptions" />
        </NFormItem>
        <NFormItem v-if="form.target_type === 'biz_group'" label="目标 IDs">
          <NInput v-model:value="form.target_ids" placeholder='JSON 数组，如 [1,2,3]' />
        </NFormItem>
        <NFormItem label="工具白名单">
          <NInput
            v-model:value="form.allowed_tools"
            placeholder='留空=全部只读工具，或 JSON 数组 ["tool_a","tool_b"]'
          />
        </NFormItem>
        <NFormItem label="输出渠道">
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
          <NButton @click="showModal = false">取消</NButton>
          <NButton type="primary" :loading="formLoading" @click="handleSubmit">
            {{ editingTask ? '更新' : '创建' }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
