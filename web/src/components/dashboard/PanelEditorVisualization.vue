<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NRadioGroup, NRadioButton, NSlider, NSwitch, NInputNumber, NInput, NFormItem } from 'naive-ui'
import type { PanelOptions, ThresholdItem, ValueMapping, PanelType } from '@/types/dashboard'
import PanelEditorThresholds from './PanelEditorThresholds.vue'
import PanelEditorValueMapping from './PanelEditorValueMapping.vue'

const { t } = useI18n()

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

const drawStyleOptions = computed(() => [
  { label: t('dashboardEditor.styleLine'), value: 'line' },
  { label: t('dashboardEditor.styleBars'), value: 'bars' },
  { label: t('dashboardEditor.stylePoints'), value: 'points' },
])

const stackingOptions = computed(() => [
  { label: t('dashboardEditor.stackingNone'), value: 'none' },
  { label: t('dashboardEditor.stackingNormal'), value: 'normal' },
])

const legendPositionOptions = computed(() => [
  { label: t('dashboardEditor.posBottom'), value: 'bottom' },
  { label: t('dashboardEditor.posRight'), value: 'right' },
  { label: t('dashboardEditor.posHidden'), value: 'hidden' },
])

const colorModeOptions = computed(() => [
  { label: t('dashboardEditor.colorValue'), value: 'value' },
  { label: t('dashboardEditor.colorBackground'), value: 'background' },
])

const graphModeOptions = computed(() => [
  { label: t('dashboardEditor.graphNone'), value: 'none' },
  { label: t('dashboardEditor.graphArea'), value: 'area' },
])

const textModeOptions = computed(() => [
  { label: t('dashboardEditor.textAuto'), value: 'auto' },
  { label: t('dashboardEditor.colorValue'), value: 'value' },
  { label: t('dashboardEditor.textName'), value: 'name' },
  { label: t('dashboardEditor.textValueName'), value: 'value_and_name' },
])
</script>

<template>
  <div class="panel-editor-viz">
    <!-- Timeseries options -->
    <template v-if="type === 'timeseries'">
      <NFormItem :label="t('dashboardEditor.drawStyle')" label-placement="left">
        <NRadioGroup
          :value="options.drawStyle ?? 'line'"
          @update:value="(v: string) => patch({ drawStyle: v as PanelOptions['drawStyle'] })"
        >
          <NRadioButton v-for="opt in drawStyleOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </NRadioGroup>
      </NFormItem>

      <NFormItem :label="t('dashboardEditor.fillOpacity')" label-placement="left">
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

      <NFormItem :label="t('dashboardEditor.stacking')" label-placement="left">
        <NRadioGroup
          :value="options.stacking ?? 'none'"
          @update:value="(v: string) => patch({ stacking: v as PanelOptions['stacking'] })"
        >
          <NRadioButton v-for="opt in stackingOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </NRadioGroup>
      </NFormItem>

      <NFormItem :label="t('dashboardEditor.lineWidth')" label-placement="left">
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

      <NFormItem :label="t('dashboardEditor.showLegend')" label-placement="left">
        <NSwitch
          :value="options.showLegend ?? true"
          @update:value="(v: boolean) => patch({ showLegend: v })"
        />
      </NFormItem>

      <NFormItem :label="t('dashboardEditor.legendPosition')" label-placement="left">
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
      <NFormItem :label="t('dashboardEditor.colorMode')" label-placement="left">
        <NRadioGroup
          :value="options.colorMode ?? 'value'"
          @update:value="(v: string) => patch({ colorMode: v as PanelOptions['colorMode'] })"
        >
          <NRadioButton v-for="opt in colorModeOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </NRadioGroup>
      </NFormItem>

      <NFormItem :label="t('dashboardEditor.graphMode')" label-placement="left">
        <NRadioGroup
          :value="options.graphMode ?? 'none'"
          @update:value="(v: string) => patch({ graphMode: v as PanelOptions['graphMode'] })"
        >
          <NRadioButton v-for="opt in graphModeOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </NRadioGroup>
      </NFormItem>

      <NFormItem :label="t('dashboardEditor.textMode')" label-placement="left">
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
      <NFormItem :label="t('dashboardEditor.min')" label-placement="left">
        <NInputNumber
          :value="options.min ?? 0"
          size="small"
          style="width: 120px"
          @update:value="(v: number | null) => patch({ min: v ?? 0 })"
        />
      </NFormItem>
      <NFormItem :label="t('dashboardEditor.max')" label-placement="left">
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
      <NFormItem :label="t('dashboardEditor.content')" label-placement="top">
        <NInput
          :value="options.content ?? ''"
          type="textarea"
          :rows="6"
          :placeholder="t('dashboardEditor.placeholderMarkdown')"
          @update:value="(v: string) => patch({ content: v })"
        />
      </NFormItem>
    </template>

    <!-- Common options (all types except row) -->
    <template v-if="type !== 'row'">
      <div class="viz-section-divider" />

      <NFormItem :label="t('dashboardEditor.unit')" label-placement="left">
        <NInput
          :value="options.unit ?? ''"
          size="small"
          :placeholder="t('dashboardEditor.placeholderUnit')"
          style="width: 220px"
          @update:value="(v: string) => patch({ unit: v })"
        />
      </NFormItem>

      <NFormItem :label="t('dashboardEditor.decimals')" label-placement="left">
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
        <div class="viz-section-title">{{ t('dashboardEditor.thresholds') }}</div>
        <PanelEditorThresholds
          :thresholds="options.thresholds ?? []"
          @update="(items: ThresholdItem[]) => patch({ thresholds: items })"
        />
      </div>

      <div class="viz-section">
        <div class="viz-section-title">{{ t('dashboardEditor.valueMappings') }}</div>
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
