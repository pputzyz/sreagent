<script setup lang="ts">
/**
 * UnifiedDashboard.vue — Platform homepage with plugin-based widget system.
 * Users can add/remove/reorder widgets and customize content via settings panel.
 * Configuration persisted in localStorage.
 */
import { ref, reactive, computed, onMounted, watch, type Component } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, useDialog, NIcon, NSpin, NPopover, NButton, NInput } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { engineApi, incidentApi, dashboardApi, alertGroupsApi, scheduleApi, aiAgentApi, aiModuleApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import type { Incident, AlertGroupItem, Schedule } from '@/types'
import {
  PulseOutline, BugOutline, RocketOutline, SparklesOutline,
  ChevronForwardOutline, TimeOutline,
  DocumentTextOutline, CalendarOutline, SearchOutline,
  StatsChartOutline, NotificationsOutline, ShieldCheckmarkOutline,
  OptionsOutline, AddOutline, CloseOutline, CreateOutline,
  PeopleOutline, LinkOutline, BookmarkOutline,
  EyeOutline, EyeOffOutline, LibraryOutline,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const router = useRouter()
const authStore = useAuthStore()

// ═══════════════════════════════════════════════════════════
// Widget System
// ═══════════════════════════════════════════════════════════

interface WidgetDef {
  id: string
  type: string
  label: string
  icon: Component | undefined
  size: 'sm' | 'md' | 'lg' | 'full'
  enabled: boolean
  order: number
}

const STORAGE_KEY = 'sre-home-widgets'
const QUICK_KEY = 'sre-home-quick-links'
const PINNED_KEY = 'sre-home-pinned'

const WIDGET_REGISTRY: Record<string, Omit<WidgetDef, 'enabled' | 'order'>> = {
  greeting:       { id: 'greeting',       type: 'greeting',       label: '', icon: undefined,            size: 'full' },
  moduleStatus:   { id: 'moduleStatus',   type: 'moduleStatus',   label: '', icon: PulseOutline,         size: 'full' },
  myTasks:        { id: 'myTasks',        type: 'myTasks',        label: '', icon: BugOutline,           size: 'lg' },
  oncallSchedule: { id: 'oncallSchedule', type: 'oncallSchedule', label: '', icon: PeopleOutline,        size: 'lg' },
  recentActivity: { id: 'recentActivity', type: 'recentActivity', label: '', icon: TimeOutline,          size: 'lg' },
  pinnedItems:    { id: 'pinnedItems',    type: 'pinnedItems',    label: '', icon: BookmarkOutline,       size: 'lg' },
  quickAccess:    { id: 'quickAccess',    type: 'quickAccess',    label: '', icon: SearchOutline,         size: 'full' },
}

const DEFAULT_ORDER = ['greeting', 'moduleStatus', 'myTasks', 'oncallSchedule', 'recentActivity', 'pinnedItems', 'quickAccess']

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

const widgetLabels = computed(() => ({
  greeting: t('homepage.widgetGreeting'),
  moduleStatus: t('homepage.moduleStatus'),
  myTasks: t('homepage.myTasks'),
  oncallSchedule: t('homepage.oncallSchedule'),
  recentActivity: t('homepage.recentActivity'),
  pinnedItems: t('homepage.pinnedItems'),
  quickAccess: t('homepage.quickAccess'),
}))

watch(widgetLabels, (labels) => {
  for (const w of widgets.value) {
    w.label = labels[w.type as keyof typeof labels] || w.type
  }
}, { immediate: true })

const showSettings = ref(false)
const settingsTab = ref<'widgets' | 'quicklinks' | 'pinned'>('widgets')

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
  dialog.warning({
    title: t('common.confirm'),
    content: t('homepage.confirmResetLayout'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => {
      widgets.value = DEFAULT_ORDER.map((id, i) => ({ ...WIDGET_REGISTRY[id], enabled: true, order: i }))
      saveWidgets()
    },
  })
}

// ═══════════════════════════════════════════════════════════
// Quick Access — user-customizable links
// ═══════════════════════════════════════════════════════════

interface QuickLink {
  id: string
  label: string
  route: string
  icon: Component | undefined
  color: string
  bg: string
  enabled: boolean
}

const ALL_QUICK_LINKS: Omit<QuickLink, 'enabled'>[] = [
  { id: 'rules',      label: '', route: '/alert/rules',           icon: DocumentTextOutline,    color: '#3B82F6', bg: 'rgba(59,130,246,0.08)' },
  { id: 'schedule',   label: '', route: '/oncall/schedule',       icon: CalendarOutline,        color: '#F59E0B', bg: 'rgba(245,158,11,0.08)' },
  { id: 'explore',    label: '', route: '/alert/explore',         icon: SearchOutline,          color: '#06B6D4', bg: 'rgba(6,182,212,0.08)' },
  { id: 'dashboards', label: '', route: '/alert/dashboards',      icon: StatsChartOutline,      color: '#8B5CF6', bg: 'rgba(139,92,246,0.08)' },
  { id: 'notify',     label: '', route: '/oncall/notify/policies', icon: NotificationsOutline,   color: '#EF4444', bg: 'rgba(239,68,68,0.08)' },
  { id: 'suppression',label: '', route: '/alert/suppression',     icon: ShieldCheckmarkOutline, color: '#10B981', bg: 'rgba(16,185,129,0.08)' },
  { id: 'datasources',label: '', route: '/alert/datasources',     icon: PulseOutline,           color: '#0D9488', bg: 'rgba(13,148,136,0.08)' },
  { id: 'incidents',  label: '', route: '/oncall/incidents',      icon: BugOutline,             color: '#EC4899', bg: 'rgba(236,72,153,0.08)' },
  { id: 'spaces',     label: '', route: '/oncall/spaces',         icon: RocketOutline,          color: '#14B8A6', bg: 'rgba(20,184,166,0.08)' },
  { id: 'members',    label: '', route: '/platform/org/members',  icon: PeopleOutline,          color: '#6366F1', bg: 'rgba(99,102,241,0.08)' },
  { id: 'presets',    label: '', route: '/alert/presets',         icon: LibraryOutline,          color: '#0EA5E9', bg: 'rgba(14,165,233,0.08)' },
  { id: 'aiSettings', label: '', route: '/platform/ai-config', icon: SparklesOutline, color: '#D946EF', bg: 'rgba(217,70,239,0.08)' },
]

function loadQuickLinks(): QuickLink[] {
  try {
    const raw = localStorage.getItem(QUICK_KEY)
    if (raw) {
      const saved = JSON.parse(raw) as { id: string; enabled: boolean }[]
      return ALL_QUICK_LINKS.map(link => {
        const s = saved.find(s => s.id === link.id)
        return { ...link, enabled: s ? s.enabled : true }
      })
    }
  } catch { /* ignore */ }
  return ALL_QUICK_LINKS.map(link => ({ ...link, enabled: true }))
}

function saveQuickLinks() {
  localStorage.setItem(QUICK_KEY, JSON.stringify(
    quickLinks.value.map(l => ({ id: l.id, enabled: l.enabled }))
  ))
}

const quickLinks = ref<QuickLink[]>(loadQuickLinks())

// Apply i18n labels
const quickLinkLabels = computed(() => ({
  rules: t('menu.alertRules'),
  schedule: t('menu.schedule'),
  explore: t('menu.explore'),
  dashboards: t('menu.dashboards'),
  notify: t('menu.notifyPolicies'),
  suppression: t('menu.suppression'),
  datasources: t('menu.datasources'),
  incidents: t('menu.incidents'),
  spaces: t('menu.channels'),
  members: t('menu.members'),
  presets: t('menu.presetRules'),
  aiSettings: t('menu.aiConfig'),
}))

watch(quickLinkLabels, (labels) => {
  for (const l of quickLinks.value) {
    l.label = labels[l.id as keyof typeof labels] || l.id
  }
}, { immediate: true })

const enabledQuickLinks = computed(() => quickLinks.value.filter(l => l.enabled))

function toggleQuickLink(id: string) {
  const l = quickLinks.value.find(l => l.id === id)
  if (l) { l.enabled = !l.enabled; saveQuickLinks() }
}

function resetQuickLinks() {
  dialog.warning({
    title: t('common.confirm'),
    content: t('homepage.confirmResetLinks'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => {
      quickLinks.value = ALL_QUICK_LINKS.map(link => ({ ...link, enabled: true }))
      saveQuickLinks()
    },
  })
}

// ═══════════════════════════════════════════════════════════
// Pinned Items — user-created bookmarks
// ═══════════════════════════════════════════════════════════

interface PinnedItem {
  id: string
  title: string
  url: string
  color: string
}

const PIN_COLORS = ['#3B82F6', '#F59E0B', '#EF4444', '#10B981', '#8B5CF6', '#EC4899', '#0D9488', '#06B6D4']

function loadPinned(): PinnedItem[] {
  try {
    const raw = localStorage.getItem(PINNED_KEY)
    if (raw) return JSON.parse(raw)
  } catch { /* ignore */ }
  return [
    { id: 'p1', title: 'Prometheus', url: '/alert/explore', color: '#3B82F6' },
    { id: 'p2', title: 'Grafana', url: '/alert/dashboards', color: '#F59E0B' },
  ]
}

function savePinned() {
  localStorage.setItem(PINNED_KEY, JSON.stringify(pinnedItems.value))
}

const pinnedItems = ref<PinnedItem[]>(loadPinned())
const editingPin = ref<string | null>(null)
const pinForm = reactive({ title: '', url: '', color: PIN_COLORS[0] })

function addPin() {
  const id = 'p' + Date.now()
  pinnedItems.value.push({ id, title: pinForm.title, url: pinForm.url, color: pinForm.color })
  savePinned()
  pinForm.title = ''
  pinForm.url = ''
  pinForm.color = PIN_COLORS[0]
}

function removePin(id: string) {
  pinnedItems.value = pinnedItems.value.filter(p => p.id !== id)
  savePinned()
}

function startEditPin(pin: PinnedItem) {
  editingPin.value = pin.id
  pinForm.title = pin.title
  pinForm.url = pin.url
  pinForm.color = pin.color
}

function saveEditPin() {
  const pin = pinnedItems.value.find(p => p.id === editingPin.value)
  if (pin) {
    pin.title = pinForm.title
    pin.url = pinForm.url
    pin.color = pinForm.color
    savePinned()
  }
  editingPin.value = null
}

function resetPinned() {
  dialog.warning({
    title: t('common.confirm'),
    content: t('homepage.confirmResetPinned'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => {
      pinnedItems.value = [
        { id: 'p1', title: 'Prometheus', url: '/alert/explore', color: '#3B82F6' },
        { id: 'p2', title: 'Grafana', url: '/alert/dashboards', color: '#F59E0B' },
      ]
      savePinned()
    },
  })
}

// ═══════════════════════════════════════════════════════════
// Data Loading
// ═══════════════════════════════════════════════════════════

const loading = ref(true)
const engineOk = ref(false)
const engineUptime = ref('')
const engineIsLeader = ref(true) // default true for single-instance mode
const activeIncidents = ref<Incident[]>([])
const recentIncidents = ref<Incident[]>([])
const firingAlerts = ref<AlertGroupItem[]>([])
const totalRules = ref(0)
const activeAlerts = ref(0)
const oncallUsers = ref<{ scheduleName: string; userName: string }[]>([])
const aiAgentEnabled = ref(false)
const agentConversationCount = ref(0)

const userName = computed(() => authStore.user?.username || authStore.user?.email || 'SRE')

const greeting = computed(() => {
  const hour = new Date().getHours()
  const key = hour < 12 ? 'homepage.greetingMorning' : hour < 18 ? 'homepage.greetingAfternoon' : 'homepage.greetingEvening'
  return t(key, { name: userName.value })
})

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
    status: aiAgentEnabled.value ? 'ok' : 'coming',
    statusText: aiAgentEnabled.value
      ? (agentConversationCount.value > 0
        ? t('homepage.aiConversations', { count: agentConversationCount.value })
        : t('homepage.aiAvailable'))
      : t('homepage.aiDisabled'),
    route: '/ai/agent',
    color: '#8B5CF6',
    bgColor: 'rgba(139,92,246,0.08)',
  },
])

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

async function load() {
  loading.value = true
  try {
    const [engineRes, statsRes, incRes, alertRes, schedRes, aiModRes, agentConvRes] = await Promise.allSettled([
      engineApi.getStatus(),
      dashboardApi.getStats(),
      incidentApi.list({ status: 'active', page_size: 5 }),
      alertGroupsApi.list({ status: 'firing' }),
      scheduleApi.list({ page: 1, page_size: 20 }),
      aiModuleApi.getModules(),
      aiAgentApi.listConversations(1, 1),
    ])
    if (engineRes.status === 'fulfilled') {
      const d = engineRes.value.data.data
      engineOk.value = d.running
      engineUptime.value = d.uptime || ''
      engineIsLeader.value = d.is_leader !== false
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
    // Load on-call users for each schedule (parallel)
    if (schedRes.status === 'fulfilled') {
      const schedules: Schedule[] = schedRes.value.data.data?.list || schedRes.value.data.data || []
      const oncallResults = await Promise.allSettled(
        schedules.slice(0, 5).map(async (s) => {
          const res = await scheduleApi.getCurrentOnCall(s.id)
          const user = res.data.data
          if (user) {
            return { scheduleName: s.name, userName: user.username || user.email || `User #${user.id}` }
          }
          return null
        })
      )
      oncallUsers.value = oncallResults
        .filter((r): r is PromiseFulfilledResult<{ scheduleName: string; userName: string } | null> => r.status === 'fulfilled')
        .map(r => r.value)
        .filter((v): v is { scheduleName: string; userName: string } => v !== null)
    }
    if (aiModRes.status === 'fulfilled') {
      aiAgentEnabled.value = !!aiModRes.value.data.data?.agent?.enabled
    }
    if (agentConvRes.status === 'fulfilled') {
      agentConversationCount.value = agentConvRes.value.data.data?.total || 0
    }
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('homepage.loadFailed'))
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
              <!-- Tabs -->
              <div class="sp-tabs">
                <button class="sp-tab" :class="{ active: settingsTab === 'widgets' }" @click="settingsTab = 'widgets'">{{ t('homepage.tabWidgets') }}</button>
                <button class="sp-tab" :class="{ active: settingsTab === 'quicklinks' }" @click="settingsTab = 'quicklinks'">{{ t('homepage.tabQuickLinks') }}</button>
                <button class="sp-tab" :class="{ active: settingsTab === 'pinned' }" @click="settingsTab = 'pinned'">{{ t('homepage.tabPinned') }}</button>
              </div>

              <!-- Widgets tab -->
              <div v-if="settingsTab === 'widgets'">
                <div class="sp-subheader">
                  <button class="sp-reset" @click="resetWidgets">{{ t('homepage.resetLayout') }}</button>
                </div>
                <div class="sp-list">
                  <div v-for="w in widgets" :key="w.id" class="sp-item" :class="{ 'sp-item--off': !w.enabled }">
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

              <!-- Quick links tab -->
              <div v-if="settingsTab === 'quicklinks'">
                <div class="sp-subheader">
                  <button class="sp-reset" @click="resetQuickLinks">{{ t('homepage.resetLinks') }}</button>
                </div>
                <div class="sp-list">
                  <div v-for="l in quickLinks" :key="l.id" class="sp-item" :class="{ 'sp-item--off': !l.enabled }">
                    <div class="sp-item-left">
                      <div class="sp-link-dot" :style="{ background: l.color }"></div>
                      <span class="sp-item-label">{{ l.label }}</span>
                    </div>
                    <button class="sp-toggle" @click="toggleQuickLink(l.id)">
                      <n-icon :component="l.enabled ? EyeOutline : EyeOffOutline" :size="14" />
                    </button>
                  </div>
                </div>
              </div>

              <!-- Pinned items tab -->
              <div v-if="settingsTab === 'pinned'">
                <div class="sp-subheader">
                  <button class="sp-reset" @click="resetPinned">{{ t('homepage.resetPinned') }}</button>
                </div>
                <div class="sp-list">
                  <div v-for="pin in pinnedItems" :key="pin.id" class="sp-item">
                    <div class="sp-item-left">
                      <div class="sp-link-dot" :style="{ background: pin.color }"></div>
                      <span v-if="editingPin !== pin.id" class="sp-item-label">{{ pin.title }}</span>
                      <template v-else>
                        <input v-model="pinForm.title" class="sp-inline-input" :placeholder="t('homepage.pinTitle')" />
                        <input v-model="pinForm.url" class="sp-inline-input sp-inline-input--url" :placeholder="t('homepage.pinUrl')" />
                      </template>
                    </div>
                    <div class="sp-item-actions">
                      <template v-if="editingPin === pin.id">
                        <button class="sp-save" @click="saveEditPin">{{ t('common.save') }}</button>
                        <button class="sp-cancel" @click="editingPin = null">{{ t('common.cancel') }}</button>
                      </template>
                      <template v-else>
                        <button class="sp-toggle" @click="startEditPin(pin)"><n-icon :component="CreateOutline" :size="13" /></button>
                        <button class="sp-toggle sp-toggle--danger" @click="removePin(pin.id)"><n-icon :component="CloseOutline" :size="13" /></button>
                      </template>
                    </div>
                  </div>
                  <!-- Add new pin -->
                  <div class="sp-add-row">
                    <input v-model="pinForm.title" class="sp-inline-input" :placeholder="t('homepage.pinTitle')" />
                    <input v-model="pinForm.url" class="sp-inline-input sp-inline-input--url" :placeholder="t('homepage.pinUrl')" />
                    <div class="sp-color-pick">
                      <button v-for="c in PIN_COLORS" :key="c" class="sp-color-dot" :class="{ active: pinForm.color === c }" :style="{ background: c }" @click="pinForm.color = c"></button>
                    </div>
                    <button class="sp-add-btn" @click="addPin" :disabled="!pinForm.title">
                      <n-icon :component="AddOutline" :size="14" />
                      {{ t('homepage.addPin') }}
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
              <div class="greeting-bar">
                <h1 class="gc-hello">{{ greeting }}</h1>
                <span v-if="engineOk" class="gc-badge gc-badge--ok">
                  <span class="dot dot-ok"></span>
                  {{ t('homepage.engineRunning', { days: uptimeDays() }) }}
                </span>
                <span v-else class="gc-badge gc-badge--down">
                  <span class="dot dot-critical"></span>
                  {{ t('homepage.engineDown') }}
                </span>
                <span v-if="engineOk && !engineIsLeader" class="gc-badge gc-badge--standby">
                  <span class="dot dot-standby"></span>
                  {{ t('homepage.engineStandby') }}
                </span>
              </div>
            </div>

            <!-- ═══ Module Status ═══ -->
            <div v-else-if="widget.type === 'moduleStatus'" class="bento-item bento-full">
              <div class="module-strip">
                <div
                  v-for="mod in modules"
                  :key="mod.key"
                  class="mod-card"
                  role="button"
                  tabindex="0"
                  :class="{ 'mod-card--clickable': !!mod.route, 'mod-card--coming': mod.status === 'coming' }"
                  @click="mod.route && router.push(mod.route)"
                  @keydown.enter="mod.route && router.push(mod.route)"
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
                    <div v-for="inc in activeIncidents" :key="inc.id" class="task-row" role="button" tabindex="0" @click="router.push(`/oncall/incidents/${inc.id}`)" @keydown.enter="router.push(`/oncall/incidents/${inc.id}`)">
                      <span class="task-sev" :style="{ background: sevColor(inc.severity) }">{{ inc.severity?.toUpperCase() }}</span>
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
                  <div v-else class="wc-empty"><span class="wc-empty-icon">✓</span><span>{{ t('homepage.noTasks') }}</span></div>
                </div>
              </div>
            </div>

            <!-- ═══ On-Call Schedule ═══ -->
            <div v-else-if="widget.type === 'oncallSchedule'" class="bento-item bento-lg">
              <div class="widget-card">
                <div class="wc-header">
                  <n-icon :component="PeopleOutline" :size="16" class="wc-icon" />
                  <span class="wc-title">{{ t('homepage.oncallSchedule') }}</span>
                </div>
                <div class="wc-body">
                  <div v-if="oncallUsers.length" class="oncall-list">
                    <div v-for="item in oncallUsers" :key="item.scheduleName" class="oncall-row" role="button" tabindex="0" @click="router.push('/oncall/schedule')" @keydown.enter="router.push('/oncall/schedule')">
                      <div class="oncall-avatar">{{ item.userName.charAt(0).toUpperCase() }}</div>
                      <div class="oncall-info">
                        <span class="oncall-name">{{ item.userName }}</span>
                        <span class="oncall-sched">{{ item.scheduleName }}</span>
                      </div>
                      <span class="oncall-badge">{{ t('homepage.onDuty') }}</span>
                    </div>
                  </div>
                  <div v-else class="wc-empty"><span class="wc-empty-icon">—</span><span>{{ t('homepage.noOnCall') }}</span></div>
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
                    <div v-for="(item, idx) in activity" :key="idx" class="act-row" role="button" tabindex="0" @click="router.push(item.route)" @keydown.enter="router.push(item.route)">
                      <div class="act-time"><span class="act-dot" :style="{ background: sevColor(item.severity) }"></span>{{ relTime(item.time) }}</div>
                      <span class="act-text">{{ item.text }}</span>
                    </div>
                  </div>
                  <div v-else class="wc-empty"><span class="wc-empty-icon">✓</span><span>{{ t('homepage.noActivity') }}</span></div>
                </div>
              </div>
            </div>

            <!-- ═══ Pinned Items ═══ -->
            <div v-else-if="widget.type === 'pinnedItems'" class="bento-item bento-lg">
              <div class="widget-card">
                <div class="wc-header">
                  <n-icon :component="BookmarkOutline" :size="16" class="wc-icon" />
                  <span class="wc-title">{{ t('homepage.pinnedItems') }}</span>
                </div>
                <div class="wc-body">
                  <div v-if="pinnedItems.length" class="pinned-grid">
                    <div
                      v-for="pin in pinnedItems"
                      :key="pin.id"
                      class="pinned-card"
                      role="button"
                      tabindex="0"
                      @click="pin.url && router.push(pin.url)"
                      @keydown.enter="pin.url && router.push(pin.url)"
                    >
                      <div class="pinned-icon" :style="{ background: pin.color + '18', color: pin.color }">
                        {{ pin.title.charAt(0).toUpperCase() }}
                      </div>
                      <span class="pinned-title">{{ pin.title }}</span>
                    </div>
                  </div>
                  <div v-else class="wc-empty">
                    <span class="wc-empty-icon">+</span>
                    <span>{{ t('homepage.noPinned') }}</span>
                    <span class="wc-empty-hint">{{ t('homepage.addViaSettings') }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- ═══ Quick Access ═══ -->
            <div v-else-if="widget.type === 'quickAccess'" class="bento-item bento-full">
              <div class="quick-grid">
                <div v-for="link in enabledQuickLinks" :key="link.id" class="quick-btn" role="button" tabindex="0" @click="router.push(link.route)" @keydown.enter="router.push(link.route)">
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
.homepage { min-height: 100vh; font-family: var(--sre-font-sans); }
.hp-container { max-width: 1400px; margin: 0 auto; padding: 24px 32px 48px; position: relative; }

/* ===== Settings Bar ===== */
.hp-settings-bar { display: flex; justify-content: flex-end; margin-bottom: 16px; }
.hp-settings-btn {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 6px 14px; border: 1px solid var(--sre-border); border-radius: var(--sre-radius-pill);
  background: var(--sre-bg-card); color: var(--sre-text-secondary);
  font-size: 12px; font-weight: 500; cursor: pointer;
  transition: all 150ms var(--sre-ease-out); font-family: var(--sre-font-sans);
}
.hp-settings-btn:hover, .hp-settings-btn.active { background: var(--sre-primary-soft); border-color: var(--sre-primary-ring); color: var(--sre-primary); }

/* ===== Settings Panel ===== */
.settings-panel { min-width: 320px; max-width: 400px; padding: 0; }
.sp-tabs { display: flex; border-bottom: 1px solid var(--sre-border); }
.sp-tab {
  flex: 1; padding: 10px 8px; border: none; background: none;
  font-size: 12px; font-weight: 500; color: var(--sre-text-muted); cursor: pointer;
  font-family: var(--sre-font-sans); transition: color 120ms, border-color 120ms;
  border-bottom: 2px solid transparent;
}
.sp-tab:hover { color: var(--sre-text-primary); }
.sp-tab.active { color: var(--sre-primary); border-bottom-color: var(--sre-primary); }
.sp-subheader { display: flex; justify-content: flex-end; padding: 8px 16px 0; }
.sp-reset {
  font-size: 11px; color: var(--sre-primary); background: none; border: none;
  cursor: pointer; font-family: var(--sre-font-sans); font-weight: 500;
}
.sp-list { padding: 8px 0; }
.sp-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 16px; transition: background 100ms; gap: 8px;
}
.sp-item:hover { background: var(--sre-bg-hover); }
.sp-item--off { opacity: 0.5; }
.sp-item-left { display: flex; align-items: center; gap: 8px; flex: 1; min-width: 0; }
.sp-item-icon { color: var(--sre-text-muted); }
.sp-item-label { font-size: 13px; color: var(--sre-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.sp-item-actions { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
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
.sp-toggle--danger:hover { color: var(--sre-critical); }
.sp-link-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.sp-save {
  font-size: 11px; font-weight: 600; color: var(--sre-primary); background: var(--sre-primary-soft);
  border: none; border-radius: 4px; padding: 3px 10px; cursor: pointer; font-family: var(--sre-font-sans);
}
.sp-cancel {
  font-size: 11px; color: var(--sre-text-muted); background: none;
  border: none; cursor: pointer; font-family: var(--sre-font-sans);
}
.sp-inline-input {
  font-size: 12px; padding: 3px 8px; border: 1px solid var(--sre-border); border-radius: 4px;
  background: var(--sre-bg-page); color: var(--sre-text-primary); font-family: var(--sre-font-sans);
  width: 100px;
}
.sp-inline-input--url { width: 140px; }
.sp-add-row {
  display: flex; flex-wrap: wrap; align-items: center; gap: 8px;
  padding: 12px 16px; border-top: 1px solid var(--sre-border);
}
.sp-color-pick { display: flex; gap: 4px; }
.sp-color-dot {
  width: 16px; height: 16px; border-radius: 50%; border: 2px solid transparent;
  cursor: pointer; transition: border-color 100ms;
}
.sp-color-dot.active { border-color: var(--sre-text-primary); }
.sp-add-btn {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 11px; font-weight: 600; color: var(--sre-primary);
  background: var(--sre-primary-soft); border: none; border-radius: 4px;
  padding: 5px 12px; cursor: pointer; font-family: var(--sre-font-sans);
}
.sp-add-btn:disabled { opacity: 0.4; cursor: default; }

/* ===== Bento Grid ===== */
.bento-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px; }
.bento-full { grid-column: 1 / -1; }
.bento-lg { grid-column: span 1; }

/* ===== Greeting Bar ===== */
.greeting-bar { display: flex; align-items: center; gap: 16px; padding: 4px 0; }
.gc-hello {
  font-family: var(--sre-font-display); font-size: 20px; font-weight: 600;
  color: var(--sre-text-primary); margin: 0; line-height: 1.3; letter-spacing: -0.01em;
}
.gc-badge {
  display: inline-flex; align-items: center; gap: 6px;
  font-size: 12px; font-weight: 500; padding: 4px 12px;
  border-radius: var(--sre-radius-pill); white-space: nowrap;
}
.gc-badge--ok { color: var(--sre-success); background: rgba(34,197,94,0.08); }
.gc-badge--ok .dot { background: var(--sre-success); box-shadow: 0 0 0 3px rgba(34,197,94,0.12); }
.gc-badge--down { color: var(--sre-critical); background: rgba(239,68,68,0.08); }
.gc-badge--down .dot { background: var(--sre-critical); box-shadow: 0 0 0 3px rgba(239,68,68,0.12); }
.gc-badge--standby { color: var(--sre-warning); background: rgba(234,179,8,0.08); }
.gc-badge--standby .dot { background: var(--sre-warning); box-shadow: 0 0 0 3px rgba(234,179,8,0.12); }
.dot-standby { background: var(--sre-warning); }

/* ===== Module Strip ===== */
.module-strip { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
.mod-card {
  background: var(--sre-bg-card); border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-lg, 12px); padding: 16px;
  display: flex; flex-direction: column; gap: 10px;
  transition: border-color 200ms var(--sre-ease-out), box-shadow 200ms var(--sre-ease-out);
}
.mod-card--clickable { cursor: pointer; }
.mod-card--clickable:hover { border-color: var(--sre-border-strong); box-shadow: var(--sre-shadow-sm); }
.mod-card--coming { opacity: 0.55; }
.mod-icon {
  width: 36px; height: 36px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.mod-info { min-width: 0; }
.mod-label { font-size: 14px; font-weight: 600; color: var(--sre-text-primary); font-family: var(--sre-font-display); }
.mod-desc { font-size: 12px; color: var(--sre-text-tertiary); margin-top: 2px; }
.mod-status { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--sre-text-secondary); margin-top: auto; }
.mod-status-text { font-weight: 500; }

/* ===== Dots ===== */
.dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; display: inline-block; }
.dot-ok { background: var(--sre-success); box-shadow: 0 0 0 3px rgba(34,197,94,0.15); }
.dot-warning { background: var(--sre-warning); box-shadow: 0 0 0 3px rgba(245,158,11,0.15); }
.dot-critical { background: var(--sre-critical); box-shadow: 0 0 0 3px rgba(239,68,68,0.15); }
.dot-coming { background: var(--sre-text-muted); }

/* ===== Widget Cards ===== */
.widget-card {
  background: var(--sre-bg-card); border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-lg, 12px); overflow: hidden;
  height: 100%; display: flex; flex-direction: column;
}
.wc-header {
  display: flex; align-items: center; gap: 8px;
  padding: 14px 18px; border-bottom: 1px solid var(--sre-border);
}
.wc-icon { color: var(--sre-primary); }
.wc-title { font-size: 14px; font-weight: 600; color: var(--sre-text-primary); font-family: var(--sre-font-display); }
.wc-badge { font-size: 11px; font-weight: 700; color: white; background: var(--sre-critical); padding: 1px 7px; border-radius: 10px; margin-left: auto; }
.wc-body { flex: 1; overflow-y: auto; }

/* ===== Tasks ===== */
.tasks-list { display: flex; flex-direction: column; }
.task-row {
  display: flex; align-items: center; gap: 12px; padding: 12px 18px; cursor: pointer;
  transition: background 120ms var(--sre-ease-out); border-bottom: 1px solid var(--sre-border);
}
.task-row:last-child { border-bottom: none; }
.task-row:hover { background: var(--sre-bg-hover); }
.task-sev { font-size: 9px; font-weight: 700; color: var(--sre-text-inverse); padding: 2px 6px; border-radius: 4px; letter-spacing: 0.04em; flex-shrink: 0; }
.task-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.task-title { font-size: 13px; font-weight: 500; color: var(--sre-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.task-meta { font-size: 11px; color: var(--sre-text-tertiary); }
.task-arrow { color: var(--sre-text-muted); flex-shrink: 0; }

/* ===== On-Call Schedule ===== */
.oncall-list { display: flex; flex-direction: column; }
.oncall-row {
  display: flex; align-items: center; gap: 12px; padding: 12px 18px; cursor: pointer;
  transition: background 120ms var(--sre-ease-out); border-bottom: 1px solid var(--sre-border);
}
.oncall-row:last-child { border-bottom: none; }
.oncall-row:hover { background: var(--sre-bg-hover); }
.oncall-avatar {
  width: 32px; height: 32px; border-radius: 50%;
  background: var(--sre-primary-soft); color: var(--sre-primary);
  display: flex; align-items: center; justify-content: center;
  font-size: 13px; font-weight: 700; flex-shrink: 0;
}
.oncall-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
.oncall-name { font-size: 13px; font-weight: 500; color: var(--sre-text-primary); }
.oncall-sched { font-size: 11px; color: var(--sre-text-tertiary); }
.oncall-badge {
  font-size: 10px; font-weight: 600; color: var(--sre-success); background: rgba(34,197,94,0.08);
  padding: 2px 8px; border-radius: 4px; flex-shrink: 0;
}

/* ===== Activity ===== */
.activity-list { display: flex; flex-direction: column; }
.act-row {
  display: flex; flex-direction: column; gap: 4px; padding: 10px 18px; cursor: pointer;
  transition: background 120ms var(--sre-ease-out); border-bottom: 1px solid var(--sre-border);
}
.act-row:last-child { border-bottom: none; }
.act-row:hover { background: var(--sre-bg-hover); }
.act-time { display: flex; align-items: center; gap: 6px; font-size: 11px; color: var(--sre-text-muted); white-space: nowrap; }
.act-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.act-text { font-size: 13px; color: var(--sre-text-secondary); line-height: 1.4; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* ===== Pinned Items ===== */
.pinned-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(100px, 1fr)); gap: 10px; padding: 14px 18px; }
.pinned-card {
  display: flex; flex-direction: column; align-items: center; gap: 8px;
  padding: 14px 8px; border-radius: 10px; cursor: pointer;
  transition: background 120ms var(--sre-ease-out);
}
.pinned-card:hover { background: var(--sre-bg-hover); }
.pinned-icon {
  width: 36px; height: 36px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  font-size: 15px; font-weight: 700;
}
.pinned-title { font-size: 12px; font-weight: 500; color: var(--sre-text-secondary); text-align: center; }

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
.wc-empty-hint { font-size: 11px; color: var(--sre-text-muted); }

/* ===== Quick Access ===== */
.quick-grid { display: grid; grid-template-columns: repeat(6, 1fr); gap: 12px; }
.quick-btn {
  display: flex; flex-direction: column; align-items: center; gap: 10px;
  padding: 20px 12px; background: var(--sre-bg-card);
  border: 1px solid var(--sre-border); border-radius: var(--sre-radius-lg, 12px);
  cursor: pointer;
  transition: border-color 200ms var(--sre-ease-out), box-shadow 200ms var(--sre-ease-out), transform 150ms var(--sre-ease-out);
}
.quick-btn:hover { border-color: var(--sre-border-strong); box-shadow: var(--sre-shadow-sm); transform: translateY(-2px); }
.quick-icon {
  width: 40px; height: 40px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center;
}
.quick-label { font-size: 13px; font-weight: 500; color: var(--sre-text-secondary); text-align: center; }

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
  .greeting-bar { flex-direction: column; align-items: flex-start; gap: 8px; }
  .gc-hello { font-size: 18px; }
  .pinned-grid { grid-template-columns: repeat(3, 1fr); }
}
@media (prefers-reduced-motion: reduce) {
  .mod-card, .quick-btn, .task-row, .act-row, .oncall-row, .pinned-card { transition: none; }
}
</style>
