<script setup lang="ts">
import { ref, computed } from 'vue'
import { NSelect, NDatePicker, NButton, NSpace, NIcon } from 'naive-ui'
import { relativeTimeOptions } from '@/composables/useTimeRange'
import type { TimeRange } from '@/types/query'

const props = defineProps<{
  timeRange: TimeRange
  isRelative: boolean
  relativeDuration: string
}>()

const emit = defineEmits<{
  (e: 'setRelative', duration: string): void
  (e: 'setAbsolute', start: number, end: number): void
}>()

const showAbsolute = ref(false)
const absoluteRange = ref<[number, number]>([props.timeRange.start, props.timeRange.end])

const relativeOptions = relativeTimeOptions.map(o => ({
  label: o.label,
  value: o.value,
}))

function onRelativeChange(val: string) {
  emit('setRelative', val)
}

function onAbsoluteApply() {
  emit('setAbsolute', absoluteRange.value[0], absoluteRange.value[1])
  showAbsolute.value = false
}

function toggleMode() {
  if (props.isRelative) {
    absoluteRange.value = [props.timeRange.start, props.timeRange.end]
    showAbsolute.value = true
  } else {
    showAbsolute.value = false
    emit('setRelative', '1h')
  }
}

const displayLabel = computed(() => {
  if (props.isRelative) {
    const opt = relativeTimeOptions.find(o => o.value === props.relativeDuration)
    return opt?.label || props.relativeDuration
  }
  const s = new Date(props.timeRange.start).toLocaleString()
  const e = new Date(props.timeRange.end).toLocaleString()
  return `${s} — ${e}`
})
</script>

<template>
  <div class="time-range-picker">
    <NSpace size="small" align="center">
      <template v-if="!showAbsolute && isRelative">
        <NSelect
          :value="relativeDuration"
          :options="relativeOptions"
          size="small"
          style="width: 180px"
          @update:value="onRelativeChange"
        />
      </template>
      <template v-else-if="showAbsolute">
        <NDatePicker
          v-model:range-value="absoluteRange"
          type="datetimerange"
          size="small"
          style="width: 400px"
          @update:value="onAbsoluteApply"
        />
      </template>
      <template v-else>
        <span class="time-display">{{ displayLabel }}</span>
      </template>

      <NButton quaternary size="tiny" @click="toggleMode">
        {{ isRelative ? '🕐' : '⏱' }}
      </NButton>
    </NSpace>
  </div>
</template>

<style scoped>
.time-range-picker {
  display: inline-flex;
  align-items: center;
}
.time-display {
  font-size: 13px;
  color: var(--sre-text-secondary);
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
