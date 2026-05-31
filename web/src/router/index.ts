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
      { path: 'oncall/integrations', redirect: '/alert/integrations' },
      { path: 'oncall/schedule', component: () => import('@/pages/schedule/Index.vue'), meta: { title: 'menu.schedule', requiresRole: ['admin', 'team_lead'] } },
      { path: 'oncall/my-alerts', redirect: '/alert/events' },
      { path: 'oncall/config/escalation-policies', component: () => import('@/pages/oncall/EscalationPolicies.vue'), meta: { title: 'menu.escalationPolicies' } },
      { path: 'oncall/config/notify-rules', redirect: '/oncall/notify/policies' },
      { path: 'oncall/config/routing-rules', redirect: '/alert/routing-rules' },
      { path: 'oncall/config/biz-groups', component: () => import('@/pages/settings/BizGroups.vue'), meta: { title: 'menu.bizGroups', requiresRole: ['admin', 'team_lead'] } },
      { path: 'oncall/config/subscribe-rules', redirect: '/oncall/notify/subscriptions' },

      // ===== Oncall — Notification Center =====
      { path: 'oncall/notify/policies', component: () => import('@/pages/notification/Index.vue'), meta: { title: 'menu.notifyPolicies', requiresRole: ['admin', 'team_lead'] } },
      { path: 'oncall/notify/templates', component: () => import('@/pages/notification/Templates.vue'), meta: { title: 'menu.templates', requiresRole: ['admin', 'team_lead'] } },
      { path: 'oncall/notify/channels', component: () => import('@/pages/notification/Media.vue'), meta: { title: 'menu.notifyChannels', requiresRole: ['admin', 'team_lead'] } },
      { path: 'oncall/notify/subscriptions', component: () => import('@/pages/notification/Subscribe.vue'), meta: { title: 'menu.subscriptions' } },
      { path: 'oncall/notify/alert-channels', component: () => import('@/pages/notification/AlertChannels.vue'), meta: { title: 'menu.alertChannels', requiresRole: ['admin', 'team_lead'] } },

      // ===== Alert =====
      { path: 'alert', redirect: '/alert/overview' },
      { path: 'alert/overview', component: () => import('@/pages/dashboard/Index.vue'), meta: { title: 'menu.overview' } },
      { path: 'alert/rules', component: () => import('@/pages/alerts/rules/Index.vue'), meta: { title: 'menu.alertRules', requiresRole: ['admin', 'team_lead'] } },
      { path: 'alert/events', component: () => import('@/pages/alerts/events/Index.vue'), meta: { title: 'menu.activeAlerts' } },
      { path: 'alert/events/:id', component: () => import('@/pages/alerts/events/Detail.vue'), meta: { title: 'route.alertDetail' } },
      { path: 'alert/history', component: () => import('@/pages/alerts/history/Index.vue'), meta: { title: 'menu.alertHistory' } },
      { path: 'alert/suppression', component: () => import('@/pages/alerts/mute/Index.vue'), meta: { title: 'menu.muteRules', requiresRole: ['admin', 'team_lead'] } },
      { path: 'alert/suppression/inhibition', component: () => import('@/pages/alerts/inhibition/Index.vue'), meta: { title: 'menu.inhibitionRules' } },
      { path: 'alert/recording-rules', component: () => import('@/pages/alerts/recording-rules/Index.vue'), meta: { title: 'menu.recordingRules', requiresRole: ['admin', 'team_lead'] } },
      { path: 'alert/event-pipelines', component: () => import('@/pages/alerts/event-pipelines/Index.vue'), meta: { title: 'menu.eventPipelines', requiresRole: ['admin', 'team_lead'] } },
      { path: 'alert/builtin-metrics', component: () => import('@/pages/alerts/builtin-metrics/Index.vue'), meta: { title: 'menu.builtinMetrics', requiresRole: ['admin', 'team_lead'] } },
      { path: 'alert/presets', redirect: '/alert/template-library' },
      { path: 'alert/template-library', component: () => import('@/pages/alerts/TemplateLibrary.vue'), meta: { title: 'menu.templateLibrary' } },
      { path: 'alert/datasources', component: () => import('@/pages/datasources/Index.vue'), meta: { title: 'menu.datasources', requiresRole: ['admin', 'team_lead'] } },
      { path: 'alert/explore', component: () => import('@/pages/explore/Index.vue'), meta: { title: 'menu.dataQuery' } },
      { path: 'alert/metric-views', component: () => import('@/pages/alerts/metric-views/Index.vue'), meta: { title: 'menu.metricViews' } },
      { path: 'alert/saved-views', component: () => import('@/pages/alerts/SavedViews.vue'), meta: { title: 'menu.savedViews' } },
      { path: 'alert/rule-templates', redirect: '/alert/template-library' },
      { path: 'alert/es-explore', component: () => import('@/pages/explore/ESExplorer.vue'), meta: { title: 'menu.esExplorer' } },
      { path: 'alert/es-patterns', component: () => import('@/pages/alerts/es-patterns/Index.vue'), meta: { title: 'menu.esPatterns', requiresRole: ['admin', 'team_lead'] } },
      { path: 'alert/dashboards', component: () => import('@/pages/dashboards/Index.vue'), meta: { title: 'menu.dashboard', requiresRole: ['admin', 'team_lead'] } },
      { path: 'alert/dashboards/builtin', component: () => import('@/pages/dashboards/BuiltinLibrary.vue'), meta: { title: 'menu.builtinDashboards' } },
      { path: 'alert/dashboards/:id', component: () => import('@/pages/dashboards/View.vue'), meta: { title: 'menu.dashboard' } },
      // Legacy notification routes → redirect to /oncall/notify/* (v4.43.0 migration)
      { path: 'alert/notify/policies', redirect: '/oncall/notify/policies' },
      { path: 'alert/notify/templates', redirect: '/oncall/notify/templates' },
      { path: 'alert/notify/channels', redirect: '/oncall/notify/channels' },
      { path: 'alert/notify/subscriptions', redirect: '/oncall/notify/subscriptions' },

      // ===== Alert — Data Ingestion (moved from oncall notify center) =====
      { path: 'alert/integrations', component: () => import('@/pages/integrations/Index.vue'), meta: { title: 'menu.integrations', requiresRole: ['admin', 'team_lead'] } },
      { path: 'alert/routing-rules', component: () => import('@/pages/integrations/RoutingRules.vue'), meta: { title: 'menu.routingRules', requiresRole: ['admin', 'team_lead'] } },

      // ===== Platform =====
      { path: 'platform', redirect: '/platform/profile' },
      { path: 'platform/profile', component: () => import('@/pages/platform/Profile.vue'), meta: { title: 'menu.profile' } },
      { path: 'platform/org/members', component: () => import('@/pages/settings/UserManagement.vue'), meta: { title: 'menu.members', requiresRole: ['admin'] } },
      { path: 'platform/org/teams', component: () => import('@/pages/settings/TeamManagement.vue'), meta: { title: 'menu.teams', requiresRole: ['admin', 'team_lead'] } },
      { path: 'platform/org/roles', component: () => import('@/pages/platform/Roles.vue'), meta: { title: 'menu.roles' } },
      { path: 'platform/org/sso', component: () => import('@/pages/settings/SSO.vue'), meta: { title: 'menu.sso', requiresRole: ['admin'] } },
      { path: 'platform/audit', component: () => import('@/pages/settings/AuditLog.vue'), meta: { title: 'menu.audit', requiresRole: ['admin'] } },
      { path: 'platform/settings/smtp', component: () => import('@/pages/settings/SMTP.vue'), meta: { title: 'menu.smtp', requiresRole: ['admin'] } },
      { path: 'platform/settings/lark', component: () => import('@/pages/settings/LarkBotConfig.vue'), meta: { title: 'menu.larkBot', requiresRole: ['admin'] } },
      { path: 'platform/settings/ai', redirect: '/platform/ai-config' },
      { path: 'platform/llm-configs', redirect: '/platform/ai-config#llm' },
      { path: 'platform/mcp-servers', redirect: '/platform/ai-config#mcp' },
      { path: 'ai/skills', redirect: '/platform/ai-config#skills' },
      { path: 'platform/ai-config', component: () => import('@/pages/ai/ConfigView.vue'), meta: { title: 'menu.aiConfig', requiresRole: ['admin'] } },
      { path: 'platform/settings/security', component: () => import('@/pages/settings/Security.vue'), meta: { title: 'menu.security', requiresRole: ['admin'] } },
      { path: 'platform/settings/contacts', component: () => import('@/pages/settings/Contacts.vue'), meta: { title: 'menu.contacts' } },
      { path: 'platform/settings/site-info', component: () => import('@/pages/settings/SiteInfo.vue'), meta: { title: 'menu.siteInfo', requiresRole: ['admin'] } },

      // ===== Notification Center =====
      { path: 'notifications', component: () => import('@/pages/notification/Center.vue'), meta: { title: 'notification.centerTitle' } },

      // ===== AI Agent =====
      { path: 'ai/agent', component: () => import('@/pages/ai/AgentView.vue'), meta: { title: 'menu.aiAgent' } },

      // ===== Inspection (定时巡检) =====
      { path: 'platform/inspections', component: () => import('@/pages/platform/inspections/Index.vue'), meta: { title: 'menu.inspection', requiresRole: ['admin', 'team_lead'] } },
      { path: 'platform/inspections/runs/:id', component: () => import('@/pages/platform/inspections/RunDetail.vue'), meta: { title: 'menu.inspectionDetail', requiresRole: ['admin', 'team_lead'] } },

      // ===== Diagnostic Workflows =====
      { path: 'platform/diagnostic-workflows', component: () => import('@/pages/platform/DiagnosticWorkflows.vue'), meta: { title: 'menu.diagnosticWorkflows', requiresRole: ['admin', 'team_lead'] } },

      // ===== Change Events =====
      { path: 'platform/change-events', component: () => import('@/pages/platform/ChangeEvents.vue'), meta: { title: 'menu.changeEvents', requiresRole: ['admin', 'team_lead'] } },

      // ===== Task Execution (任务执行) =====
      { path: 'platform/task-tpls', component: () => import('@/pages/platform/tasks/TplIndex.vue'), meta: { title: 'menu.taskTpls', requiresRole: ['admin', 'team_lead'] } },
      { path: 'platform/tasks', component: () => import('@/pages/platform/tasks/TaskIndex.vue'), meta: { title: 'menu.tasks', requiresRole: ['admin', 'team_lead', 'member'] } },
      { path: 'platform/tasks/:id', component: () => import('@/pages/platform/tasks/TaskResult.vue'), meta: { title: 'menu.taskDetail', requiresRole: ['admin', 'team_lead', 'member'] } },
      { path: 'platform/knowledge', component: () => import('@/pages/platform/Knowledge.vue'), meta: { title: 'menu.knowledge' } },
      { path: 'platform/annotations', component: () => import('@/pages/platform/Annotations.vue'), meta: { title: 'menu.annotations' } },

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

  // Handle OAuth2 callback: extract token from URL hash fragment
  // Backend redirects to /#oauth2_token=...&expires_in=...
  if (hash && hash.includes('oauth2_token=')) {
    const params = new URLSearchParams(hash.substring(1))
    const oauth2Token = params.get('oauth2_token')
    if (oauth2Token) {
      localStorage.setItem('token', oauth2Token)
      window.history.replaceState(null, '', '/')
      next({ path: '/', replace: true })
      return
    }
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
