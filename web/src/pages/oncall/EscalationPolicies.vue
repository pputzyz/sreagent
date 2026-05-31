<script setup lang="ts">
// TODO(FE3-1): Refactor to use useCrudPage composable — this page manually reimplements
// loading/modal/form/save/delete patterns that useCrudPage already provides.
// The nested-steps form logic is unique, but list + create/edit/delete can use the composable.
import { ref, onMounted, h } from 'vue'
import {
  useMessage, useDialog, NButton, NDataTable, NIcon, NSpace, NPopconfirm,
  NModal, NForm, NFormItem, NInput, NSelect, NInputNumber, NTag, NEmpty,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { escalationApi, teamApi, userApi, scheduleApi, notifyMediaApi } from '@/api'
import type { EscalationPolicy, Team, User, Schedule, NotifyMedia } from '@/types'
import { getErrorMessage } from '@/utils/format'
import { useAuthStore } from '@/stores/auth'
import PageHeader from '@/components/common/PageHeader.vue'
import { AddOutline, TrashOutline } from '@vicons/ionicons5'

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()
const authStore = useAuthStore()

const loading = ref(false)
const policies = ref<EscalationPolicy[]>([])
const teams = ref<Team[]>([])
const users = ref<User[]>([])
const schedules = ref<Schedule[]>([])
const channels = ref<NotifyMedia[]>([])

// Modal state
const showModal = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const form = ref({
  name: '',
  description: '',
  team_id: null as number | null,
  steps: [] as { step_order: number; target_type: string; target_id: number; delay_minutes: number; notify_channel_id: number | null }[],
})

const targetTypeOptions = [
  { label: t('escalation.user'), value: 'user' },
  { label: t('escalation.team'), value: 'team' },
  { label: t('escalation.schedule'), value: 'schedule' },
]

async function fetchPolicies() {
  loading.value = true
  try {
    const res = await escalationApi.list({ page: 1, page_size: 100 })
    policies.value = res.data.data?.list || []
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function fetchSupportData() {
  try {
    const [teamRes, userRes, schedRes, chanRes] = await Promise.all([
      teamApi.list({ page: 1, page_size: 100 }),
      userApi.list({ page: 1, page_size: 100 }),
      scheduleApi.list({ page: 1, page_size: 100 }),
      notifyMediaApi.list({ page: 1, page_size: 200 }),
    ])
    teams.value = teamRes.data.data?.list || []
    users.value = userRes.data.data?.list || []
    schedules.value = schedRes.data.data?.list || []
    channels.value = chanRes.data.data?.list || []
  } catch { /* ignore */ }
}

function openCreate() {
  editingId.value = null
  form.value = { name: '', description: '', team_id: null, steps: [] }
  showModal.value = true
}

function openEdit(policy: EscalationPolicy) {
  editingId.value = policy.id
  form.value = {
    name: policy.name,
    description: policy.description || '',
    team_id: policy.team_id || null,
    steps: (policy.steps || []).map((s, i) => ({
      step_order: s.step_order ?? i + 1,
      target_type: s.target_type || 'user',
      target_id: s.target_id || 0,
      delay_minutes: s.delay_minutes || 5,
      notify_channel_id: s.notify_channel_id || null,
    })),
  }
  showModal.value = true
}

function addStep() {
  form.value.steps.push({
    step_order: form.value.steps.length + 1,
    target_type: 'user',
    target_id: 0,
    delay_minutes: 5,
    notify_channel_id: null,
  })
}

function removeStep(index: number) {
  form.value.steps.splice(index, 1)
  form.value.steps.forEach((s, i) => { s.step_order = i + 1 })
}

async function handleSave() {
  if (!form.value.name.trim()) {
    message.warning(t('common.required'))
    return
  }
  for (let i = 0; i < form.value.steps.length; i++) {
    const step = form.value.steps[i]
    if (!step.target_id || step.target_id === 0) {
      message.warning(t('escalation.stepTargetRequired', { n: i + 1 }))
      return
    }
  }
  saving.value = true
  try {
    const payload = {
      name: form.value.name,
      description: form.value.description,
      team_id: form.value.team_id || undefined,
      steps: form.value.steps,
    }
    if (editingId.value) {
      await escalationApi.update(editingId.value, payload)
      message.success(t('common.savedSuccess'))
    } else {
      await escalationApi.create(payload)
      message.success(t('common.createSuccess'))
    }
    showModal.value = false
    fetchPolicies()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

function handleDelete(id: number) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('common.confirmDeleteMsg'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await escalationApi.delete(id)
        message.success(t('common.deleteSuccess'))
        fetchPolicies()
      } catch (err: unknown) {
        message.error(getErrorMessage(err))
      }
    },
  })
}

function getTargetName(type: string, id: number): string {
  if (type === 'user') {
    const u = users.value.find(u => u.id === id)
    return u ? (u.display_name || u.username) : `#${id}`
  }
  if (type === 'team') {
    const team = teams.value.find(t => t.id === id)
    return team ? team.name : `#${id}`
  }
  if (type === 'schedule') {
    const s = schedules.value.find(s => s.id === id)
    return s ? s.name : `#${id}`
  }
  return `#${id}`
}

const columns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: t('common.name'), key: 'name', ellipsis: { tooltip: true } },
  { title: t('common.description'), key: 'description', ellipsis: { tooltip: true } },
  {
    title: t('escalation.steps'),
    key: 'steps',
    width: 100,
    render: (row: EscalationPolicy) => (row.steps || []).length,
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 160,
    render: (row: EscalationPolicy) =>
      h(NSpace, { size: 'small' }, () => [
        ...(authStore.canManage ? [
          h(NButton, { size: 'tiny', secondary: true, onClick: () => openEdit(row) }, () => t('common.edit')),
          h(NButton, { size: 'tiny', type: 'error', secondary: true, onClick: () => handleDelete(row.id) }, () => t('common.delete')),
        ] : []),
      ]),
  },
]

onMounted(() => {
  fetchPolicies()
  fetchSupportData()
})
</script>

<template>
  <div class="page-container">
    <PageHeader :title="t('escalation.policies')" :subtitle="t('escalation.policiesSubtitle')">
      <template #actions>
        <n-button v-if="authStore.canManage" type="primary" size="small" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('escalation.createPolicy') }}
        </n-button>
      </template>
    </PageHeader>

    <n-data-table
      :columns="columns"
      :data="policies"
      :loading="loading"
      :bordered="false"
      size="small"
      striped
    >
      <template #empty>
        <n-empty :description="t('escalation.noPolicies') || t('common.noData')" />
      </template>
    </n-data-table>

    <!-- Create/Edit Modal -->
    <n-modal
      v-model:show="showModal"
      preset="card"
      :title="editingId ? t('escalation.editPolicy') : t('escalation.createPolicy')"
      style="max-width: 700px"
      :bordered="false"
    >
      <n-form label-placement="top">
        <n-form-item :label="t('common.name')" required>
          <n-input v-model:value="form.name" :placeholder="t('common.name')" />
        </n-form-item>
        <n-form-item :label="t('common.description')">
          <n-input v-model:value="form.description" type="textarea" :rows="2" />
        </n-form-item>
        <n-form-item :label="t('escalation.team')">
          <n-select
            v-model:value="form.team_id"
            :options="teams.map(t => ({ label: t.name, value: t.id }))"
            clearable
            :placeholder="t('escalation.teamPlaceholder')"
          />
        </n-form-item>

        <div class="steps-header">
          <span class="steps-title">{{ t('escalation.steps') }}</span>
          <n-button size="tiny" secondary @click="addStep">
            <template #icon><n-icon :component="AddOutline" /></template>
            {{ t('escalation.addStep') }}
          </n-button>
        </div>

        <div v-for="(step, idx) in form.steps" :key="idx" class="step-card">
          <div class="step-card-header">
            <n-tag size="small">{{ t('escalation.step', { n: step.step_order }) }}</n-tag>
            <n-button size="tiny" type="error" text @click="removeStep(idx)">
              <template #icon><n-icon :component="TrashOutline" /></template>
            </n-button>
          </div>
          <n-form-item :label="t('escalation.targetType')" label-placement="left" label-width="100">
            <n-select v-model:value="step.target_type" :options="targetTypeOptions" />
          </n-form-item>
          <n-form-item :label="t('escalation.target')" label-placement="left" label-width="100">
            <n-select
              v-if="step.target_type === 'user'"
              v-model:value="step.target_id"
              :options="users.map(u => ({ label: u.display_name || u.username, value: u.id }))"
              filterable
            />
            <n-select
              v-else-if="step.target_type === 'team'"
              v-model:value="step.target_id"
              :options="teams.map(t => ({ label: t.name, value: t.id }))"
              filterable
            />
            <n-select
              v-else
              v-model:value="step.target_id"
              :options="schedules.map(s => ({ label: s.name, value: s.id }))"
              filterable
            />
          </n-form-item>
          <n-form-item :label="t('escalation.delay')" label-placement="left" label-width="100">
            <n-input-number v-model:value="step.delay_minutes" :min="0" :max="1440" class="step-delay-input">
              <template #suffix>{{ t('common.minutes') }}</template>
            </n-input-number>
          </n-form-item>
          <n-form-item :label="t('escalation.notifyChannel')" label-placement="left" label-width="100">
            <n-select
              v-model:value="step.notify_channel_id"
              :options="channels.map(c => ({ label: c.name, value: c.id }))"
              clearable
              filterable
              :placeholder="t('escalation.notifyChannelPlaceholder')"
            />
          </n-form-item>
        </div>

        <div v-if="form.steps.length === 0" class="steps-empty">
          {{ t('escalation.noSteps') }}
        </div>
      </n-form>

      <template #action>
        <n-space>
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="handleSave">{{ t('common.save') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.steps-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.steps-title {
  font-weight: 600;
}
.step-card {
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 8px;
}
.step-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.step-delay-input {
  width: 120px;
}
.steps-empty {
  text-align: center;
  color: var(--sre-text-tertiary);
  padding: 16px;
}
</style>
