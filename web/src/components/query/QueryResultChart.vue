<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import {
  TooltipComponent,
  LegendComponent,
  GridComponent,
  DataZoomComponent,
} from 'echarts/components'
import { NTabs, NTabPane, NDataTable } from 'naive-ui'
import VChart from 'vue-echarts'
import type { QueryTarget, TimeRange, QuerySeriesItem } from '@/types/query'
import { formatValue, type ValueFormat } from '@/utils/valueFormatter'

use([CanvasRenderer, LineChart, TooltipComponent, LegendComponent, GridComponent, DataZoomComponent])

const props = withDefaults(defineProps<{
  targets: QueryTarget[]
  timeRange: TimeRange
  height?: number
  valueFormat?: ValueFormat
}>(), {
  height: 400,
  valueFormat: 'short',
})

const { t } = useI18n()
const legendMode = ref<'chart' | 'table'>('chart')

function applyLegendFormat(format: string, labels: Record<string, string>): string {
  if (!format) {
    return Object.entries(labels)
      .filter(([k]) => k !== '__name__')
      .map(([k, v]) => `${k}="${v}"`)
      .join(', ')
  }
  return format
    .replace(/\{\{(\w+)\}\}/g, (_, key) => labels[key] || '')
    .replace(/\$(\w+)/g, (_, key) => labels[key] || '')
}

function calcStats(values: Array<{ ts: number; value: number }>) {
  if (!values.length) return { min: 0, max: 0, avg: 0, last: 0 }
  let min = Infinity, max = -Infinity, sum = 0
  for (const v of values) {
    if (v.value < min) min = v.value
    if (v.value > max) max = v.value
    sum += v.value
  }
  return {
    min,
    max,
    avg: sum / values.length,
    last: values[values.length - 1].value,
  }
}

// Legend table data
const legendColumns = computed(() => [
  { title: t('query.series'), key: 'name', ellipsis: { tooltip: true } },
  { title: t('query.min'), key: 'min', width: 100, align: 'right' as const },
  { title: t('query.max'), key: 'max', width: 100, align: 'right' as const },
  { title: t('query.avg'), key: 'avg', width: 100, align: 'right' as const },
  { title: t('query.last'), key: 'last', width: 100, align: 'right' as const },
])

const legendData = computed(() => {
  const rows: Array<{ key: string; name: string; min: string; max: string; avg: string; last: string }> = []
  for (const t of props.targets) {
    if (!t.enabled || t.resultType !== 'matrix') continue
    for (let i = 0; i < t.series.length; i++) {
      const s = t.series[i]
      const name = applyLegendFormat(t.legendFormat, s.labels)
      const stats = calcStats(s.values)
      rows.push({
        key: `${t.id}-${i}`,
        name,
        min: formatValue(stats.min, props.valueFormat),
        max: formatValue(stats.max, props.valueFormat),
        avg: formatValue(stats.avg, props.valueFormat),
        last: formatValue(stats.last, props.valueFormat),
      })
    }
  }
  return rows
})

// Design system colors for ECharts (reads CSS variables at render time)
const chartTheme = computed(() => {
  const s = getComputedStyle(document.documentElement)
  return {
    axisPointer: s.getPropertyValue('--sre-text-tertiary').trim() || '#64748b',
    legendPage: s.getPropertyValue('--sre-text-muted').trim() || '#94a3b8',
    splitLine: s.getPropertyValue('--sre-border').trim() || '#1e293b',
  }
})

const chartOption = computed(() => {
  const allSeries = props.targets
    .filter(t => t.enabled && t.resultType === 'matrix')
    .flatMap(t =>
      t.series.map(s => ({
        name: applyLegendFormat(t.legendFormat, s.labels),
        type: 'line' as const,
        data: s.values.map(v => [v.ts, v.value]),
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 1.5 },
        emphasis: { lineStyle: { width: 2.5 } },
      }))
    )

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross', lineStyle: { color: chartTheme.value.axisPointer } },
      valueFormatter: (val: number) => formatValue(val, props.valueFormat),
    },
    legend: {
      type: 'scroll',
      bottom: 0,
      textStyle: { fontSize: 11 },
      pageTextStyle: { color: chartTheme.value.legendPage },
    },
    grid: { left: 60, right: 20, top: 30, bottom: 60 },
    xAxis: {
      type: 'time',
      min: props.timeRange.start,
      max: props.timeRange.end,
      axisLabel: {
        fontSize: 11,
        formatter: (val: number) => {
          const d = new Date(val)
          return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
        },
      },
    },
    yAxis: {
      type: 'value',
      axisLabel: { fontSize: 11, formatter: (val: number) => formatValue(val, props.valueFormat) },
      splitLine: { lineStyle: { type: 'dashed', color: chartTheme.value.splitLine } },
    },
    series: allSeries,
    dataZoom: [
      { type: 'inside', xAxisIndex: 0 },
      { type: 'slider', xAxisIndex: 0, bottom: 25, height: 20 },
    ],
    animation: false,
    large: true,
    largeThreshold: 1000,
  }
})

const hasData = computed(() =>
  props.targets.some(t => t.enabled && t.resultType === 'matrix' && t.series.length > 0)
)
</script>

<template>
  <div class="chart-container">
    <div v-if="hasData" class="chart-wrapper">
      <div class="legend-toggle">
        <NTabs v-model:value="legendMode" type="segment" size="small" animated>
          <NTabPane name="chart" :tab="t('query.chart')" />
          <NTabPane name="table" :tab="t('query.stats')" />
        </NTabs>
      </div>

      <VChart
        v-show="legendMode === 'chart'"
        :option="chartOption"
        :style="{ height: height + 'px', width: '100%' }"
        autoresize
      />

      <NDataTable
        v-if="legendMode === 'table'"
        :columns="legendColumns"
        :data="legendData"
        :max-height="height"
        size="small"
        striped
        :pagination="false"
        :scroll-x="600"
      />
    </div>

    <div v-else class="empty-chart" :style="{ height: height + 'px' }">
      <span>{{ t('common.noData') }}</span>
    </div>
  </div>
</template>

<style scoped>
.chart-container {
  background: var(--sre-bg-card);
  border-radius: 6px;
}
.chart-wrapper {
  position: relative;
}
.legend-toggle {
  display: flex;
  justify-content: flex-end;
  padding: 4px 8px;
}
.empty-chart {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--sre-text-tertiary);
  font-size: 14px;
}
</style>
