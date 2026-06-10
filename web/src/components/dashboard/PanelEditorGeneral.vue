<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NInput, NSelect, NSwitch, NFormItem } from 'naive-ui'
import type { PanelType } from '@/types/dashboard'

const { t } = useI18n()

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

const panelTypeOptions = computed(() => [
  { label: t('dashboardEditor.panelTypeTimeSeries'), value: 'timeseries' },
  { label: t('dashboardEditor.panelTypeStat'), value: 'stat' },
  { label: t('dashboardEditor.panelTypeGauge'), value: 'gauge' },
  { label: t('dashboardEditor.panelTypeBar'), value: 'bar' },
  { label: t('dashboardEditor.panelTypePie'), value: 'pie' },
  { label: t('dashboardEditor.panelTypeTable'), value: 'table' },
  { label: t('dashboardEditor.panelTypeText'), value: 'text' },
  { label: t('dashboardEditor.panelTypeRow'), value: 'row' },
])
</script>

<template>
  <div class="panel-editor-general">
    <NFormItem :label="t('dashboardEditor.title')" label-placement="left">
      <NInput
        :value="title"
        :placeholder="t('dashboardEditor.panelTitle')"
        @update:value="(v: string) => emit('update:title', v)"
      />
    </NFormItem>

    <NFormItem :label="t('dashboardEditor.description')" label-placement="left">
      <NInput
        :value="description"
        type="textarea"
        :placeholder="t('dashboardEditor.optionalDescription')"
        :rows="2"
        @update:value="(v: string) => emit('update:description', v)"
      />
    </NFormItem>

    <NFormItem :label="t('dashboardEditor.type')" label-placement="left">
      <NSelect
        :value="type"
        :options="panelTypeOptions"
        @update:value="(v: PanelType) => emit('update:type', v)"
      />
    </NFormItem>

    <NFormItem :label="t('dashboardEditor.transparent')" label-placement="left">
      <NSwitch
        :value="transparent"
        @update:value="(v: boolean) => emit('update:transparent', v)"
      />
    </NFormItem>

    <NFormItem v-if="variableOptions.length > 0" :label="t('dashboardEditor.repeat')" label-placement="left">
      <NSelect
        :value="repeatByVariable ?? undefined"
        :options="variableOptions"
        clearable
        :placeholder="t('dashboardEditor.repeatByVariable')"
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
