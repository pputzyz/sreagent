import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
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
      // Root redirect
      { path: '', redirect: '/oncall/overview' },

      // ===== On-Call =====
      { path: 'oncall', redirect: '/oncall/overview' },
      { path: 'oncall/overview', component: () => import('@/pages/dashboard/IncidentDashboard.vue'), meta: { title: '概览' } },
      { path: 'oncall/spaces', component: () => import('@/pages/channels/Index.vue'), meta: { title: '协作空间', requiresRole: ['admin', 'team_lead', 'member'] } },
      { path: 'oncall/spaces/:id', component: () => import('@/pages/channels/Detail.vue'), meta: { title: '协作空间', requiresRole: ['admin', 'team_lead', 'member'] } },
      { path: 'oncall/incidents', component: () => import('@/pages/incidents/Index.vue'), meta: { title: '故障列表' } },
      { path: 'oncall/incidents/:id', component: () => import('@/pages/incidents/Detail.vue'), meta: { title: '故障详情' } },
      { path: 'oncall/status-page', component: () => import('@/pages/oncall/StatusPage.vue'), meta: { title: '状态页面' } },
      { path: 'oncall/postmortems', component: () => import('@/pages/incidents/PostMortems.vue'), meta: { title: '故障复盘' } },
      { path: 'oncall/integrations', component: () => import('@/pages/integrations/Index.vue'), meta: { title: '集成中心', requiresRole: ['admin', 'team_lead'] } },
      { path: 'oncall/schedule', component: () => import('@/pages/schedule/Index.vue'), meta: { title: '值班管理', requiresRole: ['admin', 'team_lead'] } },
      { path: 'oncall/config/notify-rules', component: () => import('@/pages/notification/Rules.vue'), meta: { title: '通道通知规则' } },
      { path: 'oncall/config/routing-rules', component: () => import('@/pages/integrations/RoutingRules.vue'), meta: { title: '路由规则' } },
      { path: 'oncall/config/biz-groups', component: () => import('@/pages/settings/BizGroups.vue'), meta: { title: '业务分组' } },
      { path: 'oncall/config/subscribe-rules', component: () => import('@/pages/notification/Subscribe.vue'), meta: { title: '订阅规则' } },

      // ===== Alert =====
      { path: 'alert', redirect: '/alert/overview' },
      { path: 'alert/overview', component: () => import('@/pages/dashboard/Index.vue'), meta: { title: '概览' } },
      { path: 'alert/rules', component: () => import('@/pages/alerts/rules/Index.vue'), meta: { title: '告警规则' } },
      { path: 'alert/events', component: () => import('@/pages/alerts/events/Index.vue'), meta: { title: '活跃告警' } },
      { path: 'alert/events/:id', component: () => import('@/pages/alerts/events/Detail.vue'), meta: { title: '告警详情' } },
      { path: 'alert/history', component: () => import('@/pages/alerts/history/Index.vue'), meta: { title: '告警历史' } },
      { path: 'alert/suppression', component: () => import('@/pages/alerts/mute/Index.vue'), meta: { title: '静默规则' } },
      { path: 'alert/suppression/inhibition', component: () => import('@/pages/alerts/inhibition/Index.vue'), meta: { title: '抑制规则' } },
      { path: 'alert/datasources', component: () => import('@/pages/datasources/Index.vue'), meta: { title: '数据源' } },
      { path: 'alert/explore', component: () => import('@/pages/explore/Index.vue'), meta: { title: '数据查询' } },
      { path: 'alert/dashboards', component: () => import('@/pages/dashboard-v2/Index.vue'), meta: { title: '仪表盘' } },
      { path: 'alert/dashboards/:id', component: () => import('@/pages/dashboard-v2/View.vue'), meta: { title: '仪表盘' } },
      { path: 'alert/notify/policies', component: () => import('@/pages/notification/Index.vue'), meta: { title: '通知策略' } },
      { path: 'alert/notify/templates', component: () => import('@/pages/notification/Templates.vue'), meta: { title: '消息模板' } },
      { path: 'alert/notify/channels', component: () => import('@/pages/notification/Media.vue'), meta: { title: '通知渠道' } },
      { path: 'alert/notify/subscriptions', component: () => import('@/pages/notification/Subscribe.vue'), meta: { title: '订阅管理' } },

      // ===== Platform =====
      { path: 'platform', redirect: '/platform/profile' },
      { path: 'platform/profile', component: () => import('@/pages/platform/Profile.vue'), meta: { title: '个人中心' } },
      { path: 'pet', component: () => import('@/pages/pet/Index.vue'), meta: { title: '我的宠物' } },
      { path: 'platform/org/members', component: () => import('@/pages/settings/UserManagement.vue'), meta: { title: '成员管理', requiresRole: ['admin'] } },
      { path: 'platform/org/teams', component: () => import('@/pages/settings/TeamManagement.vue'), meta: { title: '团队管理', requiresRole: ['admin', 'team_lead'] } },
      { path: 'platform/org/roles', component: () => import('@/pages/platform/Roles.vue'), meta: { title: '角色权限' } },
      { path: 'platform/org/sso', component: () => import('@/pages/settings/SSO.vue'), meta: { title: '单点登录', requiresRole: ['admin'] } },
      { path: 'platform/audit', component: () => import('@/pages/settings/AuditLogs.vue'), meta: { title: '审计日志', requiresRole: ['admin'] } },
      { path: 'platform/settings/smtp', component: () => import('@/pages/settings/SMTP.vue'), meta: { title: '邮件服务', requiresRole: ['admin'] } },
      { path: 'platform/settings/lark', component: () => import('@/pages/settings/LarkBot.vue'), meta: { title: '飞书机器人', requiresRole: ['admin'] } },
      { path: 'platform/settings/ai', component: () => import('@/pages/settings/AI.vue'), meta: { title: 'AI 配置', requiresRole: ['admin'] } },
      { path: 'platform/settings/security', component: () => import('@/pages/settings/Security.vue'), meta: { title: '安全设置', requiresRole: ['admin'] } },

      // ===== Legacy Redirects =====
      { path: 'dashboard', redirect: '/oncall/overview' },
      { path: 'channels', redirect: '/oncall/spaces' },
      { path: 'channels/:id', redirect: (to: any) => `/oncall/spaces/${to.params.id}` },
      { path: 'incidents', redirect: '/oncall/incidents' },
      { path: 'incidents/:id', redirect: (to: any) => `/oncall/incidents/${to.params.id}` },
      { path: 'alerts-v2', redirect: '/oncall/incidents' },
      { path: 'alerts-v2/:id', redirect: (to: any) => `/oncall/incidents/${to.params.id}` },
      { path: 'incident-dashboard', redirect: '/oncall/overview' },
      { path: 'integrations', redirect: '/oncall/integrations' },
      { path: 'schedule', redirect: '/oncall/schedule' },
      { path: 'alerts', redirect: '/alert/rules' },
      { path: 'alerts/rules', redirect: '/alert/rules' },
      { path: 'alerts/events', redirect: '/alert/events' },
      { path: 'alerts/events/:id', redirect: (to: any) => `/alert/events/${to.params.id}` },
      { path: 'alerts/history', redirect: '/alert/history' },
      { path: 'alerts/mute-rules', redirect: '/alert/suppression' },
      { path: 'alerts/inhibition-rules', redirect: '/alert/suppression/inhibition' },
      { path: 'datasources', redirect: '/alert/datasources' },
      { path: 'query', redirect: '/alert/explore' },
      { path: 'explore', redirect: '/alert/explore' },
      { path: 'dashboards-v2', redirect: '/alert/dashboards' },
      { path: 'dashboards-v2/:id', redirect: (to: any) => `/alert/dashboards/${to.params.id}` },
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
      next({ path: '/oncall/overview', replace: true })
      return
    }
  }

  // Also handle query param for backwards compatibility
  const oidcTokenQuery = to.query.oidc_token as string | undefined
  if (oidcTokenQuery) {
    localStorage.setItem('token', oidcTokenQuery)
    next({ path: '/oncall/overview', replace: true })
    return
  }

  const token = localStorage.getItem('token')

  if (to.meta.requiresAuth !== false && !token) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else if (to.name === 'Login' && token) {
    next({ path: '/oncall/overview' })
  } else if (to.meta.requiresRole) {
    // Route-level role guard: prefer store, fall back to localStorage (pre-hydration)
    const authStore = useAuthStore()
    const role = authStore.user?.role || localStorage.getItem('user_role')
    const allowedRoles = to.meta.requiresRole as string[]
    if (role && !allowedRoles.includes(role)) {
      next({ path: '/oncall/overview' })
    } else {
      next()
    }
  } else {
    next()
  }
})

export default router
