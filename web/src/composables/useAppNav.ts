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

// ===== Public Types =====

export type AppKey = 'home' | 'oncall' | 'alert' | 'platform'

export interface MenuItem {
  label: string
  key: string
  icon?: Component
  children?: MenuItem[]
  show?: boolean
}

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
    path.startsWith('/alerts-v2') ||
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
  if (path.startsWith('/settings')) return 'platform'

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
              { label: t('menu.overview'), key: '/oncall/overview' },
            ],
          },
          {
            label: t('menu.channels'),
            items: [
              { label: t('menu.channels'),       key: '/oncall/spaces' },
              { label: t('menu.incidents'),       key: '/oncall/incidents' },
              { label: t('menu.statusPage'),      key: '/oncall/status-page' },
              { label: t('menu.postmortems'),     key: '/oncall/postmortems' },
            ],
          },
          {
            items: (() => {
              const items: MenuItem[] = []
              if (authStore.canManage) {
                items.push(
                  { label: t('menu.integrations'), key: '/oncall/integrations' },
                  { label: t('menu.schedule'),     key: '/oncall/schedule' },
                )
              }
              return items
            })(),
          },
          {
            label: t('menu.configCenter'),
            items: [
              { label: t('menu.notifyChannels'),  key: '/oncall/config/notify-rules' },
              { label: t('menu.routingRules'),     key: '/oncall/config/routing-rules' },
              { label: t('menu.bizGroups'),        key: '/oncall/config/biz-groups' },
              { label: t('menu.subscriptions'),    key: '/oncall/config/subscribe-rules' },
            ],
          },
        ]

      // ──────────────── ALERT ────────────────
      case 'alert':
        return [
          {
            items: [
              { label: t('menu.overview'), key: '/alert/overview' },
            ],
          },
          {
            label: t('menu.alerts'),
            items: [
              { label: t('menu.alertRules'),      key: '/alert/rules' },
              { label: t('menu.activeAlerts'),     key: '/alert/events' },
              { label: t('menu.alertHistory'),     key: '/alert/history' },
              { label: t('menu.muteRules'),        key: '/alert/suppression' },
            ],
          },
          {
            label: t('menu.data'),
            items: [
              { label: t('menu.datasources'), key: '/alert/datasources' },
              { label: t('menu.dataQuery'),   key: '/alert/explore' },
              { label: t('menu.dashboard'),   key: '/alert/dashboards' },
            ],
          },
          {
            label: t('menu.notification'),
            items: [
              { label: t('menu.notifyPolicies'),  key: '/alert/notify/policies' },
              { label: t('menu.templates'),        key: '/alert/notify/templates' },
              { label: t('menu.notifyChannels'),   key: '/alert/notify/channels' },
              { label: t('menu.subscriptions'),    key: '/alert/notify/subscriptions' },
            ],
          },
        ]

      // ──────────────── PLATFORM ────────────────
      case 'platform':
        return [
          {
            items: [
              { label: t('menu.profile'), key: '/platform/profile' },
            ],
          },
          {
            label: t('menu.orgManagement'),
            items: (() => {
              const items: MenuItem[] = []
              if (authStore.isAdmin)     items.push({ label: t('menu.members'), key: '/platform/org/members' })
              if (authStore.canManage)   items.push({ label: t('menu.teams'), key: '/platform/org/teams' })
              items.push({ label: t('menu.roles'), key: '/platform/org/roles' })
              if (authStore.isAdmin)     items.push({ label: t('menu.sso'), key: '/platform/org/sso' })
              return items
            })(),
          },
          {
            items: (() => {
              const items: MenuItem[] = []
              if (authStore.isAdmin) {
                items.push({ label: t('menu.audit'), key: '/platform/audit' })
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
                  { label: t('menu.smtp'),      key: '/platform/settings/smtp' },
                  { label: t('menu.larkBot'),    key: '/platform/settings/lark' },
                  { label: t('menu.aiConfig'),   key: '/platform/settings/ai' },
                  { label: t('menu.security'),   key: '/platform/settings/security' },
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
