<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NSelect, NInput, NInputNumber, NColorPicker, NIcon } from 'naive-ui'
import { TrashOutline } from '@vicons/ionicons5'
import type { ValueMapping } from '@/types/dashboard'

const { t } = useI18n()

const props = defineProps<{
  valueMappings: ValueMapping[]
}>()

const emit = defineEmits<{
  (e: 'update', mappings: ValueMapping[]): void
}>()

const typeOptions = computed(() => [
  { label: t('dashboardEditor.mappingTypeValue'), value: 'value' },
  { label: t('dashboardEditor.mappingTypeRange'), value: 'range' },
  { label: t('dashboardEditor.mappingTypeSpecial'), value: 'special' },
])

const specialOptions = computed(() => [
  { label: t('dashboardEditor.mappingTypeNull'), value: 'null' },
  { label: 'NaN', value: 'NaN' },
  { label: t('dashboardEditor.mappingTypeEmpty'), value: 'empty' },
  { label: t('dashboardEditor.mappingTypeNoData'), value: 'no_data' },
])

function addMapping() {
  const updated = [...props.valueMappings, {
    type: 'value' as const,
    match: { value: '' },
    result: { text: '', color: '' },
  }]
  emit('update', updated)
}

function removeMapping(index: number) {
  emit('update', props.valueMappings.filter((_, i) => i !== index))
}

function updateType(index: number, type: 'value' | 'range' | 'special') {
  const updated = props.valueMappings.map((m, i) => {
    if (i !== index) return m
    const match = type === 'range' ? { from: 0, to: 100 } : { value: '' }
    return { ...m, type, match }
  })
  emit('update', updated)
}

function updateMatchValue(index: number, val: string) {
  const updated = props.valueMappings.map((m, i) => {
    if (i !== index) return m
    return { ...m, match: { ...m.match, value: val } }
  })
  emit('update', updated)
}

function updateMatchFrom(index: number, val: number | null) {
  const updated = props.valueMappings.map((m, i) => {
    if (i !== index) return m
    return { ...m, match: { ...m.match, from: val ?? 0 } }
  })
  emit('update', updated)
}

function updateMatchTo(index: number, val: number | null) {
  const updated = props.valueMappings.map((m, i) => {
    if (i !== index) return m
    return { ...m, match: { ...m.match, to: val ?? 0 } }
  })
  emit('update', updated)
}

function updateResultText(index: number, text: string) {
  const updated = props.valueMappings.map((m, i) => {
    if (i !== index) return m
    return { ...m, result: { ...m.result, text } }
  })
  emit('update', updated)
}

function updateResultColor(index: number, color: string) {
  const updated = props.valueMappings.map((m, i) => {
    if (i !== index) return m
    return { ...m, result: { ...m.result, color } }
  })
  emit('update', updated)
}
</script>

<template>
  <div class="value-mapping-editor">
    <div
      v-for="(mapping, i) in valueMappings"
      :key="i"
      class="mapping-row"
    >
      <NSelect
        :value="mapping.type"
        :options="typeOptions"
        size="small"
        style="width: 100px"
        @update:value="(v: string) => updateType(i, v as 'value' | 'range' | 'special')"
      />

      <!-- Match input depends on type -->
      <template v-if="mapping.type === 'value'">
        <NInput
          :value="mapping.match?.value ?? ''"
          size="small"
          :placeholder="t('dashboardEditor.matchValue')"
          style="width: 140px"
          @update:value="(v: string) => updateMatchValue(i, v)"
        />
      </template>
      <template v-else-if="mapping.type === 'range'">
        <NInputNumber
          :value="mapping.match?.from ?? 0"
          size="small"
          :placeholder="t('dashboardEditor.from')"
          style="width: 90px"
          @update:value="(v: number | null) => updateMatchFrom(i, v)"
        />
        <span class="range-sep">-</span>
        <NInputNumber
          :value="mapping.match?.to ?? 100"
          size="small"
          :placeholder="t('dashboardEditor.to')"
          style="width: 90px"
          @update:value="(v: number | null) => updateMatchTo(i, v)"
        />
      </template>
      <template v-else>
        <NSelect
          :value="mapping.match?.value ?? ''"
          :options="specialOptions"
          size="small"
          style="width: 140px"
          @update:value="(v: string) => updateMatchValue(i, v)"
        />
      </template>

      <NInput
        :value="mapping.result?.text ?? ''"
        size="small"
        :placeholder="t('dashboardEditor.displayText')"
        style="width: 120px"
        @update:value="(v: string) => updateResultText(i, v)"
      />

      <NColorPicker
        :value="mapping.result?.color ?? '#ffffff'"
        size="small"
        :show-alpha="false"
        style="width: 70px"
        @update:value="(v: string) => updateResultColor(i, v)"
      />

      <NButton
        quaternary
        size="tiny"
        type="error"
        @click="removeMapping(i)"
      >
        <template #icon><NIcon :component="TrashOutline" /></template>
      </NButton>
    </div>

    <NButton dashed size="small" @click="addMapping">
      {{ t('dashboardEditor.addMapping') }}
    </NButton>
  </div>
</template>

<style scoped>
.value-mapping-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.mapping-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.range-sep {
  color: var(--sre-text-tertiary);
  font-size: 12px;
}
</style>
