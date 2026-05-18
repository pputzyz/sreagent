import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw, RouteLocation } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/pages/Login.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/',
    component: () => import('@/layouts/AppShell.vue'),
    meta: { requiresAuth: true },
    children: [
      // Root — Unified Overview
      { path: '', component: () => import('@/pages/dashboard/UnifiedDashboard.vue'), meta: { title: 'menu.overview' } },

      // ===== On-Call =====
      { path: 'oncall', redirect: '/oncall/overview' },
      { path: 'oncall/overview', component: () => import('@/pages/dashboard/IncidentDashboard.vue'), meta: { title: 'menu.overview' } },
      { path: 'oncall/spaces', component: () => import('@/pages/channels/Index.vue'), meta: { title: 'menu.channels', requiresRole: ['admin', 'team_lead', 'member'] } },
      { path: 'oncall/spaces/:id', component: () => import('@/pages/channels/Detail.vue'), meta: { title: 'menu.channels', requiresRole: ['admin', 'team_lead', 'member'] } },
      { path: 'oncall/incidents', component: () => import('@/pages/incidents/Index.vue'), meta: { title: 'menu.incidents' } },
      { path: 'oncall/incidents/:id', component: () => import('@/pages/incidents/Detail.vue'), meta: { title: 'route.incidentDetail' } },
      { path: 'oncall/status-page', component: () => import('@/pages/oncall/StatusPage.vue'), meta: { title: 'menu.statusPage' } },
      { path: 'oncall/postmortems', component: () => import('@/pages/incidents/PostMortems.vue'), meta: { title: 'menu.postmortems' } },
      { path: 'oncall/integrations', component: () => import('@/pages/integrations/Index.vue'), meta: { title: 'menu.integrations', requiresRole: ['admin', 'team_lead'] } },
      { path: 'oncall/schedule', component: () => import('@/pages/schedule/Index.vue'), meta: { title: 'menu.schedule', requiresRole: ['admin', 'team_lead'] } },
      { path: 'oncall/config/notify-rules', component: () => import('@/pages/notification/Rules.vue'), meta: { title: 'menu.notifyChannels' } },
      { path: 'oncall/config/routing-rules', component: () => import('@/pages/integrations/RoutingRules.vue'), meta: { title: 'menu.routingRules' } },
      { path: 'oncall/config/biz-groups', component: () => import('@/pages/settings/BizGroups.vue'), meta: { title: 'menu.bizGroups' } },
      { path: 'oncall/config/subscribe-rules', component: () => import('@/pages/notification/Subscribe.vue'), meta: { title: 'menu.subscribeRules' } },

      // ===== Alert =====
      { path: 'alert', redirect: '/alert/overview' },
      { path: 'alert/overview', component: () => import('@/pages/dashboard/Index.vue'), meta: { title: 'menu.overview' } },
      { path: 'alert/rules', component: () => import('@/pages/alerts/rules/Index.vue'), meta: { title: 'menu.alertRules' } },
      { path: 'alert/events', component: () => import('@/pages/alerts/events/Index.vue'), meta: { title: 'menu.activeAlerts' } },
      { path: 'alert/events/:id', component: () => import('@/pages/alerts/events/Detail.vue'), meta: { title: 'route.alertDetail' } },
      { path: 'alert/history', component: () => import('@/pages/alerts/history/Index.vue'), meta: { title: 'menu.alertHistory' } },
      { path: 'alert/suppression', component: () => import('@/pages/alerts/mute/Index.vue'), meta: { title: 'menu.muteRules' } },
      { path: 'alert/suppression/inhibition', component: () => import('@/pages/alerts/inhibition/Index.vue'), meta: { title: 'menu.inhibitionRules' } },
      { path: 'alert/presets', component: () => import('@/pages/alerts/Presets.vue'), meta: { title: '预置规则库' } },
      { path: 'alert/datasources', component: () => import('@/pages/datasources/Index.vue'), meta: { title: 'menu.datasources' } },
      { path: 'alert/explore', component: () => import('@/pages/explore/Index.vue'), meta: { title: 'menu.dataQuery' } },
      { path: 'alert/dashboards', component: () => import('@/pages/dashboard-v2/Index.vue'), meta: { title: 'menu.dashboard' } },
      { path: 'alert/dashboards/:id', component: () => import('@/pages/dashboard-v2/View.vue'), meta: { title: 'menu.dashboard' } },
      { path: 'alert/notify/policies', component: () => import('@/pages/notification/Index.vue'), meta: { title: 'menu.notifyPolicies' } },
      { path: 'alert/notify/templates', component: () => import('@/pages/notification/Templates.vue'), meta: { title: 'menu.templates' } },
      { path: 'alert/notify/channels', component: () => import('@/pages/notification/Media.vue'), meta: { title: 'menu.notifyChannels' } },
      { path: 'alert/notify/subscriptions', component: () => import('@/pages/notification/Subscribe.vue'), meta: { title: 'menu.subscriptions' } },

      // ===== Platform =====
      { path: 'platform', redirect: '/platform/profile' },
      { path: 'platform/profile', component: () => import('@/pages/platform/Profile.vue'), meta: { title: 'menu.profile' } },
      { path: 'pet', component: () => import('@/pages/pet/Index.vue'), meta: { title: 'route.pet' } },
      { path: 'platform/org/members', component: () => import('@/pages/settings/UserManagement.vue'), meta: { title: 'menu.members', requiresRole: ['admin'] } },
      { path: 'platform/org/teams', component: () => import('@/pages/settings/TeamManagement.vue'), meta: { title: 'menu.teams', requiresRole: ['admin', 'team_lead'] } },
      { path: 'platform/org/roles', component: () => import('@/pages/platform/Roles.vue'), meta: { title: 'menu.roles' } },
      { path: 'platform/org/sso', component: () => import('@/pages/settings/SSO.vue'), meta: { title: 'menu.sso', requiresRole: ['admin'] } },
      { path: 'platform/audit', component: () => import('@/pages/settings/AuditLogs.vue'), meta: { title: 'menu.audit', requiresRole: ['admin'] } },
      { path: 'platform/settings/smtp', component: () => import('@/pages/settings/SMTP.vue'), meta: { title: 'menu.smtp', requiresRole: ['admin'] } },
      { path: 'platform/settings/lark', component: () => import('@/pages/settings/LarkBot.vue'), meta: { title: 'menu.larkBot', requiresRole: ['admin'] } },
      { path: 'platform/settings/ai', component: () => import('@/pages/settings/AI.vue'), meta: { title: 'menu.aiConfig', requiresRole: ['admin'] } },
      { path: 'platform/settings/ai-settings', component: () => import('@/pages/settings/AISettings.vue'), meta: { title: 'AI 配置', requiresRole: ['admin'] } },
      { path: 'platform/settings/security', component: () => import('@/pages/settings/Security.vue'), meta: { title: 'menu.security', requiresRole: ['admin'] } },

      // ===== Legacy Redirects =====
      { path: 'dashboard', redirect: '/oncall/overview' },
      { path: 'channels', redirect: '/oncall/spaces' },
      { path: 'channels/:id', redirect: (to: RouteLocation) => `/oncall/spaces/${to.params.id}` },
      { path: 'incidents', redirect: '/oncall/incidents' },
      { path: 'incidents/:id', redirect: (to: RouteLocation) => `/oncall/incidents/${to.params.id}` },
      { path: 'alerts-v2', redirect: '/oncall/incidents' },
      { path: 'alerts-v2/:id', redirect: (to: RouteLocation) => `/oncall/incidents/${to.params.id}` },
      { path: 'incident-dashboard', redirect: '/oncall/overview' },
      { path: 'integrations', redirect: '/oncall/integrations' },
      { path: 'schedule', redirect: '/oncall/schedule' },
      { path: 'alerts', redirect: '/alert/rules' },
      { path: 'alerts/rules', redirect: '/alert/rules' },
      { path: 'alerts/events', redirect: '/alert/events' },
      { path: 'alerts/events/:id', redirect: (to: RouteLocation) => `/alert/events/${to.params.id}` },
      { path: 'alerts/history', redirect: '/alert/history' },
      { path: 'alerts/mute-rules', redirect: '/alert/suppression' },
      { path: 'alerts/inhibition-rules', redirect: '/alert/suppression/inhibition' },
      { path: 'datasources', redirect: '/alert/datasources' },
      { path: 'query', redirect: '/alert/explore' },
      { path: 'explore', redirect: '/alert/explore' },
      { path: 'dashboards-v2', redirect: '/alert/dashboards' },
      { path: 'dashboards-v2/:id', redirect: (to: RouteLocation) => `/alert/dashboards/${to.params.id}` },
      { path: 'notification', redirect: '/alert/notify/policies' },
      { path: 'settings', redirect: '/platform/profile' },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Navigation guard
router.beforeEach((to, _from, next) => {
  // Handle OIDC callback: extract token from URL hash fragment
  // Backend redirects to /#oidc_token=...&expires_in=...
  const hash = window.location.hash
  if (hash && hash.includes('oidc_token=')) {
    const params = new URLSearchParams(hash.substring(1)) // strip leading #
    const oidcToken = params.get('oidc_token')
    if (oidcToken) {
      localStorage.setItem('token', oidcToken)
      // Clear the hash fragment
      window.history.replaceState(null, '', '/')
      next({ path: '/', replace: true })
      return
    }
  }

  // Also handle query param for backwards compatibility
  const oidcTokenQuery = to.query.oidc_token as string | undefined
  if (oidcTokenQuery) {
    localStorage.setItem('token', oidcTokenQuery)
    next({ path: '/', replace: true })
    return
  }

  const token = localStorage.getItem('token')

  if (to.meta.requiresAuth !== false && !token) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else if (to.name === 'Login' && token) {
    next({ path: '/' })
  } else if (to.meta.requiresRole) {
    // Route-level role guard: prefer store, fall back to localStorage (pre-hydration)
    const authStore = useAuthStore()
    const role = authStore.user?.role || localStorage.getItem('user_role')
    const allowedRoles = to.meta.requiresRole as string[]
    if (role && !allowedRoles.includes(role)) {
      next({ path: '/' })
    } else {
      next()
    }
  } else {
    next()
  }
})

export default router
