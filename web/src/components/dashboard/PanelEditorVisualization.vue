<script setup lang="ts">
import { NRadioGroup, NRadioButton, NSlider, NSwitch, NInputNumber, NInput, NFormItem } from 'naive-ui'
import type { PanelOptions, ThresholdItem, ValueMapping, PanelType } from '@/types/dashboard'
import PanelEditorThresholds from './PanelEditorThresholds.vue'
import PanelEditorValueMapping from './PanelEditorValueMapping.vue'

const props = defineProps<{
  type: PanelType
  options: PanelOptions
}>()

const emit = defineEmits<{
  (e: 'update', options: PanelOptions): void
}>()

function patch(partial: Partial<PanelOptions>) {
  emit('update', { ...props.options, ...partial })
}

const drawStyleOptions = [
  { label: 'Line', value: 'line' },
  { label: 'Bars', value: 'bars' },
  { label: 'Points', value: 'points' },
]

const stackingOptions = [
  { label: 'None', value: 'none' },
  { label: 'Normal', value: 'normal' },
]

const legendPositionOptions = [
  { label: 'Bottom', value: 'bottom' },
  { label: 'Right', value: 'right' },
  { label: 'Hidden', value: 'hidden' },
]

const colorModeOptions = [
  { label: 'Value', value: 'value' },
  { label: 'Background', value: 'background' },
]

const graphModeOptions = [
  { label: 'None', value: 'none' },
  { label: 'Area', value: 'area' },
]

const textModeOptions = [
  { label: 'Auto', value: 'auto' },
  { label: 'Value', value: 'value' },
  { label: 'Name', value: 'name' },
  { label: 'Value+Name', value: 'value_and_name' },
]
</script>

<template>
  <div class="panel-editor-viz">
    <!-- Timeseries options -->
    <template v-if="type === 'timeseries'">
      <NFormItem label="Draw Style" label-placement="left">
        <NRadioGroup
          :value="options.drawStyle ?? 'line'"
          @update:value="(v: string) => patch({ drawStyle: v as PanelOptions['drawStyle'] })"
        >
          <NRadioButton v-for="opt in drawStyleOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </NRadioGroup>
      </NFormItem>

      <NFormItem label="Fill Opacity" label-placement="left">
        <NSlider
          :value="options.fillOpacity ?? 0"
          :min="0"
          :max="100"
          :step="5"
          style="width: 200px"
          @update:value="(v: number) => patch({ fillOpacity: v })"
        />
        <span class="slider-val">{{ options.fillOpacity ?? 0 }}%</span>
      </NFormItem>

      <NFormItem label="Stacking" label-placement="left">
        <NRadioGroup
          :value="options.stacking ?? 'none'"
          @update:value="(v: string) => patch({ stacking: v as PanelOptions['stacking'] })"
        >
          <NRadioButton v-for="opt in stackingOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </NRadioGroup>
      </NFormItem>

      <NFormItem label="Line Width" label-placement="left">
        <NSlider
          :value="options.lineWidth ?? 1"
          :min="1"
          :max="10"
          :step="1"
          style="width: 200px"
          @update:value="(v: number) => patch({ lineWidth: v })"
        />
        <span class="slider-val">{{ options.lineWidth ?? 1 }}px</span>
      </NFormItem>

      <NFormItem label="Show Legend" label-placement="left">
        <NSwitch
          :value="options.showLegend ?? true"
          @update:value="(v: boolean) => patch({ showLegend: v })"
        />
      </NFormItem>

      <NFormItem label="Legend Position" label-placement="left">
        <NRadioGroup
          :value="options.legendPosition ?? 'bottom'"
          @update:value="(v: string) => patch({ legendPosition: v as PanelOptions['legendPosition'] })"
        >
          <NRadioButton v-for="opt in legendPositionOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </NRadioGroup>
      </NFormItem>
    </template>

    <!-- Stat options -->
    <template v-if="type === 'stat'">
      <NFormItem label="Color Mode" label-placement="left">
        <NRadioGroup
          :value="options.colorMode ?? 'value'"
          @update:value="(v: string) => patch({ colorMode: v as PanelOptions['colorMode'] })"
        >
          <NRadioButton v-for="opt in colorModeOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </NRadioGroup>
      </NFormItem>

      <NFormItem label="Graph Mode" label-placement="left">
        <NRadioGroup
          :value="options.graphMode ?? 'none'"
          @update:value="(v: string) => patch({ graphMode: v as PanelOptions['graphMode'] })"
        >
          <NRadioButton v-for="opt in graphModeOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </NRadioGroup>
      </NFormItem>

      <NFormItem label="Text Mode" label-placement="left">
        <NRadioGroup
          :value="options.textMode ?? 'auto'"
          @update:value="(v: string) => patch({ textMode: v as PanelOptions['textMode'] })"
        >
          <NRadioButton v-for="opt in textModeOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </NRadioGroup>
      </NFormItem>
    </template>

    <!-- Gauge options -->
    <template v-if="type === 'gauge'">
      <NFormItem label="Min" label-placement="left">
        <NInputNumber
          :value="options.min ?? 0"
          size="small"
          style="width: 120px"
          @update:value="(v: number | null) => patch({ min: v ?? 0 })"
        />
      </NFormItem>
      <NFormItem label="Max" label-placement="left">
        <NInputNumber
          :value="options.max ?? 100"
          size="small"
          style="width: 120px"
          @update:value="(v: number | null) => patch({ max: v ?? 100 })"
        />
      </NFormItem>
    </template>

    <!-- Text options -->
    <template v-if="type === 'text'">
      <NFormItem label="Content" label-placement="top">
        <NInput
          :value="options.content ?? ''"
          type="textarea"
          :rows="6"
          placeholder="Markdown or HTML content"
          @update:value="(v: string) => patch({ content: v })"
        />
      </NFormItem>
    </template>

    <!-- Common options (all types except row) -->
    <template v-if="type !== 'row'">
      <div class="viz-section-divider" />

      <NFormItem label="Unit" label-placement="left">
        <NInput
          :value="options.unit ?? ''"
          size="small"
          placeholder="bytes, short, percent, seconds, etc."
          style="width: 220px"
          @update:value="(v: string) => patch({ unit: v })"
        />
      </NFormItem>

      <NFormItem label="Decimals" label-placement="left">
        <NInputNumber
          :value="options.decimals"
          size="small"
          :min="0"
          :max="10"
          style="width: 100px"
          @update:value="(v: number | null) => patch({ decimals: v ?? undefined })"
        />
      </NFormItem>

      <div class="viz-section">
        <div class="viz-section-title">Thresholds</div>
        <PanelEditorThresholds
          :thresholds="options.thresholds ?? []"
          @update="(items: ThresholdItem[]) => patch({ thresholds: items })"
        />
      </div>

      <div class="viz-section">
        <div class="viz-section-title">Value Mappings</div>
        <PanelEditorValueMapping
          :value-mappings="options.valueMappings ?? []"
          @update="(items: ValueMapping[]) => patch({ valueMappings: items })"
        />
      </div>
    </template>
  </div>
</template>

<style scoped>
.panel-editor-viz {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 0;
}
.slider-val {
  font-size: 12px;
  color: var(--sre-text-secondary);
  min-width: 40px;
  text-align: right;
  margin-left: 8px;
}
.viz-section-divider {
  height: 1px;
  background: var(--sre-border);
  margin: 8px 0;
}
.viz-section {
  margin-top: 8px;
}
.viz-section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  margin-bottom: 8px;
}
</style>
