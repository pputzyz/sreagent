<script setup lang="ts">
/**
 * IncidentDashboard.vue — v2 enhanced dashboard showing incident/channel/team stats.
 * Phase 5.6
 */
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { dashboardV2StatsApi } from '@/api'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

const loading = ref(false)
const firstLoaded = ref(false)
const days = ref(30)

const incidentStats = ref<any>(null)
const channelStats = ref<any[]>([])
const teamStats = ref<any[]>([])
const incidentTrend = ref<any[]>([])

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

function formatSeconds(s: number) {
  if (!s || s === 0) return '—'
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

// Top 5 channels by total
const topChannels = computed(() => [...channelStats.value].slice(0, 5))
const topTeams = computed(() => [...teamStats.value].slice(0, 5))

// Trend sparkline: max triggered value for scaling
const trendMax = computed(() => Math.max(...incidentTrend.value.map(p => p.triggered + p.closed), 1))

onMounted(load)
</script>

<template>
  <div class="incident-dashboard">
    <!-- Header -->
    <div class="dash-header">
      <h2 class="dash-title">{{ t('dashboardV2.incidentStats') }}</h2>
      <div class="dash-controls">
        <n-select
          v-model:value="days"
          :options="[
            { label: '7 ' + t('dashboardV2.days'), value: 7 },
            { label: '30 ' + t('dashboardV2.days'), value: 30 },
            { label: '90 ' + t('dashboardV2.days'), value: 90 },
          ]"
          style="width:110px"
          size="small"
          @update:value="load"
        />
        <n-button size="small" quaternary circle :loading="loading" @click="load">
          <template #icon><n-icon><svg viewBox="0 0 24 24"><path fill="currentColor" d="M17.65 6.35A7.958 7.958 0 0 0 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0 1 12 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/></svg></n-icon></template>
        </n-button>
      </div>
    </div>

    <!-- Loading skeleton -->
    <LoadingSkeleton v-if="loading && !firstLoaded" :rows="5" variant="kpi" />

    <n-spin v-else :show="loading">
      <!-- Top stat cards -->
      <div v-if="incidentStats" class="stat-cards sre-stagger">
        <div class="stat-card sre-lift" @click="router.push('/incidents?status=triggered')">
          <div class="stat-label">{{ t('dashboardV2.activeIncidents') }}</div>
          <div class="stat-value stat-value--critical">{{ incidentStats.active_incidents ?? 0 }}</div>
        </div>
        <div class="stat-card sre-lift">
          <div class="stat-label">{{ t('dashboardV2.closedToday') }}</div>
          <div class="stat-value stat-value--success">{{ incidentStats.closed_today ?? 0 }}</div>
        </div>
        <div class="stat-card sre-lift">
          <div class="stat-label">{{ t('dashboardV2.criticalActive') }}</div>
          <div class="stat-value stat-value--critical">{{ incidentStats.critical_active ?? 0 }}</div>
        </div>
        <div class="stat-card sre-lift">
          <div class="stat-label">{{ t('dashboardV2.avgMTTR') }}</div>
          <div class="stat-value">{{ formatSeconds(incidentStats.avg_mttr_seconds) }}</div>
        </div>
        <div class="stat-card sre-lift">
          <div class="stat-label">{{ t('dashboardV2.totalPostMortems') }}</div>
          <div class="stat-value">{{ incidentStats.total_post_mortems ?? 0 }}</div>
          <div class="stat-sub">{{ incidentStats.published_post_mortems ?? 0 }} {{ t('dashboardV2.publishedPostMortems') }}</div>
        </div>
      </div>

      <!-- Incident trend bars -->
      <n-card :bordered="false" class="section-card" v-if="incidentTrend.length">
        <div class="section-title">{{ t('dashboardV2.incidentTrend') }}</div>
        <div class="trend-chart">
          <div v-for="point in incidentTrend" :key="point.date" class="trend-day">
            <div class="trend-bars">
              <div
                class="trend-bar trend-bar--triggered"
                :style="{ height: `${(point.triggered / trendMax) * 80}px` }"
                :title="`${point.date}: ${point.triggered} triggered`"
              />
              <div
                class="trend-bar trend-bar--closed"
                :style="{ height: `${(point.closed / trendMax) * 80}px` }"
                :title="`${point.date}: ${point.closed} closed`"
              />
            </div>
            <div class="trend-label">{{ point.date.substring(5) }}</div>
          </div>
        </div>
        <div class="trend-legend">
          <span class="legend-dot legend-dot--triggered" /> {{ t('dashboard.triggered') }}
          <span class="legend-dot legend-dot--closed" style="margin-left:12px" /> {{ t('dashboard.closed') }}
        </div>
      </n-card>

      <!-- Channel stats table -->
      <div class="two-col">
        <n-card :bordered="false" class="section-card">
          <div class="section-title">{{ t('dashboardV2.channelStats') }}</div>
          <table class="stats-table">
            <thead>
              <tr>
                <th>{{ t('channel.name') }}</th>
                <th>{{ t('dashboard.total') }}</th>
                <th>{{ t('dashboard.pending') }}</th>
                <th>{{ t('dashboard.urgent') }}</th>
                <th>{{ t('dashboard.closed') }}</th>
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
                <td><strong>{{ row.total }}</strong></td>
                <td><span class="stat-value--critical">{{ row.triggered }}</span></td>
                <td><span class="stat-value--critical">{{ row.critical }}</span></td>
                <td><span class="stat-value--success">{{ row.closed }}</span></td>
              </tr>
              <tr v-if="topChannels.length === 0">
                <td colspan="5" class="stats-empty">{{ t('dashboard.noData') }}</td>
              </tr>
            </tbody>
          </table>
        </n-card>

        <!-- Team stats table -->
        <n-card :bordered="false" class="section-card">
          <div class="section-title">{{ t('dashboardV2.teamStats') }}</div>
          <table class="stats-table">
            <thead>
              <tr>
                <th>{{ t('common.name') }}</th>
                <th>{{ t('dashboard.total') }}</th>
                <th>{{ t('dashboard.urgent') }}</th>
                <th>{{ t('dashboard.closed') }}</th>
                <th>{{ t('dashboard.avgMttr') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in topTeams" :key="row.team_id" class="stats-row">
                <td class="stats-name">{{ row.team_name || t('dashboard.ungrouped') }}</td>
                <td><strong>{{ row.total }}</strong></td>
                <td><span class="stat-value--critical">{{ row.critical }}</span></td>
                <td><span class="stat-value--success">{{ row.closed }}</span></td>
                <td>{{ formatSeconds(row.avg_mttr_seconds) }}</td>
              </tr>
              <tr v-if="topTeams.length === 0">
                <td colspan="5" class="stats-empty">{{ t('dashboard.noData') }}</td>
              </tr>
            </tbody>
          </table>
        </n-card>
      </div>

    </n-spin>
  </div>
</template>

<style scoped>
.incident-dashboard { max-width: 1400px; }

.dash-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.dash-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--sre-text-primary);
  margin: 0;
}

.dash-controls { display: flex; align-items: center; gap: 8px; }

.stat-cards {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 14px;
  margin-bottom: 20px;
}

.stat-card {
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
  padding: 18px 16px;
  cursor: pointer;
  transition: border-color var(--sre-duration-fast) ease, background var(--sre-duration-fast) ease;
}

.stat-card:hover {
  border-color: var(--sre-border-strong);
  background: var(--sre-bg-hover);
}

.stat-label {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
  color: var(--sre-text-primary);
}

.stat-value--critical { color: var(--sre-critical); }
.stat-value--success  { color: var(--sre-success); }

.stat-sub {
  font-size: 11px;
  color: var(--sre-text-secondary);
  margin-top: 4px;
}

.section-card {
  border-radius: var(--sre-radius-md);
  margin-bottom: 16px;
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 14px;
}

/* Trend chart */
.trend-chart {
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 100px;
  padding-bottom: 4px;
}

.trend-day {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.trend-bars {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 80px;
}

.trend-bar {
  width: 8px;
  border-radius: 3px 3px 0 0;
  min-height: 2px;
  transition: height 0.3s;
}

.trend-bar--triggered { background: var(--sre-critical); }
.trend-bar--closed    { background: var(--sre-success); }

.trend-label {
  font-size: 9px;
  color: var(--sre-text-secondary);
  white-space: nowrap;
}

.trend-legend {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--sre-text-secondary);
  margin-top: 8px;
}

.legend-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.legend-dot--triggered { background: var(--sre-critical); }
.legend-dot--closed    { background: var(--sre-success); }

/* Two-column layout */
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
  color: var(--sre-text-secondary);
  padding: 4px 8px 8px;
  font-weight: 600;
  border-bottom: var(--sre-hairline);
}

.stats-table td {
  padding: 8px;
  border-bottom: var(--sre-hairline);
  color: var(--sre-text-primary);
}

.stats-row {
  cursor: pointer;
  transition: background 0.15s;
}

.stats-row:hover { background: var(--sre-bg-hover); }

.stats-name {
  font-weight: 500;
}

.stats-empty {
  text-align: center;
  color: var(--sre-text-secondary);
  padding: 24px 16px;
}

@media (max-width: 900px) {
  .two-col { grid-template-columns: 1fr; }
  .stat-cards { grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); }
}
</style>
