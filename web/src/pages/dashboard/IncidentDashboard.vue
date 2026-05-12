<script setup lang="ts">
/**
 * IncidentDashboard.vue — Incident stats dashboard.
 */
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { dashboardV2StatsApi } from '@/api'
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
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
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
    { label: t('dashboardV2.activeIncidents'), value: s.active_incidents ?? 0, tone: 'critical' as const, icon: BugOutline, route: '/incidents?status=triggered' },
    { label: t('dashboardV2.closedToday'), value: s.closed_today ?? 0, tone: 'success' as const, icon: CheckmarkCircleOutline },
    { label: t('dashboardV2.criticalActive'), value: s.critical_active ?? 0, tone: 'critical' as const, icon: AlertCircleOutline },
    { label: t('dashboardV2.avgMTTR'), value: formatSeconds(s.avg_mttr_seconds), tone: 'info' as const, icon: TimerOutline },
    { label: t('dashboardV2.totalPostMortems'), value: s.total_post_mortems ?? 0, tone: 'info' as const, icon: DocumentTextOutline, sub: `${s.published_post_mortems ?? 0} ${t('dashboardV2.published') || 'published'}` },
  ]
})

onMounted(load)
</script>

<template>
  <div class="incident-dashboard">
    <!-- KPI Row -->
    <LoadingSkeleton v-if="loading && !firstLoaded" :rows="5" variant="kpi" />

    <template v-else>
      <n-spin :show="loading">
        <section v-if="incidentStats" class="kpi-grid sre-stagger">
          <div
            v-for="k in kpis"
            :key="k.label"
            class="kpi-card sre-lift"
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
        </section>

        <!-- Trend + Controls -->
        <div class="section-row">
          <div class="chart-card surface-clay">
            <div class="chart-card__header">
              <span class="chart-card__title">{{ t('dashboardV2.incidentTrend') }}</span>
              <div class="chart-card__actions">
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
            <div class="chart-card__body">
              <div v-if="incidentTrend.length" class="trend-chart">
                <div v-for="point in incidentTrend" :key="point.date" class="trend-day">
                  <div class="trend-bars">
                    <div
                      class="trend-bar trend-bar--triggered"
                      :style="{ height: `${Math.max((point.triggered / trendMax) * 80, 2)}px` }"
                      :title="`${point.date}: ${point.triggered} triggered`"
                    />
                    <div
                      class="trend-bar trend-bar--closed"
                      :style="{ height: `${Math.max((point.closed / trendMax) * 80, 2)}px` }"
                      :title="`${point.date}: ${point.closed} closed`"
                    />
                  </div>
                  <div class="trend-label">{{ point.date.substring(5) }}</div>
                </div>
              </div>
              <div v-else class="chart-empty text-muted">{{ t('dashboard.noData') }}</div>
              <div v-if="incidentTrend.length" class="trend-legend">
                <span class="legend-dot legend-dot--triggered" /> {{ t('dashboardV2.triggered') || 'Triggered' }}
                <span class="legend-dot legend-dot--closed" /> {{ t('dashboardV2.closed') || 'Closed' }}
              </div>
            </div>
          </div>
        </div>

        <!-- Two-column: Channel + Team Stats -->
        <div class="two-col">
          <div class="chart-card surface-clay">
            <div class="chart-card__header">
              <span class="chart-card__title">{{ t('dashboardV2.channelStats') }}</span>
            </div>
            <div class="chart-card__body" style="padding-top:0">
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
                    @click="row.channel_id && router.push(`/channels/${row.channel_id}`)"
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
          </div>

          <div class="chart-card surface-clay">
            <div class="chart-card__header">
              <span class="chart-card__title">{{ t('dashboardV2.teamStats') }}</span>
            </div>
            <div class="chart-card__body" style="padding-top:0">
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
  gap: 20px;
}

/* KPI Row */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 16px;
}

.kpi-card {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 20px 22px;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-lg);
  transition: border-color var(--sre-duration-base) var(--sre-ease-out),
              box-shadow var(--sre-duration-base) var(--sre-ease-out),
              transform var(--sre-duration-base) var(--sre-ease-out);
  position: relative;
  overflow: hidden;
}
.kpi-card::after {
  content: '';
  position: absolute;
  top: 0; left: 12px; right: 12px;
  height: 3px;
  border-radius: 0 0 3px 3px;
  background: var(--sre-text-tertiary);
}
.kpi-card[data-tone="critical"]::after { background: var(--sre-critical); }
.kpi-card[data-tone="success"]::after  { background: var(--sre-success); }
.kpi-card[data-tone="info"]::after     { background: var(--sre-info); }
.kpi-card:hover {
  border-color: var(--sre-border-strong);
  box-shadow: var(--sre-shadow-md);
  transform: translateY(-1px);
}
.kpi-card.clickable { cursor: pointer; }
.kpi-card:active { transform: translateY(0); transition-duration: 80ms; }

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
.kpi-card[data-tone="critical"] .kpi-icon-wrap { color: var(--sre-critical); background: var(--sre-critical-soft); border-color: transparent; }
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

/* Chart Card */
.chart-card {
  padding: 0;
  overflow: hidden;
}

.chart-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px 0;
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
  padding: 12px 20px 16px;
}

.chart-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 120px;
  font-size: 13px;
}

.section-row { display: flex; flex-direction: column; }

/* Trend bars */
.trend-chart {
  display: flex;
  align-items: flex-end;
  gap: 3px;
  height: 100px;
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
  height: 80px;
}

.trend-bar {
  width: 7px;
  border-radius: 3px 3px 0 0;
  height: 2px;
}
.trend-bar--triggered { background: var(--sre-critical); }
.trend-bar--closed    { background: var(--sre-primary); }

.trend-label {
  font-size: 9px;
  color: var(--sre-text-tertiary);
  white-space: nowrap;
}

.trend-legend {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--sre-text-secondary);
  margin-top: 10px;
}

.legend-dot {
  display: inline-block;
  width: 8px; height: 8px;
  border-radius: 50%;
}
.legend-dot--triggered { background: var(--sre-critical); }
.legend-dot--closed    { background: var(--sre-primary); }

/* Two columns */
.two-col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

/* Stats table */
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

@media (max-width: 1100px) {
  .kpi-grid { grid-template-columns: repeat(3, 1fr); }
}

@media (max-width: 768px) {
  .kpi-grid { grid-template-columns: repeat(2, 1fr); }
  .two-col { grid-template-columns: 1fr; }
}
</style>
