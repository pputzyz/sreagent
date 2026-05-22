<template>
  <div class="cron-input">
    <n-select
      :value="presetValue"
      :options="presetOptions"
      :placeholder="t('cronInput.selectPreset')"
      clearable
      @update:value="onPresetChange"
      style="margin-bottom: 8px"
    />
    <n-input
      :value="modelValue"
      :placeholder="t('cronInput.placeholder')"
      @update:value="onInputChange"
    />
    <div v-if="nextRuns.length > 0" class="next-runs">
      <n-text depth="3" style="font-size: 12px">{{ t('cronInput.nextRunTime') }}:</n-text>
      <n-text v-for="(t, i) in nextRuns" :key="i" style="font-size: 12px; display: block">
        {{ t }}
      </n-text>
    </div>
    <n-text v-if="error" type="error" style="font-size: 12px">{{ error }}</n-text>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { NSelect, NInput, NText } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { inspectionApi } from '@/api/inspection'

const { t } = useI18n()

interface Props {
  modelValue: string
}

const props = defineProps<Props>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const presetOptions = computed(() => [
  { label: t('cronInput.presets.daily9am'), value: '0 0 9 * * *' },
  { label: t('cronInput.presets.daily9am18pm'), value: '0 0 9,18 * * *' },
  { label: t('cronInput.presets.hourly'), value: '0 0 * * * *' },
  { label: t('cronInput.presets.every6h'), value: '0 0 */6 * * *' },
  { label: t('cronInput.presets.monday9am'), value: '0 0 9 * * 1' },
  { label: t('cronInput.presets.weekdays9am'), value: '0 0 9 * * 1-5' },
  { label: t('cronInput.presets.monthly1st9am'), value: '0 0 9 1 * *' },
])

const presetValue = ref<string | null>(null)
const nextRuns = ref<string[]>([])
const error = ref('')

function onPresetChange(val: string | null) {
  if (val) {
    emit('update:modelValue', val)
  }
}

function onInputChange(val: string) {
  emit('update:modelValue', val)
}

let debounceTimer: ReturnType<typeof setTimeout> | null = null

watch(() => props.modelValue, (val) => {
  // Match preset
  const match = presetOptions.value.find(p => p.value === val)
  presetValue.value = match ? match.value : null

  // Validate with debounce
  if (debounceTimer) clearTimeout(debounceTimer)
  if (!val || val.trim() === '') {
    nextRuns.value = []
    error.value = ''
    return
  }
  debounceTimer = setTimeout(async () => {
    try {
      const res = await inspectionApi.validateCron(val)
      if (res.data.data?.valid) {
        nextRuns.value = res.data.data.next_runs.map((t: string) => new Date(t).toLocaleString())
        error.value = ''
      } else {
        nextRuns.value = []
        error.value = t('cronInput.invalidExpr')
      }
    } catch {
      nextRuns.value = []
      error.value = t('cronInput.validateFailed')
    }
  }, 500)
}, { immediate: true })
</script>

<style scoped>
.cron-input {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.next-runs {
  margin-top: 4px;
}
</style>
