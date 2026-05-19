<script setup lang="ts">
/**
 * SnoozeModal — snooze an incident for a preset or custom duration.
 * Extracted from incident/Detail.vue (FlashCat Phase 6).
 */
import { ref, computed, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { incidentApi } from '@/api'
import { getErrorMessage } from '@/utils/format'

const props = defineProps<{
  show: boolean
  incidentId: number
}>()

const emit = defineEmits<{
  (e: 'update:show', val: boolean): void
  (e: 'done'): void
}>()

const { t } = useI18n()
const message = useMessage()

const triggerEl = ref<HTMLElement | null>(null)

watch(() => props.show, (v) => {
  if (v) triggerEl.value = document.activeElement as HTMLElement
})

function handleAfterLeave() {
  triggerEl.value?.focus()
}

const loading = ref(false)
const duration = ref<number | null>(null)
const customUntil = ref('')

const presets = computed(() => [
  { label: '15m', minutes: 15 },
  { label: '30m', minutes: 30 },
  { label: '1h', minutes: 60 },
  { label: '2h', minutes: 120 },
  { label: '4h', minutes: 240 },
  { label: t('query.timeCustom'), minutes: -1 },
])

async function doSnooze() {
  let until: string
  if (duration.value === -1) {
    if (!customUntil.value) { message.warning(t('incident.selectSnoozeEnd')); return }
    until = new Date(customUntil.value).toISOString()
  } else if (duration.value) {
    const d = new Date()
    d.setMinutes(d.getMinutes() + duration.value)
    until = d.toISOString()
  } else {
    message.warning(t('incident.selectSnoozeDuration')); return
  }
  loading.value = true
  try {
    await incidentApi.snooze(props.incidentId, until)
    message.success(t('incident.snoozeSuccess'))
    emit('update:show', false)
    duration.value = null
    customUntil.value = ''
    emit('done')
  } catch (e: unknown) { message.error(getErrorMessage(e) || t('incident.opFailed')) } finally { loading.value = false }
}
</script>

<template>
  <n-modal
    :show="show"
    :title="t('incident.snoozeIncident')"
    preset="card"
    class="snooze-modal"
    :bordered="false"
    @update:show="emit('update:show', $event)"
    @after-leave="handleAfterLeave"
  >
    <div class="snooze-presets">
      <button
        v-for="p in presets" :key="p.minutes"
        class="preset-btn" :class="{ active: duration === p.minutes }"
        @click="duration = p.minutes"
      >{{ p.label }}</button>
    </div>
    <div v-if="duration === -1" class="custom-picker">
      <n-date-picker
        v-model:formatted-value="customUntil"
        type="datetime"
        :is-date-disabled="(ts: number) => ts < Date.now()"
        class="custom-datepicker"
      />
    </div>
    <template #footer>
      <n-space justify="end">
        <n-button @click="emit('update:show', false)">{{ t('incident.cancelBtn') }}</n-button>
        <n-button type="primary" :loading="loading" @click="doSnooze">{{ t('incident.confirmSnooze') }}</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
.snooze-modal {
  width: 420px;
}

.snooze-presets {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.preset-btn {
  background: var(--sre-bg-elevated);
  border: var(--sre-hairline);
  color: var(--sre-text-secondary);
  font-family: inherit;
  font-size: 12px;
  padding: 6px 12px;
  border-radius: var(--sre-radius-sm);
  cursor: pointer;
  transition: all 120ms ease;
}

.preset-btn:hover {
  color: var(--sre-text-primary);
  border-color: var(--sre-border-strong);
}

.preset-btn.active {
  background: var(--sre-primary-soft);
  border-color: var(--sre-primary);
  color: var(--sre-primary);
}

.custom-picker {
  margin-top: 12px;
}

.custom-datepicker {
  width: 100%;
}
</style>
