<script setup lang="ts">
import { NInput, NSelect, NSwitch, NFormItem } from 'naive-ui'
import type { PanelType } from '@/types/dashboard'

const props = defineProps<{
  title: string
  description: string
  type: PanelType
  transparent: boolean
  repeatByVariable: string | undefined
  variableOptions: { label: string; value: string }[]
}>()

const emit = defineEmits<{
  (e: 'update:title', val: string): void
  (e: 'update:description', val: string): void
  (e: 'update:type', val: PanelType): void
  (e: 'update:transparent', val: boolean): void
  (e: 'update:repeatByVariable', val: string | undefined): void
}>()

const panelTypeOptions = [
  { label: 'Time Series', value: 'timeseries' },
  { label: 'Stat', value: 'stat' },
  { label: 'Gauge', value: 'gauge' },
  { label: 'Bar', value: 'bar' },
  { label: 'Pie', value: 'pie' },
  { label: 'Table', value: 'table' },
  { label: 'Text', value: 'text' },
  { label: 'Row', value: 'row' },
]
</script>

<template>
  <div class="panel-editor-general">
    <NFormItem label="Title" label-placement="left">
      <NInput
        :value="title"
        placeholder="Panel title"
        @update:value="(v: string) => emit('update:title', v)"
      />
    </NFormItem>

    <NFormItem label="Description" label-placement="left">
      <NInput
        :value="description"
        type="textarea"
        placeholder="Optional description"
        :rows="2"
        @update:value="(v: string) => emit('update:description', v)"
      />
    </NFormItem>

    <NFormItem label="Type" label-placement="left">
      <NSelect
        :value="type"
        :options="panelTypeOptions"
        @update:value="(v: PanelType) => emit('update:type', v)"
      />
    </NFormItem>

    <NFormItem label="Transparent" label-placement="left">
      <NSwitch
        :value="transparent"
        @update:value="(v: boolean) => emit('update:transparent', v)"
      />
    </NFormItem>

    <NFormItem v-if="variableOptions.length > 0" label="Repeat" label-placement="left">
      <NSelect
        :value="repeatByVariable ?? undefined"
        :options="variableOptions"
        clearable
        placeholder="Repeat by variable"
        @update:value="(v: string | undefined) => emit('update:repeatByVariable', v)"
      />
    </NFormItem>
  </div>
</template>

<style scoped>
.panel-editor-general {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px 0;
}
</style>
