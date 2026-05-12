<template>
  <div class="kv-editor">
    <div v-for="(item, idx) in modelValue" :key="idx" class="kv-row">
      <n-input
        v-model:value="item.key"
        :placeholder="resolvedKeyPlaceholder"
        size="small"
        @update:value="emitUpdate"
      />
      <n-input
        v-model:value="item.value"
        :placeholder="resolvedValuePlaceholder"
        size="small"
        @update:value="emitUpdate"
      />
      <n-button size="small" quaternary type="error" @click="removeItem(idx)">
        <template #icon><n-icon :component="CloseOutline" /></template>
      </n-button>
    </div>
    <n-button dashed size="small" @click="addItem">
      <template #icon><n-icon :component="AddOutline" /></template>
      {{ resolvedAddLabel }}
    </n-button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NInput, NButton, NIcon } from 'naive-ui'
import { AddOutline, CloseOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

export interface KVItem {
  key: string
  value: string
}

const props = withDefaults(defineProps<{
  modelValue: KVItem[]
  keyPlaceholder?: string
  valuePlaceholder?: string
  addLabel?: string
}>(), {
  keyPlaceholder: '',
  valuePlaceholder: '',
  addLabel: '',
})

const resolvedKeyPlaceholder = computed(() => props.keyPlaceholder || t('common.key'))
const resolvedValuePlaceholder = computed(() => props.valuePlaceholder || t('common.value'))
const resolvedAddLabel = computed(() => props.addLabel || t('common.add'))

const emit = defineEmits<{
  'update:modelValue': [value: KVItem[]]
}>()

function addItem() {
  const updated = [...props.modelValue, { key: '', value: '' }]
  emit('update:modelValue', updated)
}

function removeItem(index: number) {
  const updated = props.modelValue.filter((_, i) => i !== index)
  emit('update:modelValue', updated)
}

function emitUpdate() {
  emit('update:modelValue', [...props.modelValue])
}
</script>

<style scoped>
.kv-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.kv-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.kv-row .n-input {
  flex: 1;
}
</style>
