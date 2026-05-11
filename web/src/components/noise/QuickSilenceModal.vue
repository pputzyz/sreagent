<script setup lang="ts">
/**
 * QuickSilenceModal — 快速静默弹窗 (Phase 2.5)
 * 基于传入的 labels 预填 MuteRule，用户选择时长后一键创建。
 * 可在 Incident 详情、Alert v2 详情等页面复用。
 */
import { ref, computed, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { muteRuleApi } from '@/api'

const props = defineProps<{
  show: boolean
  labels?: Record<string, string>   // alert/incident labels to pre-fill
  title?: string                     // pre-filled rule name prefix
}>()

const emit = defineEmits<{
  (e: 'update:show', val: boolean): void
  (e: 'created'): void
}>()

const { t, locale } = useI18n()
const message = useMessage()
const saving = ref(false)

// Duration presets in minutes
const durationPresets = computed(() => [
  { label: t('channel.silence30m'), value: 30 },
  { label: t('channel.silence1h'),  value: 60 },
  { label: t('channel.silence2h'),  value: 120 },
  { label: t('channel.silence4h'),  value: 240 },
  { label: t('channel.silence8h'),  value: 480 },
  { label: t('channel.silence24h'), value: 1440 },
  { label: t('channel.silenceCustom'),  value: 0 },
])

const selectedDuration = ref(60)
const customMinutes = ref(60)
const ruleName = ref('')
const reason = ref('')

// Selected labels to match on (user can deselect unwanted ones)
const labelSelections = ref<{ key: string; value: string; selected: boolean }[]>([])

watch(() => props.show, (v) => {
  if (!v) return
  // Reset on open
  selectedDuration.value = 60
  customMinutes.value = 60
  reason.value = ''
  ruleName.value = (props.title ? t('channel.quickSilence') + ': ' + props.title : t('channel.quickSilence'))
  labelSelections.value = Object.entries(props.labels ?? {})
    // Exclude internal hints
    .filter(([k]) => !k.startsWith('_'))
    .map(([key, value]) => ({ key, value, selected: true }))
})

const effectiveMinutes = computed(() =>
  selectedDuration.value === 0 ? customMinutes.value : selectedDuration.value
)

const startTime = computed(() => {
  const now = new Date()
  return now.toISOString()
})

const endTime = computed(() => {
  const end = new Date(Date.now() + effectiveMinutes.value * 60_000)
  return end.toISOString()
})

async function create() {
  if (!ruleName.value.trim()) {
    message.warning(t('channel.silenceNameRequired'))
    return
  }
  saving.value = true
  try {
    // Build match_labels from selected labels
    const matchLabels: Record<string, string> = {}
    for (const item of labelSelections.value) {
      if (item.selected) {
        matchLabels[item.key] = item.value
      }
    }

    await muteRuleApi.create({
      name: ruleName.value,
      description: reason.value || t('channel.silenceCreated', { n: effectiveMinutes.value }),
      match_labels: matchLabels,
      start_time: String(Math.floor(Date.now() / 1000)),
      end_time: String(Math.floor((Date.now() + effectiveMinutes.value * 60_000) / 1000)),
      is_enabled: true,
      severities: '',
      rule_ids: '',
    })

    message.success(t('channel.silenceCreated', { n: effectiveMinutes.value }))
    emit('update:show', false)
    emit('created')
  } catch (e: any) {
    message.error(e?.message ?? t('channel.silenceCreateFailed'))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <n-modal
    :show="show"
    :title="t('channel.quickSilence')"
    preset="card"
    style="width:460px"
    :bordered="false"
    @update:show="$emit('update:show', $event)"
  >
    <n-form label-placement="top" size="small">

      <!-- Rule name -->
      <n-form-item :label="t('channel.silenceNameLabel')" required>
        <n-input v-model:value="ruleName" :placeholder="t('channel.silenceNamePlaceholder')" />
      </n-form-item>

      <!-- Duration selector -->
      <n-form-item :label="t('channel.silenceDurationLabel')">
        <n-radio-group v-model:value="selectedDuration">
          <n-space wrap>
            <n-radio
              v-for="p in durationPresets"
              :key="p.value"
              :value="p.value"
            >{{ p.label }}</n-radio>
          </n-space>
        </n-radio-group>
        <n-input-number
          v-if="selectedDuration === 0"
          v-model:value="customMinutes"
          :min="1"
          :max="43200"
          style="width:120px;margin-top:8px"
          :placeholder="t('channel.silenceCustomMinutesPlaceholder')"
        />
      </n-form-item>

      <!-- Time range display -->
      <n-form-item :label="t('channel.silenceEffectiveTime')">
        <n-space>
          <n-tag size="small">{{ new Date(startTime).toLocaleString(locale, { hour12: false }) }}</n-tag>
          <span style="color:var(--sre-text-secondary)">→</span>
          <n-tag size="small" type="warning">{{ new Date(endTime).toLocaleString(locale, { hour12: false }) }}</n-tag>
        </n-space>
      </n-form-item>

      <!-- Label matching -->
      <n-form-item v-if="labelSelections.length > 0" :label="t('channel.silenceMatchLabels')">
        <div class="label-list">
          <div
            v-for="item in labelSelections"
            :key="item.key"
            class="label-item"
            :class="{ selected: item.selected }"
            @click="item.selected = !item.selected"
          >
            <n-checkbox v-model:checked="item.selected" @click.stop />
            <span class="label-key">{{ item.key }}</span>
            <span class="label-eq">=</span>
            <span class="label-val">{{ item.value }}</span>
          </div>
        </div>
      </n-form-item>

      <!-- Reason -->
      <n-form-item :label="t('channel.silenceReasonLabel')">
        <n-input
          v-model:value="reason"
          type="textarea"
          :rows="2"
          :placeholder="t('channel.silenceReasonPlaceholder')"
        />
      </n-form-item>

    </n-form>

    <template #footer>
      <n-space justify="end">
        <n-button @click="$emit('update:show', false)">{{ t('common.cancel') }}</n-button>
        <n-button type="warning" :loading="saving" @click="create">
          {{ t('channel.silenceNow', { n: effectiveMinutes }) }}
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
.label-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 160px;
  overflow-y: auto;
}

.label-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 8px;
  border-radius: 6px;
  cursor: pointer;
  border: 1px solid var(--sre-border);
  background: var(--sre-bg-page);
  transition: background 0.15s, border-color 0.15s;
  font-family: monospace;
  font-size: 12px;
}

.label-item.selected {
  border-color: var(--sre-primary);
  background: var(--sre-primary-soft);
}

.label-key  { color: var(--sre-primary); font-weight: 600; }
.label-eq   { color: var(--sre-text-secondary); }
.label-val  { color: var(--sre-text-primary); }
</style>
