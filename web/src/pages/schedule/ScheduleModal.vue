<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { scheduleApi } from '@/api'
import type { Schedule, Team } from '@/types'
import { getErrorMessage } from '@/utils/format'

const props = defineProps<{
  teams: Team[]
}>()

const emit = defineEmits<{
  saved: []
}>()

const message = useMessage()
const { t } = useI18n()

const show = ref(false)
const modalTitle = ref('')
const editingId = ref<number | null>(null)
const saving = ref(false)

const form = reactive({
  name: '',
  team_id: undefined as number | undefined,
  description: '',
  rotation_type: 'daily' as 'daily' | 'weekly' | 'custom',
  timezone: 'Asia/Shanghai',
  handoff_time: '09:00',
  handoff_day: 1,
  rotation_period_days: 1,
  severity_filter: '',
  is_enabled: true,
})

const rotationOptions = [
  { label: () => t('schedule.daily'), value: 'daily' },
  { label: () => t('schedule.weekly'), value: 'weekly' },
  { label: () => t('schedule.custom'), value: 'custom' },
]

const dayOfWeekOptions = [
  { label: () => t('schedule.monday'), value: 1 },
  { label: () => t('schedule.tuesday'), value: 2 },
  { label: () => t('schedule.wednesday'), value: 3 },
  { label: () => t('schedule.thursday'), value: 4 },
  { label: () => t('schedule.friday'), value: 5 },
  { label: () => t('schedule.saturday'), value: 6 },
  { label: () => t('schedule.sunday'), value: 0 },
]

const timezoneOptions = [
  { label: 'Asia/Shanghai (CST)', value: 'Asia/Shanghai' },
  { label: 'UTC', value: 'UTC' },
  { label: 'America/New_York (EST)', value: 'America/New_York' },
  { label: 'America/Los_Angeles (PST)', value: 'America/Los_Angeles' },
  { label: 'Europe/London (GMT)', value: 'Europe/London' },
  { label: 'Asia/Tokyo (JST)', value: 'Asia/Tokyo' },
]

const teamOptions = computed(() =>
  props.teams.map(tm => ({ label: tm.name, value: tm.id }))
)

function openCreate() {
  editingId.value = null
  modalTitle.value = t('schedule.create')
  Object.assign(form, {
    name: '',
    team_id: undefined,
    description: '',
    rotation_type: 'daily',
    timezone: 'Asia/Shanghai',
    handoff_time: '09:00',
    handoff_day: 1,
    rotation_period_days: 1,
    severity_filter: '',
    is_enabled: true,
  })
  show.value = true
}

function openEdit(s: Schedule) {
  editingId.value = s.id
  modalTitle.value = t('schedule.edit')
  Object.assign(form, {
    name: s.name,
    team_id: s.team_id,
    description: s.description,
    rotation_type: s.rotation_type,
    timezone: s.timezone,
    handoff_time: s.handoff_time,
    handoff_day: s.handoff_day,
    rotation_period_days: s.rotation_period_days ?? 1,
    severity_filter: s.severity_filter || '',
    is_enabled: s.is_enabled,
  })
  show.value = true
}

async function handleSave() {
  if (!form.name.trim()) {
    message.warning(t('schedule.nameRequired'))
    return
  }

  saving.value = true
  try {
    const payload = { ...form }
    if (editingId.value) {
      await scheduleApi.update(editingId.value, payload)
      message.success(t('schedule.scheduleUpdated'))
    } else {
      await scheduleApi.create(payload)
      message.success(t('schedule.scheduleCreated'))
    }
    show.value = false
    emit('saved')
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

defineExpose({ openCreate, openEdit })
</script>

<template>
  <n-modal v-model:show="show" preset="card" :title="modalTitle" style="width: 560px" :bordered="false">
    <n-form label-placement="top">
      <n-form-item :label="t('common.name')" required>
        <n-input v-model:value="form.name" :placeholder="t('scheduleMgmt.namePlaceholder')" />
      </n-form-item>
      <n-form-item :label="t('schedule.team')">
        <n-select
          v-model:value="form.team_id"
          :options="teamOptions"
          :placeholder="t('schedule.selectTeamOptional')"
          clearable
        />
      </n-form-item>
      <n-form-item :label="t('common.description')">
        <n-input v-model:value="form.description" type="textarea" :rows="2" />
      </n-form-item>
      <n-grid :x-gap="12" :cols="2">
        <n-gi>
          <n-form-item :label="t('schedule.rotationType')">
            <n-select v-model:value="form.rotation_type" :options="rotationOptions" />
          </n-form-item>
        </n-gi>
        <n-gi>
          <n-form-item :label="t('schedule.timezone')">
            <n-select v-model:value="form.timezone" :options="timezoneOptions" filterable />
          </n-form-item>
        </n-gi>
      </n-grid>
      <n-grid :x-gap="12" :cols="2">
        <n-gi>
          <n-form-item :label="t('schedule.handoffTime')">
            <n-input v-model:value="form.handoff_time" :placeholder="t('scheduleMgmt.handoffTimePlaceholder')" />
          </n-form-item>
        </n-gi>
        <n-gi>
          <n-form-item :label="t('schedule.severityFilter')">
            <n-input v-model:value="form.severity_filter" :placeholder="t('scheduleMgmt.severityFilterPlaceholder')" />
          </n-form-item>
        </n-gi>
      </n-grid>
      <n-form-item v-if="form.rotation_type === 'weekly'" :label="t('schedule.handoffDay') || 'Handoff Day'">
        <n-select v-model:value="form.handoff_day" :options="dayOfWeekOptions" />
      </n-form-item>
      <n-form-item v-if="form.rotation_type === 'custom'" :label="t('schedule.rotationPeriodDays') || 'Rotation Period (days)'">
        <n-input-number v-model:value="form.rotation_period_days" :min="1" :max="365" style="width: 100%" />
      </n-form-item>
      <n-form-item :label="t('common.enabled')">
        <n-switch v-model:value="form.is_enabled" />
      </n-form-item>
    </n-form>
    <template #action>
      <n-space justify="end">
        <n-button @click="show = false">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="saving" @click="handleSave">
          {{ editingId ? t('common.update') : t('common.create') }}
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>
