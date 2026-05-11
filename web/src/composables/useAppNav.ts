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

export type AppKey = 'oncall' | 'alert' | 'platform'

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
      // ──────────────── ONCALL ────────────────
      case 'oncall':
        return [
          {
            items: [
              { label: t('menu.overview'), key: '/oncall/overview' },
            ],
          },
          {
            label: '协作空间',
            items: [
              { label: t('menu.channels'),       key: '/oncall/channels' },
              { label: t('menu.incidents'),       key: '/oncall/incidents' },
              { label: '状态页面',                key: '/oncall/status-page' },
              { label: '故障复盘',                key: '/oncall/postmortem' },
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
            label: '配置中心',
            items: [
              { label: t('menu.notifyChannels'),  key: '/oncall/notify-rules' },
              { label: '路由规则',                 key: '/oncall/routing-rules' },
              { label: '业务分组',                 key: '/oncall/biz-groups' },
              { label: t('menu.subscriptions'),    key: '/oncall/subscriptions' },
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
              { label: t('menu.muteRules'),        key: '/alert/mute-rules' },
            ],
          },
          {
            label: t('menu.data'),
            items: [
              { label: t('menu.datasources'), key: '/alert/datasources' },
              { label: t('menu.dataQuery'),   key: '/alert/query' },
              { label: '仪表盘',              key: '/alert/dashboards' },
            ],
          },
          {
            label: t('menu.notification'),
            items: [
              { label: t('menu.notifyPolicies'),  key: '/alert/notify-policies' },
              { label: t('menu.templates'),        key: '/alert/templates' },
              { label: t('menu.notifyChannels'),   key: '/alert/channels' },
              { label: t('menu.subscriptions'),    key: '/alert/subscriptions' },
            ],
          },
        ]

      // ──────────────── PLATFORM ────────────────
      case 'platform':
        return [
          {
            items: [
              { label: '个人中心', key: '/platform/profile' },
            ],
          },
          {
            label: '组织管理',
            items: (() => {
              const items: MenuItem[] = []
              if (authStore.isAdmin)     items.push({ label: '成员管理', key: '/platform/users' })
              if (authStore.canManage)   items.push({ label: '团队管理', key: '/platform/teams' })
              items.push({ label: '角色权限', key: '/platform/roles' })
              if (authStore.isAdmin)     items.push({ label: '单点登录', key: '/platform/oidc' })
              return items
            })(),
          },
          {
            items: (() => {
              const items: MenuItem[] = []
              if (authStore.isAdmin) {
                items.push({ label: '审计日志', key: '/platform/audit-log' })
              }
              return items
            })(),
          },
          {
            label: '系统设置',
            items: (() => {
              const items: MenuItem[] = []
              if (authStore.isAdmin) {
                items.push(
                  { label: '邮件服务',  key: '/platform/smtp' },
                  { label: '飞书机器人', key: '/platform/lark-bot' },
                  { label: 'AI 配置',    key: '/platform/ai-config' },
                  { label: '安全设置',   key: '/platform/security' },
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
