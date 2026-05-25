<script setup lang="ts">
import { NButton, NInputNumber, NColorPicker, NIcon } from 'naive-ui'
import { TrashOutline } from '@vicons/ionicons5'
import type { ThresholdItem } from '@/types/dashboard'

const props = defineProps<{
  thresholds: ThresholdItem[]
}>()

const emit = defineEmits<{
  (e: 'update', thresholds: ThresholdItem[]): void
}>()

function addThreshold() {
  const sorted = [...props.thresholds].sort((a, b) => a.value - b.value)
  const lastVal = sorted.length > 0 ? sorted[sorted.length - 1].value : 0
  const updated = [...props.thresholds, { value: lastVal + 10, color: '#E6573E' }]
  emit('update', sortThresholds(updated))
}

function removeThreshold(index: number) {
  const updated = props.thresholds.filter((_, i) => i !== index)
  emit('update', updated)
}

function updateValue(index: number, val: number | null) {
  if (val == null) return
  const updated = props.thresholds.map((t, i) => i === index ? { ...t, value: val } : t)
  emit('update', sortThresholds(updated))
}

function updateColor(index: number, color: string) {
  const updated = props.thresholds.map((t, i) => i === index ? { ...t, color } : t)
  emit('update', updated)
}

function sortThresholds(items: ThresholdItem[]): ThresholdItem[] {
  return [...items].sort((a, b) => a.value - b.value)
}
</script>

<template>
  <div class="threshold-editor">
    <div
      v-for="(th, i) in thresholds"
      :key="i"
      class="threshold-row"
    >
      <NInputNumber
        :value="th.value"
        size="small"
        style="width: 120px"
        @update:value="(v: number | null) => updateValue(i, v)"
      />
      <NColorPicker
        :value="th.color"
        size="small"
        :show-alpha="false"
        style="width: 80px"
        @update:value="(v: string) => updateColor(i, v)"
      />
      <NButton
        quaternary
        size="tiny"
        type="error"
        @click="removeThreshold(i)"
      >
        <template #icon><NIcon :component="TrashOutline" /></template>
      </NButton>
    </div>
    <NButton dashed size="small" @click="addThreshold">
      + Add Threshold
    </NButton>
  </div>
</template>

<style scoped>
.threshold-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.threshold-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
