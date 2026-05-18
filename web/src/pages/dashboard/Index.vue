<script setup lang="ts">
import { shallowRef, computed, onMounted, onUnmounted } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { dashboardApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import type { DashboardStats, MTTRStats, AlertTrendPoint, TopRuleItem } from '@/types'
import {
  RefreshOutline, PulseOutline, TimerOutline, CheckmarkCircleOutline,
} from '@vicons/ionicons5'

const message = useMessage()
const { t } = useI18n()
const router = useRouter()

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
  if (sec < 60) return t('common.secsAgo', { n: sec })
  const min = Math.floor(sec / 60)
  if (min < 60) return t('common.minsAgo', { n: min })
  const h = Math.floor(min / 60)
  return t('common.hoursAgo', { n: h })
}

const lastSyncText = shallowRef(formatRelative(Date.now()))
let syncInterval: ReturnType<typeof setInterval>
onMounted(() => { syncInterval = setInterval(() => { lastSyncText.value = formatRelative(lastSyncAt.value) }, 30_000) })
onUnmounted(() => clearInterval(syncInterval))

// Severity summary
const sevTotal = computed(() => {
  const s = stats.value.severity_breakdown
  return (s?.critical ?? 0) + (s?.warning ?? 0) + (s?.info ?? 0)
})
const sevPct = (key: 'critical' | 'warning' | 'info') => {
  if (sevTotal.value === 0) return 0
  return Math.round(((stats.value.severity_breakdown?.[key] ?? 0) / sevTotal.value) * 100)
}

// Trend chart helpers
const trendMax = computed(() => Math.max(...trendData.value.map(d => Math.max(d.fired_count, d.resolved_count)), 1))

function trendAreaPath(data: number[], max: number, w: number, h: number): string {
  if (data.length < 2) return ''
  const pts = data.map((v, i) => [(i / (data.length - 1)) * w, h - (v / max) * h * 0.85 - h * 0.05])
  let d = `M${pts[0][0]},${pts[0][1]}`
  for (let i = 0; i < pts.length - 1; i++) {
    const cx = (pts[i][0] + pts[i + 1][0]) / 2
    d += ` C${cx},${pts[i][1]} ${cx},${pts[i + 1][1]} ${pts[i + 1][0]},${pts[i + 1][1]}`
  }
  return d
}

function trendFillPath(data: number[], max: number, w: number, h: number): string {
  const line = trendAreaPath(data, max, w, h)
  if (!line) return ''
  const lastX = ((data.length - 1) / (data.length - 1)) * w
  return `${line} L${lastX},${h} L0,${h} Z`
}

// Hover state for trend chart
const hoverIdx = shallowRef(-1)
const chartW = 600
const chartH = 170

function onChartMove(e: MouseEvent) {
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  const x = e.clientX - rect.left
  hoverIdx.value = Math.round((x / rect.width) * (trendData.value.length - 1))
}

function onChartLeave() { hoverIdx.value = -1 }

const hoveredPoint = computed(() => {
  const i = hoverIdx.value
  if (i < 0 || i >= trendData.value.length) return null
  return trendData.value[i]
})

// Refresh
async function refresh() {
  refreshing.value = true; errorMsg.value = ''
  try {
    const [sr, mr, tr, top] = await Promise.allSettled([
      dashboardApi.getStats(),
      dashboardApi.getMTTRStats(range.value === 1 ? 24 : range.value === 7 ? 168 : 720),
      dashboardApi.getAlertTrend(range.value === 1 ? 1 : range.value),
      dashboardApi.getTopRules(range.value === 1 ? 1 : range.value, 5),
    ])
    if (sr.status === 'fulfilled') stats.value = sr.value.data.data
    if (mr.status === 'fulfilled') mttrStats.value = mr.value.data.data
    if (tr.status === 'fulfilled') trendData.value = tr.value.data.data || []
    if (top.status === 'fulfilled') topRules.value = top.value.data.data || []
    const failed = [sr, mr, tr, top].filter(r => r.status === 'rejected')
    if (failed.length === 4) errorMsg.value = t('dashboard.loadFailed')
    lastSyncAt.value = Date.now()
  } catch (err: unknown) {
    errorMsg.value = getErrorMessage(err) || t('dashboard.loadFailed')
    message.error(errorMsg.value)
  } finally {
    refreshing.value = false
  }
}

function onRangeChange(v: 1 | 7 | 30) { range.value = v; refresh() }
function cycleRange() { range.value = range.value === 1 ? 7 : range.value === 7 ? 30 : 1; refresh() }

onMounted(refresh)
</script>

<template>
  <div class="dashboard">
    <!-- Error banner -->
    <n-alert v-if="errorMsg" type="error" :show-icon="true" closable class="dash-error" @close="errorMsg = ''">
      {{ errorMsg }}
    </n-alert>

    <div class="bento">
      <!-- ===== TREND CHART ===== -->
      <div class="card card-trend">
        <div class="card-head">
          <div class="card-title">
            <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-primary), var(--sre-coral))">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
            </span>
            {{ t('dashboard.alertTrend') }}
          </div>
          <div class="card-actions">
            <span class="card-tag" @click="cycleRange">{{ range === 1 ? '24h' : range === 7 ? '7d' : '30d' }}</span>
            <n-button quaternary circle size="tiny" :loading="refreshing" @click="refresh">
              <template #icon><n-icon :component="RefreshOutline" :size="12" /></template>
            </n-button>
          </div>
        </div>
        <div class="trend-chart" @mousemove="onChartMove" @mouseleave="onChartLeave">
          <svg v-if="trendData.length" :viewBox="`0 0 ${chartW} ${chartH}`" preserveAspectRatio="none" class="trend-svg">
            <defs>
              <linearGradient id="gFired" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="var(--sre-primary)" stop-opacity="0.25" />
                <stop offset="100%" stop-color="var(--sre-primary)" stop-opacity="0.02" />
              </linearGradient>
              <linearGradient id="gResolved" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="var(--sre-success)" stop-opacity="0.18" />
                <stop offset="100%" stop-color="var(--sre-success)" stop-opacity="0.02" />
              </linearGradient>
            </defs>
            <!-- grid lines -->
            <line v-for="i in 4" :key="i" :x1="0" :y1="(chartH / 4) * i" :x2="chartW" :y2="(chartH / 4) * i" stroke="var(--sre-border)" stroke-width="0.5" />
            <!-- resolved area + line -->
            <path :d="trendFillPath(trendData.map(d => d.resolved_count), trendMax, chartW, chartH)" fill="url(#gResolved)" />
            <path :d="trendAreaPath(trendData.map(d => d.resolved_count), trendMax, chartW, chartH)" fill="none" stroke="var(--sre-success)" stroke-width="2" stroke-linecap="round" />
            <!-- fired area + line -->
            <path :d="trendFillPath(trendData.map(d => d.fired_count), trendMax, chartW, chartH)" fill="url(#gFired)" />
            <path :d="trendAreaPath(trendData.map(d => d.fired_count), trendMax, chartW, chartH)" fill="none" stroke="var(--sre-primary)" stroke-width="2.5" stroke-linecap="round" />
            <!-- hover line -->
            <line v-if="hoverIdx >= 0 && hoverIdx < trendData.length"
              :x1="(hoverIdx / (trendData.length - 1)) * chartW" :y1="0"
              :x2="(hoverIdx / (trendData.length - 1)) * chartW" :y2="chartH"
              stroke="var(--sre-primary)" stroke-width="1" stroke-dasharray="4 3" opacity="0.5" />
          </svg>
          <div v-else class="chart-empty">{{ t('dashboard.noData') }}</div>
          <!-- tooltip -->
          <div v-if="hoveredPoint" class="trend-tooltip" :style="{ left: `${Math.min((hoverIdx / Math.max(trendData.length - 1, 1)) * 100, 85)}%` }">
            <div class="tt-time">{{ hoveredPoint.date }}</div>
            <div class="tt-row"><span class="tt-dot" style="background: var(--sre-primary)"></span>{{ t('dashboard.fired') }} <span class="tt-val">{{ hoveredPoint.fired_count }}</span></div>
            <div class="tt-row"><span class="tt-dot" style="background: var(--sre-success)"></span>{{ t('dashboard.resolved') }} <span class="tt-val">{{ hoveredPoint.resolved_count }}</span></div>
          </div>
        </div>
        <div class="chart-legend">
          <span class="legend-item"><span class="legend-line" style="background: var(--sre-primary)"></span>{{ t('dashboard.fired') }}</span>
          <span class="legend-item"><span class="legend-line" style="background: var(--sre-success)"></span>{{ t('dashboard.resolved') }}</span>
        </div>
      </div>

      <!-- ===== SEVERITY SUMMARY ===== -->
      <div class="card card-severity">
        <div class="card-head">
          <div class="card-title">
            <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-critical), var(--sre-coral))">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/></svg>
            </span>
            {{ t('dashboard.severityDistribution') }}
          </div>
          <span class="card-tag" @click="cycleRange">{{ range === 1 ? '24h' : range === 7 ? '7d' : '30d' }}</span>
        </div>
        <!-- stacked bar -->
        <div class="sev-bar">
          <div class="sev-seg" :style="{ flex: sevPct('critical') }" :title="`P0: ${stats.severity_breakdown?.critical ?? 0}`"></div>
          <div class="sev-seg sev-seg--w" :style="{ flex: sevPct('warning') }" :title="`P1: ${stats.severity_breakdown?.warning ?? 0}`"></div>
          <div class="sev-seg sev-seg--i" :style="{ flex: sevPct('info') }" :title="`P2: ${stats.severity_breakdown?.info ?? 0}`"></div>
        </div>
        <div class="sev-grid">
          <div class="sev-item"><span class="sev-dot" style="background: var(--sre-critical)"></span><span class="sev-label">P0</span><span class="sev-count" style="color: var(--sre-critical)">{{ stats.severity_breakdown?.critical ?? 0 }}</span></div>
          <div class="sev-item"><span class="sev-dot" style="background: var(--sre-warning)"></span><span class="sev-label">P1</span><span class="sev-count" style="color: var(--sre-warning)">{{ stats.severity_breakdown?.warning ?? 0 }}</span></div>
          <div class="sev-item"><span class="sev-dot" style="background: var(--sre-info)"></span><span class="sev-label">P2</span><span class="sev-count" style="color: var(--sre-info)">{{ stats.severity_breakdown?.info ?? 0 }}</span></div>
        </div>
      </div>

      <!-- ===== ACTIVE ALERTS (KPI) ===== -->
      <div class="card card-kpi">
        <div class="card-head">
          <div class="card-title">
            <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-primary), var(--sre-coral))">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/></svg>
            </span>
            {{ t('dashboard.activeAlerts') }}
          </div>
        </div>
        <div class="kpi-stack">
          <div class="kpi-row">
            <div class="kpi-icon" style="background: linear-gradient(135deg, var(--sre-critical), var(--sre-rose-light))"><n-icon :component="PulseOutline" :size="16" color="#fff" /></div>
            <div class="kpi-info"><div class="kpi-value" style="color: var(--sre-critical)">{{ formatNumber(stats.active_alerts) }}</div><div class="kpi-label">{{ t('dashboard.activeAlerts') }}</div></div>
          </div>
          <div class="kpi-row">
            <div class="kpi-icon" style="background: linear-gradient(135deg, var(--sre-success), var(--sre-emerald-light))"><n-icon :component="TimerOutline" :size="16" color="#fff" /></div>
            <div class="kpi-info"><div class="kpi-value" style="color: var(--sre-success)">{{ formatMMSS(mttrStats.mtta?.p50 ?? -1) }}</div><div class="kpi-label">{{ t('dashboard.mtta') }}</div></div>
          </div>
          <div class="kpi-row">
            <div class="kpi-icon" style="background: linear-gradient(135deg, var(--sre-info), var(--sre-sky))"><n-icon :component="CheckmarkCircleOutline" :size="16" color="#fff" /></div>
            <div class="kpi-info"><div class="kpi-value" style="color: var(--sre-info)">{{ formatMMSS(mttrStats.mttr?.p50 ?? -1) }}</div><div class="kpi-label">{{ t('dashboard.mttr') }}</div></div>
          </div>
          <div class="kpi-row">
            <div class="kpi-icon" style="background: linear-gradient(135deg, var(--sre-lavender), var(--sre-violet-light))"><n-icon :component="CheckmarkCircleOutline" :size="16" color="#fff" /></div>
            <div class="kpi-info"><div class="kpi-value" style="color: var(--sre-lavender)">{{ formatNumber(stats.resolved_today) }}</div><div class="kpi-label">{{ t('dashboard.resolvedToday') }}</div></div>
          </div>
        </div>
      </div>

      <!-- ===== TOP RULES ===== -->
      <div class="card card-rules">
        <div class="card-head">
          <div class="card-title">
            <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-lavender), var(--sre-violet-light))">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
            </span>
            {{ t('dashboard.topNoisyRules') }}
          </div>
        </div>
        <div v-if="topRules.length" class="rules-list">
          <div v-for="r in topRules" :key="r.alert_name" class="rule-item" role="button" :aria-label="`${r.alert_name}: ${r.count}`" @click="router.push('/alert/rules')">
            <span class="rule-status active"></span>
            <div class="rule-info"><div class="rule-name">{{ r.alert_name }}</div></div>
            <span class="rule-fire">{{ r.count }}</span>
          </div>
        </div>
        <div v-else class="chart-empty">{{ t('dashboard.noData') }}</div>
      </div>

      <!-- ===== QUICK ACTIONS ===== -->
      <div class="card card-quick">
        <div class="card-head">
          <div class="card-title">
            <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-primary), var(--sre-critical))">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
            </span>
            {{ t('dashboard.quickActions') }}
          </div>
        </div>
        <div class="actions-grid">
          <div class="action-btn" role="button" :aria-label="t('menu.alertRules')" @click="router.push('/alert/rules')">
            <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-critical), var(--sre-rose-light))">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/></svg>
            </div>
            <span class="action-label">{{ t('menu.alertRules') }}</span>
          </div>
          <div class="action-btn" role="button" :aria-label="t('menu.schedule')" @click="router.push('/oncall/schedule')">
            <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-info), var(--sre-sky))">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
            </div>
            <span class="action-label">{{ t('menu.schedule') }}</span>
          </div>
          <div class="action-btn" role="button" :aria-label="t('menu.explore')" @click="router.push('/alert/explore')">
            <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-success), var(--sre-emerald-light))">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
            </div>
            <span class="action-label">{{ t('menu.explore') }}</span>
          </div>
          <div class="action-btn" role="button" :aria-label="t('menu.dashboards')" @click="router.push('/alert/dashboards')">
            <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-lavender), var(--sre-violet-light))">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/></svg>
            </div>
            <span class="action-label">{{ t('menu.dashboards') }}</span>
          </div>
          <div class="action-btn" role="button" :aria-label="t('menu.notifyRules')" @click="router.push('/oncall/config/notify-rules')">
            <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-amber), var(--sre-coral))">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/></svg>
            </div>
            <span class="action-label">{{ t('menu.notifyRules') }}</span>
          </div>
          <div class="action-btn" role="button" :aria-label="t('menu.suppression')" @click="router.push('/alert/suppression')">
            <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-mint), var(--sre-success))">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
            </div>
            <span class="action-label">{{ t('menu.suppression') }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <div class="dash-footer">
      <span class="text-muted" style="font-size: 11px">{{ t('dashboard.lastSync') }} · {{ lastSyncText }}</span>
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

/* ===== BENTO GRID ===== */
.bento {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 16px;
}

.bento > .card {
  animation: card-enter 400ms var(--sre-ease-out) both;
}
.bento > .card:nth-child(1) { animation-delay: 0ms; }
.bento > .card:nth-child(2) { animation-delay: 60ms; }
.bento > .card:nth-child(3) { animation-delay: 120ms; }
.bento > .card:nth-child(4) { animation-delay: 180ms; }
.bento > .card:nth-child(5) { animation-delay: 240ms; }

@keyframes card-enter {
  from { opacity: 0; transform: translateY(12px); }
  to   { opacity: 1; transform: translateY(0); }
}

.card { grid-column: span 12; }

/* Card sizes */
.card-trend    { grid-column: span 8; }
.card-severity { grid-column: span 4; }
.card-kpi      { grid-column: span 4; }
.card-rules    { grid-column: span 4; }
.card-quick    { grid-column: span 4; }

/* ===== CARD BASE ===== */
.card {
  background: var(--sre-bg-card);
  border-radius: var(--sre-radius-xl);
  border: 1px solid var(--sre-border);
  padding: 20px;
  position: relative;
  overflow: hidden;
  transition: transform 250ms var(--sre-ease-out), box-shadow 250ms var(--sre-ease-out), border-color 250ms var(--sre-ease-out);
}

.card:hover {
  transform: translateY(-2px);
  box-shadow: var(--sre-shadow-lift);
  border-color: var(--sre-border-strong);
}

/* ===== CARD HEADER ===== */
.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  position: relative;
  z-index: 2;
}

.card-title {
  font-family: var(--sre-font-display);
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.card-icon {
  width: 22px; height: 22px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-icon svg {
  width: 12px; height: 12px;
  color: #fff;
}

.card-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.card-tag {
  font-size: 10px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 10px;
  background: var(--sre-bg-sunken);
  color: var(--sre-text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.card-tag:hover {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
}

/* ===== TREND CHART ===== */
.trend-chart {
  position: relative;
  height: 170px;
  cursor: crosshair;
}

.trend-svg {
  width: 100%;
  height: 100%;
}

.chart-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 170px;
  font-size: 13px;
  color: var(--sre-text-tertiary);
}

.trend-tooltip {
  position: absolute;
  top: 16px;
  transform: translateX(12px);
  background: var(--sre-bg-elevated);
  color: var(--sre-text-primary);
  font-size: 11px;
  padding: 6px 10px;
  border-radius: 8px;
  border: 1px solid var(--sre-border);
  box-shadow: var(--sre-shadow-md);
  pointer-events: none;
  z-index: 10;
  display: flex;
  flex-direction: column;
  gap: 2px;
  white-space: nowrap;
}

.tt-time {
  color: var(--sre-text-tertiary);
  font-size: 10px;
}

.tt-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.tt-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.tt-val {
  font-family: var(--sre-font-mono);
  font-weight: 500;
  margin-left: auto;
}

.chart-legend {
  display: flex;
  gap: 16px;
  margin-top: 10px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  color: var(--sre-text-secondary);
}

.legend-line {
  width: 10px;
  height: 3px;
  border-radius: 2px;
}

/* ===== SEVERITY ===== */
.sev-bar {
  display: flex;
  gap: 2px;
  height: 8px;
  border-radius: 4px;
  overflow: hidden;
  margin-bottom: 16px;
}

.sev-seg {
  background: var(--sre-critical);
  min-width: 2px;
  transition: flex-grow 0.5s var(--sre-ease-out);
  cursor: pointer;
}

.sev-seg:hover { opacity: 0.8; }

.sev-seg--w { background: var(--sre-warning); }
.sev-seg--i { background: var(--sre-info); }

.sev-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 8px;
}

.sev-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: var(--sre-radius-sm);
  transition: background 0.2s;
  cursor: default;
}

.sev-item:hover {
  background: var(--sre-bg-sunken);
}

.sev-dot {
  width: 8px;
  height: 8px;
  border-radius: 3px;
  flex-shrink: 0;
}

.sev-label {
  font-size: 12px;
  color: var(--sre-text-secondary);
  flex: 1;
}

.sev-count {
  font-family: var(--sre-font-display);
  font-size: 16px;
  font-weight: 700;
}

/* ===== KPI STACK ===== */
.kpi-stack {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.kpi-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: var(--sre-bg-sunken);
  border-radius: var(--sre-radius-sm);
  transition: all 0.2s;
  cursor: default;
}

.kpi-row:hover {
  background: var(--sre-primary-soft);
  transform: translateX(2px);
}

.kpi-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.kpi-info { flex: 1; }

.kpi-value {
  font-family: var(--sre-font-display);
  font-size: 18px;
  font-weight: 800;
  letter-spacing: -0.03em;
  line-height: 1;
}

.kpi-label {
  font-size: 11px;
  color: var(--sre-text-secondary);
  margin-top: 2px;
}

/* ===== RULES ===== */
.rules-list {
  display: flex;
  flex-direction: column;
}

.rule-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 0;
  border-bottom: 1px solid var(--sre-border);
  cursor: pointer;
  transition: background 0.15s;
}

.rule-item:last-child { border-bottom: none; }

.rule-item:hover {
  background: var(--sre-bg-hover);
  margin: 0 -12px;
  padding: 9px 12px;
  border-radius: var(--sre-radius-sm);
}

.rule-status {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.rule-status.active {
  background: var(--sre-success);
  box-shadow: 0 0 4px var(--sre-success);
}

.rule-info { flex: 1; min-width: 0; }

.rule-name {
  font-size: 12px;
  color: var(--sre-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rule-fire {
  font-size: 11px;
  font-family: var(--sre-font-mono);
  color: var(--sre-text-tertiary);
  flex-shrink: 0;
}

/* ===== QUICK ACTIONS ===== */
.actions-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.action-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px 10px;
  border-radius: var(--sre-radius-sm);
  border: 1px solid var(--sre-border);
  background: var(--sre-bg-card);
  cursor: pointer;
  transition: all 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.action-btn:hover {
  border-color: var(--sre-primary);
  background: var(--sre-primary-soft);
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(249, 115, 22, 0.12);
}

.action-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.25s;
}

.action-btn:hover .action-icon {
  transform: scale(1.12) rotate(-3deg);
}

.action-icon svg {
  width: 18px;
  height: 18px;
  color: #fff;
}

.action-label {
  font-size: 11px;
  font-weight: 500;
  color: var(--sre-text-secondary);
  text-align: center;
}

.action-btn:hover .action-label {
  color: var(--sre-primary);
}

/* ===== FOOTER ===== */
.dash-footer {
  text-align: right;
  font-size: 11px;
}

/* ===== RESPONSIVE ===== */
@media (max-width: 1200px) {
  .card-trend    { grid-column: span 12; }
  .card-severity { grid-column: span 6; }
  .card-kpi      { grid-column: span 6; }
  .card-rules    { grid-column: span 6; }
  .card-quick    { grid-column: span 6; }
}

@media (max-width: 768px) {
  .bento {
    grid-template-columns: 1fr;
  }
  .card { grid-column: span 1 !important; }
  .actions-grid { grid-template-columns: repeat(2, 1fr); }
  .sev-grid { grid-template-columns: 1fr 1fr 1fr; }
}
</style>
