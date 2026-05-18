<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { scheduleApi } from '@/api'
import type { OnCallShift, User } from '@/types'
import { getErrorMessage } from '@/utils/format'

const props = defineProps<{
  scheduleId: number | null
  users: User[]
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
  user_id: null as number | null,
  start_time: null as number | null,
  end_time: null as number | null,
  severity_filter: [] as string[],
  note: '',
})

const severityOptions = [
  { label: () => t('alert.critical'), value: 'critical' },
  { label: () => t('alert.warning'), value: 'warning' },
  { label: () => t('alert.info'), value: 'info' },
]

const userOptions = computed(() =>
  props.users.map(u => {
    const type = u.user_type && u.user_type !== 'human' ? ` [${u.user_type === 'bot' ? '\u{1F916}' : '\u{1F4E2}'}]` : ''
    return {
      label: (u.display_name || u.username) + type,
      value: u.id,
    }
  })
)

function openCreate(day?: Date, hour?: number) {
  editingId.value = null
  modalTitle.value = t('schedule.createShift')
  const startDate = day ? new Date(day) : new Date()
  startDate.setHours(hour ?? 9, 0, 0, 0)
  const endDate = new Date(startDate)
  endDate.setHours(startDate.getHours() + 8, 0, 0, 0)
  Object.assign(form, {
    user_id: null,
    start_time: startDate.getTime(),
    end_time: endDate.getTime(),
    severity_filter: [],
    note: '',
  })
  show.value = true
}

function openEdit(shift: OnCallShift) {
  editingId.value = shift.id
  modalTitle.value = t('schedule.editShift')
  const severities = shift.severity_filter ? shift.severity_filter.split(',').filter(Boolean) : []
  Object.assign(form, {
    user_id: shift.user_id,
    start_time: new Date(shift.start_time).getTime(),
    end_time: new Date(shift.end_time).getTime(),
    severity_filter: severities,
    note: shift.note || '',
  })
  show.value = true
}

async function handleSave() {
  if (!props.scheduleId) return
  if (!form.user_id) {
    message.warning(t('schedule.userRequired'))
    return
  }
  if (!form.start_time || !form.end_time) {
    message.warning(t('schedule.timeRequired'))
    return
  }

  saving.value = true
  try {
    const payload: Partial<OnCallShift> = {
      user_id: form.user_id,
      start_time: new Date(form.start_time).toISOString(),
      end_time: new Date(form.end_time).toISOString(),
      severity_filter: form.severity_filter.join(','),
      note: form.note,
    }
    if (editingId.value) {
      await scheduleApi.updateShift(props.scheduleId, editingId.value, payload)
      message.success(t('schedule.shiftUpdated'))
    } else {
      await scheduleApi.createShift(props.scheduleId, payload)
      message.success(t('schedule.shiftCreated'))
    }
    show.value = false
    emit('saved')
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function handleDelete() {
  if (!props.scheduleId || !editingId.value) return
  try {
    await scheduleApi.deleteShift(props.scheduleId, editingId.value)
    message.success(t('schedule.shiftDeleted'))
    show.value = false
    emit('saved')
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

defineExpose({ openCreate, openEdit })
</script>

<template>
  <n-modal v-model:show="show" preset="card" :title="modalTitle" style="width: 520px" :bordered="false">
    <n-form label-placement="top">
      <n-form-item :label="t('schedule.shiftUser')" required>
        <n-select
          v-model:value="form.user_id"
          :options="userOptions"
          :placeholder="t('schedule.selectUser')"
          filterable
        />
      </n-form-item>
      <n-grid :x-gap="12" :cols="2">
        <n-gi>
          <n-form-item :label="t('schedule.startTime')">
            <n-date-picker v-model:value="form.start_time" type="datetime" style="width: 100%" />
          </n-form-item>
        </n-gi>
        <n-gi>
          <n-form-item :label="t('schedule.endTime')">
            <n-date-picker v-model:value="form.end_time" type="datetime" style="width: 100%" />
          </n-form-item>
        </n-gi>
      </n-grid>
      <n-form-item :label="t('schedule.severityFilter')">
        <n-select
          v-model:value="form.severity_filter"
          :options="severityOptions"
          multiple
          :placeholder="t('schedule.allSeverities')"
        />
      </n-form-item>
      <n-form-item :label="t('schedule.note')">
        <n-input v-model:value="form.note" type="textarea" :rows="2" />
      </n-form-item>
    </n-form>
    <template #action>
      <n-space justify="end">
        <n-button v-if="editingId" type="error" quaternary @click="handleDelete">
          {{ t('common.delete') }}
        </n-button>
        <n-button @click="show = false">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="saving" @click="handleSave">
          {{ editingId ? t('common.update') : t('common.create') }}
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>
