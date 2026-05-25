<script setup lang="ts">
/**
 * MetricChartControls — Nightingale PromGraphCpt-style controls bar
 * for metric query results: max data points, min step, chart type,
 * settings (legend, tooltip), share URL.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NInputNumber, NButton, NButtonGroup, NIcon, NPopover,
  NSwitch, NSelect, NDivider, useMessage,
} from 'naive-ui'
import {
  SettingsOutline, ShareSocialOutline,
  TrendingUpOutline, AnalyticsOutline,
} from '@vicons/ionicons5'

export interface ChartSettings {
  maxDataPoints: number | null
  minStep: number | null
  chartType: 'line' | 'area'
  showLegend: boolean
  sharedTooltip: boolean
  tooltipSort: 'desc' | 'asc'
}

const props = defineProps<{
  modelValue: ChartSettings
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: ChartSettings): void
}>()

const { t } = useI18n()
const message = useMessage()

const settings = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

function update<K extends keyof ChartSettings>(key: K, val: ChartSettings[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: val })
}

function shareUrl() {
  const url = window.location.href
  navigator.clipboard?.writeText(url).then(() => {
    message.success(t('query.urlCopied'))
  }).catch(() => {
    message.warning(url)
  })
}
</script>

<template>
  <div class="metric-chart-controls">
    <!-- Max data points -->
    <div class="control-item">
      <span class="control-label">{{ t('query.maxDataPoints') }}</span>
      <NInputNumber
        :value="settings.maxDataPoints"
        :placeholder="'240'"
        :min="1"
        :max="10000"
        :show-button="false"
        size="tiny"
        class="control-input-sm"
        @update:value="(v: number | null) => update('maxDataPoints', v)"
      />
    </div>

    <!-- Min step -->
    <div class="control-item">
      <span class="control-label">{{ t('query.minStep') }}</span>
      <NInputNumber
        :value="settings.minStep"
        :placeholder="'15'"
        :min="1"
        :max="86400"
        :show-button="false"
        size="tiny"
        class="control-input-xs"
        @update:value="(v: number | null) => update('minStep', v)"
      />
      <span class="control-unit">s</span>
    </div>

    <!-- Chart type toggle -->
    <NButtonGroup size="tiny">
      <NButton
        :type="settings.chartType === 'line' ? 'primary' : 'default'"
        :secondary="settings.chartType !== 'line'"
        @click="update('chartType', 'line')"
      >
        <template #icon><NIcon><TrendingUpOutline /></NIcon></template>
        {{ t('query.chartLine') }}
      </NButton>
      <NButton
        :type="settings.chartType === 'area' ? 'primary' : 'default'"
        :secondary="settings.chartType !== 'area'"
        @click="update('chartType', 'area')"
      >
        <template #icon><NIcon><AnalyticsOutline /></NIcon></template>
        {{ t('query.chartArea') }}
      </NButton>
    </NButtonGroup>

    <!-- Settings gear -->
    <NPopover trigger="click" placement="bottom-end" :style="{ width: '260px' }">
      <template #trigger>
        <NButton size="tiny" quaternary>
          <template #icon><NIcon><SettingsOutline /></NIcon></template>
        </NButton>
      </template>
      <div class="settings-popover">
        <div class="settings-row">
          <span class="settings-label">{{ t('query.showLegend') }}</span>
          <NSwitch :value="settings.showLegend" size="small" @update:value="(v: boolean) => update('showLegend', v)" />
        </div>
        <NDivider style="margin: 8px 0;" />
        <div class="settings-row">
          <span class="settings-label">{{ t('query.sharedTooltip') }}</span>
          <NSwitch :value="settings.sharedTooltip" size="small" @update:value="(v: boolean) => update('sharedTooltip', v)" />
        </div>
        <div v-if="settings.sharedTooltip" class="settings-row">
          <span class="settings-label">{{ t('query.tooltipSort') }}</span>
          <NSelect
            :value="settings.tooltipSort"
            :options="[{ label: 'Desc', value: 'desc' }, { label: 'Asc', value: 'asc' }]"
            size="tiny"
            class="control-select-sort"
            @update:value="(v: 'desc' | 'asc') => update('tooltipSort', v)"
          />
        </div>
      </div>
    </NPopover>

    <!-- Share -->
    <NButton size="tiny" quaternary @click="shareUrl">
      <template #icon><NIcon><ShareSocialOutline /></NIcon></template>
    </NButton>
  </div>
</template>

<style scoped>
.metric-chart-controls {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.control-item {
  display: flex;
  align-items: center;
  gap: 4px;
}
.control-label {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  white-space: nowrap;
}
.control-input-sm { width: 70px; }
.control-input-xs { width: 56px; }
.control-unit { font-size: 12px; color: var(--sre-text-tertiary); }
.control-select-sort { width: 80px; }
.settings-popover { padding: 4px 0; }
.settings-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 0;
}
.settings-label {
  font-size: 13px;
  color: var(--sre-text-secondary);
}
</style>
