<script setup lang="ts">
/**
 * UnifiedDashboard.vue — Platform-level homepage.
 * Shows module health, my tasks, recent activity, and quick navigation.
 * NOT a data dashboard — charts live in sub-module dashboards.
 */
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, NIcon, NSpin } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { engineApi, incidentApi, dashboardApi, alertGroupsApi } from '@/api'
import type { Incident, AlertGroupItem } from '@/types'
import {
  PulseOutline, BugOutline, RocketOutline, SparklesOutline,
  ChevronForwardOutline, TimeOutline,
  DocumentTextOutline, CalendarOutline, SearchOutline,
  StatsChartOutline, NotificationsOutline, ShieldCheckmarkOutline,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()
const authStore = useAuthStore()

const loading = ref(true)
const engineOk = ref(false)
const engineUptime = ref('')
const activeIncidents = ref<Incident[]>([])
const recentIncidents = ref<Incident[]>([])
const firingAlerts = ref<AlertGroupItem[]>([])
const totalRules = ref(0)
const activeAlerts = ref(0)

const userName = computed(() => authStore.user?.username || authStore.user?.email || 'SRE')

const greeting = computed(() => {
  const hour = new Date().getHours()
  const key = hour < 12 ? 'homepage.greetingMorning' : hour < 18 ? 'homepage.greetingAfternoon' : 'homepage.greetingEvening'
  return t(key, { name: userName.value })
})

// Module health cards
const modules = computed(() => [
  {
    key: 'monitor',
    label: t('homepage.monitor'),
    desc: t('homepage.monitorDesc', { rules: totalRules.value }),
    icon: PulseOutline,
    status: engineOk.value ? 'ok' : 'down',
    statusText: engineOk.value ? t('homepage.nActive', { count: activeAlerts.value }) : t('homepage.engineDown'),
    route: '/alert/overview',
  },
  {
    key: 'oncall',
    label: t('homepage.oncall'),
    desc: t('homepage.oncallDesc', { active: activeIncidents.value.length }),
    icon: BugOutline,
    status: activeIncidents.value.length === 0 ? 'ok' : activeIncidents.value.some(i => i.severity === 'critical') ? 'critical' : 'warning',
    statusText: activeIncidents.value.length === 0 ? t('homepage.allHealthy') : t('homepage.nActive', { count: activeIncidents.value.length }),
    route: '/oncall/overview',
  },
  {
    key: 'deploy',
    label: t('homepage.deployAgent'),
    desc: t('homepage.deployDesc'),
    icon: RocketOutline,
    status: 'coming',
    statusText: t('homepage.comingSoon'),
    route: null,
  },
  {
    key: 'ai',
    label: t('homepage.aiAgent'),
    desc: t('homepage.aiDesc'),
    icon: SparklesOutline,
    status: 'coming',
    statusText: t('homepage.comingSoon'),
    route: null,
  },
])

// Activity feed: merge incidents + alert groups, sorted by time
interface ActivityItem {
  time: string
  type: 'incident' | 'alert'
  text: string
  severity?: string
  route: string
}

const activity = computed<ActivityItem[]>(() => {
  const items: ActivityItem[] = []

  for (const inc of recentIncidents.value.slice(0, 5)) {
    items.push({
      time: inc.triggered_at || inc.created_at,
      type: 'incident',
      text: t('homepage.incidentCreated', { id: inc.id }) + (inc.title ? `: ${inc.title}` : ''),
      severity: inc.severity,
      route: `/oncall/incidents/${inc.id}`,
    })
  }

  for (const ag of firingAlerts.value.slice(0, 5)) {
    items.push({
      time: ag.latest_fired_at,
      type: 'alert',
      text: t('homepage.alertFired', { name: ag.alert_name }) + (ag.total_count > 1 ? ` (${ag.total_count})` : ''),
      severity: Object.keys(ag.severity_breakdown || {})[0],
      route: '/alert/events',
    })
  }

  items.sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime())
  return items.slice(0, 10)
})

// Helpers
function relTime(ts: string): string {
  if (!ts) return ''
  const diff = Math.floor((Date.now() - new Date(ts).getTime()) / 1000)
  if (diff < 60) return t('common.secsAgo', { n: diff })
  if (diff < 3600) return t('common.minsAgo', { n: Math.floor(diff / 60) })
  if (diff < 86400) return t('common.hoursAgo', { n: Math.floor(diff / 3600) })
  return t('common.daysAgo', { n: Math.floor(diff / 86400) })
}

function sevColor(sev?: string): string {
  if (sev === 'critical') return 'var(--sre-critical)'
  if (sev === 'warning') return 'var(--sre-warning)'
  return 'var(--sre-info)'
}

function statusDotClass(status: string): string {
  if (status === 'ok') return 'dot-ok'
  if (status === 'warning') return 'dot-warning'
  if (status === 'critical') return 'dot-critical'
  return 'dot-coming'
}

// Data loading
async function load() {
  loading.value = true
  try {
    const [engineRes, statsRes, incRes, alertRes] = await Promise.allSettled([
      engineApi.getStatus(),
      dashboardApi.getStats(),
      incidentApi.list({ status: 'active', page_size: 5 }),
      alertGroupsApi.list({ status: 'firing' }),
    ])

    if (engineRes.status === 'fulfilled') {
      const d = engineRes.value.data.data
      engineOk.value = d.running
      engineUptime.value = d.uptime || ''
    }
    if (statsRes.status === 'fulfilled') {
      const d = statsRes.value.data.data
      totalRules.value = d.total_rules
      activeAlerts.value = d.active_alerts
    }
    if (incRes.status === 'fulfilled') {
      const items = incRes.value.data.data?.list || incRes.value.data.data || []
      activeIncidents.value = items
      recentIncidents.value = items
    }
    if (alertRes.status === 'fulfilled') {
      firingAlerts.value = alertRes.value.data.data || []
    }
  } catch (e: any) {
    message.error(e?.message || t('homepage.loadFailed'))
  } finally {
    loading.value = false
  }
}

function uptimeDays(): number {
  if (!engineUptime.value) return 0
  const match = engineUptime.value.match(/(\d+)d/)
  return match ? parseInt(match[1]) : 0
}

onMounted(load)
</script>

<template>
  <div class="homepage">
    <NSpin :show="loading">
      <!-- ① Greeting + Context -->
      <header class="hp-header">
        <div class="hp-greeting">
          <h1 class="hp-hello">{{ greeting }}</h1>
          <p class="hp-context">
            <span v-if="engineOk" class="hp-engine-ok">
              <span class="dot dot-ok"></span>
              {{ t('homepage.engineRunning', { days: uptimeDays() }) }}
            </span>
            <span v-else class="hp-engine-down">
              <span class="dot dot-critical"></span>
              {{ t('homepage.engineDown') }}
            </span>
          </p>
        </div>
      </header>

      <!-- ② Module Health Cards -->
      <section class="hp-section">
        <h2 class="hp-section-title">{{ t('homepage.moduleStatus') }}</h2>
        <div class="module-grid">
          <div
            v-for="mod in modules"
            :key="mod.key"
            class="module-card"
            :class="{ clickable: !!mod.route, 'module-card-primary': mod.key === 'monitor' }"
            @click="mod.route && router.push(mod.route)"
          >
            <div class="mod-body">
              <div class="mod-header">
                <div class="mod-icon">
                  <n-icon :component="mod.icon" :size="18" />
                </div>
                <div class="mod-info">
                  <div class="mod-label">{{ mod.label }}</div>
                  <div class="mod-desc">{{ mod.desc }}</div>
                </div>
              </div>
              <div class="mod-status">
                <span class="dot" :class="statusDotClass(mod.status)"></span>
                <span class="mod-status-text">{{ mod.statusText }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- ③ My Tasks -->
      <section class="hp-section">
        <h2 class="hp-section-title">{{ t('homepage.myTasks') }}</h2>
        <div class="tasks-list" v-if="activeIncidents.length">
          <div
            v-for="inc in activeIncidents"
            :key="inc.id"
            class="task-row"
            @click="router.push(`/oncall/incidents/${inc.id}`)"
          >
            <span class="task-sev" :style="{ background: sevColor(inc.severity) }">
              {{ inc.severity?.toUpperCase() }}
            </span>
            <div class="task-body">
              <span class="task-title">{{ inc.title || `#${inc.id}` }}</span>
              <span class="task-meta">
                {{ relTime(inc.triggered_at || inc.created_at) }}
                <template v-if="inc.assigned_user"> · {{ inc.assigned_user.username }}</template>
                <template v-else> · {{ t('homepage.unclaimed') }}</template>
              </span>
            </div>
            <n-icon :component="ChevronForwardOutline" :size="14" class="task-arrow" />
          </div>
        </div>
        <div v-else class="hp-empty">{{ t('homepage.noTasks') }}</div>
      </section>

      <!-- ④ Recent Activity -->
      <section class="hp-section">
        <h2 class="hp-section-title">{{ t('homepage.recentActivity') }}</h2>
        <div class="activity-list" v-if="activity.length">
          <div v-for="(item, idx) in activity" :key="idx" class="activity-item"
            @click="router.push(item.route)">
            <div class="act-time">
              <n-icon :component="TimeOutline" :size="12" />
              {{ relTime(item.time) }}
            </div>
            <span class="act-dot" :style="{ background: sevColor(item.severity) }"></span>
            <span class="act-text">{{ item.text }}</span>
          </div>
        </div>
        <div v-else class="hp-empty">{{ t('homepage.noActivity') }}</div>
      </section>

      <!-- ⑤ Quick Access -->
      <section class="hp-section">
        <h2 class="hp-section-title">{{ t('homepage.quickAccess') }}</h2>
        <div class="access-grid">
          <div class="access-btn" @click="router.push('/alert/rules')">
            <n-icon :component="DocumentTextOutline" :size="16" class="access-icon" />
            <span class="access-label">{{ t('menu.alertRules') }}</span>
          </div>
          <div class="access-btn" @click="router.push('/oncall/schedule')">
            <n-icon :component="CalendarOutline" :size="16" class="access-icon" />
            <span class="access-label">{{ t('menu.schedule') }}</span>
          </div>
          <div class="access-btn" @click="router.push('/alert/explore')">
            <n-icon :component="SearchOutline" :size="16" class="access-icon" />
            <span class="access-label">{{ t('menu.explore') }}</span>
          </div>
          <div class="access-btn" @click="router.push('/alert/dashboards')">
            <n-icon :component="StatsChartOutline" :size="16" class="access-icon" />
            <span class="access-label">{{ t('menu.dashboards') }}</span>
          </div>
          <div class="access-btn" @click="router.push('/alert/notify/policies')">
            <n-icon :component="NotificationsOutline" :size="16" class="access-icon" />
            <span class="access-label">{{ t('menu.notifyPolicies') }}</span>
          </div>
          <div class="access-btn" @click="router.push('/alert/suppression')">
            <n-icon :component="ShieldCheckmarkOutline" :size="16" class="access-icon" />
            <span class="access-label">{{ t('menu.suppression') }}</span>
          </div>
        </div>
      </section>
    </NSpin>
  </div>
</template>

<style scoped>
.homepage {
  max-width: 1100px;
  display: flex;
  flex-direction: column;
  gap: 28px;
  font-family: var(--sre-font-sans);
}

/* ===== Header ===== */
.hp-header {
  padding: 4px 0;
}

.hp-hello {
  font-family: var(--sre-font-display);
  font-size: var(--sre-fs-3xl);
  font-weight: var(--sre-fw-bold);
  color: var(--sre-text-primary);
  margin: 0;
  line-height: var(--sre-lh-tight);
}

.hp-context {
  margin: 6px 0 0;
  font-size: var(--sre-fs-sm);
  color: var(--sre-text-tertiary);
  display: flex;
  align-items: center;
  gap: 6px;
}

.hp-engine-ok, .hp-engine-down {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

/* ===== Section ===== */
.hp-section-title {
  font-family: var(--sre-font-display);
  font-size: var(--sre-fs-xs);
  font-weight: var(--sre-fw-semibold);
  color: var(--sre-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  margin: 0 0 12px;
}

.hp-empty {
  padding: 24px;
  text-align: center;
  color: var(--sre-text-muted);
  font-size: var(--sre-fs-sm);
  background: var(--sre-bg-card);
  border-radius: var(--sre-radius-lg);
  border: 1px solid var(--sre-border);
}

/* ===== Module Cards ===== */
.module-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.module-card {
  background: var(--sre-bg-card);
  border-radius: var(--sre-radius-lg);
  border: 1px solid var(--sre-border);
  overflow: hidden;
  transition: border-color 200ms var(--sre-ease-out), box-shadow 200ms var(--sre-ease-out);
}

.module-card-primary {
  grid-column: span 2;
}

.module-card.clickable {
  cursor: pointer;
}

.module-card.clickable:hover {
  border-color: var(--sre-border-strong);
  box-shadow: var(--sre-shadow-sm);
}

.mod-body {
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.mod-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.mod-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.mod-info {
  min-width: 0;
}

.mod-label {
  font-size: var(--sre-fs-md);
  font-weight: var(--sre-fw-semibold);
  color: var(--sre-text-primary);
}

.mod-desc {
  font-size: var(--sre-fs-2xs);
  color: var(--sre-text-tertiary);
  margin-top: 1px;
}

.mod-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--sre-fs-2xs);
  color: var(--sre-text-secondary);
}

.mod-status-text {
  font-weight: var(--sre-fw-medium);
}

/* ===== Dots ===== */
.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
  display: inline-block;
}

.dot-ok {
  background: var(--sre-success);
  box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.15);
}

.dot-warning {
  background: var(--sre-warning);
  box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.15);
}

.dot-critical {
  background: var(--sre-critical);
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.15);
}

.dot-coming {
  background: var(--sre-text-muted);
}

/* ===== Tasks List ===== */
.tasks-list {
  background: var(--sre-bg-card);
  border-radius: var(--sre-radius-lg);
  border: 1px solid var(--sre-border);
  overflow: hidden;
}

.task-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 150ms var(--sre-ease-out);
  border-bottom: 1px solid var(--sre-border);
}

.task-row:last-child {
  border-bottom: none;
}

.task-row:hover {
  background: var(--sre-bg-hover);
}

.task-sev {
  font-size: 9px;
  font-weight: var(--sre-fw-bold);
  color: #fff;
  padding: 2px 6px;
  border-radius: 4px;
  letter-spacing: 0.04em;
  flex-shrink: 0;
}

.task-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.task-title {
  font-size: var(--sre-fs-md);
  font-weight: var(--sre-fw-medium);
  color: var(--sre-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.task-meta {
  font-size: var(--sre-fs-2xs);
  color: var(--sre-text-tertiary);
}

.task-arrow {
  color: var(--sre-text-muted);
  flex-shrink: 0;
}

/* ===== Activity Timeline ===== */
.activity-list {
  background: var(--sre-bg-card);
  border-radius: var(--sre-radius-lg);
  border: 1px solid var(--sre-border);
  padding: 4px 0;
}

.activity-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 150ms var(--sre-ease-out);
}

.activity-item:hover {
  background: var(--sre-bg-hover);
}

.act-time {
  font-size: var(--sre-fs-2xs);
  color: var(--sre-text-muted);
  white-space: nowrap;
  min-width: 60px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.act-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-top: 5px;
}

.act-text {
  font-size: var(--sre-fs-sm);
  color: var(--sre-text-secondary);
  line-height: var(--sre-lh-snug);
}

/* ===== Quick Access ===== */
.access-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 10px;
}

.access-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 14px 10px;
  background: var(--sre-bg-card);
  border-radius: var(--sre-radius-md);
  border: 1px solid var(--sre-border);
  cursor: pointer;
  transition: border-color 200ms var(--sre-ease-out), background 200ms var(--sre-ease-out);
}

.access-btn:hover {
  border-color: var(--sre-border-strong);
  background: var(--sre-bg-hover);
}

.access-icon {
  color: var(--sre-text-muted);
  flex-shrink: 0;
}

.access-label {
  font-size: var(--sre-fs-sm);
  font-weight: var(--sre-fw-medium);
  color: var(--sre-text-secondary);
}

/* ===== Responsive ===== */
@media (max-width: 1200px) {
  .module-grid { grid-template-columns: repeat(2, 1fr); }
  .module-card-primary { grid-column: span 2; }
  .access-grid { grid-template-columns: repeat(3, 1fr); }
}

@media (max-width: 768px) {
  .module-grid { grid-template-columns: 1fr; }
  .module-card-primary { grid-column: span 1; }
  .access-grid { grid-template-columns: repeat(2, 1fr); }
}

/* ===== Reduced Motion ===== */
@media (prefers-reduced-motion: reduce) {
  .module-card, .task-row, .activity-item, .access-btn {
    transition: none;
  }
}
</style>
