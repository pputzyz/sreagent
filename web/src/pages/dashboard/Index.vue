<script setup lang="ts">
import { shallowRef, computed, onMounted, onUnmounted, type Component } from 'vue'
import { useMessage, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart, LineChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { dashboardApi } from '@/api'
import type { DashboardStats, MTTRStats, AlertTrendPoint, TopRuleItem } from '@/types'
import {
  RefreshOutline, PulseOutline, TimerOutline, CheckmarkCircleOutline, TrendingUpOutline,
} from '@vicons/ionicons5'

use([CanvasRenderer, PieChart, LineChart, TooltipComponent, LegendComponent, GridComponent])

const message = useMessage()
const { t } = useI18n()

const range = shallowRef<1 | 7 | 30>(7)
const refreshing = shallowRef(false)
const lastSyncAt = shallowRef<number>(Date.now())
const errorMsg = shallowRef<string>('')

const stats = shallowRef<DashboardStats>({
  total_datasources: 0, total_rules: 0, active_alerts: 0,
  resolved_today: 0, total_users: 0, total_teams: 0,
  severity_breakdown: { critical: 0, warning: 0, info: 0 },
})

const emptyMetric = { mean: -1, p50: -1, p95: -1, count: 0 }
const mttrStats = shallowRef<MTTRStats>({
  window_hours: 24, mtta: { ...emptyMetric }, mttr: { ...emptyMetric },
  by_severity: [], mtta_seconds: -1, mttr_seconds: -1,
  acked_count: 0, resolved_count: 0,
})

const trendData = shallowRef<AlertTrendPoint[]>([])
const topRules = shallowRef<TopRuleItem[]>([])

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
  if (min < 60) return `${min}m ago`
  const h = Math.floor(min / 60)
  return `${h}h ago`
}

const lastSyncText = shallowRef(formatRelative(Date.now()))
let syncInterval: ReturnType<typeof setInterval>
onMounted(() => { syncInterval = setInterval(() => { lastSyncText.value = formatRelative(lastSyncAt.value) }, 30_000) })
onUnmounted(() => clearInterval(syncInterval))

const apiHours = computed(() => range.value === 1 ? 24 : range.value === 7 ? 168 : 720)
const apiDays = computed(() => range.value === 1 ? 1 : range.value)

type KpiDef = { label: string; value: string; tone: 'critical' | 'warning' | 'success' | 'info'; icon: Component; sub?: string }
const kpis = computed<KpiDef[]>(() => [
  {
    label: t('dashboard.activeAlerts'),
    value: refreshing.value ? '—' : formatNumber(stats.value.active_alerts),
    tone: stats.value.active_alerts > 0 ? 'critical' : 'success',
    icon: PulseOutline,
  },
  {
    label: 'MTTA',
    value: formatMMSS(mttrStats.value.mtta?.p50 ?? -1),
    tone: 'info',
    icon: TimerOutline,
    sub: `avg ${formatMMSS(mttrStats.value.mtta?.mean ?? -1)}`,
  },
  {
    label: 'MTTR',
    value: formatMMSS(mttrStats.value.mttr?.p50 ?? -1),
    tone: 'success',
    icon: CheckmarkCircleOutline,
    sub: `avg ${formatMMSS(mttrStats.value.mttr?.mean ?? -1)}`,
  },
  {
    label: t('dashboard.resolvedToday'),
    value: formatNumber(stats.value.resolved_today),
    tone: 'success',
    icon: TrendingUpOutline,
  },
])

// Theme-aware chart palette
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

function chartToken(name: string): string {
  if (typeof document === 'undefined') return '#000'
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

function hexToRgba(hex: string, alpha: number): string {
  hex = hex.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  return `rgba(${r},${g},${b},${alpha})`
}

const cp = computed(() => {
  const light = isLightTheme.value
  return {
    tooltipBg: light ? 'rgba(17,24,39,0.92)' : 'rgba(15,23,42,0.92)',
    tooltipText: '#f1f5f9',
    legend: light ? 'rgba(17,24,39,0.65)' : 'rgba(203,213,225,0.65)',
    axisLabel: light ? 'rgba(17,24,39,0.50)' : 'rgba(203,213,225,0.45)',
    axisLine: light ? 'rgba(17,24,39,0.10)' : 'rgba(203,213,225,0.07)',
    splitLine: light ? 'rgba(17,24,39,0.05)' : 'rgba(203,213,225,0.04)',
    pieCenter: light ? 'rgba(17,24,39,0.92)' : 'rgba(203,213,225,0.90)',
    pieMuted: light ? 'rgba(17,24,39,0.45)' : 'rgba(203,213,225,0.40)',
    critical: chartToken('--sre-critical'),
    warning: chartToken('--sre-warning'),
    info: chartToken('--sre-info'),
    success: chartToken('--sre-success'),
  }
})

const chartFont = { fontFamily: 'var(--sre-font-sans)' }

const trendChartOption = computed(() => ({
  backgroundColor: 'transparent',
  tooltip: {
    trigger: 'axis' as const,
    backgroundColor: cp.value.tooltipBg,
    borderColor: 'transparent',
    textStyle: { color: cp.value.tooltipText, fontSize: 12, ...chartFont },
  },
  legend: {
    data: [t('dashboard.fired'), t('dashboard.resolved')],
    bottom: 0,
    textStyle: { color: cp.value.legend, fontSize: 11, ...chartFont },
    itemWidth: 14, itemHeight: 2, icon: 'rect' as const,
  },
  grid: { left: 4, right: 8, bottom: 36, top: 8, containLabel: true },
  xAxis: {
    type: 'category' as const,
    data: trendData.value.map(d => d.date),
    boundaryGap: false,
    axisLabel: { color: cp.value.axisLabel, fontSize: 10, ...chartFont },
    axisLine: { lineStyle: { color: cp.value.axisLine } },
    axisTick: { show: false },
  },
  yAxis: {
    type: 'value' as const,
    axisLabel: { color: cp.value.axisLabel, fontSize: 10, ...chartFont },
    axisLine: { show: false },
    axisTick: { show: false },
    splitLine: { lineStyle: { color: cp.value.splitLine, type: 'dashed' as const } },
  },
  series: [
    {
      name: t('dashboard.fired'),
      type: 'line' as const,
      smooth: false, showSymbol: false,
      data: trendData.value.map(d => d.fired_count),
      lineStyle: { color: cp.value.critical, width: 1.5 },
      itemStyle: { color: cp.value.critical },
    },
    {
      name: t('dashboard.resolved'),
      type: 'line' as const,
      smooth: false, showSymbol: false,
      data: trendData.value.map(d => d.resolved_count),
      lineStyle: { color: cp.value.success, width: 1.5 },
      itemStyle: { color: cp.value.success },
      areaStyle: {
        color: {
          type: 'linear' as const, x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: hexToRgba(cp.value.success, 0.20) },
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
      trigger: 'item' as const,
      backgroundColor: cp.value.tooltipBg,
      borderColor: 'transparent',
      textStyle: { color: cp.value.tooltipText, fontSize: 12, ...chartFont },
      formatter: '{b}: {c} ({d}%)' as const,
    },
    series: [{
      type: 'pie' as const,
      radius: ['55%', '80%'],
      center: ['50%', '50%'],
      avoidLabelOverlap: false,
      itemStyle: { borderColor: 'transparent', borderWidth: 3 },
      label: {
        show: true, position: 'center' as const,
        formatter: () => `{n|${total}}\n{l|${t('dashboard.active')}}`,
        rich: {
          n: { fontSize: 24, fontWeight: 600, color: cp.value.pieCenter, ...chartFont },
          l: { fontSize: 10, color: cp.value.pieMuted, letterSpacing: 1, padding: [4, 0, 0, 0] },
        },
      },
      emphasis: { label: { show: true }, scaleSize: 6 },
      labelLine: { show: false },
      data: [
        { value: sev.critical ?? 0, name: t('alert.critical'), itemStyle: { color: cp.value.critical } },
        { value: sev.warning ?? 0, name: t('alert.warning'), itemStyle: { color: cp.value.warning } },
        { value: sev.info ?? 0, name: t('alert.info'), itemStyle: { color: cp.value.info } },
      ],
    }],
  }
})

const topRulesMax = computed(() => topRules.value.reduce((m, r) => Math.max(m, r.count), 0) || 1)

async function refresh() {
  refreshing.value = true; errorMsg.value = ''
  try {
    const [sr, mr, tr, top] = await Promise.allSettled([
      dashboardApi.getStats(),
      dashboardApi.getMTTRStats(apiHours.value),
      dashboardApi.getAlertTrend(apiDays.value),
      dashboardApi.getTopRules(apiDays.value, 8),
    ])
    if (sr.status === 'fulfilled') stats.value = sr.value.data.data
    if (mr.status === 'fulfilled') mttrStats.value = mr.value.data.data
    if (tr.status === 'fulfilled') trendData.value = tr.value.data.data || []
    if (top.status === 'fulfilled') topRules.value = top.value.data.data || []
    const failed = [sr, mr, tr, top].filter(r => r.status === 'rejected')
    if (failed.length === 4) errorMsg.value = t('dashboard.loadFailed')
    lastSyncAt.value = Date.now()
    lastSyncText.value = formatRelative(Date.now())
  } catch (err: any) {
    errorMsg.value = err?.message || t('dashboard.loadFailed')
    message.error(errorMsg.value)
  } finally {
    refreshing.value = false
  }
}

function onRangeChange(v: 1 | 7 | 30) { range.value = v; refresh() }

onMounted(refresh)
</script>

<template>
  <div class="dashboard">
    <!-- Alert Banner -->
    <n-alert v-if="errorMsg" type="error" :show-icon="true" closable class="dash-error" @close="errorMsg = ''">
      {{ errorMsg }}
    </n-alert>

    <!-- KPI Row -->
    <section class="kpi-grid sre-stagger">
      <div
        v-for="k in kpis"
        :key="k.label"
        class="kpi-card sre-lift"
        :data-tone="k.tone"
      >
        <div class="kpi-icon-wrap">
          <n-icon :component="k.icon" :size="20" />
        </div>
        <div class="kpi-body">
          <div class="kpi-value number-display">{{ k.value }}</div>
          <div class="kpi-label">{{ k.label }}</div>
          <div v-if="k.sub" class="kpi-sub text-muted">{{ k.sub }}</div>
        </div>
      </div>
    </section>

    <!-- Alert Trend -->
    <section class="chart-card surface-clay">
      <div class="chart-card__header">
        <span class="chart-card__title">{{ t('dashboard.alertTrend') }}</span>
        <div class="chart-card__actions">
          <n-radio-group :value="range" size="small" @update:value="onRangeChange">
            <n-radio-button :value="1">{{ t('dashboard.window24h') }}</n-radio-button>
            <n-radio-button :value="7">{{ t('dashboard.last7d') }}</n-radio-button>
            <n-radio-button :value="30">{{ t('dashboard.last30d') }}</n-radio-button>
          </n-radio-group>
          <n-button quaternary circle size="tiny" :loading="refreshing" @click="refresh">
            <template #icon><n-icon :component="RefreshOutline" :size="14" /></template>
          </n-button>
        </div>
      </div>
      <div class="chart-card__body">
        <v-chart v-if="trendData.length" :option="trendChartOption" autoresize style="height: 280px" />
        <div v-else class="chart-empty text-muted">{{ t('dashboard.noData') }}</div>
      </div>
    </section>

    <!-- Two-up: Top rules + Severity -->
    <section class="two-up">
      <!-- Top Noisy Rules -->
      <div class="chart-card surface-clay">
        <div class="chart-card__header">
          <span class="chart-card__title">{{ t('dashboard.topNoisyRules') }}</span>
        </div>
        <div class="chart-card__body">
          <div v-if="!topRules.length" class="chart-empty text-muted">{{ t('dashboard.noData') }}</div>
          <div v-else class="rules-list">
            <div v-for="(r, i) in topRules" :key="r.alert_name" class="rule-row">
              <span class="rule-rank">{{ i + 1 }}</span>
              <span class="rule-name" :title="r.alert_name">{{ r.alert_name }}</span>
              <div class="rule-bar-track">
                <div
                  class="rule-bar-fill"
                  :style="{ width: Math.max(((r.count / topRulesMax) * 100), 2) + '%' }"
                />
              </div>
              <span class="rule-count number-display">{{ r.count }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Severity Distribution -->
      <div class="chart-card surface-clay">
        <div class="chart-card__header">
          <span class="chart-card__title">{{ t('dashboard.severityDistribution') }}</span>
        </div>
        <div class="chart-card__body severity-body">
          <v-chart :option="severityChartOption" autoresize style="height: 220px; flex: 1; min-width: 0" />
          <div class="sev-legend">
            <div class="sev-item">
              <span class="sev-swatch sev-swatch--critical" />
              <span class="sev-name">{{ t('alert.critical') }}</span>
              <span class="sev-num number-display">{{ stats.severity_breakdown?.critical ?? 0 }}</span>
            </div>
            <div class="sev-item">
              <span class="sev-swatch sev-swatch--warning" />
              <span class="sev-name">{{ t('alert.warning') }}</span>
              <span class="sev-num number-display">{{ stats.severity_breakdown?.warning ?? 0 }}</span>
            </div>
            <div class="sev-item">
              <span class="sev-swatch sev-swatch--info" />
              <span class="sev-name">{{ t('alert.info') }}</span>
              <span class="sev-num number-display">{{ stats.severity_breakdown?.info ?? 0 }}</span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Last sync -->
    <div class="dash-footer">
      <span class="text-muted">{{ t('dashboard.lastSync') }} · {{ lastSyncText }}</span>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  max-width: 1440px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  font-family: var(--sre-font-sans);
}

.dash-error { margin-bottom: -8px; }

/* ===== KPI Row ===== */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.kpi-card {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: var(--sre-card-pad-compact) 18px;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-lg);
  transition: border-color var(--sre-duration-base) var(--sre-ease-out),
              box-shadow var(--sre-duration-base) var(--sre-ease-out),
              transform var(--sre-duration-base) var(--sre-ease-out);
  position: relative;
  overflow: hidden;
}

/* Colored top line per tone */
.kpi-card::after {
  content: '';
  position: absolute;
  top: 0; left: 12px; right: 12px;
  height: 3px;
  border-radius: 0 0 3px 3px;
  background: var(--sre-text-tertiary);
  transition: background var(--sre-duration-fast) var(--sre-ease-out);
}
.kpi-card[data-tone="critical"]::after { background: var(--sre-critical); }
.kpi-card[data-tone="warning"]::after  { background: var(--sre-warning); }
.kpi-card[data-tone="success"]::after  { background: var(--sre-success); }
.kpi-card[data-tone="info"]::after     { background: var(--sre-info); }

.kpi-card:hover {
  border-color: var(--sre-border-strong);
  box-shadow: var(--sre-shadow-md);
  transform: translateY(-1px);
}

.kpi-icon-wrap {
  width: 42px; height: 42px;
  border-radius: var(--sre-radius-md);
  background: var(--sre-bg-elevated);
  border: 1px solid var(--sre-border);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--sre-text-secondary);
  transition: color var(--sre-duration-fast) var(--sre-ease-out),
              border-color var(--sre-duration-fast) var(--sre-ease-out),
              background var(--sre-duration-fast) var(--sre-ease-out);
}
.kpi-card[data-tone="critical"] .kpi-icon-wrap { color: var(--sre-critical); background: var(--sre-critical-soft); border-color: transparent; }
.kpi-card[data-tone="warning"]  .kpi-icon-wrap { color: var(--sre-warning); background: var(--sre-warning-soft); border-color: transparent; }
.kpi-card[data-tone="success"]  .kpi-icon-wrap { color: var(--sre-primary); background: var(--sre-primary-soft); border-color: transparent; }
.kpi-card[data-tone="info"]     .kpi-icon-wrap { color: var(--sre-info); background: var(--sre-info-soft); border-color: transparent; }

.kpi-body { flex: 1; min-width: 0; }

.kpi-value {
  font-size: 28px;
  font-weight: 700;
  line-height: 1.1;
  color: var(--sre-text-primary);
  letter-spacing: -0.02em;
}

.kpi-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--sre-text-secondary);
  margin-top: 2px;
}

.kpi-sub {
  font-size: 11px;
  margin-top: 1px;
}

/* ===== Chart Card ===== */
.chart-card {
  padding: 0;
  overflow: hidden;
}

.chart-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sre-card-pad-relaxed) var(--sre-card-pad-relaxed) 0;
}

.chart-card__title {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
  letter-spacing: -0.005em;
}

.chart-card__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.chart-card__body {
  padding: 14px var(--sre-card-pad-relaxed) var(--sre-card-pad-relaxed);
}

.chart-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  font-size: 13px;
}

/* ===== Two-up ===== */
.two-up {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--sre-card-pad-relaxed);
}

/* ===== Top Noisy Rules ===== */
.rules-list {
  display: flex;
  flex-direction: column;
}

.rule-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 8px;
  border-radius: var(--sre-radius-sm);
  transition: background-color var(--sre-duration-fast) var(--sre-ease-out);
}
.rule-row:hover { background: var(--sre-bg-hover); }

.rule-rank {
  width: 22px; height: 22px;
  border-radius: 6px;
  background: var(--sre-bg-elevated);
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.rule-row:nth-child(1) .rule-rank { background: var(--sre-primary-soft); color: var(--sre-primary); }
.rule-row:nth-child(2) .rule-rank { background: var(--sre-accent-soft); color: var(--sre-accent); }
.rule-row:nth-child(3) .rule-rank { background: var(--sre-info-soft); color: var(--sre-info); }

.rule-name {
  flex: 1;
  font-size: 13px;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.rule-bar-track {
  width: 100px;
  height: 5px;
  border-radius: 3px;
  background: var(--sre-bg-elevated);
  overflow: hidden;
  flex-shrink: 0;
}

.rule-bar-fill {
  height: 100%;
  background: var(--sre-primary);
  border-radius: 3px;
  transition: width 500ms var(--sre-ease-out);
}

.rule-count {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
  min-width: 32px;
  text-align: right;
  flex-shrink: 0;
}

/* ===== Severity ===== */
.severity-body {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sev-legend {
  display: flex;
  flex-direction: column;
  gap: 14px;
  flex-shrink: 0;
  min-width: 110px;
}

.sev-item {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}

.sev-swatch {
  width: 10px; height: 10px;
  border-radius: 3px;
  flex-shrink: 0;
}
.sev-swatch--critical { background: var(--sre-critical); }
.sev-swatch--warning  { background: var(--sre-warning); }
.sev-swatch--info     { background: var(--sre-info); }

.sev-name {
  flex: 1;
  color: var(--sre-text-secondary);
}

.sev-num {
  font-weight: 600;
  color: var(--sre-text-primary);
  min-width: 24px;
  text-align: right;
}

/* ===== Footer ===== */
.dash-footer {
  text-align: right;
  font-size: 11px;
}

/* ===== Responsive ===== */
@media (max-width: 1100px) {
  .kpi-grid { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 768px) {
  .kpi-grid { grid-template-columns: 1fr; }
  .two-up { grid-template-columns: 1fr; }
  .rule-bar-track { width: 60px; }
  .severity-body { flex-direction: column; }
  .sev-legend { flex-direction: row; flex-wrap: wrap; gap: 12px 20px; min-width: unset; }
}
</style>
