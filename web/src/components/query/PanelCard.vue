<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, h, shallowRef } from 'vue'
import { NDataTable, NEmpty, NSpin as NSpinComponent } from 'naive-ui'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart, PieChart, GaugeChart, ScatterChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { datasourceApi } from '@/api'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { PanelConfig, PanelTarget } from '@/types/dashboard'
import type { QueryResponse } from '@/types'

const { t } = useI18n()
const router = useRouter()

use([CanvasRenderer, LineChart, BarChart, PieChart, GaugeChart, ScatterChart, TooltipComponent, LegendComponent, GridComponent])

// FE4-11: timeRange is the global dashboard time range, passed from parent.
// All panels in a dashboard share the same timeRange instance, ensuring sync.
const props = defineProps<{
  panel: PanelConfig
  timeRange: { start: number; end: number }
}>()

const loading = ref(false)
const error = ref('')

// FE4-10: Fullscreen mode
const cardRef = ref<HTMLElement | null>(null)
const isFullscreen = ref(false)

function toggleFullscreen() {
  if (!cardRef.value) return
  if (!document.fullscreenElement) {
    cardRef.value.requestFullscreen().catch(() => {})
  } else {
    document.exitFullscreen().catch(() => {})
  }
}

function onFullscreenChange() {
  isFullscreen.value = !!document.fullscreenElement
}

onMounted(() => document.addEventListener('fullscreenchange', onFullscreenChange))
onUnmounted(() => document.removeEventListener('fullscreenchange', onFullscreenChange))

// FE4-12: Open panel query in Explore
function openInExplore() {
  const target = props.panel.targets?.[0]
  if (!target?.datasourceId || !target?.expression?.trim()) return
  router.push({
    path: '/alert/explore',
    query: {
      datasource_id: String(target.datasourceId),
      expression: target.expression,
    },
  })
}

const series = ref<QueryResponse['series']>([])
const resultType = ref<'vector' | 'matrix' | 'logs' | null>(null)

const stepAuto = computed(() => {
  const diff = (props.timeRange.end - props.timeRange.start) / 1000
  if (diff <= 3600) return '15s'
  if (diff <= 21600) return '1m'
  if (diff <= 86400) return '5m'
  return '15m'
})

async function fetchData() {
  const targets = props.panel.targets
  if (!targets?.length) return

  loading.value = true
  error.value = ''
  series.value = []

  try {
    const allSeries: QueryResponse['series'] = []
    let type: 'vector' | 'matrix' | 'logs' | null = null

    for (const t of targets) {
      if (!t.datasourceId || !t.expression?.trim()) continue
      const res = await datasourceApi.rangeQuery(t.datasourceId, {
        expression: t.expression,
        start: Math.floor(props.timeRange.start / 1000),
        end: Math.floor(props.timeRange.end / 1000),
        step: stepAuto.value,
      })
      const data = res.data.data
      if (data.result_type) type = data.result_type
      if (data.series) {
        for (const s of data.series) {
          const labelStr = Object.entries(s.labels || {})
            .filter(([k]) => k !== '__name__')
            .map(([k, v]) => `${k}=${v}`)
            .join(',')
          const name = t.legendFormat
            ? t.legendFormat.replace(/\{\{\.label\}\}/g, labelStr)
            : (labelStr || s.labels?.__name__ || 'value')
          allSeries.push({ ...s, labels: { ...s.labels, __panel_name: name } })
        }
      }
    }
    resultType.value = type
    series.value = allSeries
  } catch (err: unknown) {
    const e = err as { response?: { data?: { message?: string } }; message?: string }
    error.value = e?.response?.data?.message || e?.message || t('query.queryFailed')
  } finally {
    loading.value = false
  }
}

interface ChartSeriesItem {
  name: string
  type: string
  smooth?: boolean
  symbol?: string
  barMaxWidth?: number
  data: [string, number][]
  areaStyle?: { opacity: number }
  stack?: string
  lineStyle?: { width: number }
  markLine?: { silent: boolean; data: unknown[] }
}

const statValue = computed(() => {
  if (!series.value.length) return null
  const s = series.value[0]
  if (s.values?.length) return s.values[s.values.length - 1].value
  return null
})

const statColor = computed((): string => {
  const val = statValue.value
  const thresholds = props.panel.options?.thresholds as { value: number; color: string }[] | undefined
  if (val == null || !thresholds?.length) {
    return (props.panel.options?.color as string) || 'var(--sre-text-primary)'
  }
  const sorted = [...thresholds].sort((a, b) => a.value - b.value)
  let color: string = (props.panel.options?.color as string) || sorted[0]?.color || 'var(--sre-text-primary)'
  for (const t of sorted) {
    if (val >= t.value) color = t.color
  }
  return color
})

const statSeriesName = computed(() => {
  if (!series.value.length) return ''
  return series.value[0].labels?.__panel_name || series.value[0].labels?.__name__ || 'value'
})

const unitLabel = computed(() => props.panel.options?.unit ?? '')
const decimalsVal = computed(() => props.panel.options?.decimals)

function formatValue(val: number): string {
  const d = decimalsVal.value
  const formatted = d != null ? val.toFixed(d) : String(val)
  return unitLabel.value ? `${formatted} ${unitLabel.value}` : formatted
}

const chartOption = computed(() => {
  if (!series.value.length) return null

  const opts = props.panel.options
  const xData: string[] = []
  const seriesList: ChartSeriesItem[] = []
  const seen = new Map<string, boolean>()
  const drawStyle = opts?.drawStyle ?? 'line'
  const fillOpacity = opts?.fillOpacity ?? 0
  const stacking = opts?.stacking ?? 'none'
  const lineWidth = opts?.lineWidth ?? 1

  for (const s of series.value) {
    const name = s.labels?.__panel_name || s.labels?.__name__ || 'value'
    if (!seen.has(name)) {
      seen.set(name, true)
      const seriesType = drawStyle === 'bars' ? 'bar' : 'line'
      const item: ChartSeriesItem = {
        name,
        type: seriesType,
        smooth: drawStyle === 'line',
        symbol: drawStyle === 'points' ? 'circle' : 'none',
        barMaxWidth: drawStyle === 'bars' ? 40 : undefined,
        data: [] as [string, number][],
        areaStyle: (fillOpacity > 0 && drawStyle !== 'bars') ? { opacity: fillOpacity / 100 } : undefined,
        stack: stacking === 'normal' ? 'total' : undefined,
        lineStyle: drawStyle !== 'bars' ? { width: lineWidth } : undefined,
      }
      seriesList.push(item)
    }
    const target = seriesList.find(sl => sl.name === name)
    if (target) {
      for (const v of s.values) {
        const ts = new Date(v.ts * 1000).toLocaleTimeString()
        target.data.push([ts, v.value])
      }
    }
  }

  // Threshold markLines
  const thresholds = opts?.thresholds
  const markLineData: { yAxis: number; lineStyle: { type: string; color: string }; label: { formatter: string } }[] = []
  if (thresholds?.length) {
    for (const th of thresholds) {
      markLineData.push({
        yAxis: th.value,
        lineStyle: { type: 'dashed', color: th.color },
        label: { formatter: String(th.value) },
      })
    }
  }

  // Legend position
  const legendPos = opts?.legendPosition ?? 'bottom'
  const showLegend = opts?.showLegend !== false
  let legendConfig: Record<string, unknown>
  if (!showLegend || legendPos === 'hidden') {
    legendConfig = { show: false }
  } else if (legendPos === 'right') {
    legendConfig = { type: 'scroll', right: 0, top: 0, orient: 'vertical', textStyle: { fontSize: 11 } }
  } else {
    legendConfig = { type: 'scroll', bottom: 0, textStyle: { fontSize: 11 } }
  }

  // Apply markLine to first series if thresholds exist
  if (markLineData.length && seriesList.length > 0) {
    seriesList[0].markLine = { silent: true, data: markLineData }
  }

  // Tooltip with unit
  const tooltipFormatter = unitLabel.value
    ? { trigger: 'axis' as const, formatter: (params: { seriesName: string; value: [string, number] }[]) => {
        if (!Array.isArray(params)) return ''
        let html = `<div style="font-size:12px">${params[0]?.value?.[0] ?? ''}</div>`
        for (const p of params) {
          html += `<div>${p.seriesName}: <b>${formatValue(p.value?.[1] ?? 0)}</b></div>`
        }
        return html
      }}
    : { trigger: 'axis' as const }

  if (resultType.value === 'matrix') {
    const allTimes = new Set<string>()
    for (const sl of seriesList) {
      for (const d of sl.data) allTimes.add(d[0])
    }
    const sorted = Array.from(allTimes).sort()
    for (const sl of seriesList) {
      const timeMap = new Map(sl.data.map((d: [string, number]) => [d[0], d[1]]))
      sl.data = sorted.map(t => [t, timeMap.get(t) ?? 0] as [string, number])
    }
    return {
      tooltip: tooltipFormatter,
      legend: legendConfig,
      grid: { left: 50, right: legendPos === 'right' && showLegend ? 80 : 16, top: 12, bottom: legendPos === 'bottom' && showLegend ? 40 : 30 },
      xAxis: { type: 'category' as const, data: sorted },
      yAxis: {
        type: 'value' as const,
        axisLabel: unitLabel.value ? { formatter: (val: number) => formatValue(val) } : undefined,
      },
      series: seriesList,
    }
  }

  return {
    tooltip: tooltipFormatter,
    legend: { show: false },
    grid: { left: 50, right: 16, top: 12, bottom: 30 },
    xAxis: { type: 'category' as const, data: xData },
    yAxis: {
      type: 'value' as const,
      axisLabel: unitLabel.value ? { formatter: (val: number) => formatValue(val) } : undefined,
    },
    series: seriesList,
  }
})

const barOption = computed(() => {
  const base = chartOption.value
  if (!base?.series) return null
  const seriesList = base.series.map((s: ChartSeriesItem) => ({ ...s, type: 'bar', smooth: undefined, barMaxWidth: 40 }))
  return { ...base, series: seriesList, tooltip: { trigger: 'axis' as const } }
})

const gaugeOption = computed(() => {
  const val = statValue.value
  if (val == null) return null
  const max = (props.panel.options?.max as number) ?? 100
  const thresholds = (props.panel.options?.thresholds as { value: number; color: string }[] | undefined) ?? []
  const detailFormatter = props.panel.options?.unit ? `{value} ${props.panel.options.unit}` : '{value}'
  return {
    tooltip: { formatter: `{b}: {c}${props.panel.options?.unit ? ' ' + props.panel.options.unit : ''}` },
    series: [{
      type: 'gauge',
      startAngle: 210,
      endAngle: -30,
      min: props.panel.options?.min ?? 0,
      max,
      center: ['50%', '58%'],
      radius: '85%',
      axisLine: {
        lineStyle: {
          width: 20,
          color: thresholds.length
            ? thresholds.map(t => [(t.value as number) / max, t.color])
            : [[0.6, 'var(--sre-success)'], [0.8, 'var(--sre-warning)'], [1, 'var(--sre-danger)']],
        },
      },
      pointer: { length: '60%', width: 6, itemStyle: { color: 'var(--sre-text-primary)' } },
      detail: { valueAnimation: true, formatter: detailFormatter, fontSize: 20, offsetCenter: [0, '60%'] },
      data: [{ value: val, name: statSeriesName.value }],
    }],
  }
})

const pieOption = computed(() => {
  if (!series.value.length) return null
  const data: { name: string; value: number }[] = []
  for (const s of series.value) {
    const name = s.labels?.__panel_name || s.labels?.__name__ || 'value'
    const vals = s.values
    if (vals?.length) data.push({ name, value: vals[vals.length - 1].value })
  }
  return {
    tooltip: { trigger: 'item' as const },
    legend: { type: 'scroll' as const, bottom: 0, textStyle: { fontSize: 11 } },
    series: [{
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['50%', '48%'],
      emphasis: { label: { fontSize: 16, fontWeight: 'bold' } },
      label: { formatter: '{b}\n{d}%', fontSize: 11 },
      data,
    }],
  }
})

const tableData = computed(() => {
  const rows: { labels: Record<string, string>; value: number; _key: number }[] = []
  let idx = 0
  for (const s of series.value) {
    for (const v of s.values) {
      rows.push({ labels: s.labels, value: v.value, _key: idx++ })
    }
  }
  return rows
})

const tableColumns = computed(() => {
  const keys = new Set<string>()
  for (const s of series.value) {
    Object.keys(s.labels || {}).forEach(k => { if (k !== '__panel_name') keys.add(k) })
  }
  interface TableColumn {
    title: string
    key: string
    width?: number
    ellipsis?: { tooltip: boolean }
    render: (row: { labels: Record<string, string>; value: number }) => string
  }
  const cols: TableColumn[] = Array.from(keys).map(k => ({
    title: k,
    key: k,
    ellipsis: { tooltip: true },
    render(row) { return row.labels?.[k] || '-' },
  }))
  cols.push({ title: t('query.value'), key: 'value', width: 120, render(row) { return row.value?.toFixed(4) || '-' } })
  return cols
})

function renderMarkdown(md: string): string {
  let html = md
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
  // headings
  html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>')
  html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>')
  html = html.replace(/^# (.+)$/gm, '<h1>$1</h1>')
  // bold and italic
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
  html = html.replace(/\*(.+?)\*/g, '<em>$1</em>')
  // links — only allow http/https to prevent javascript: XSS
  html = html.replace(/\[(.+?)\]\((.+?)\)/g, (_match, text, url) => {
    const safeUrl = url.startsWith('http://') || url.startsWith('https://') ? url : '#'
    return `<a href="${safeUrl}" target="_blank">${text}</a>`
  })
  // unordered lists
  html = html.replace(/^- (.+)$/gm, '<li>$1</li>')
  html = html.replace(/(<li>.*<\/li>)/s, '<ul>$1</ul>')
  // line breaks
  html = html.replace(/\n/g, '<br>')
  return html
}

let timeout: ReturnType<typeof setTimeout>
watch(() => [props.timeRange, props.panel.targets], () => {
  clearTimeout(timeout)
  timeout = setTimeout(fetchData, 100)
}, { deep: true })

onMounted(fetchData)
</script>

<template>
  <div ref="cardRef" class="panel-card" :class="{ 'panel-fullscreen': isFullscreen }">
    <div class="panel-card-header">
      <span class="panel-title">{{ panel.title || t('query.panel') }}</span>
      <NSpinComponent v-if="loading" :size="14" />
      <button class="panel-fs-btn" :title="isFullscreen ? 'Exit fullscreen' : 'Fullscreen'" @click="toggleFullscreen">
        <svg v-if="!isFullscreen" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3H5a2 2 0 00-2 2v3m18 0V5a2 2 0 00-2-2h-3m0 18h3a2 2 0 002-2v-3M3 16v3a2 2 0 002 2h3"/></svg>
        <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 14h3a2 2 0 012 2v3m4-5h3a2 2 0 002-2V9M15 3v3a2 2 0 002 2h3M4 10V7a2 2 0 012-2h3"/></svg>
      </button>
      <!-- FE4-12: Open panel query in Explore -->
      <button
        v-if="panel.targets?.[0]?.datasourceId && panel.targets?.[0]?.expression"
        class="panel-fs-btn"
        title="Open in Explore"
        @click="openInExplore"
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
      </button>
      <span v-if="error" class="panel-error">{{ error }}</span>
    </div>
    <div class="panel-card-body">
      <template v-if="loading && !series.length">
        <div class="panel-loading"><NSpinComponent :size="24" /></div>
      </template>
      <template v-else-if="error && !series.length">
        <NEmpty :description="error" size="small" />
      </template>
      <template v-else-if="!series.length">
        <NEmpty :description="t('query.noResults')" size="small" />
      </template>

      <!-- Timeseries -->
      <template v-else-if="panel.type === 'timeseries' || !panel.type">
        <VChart v-if="chartOption" :option="chartOption" autoresize style="height: 100%" />
      </template>

      <!-- Stat -->
      <template v-else-if="panel.type === 'stat'">
        <div class="stat-display" :style="{ color: statColor }">
          <div class="stat-value">{{ statValue?.toFixed(2) ?? '-' }}</div>
          <div class="stat-label">{{ statSeriesName }}</div>
        </div>
      </template>

      <!-- Gauge -->
      <template v-else-if="panel.type === 'gauge'">
        <VChart v-if="gaugeOption" :option="gaugeOption" autoresize style="height: 100%" />
      </template>

      <!-- Bar -->
      <template v-else-if="panel.type === 'bar'">
        <VChart v-if="barOption" :option="barOption" autoresize style="height: 100%" />
      </template>

      <!-- Pie -->
      <template v-else-if="panel.type === 'pie'">
        <VChart v-if="pieOption" :option="pieOption" autoresize style="height: 100%" />
      </template>

      <!-- Table -->
      <template v-else-if="panel.type === 'table'">
        <NDataTable
          :columns="tableColumns"
          :data="tableData"
          :max-height="280"
          :row-key="(row: Record<string, unknown>) => String(row._key)"
          size="small"
          striped
        />
      </template>

      <!-- Text -->
      <template v-else-if="panel.type === 'text'">
        <div class="text-panel" v-html="renderMarkdown(panel.options?.content ?? '')" />
      </template>

      <!-- Row -->
      <template v-else-if="panel.type === 'row'">
        <div class="row-panel">
          <span class="row-title">{{ panel.title }}</span>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.panel-card {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  height: 100%;
}
.panel-card.panel-fullscreen {
  border-radius: 0;
  height: 100vh;
}
.panel-fs-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 3px;
  background: transparent;
  color: var(--sre-text-muted);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
  flex-shrink: 0;
}
.panel-fs-btn:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-primary);
}
.panel-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--sre-border);
}
.panel-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.panel-error {
  font-size: 11px;
  color: var(--sre-danger);
}
.panel-card-body {
  flex: 1;
  min-height: 0;
  padding: 8px;
}
.panel-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 120px;
}
.stat-display {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 100px;
}
.stat-value {
  font-size: 36px;
  font-weight: 700;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}
.stat-label {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  margin-top: 4px;
}
.text-panel {
  font-size: 13px;
  color: var(--sre-text-primary);
  line-height: 1.6;
  overflow-y: auto;
  height: 100%;
}
.text-panel :deep(h1) { font-size: 20px; font-weight: 700; margin: 8px 0 4px; }
.text-panel :deep(h2) { font-size: 16px; font-weight: 600; margin: 6px 0 3px; }
.text-panel :deep(h3) { font-size: 14px; font-weight: 600; margin: 4px 0 2px; }
.text-panel :deep(strong) { font-weight: 700; }
.text-panel :deep(em) { font-style: italic; }
.text-panel :deep(a) { color: var(--sre-primary); text-decoration: underline; }
.text-panel :deep(ul) { padding-left: 20px; margin: 4px 0; }
.text-panel :deep(li) { margin: 2px 0; }
.row-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--sre-bg-sunken);
  border-radius: 4px;
  height: 100%;
}
.row-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
}
</style>
