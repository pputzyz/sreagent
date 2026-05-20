/**
 * useAppNav — Centralized navigation state for the 3-app layout (oncall / alert / platform).
 *
 * This composable is the single source of truth for:
 *   - Which "app" is active based on the current route
 *   - The sidebar menu sections for each app
 *   - The currently highlighted menu key (longest prefix match)
 *   - The page title displayed in the main header
 */
import { ref, computed, watch } from 'vue'
import type { Ref, Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import {
  // ONCALL
  HomeOutline,
  ChatbubblesOutline,
  AlertCircleOutline,
  GlobeOutline,
  DocumentTextOutline,
  LinkOutline,
  CalendarOutline,
  NotificationsOutline,
  GitBranchOutline,
  FolderOpenOutline,
  MailOutline,
  SwapVerticalOutline,
  // ALERT
  StatsChartOutline,
  ListOutline,
  LibraryOutline,
  FlashOutline,
  TimeOutline,
  VolumeMuteOutline,
  ServerOutline,
  SearchOutline,
  PieChartOutline,
  CopyOutline,
  SendOutline,
  // PLATFORM
  PersonOutline,
  PeopleOutline,
  PeopleCircleOutline,
  ShieldCheckmarkOutline,
  KeyOutline,
  EyeOutline,
  ChatbubbleEllipsesOutline,
  HardwareChipOutline,
  SparklesOutline,
  ShieldOutline,
} from '@vicons/ionicons5'

// ===== Public Types =====

export type AppKey = 'home' | 'oncall' | 'alert' | 'platform'

export interface MenuItem {
  label: string
  key: string
  icon?: Component
  iconColor?: string
  children?: MenuItem[]
  show?: boolean
}

// Icon → color mapping for sidebar nav icons
export const iconColorMap = new Map<Component, string>([
  // ONCALL
  [HomeOutline,             '#F59E0B'], // amber — home/warmth
  [ChatbubblesOutline,      '#14B8A6'], // teal — communication
  [AlertCircleOutline,      '#F43F5E'], // rose — incident/danger
  [GlobeOutline,            '#10B981'], // emerald — status/healthy
  [DocumentTextOutline,     '#64748B'], // slate — documentation
  [LinkOutline,             '#8B5CF6'], // violet — connections
  [CalendarOutline,         '#3B82F6'], // blue — time/schedule
  [NotificationsOutline,    '#F59E0B'], // amber — alert bell
  [GitBranchOutline,        '#6366F1'], // indigo — logic/routing
  [FolderOpenOutline,       '#D97706'], // amber-dark — organization
  [MailOutline,             '#0EA5E9'], // sky — mail/subscribe
  // ALERT
  [StatsChartOutline,       '#3B82F6'], // blue — metrics
  [ListOutline,             '#64748B'], // slate — rules list
  [LibraryOutline,          '#6366F1'], // indigo — preset library
  [FlashOutline,            '#EF4444'], // red — active alert
  [TimeOutline,             '#78716C'], // stone — history/archive
  [VolumeMuteOutline,       '#A8A29E'], // neutral — muted
  [ServerOutline,           '#06B6D4'], // cyan — data source
  [SearchOutline,           '#3B82F6'], // blue — explore/query
  [PieChartOutline,         '#8B5CF6'], // violet — dashboard/chart
  [CopyOutline,             '#14B8A6'], // teal — templates
  [SendOutline,             '#0EA5E9'], // sky — delivery/send
  // PLATFORM
  [PersonOutline,           '#3B82F6'], // blue — profile
  [PeopleOutline,           '#10B981'], // emerald — members
  [PeopleCircleOutline,     '#3B82F6'], // blue — teams
  [ShieldCheckmarkOutline,  '#F59E0B'], // amber — roles/permissions
  [KeyOutline,              '#6366F1'], // indigo — auth/SSO
  [EyeOutline,              '#64748B'], // slate — audit/monitor
  [ChatbubbleEllipsesOutline, '#3B82F6'], // blue — lark/chat
  [HardwareChipOutline,     '#8B5CF6'], // violet — AI/hardware
  [SparklesOutline,         '#A855F7'], // purple — AI modules/magic
  [ShieldOutline,           '#EF4444'], // red — security/alert
])

export interface MenuSection {
  label?: string
  items: MenuItem[]
}

// ===== Singleton reactive state =====

const activeApp: Ref<AppKey> = ref('oncall')

// ===== Route → App mapping =====

function resolveAppFromPath(path: string): AppKey {
  // Homepage — platform root
  if (path === '/') return 'home'

  // New 3-app prefixed routes
  if (path.startsWith('/oncall'))   return 'oncall'
  if (path.startsWith('/alert'))    return 'alert'
  if (path.startsWith('/platform')) return 'platform'

  // Legacy routes — oncall
  if (
    path.startsWith('/incident-dashboard') ||
    path.startsWith('/channels') ||
    path.startsWith('/incidents') ||
    path.startsWith('/schedule') ||
    path.startsWith('/integrations')
  ) return 'oncall'

  // Legacy routes — alert
  if (
    path.startsWith('/alerts') ||
    path.startsWith('/datasources') ||
    path.startsWith('/query') ||
    path.startsWith('/dashboards') ||
    path.startsWith('/notification')
  ) return 'alert'

  // Legacy routes — platform
  if (path.startsWith('/settings') || path.startsWith('/ai')) return 'platform'

  // Default
  return 'oncall'
}

// ===== Default route per app =====

const appDefaultRoute: Record<AppKey, string> = {
  home:     '/',
  oncall:   '/oncall/overview',
  alert:    '/alert/overview',
  platform: '/platform/profile',
}

// ===== Composable =====

export function useAppNav() {
  const route = useRoute()
  const router = useRouter()
  const { t } = useI18n()
  const authStore = useAuthStore()

  // ---------- switchApp ----------

  function switchApp(app: AppKey) {
    activeApp.value = app
    // Navigate to the app's default route only if the current route doesn't belong to it
    if (resolveAppFromPath(route.path) !== app) {
      router.push(appDefaultRoute[app])
    }
  }

  // ---------- menuSections ----------

  const menuSections = computed<MenuSection[]>(() => {
    switch (activeApp.value) {
      case 'home':
        return []
      // ──────────────── ONCALL ────────────────
      case 'oncall':
        return [
          {
            items: [
              { label: t('menu.overview'), key: '/oncall/overview', icon: HomeOutline },
              { label: t('myAlerts.title'), key: '/oncall/my-alerts', icon: AlertCircleOutline },
            ],
          },
          {
            label: t('menu.channels'),
            items: [
              { label: t('menu.channels'),       key: '/oncall/spaces', icon: ChatbubblesOutline },
              { label: t('menu.incidents'),       key: '/oncall/incidents', icon: AlertCircleOutline },
              { label: t('menu.statusPage'),      key: '/oncall/status-page', icon: GlobeOutline },
              { label: t('menu.postmortems'),     key: '/oncall/postmortems', icon: DocumentTextOutline },
            ],
          },
          {
            items: (() => {
              const items: MenuItem[] = []
              if (authStore.canManage) {
                items.push(
                  { label: t('menu.integrations'), key: '/oncall/integrations', icon: LinkOutline },
                  { label: t('menu.schedule'),     key: '/oncall/schedule', icon: CalendarOutline },
                )
              }
              return items
            })(),
          },
          {
            label: t('menu.configCenter'),
            items: [
              { label: t('menu.notifyChannels'),  key: '/oncall/config/notify-rules', icon: NotificationsOutline },
              { label: t('menu.routingRules'),     key: '/oncall/config/routing-rules', icon: GitBranchOutline },
              { label: t('menu.bizGroups'),        key: '/oncall/config/biz-groups', icon: FolderOpenOutline },
              { label: t('menu.subscriptions'),    key: '/oncall/config/subscribe-rules', icon: MailOutline },
              { label: t('menu.escalationPolicies'), key: '/oncall/config/escalation-policies', icon: SwapVerticalOutline },
            ],
          },
        ]

      // ──────────────── ALERT ────────────────
      case 'alert':
        return [
          {
            items: [
              { label: t('menu.overview'), key: '/alert/overview', icon: StatsChartOutline },
            ],
          },
          {
            label: t('menu.alerts'),
            items: [
              { label: t('menu.alertRules'),      key: '/alert/rules', icon: ListOutline },
              { label: t('menu.presetRules'),       key: '/alert/presets', icon: LibraryOutline },
              { label: t('menu.activeAlerts'),     key: '/alert/events', icon: FlashOutline },
              { label: t('menu.alertHistory'),     key: '/alert/history', icon: TimeOutline },
              { label: t('menu.muteRules'),        key: '/alert/suppression', icon: VolumeMuteOutline },
            ],
          },
          {
            label: t('menu.data'),
            items: [
              { label: t('menu.datasources'), key: '/alert/datasources', icon: ServerOutline },
              { label: t('menu.dataQuery'),   key: '/alert/explore', icon: SearchOutline },
              { label: t('menu.dashboard'),   key: '/alert/dashboards', icon: PieChartOutline },
            ],
          },
          {
            label: t('menu.notification'),
            items: [
              { label: t('menu.notifyPolicies'),  key: '/alert/notify/policies', icon: NotificationsOutline },
              { label: t('menu.templates'),        key: '/alert/notify/templates', icon: CopyOutline },
              { label: t('menu.notifyChannels'),   key: '/alert/notify/channels', icon: SendOutline },
              { label: t('menu.subscriptions'),    key: '/alert/notify/subscriptions', icon: MailOutline },
            ],
          },
        ]

      // ──────────────── PLATFORM ────────────────
      case 'platform':
        return [
          {
            items: [
              { label: t('menu.profile'), key: '/platform/profile', icon: PersonOutline },
            ],
          },
          {
            label: t('menu.orgManagement'),
            items: (() => {
              const items: MenuItem[] = []
              if (authStore.isAdmin)     items.push({ label: t('menu.members'), key: '/platform/org/members', icon: PeopleOutline })
              if (authStore.canManage)   items.push({ label: t('menu.teams'), key: '/platform/org/teams', icon: PeopleCircleOutline })
              items.push({ label: t('menu.roles'), key: '/platform/org/roles', icon: ShieldCheckmarkOutline })
              if (authStore.isAdmin)     items.push({ label: t('menu.sso'), key: '/platform/org/sso', icon: KeyOutline })
              return items
            })(),
          },
          {
            items: (() => {
              const items: MenuItem[] = []
              if (authStore.isAdmin) {
                items.push({ label: t('menu.audit'), key: '/platform/audit', icon: EyeOutline })
              }
              return items
            })(),
          },
          {
            label: t('menu.systemSettings'),
            items: (() => {
              const items: MenuItem[] = []
              if (authStore.isAdmin) {
                items.push(
                  { label: t('menu.smtp'),      key: '/platform/settings/smtp', icon: MailOutline },
                  { label: t('menu.larkBot'),    key: '/platform/settings/lark', icon: ChatbubbleEllipsesOutline },
                  { label: t('menu.aiConfig'),   key: '/platform/settings/ai', icon: HardwareChipOutline },
                  { label: t('menu.aiModuleConfig'), key: '/platform/settings/ai-settings', icon: SparklesOutline },
                  { label: t('menu.aiAgent'),    key: '/ai/agent', icon: SparklesOutline },
                  { label: t('menu.security'),   key: '/platform/settings/security', icon: ShieldOutline },
                )
              }
              return items
            })(),
          },
        ]
    }
  })

  // ---------- flatMenuOptions ----------

  const flatMenuOptions = computed<MenuItem[]>(() => {
    return menuSections.value.flatMap(s => s.items)
  })

  // ---------- activeMenuKey (longest prefix match) ----------

  const activeMenuKey = computed<string>(() => {
    const p = route.path
    let best = ''
    for (const item of flatMenuOptions.value) {
      const k = item.key
      if (p === k || p.startsWith(k + '/')) {
        if (k.length > best.length) best = k
      }
    }
    return best
  })

  // ---------- pageTitle ----------

  const pageTitle = computed<string>(() => {
    const item = flatMenuOptions.value.find(m => m.key === activeMenuKey.value)
    return item?.label || ''
  })

  // ---------- Watch route changes ----------

  watch(() => route.path, (p) => {
    activeApp.value = resolveAppFromPath(p)
  }, { immediate: true })

  // ---------- Return ----------

  return {
    activeApp,
    switchApp,
    menuSections,
    activeMenuKey,
    pageTitle,
  }
}
