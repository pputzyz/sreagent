<template>
  <div class="kv-editor">
    <div v-for="(item, idx) in modelValue" :key="idx" class="kv-row">
      <n-select
        v-if="keyOptions"
        :value="item.key || undefined"
        :options="getKeySelectOptions()"
        :placeholder="resolvedKeyPlaceholder"
        filterable
        clearable
        size="small"
        style="flex: 1"
        @update:value="(v: string) => { item.key = v || ''; emitUpdate(); $emit('keyChange', idx, v || '') }"
      />
      <n-input
        v-else
        v-model:value="item.key"
        :placeholder="resolvedKeyPlaceholder"
        size="small"
        @update:value="emitUpdate"
      />
      <n-select
        v-if="valueOptions"
        :value="item.value || undefined"
        :options="getValueSelectOptions()"
        :placeholder="resolvedValuePlaceholder"
        filterable
        clearable
        size="small"
        style="flex: 1"
        @update:value="(v: string) => { item.value = v || ''; emitUpdate() }"
      />
      <n-input
        v-else
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
import { NInput, NButton, NIcon, NAutoComplete, NSelect } from 'naive-ui'
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
  keyOptions?: string[]
  valueOptions?: string[]
}>(), {
  keyPlaceholder: '',
  valuePlaceholder: '',
  addLabel: '',
  keyOptions: undefined,
  valueOptions: undefined,
})

const resolvedKeyPlaceholder = computed(() => props.keyPlaceholder || t('common.key'))
const resolvedValuePlaceholder = computed(() => props.valuePlaceholder || t('common.value'))
const resolvedAddLabel = computed(() => props.addLabel || t('common.add'))

const emit = defineEmits<{
  'update:modelValue': [value: KVItem[]]
  'keyChange': [index: number, key: string]
  'keyFocus': [index: number]
}>()

function getKeyOptions(input: string) {
  if (!props.keyOptions) return []
  const q = input.toLowerCase()
  return props.keyOptions.filter(k => k.toLowerCase().includes(q)).map(k => ({ label: k, value: k }))
}

function getKeySelectOptions() {
  if (!props.keyOptions) return []
  return props.keyOptions.map(k => ({ label: k, value: k }))
}

function getValueOptions(input: string) {
  if (!props.valueOptions) return []
  const q = input.toLowerCase()
  return props.valueOptions.filter(v => v.toLowerCase().includes(q)).map(v => ({ label: v, value: v }))
}

function getValueSelectOptions() {
  if (!props.valueOptions) return []
  return props.valueOptions.map(v => ({ label: v, value: v }))
}

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
