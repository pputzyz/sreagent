<script setup lang="ts">
import { shallowRef, computed, onMounted, onUnmounted } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart, LineChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { dashboardApi } from '@/api'
import type { DashboardStats, MTTRStats, AlertTrendPoint, TopRuleItem } from '@/types'
import { RefreshOutline } from '@vicons/ionicons5'

use([CanvasRenderer, PieChart, LineChart, TooltipComponent, LegendComponent, GridComponent])

const message = useMessage()
const { t } = useI18n()

// Range selector: 1=today, 7=7d, 30=30d  (mapped to API: hours for MTTR, days for trend)
const range = shallowRef<1 | 7 | 30>(7)
const refreshing = shallowRef(false)
const lastSyncAt = shallowRef<number>(Date.now())
const errorMsg = shallowRef<string>('')

const stats = shallowRef<DashboardStats>({
  total_datasources: 0,
  total_rules: 0,
  active_alerts: 0,
  resolved_today: 0,
  total_users: 0,
  total_teams: 0,
  severity_breakdown: { critical: 0, warning: 0, info: 0 },
})

const emptyMetric = { mean: -1, p50: -1, p95: -1, count: 0 }
const mttrStats = shallowRef<MTTRStats>({
  window_hours: 24,
  mtta: { ...emptyMetric },
  mttr: { ...emptyMetric },
  by_severity: [],
  mtta_seconds: -1,
  mttr_seconds: -1,
  acked_count: 0,
  resolved_count: 0,
})

const trendData = shallowRef<AlertTrendPoint[]>([])
const topRules = shallowRef<TopRuleItem[]>([])

// ===== formatters =====
function formatNumber(n: number): string {
  if (n < 1000) return String(n)
  if (n < 1_000_000) return (n / 1000).toFixed(n < 10_000 ? 1 : 0) + 'k'
  return (n / 1_000_000).toFixed(1) + 'M'
}

function formatMMSS(seconds: number): string {
  if (seconds < 0) return '—'
  const m = Math.floor(seconds / 60)
  const s = Math.round(seconds % 60)
  if (m < 100) return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  const h = Math.floor(m / 60)
  return `${h}h ${m % 60}m`
}

function formatRelative(ts: number): string {
  const diff = Math.max(0, Date.now() - ts)
  const sec = Math.floor(diff / 1000)
  if (sec < 60) return `${sec}s ago`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min} minute${min === 1 ? '' : 's'} ago`
  const h = Math.floor(min / 60)
  return `${h} hour${h === 1 ? '' : 's'} ago`
}

const lastSyncText = shallowRef(formatRelative(Date.now()))
function tickRelative() {
  lastSyncText.value = formatRelative(lastSyncAt.value)
}
setInterval(tickRelative, 30_000)

// ===== api hours/days mapping =====
const apiHours = computed(() => range.value === 1 ? 24 : range.value === 7 ? 168 : 720)
const apiDays = computed(() => range.value === 1 ? 1 : range.value)

// ===== KPI cards =====
type Tone = 'critical' | 'warning' | 'success' | 'info'
interface KpiCard {
  label: string
  value: string
  tone: Tone
  delta?: { up: boolean; pct: number } | null
}

const kpis = computed<KpiCard[]>(() => [
  {
    label: t('dashboard.activeAlerts'),
    value: formatNumber(stats.value.active_alerts),
    tone: stats.value.active_alerts > 0 ? 'critical' : 'success',
  },
  {
    label: 'MTTA',
    value: formatMMSS(mttrStats.value.mtta?.p50 ?? -1),
    tone: 'success',
  },
  {
    label: 'MTTR',
    value: formatMMSS(mttrStats.value.mttr?.p50 ?? -1),
    tone: 'success',
  },
  {
    label: t('dashboard.resolvedToday'),
    value: formatNumber(stats.value.resolved_today),
    tone: 'info',
  },
])

// ===== theme-aware chart palette =====
const isLightTheme = shallowRef<boolean>(typeof document !== 'undefined' && document.body.classList.contains('light-theme'))
let themeObserver: MutationObserver | null = null
onMounted(() => {
  if (typeof document === 'undefined') return
  themeObserver = new MutationObserver(() => {
    isLightTheme.value = document.body.classList.contains('light-theme')
  })
  themeObserver.observe(document.body, { attributes: true, attributeFilter: ['class'] })
})
onUnmounted(() => { themeObserver?.disconnect() })

function getChartColor(token: string): string {
  if (typeof document === 'undefined') return '#000000'
  return getComputedStyle(document.documentElement).getPropertyValue(token).trim()
}

function hexToRgba(hex: string, alpha: number): string {
  hex = hex.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  return `rgba(${r},${g},${b},${alpha})`
}

const chartPalette = computed(() => {
  // light-on-light is unreadable: switch to slate-on-light when light theme is active.
  // Values mirror the WCAG-compliant text tokens defined in global.css `body.light-theme`.
  const light = isLightTheme.value
  return {
    tooltipBg: light ? 'rgba(15,23,42,0.92)' : 'rgba(0,0,0,0.85)',
    tooltipText: light ? '#ffffff' : '#ffffff',
    legend: light ? 'rgba(15,23,42,0.72)' : 'rgba(255,255,255,0.7)',
    axisLabel: light ? 'rgba(15,23,42,0.56)' : 'rgba(255,255,255,0.5)',
    axisLine: light ? 'rgba(15,23,42,0.10)' : 'rgba(255,255,255,0.08)',
    splitLine: light ? 'rgba(15,23,42,0.06)' : 'rgba(255,255,255,0.05)',
    pieCenterPrimary: light ? 'rgba(15,23,42,0.92)' : 'rgba(255,255,255,0.92)',
    pieCenterMuted: light ? 'rgba(15,23,42,0.50)' : 'rgba(255,255,255,0.45)',
    // Severity colors (resolved from CSS design tokens at runtime)
    critical: getChartColor('--sre-critical'),
    warning: getChartColor('--sre-warning'),
    info: getChartColor('--sre-info'),
    success: getChartColor('--sre-success'),
    fired: getChartColor('--sre-critical'),
    resolved: getChartColor('--sre-success'),
  }
})

// ===== chart font config =====
const chartFont = {
  fontFamily: 'var(--sre-font-sans)',
  fontFeatureSettings: '"tnum"',
}

// ===== charts =====
const trendChartOption = computed(() => ({
  backgroundColor: 'transparent',
  tooltip: {
    trigger: 'axis',
    backgroundColor: chartPalette.value.tooltipBg,
    borderColor: 'transparent',
    textStyle: { color: chartPalette.value.tooltipText, fontSize: 12, ...chartFont },
  },
  legend: {
    data: [t('dashboard.fired'), t('dashboard.resolved')],
    bottom: 0,
    textStyle: { color: chartPalette.value.legend, fontSize: 11, ...chartFont },
    itemWidth: 14,
    itemHeight: 2,
    icon: 'rect',
  },
  grid: { left: 8, right: 12, bottom: 36, top: 12, containLabel: true },
  xAxis: {
    type: 'category',
    data: trendData.value.map(d => d.date),
    boundaryGap: false,
    axisLabel: { color: chartPalette.value.axisLabel, fontSize: 11, ...chartFont },
    axisLine: { lineStyle: { color: chartPalette.value.axisLine } },
    axisTick: { show: false },
  },
  yAxis: {
    type: 'value',
    axisLabel: { color: chartPalette.value.axisLabel, fontSize: 11, ...chartFont },
    axisLine: { show: false },
    axisTick: { show: false },
    splitLine: { lineStyle: { color: chartPalette.value.splitLine, type: 'dashed' } },
  },
  series: [
    {
      name: t('dashboard.fired'),
      type: 'line',
      smooth: false,
      showSymbol: false,
      data: trendData.value.map(d => d.fired_count),
      lineStyle: { color: chartPalette.value.fired, width: 1.5 },
      itemStyle: { color: chartPalette.value.fired },
    },
    {
      name: t('dashboard.resolved'),
      type: 'line',
      smooth: false,
      showSymbol: false,
      data: trendData.value.map(d => d.resolved_count),
      lineStyle: { color: chartPalette.value.resolved, width: 1.5 },
      itemStyle: { color: chartPalette.value.resolved },
      areaStyle: {
        color: {
          type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: hexToRgba(chartPalette.value.success, 0.25) },
            { offset: 1, color: 'transparent' },
          ],
        },
      },
    },
  ],
}))

const severityChartOption = computed(() => {
  const sev = stats.value.severity_breakdown || { critical: 0, warning: 0, info: 0 }
  const total = (sev.critical ?? 0) + (sev.warning ?? 0) + (sev.info ?? 0)
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      backgroundColor: chartPalette.value.tooltipBg,
      borderColor: 'transparent',
      textStyle: { color: chartPalette.value.tooltipText, fontSize: 12, ...chartFont },
      formatter: '{b}: {c} ({d}%)',
    },
    series: [{
      type: 'pie',
      radius: ['56%', '82%'],
      center: ['50%', '50%'],
      avoidLabelOverlap: false,
      itemStyle: { borderColor: 'var(--sre-bg-card)', borderWidth: 2 },
      label: {
        show: true,
        position: 'center',
        formatter: () => `{n|${total}}\n{l|Active}`,
        rich: {
          n: { fontSize: 22, fontWeight: 600, color: chartPalette.value.pieCenterPrimary, ...chartFont },
          l: { fontSize: 10, color: chartPalette.value.pieCenterMuted, letterSpacing: 1, padding: [4, 0, 0, 0] },
        },
      },
      emphasis: { label: { show: true } },
      labelLine: { show: false },
      data: [
        { value: sev.critical ?? 0, name: t('alert.critical'), itemStyle: { color: chartPalette.value.critical } },
        { value: sev.warning ?? 0,  name: t('alert.warning'),  itemStyle: { color: chartPalette.value.warning } },
        { value: sev.info ?? 0,     name: t('alert.info'),     itemStyle: { color: chartPalette.value.info } },
      ],
    }],
  }
})

const topRulesMax = computed(() => topRules.value.reduce((m, r) => Math.max(m, r.count), 0) || 1)

// ===== fetching =====
async function refresh() {
  refreshing.value = true
  errorMsg.value = ''
  try {
    const [statsRes, mttrRes, trendRes, topRes] = await Promise.allSettled([
      dashboardApi.getStats(),
      dashboardApi.getMTTRStats(apiHours.value),
      dashboardApi.getAlertTrend(apiDays.value),
      dashboardApi.getTopRules(apiDays.value, 8),
    ])
    if (statsRes.status === 'fulfilled') stats.value = statsRes.value.data.data
    if (mttrRes.status === 'fulfilled') mttrStats.value = mttrRes.value.data.data
    if (trendRes.status === 'fulfilled') trendData.value = trendRes.value.data.data || []
    if (topRes.status === 'fulfilled') topRules.value = topRes.value.data.data || []

    const failed = [statsRes, mttrRes, trendRes, topRes].filter(r => r.status === 'rejected')
    if (failed.length === 4) {
      errorMsg.value = t('dashboard.loadFailed')
    }
    lastSyncAt.value = Date.now()
    tickRelative()
  } catch (err: any) {
    errorMsg.value = err?.message || t('dashboard.loadFailed')
    message.error(errorMsg.value)
  } finally {
    refreshing.value = false
  }
}

function onRangeChange(v: 1 | 7 | 30) {
  range.value = v
  refresh()
}

onMounted(refresh)
</script>

<template>
  <div class="dashboard">
    <!-- Header -->
    <header class="dash-header">
      <div class="dash-header__left">
        <h1 class="dash-title">Dashboard</h1>
        <div class="dash-subtitle tnum">Last sync · {{ lastSyncText }}</div>
      </div>
      <div class="dash-header__right">
        <n-radio-group :value="range" size="small" @update:value="onRangeChange">
          <n-radio-button :value="1">{{ t('dashboard.window24h') }}</n-radio-button>
          <n-radio-button :value="7">{{ t('dashboard.last7d') }}</n-radio-button>
          <n-radio-button :value="30">{{ t('dashboard.last30d') }}</n-radio-button>
        </n-radio-group>
        <n-button quaternary circle size="small" :loading="refreshing" @click="refresh">
          <template #icon><n-icon :component="RefreshOutline" /></template>
        </n-button>
      </div>
    </header>

    <n-alert v-if="errorMsg" type="error" :show-icon="true" closable class="dash-error" @close="errorMsg = ''">
      {{ errorMsg }}
    </n-alert>

    <!-- KPI Row -->
    <section class="kpi-grid sre-stagger">
      <div
        v-for="(k, i) in kpis"
        :key="i"
        class="kpi-card sre-lift"
        :style="{ '--sre-stagger-i': i }"
      >
        <div class="kpi-value sre-stat-value tnum">{{ refreshing && k.value === '0' ? '—' : k.value }}</div>
        <div class="sre-label-eyebrow">{{ k.label }}</div>
        <div class="kpi-stripe" :data-tone="k.tone"></div>
      </div>
    </section>

    <!-- Alert Trend -->
    <section class="panel">
      <div class="panel-header">
        <span class="sre-label-eyebrow">Alert Trend</span>
      </div>
      <div class="panel-chart">
        <v-chart :option="trendChartOption" autoresize style="height: 280px" />
      </div>
    </section>

    <!-- Two-up: Top rules + Severity -->
    <section class="two-up">
      <div class="panel">
        <div class="panel-header">
          <span class="sre-label-eyebrow">Top Noisy Rules</span>
        </div>
        <div class="rules-list">
          <div v-if="!topRules.length" class="rules-empty">{{ t('dashboard.noData') }}</div>
          <div v-for="r in topRules" :key="r.alert_name" class="rule-row">
            <div class="rule-name" :title="r.alert_name">{{ r.alert_name }}</div>
            <div class="rule-bar">
              <div class="rule-bar__fill" :style="{ width: ((r.count / topRulesMax) * 100) + '%' }"></div>
            </div>
            <div class="rule-count tnum">{{ r.count }}</div>
          </div>
        </div>
      </div>

      <div class="panel">
        <div class="panel-header">
          <span class="sre-label-eyebrow">Severity Distribution</span>
        </div>
        <div class="severity-body">
          <v-chart :option="severityChartOption" autoresize style="height: 220px; flex: 1; min-width: 0" />
          <div class="sev-legend">
            <div class="sev-item">
              <span class="sev-swatch sev-swatch--critical"></span>
              <span class="sev-name">{{ t('alert.critical') }}</span>
              <span class="sev-num tnum">{{ stats.severity_breakdown?.critical ?? 0 }}</span>
            </div>
            <div class="sev-item">
              <span class="sev-swatch sev-swatch--warning"></span>
              <span class="sev-name">{{ t('alert.warning') }}</span>
              <span class="sev-num tnum">{{ stats.severity_breakdown?.warning ?? 0 }}</span>
            </div>
            <div class="sev-item">
              <span class="sev-swatch sev-swatch--info"></span>
              <span class="sev-name">{{ t('alert.info') }}</span>
              <span class="sev-num tnum">{{ stats.severity_breakdown?.info ?? 0 }}</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.dashboard {
  max-width: 1440px;
  font-family: var(--sre-font-sans);
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* ===== Header ===== */
.dash-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.dash-header__left { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.dash-header__right { display: flex; align-items: center; gap: 8px; }
.dash-title {
  font-family: var(--sre-font-sans);
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--sre-text-primary);
  margin: 0;
  line-height: 1.2;
}
.dash-subtitle {
  font-size: 13px;
  color: var(--sre-text-secondary);
  font-feature-settings: "tnum";
}
.dash-error { margin-bottom: -8px; }

/* ===== KPI ===== */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
}
.kpi-card {
  position: relative;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
  padding: 20px;
  overflow: hidden;
  transition: border-color var(--sre-duration-base) ease, transform var(--sre-duration-base) ease;
}
.kpi-value {
  color: var(--sre-text-primary);
  margin-bottom: 8px;
  font-family: var(--sre-font-sans);
  font-size: 32px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.1;
}
.kpi-stripe {
  position: absolute;
  left: 0; right: 0; bottom: 0;
  height: 3px;
  background: var(--sre-text-tertiary);
}
.kpi-stripe[data-tone="critical"] { background: var(--sre-critical); }
.kpi-stripe[data-tone="warning"]  { background: var(--sre-warning); }
.kpi-stripe[data-tone="success"]  { background: var(--sre-primary); }
.kpi-stripe[data-tone="info"]     { background: var(--sre-info); }

/* ===== Panel (chart container) ===== */
.panel {
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
  padding: 20px 24px;
}
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.panel-chart { width: 100%; }

/* ===== Two-up ===== */
.two-up {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
@media (max-width: 960px) {
  .two-up { grid-template-columns: 1fr; }
}

/* ===== Top noisy rules ===== */
.rules-list { display: flex; flex-direction: column; }
.rules-empty {
  padding: 24px 0;
  text-align: center;
  color: var(--sre-text-tertiary);
  font-size: 13px;
}
.rule-row {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 2fr) auto;
  gap: 16px;
  align-items: center;
  padding: 12px 8px;
  border-radius: var(--sre-radius-sm);
  transition: background-color 150ms ease;
}
.rule-row:hover { background: var(--sre-bg-hover); }
.rule-name {
  font-family: var(--sre-font-sans);
  font-size: 13px;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rule-bar {
  height: 4px;
  border-radius: 2px;
  background: var(--sre-bg-elevated);
  overflow: hidden;
}
.rule-bar__fill {
  height: 100%;
  background: var(--sre-primary);
  border-radius: 2px;
  transition: width 600ms cubic-bezier(0.4, 0, 0.2, 1);
}
.rule-count {
  font-family: var(--sre-font-mono);
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-primary);
  font-variant-numeric: tabular-nums;
  min-width: 36px;
  text-align: right;
}

/* ===== Severity ===== */
.severity-body {
  display: flex;
  align-items: center;
  gap: 16px;
}
.sev-legend {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex-shrink: 0;
  min-width: 120px;
}
.sev-item {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}
.sev-swatch {
  width: 8px;
  height: 8px;
  border-radius: 2px;
  flex-shrink: 0;
}
.sev-swatch--critical { background: var(--sre-critical); }
.sev-swatch--warning  { background: var(--sre-warning); }
.sev-swatch--info     { background: var(--sre-info); }
.sev-name {
  flex: 1;
  color: var(--sre-text-secondary);
  text-transform: capitalize;
}
.sev-num {
  font-family: var(--sre-font-mono);
  font-feature-settings: "tnum";
  font-weight: 600;
  color: var(--sre-text-primary);
  min-width: 28px;
  text-align: right;
}
</style>
