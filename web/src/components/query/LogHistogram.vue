<script setup lang="ts">
/**
 * LogHistogram — bar chart showing log volume over time buckets.
 * Inspired by Nightingale's LogsViewer HistogramChart.
 *
 * Features:
 * - Click a bar to zoom into that time bucket
 * - Brush selection to zoom into a range (drag to select)
 * - Responsive height (120px default)
 * - Dark/light theme aware
 */
import { ref, computed, onMounted, shallowRef, type Component } from 'vue'

const props = withDefaults(defineProps<{
  buckets: Array<{ timestamp: string | number; count: number }>
  height?: number
  loading?: boolean
}>(), {
  height: 120,
  loading: false,
})

const emit = defineEmits<{
  (e: 'barClick', start: number, end: number): void
  (e: 'brushSelect', start: number, end: number): void
}>()

// Lazy load ECharts
const ChartReady = ref(false)
const VChart = shallowRef<Component | null>(null)
const chartRef = ref<any>(null)

async function loadECharts() {
  try {
    const [{ use }, { CanvasRenderer }, { BarChart }, components, vc] = await Promise.all([
      import('echarts/core'),
      import('echarts/renderers'),
      import('echarts/charts'),
      import('echarts/components'),
      import('vue-echarts'),
    ])
    use([
      CanvasRenderer, BarChart,
      components.TooltipComponent, components.GridComponent,
      components.DataZoomComponent, components.BrushComponent,
    ])
    VChart.value = vc.default
    ChartReady.value = true
  } catch (e) {
    console.warn('[LogHistogram] ECharts load failed:', e)
  }
}

function getThemeColor(varName: string, fallback: string): string {
  if (typeof document === 'undefined') return fallback
  return getComputedStyle(document.documentElement).getPropertyValue(varName).trim() || fallback
}

const chartOption = computed(() => {
  if (!props.buckets?.length) return null

  const data: [number, number][] = props.buckets.map(b => {
    const ts = typeof b.timestamp === 'string' ? new Date(b.timestamp).getTime() : b.timestamp * 1000
    return [ts, b.count]
  })

  const textColor = getThemeColor('--sre-text-tertiary', '#94a3b8')
  const barColor = getThemeColor('--sre-primary', '#0d9488')

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      confine: true,
      formatter: (params: Array<{ value: [number, number] }>) => {
        if (!params[0]) return ''
        const date = new Date(params[0].value[0])
        const time = date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' })
        return `<div style="font-size:12px"><strong>${time}</strong><br/>${params[0].value[1]} logs</div>`
      },
    },
    grid: { left: 40, right: 12, top: 8, bottom: 24 },
    xAxis: {
      type: 'time',
      axisLabel: {
        fontSize: 10,
        color: textColor,
        formatter: (val: number) => {
          const d = new Date(val)
          return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
        },
      },
      axisLine: { lineStyle: { color: getThemeColor('--sre-border', '#e2e8f0') } },
      splitLine: { show: false },
    },
    yAxis: {
      type: 'value',
      axisLabel: { fontSize: 10, color: textColor },
      splitLine: { lineStyle: { type: 'dashed', color: getThemeColor('--sre-border', '#e2e8f0') } },
    },
    series: [{
      type: 'bar',
      data,
      barMaxWidth: 20,
      itemStyle: {
        color: barColor,
        opacity: 0.85,
        borderRadius: [2, 2, 0, 0],
      },
      emphasis: {
        itemStyle: { opacity: 1 },
      },
    }],
    brush: {
      toolbox: ['lineX', 'clear'],
      xAxisIndex: 0,
      brushStyle: { borderWidth: 1, color: 'rgba(13, 148, 136, 0.15)', borderColor: 'rgba(13, 148, 136, 0.6)' },
      throttleType: 'debounce',
      throttleDelay: 300,
    },
    toolbox: { show: false },
    dataZoom: [
      { type: 'inside', xAxisIndex: 0, zoomOnMouseWheel: true, moveOnMouseMove: true },
    ],
  }
})

function handleClick(params: { dataIndex: number; componentType?: string }) {
  // Ignore brush events that fire as click
  if (params.componentType === 'brush') return
  if (!props.buckets?.[params.dataIndex]) return
  const bucket = props.buckets[params.dataIndex]
  const ts = typeof bucket.timestamp === 'string' ? new Date(bucket.timestamp).getTime() / 1000 : bucket.timestamp

  // Calculate bucket duration from adjacent buckets
  const idx = params.dataIndex
  let duration = 60 // default 1 minute
  if (idx < props.buckets.length - 1) {
    const raw = props.buckets[idx + 1].timestamp
    const nextTs = typeof raw === 'number' ? raw : new Date(raw).getTime() / 1000
    duration = Math.max(nextTs - ts, 1)
  } else if (idx > 0) {
    const raw = props.buckets[idx - 1].timestamp
    const prevTs = typeof raw === 'number' ? raw : new Date(raw).getTime() / 1000
    duration = Math.max(ts - prevTs, 1)
  }

  emit('barClick', ts, ts + duration)
}

function handleBrushEnd(params: { areas?: Array<{ coordRange?: [number, number][]; range?: number[][] }> }) {
  const areas = params.areas
  if (!areas?.length) return
  const area = areas[0]
  // coordRange for xAxisIndex gives [min, max] in data coordinates
  if (area.coordRange) {
    const [min, max] = area.coordRange[0]
    emit('brushSelect', Math.floor(min / 1000), Math.floor(max / 1000))
  }
}

onMounted(() => { loadECharts() })
</script>

<template>
  <div class="log-histogram" :style="{ height: `${height}px` }">
    <div v-if="loading" class="histogram-loading">
      <NSpin size="small" />
    </div>
    <template v-else-if="ChartReady && VChart && chartOption">
      <component
        :is="VChart"
        ref="chartRef"
        :option="chartOption"
        :autoresize="true"
        :style="{ width: '100%', height: '100%' }"
        @click="handleClick"
        @brush-end="handleBrushEnd"
      />
    </template>
    <div v-else-if="!buckets?.length" class="histogram-empty">
      {{ $t('query.noHistogramData') }}
    </div>
  </div>
</template>

<style scoped>
.log-histogram {
  position: relative;
  border-radius: 6px;
  overflow: hidden;
  background: var(--sre-bg-sunken, #f8fafc);
  border: 1px solid var(--sre-border);
}
.histogram-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}
.histogram-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 12px;
  color: var(--sre-text-tertiary);
}
</style>
