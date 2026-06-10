<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NButton, NSelect, NInput, NSwitch, NIcon, NSpace } from 'naive-ui'
import { TrashOutline } from '@vicons/ionicons5'

const { t } = useI18n()
import PromQLEditor from '@/components/query/PromQLEditor.vue'
import type { PanelTarget } from '@/types/dashboard'
import type { DataSource } from '@/types'

const props = defineProps<{
  targets: PanelTarget[]
  datasources: DataSource[]
}>()

const emit = defineEmits<{
  (e: 'update', targets: PanelTarget[]): void
}>()

const datasourceOptions = props.datasources.map(ds => ({
  label: ds.name,
  value: ds.id,
}))

const yAxisOptions = [
  { label: 'Left', value: 'left' },
  { label: 'Right', value: 'right' },
]

function refIdForIndex(i: number): string {
  return String.fromCharCode(65 + i)
}

function addTarget() {
  const last = props.targets.length > 0 ? props.targets[props.targets.length - 1] : null
  const updated = [...props.targets, {
    datasourceId: last?.datasourceId ?? 0,
    expression: '',
    legendFormat: '',
    refId: refIdForIndex(props.targets.length),
    hide: false,
    yAxisPosition: 'left' as const,
  }]
  emit('update', updated)
}

function removeTarget(index: number) {
  const updated = props.targets.filter((_, i) => i !== index).map((t, i) => ({
    ...t,
    refId: refIdForIndex(i),
  }))
  emit('update', updated)
}

function updateTargetField<K extends keyof PanelTarget>(index: number, key: K, val: PanelTarget[K]) {
  const updated = props.targets.map((t, i) => i === index ? { ...t, [key]: val } : t)
  emit('update', updated)
}
</script>

<template>
  <div class="panel-editor-query">
    <div
      v-for="(target, i) in targets"
      :key="i"
      class="query-row"
      :class="{ hidden: target.hide }"
    >
      <div class="query-row-header">
        <span class="query-ref">{{ refIdForIndex(i) }}</span>
        <NSpace size="small">
          <NSwitch
            :value="!target.hide"
            size="small"
            @update:value="(v: boolean) => updateTargetField(i, 'hide', !v)"
          />
          <NButton
            v-if="targets.length > 1"
            quaternary
            size="tiny"
            type="error"
            @click="removeTarget(i)"
          >
            <template #icon><NIcon :component="TrashOutline" /></template>
          </NButton>
        </NSpace>
      </div>

      <div class="query-row-controls">
        <NSelect
          :value="target.datasourceId"
          :options="datasourceOptions"
          size="small"
          :placeholder="t('dashboardEditor.datasource')"
          style="width: 180px"
          @update:value="(v: number) => updateTargetField(i, 'datasourceId', v)"
        />
        <NSelect
          :value="target.yAxisPosition ?? 'left'"
          :options="yAxisOptions"
          size="small"
          style="width: 90px"
          @update:value="(v: 'left' | 'right') => updateTargetField(i, 'yAxisPosition', v)"
        />
      </div>

      <div class="query-row-editor">
        <PromQLEditor
          :model-value="target.expression"
          :datasource-id="target.datasourceId"
          :placeholder="t('dashboardEditor.promqlExpression')"
          @update:model-value="(v: string) => updateTargetField(i, 'expression', v)"
        />
      </div>

      <NInput
        :value="target.legendFormat"
        size="small"
        :placeholder="t('dashboardEditor.legendPlaceholder')"
        @update:value="(v: string) => updateTargetField(i, 'legendFormat', v)"
      />
    </div>

    <NButton dashed size="small" @click="addTarget">
      {{ t('dashboardEditor.addQuery') }}
    </NButton>
  </div>
</template>

<style scoped>
.panel-editor-query {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 0;
}
.query-row {
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  padding: 10px;
  background: var(--sre-bg-sunken);
  transition: opacity 0.2s;
}
.query-row.hidden {
  opacity: 0.5;
}
.query-row-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.query-ref {
  font-size: 12px;
  font-weight: 700;
  color: var(--sre-text-secondary);
}
.query-row-controls {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
.query-row-editor {
  margin-bottom: 8px;
}
</style>
