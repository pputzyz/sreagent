<template>
  <div class="cron-input">
    <n-select
      :value="presetValue"
      :options="presetOptions"
      placeholder="选择预设或自定义"
      clearable
      @update:value="onPresetChange"
      style="margin-bottom: 8px"
    />
    <n-input
      :value="modelValue"
      placeholder="秒 分 时 日 月 周 (例: 0 0 9 * * 1-5)"
      @update:value="onInputChange"
    />
    <div v-if="nextRuns.length > 0" class="next-runs">
      <n-text depth="3" style="font-size: 12px">最近触发时间:</n-text>
      <n-text v-for="(t, i) in nextRuns" :key="i" style="font-size: 12px; display: block">
        {{ t }}
      </n-text>
    </div>
    <n-text v-if="error" type="error" style="font-size: 12px">{{ error }}</n-text>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { NSelect, NInput, NText } from 'naive-ui'
import { inspectionApi } from '@/api/inspection'

interface Props {
  modelValue: string
}

const props = defineProps<Props>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const presetOptions = [
  { label: '每天 9:00', value: '0 0 9 * * *' },
  { label: '每天 9:00 和 18:00', value: '0 0 9,18 * * *' },
  { label: '每小时', value: '0 0 * * * *' },
  { label: '每 6 小时', value: '0 0 */6 * * *' },
  { label: '每周一 9:00', value: '0 0 9 * * 1' },
  { label: '工作日 9:00', value: '0 0 9 * * 1-5' },
  { label: '每月 1 号 9:00', value: '0 0 9 1 * *' },
]

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
  const match = presetOptions.find(p => p.value === val)
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
        nextRuns.value = res.data.data.next_runs.map(t => new Date(t).toLocaleString())
        error.value = ''
      } else {
        nextRuns.value = []
        error.value = '无效的 cron 表达式'
      }
    } catch {
      nextRuns.value = []
      error.value = '校验失败'
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
