<script setup lang="ts">
/**
 * IncidentDashboard.vue — Incident stats dashboard (bento grid layout).
 */
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { dashboardV2StatsApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import {
  BugOutline, CheckmarkCircleOutline, AlertCircleOutline, TimerOutline,
  DocumentTextOutline, RefreshOutline,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

const loading = ref(false)
const firstLoaded = ref(false)
const days = ref(30)

interface IncidentStats {
  active_incidents?: number
  closed_today?: number
  critical_active?: number
  avg_mttr_seconds?: number
  total_post_mortems?: number
  published_post_mortems?: number
}
interface ChannelStatsRow {
  channel_id?: number
  channel_name?: string
  total: number
  triggered: number
  critical: number
  closed: number
}
interface TeamStatsRow {
  team_id?: number
  team_name?: string
  total: number
  critical: number
  closed: number
  avg_mttr_seconds?: number
}
interface TrendPoint {
  date: string
  triggered: number
  closed: number
}

const incidentStats = ref<IncidentStats | null>(null)
const channelStats = ref<ChannelStatsRow[]>([])
const teamStats = ref<TeamStatsRow[]>([])
const incidentTrend = ref<TrendPoint[]>([])

async function load() {
  loading.value = true
  try {
    const [isRes, csRes, tsRes, itRes] = await Promise.all([
      dashboardV2StatsApi.incidentStats(),
      dashboardV2StatsApi.channelStats(days.value),
      dashboardV2StatsApi.teamStats(days.value),
      dashboardV2StatsApi.incidentTrend(days.value),
    ])
    incidentStats.value = isRes.data.data
    channelStats.value = csRes.data.data ?? []
    teamStats.value = tsRes.data.data ?? []
    incidentTrend.value = itRes.data.data ?? []
    firstLoaded.value = true
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

function formatSeconds(s: number | undefined) {
  if (!s || s === 0) return '—'
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

const topChannels = computed(() => [...channelStats.value].slice(0, 6))
const topTeams = computed(() => [...teamStats.value].slice(0, 6))
const trendMax = computed(() => Math.max(...incidentTrend.value.map(p => p.triggered + p.closed), 1))

const kpis = computed(() => {
  const s = incidentStats.value
  if (!s) return []
  return [
    { label: t('dashboardV2.activeIncidents'), value: s.active_incidents ?? 0, tone: 'critical' as const, icon: BugOutline, route: '/oncall/incidents?status=triggered' },
    { label: t('dashboardV2.closedToday'), value: s.closed_today ?? 0, tone: 'success' as const, icon: CheckmarkCircleOutline },
    { label: t('dashboardV2.criticalActive'), value: s.critical_active ?? 0, tone: 'critical' as const, icon: AlertCircleOutline },
    { label: t('dashboardV2.avgMTTR'), value: formatSeconds(s.avg_mttr_seconds), tone: 'info' as const, icon: TimerOutline },
    { label: t('dashboardV2.totalPostMortems'), value: s.total_post_mortems ?? 0, tone: 'info' as const, icon: DocumentTextOutline, sub: `${s.published_post_mortems ?? 0} ${t('dashboardV2.published')}` },
  ]
})

// Active incident severity ratio for the summary card
const sevRatio = computed(() => {
  const s = incidentStats.value
  if (!s) return { critical: 0, normal: 0, total: 0 }
  const crit = s.critical_active ?? 0
  const active = s.active_incidents ?? 0
  return { critical: crit, normal: active - crit, total: active }
})

function cycleDays() {
  days.value = days.value === 7 ? 30 : days.value === 30 ? 90 : 7
  load()
}

onMounted(load)
</script>

<template>
  <div class="incident-dashboard">
    <!-- Loading skeleton -->
    <LoadingSkeleton v-if="loading && !firstLoaded" :rows="5" variant="kpi" />

    <template v-else>
      <n-spin :show="loading">
        <div class="bento">

          <!-- ===== KPI ROW (full width) ===== -->
          <div class="card card-kpis" v-if="incidentStats">
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
                  <n-icon :component="k.icon" :size="20" />
                </div>
                <div class="kpi-body">
                  <div class="kpi-value number-display">{{ k.value }}</div>
                  <div class="kpi-label">{{ k.label }}</div>
                  <div v-if="k.sub" class="kpi-sub text-muted">{{ k.sub }}</div>
                </div>
              </div>
            </div>
          </div>

          <!-- ===== TREND CHART (8 cols) ===== -->
          <div class="card card-trend">
            <div class="card-head">
              <div class="card-title">
                <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-primary), var(--sre-amber))">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
                </span>
                {{ t('dashboardV2.incidentTrend') }}
              </div>
              <div class="card-actions">
                <n-select
                  v-model:value="days"
                  :options="[
                    { label: '7 ' + t('dashboardV2.days'), value: 7 },
                    { label: '30 ' + t('dashboardV2.days'), value: 30 },
                    { label: '90 ' + t('dashboardV2.days'), value: 90 },
                  ]"
                  style="width:110px"
                  size="tiny"
                  @update:value="load"
                />
                <n-button quaternary circle size="tiny" :loading="loading" @click="load">
                  <template #icon><n-icon :component="RefreshOutline" :size="14" /></template>
                </n-button>
              </div>
            </div>
            <div class="trend-chart" v-if="incidentTrend.length">
              <div v-for="point in incidentTrend" :key="point.date" class="trend-day">
                <div class="trend-bars">
                  <div
                    class="trend-bar trend-bar--triggered"
                    :style="{ height: `${Math.max((point.triggered / trendMax) * 80, 2)}px` }"
                    :title="`${point.date}: ${point.triggered} ${t('tooltip.triggered')}`"
                  />
                  <div
                    class="trend-bar trend-bar--closed"
                    :style="{ height: `${Math.max((point.closed / trendMax) * 80, 2)}px` }"
                    :title="`${point.date}: ${point.closed} ${t('tooltip.closed')}`"
                  />
                </div>
                <div class="trend-label">{{ point.date.substring(5) }}</div>
              </div>
            </div>
            <div v-else class="chart-empty">{{ t('dashboard.noData') }}</div>
            <div v-if="incidentTrend.length" class="trend-legend">
              <span class="legend-item"><span class="legend-dot" style="background: var(--sre-critical)"></span>{{ t('dashboardV2.triggered') }}</span>
              <span class="legend-item"><span class="legend-dot" style="background: var(--sre-primary)"></span>{{ t('dashboardV2.closed') }}</span>
            </div>
          </div>

          <!-- ===== ACTIVE INCIDENTS SUMMARY (4 cols) ===== -->
          <div class="card card-active">
            <div class="card-head">
              <div class="card-title">
                <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-critical), var(--sre-rose-light))">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/></svg>
                </span>
                {{ t('dashboardV2.activeIncidents') }}
              </div>
            </div>
            <div class="active-stack" v-if="incidentStats">
              <div class="active-hero">
                <div class="active-hero__value number-display" style="color: var(--sre-critical)">
                  {{ incidentStats.active_incidents ?? 0 }}
                </div>
                <div class="active-hero__label">{{ t('dashboardV2.activeIncidents') }}</div>
              </div>
              <!-- Severity bar -->
              <div class="sev-bar" v-if="sevRatio.total > 0">
                <div
                  class="sev-seg sev-seg--crit"
                  :style="{ flex: sevRatio.critical }"
                  :title="`${t('tooltip.criticalLabel')}: ${sevRatio.critical}`"
                ></div>
                <div
                  class="sev-seg sev-seg--normal"
                  :style="{ flex: sevRatio.normal }"
                  :title="`${t('tooltip.normalLabel')}: ${sevRatio.normal}`"
                ></div>
              </div>
              <div class="active-meta">
                <div class="active-meta__item">
                  <span class="active-meta__dot" style="background: var(--sre-critical)"></span>
                  <span class="active-meta__label">{{ t('dashboardV2.criticalActive') }}</span>
                  <span class="active-meta__val" style="color: var(--sre-critical)">{{ incidentStats.critical_active ?? 0 }}</span>
                </div>
                <div class="active-meta__item">
                  <span class="active-meta__dot" style="background: var(--sre-success)"></span>
                  <span class="active-meta__label">{{ t('dashboardV2.closedToday') }}</span>
                  <span class="active-meta__val" style="color: var(--sre-success)">{{ incidentStats.closed_today ?? 0 }}</span>
                </div>
                <div class="active-meta__item">
                  <span class="active-meta__dot" style="background: var(--sre-info)"></span>
                  <span class="active-meta__label">{{ t('dashboardV2.avgMTTR') }}</span>
                  <span class="active-meta__val" style="color: var(--sre-info)">{{ formatSeconds(incidentStats.avg_mttr_seconds) }}</span>
                </div>
              </div>
            </div>
            <div v-else class="chart-empty">{{ t('dashboard.noData') }}</div>
          </div>

          <!-- ===== CHANNEL STATS (6 cols) ===== -->
          <div class="card card-channels">
            <div class="card-head">
              <div class="card-title">
                <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-lavender), var(--sre-violet-light))">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/></svg>
                </span>
                {{ t('dashboardV2.channelStats') }}
              </div>
            </div>
            <table class="stats-table">
              <thead>
                <tr>
                  <th>{{ t('channel.name') }}</th>
                  <th class="num-col">{{ t('dashboard.total') }}</th>
                  <th class="num-col">{{ t('dashboard.pending') }}</th>
                  <th class="num-col">{{ t('dashboard.urgent') }}</th>
                  <th class="num-col">{{ t('dashboard.closed') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in topChannels"
                  :key="row.channel_id"
                  class="stats-row"
                  @click="row.channel_id && router.push(`/oncall/spaces/${row.channel_id}`)"
                >
                  <td class="stats-name">{{ row.channel_name || '—' }}</td>
                  <td class="num-col"><strong>{{ row.total }}</strong></td>
                  <td class="num-col"><span class="text-critical">{{ row.triggered }}</span></td>
                  <td class="num-col"><span class="text-critical">{{ row.critical }}</span></td>
                  <td class="num-col"><span class="text-success">{{ row.closed }}</span></td>
                </tr>
                <tr v-if="topChannels.length === 0">
                  <td colspan="5" class="stats-empty">{{ t('dashboard.noData') }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- ===== TEAM STATS (6 cols) ===== -->
          <div class="card card-teams">
            <div class="card-head">
              <div class="card-title">
                <span class="card-icon" style="background: linear-gradient(135deg, var(--sre-info), var(--sre-sky))">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
                </span>
                {{ t('dashboardV2.teamStats') }}
              </div>
            </div>
            <table class="stats-table">
              <thead>
                <tr>
                  <th>{{ t('common.name') }}</th>
                  <th class="num-col">{{ t('dashboard.total') }}</th>
                  <th class="num-col">{{ t('dashboard.urgent') }}</th>
                  <th class="num-col">{{ t('dashboard.closed') }}</th>
                  <th class="num-col">{{ t('dashboard.avgMttr') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in topTeams" :key="row.team_id" class="stats-row">
                  <td class="stats-name">{{ row.team_name || t('dashboard.ungrouped') }}</td>
                  <td class="num-col"><strong>{{ row.total }}</strong></td>
                  <td class="num-col"><span class="text-critical">{{ row.critical }}</span></td>
                  <td class="num-col"><span class="text-success">{{ row.closed }}</span></td>
                  <td class="num-col">{{ formatSeconds(row.avg_mttr_seconds) }}</td>
                </tr>
                <tr v-if="topTeams.length === 0">
                  <td colspan="5" class="stats-empty">{{ t('dashboard.noData') }}</td>
                </tr>
              </tbody>
            </table>
          </div>

        </div>
      </n-spin>
    </template>
  </div>
</template>

<style scoped>
.incident-dashboard {
  max-width: 1440px;
  display: flex;
  flex-direction: column;
  gap: 0;
  font-family: var(--sre-font-sans);
}

/* ===== BENTO GRID ===== */
.bento {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 16px;
}

.card { grid-column: span 12; }

/* Card sizes */
.card-kpis    { grid-column: 1 / -1; }
.card-trend   { grid-column: span 8; }
.card-active  { grid-column: span 4; }
.card-channels { grid-column: span 6; }
.card-teams    { grid-column: span 6; }

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
  box-shadow: var(--sre-shadow-lg);
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

/* ===== KPI ROW ===== */
.card-kpis {
  padding: 16px 20px;
}

.kpis-flex {
  display: flex;
  gap: 12px;
}

.kpi-item {
  flex: 1;
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 16px 18px;
  background: var(--sre-bg-sunken);
  border-radius: var(--sre-radius-md);
  border: 1px solid transparent;
  position: relative;
  overflow: hidden;
  transition: border-color 250ms var(--sre-ease-out), box-shadow 250ms var(--sre-ease-out), transform 250ms var(--sre-ease-out), background 250ms var(--sre-ease-out);
}

.kpi-item::after {
  content: '';
  position: absolute;
  top: 0; left: 8px; right: 8px;
  height: 3px;
  border-radius: 0 0 3px 3px;
  opacity: 0;
  transition: opacity 250ms var(--sre-ease-out);
}

.kpi-item[data-tone="critical"]::after { background: var(--sre-critical); }
.kpi-item[data-tone="success"]::after  { background: var(--sre-success); }
.kpi-item[data-tone="info"]::after     { background: var(--sre-info); }

.kpi-item:hover {
  border-color: var(--sre-border-strong);
  box-shadow: var(--sre-shadow-md);
  transform: translateY(-2px);
  background: var(--sre-bg-card);
}

.kpi-item:hover::after { opacity: 1; }

.kpi-item.clickable { cursor: pointer; }
.kpi-item:active { transform: translateY(0); transition-duration: 80ms; }

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
}

.kpi-item[data-tone="critical"] .kpi-icon-wrap { color: var(--sre-critical); background: var(--sre-critical-soft); border-color: transparent; }
.kpi-item[data-tone="success"]  .kpi-icon-wrap { color: var(--sre-primary); background: var(--sre-primary-soft); border-color: transparent; }
.kpi-item[data-tone="info"]     .kpi-icon-wrap { color: var(--sre-info); background: var(--sre-info-soft); border-color: transparent; }

.kpi-body { flex: 1; min-width: 0; }

.kpi-value {
  font-family: var(--sre-font-display);
  font-size: 28px;
  font-weight: 800;
  line-height: 1.1;
  color: var(--sre-text-primary);
  letter-spacing: -0.03em;
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

/* ===== TREND CHART ===== */
.trend-chart {
  display: flex;
  align-items: flex-end;
  gap: 3px;
  height: 120px;
}

.trend-day {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  min-width: 0;
}

.trend-bars {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 100px;
}

.trend-bar {
  width: 7px;
  border-radius: 3px 3px 0 0;
  height: 2px;
  transition: opacity 0.15s;
}

.trend-bar:hover { opacity: 0.8; }

.trend-bar--triggered { background: var(--sre-critical); }
.trend-bar--closed    { background: var(--sre-primary); }

.trend-label {
  font-size: 9px;
  color: var(--sre-text-tertiary);
  white-space: nowrap;
}

.trend-legend {
  display: flex;
  gap: 16px;
  margin-top: 12px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  color: var(--sre-text-secondary);
}

.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.chart-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 120px;
  font-size: 13px;
  color: var(--sre-text-tertiary);
}

/* ===== ACTIVE INCIDENTS SUMMARY ===== */
.active-stack {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.active-hero {
  text-align: center;
  padding: 8px 0 4px;
}

.active-hero__value {
  font-family: var(--sre-font-display);
  font-size: 48px;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.04em;
}

.active-hero__label {
  font-size: 12px;
  font-weight: 500;
  color: var(--sre-text-secondary);
  margin-top: 4px;
}

/* Severity bar */
.sev-bar {
  display: flex;
  gap: 2px;
  height: 6px;
  border-radius: 3px;
  overflow: hidden;
}

.sev-seg {
  min-width: 2px;
  transition: flex-grow 0.5s var(--sre-ease-out);
}

.sev-seg--crit   { background: var(--sre-critical); }
.sev-seg--normal { background: var(--sre-primary); }

/* Active meta items */
.active-meta {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.active-meta__item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: var(--sre-radius-sm);
  transition: background 0.2s;
}

.active-meta__item:hover {
  background: var(--sre-bg-sunken);
}

.active-meta__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.active-meta__label {
  flex: 1;
  font-size: 12px;
  color: var(--sre-text-secondary);
}

.active-meta__val {
  font-family: var(--sre-font-display);
  font-size: 15px;
  font-weight: 700;
}

/* ===== STATS TABLES ===== */
.stats-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.stats-table th {
  text-align: left;
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-tertiary);
  padding: 6px 10px 10px;
  letter-spacing: 0.03em;
  text-transform: uppercase;
  border-bottom: var(--sre-hairline);
}

.stats-table td {
  padding: 9px 10px;
  border-bottom: var(--sre-hairline);
  color: var(--sre-text-primary);
}

.num-col { text-align: right; }

.stats-row {
  cursor: pointer;
  transition: background 0.15s;
}

.stats-row:hover { background: var(--sre-bg-hover); }

.stats-name {
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 160px;
}

.stats-empty {
  text-align: center;
  color: var(--sre-text-secondary);
  padding: 32px 16px;
}

.text-critical { color: var(--sre-critical); font-weight: 500; }
.text-success  { color: var(--sre-success); font-weight: 500; }

/* ===== RESPONSIVE ===== */
@media (max-width: 1200px) {
  .card-kpis    { grid-column: 1 / -1; }
  .card-trend   { grid-column: span 12; }
  .card-active  { grid-column: span 12; }
  .card-channels { grid-column: span 6; }
  .card-teams    { grid-column: span 6; }
  .kpis-flex { flex-wrap: wrap; }
  .kpi-item { flex: 1 1 calc(33.333% - 8px); min-width: 200px; }
}

@media (max-width: 768px) {
  .bento {
    grid-template-columns: 1fr;
  }
  .card { grid-column: span 1 !important; }
  .kpis-flex { flex-direction: column; }
  .kpi-item { flex: 1 1 100%; min-width: 0; }
  .active-hero__value { font-size: 36px; }
}
</style>
