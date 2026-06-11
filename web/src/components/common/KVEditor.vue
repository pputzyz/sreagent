<template>
  <div class="kv-editor">
    <div v-for="(item, idx) in modelValue" :key="rowId(item)" class="kv-row">
      <n-select
        v-if="keyOptions"
        :value="item.key || undefined"
        :options="getKeySelectOptions()"
        :placeholder="resolvedKeyPlaceholder"
        filterable
        clearable
        size="small"
        style="flex: 1"
        :status="keyStatus(idx).status"
        @update:value="(v: string) => onFieldChange(idx, 'key', v || '')"
      />
      <n-input
        v-else
        :value="item.key"
        :placeholder="resolvedKeyPlaceholder"
        size="small"
        :status="keyStatus(idx).status"
        :maxlength="maxKeyLength"
        @update:value="(v: string) => onFieldChange(idx, 'key', v)"
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
        @update:value="(v: string) => onFieldChange(idx, 'value', v || '')"
      />
      <n-input
        v-else
        :value="item.value"
        :placeholder="resolvedValuePlaceholder"
        size="small"
        :maxlength="maxValueLength"
        @update:value="(v: string) => onFieldChange(idx, 'value', v)"
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

let _nextKvId = 0
const _rowIdMap = new WeakMap<object, number>()
function rowId(row: object): number {
  let id = _rowIdMap.get(row)
  if (id === undefined) { id = ++_nextKvId; _rowIdMap.set(row, id) }
  return id
}
/** Carry the row id over to a replacement object so :key stays stable.
 *  Without this, every keystroke (which creates a new object via spread)
 *  would get a NEW id -> Vue rebuilds the row -> the input loses focus. */
function inheritRowId(oldRow: object, newRow: object) {
  const id = _rowIdMap.get(oldRow)
  if (id !== undefined) _rowIdMap.set(newRow, id)
}

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
  maxKeyLength?: number
  maxValueLength?: number
  disallowDuplicateKeys?: boolean
}>(), {
  keyPlaceholder: '',
  valuePlaceholder: '',
  addLabel: '',
  keyOptions: undefined,
  valueOptions: undefined,
  maxKeyLength: 128,
  maxValueLength: 512,
  disallowDuplicateKeys: true,
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
  // Prevent adding when the last row has an empty key
  const last = props.modelValue[props.modelValue.length - 1]
  if (last && !last.key.trim()) return
  const updated = [...props.modelValue, { key: '', value: '' }]
  emit('update:modelValue', updated)
}

function removeItem(index: number) {
  const updated = props.modelValue.filter((_, i) => i !== index)
  emit('update:modelValue', updated)
}

function onFieldChange(index: number, field: 'key' | 'value', val: string) {
  const updated = props.modelValue.map((item, i) => {
    if (i !== index) return item
    const next = { ...item, [field]: val }
    inheritRowId(item, next)
    return next
  })
  emit('update:modelValue', updated)
  if (field === 'key') emit('keyChange', index, val)
}

/** Returns validation feedback for a given row index. */
function keyStatus(idx: number): { status?: 'error'; feedback?: string } {
  const item = props.modelValue[idx]
  if (!item) return {}
  const key = item.key.trim()
  if (!key) return { status: 'error', feedback: '' }
  if (props.maxKeyLength && key.length > props.maxKeyLength) return { status: 'error', feedback: '' }
  if (props.disallowDuplicateKeys) {
    const dup = props.modelValue.some((other, i) => i !== idx && other.key.trim() === key)
    if (dup) return { status: 'error', feedback: '' }
  }
  return {}
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
