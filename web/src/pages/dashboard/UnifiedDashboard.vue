<script setup lang="ts">
/**
 * UnifiedDashboard.vue — Platform homepage with plugin-based widget system.
 * Users can add/remove/reorder widgets via settings panel.
 * Configuration persisted in localStorage.
 */
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, NIcon, NSpin, NPopover, NButton } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { engineApi, incidentApi, dashboardApi, alertGroupsApi } from '@/api'
import type { Incident, AlertGroupItem } from '@/types'
import {
  PulseOutline, BugOutline, RocketOutline, SparklesOutline,
  ChevronForwardOutline, TimeOutline,
  DocumentTextOutline, CalendarOutline, SearchOutline,
  StatsChartOutline, NotificationsOutline, ShieldCheckmarkOutline,
  OptionsOutline, AddOutline, CloseOutline,
  ReorderThreeOutline, EyeOutline, EyeOffOutline,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()
const authStore = useAuthStore()

// ═══════════════════════════════════════════════════════════
// Widget System
// ═══════════════════════════════════════════════════════════

interface WidgetDef {
  id: string
  type: string
  label: string
  icon: any
  size: 'sm' | 'md' | 'lg' | 'full'
  enabled: boolean
  order: number
}

const STORAGE_KEY = 'sre-home-widgets'

const WIDGET_REGISTRY: Record<string, Omit<WidgetDef, 'enabled' | 'order'>> = {
  greeting:       { id: 'greeting',       type: 'greeting',       label: '', icon: null,            size: 'full' },
  moduleStatus:   { id: 'moduleStatus',   type: 'moduleStatus',   label: '', icon: PulseOutline,    size: 'full' },
  myTasks:        { id: 'myTasks',        type: 'myTasks',        label: '', icon: BugOutline,      size: 'lg' },
  recentActivity: { id: 'recentActivity', type: 'recentActivity', label: '', icon: TimeOutline,     size: 'lg' },
  quickAccess:    { id: 'quickAccess',    type: 'quickAccess',    label: '', icon: SearchOutline,   size: 'full' },
}

const DEFAULT_ORDER = ['greeting', 'moduleStatus', 'myTasks', 'recentActivity', 'quickAccess']

function loadWidgets(): WidgetDef[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const saved = JSON.parse(raw) as { id: string; enabled: boolean; order: number }[]
      const result: WidgetDef[] = []
      for (const s of saved) {
        const reg = WIDGET_REGISTRY[s.id]
        if (reg) result.push({ ...reg, enabled: s.enabled, order: s.order })
      }
      // Add any new widgets not in saved config
      for (const id of DEFAULT_ORDER) {
        if (!result.find(w => w.id === id)) {
          const reg = WIDGET_REGISTRY[id]
          result.push({ ...reg, enabled: true, order: result.length })
        }
      }
      return result
    }
  } catch { /* ignore */ }
  return DEFAULT_ORDER.map((id, i) => ({ ...WIDGET_REGISTRY[id], enabled: true, order: i }))
}

function saveWidgets() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(
    widgets.value.map(w => ({ id: w.id, enabled: w.enabled, order: w.order }))
  ))
}

const widgets = ref<WidgetDef[]>(loadWidgets())

// Label mapping (must be computed for i18n reactivity)
const widgetLabels = computed(() => ({
  greeting: t('homepage.widgetGreeting'),
  moduleStatus: t('homepage.moduleStatus'),
  myTasks: t('homepage.myTasks'),
  recentActivity: t('homepage.recentActivity'),
  quickAccess: t('homepage.quickAccess'),
}))

// Apply labels
watch(widgetLabels, (labels) => {
  for (const w of widgets.value) {
    w.label = labels[w.type as keyof typeof labels] || w.type
  }
}, { immediate: true })

const showSettings = ref(false)
const enabledWidgets = computed(() =>
  widgets.value.filter(w => w.enabled).sort((a, b) => a.order - b.order)
)

function toggleWidget(id: string) {
  const w = widgets.value.find(w => w.id === id)
  if (w) { w.enabled = !w.enabled; saveWidgets() }
}

function moveWidget(id: string, dir: -1 | 1) {
  const sorted = widgets.value.filter(w => w.enabled).sort((a, b) => a.order - b.order)
  const idx = sorted.findIndex(w => w.id === id)
  if (idx < 0) return
  const swapIdx = idx + dir
  if (swapIdx < 0 || swapIdx >= sorted.length) return
  const tmp = sorted[idx].order
  sorted[idx].order = sorted[swapIdx].order
  sorted[swapIdx].order = tmp
  saveWidgets()
}

function resetWidgets() {
  widgets.value = DEFAULT_ORDER.map((id, i) => ({ ...WIDGET_REGISTRY[id], enabled: true, order: i }))
  saveWidgets()
}

// ═══════════════════════════════════════════════════════════
// Data Loading
// ═══════════════════════════════════════════════════════════

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
    color: '#3B82F6',
    bgColor: 'rgba(59,130,246,0.08)',
  },
  {
    key: 'oncall',
    label: t('homepage.oncall'),
    desc: t('homepage.oncallDesc', { active: activeIncidents.value.length }),
    icon: BugOutline,
    status: activeIncidents.value.length === 0 ? 'ok' : activeIncidents.value.some(i => i.severity === 'critical') ? 'critical' : 'warning',
    statusText: activeIncidents.value.length === 0 ? t('homepage.allHealthy') : t('homepage.nActive', { count: activeIncidents.value.length }),
    route: '/oncall/overview',
    color: '#EC4899',
    bgColor: 'rgba(236,72,153,0.08)',
  },
  {
    key: 'deploy',
    label: t('homepage.deployAgent'),
    desc: t('homepage.deployDesc'),
    icon: RocketOutline,
    status: 'coming',
    statusText: t('homepage.comingSoon'),
    route: null,
    color: '#10B981',
    bgColor: 'rgba(16,185,129,0.08)',
  },
  {
    key: 'ai',
    label: t('homepage.aiAgent'),
    desc: t('homepage.aiDesc'),
    icon: SparklesOutline,
    status: 'coming',
    statusText: t('homepage.comingSoon'),
    route: null,
    color: '#8B5CF6',
    bgColor: 'rgba(139,92,246,0.08)',
  },
])

// Activity feed
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

// Quick access items with colors
const quickLinks = computed(() => [
  { label: t('menu.alertRules'),     icon: DocumentTextOutline,        route: '/alert/rules',            color: '#3B82F6', bg: 'rgba(59,130,246,0.08)' },
  { label: t('menu.schedule'),       icon: CalendarOutline,            route: '/oncall/schedule',        color: '#F59E0B', bg: 'rgba(245,158,11,0.08)' },
  { label: t('menu.explore'),        icon: SearchOutline,              route: '/alert/explore',          color: '#06B6D4', bg: 'rgba(6,182,212,0.08)' },
  { label: t('menu.dashboards'),     icon: StatsChartOutline,          route: '/alert/dashboards',       color: '#8B5CF6', bg: 'rgba(139,92,246,0.08)' },
  { label: t('menu.notifyPolicies'), icon: NotificationsOutline,       route: '/alert/notify/policies',  color: '#EF4444', bg: 'rgba(239,68,68,0.08)' },
  { label: t('menu.suppression'),    icon: ShieldCheckmarkOutline,     route: '/alert/suppression',      color: '#10B981', bg: 'rgba(16,185,129,0.08)' },
])

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

function uptimeDays(): number {
  if (!engineUptime.value) return 0
  const match = engineUptime.value.match(/(\d+)d/)
  return match ? parseInt(match[1]) : 0
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

onMounted(load)
</script>

<template>
  <div class="homepage">
    <NSpin :show="loading">
      <div class="hp-container">
        <!-- Settings button -->
        <div class="hp-settings-bar">
          <n-popover v-model:show="showSettings" trigger="click" placement="bottom-end" :show-arrow="false">
            <template #trigger>
              <button class="hp-settings-btn" :class="{ active: showSettings }">
                <n-icon :component="OptionsOutline" :size="16" />
                <span>{{ t('homepage.customize') }}</span>
              </button>
            </template>
            <div class="settings-panel">
              <div class="sp-header">
                <span class="sp-title">{{ t('homepage.widgetSettings') }}</span>
                <button class="sp-reset" @click="resetWidgets">{{ t('homepage.resetLayout') }}</button>
              </div>
              <div class="sp-list">
                <div
                  v-for="w in widgets"
                  :key="w.id"
                  class="sp-item"
                  :class="{ 'sp-item--off': !w.enabled }"
                >
                  <div class="sp-item-left">
                    <n-icon v-if="w.icon" :component="w.icon" :size="14" class="sp-item-icon" />
                    <span class="sp-item-label">{{ w.label }}</span>
                  </div>
                  <div class="sp-item-actions">
                    <button v-if="w.enabled" class="sp-arrow" @click="moveWidget(w.id, -1)" :disabled="w.order === 0">↑</button>
                    <button v-if="w.enabled" class="sp-arrow" @click="moveWidget(w.id, 1)">↓</button>
                    <button class="sp-toggle" @click="toggleWidget(w.id)">
                      <n-icon :component="w.enabled ? EyeOutline : EyeOffOutline" :size="14" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </n-popover>
        </div>

        <!-- Widget Grid -->
        <div class="bento-grid">
          <template v-for="widget in enabledWidgets" :key="widget.id">

            <!-- ═══ Greeting ═══ -->
            <div v-if="widget.type === 'greeting'" class="bento-item bento-full">
              <div class="greeting-card">
                <div class="gc-content">
                  <h1 class="gc-hello">{{ greeting }}</h1>
                  <p class="gc-context">
                    <span v-if="engineOk" class="gc-engine gc-engine--ok">
                      <span class="dot dot-ok"></span>
                      {{ t('homepage.engineRunning', { days: uptimeDays() }) }}
                    </span>
                    <span v-else class="gc-engine gc-engine--down">
                      <span class="dot dot-critical"></span>
                      {{ t('homepage.engineDown') }}
                    </span>
                  </p>
                </div>
                <div class="gc-deco">
                  <div class="gc-orb gc-orb--1"></div>
                  <div class="gc-orb gc-orb--2"></div>
                  <div class="gc-orb gc-orb--3"></div>
                </div>
              </div>
            </div>

            <!-- ═══ Module Status ═══ -->
            <div v-else-if="widget.type === 'moduleStatus'" class="bento-item bento-full">
              <div class="module-strip">
                <div
                  v-for="mod in modules"
                  :key="mod.key"
                  class="mod-card"
                  :class="{ 'mod-card--clickable': !!mod.route }"
                  @click="mod.route && router.push(mod.route)"
                >
                  <div class="mod-icon" :style="{ background: mod.bgColor, color: mod.color }">
                    <n-icon :component="mod.icon" :size="18" />
                  </div>
                  <div class="mod-info">
                    <div class="mod-label">{{ mod.label }}</div>
                    <div class="mod-desc">{{ mod.desc }}</div>
                  </div>
                  <div class="mod-status">
                    <span class="dot" :class="statusDotClass(mod.status)"></span>
                    <span class="mod-status-text">{{ mod.statusText }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- ═══ My Tasks ═══ -->
            <div v-else-if="widget.type === 'myTasks'" class="bento-item bento-lg">
              <div class="widget-card">
                <div class="wc-header">
                  <n-icon :component="BugOutline" :size="16" class="wc-icon" />
                  <span class="wc-title">{{ t('homepage.myTasks') }}</span>
                  <span v-if="activeIncidents.length" class="wc-badge">{{ activeIncidents.length }}</span>
                </div>
                <div class="wc-body">
                  <div v-if="activeIncidents.length" class="tasks-list">
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
                  <div v-else class="wc-empty">
                    <span class="wc-empty-icon">✓</span>
                    <span>{{ t('homepage.noTasks') }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- ═══ Recent Activity ═══ -->
            <div v-else-if="widget.type === 'recentActivity'" class="bento-item bento-lg">
              <div class="widget-card">
                <div class="wc-header">
                  <n-icon :component="TimeOutline" :size="16" class="wc-icon" />
                  <span class="wc-title">{{ t('homepage.recentActivity') }}</span>
                </div>
                <div class="wc-body">
                  <div v-if="activity.length" class="activity-list">
                    <div
                      v-for="(item, idx) in activity"
                      :key="idx"
                      class="act-row"
                      @click="router.push(item.route)"
                    >
                      <div class="act-time">
                        <span class="act-dot" :style="{ background: sevColor(item.severity) }"></span>
                        {{ relTime(item.time) }}
                      </div>
                      <span class="act-text">{{ item.text }}</span>
                    </div>
                  </div>
                  <div v-else class="wc-empty">
                    <span class="wc-empty-icon">✓</span>
                    <span>{{ t('homepage.noActivity') }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- ═══ Quick Access ═══ -->
            <div v-else-if="widget.type === 'quickAccess'" class="bento-item bento-full">
              <div class="quick-grid">
                <div
                  v-for="link in quickLinks"
                  :key="link.route"
                  class="quick-btn"
                  @click="router.push(link.route)"
                >
                  <div class="quick-icon" :style="{ background: link.bg, color: link.color }">
                    <n-icon :component="link.icon" :size="18" />
                  </div>
                  <span class="quick-label">{{ link.label }}</span>
                </div>
              </div>
            </div>

          </template>
        </div>
      </div>
    </NSpin>
  </div>
</template>

<style scoped>
.homepage {
  min-height: 100vh;
  font-family: var(--sre-font-sans);
}

.hp-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 24px 32px 48px;
  position: relative;
}

/* ===== Settings Bar ===== */
.hp-settings-bar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 16px;
}

.hp-settings-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-pill);
  background: var(--sre-bg-card);
  color: var(--sre-text-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 150ms var(--sre-ease-out);
  font-family: var(--sre-font-sans);
}
.hp-settings-btn:hover, .hp-settings-btn.active {
  background: var(--sre-primary-soft);
  border-color: var(--sre-primary-ring);
  color: var(--sre-primary);
}

/* ===== Settings Panel ===== */
.settings-panel { min-width: 280px; padding: 0; }
.sp-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 16px; border-bottom: 1px solid var(--sre-border);
}
.sp-title { font-size: 13px; font-weight: 600; color: var(--sre-text-primary); }
.sp-reset {
  font-size: 11px; color: var(--sre-primary); background: none; border: none;
  cursor: pointer; font-family: var(--sre-font-sans); font-weight: 500;
}
.sp-list { padding: 8px 0; }
.sp-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 16px; transition: background 100ms;
}
.sp-item:hover { background: var(--sre-bg-hover); }
.sp-item--off { opacity: 0.5; }
.sp-item-left { display: flex; align-items: center; gap: 8px; }
.sp-item-icon { color: var(--sre-text-muted); }
.sp-item-label { font-size: 13px; color: var(--sre-text-primary); }
.sp-item-actions { display: flex; align-items: center; gap: 4px; }
.sp-arrow {
  width: 24px; height: 24px; border: none; background: none;
  color: var(--sre-text-muted); cursor: pointer; font-size: 14px;
  border-radius: 4px; display: flex; align-items: center; justify-content: center;
}
.sp-arrow:hover { background: var(--sre-bg-hover); color: var(--sre-text-primary); }
.sp-arrow:disabled { opacity: 0.3; cursor: default; }
.sp-toggle {
  width: 28px; height: 28px; border: none; background: none;
  color: var(--sre-text-muted); cursor: pointer; border-radius: 4px;
  display: flex; align-items: center; justify-content: center;
}
.sp-toggle:hover { background: var(--sre-bg-hover); color: var(--sre-primary); }

/* ===== Bento Grid ===== */
.bento-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}
.bento-full { grid-column: 1 / -1; }
.bento-lg { grid-column: span 1; }

/* ===== Greeting Card ===== */
.greeting-card {
  position: relative;
  overflow: hidden;
  padding: 36px 40px;
  border-radius: var(--sre-radius-xl, 16px);
  background: linear-gradient(135deg, var(--sre-primary) 0%, #FB923C 60%, #F59E0B 100%);
  color: white;
}
.gc-content { position: relative; z-index: 1; }
.gc-hello {
  font-family: var(--sre-font-display);
  font-size: 28px;
  font-weight: 700;
  margin: 0;
  line-height: 1.2;
  letter-spacing: -0.02em;
}
.gc-context {
  margin: 8px 0 0;
  font-size: 14px;
  opacity: 0.9;
  display: flex;
  align-items: center;
  gap: 8px;
}
.gc-engine { display: inline-flex; align-items: center; gap: 6px; }
.gc-engine--ok .dot { background: #4ADE80; box-shadow: 0 0 0 3px rgba(74,222,128,0.3); }
.gc-engine--down .dot { background: #FCA5A5; box-shadow: 0 0 0 3px rgba(252,165,165,0.3); }

/* Decorative orbs */
.gc-deco { position: absolute; inset: 0; pointer-events: none; }
.gc-orb {
  position: absolute;
  border-radius: 50%;
  background: rgba(255,255,255,0.12);
}
.gc-orb--1 { width: 200px; height: 200px; top: -60px; right: -40px; }
.gc-orb--2 { width: 120px; height: 120px; bottom: -30px; right: 140px; }
.gc-orb--3 { width: 80px; height: 80px; top: 20px; right: 200px; background: rgba(255,255,255,0.06); }

/* ===== Module Strip ===== */
.module-strip {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}
.mod-card {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-lg, 12px);
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: border-color 200ms var(--sre-ease-out), box-shadow 200ms var(--sre-ease-out);
}
.mod-card--clickable { cursor: pointer; }
.mod-card--clickable:hover {
  border-color: var(--sre-border-strong);
  box-shadow: var(--sre-shadow-sm);
}
.mod-icon {
  width: 36px; height: 36px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.mod-info { min-width: 0; }
.mod-label {
  font-size: 14px; font-weight: 600; color: var(--sre-text-primary);
  font-family: var(--sre-font-display);
}
.mod-desc {
  font-size: 12px; color: var(--sre-text-tertiary); margin-top: 2px;
}
.mod-status {
  display: flex; align-items: center; gap: 6px;
  font-size: 12px; color: var(--sre-text-secondary); margin-top: auto;
}
.mod-status-text { font-weight: 500; }

/* ===== Dots ===== */
.dot {
  width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; display: inline-block;
}
.dot-ok { background: var(--sre-success); box-shadow: 0 0 0 3px rgba(34,197,94,0.15); }
.dot-warning { background: var(--sre-warning); box-shadow: 0 0 0 3px rgba(245,158,11,0.15); }
.dot-critical { background: var(--sre-critical); box-shadow: 0 0 0 3px rgba(239,68,68,0.15); }
.dot-coming { background: var(--sre-text-muted); }

/* ===== Widget Cards (Tasks & Activity) ===== */
.widget-card {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-lg, 12px);
  overflow: hidden;
  height: 100%;
  display: flex;
  flex-direction: column;
}
.wc-header {
  display: flex; align-items: center; gap: 8px;
  padding: 14px 18px; border-bottom: 1px solid var(--sre-border);
}
.wc-icon { color: var(--sre-primary); }
.wc-title {
  font-size: 14px; font-weight: 600; color: var(--sre-text-primary);
  font-family: var(--sre-font-display);
}
.wc-badge {
  font-size: 11px; font-weight: 700; color: white;
  background: var(--sre-critical); padding: 1px 7px; border-radius: 10px;
  margin-left: auto;
}
.wc-body { flex: 1; overflow-y: auto; }

/* ===== Tasks List ===== */
.tasks-list { display: flex; flex-direction: column; }
.task-row {
  display: flex; align-items: center; gap: 12px;
  padding: 12px 18px; cursor: pointer;
  transition: background 120ms var(--sre-ease-out);
  border-bottom: 1px solid var(--sre-border);
}
.task-row:last-child { border-bottom: none; }
.task-row:hover { background: var(--sre-bg-hover); }
.task-sev {
  font-size: 9px; font-weight: 700; color: #fff;
  padding: 2px 6px; border-radius: 4px; letter-spacing: 0.04em; flex-shrink: 0;
}
.task-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.task-title {
  font-size: 13px; font-weight: 500; color: var(--sre-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.task-meta { font-size: 11px; color: var(--sre-text-tertiary); }
.task-arrow { color: var(--sre-text-muted); flex-shrink: 0; }

/* ===== Activity List ===== */
.activity-list { display: flex; flex-direction: column; }
.act-row {
  display: flex; flex-direction: column; gap: 4px;
  padding: 10px 18px; cursor: pointer;
  transition: background 120ms var(--sre-ease-out);
  border-bottom: 1px solid var(--sre-border);
}
.act-row:last-child { border-bottom: none; }
.act-row:hover { background: var(--sre-bg-hover); }
.act-time {
  display: flex; align-items: center; gap: 6px;
  font-size: 11px; color: var(--sre-text-muted); white-space: nowrap;
}
.act-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.act-text {
  font-size: 13px; color: var(--sre-text-secondary);
  line-height: 1.4; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}

/* ===== Empty State ===== */
.wc-empty {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 8px; padding: 36px 16px; color: var(--sre-text-muted); font-size: 13px;
}
.wc-empty-icon {
  width: 32px; height: 32px; border-radius: 50%;
  background: var(--sre-success); color: white;
  display: flex; align-items: center; justify-content: center;
  font-size: 16px; font-weight: 700;
}

/* ===== Quick Access ===== */
.quick-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 12px;
}
.quick-btn {
  display: flex; flex-direction: column; align-items: center; gap: 10px;
  padding: 20px 12px;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-lg, 12px);
  cursor: pointer;
  transition: border-color 200ms var(--sre-ease-out), box-shadow 200ms var(--sre-ease-out), transform 150ms var(--sre-ease-out);
}
.quick-btn:hover {
  border-color: var(--sre-border-strong);
  box-shadow: var(--sre-shadow-sm);
  transform: translateY(-2px);
}
.quick-icon {
  width: 40px; height: 40px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center;
}
.quick-label {
  font-size: 13px; font-weight: 500; color: var(--sre-text-secondary);
  text-align: center;
}

/* ===== Responsive ===== */
@media (max-width: 1200px) {
  .module-strip { grid-template-columns: repeat(2, 1fr); }
  .quick-grid { grid-template-columns: repeat(3, 1fr); }
}

@media (max-width: 768px) {
  .hp-container { padding: 16px; }
  .bento-grid { grid-template-columns: 1fr; }
  .bento-lg { grid-column: span 1; }
  .module-strip { grid-template-columns: 1fr; }
  .quick-grid { grid-template-columns: repeat(2, 1fr); }
  .greeting-card { padding: 24px; }
  .gc-hello { font-size: 22px; }
}

/* ===== Reduced Motion ===== */
@media (prefers-reduced-motion: reduce) {
  .mod-card, .quick-btn, .task-row, .act-row {
    transition: none;
  }
}
</style>
