<script setup lang="ts">
/**
 * UnifiedDashboard.vue — Combined overview dashboard for all modules.
 * Shows incident stats, alert trends, severity distribution, and quick navigation.
 */
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { dashboardApi, dashboardV2StatsApi } from '@/api'
import type { DashboardStats, MTTRStats, AlertTrendPoint, TopRuleItem } from '@/types'
import {
  BugOutline, PulseOutline, TimerOutline, CheckmarkCircleOutline,
  AlertCircleOutline, DocumentTextOutline, RefreshOutline,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

const loading = ref(false)
const days = ref(7)

// ===== Data =====
const dashStats = ref<DashboardStats>({
  total_datasources: 0, total_rules: 0, active_alerts: 0,
  resolved_today: 0, total_users: 0, total_teams: 0,
  severity_breakdown: { critical: 0, warning: 0, info: 0 },
})

const emptyMetric = { mean: -1, p50: -1, p95: -1, count: 0 }
const mttrStats = ref<MTTRStats>({
  window_hours: 24, mtta: { ...emptyMetric }, mttr: { ...emptyMetric },
  by_severity: [], mtta_seconds: -1, mttr_seconds: -1,
  acked_count: 0, resolved_count: 0,
})

interface IncidentStats {
  active_incidents?: number
  closed_today?: number
  critical_active?: number
  avg_mttr_seconds?: number
}
const incidentStats = ref<IncidentStats>({})

const alertTrend = ref<AlertTrendPoint[]>([])
const topRules = ref<TopRuleItem[]>([])

interface TrendPoint { date: string; triggered: number; closed: number }
const incidentTrend = ref<TrendPoint[]>([])
const incidentTrendMax = computed(() => Math.max(...incidentTrend.value.map(d => Math.max(d.triggered, d.closed)), 1))

// ===== KPIs =====
const kpis = computed(() => [
  { label: t('dashboardV2.activeIncidents'), value: incidentStats.value.active_incidents ?? 0, tone: 'critical', icon: BugOutline, route: '/oncall/incidents' },
  { label: t('dashboard.activeAlerts'), value: dashStats.value.active_alerts, tone: 'warning', icon: PulseOutline, route: '/alert/events' },
  { label: t('dashboardV2.closedToday'), value: incidentStats.value.closed_today ?? 0, tone: 'success', icon: CheckmarkCircleOutline },
  { label: t('dashboardV2.criticalActive'), value: incidentStats.value.critical_active ?? 0, tone: 'critical', icon: AlertCircleOutline },
  { label: t('dashboard.mtta'), value: formatMMSS(mttrStats.value.mtta?.p50 ?? -1), tone: 'success', icon: TimerOutline },
  { label: t('dashboard.mttr'), value: formatMMSS(mttrStats.value.mttr?.p50 ?? -1), tone: 'info', icon: CheckmarkCircleOutline },
])

// ===== Helpers =====
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

// Severity summary
const sevTotal = computed(() => {
  const s = dashStats.value.severity_breakdown
  return (s?.critical ?? 0) + (s?.warning ?? 0) + (s?.info ?? 0)
})
const sevPct = (key: 'critical' | 'warning' | 'info') => {
  if (sevTotal.value === 0) return 0
  return Math.round(((dashStats.value.severity_breakdown?.[key] ?? 0) / sevTotal.value) * 100)
}

// Alert trend chart (SVG area)
const alertTrendMax = computed(() => Math.max(...alertTrend.value.map(d => Math.max(d.fired_count, d.resolved_count)), 1))
const chartW = 600
const chartH = 170
const hoverIdx = ref(-1)

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

function onChartMove(e: MouseEvent) {
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  const x = e.clientX - rect.left
  hoverIdx.value = Math.round((x / rect.width) * (alertTrend.value.length - 1))
}
function onChartLeave() { hoverIdx.value = -1 }

const hoveredPoint = computed(() => {
  const i = hoverIdx.value
  if (i < 0 || i >= alertTrend.value.length) return null
  return alertTrend.value[i]
})

// ===== Data Loading =====
async function load() {
  loading.value = true
  try {
    const [dsRes, mttrRes, alertRes, topRes, incRes, incTrendRes] = await Promise.allSettled([
      dashboardApi.getStats(),
      dashboardApi.getMTTRStats(days.value === 7 ? 168 : days.value === 30 ? 720 : 2160),
      dashboardApi.getAlertTrend(days.value),
      dashboardApi.getTopRules(days.value, 5),
      dashboardV2StatsApi.incidentStats(),
      dashboardV2StatsApi.incidentTrend(days.value),
    ])
    if (dsRes.status === 'fulfilled') dashStats.value = dsRes.value.data.data
    if (mttrRes.status === 'fulfilled') mttrStats.value = mttrRes.value.data.data
    if (alertRes.status === 'fulfilled') alertTrend.value = alertRes.value.data.data || []
    if (topRes.status === 'fulfilled') topRules.value = topRes.value.data.data || []
    if (incRes.status === 'fulfilled') incidentStats.value = incRes.value.data.data || {}
    if (incTrendRes.status === 'fulfilled') incidentTrend.value = incTrendRes.value.data.data || []
  } catch (e: any) {
    message.error(e?.message || t('dashboard.loadFailed'))
  } finally {
    loading.value = false
  }
}

function cycleDays() {
  days.value = days.value === 7 ? 30 : days.value === 30 ? 90 : 7
  load()
}

onMounted(load)
</script>

<template>
  <div class="unified-dashboard">
    <n-spin :show="loading">
      <div class="bento">

        <!-- ===== KPI ROW (full width) ===== -->
        <div class="card card-kpis">
          <div class="kpis-flex">
            <div
              v-for="k in kpis"
              :key="k.label"
              class="kpi-item"
              :data-tone="k.tone"
              :class="{ clickable: !!k.route }"
              @click="k.route && router.push(k.route)"
            >
              <div class="kpi-icon-wrap">
                <n-icon :component="k.icon" :size="18" />
              </div>
              <div class="kpi-body">
                <div class="kpi-value number-display">{{ k.value }}</div>
                <div class="kpi-label">{{ k.label }}</div>
              </div>
            </div>
          </div>
        </div>

        <!-- ===== ALERT TREND (8 cols) ===== -->
        <div class="card card-alert-trend">
          <div class="card-head">
            <div class="card-title">
              <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-primary), var(--sre-coral))">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
              </span>
              {{ t('dashboard.alertTrend') }}
            </div>
            <div class="card-actions">
              <span class="card-tag" @click="cycleDays">{{ days }}d</span>
              <n-button quaternary circle size="tiny" :loading="loading" @click="load">
                <template #icon><n-icon :component="RefreshOutline" :size="12" /></template>
              </n-button>
            </div>
          </div>
          <div class="trend-chart" @mousemove="onChartMove" @mouseleave="onChartLeave">
            <svg v-if="alertTrend.length" :viewBox="`0 0 ${chartW} ${chartH}`" preserveAspectRatio="none" class="trend-svg">
              <defs>
                <linearGradient id="uGFired" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="var(--sre-primary)" stop-opacity="0.25" />
                  <stop offset="100%" stop-color="var(--sre-primary)" stop-opacity="0.02" />
                </linearGradient>
                <linearGradient id="uGResolved" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="var(--sre-success)" stop-opacity="0.18" />
                  <stop offset="100%" stop-color="var(--sre-success)" stop-opacity="0.02" />
                </linearGradient>
              </defs>
              <line v-for="i in 4" :key="i" :x1="0" :y1="(chartH / 4) * i" :x2="chartW" :y2="(chartH / 4) * i" stroke="var(--sre-border)" stroke-width="0.5" />
              <path :d="trendFillPath(alertTrend.map(d => d.resolved_count), alertTrendMax, chartW, chartH)" fill="url(#uGResolved)" />
              <path :d="trendAreaPath(alertTrend.map(d => d.resolved_count), alertTrendMax, chartW, chartH)" fill="none" stroke="var(--sre-success)" stroke-width="2" stroke-linecap="round" />
              <path :d="trendFillPath(alertTrend.map(d => d.fired_count), alertTrendMax, chartW, chartH)" fill="url(#uGFired)" />
              <path :d="trendAreaPath(alertTrend.map(d => d.fired_count), alertTrendMax, chartW, chartH)" fill="none" stroke="var(--sre-primary)" stroke-width="2.5" stroke-linecap="round" />
              <line v-if="hoverIdx >= 0 && hoverIdx < alertTrend.length"
                :x1="(hoverIdx / (alertTrend.length - 1)) * chartW" :y1="0"
                :x2="(hoverIdx / (alertTrend.length - 1)) * chartW" :y2="chartH"
                stroke="var(--sre-primary)" stroke-width="1" stroke-dasharray="4 3" opacity="0.5" />
            </svg>
            <div v-else class="chart-empty">{{ t('dashboard.noData') }}</div>
            <div v-if="hoveredPoint" class="trend-tooltip" :style="{ left: `${Math.min((hoverIdx / Math.max(alertTrend.length - 1, 1)) * 100, 85)}%` }">
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

        <!-- ===== SEVERITY DISTRIBUTION (4 cols) ===== -->
        <div class="card card-severity">
          <div class="card-head">
            <div class="card-title">
              <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-critical), var(--sre-coral))">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/></svg>
              </span>
              {{ t('dashboard.severityDistribution') }}
            </div>
          </div>
          <div class="sev-bar">
            <div class="sev-seg" :style="{ flex: sevPct('critical') }"></div>
            <div class="sev-seg sev-seg--w" :style="{ flex: sevPct('warning') }"></div>
            <div class="sev-seg sev-seg--i" :style="{ flex: sevPct('info') }"></div>
          </div>
          <div class="sev-grid">
            <div class="sev-item"><span class="sev-dot" style="background: var(--sre-critical)"></span><span class="sev-label">P0</span><span class="sev-count" style="color: var(--sre-critical)">{{ dashStats.severity_breakdown?.critical ?? 0 }}</span></div>
            <div class="sev-item"><span class="sev-dot" style="background: var(--sre-warning)"></span><span class="sev-label">P1</span><span class="sev-count" style="color: var(--sre-warning)">{{ dashStats.severity_breakdown?.warning ?? 0 }}</span></div>
            <div class="sev-item"><span class="sev-dot" style="background: var(--sre-info)"></span><span class="sev-label">P2</span><span class="sev-count" style="color: var(--sre-info)">{{ dashStats.severity_breakdown?.info ?? 0 }}</span></div>
          </div>
        </div>

        <!-- ===== INCIDENT TREND (8 cols) ===== -->
        <div class="card card-incident-trend">
          <div class="card-head">
            <div class="card-title">
              <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-primary), var(--sre-amber))">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
              </span>
              {{ t('dashboardV2.incidentTrend') }}
            </div>
          </div>
          <div class="inc-trend-chart" v-if="incidentTrend.length">
            <div v-for="point in incidentTrend" :key="point.date" class="inc-trend-day">
              <div class="inc-trend-bars">
                <div
                  class="inc-trend-bar inc-trend-bar--triggered"
                  :style="{ height: `${Math.max((point.triggered / incidentTrendMax) * 80, 2)}px` }"
                  :title="`${point.date}: ${point.triggered} ${t('tooltip.triggered')}`"
                />
                <div
                  class="inc-trend-bar inc-trend-bar--closed"
                  :style="{ height: `${Math.max((point.closed / incidentTrendMax) * 80, 2)}px` }"
                  :title="`${point.date}: ${point.closed} ${t('tooltip.closed')}`"
                />
              </div>
              <div class="inc-trend-label">{{ point.date.substring(5) }}</div>
            </div>
          </div>
          <div v-else class="chart-empty">{{ t('dashboard.noData') }}</div>
        </div>

        <!-- ===== TOP RULES (4 cols) ===== -->
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
            <div v-for="r in topRules" :key="r.alert_name" class="rule-item" role="button" @click="router.push('/alert/rules')">
              <span class="rule-status active"></span>
              <div class="rule-info"><div class="rule-name">{{ r.alert_name }}</div></div>
              <span class="rule-fire">{{ r.count }}</span>
            </div>
          </div>
          <div v-else class="chart-empty">{{ t('dashboard.noData') }}</div>
        </div>

        <!-- ===== QUICK ACTIONS (full width) ===== -->
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
            <div class="action-btn" role="button" @click="router.push('/alert/rules')">
              <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-critical), var(--sre-rose-light))">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/></svg>
              </div>
              <span class="action-label">{{ t('menu.rules') }}</span>
            </div>
            <div class="action-btn" role="button" @click="router.push('/oncall/schedule')">
              <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-info), var(--sre-sky))">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
              </div>
              <span class="action-label">{{ t('menu.schedule') }}</span>
            </div>
            <div class="action-btn" role="button" @click="router.push('/oncall/incidents')">
              <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-amber), var(--sre-coral))">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/></svg>
              </div>
              <span class="action-label">{{ t('menu.incidents') }}</span>
            </div>
            <div class="action-btn" role="button" @click="router.push('/alert/explore')">
              <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-success), var(--sre-emerald-light))">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
              </div>
              <span class="action-label">{{ t('menu.explore') }}</span>
            </div>
            <div class="action-btn" role="button" @click="router.push('/alert/dashboards')">
              <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-lavender), var(--sre-violet-light))">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/></svg>
              </div>
              <span class="action-label">{{ t('menu.dashboards') }}</span>
            </div>
            <div class="action-btn" role="button" @click="router.push('/alert/suppression')">
              <div class="action-icon" style="background: linear-gradient(135deg, var(--sre-mint), var(--sre-success))">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
              </div>
              <span class="action-label">{{ t('menu.suppression') }}</span>
            </div>
          </div>
        </div>

      </div>
    </n-spin>
  </div>
</template>

<style scoped>
.unified-dashboard {
  max-width: 1440px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  font-family: var(--sre-font-sans);
}

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
.bento > .card:nth-child(6) { animation-delay: 300ms; }

@keyframes card-enter {
  from { opacity: 0; transform: translateY(12px); }
  to   { opacity: 1; transform: translateY(0); }
}

.card { grid-column: span 12; }

/* Card sizes */
.card-kpis            { grid-column: span 12; }
.card-alert-trend     { grid-column: span 8; }
.card-severity        { grid-column: span 4; }
.card-incident-trend  { grid-column: span 8; }
.card-rules           { grid-column: span 4; }
.card-quick           { grid-column: span 12; }

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
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.card-icon svg {
  width: 14px;
  height: 14px;
  color: #fff;
}

.card-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-tag {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-primary);
  background: var(--sre-primary-soft);
  padding: 2px 8px;
  border-radius: 6px;
  cursor: pointer;
  user-select: none;
}

/* ===== KPI ROW ===== */
.kpis-flex {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.kpi-item {
  flex: 1;
  min-width: 140px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: var(--sre-radius-md);
  background: var(--sre-bg-elevated);
  border: 1px solid var(--sre-border);
  transition: border-color 200ms var(--sre-ease-out);
}

.kpi-item.clickable {
  cursor: pointer;
}

.kpi-item.clickable:hover {
  border-color: var(--sre-primary);
}

.kpi-icon-wrap {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.kpi-item[data-tone="critical"] .kpi-icon-wrap { background: linear-gradient(135deg, var(--sre-critical), var(--sre-rose-light)); color: #fff; }
.kpi-item[data-tone="warning"] .kpi-icon-wrap  { background: linear-gradient(135deg, var(--sre-warning), var(--sre-amber)); color: #fff; }
.kpi-item[data-tone="success"] .kpi-icon-wrap  { background: linear-gradient(135deg, var(--sre-success), var(--sre-emerald-light)); color: #fff; }
.kpi-item[data-tone="info"] .kpi-icon-wrap     { background: linear-gradient(135deg, var(--sre-info), var(--sre-sky)); color: #fff; }

.kpi-body { min-width: 0; }

.kpi-value {
  font-size: 20px;
  font-weight: 700;
  line-height: 1.2;
  color: var(--sre-text-primary);
}

.kpi-label {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  margin-top: 2px;
}

/* ===== TREND CHART (shared) ===== */
.trend-chart {
  position: relative;
  height: 170px;
}

.trend-svg {
  width: 100%;
  height: 100%;
  display: block;
}

.chart-empty {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--sre-text-muted);
  font-size: 13px;
}

.trend-tooltip {
  position: absolute;
  bottom: 8px;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-sm);
  padding: 8px 10px;
  font-size: 11px;
  pointer-events: none;
  z-index: 5;
  box-shadow: var(--sre-shadow-sm);
}

.tt-time { font-weight: 600; color: var(--sre-text-primary); margin-bottom: 4px; }
.tt-row { display: flex; align-items: center; gap: 4px; color: var(--sre-text-secondary); }
.tt-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.tt-val { font-weight: 600; color: var(--sre-text-primary); margin-left: auto; }

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
  color: var(--sre-text-tertiary);
}

.legend-line {
  width: 14px;
  height: 3px;
  border-radius: 2px;
}

/* ===== SEVERITY ===== */
.sev-bar {
  display: flex;
  height: 8px;
  border-radius: 4px;
  overflow: hidden;
  gap: 2px;
  margin-bottom: 16px;
}

.sev-seg {
  background: var(--sre-critical);
  border-radius: 4px;
  min-width: 4px;
  transition: flex 300ms var(--sre-ease-out);
}

.sev-seg--w { background: var(--sre-warning); }
.sev-seg--i { background: var(--sre-info); }

.sev-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.sev-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}

.sev-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.sev-label {
  font-weight: 600;
  color: var(--sre-text-secondary);
}

.sev-count {
  font-weight: 700;
  margin-left: auto;
}

/* ===== INCIDENT TREND (CSS bars) ===== */
.inc-trend-chart {
  display: flex;
  gap: 3px;
  align-items: flex-end;
  height: 120px;
  overflow-x: auto;
}

.inc-trend-day {
  flex: 1;
  min-width: 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.inc-trend-bars {
  display: flex;
  gap: 2px;
  align-items: flex-end;
  height: 80px;
}

.inc-trend-bar {
  width: 8px;
  border-radius: 3px 3px 0 0;
  min-height: 2px;
  transition: height 300ms var(--sre-ease-out);
}

.inc-trend-bar--triggered { background: var(--sre-critical); }
.inc-trend-bar--closed    { background: var(--sre-success); }

.inc-trend-label {
  font-size: 9px;
  color: var(--sre-text-muted);
  white-space: nowrap;
}

/* ===== TOP RULES ===== */
.rules-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.rule-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--sre-radius-sm);
  cursor: pointer;
  transition: background 150ms var(--sre-ease-out);
}

.rule-item:hover {
  background: var(--sre-bg-hover);
}

.rule-status {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.rule-status.active {
  background: var(--sre-success);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.18);
}

.rule-info { flex: 1; min-width: 0; }

.rule-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--sre-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rule-fire {
  font-size: 12px;
  font-weight: 700;
  color: var(--sre-critical);
  flex-shrink: 0;
}

/* ===== QUICK ACTIONS ===== */
.actions-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 10px;
}

.action-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 14px 8px;
  border-radius: var(--sre-radius-md);
  cursor: pointer;
  transition: background 150ms var(--sre-ease-out), transform 150ms var(--sre-ease-out);
}

.action-btn:hover {
  background: var(--sre-bg-hover);
  transform: translateY(-1px);
}

.action-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
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

/* ===== RESPONSIVE ===== */
@media (max-width: 1200px) {
  .card-alert-trend,
  .card-incident-trend { grid-column: span 12; }
  .card-severity,
  .card-rules { grid-column: span 6; }
  .actions-grid { grid-template-columns: repeat(3, 1fr); }
}

@media (max-width: 768px) {
  .card-severity,
  .card-rules { grid-column: span 12; }
  .kpis-flex { flex-direction: column; }
  .kpi-item { min-width: 0; }
  .actions-grid { grid-template-columns: repeat(2, 1fr); }
}

/* ===== REDUCED MOTION ===== */
@media (prefers-reduced-motion: reduce) {
  .bento > .card { animation: none; }
  .card:hover { transform: none; }
}
</style>
