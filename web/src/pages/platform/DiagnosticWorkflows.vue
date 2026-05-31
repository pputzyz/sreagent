<script setup lang="ts">
import { ref, computed, onMounted, h, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import {
  NButton, NIcon, NTag, NSwitch, NDrawer, NDrawerContent,
  NForm, NFormItem, NInput, NInputNumber, NSelect, NSpace, NDataTable,
  NPagination, NEmpty, NTabs, NTabPane, NCollapse, NCollapseItem,
  NTimeline, NTimelineItem, NProgress,
} from 'naive-ui'
import {
  AddOutline, SearchOutline, PlayOutline, TrashOutline, SwapVerticalOutline,
  CheckmarkCircleOutline, CloseCircleOutline, TimeOutline, EllipseOutline,
} from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import { diagnosticApi } from '@/api/diagnostic'
import type { DiagnosticWorkflow, DiagnosticWorkflowStep, DiagnosticRun, DiagnosticRunStep } from '@/api/diagnostic'
import { getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const authStore = useAuthStore()

// ─── Tab state ───
const activeTab = ref('workflows')

// ─── Workflows state ───
const workflows = ref<DiagnosticWorkflow[]>([])
const wfTotal = ref(0)
const wfLoading = ref(false)
const wfPage = ref(1)
const wfPageSize = ref(20)
const wfSearch = ref('')

// ─── Runs state ───
const runs = ref<DiagnosticRun[]>([])
const runTotal = ref(0)
const runLoading = ref(false)
const runPage = ref(1)
const runPageSize = ref(20)

// ─── Drawer ───
const showDrawer = ref(false)
const drawerMode = ref<'create' | 'edit'>('create')
const editingId = ref<number | null>(null)
const saving = ref(false)

// ─── Run dialog ───
const showRunDialog = ref(false)
const runTargetId = ref<number | null>(null)
const runIncidentId = ref<number | null>(null)
const runSubmitting = ref(false)

// ─── Run detail drawer ───
const showRunDetail = ref(false)
const runDetail = ref<DiagnosticRun | null>(null)
const runDetailSteps = ref<DiagnosticRunStep[]>([])

// ─── Form ───
const form = ref<Partial<DiagnosticWorkflow>>({
  name: '',
  description: '',
  trigger_labels: {},
  trigger_severity: '',
  category: 'general',
  enabled: true,
  max_steps: 10,
  require_approval: true,
})

const steps = ref<DiagnosticWorkflowStep[]>([])

// ─── KV editor for trigger_labels ───
const triggerLabelsStr = ref('{}')

function syncTriggerLabels() {
  try {
    form.value.trigger_labels = JSON.parse(triggerLabelsStr.value)
  } catch { /* keep previous value */ }
}

function syncTriggerLabelsStr() {
  triggerLabelsStr.value = JSON.stringify(form.value.trigger_labels || {}, null, 2)
}

// ─── Options ───
const stepTypeOptions = computed(() => [
  { label: t('diagnostic.stepQuery'), value: 'query' },
  { label: t('diagnostic.stepCommand'), value: 'command' },
  { label: t('diagnostic.stepCheck'), value: 'check' },
])

const onFailureOptions = computed(() => [
  { label: t('diagnostic.continue'), value: 'continue' },
  { label: t('diagnostic.stop'), value: 'stop' },
])

const runStatusOptions = computed(() => [
  { label: t('diagnostic.statusPending'), value: 'pending' },
  { label: t('diagnostic.statusRunning'), value: 'running' },
  { label: t('diagnostic.statusCompleted'), value: 'completed' },
  { label: t('diagnostic.statusFailed'), value: 'failed' },
])

// ─── Filtered workflows ───
const filteredWorkflows = computed(() => {
  const q = wfSearch.value.trim().toLowerCase()
  if (!q) return workflows.value
  return workflows.value.filter(w =>
    w.name.toLowerCase().includes(q) ||
    (w.description || '').toLowerCase().includes(q) ||
    w.category.toLowerCase().includes(q)
  )
})

// ─── Workflows API ───
async function fetchWorkflows() {
  wfLoading.value = true
  try {
    const resp = await diagnosticApi.listWorkflows({ page: wfPage.value, page_size: wfPageSize.value })
    workflows.value = resp.data.data?.list || []
    wfTotal.value = resp.data.data?.total || 0
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    wfLoading.value = false
  }
}

// ─── Runs API ───
async function fetchRuns() {
  runLoading.value = true
  try {
    const resp = await diagnosticApi.listRuns({ page: runPage.value, page_size: runPageSize.value })
    runs.value = resp.data.data?.list || []
    runTotal.value = resp.data.data?.total || 0
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    runLoading.value = false
  }
}

// ─── CRUD ───
function openCreate() {
  drawerMode.value = 'create'
  editingId.value = null
  resetForm()
  showDrawer.value = true
}

async function openEdit(wf: DiagnosticWorkflow) {
  drawerMode.value = 'edit'
  editingId.value = wf.id
  try {
    const resp = await diagnosticApi.getWorkflow(wf.id)
    const data = resp.data.data
    form.value = { ...data.workflow }
    steps.value = data.steps.map((s, i) => ({ ...s, step_order: i + 1 }))
    syncTriggerLabelsStr()
    showDrawer.value = true
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  }
}

function resetForm() {
  form.value = {
    name: '',
    description: '',
    trigger_labels: {},
    trigger_severity: '',
    category: 'general',
    enabled: true,
    max_steps: 10,
    require_approval: true,
  }
  steps.value = []
  triggerLabelsStr.value = '{}'
}

async function handleSave() {
  if (!form.value.name?.trim()) {
    message.warning(t('common.required'))
    return
  }
  syncTriggerLabels()
  saving.value = true
  try {
    const stepsPayload = steps.value.map((s, i) => ({
      ...s,
      step_order: i + 1,
    }))
    if (drawerMode.value === 'edit' && editingId.value) {
      await diagnosticApi.updateWorkflow(editingId.value, form.value)
      await diagnosticApi.replaceSteps(editingId.value, stepsPayload)
      message.success(t('common.updateSuccess'))
    } else {
      await diagnosticApi.createWorkflow({ workflow: form.value, steps: stepsPayload })
      message.success(t('common.createSuccess'))
    }
    showDrawer.value = false
    fetchWorkflows()
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    saving.value = false
  }
}

function confirmDelete(wf: DiagnosticWorkflow) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('common.confirmDeleteMsg'),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await diagnosticApi.deleteWorkflow(wf.id)
        message.success(t('common.deleteSuccess'))
        fetchWorkflows()
      } catch (e: unknown) {
        message.error(getErrorMessage(e))
      }
    },
  })
}

async function toggleEnabled(wf: DiagnosticWorkflow) {
  try {
    await diagnosticApi.updateWorkflow(wf.id, { enabled: !wf.enabled })
    wf.enabled = !wf.enabled
    message.success(t('common.success'))
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  }
}

// ─── Run ───
function openRunDialog(wf: DiagnosticWorkflow) {
  runTargetId.value = wf.id
  runIncidentId.value = null
  showRunDialog.value = true
}

async function handleStartRun() {
  if (!runTargetId.value) return
  runSubmitting.value = true
  try {
    await diagnosticApi.startRun(runTargetId.value, runIncidentId.value || undefined)
    message.success(t('common.success'))
    showRunDialog.value = false
    if (activeTab.value === 'runs') fetchRuns()
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    runSubmitting.value = false
  }
}

// ─── Steps editor ───
// FE7-8: Drag-and-drop reordering for workflow steps
const dragIndex = ref<number | null>(null)
const dragOverIndex = ref<number | null>(null)

function onDragStart(idx: number, e: DragEvent) {
  dragIndex.value = idx
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', String(idx))
  }
}

function onDragOver(idx: number, e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  dragOverIndex.value = idx
}

function onDragLeave() {
  dragOverIndex.value = null
}

function onDrop(idx: number, e: DragEvent) {
  e.preventDefault()
  dragOverIndex.value = null
  if (dragIndex.value === null || dragIndex.value === idx) return
  const arr = [...steps.value]
  const [moved] = arr.splice(dragIndex.value, 1)
  arr.splice(idx, 0, moved)
  // Update step_order
  arr.forEach((s, i) => { s.step_order = i + 1 })
  steps.value = arr
  dragIndex.value = null
}

function onDragEnd() {
  dragIndex.value = null
  dragOverIndex.value = null
}

function moveStep(idx: number, direction: -1 | 1) {
  const target = idx + direction
  if (target < 0 || target >= steps.value.length) return
  const arr = [...steps.value]
  ;[arr[idx], arr[target]] = [arr[target], arr[idx]]
  arr.forEach((s, i) => { s.step_order = i + 1 })
  steps.value = arr
}

function addStep() {
  steps.value.push({
    step_order: steps.value.length + 1,
    name: '',
    step_type: 'query',
    datasource_id: null,
    expression: '',
    condition_expr: '',
    auto_advance: true,
    timeout_seconds: 30,
    on_failure: 'continue',
  })
}

function removeStep(index: number) {
  steps.value.splice(index, 1)
}

// ─── Run detail ───
// FE7-10: Timeline visualization for run results
function stepTimelineType(status: string): 'success' | 'warning' | 'error' | 'info' {
  if (status === 'completed') return 'success'
  if (status === 'running') return 'warning'
  if (status === 'failed') return 'error'
  return 'info'
}

function stepTimelineIcon(status: string) {
  if (status === 'completed') return CheckmarkCircleOutline
  if (status === 'failed') return CloseCircleOutline
  if (status === 'running') return TimeOutline
  return EllipseOutline
}

const runProgress = computed(() => {
  if (!runDetailSteps.value.length) return 0
  const done = runDetailSteps.value.filter(s => s.status === 'completed' || s.status === 'failed').length
  return Math.round((done / runDetailSteps.value.length) * 100)
})

async function openRunDetail(run: DiagnosticRun) {
  try {
    const resp = await diagnosticApi.getRun(run.id)
    const data = resp.data.data
    runDetail.value = data.run
    runDetailSteps.value = data.steps || []
    showRunDetail.value = true
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  }
}

// ─── Helpers ───
function runStatusTag(status: string): 'success' | 'warning' | 'error' | 'default' {
  if (status === 'completed') return 'success'
  if (status === 'running') return 'warning'
  if (status === 'failed') return 'error'
  return 'default'
}

function stepStatusTag(status: string): 'success' | 'warning' | 'error' | 'default' {
  if (status === 'completed') return 'success'
  if (status === 'running') return 'warning'
  if (status === 'failed') return 'error'
  return 'default'
}

function truncateTime(s: string | null): string {
  if (!s) return '-'
  return s.replace('T', ' ').substring(0, 19)
}

// ─── Workflow columns ───
const wfColumns = computed<DataTableColumns<DiagnosticWorkflow>>(() => [
  {
    title: t('common.name'),
    key: 'name',
    minWidth: 180,
    ellipsis: { tooltip: true },
    render: (row) =>
      h('a', {
        style: 'color: var(--sre-primary); cursor: pointer; text-decoration: none;',
        onClick: () => openEdit(row),
      }, row.name),
  },
  {
    title: t('common.type'),
    key: 'category',
    width: 120,
    render: (row) =>
      h(NTag, { size: 'small', bordered: false, type: 'info' }, () => row.category),
  },
  {
    title: t('common.enabled'),
    key: 'enabled',
    width: 90,
    render: (row) =>
      h(NSwitch, {
        value: row.enabled,
        size: 'small',
        'onUpdate:value': () => toggleEnabled(row),
      }),
  },
  {
    title: t('diagnostic.maxSteps'),
    key: 'max_steps',
    width: 100,
    render: (row) => row.max_steps,
  },
  {
    title: t('diagnostic.requireApproval'),
    key: 'require_approval',
    width: 110,
    render: (row) =>
      h(NTag, {
        size: 'small',
        bordered: false,
        type: row.require_approval ? 'warning' : 'default',
      }, () => row.require_approval ? t('common.yes') : t('common.no')),
  },
  {
    title: t('common.createdAt'),
    key: 'created_at',
    width: 170,
    render: (row) => truncateTime(row.created_at),
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 180,
    fixed: 'right',
    render: (row) =>
      h(NSpace, { size: 'small' }, () => [
        h(NButton, {
          size: 'tiny',
          quaternary: true,
          type: 'primary',
          onClick: () => openRunDialog(row),
        }, { default: () => t('diagnostic.startRun'), icon: () => h(NIcon, { size: 14, component: PlayOutline }) }),
        h(NButton, {
          size: 'tiny',
          quaternary: true,
          type: 'error',
          onClick: () => confirmDelete(row),
        }, { default: () => t('common.delete'), icon: () => h(NIcon, { size: 14, component: TrashOutline }) }),
      ]),
  },
])

// ─── Runs columns ───
const runColumns = computed<DataTableColumns<DiagnosticRun>>(() => [
  {
    title: t('common.name'),
    key: 'workflow_id',
    minWidth: 160,
    render: (row) => {
      const wf = workflows.value.find(w => w.id === row.workflow_id)
      return wf ? wf.name : `#${row.workflow_id}`
    },
  },
  {
    title: t('diagnostic.incidentId'),
    key: 'incident_id',
    width: 120,
    render: (row) => row.incident_id ?? '-',
  },
  {
    title: t('common.status'),
    key: 'status',
    width: 110,
    render: (row) =>
      h(NTag, { size: 'small', bordered: false, type: runStatusTag(row.status) }, () => row.status),
  },
  {
    title: t('diagnostic.steps'),
    key: 'current_step',
    width: 100,
  },
  {
    title: t('common.createdAt'),
    key: 'started_at',
    width: 170,
    render: (row) => truncateTime(row.started_at),
  },
  {
    title: t('common.updatedAt'),
    key: 'completed_at',
    width: 170,
    render: (row) => truncateTime(row.completed_at),
  },
])

// ─── Init ───
watch(activeTab, (tab) => {
  if (tab === 'runs') fetchRuns()
})

onMounted(() => {
  fetchWorkflows()
})
</script>

<template>
  <div class="diagnostic-page">
    <PageHeader :title="t('diagnostic.title')" :subtitle="t('diagnostic.subtitle')">
      <template #actions>
        <n-button
          v-if="authStore.canManage"
          type="primary"
          size="small"
          @click="openCreate"
        >
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('common.create') }}
        </n-button>
      </template>
    </PageHeader>

    <n-tabs v-model:value="activeTab" type="line" animated>
      <!-- ===== Workflows Tab ===== -->
      <n-tab-pane name="workflows" :tab="t('diagnostic.workflows')">
        <div class="toolbar">
          <n-input
            v-model:value="wfSearch"
            size="small"
            :placeholder="t('common.search')"
            clearable
            style="width: 260px"
          >
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <span class="count tnum">{{ filteredWorkflows.length }} / {{ workflows.length }}</span>
        </div>

        <n-empty
          v-if="!wfLoading && workflows.length === 0"
          :description="t('common.noData')"
          style="padding: 60px 0"
        >
          <template #extra>
            <n-button v-if="authStore.canManage" type="primary" size="small" @click="openCreate">{{ t('common.create') }}</n-button>
          </template>
        </n-empty>

        <template v-else>
          <n-data-table
            :columns="wfColumns"
            :data="filteredWorkflows"
            :loading="wfLoading"
            :row-key="(row: DiagnosticWorkflow) => row.id"
            size="small"
            :bordered="false"
            striped
            :scroll-x="1000"
          />

          <div class="page-pagination" v-if="wfTotal > 0">
            <n-pagination
              v-model:page="wfPage"
              v-model:page-size="wfPageSize"
              :item-count="wfTotal"
              :page-sizes="[20, 50, 100]"
              show-size-picker
              @update:page="fetchWorkflows"
              @update:page-size="(ps: number) => { wfPageSize = ps; wfPage = 1; fetchWorkflows() }"
            />
          </div>
        </template>
      </n-tab-pane>

      <!-- ===== Runs Tab ===== -->
      <n-tab-pane name="runs" :tab="t('diagnostic.runs')">
        <n-empty
          v-if="!runLoading && runs.length === 0"
          :description="t('common.noData')"
          style="padding: 60px 0"
        />

        <template v-else>
          <n-data-table
            :columns="runColumns"
            :data="runs"
            :loading="runLoading"
            :row-key="(row: DiagnosticRun) => row.id"
            size="small"
            :bordered="false"
            striped
            :scroll-x="900"
            :row-props="(row: DiagnosticRun) => ({ style: 'cursor: pointer;', onClick: () => openRunDetail(row) })"
          />

          <div class="page-pagination" v-if="runTotal > 0">
            <n-pagination
              v-model:page="runPage"
              v-model:page-size="runPageSize"
              :item-count="runTotal"
              :page-sizes="[20, 50, 100]"
              show-size-picker
              @update:page="fetchRuns"
              @update:page-size="(ps: number) => { runPageSize = ps; runPage = 1; fetchRuns() }"
            />
          </div>
        </template>
      </n-tab-pane>
    </n-tabs>

    <!-- ===== Create/Edit Drawer ===== -->
    <n-drawer v-model:show="showDrawer" :width="640">
      <n-drawer-content :title="drawerMode === 'edit' ? t('common.edit') : t('common.create')">
        <n-form label-placement="top">
          <n-form-item :label="t('common.name')" required>
            <n-input v-model:value="form.name" />
          </n-form-item>

          <n-form-item :label="t('common.description')">
            <n-input v-model:value="form.description" type="textarea" :rows="2" />
          </n-form-item>

          <div style="display: flex; gap: 16px;">
            <n-form-item :label="t('common.type')" style="flex: 1;">
              <n-input v-model:value="form.category" />
            </n-form-item>
            <n-form-item :label="t('diagnostic.triggerSeverity')" style="flex: 1;">
              <n-select
                v-model:value="form.trigger_severity"
                clearable
                :options="[
                  { label: t('severity.critical'), value: 'critical' },
                  { label: t('severity.warning'), value: 'warning' },
                  { label: t('severity.info'), value: 'info' },
                ]"
              />
            </n-form-item>
          </div>

          <n-form-item :label="t('diagnostic.triggerLabels')">
            <n-input
              v-model:value="triggerLabelsStr"
              type="textarea"
              :rows="3"
              placeholder='{"env":"production"}'
              @blur="syncTriggerLabels"
            />
          </n-form-item>

          <div style="display: flex; gap: 16px;">
            <n-form-item :label="t('diagnostic.maxSteps')" style="flex: 1;">
              <n-input-number v-model:value="form.max_steps" :min="1" :max="50" style="width: 100%" />
            </n-form-item>
            <n-form-item :label="t('common.enabled')" style="flex: 1;">
              <n-switch v-model:value="form.enabled" />
            </n-form-item>
            <n-form-item :label="t('diagnostic.requireApproval')" style="flex: 1;">
              <n-switch v-model:value="form.require_approval" />
            </n-form-item>
          </div>

          <!-- Steps editor -->
          <n-collapse>
            <n-collapse-item :title="t('diagnostic.steps') + ` (${steps.length})`" name="steps">
              <div class="steps-list">
                <div
                  v-for="(step, idx) in steps"
                  :key="idx"
                  class="step-card"
                  :class="{ 'step-drag-over': dragOverIndex === idx, 'step-dragging': dragIndex === idx }"
                  draggable="true"
                  @dragstart="onDragStart(idx, $event)"
                  @dragover="onDragOver(idx, $event)"
                  @dragleave="onDragLeave"
                  @drop="onDrop(idx, $event)"
                  @dragend="onDragEnd"
                >
                  <div class="step-header">
                    <n-icon :component="SwapVerticalOutline" class="drag-handle" size="14" />
                    <span class="step-order">#{{ idx + 1 }}</span>
                    <div class="step-header-actions">
                      <n-button size="tiny" quaternary :disabled="idx === 0" @click="moveStep(idx, -1)">
                        &uarr;
                      </n-button>
                      <n-button size="tiny" quaternary :disabled="idx === steps.length - 1" @click="moveStep(idx, 1)">
                        &darr;
                      </n-button>
                      <n-button size="tiny" quaternary type="error" @click="removeStep(idx)">
                        {{ t('common.remove') }}
                      </n-button>
                    </div>
                  </div>
                  <n-form label-placement="top" size="small">
                    <div style="display: flex; gap: 12px;">
                      <n-form-item :label="t('common.name')" style="flex: 2;" required>
                        <n-input v-model:value="step.name" />
                      </n-form-item>
                      <n-form-item :label="t('diagnostic.stepType')" style="flex: 1;">
                        <n-select v-model:value="step.step_type" :options="stepTypeOptions" />
                      </n-form-item>
                    </div>
                    <div style="display: flex; gap: 12px;">
                      <n-form-item :label="t('alert.dataSource')" style="flex: 1;">
                        <n-input-number v-model:value="step.datasource_id" :min="0" style="width: 100%" />
                      </n-form-item>
                      <n-form-item :label="t('diagnostic.timeoutSeconds')" style="flex: 1;">
                        <n-input-number v-model:value="step.timeout_seconds" :min="1" :max="600" style="width: 100%" />
                      </n-form-item>
                      <n-form-item :label="t('diagnostic.onFailure')" style="flex: 1;">
                        <n-select v-model:value="step.on_failure" :options="onFailureOptions" />
                      </n-form-item>
                    </div>
                    <n-form-item :label="t('diagnostic.expression')">
                      <n-input v-model:value="step.expression" type="textarea" :rows="2" />
                    </n-form-item>
                    <n-form-item :label="t('diagnostic.conditionExpr')">
                      <n-input v-model:value="step.condition_expr" />
                    </n-form-item>
                  </n-form>
                </div>
                <n-button dashed size="small" @click="addStep" style="width: 100%;">
                  {{ t('common.add') }}
                </n-button>
              </div>
            </n-collapse-item>
          </n-collapse>
        </n-form>

        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 8px;">
            <n-button @click="showDrawer = false">{{ t('common.cancel') }}</n-button>
            <n-button type="primary" :loading="saving" @click="handleSave">
              {{ drawerMode === 'edit' ? t('common.update') : t('common.create') }}
            </n-button>
          </div>
        </template>
      </n-drawer-content>
    </n-drawer>

    <!-- ===== Run Dialog ===== -->
    <n-drawer v-model:show="showRunDialog" :width="400">
      <n-drawer-content :title="t('diagnostic.startRun')">
        <n-form label-placement="top">
          <n-form-item :label="t('diagnostic.incidentId')">
            <n-input-number
              v-model:value="runIncidentId"
              :min="0"
              style="width: 100%"
              :placeholder="t('common.optional')"
            />
          </n-form-item>
        </n-form>
        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 8px;">
            <n-button @click="showRunDialog = false">{{ t('common.cancel') }}</n-button>
            <n-button type="primary" :loading="runSubmitting" @click="handleStartRun">
              {{ t('diagnostic.startRun') }}
            </n-button>
          </div>
        </template>
      </n-drawer-content>
    </n-drawer>

    <!-- ===== Run Detail Drawer ===== -->
    <n-drawer v-model:show="showRunDetail" :width="600">
      <n-drawer-content :title="runDetail ? t('diagnostic.runDetail', { id: runDetail.id }) : ''">
        <template v-if="runDetail">
          <div class="run-detail-meta">
            <div class="meta-row">
              <span class="meta-label">{{ t('common.status') }}</span>
              <n-tag size="small" :type="runStatusTag(runDetail.status)" :bordered="false">
                {{ runDetail.status }}
              </n-tag>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('diagnostic.incidentId') }}</span>
              <span>{{ runDetail.incident_id ?? '-' }}</span>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('diagnostic.steps') }}</span>
              <span>{{ runDetail.current_step }}</span>
            </div>
            <div class="meta-row" v-if="runDetail.result_summary">
              <span class="meta-label">{{ t('common.description') }}</span>
              <span>{{ runDetail.result_summary }}</span>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('common.createdAt') }}</span>
              <span>{{ truncateTime(runDetail.started_at) }}</span>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('common.updatedAt') }}</span>
              <span>{{ truncateTime(runDetail.completed_at) }}</span>
            </div>
          </div>

          <div style="margin-top: 16px;">
            <div class="sre-label-eyebrow" style="margin-bottom: 8px;">
              {{ t('diagnostic.steps') }}
              <span v-if="runDetailSteps.length > 0" style="margin-left: 8px; font-size: 11px; color: var(--sre-text-tertiary);">
                {{ runProgress }}%
              </span>
            </div>
            <n-progress
              v-if="runDetailSteps.length > 0"
              :percentage="runProgress"
              :show-indicator="false"
              :height="4"
              style="margin-bottom: 16px;"
            />
            <div v-if="runDetailSteps.length === 0" style="color: var(--sre-text-tertiary); font-size: 13px;">
              {{ t('common.noData') }}
            </div>
            <n-timeline v-else>
              <n-timeline-item
                v-for="step in runDetailSteps"
                :key="step.id"
                :type="stepTimelineType(step.status)"
                :icon="stepTimelineIcon(step.status)"
              >
                <div class="run-step-header">
                  <span class="step-order">#{{ step.step_order }}</span>
                  <span class="run-step-name">{{ step.step_name }}</span>
                  <n-tag size="tiny" :type="stepStatusTag(step.status)" :bordered="false">
                    {{ step.status }}
                  </n-tag>
                  <span class="run-step-duration">{{ step.duration_ms }}ms</span>
                </div>
                <div v-if="step.result" class="run-step-result">{{ step.result }}</div>
                <div v-if="step.error" class="run-step-error">{{ step.error }}</div>
              </n-timeline-item>
            </n-timeline>
          </div>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<style scoped>
.diagnostic-page {
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
.steps-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.step-card {
  padding: 12px;
  border: var(--sre-hairline);
  border-radius: 8px;
  background: var(--sre-bg-elevated, rgba(255, 255, 255, 0.02));
  transition: border-color 0.15s, opacity 0.15s, transform 0.15s;
}
.step-card.step-drag-over {
  border-color: var(--sre-primary);
  border-style: dashed;
}
.step-card.step-dragging {
  opacity: 0.5;
}
.drag-handle {
  cursor: grab;
  color: var(--sre-text-tertiary);
  flex-shrink: 0;
}
.drag-handle:active {
  cursor: grabbing;
}
.step-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}
.step-header-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-left: auto;
}
.step-order {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-tertiary);
  font-weight: 600;
}
.run-detail-meta {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.meta-row {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 13px;
}
.meta-label {
  min-width: 100px;
  color: var(--sre-text-tertiary);
  font-size: 12px;
}
.run-step-card {
  padding: 10px 12px;
  border: var(--sre-hairline);
  border-radius: 6px;
  margin-bottom: 8px;
  background: var(--sre-bg-elevated, rgba(255, 255, 255, 0.02));
}
.run-step-header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.run-step-name {
  font-weight: 500;
  font-size: 13px;
  color: var(--sre-text-primary);
}
.run-step-duration {
  margin-left: auto;
  font-size: 11px;
  font-family: var(--sre-font-mono, monospace);
  color: var(--sre-text-tertiary);
}
.run-step-result {
  margin-top: 6px;
  font-size: 12px;
  color: var(--sre-text-secondary);
  font-family: var(--sre-font-mono, monospace);
  word-break: break-all;
}
.run-step-error {
  margin-top: 4px;
  font-size: 12px;
  color: var(--sre-error, #e88080);
}
</style>
