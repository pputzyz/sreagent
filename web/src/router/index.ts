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
    component: () => import('@/layouts/MainLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        redirect: '/dashboard',
      },
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/pages/dashboard/Index.vue'),
        meta: { title: 'Dashboard', icon: 'dashboard' },
      },
      {
        path: 'datasources',
        name: 'DataSources',
        component: () => import('@/pages/datasources/Index.vue'),
        meta: { title: 'Data Sources', icon: 'server' },
      },
      {
        path: 'query',
        name: 'DataQuery',
        component: () => import('@/pages/explore/Index.vue'),
        meta: { title: 'Data Query', icon: 'search' },
      },
      // Backward-compatible redirects
      { path: 'explore', redirect: '/query' },
      { path: 'datasources/query', redirect: '/query' },
      { path: 'explore/logs', redirect: '/query' },
      {
        path: 'dashboards-v2',
        name: 'DashboardV2List',
        component: () => import('@/pages/dashboard-v2/Index.vue'),
        meta: { title: 'Dashboards V2', icon: 'dashboard' },
      },
      {
        path: 'dashboards-v2/:id',
        name: 'DashboardV2View',
        component: () => import('@/pages/dashboard-v2/View.vue'),
        meta: { title: 'Dashboard', icon: 'dashboard' },
      },
      {
        path: 'alerts',
        name: 'Alerts',
        redirect: '/alerts/rules',
        children: [
          {
            path: 'rules',
            name: 'AlertRules',
            component: () => import('@/pages/alerts/rules/Index.vue'),
            meta: { title: 'Alert Rules', icon: 'rule' },
          },
          {
            path: 'events',
            name: 'AlertEvents',
            component: () => import('@/pages/alerts/events/Index.vue'),
            meta: { title: 'Active Alerts', icon: 'alert' },
          },
          {
            path: 'events/:id',
            name: 'AlertEventDetail',
            component: () => import('@/pages/alerts/events/Detail.vue'),
            meta: { title: 'Alert Detail' },
          },
          {
            path: 'history',
            name: 'AlertHistory',
            component: () => import('@/pages/alerts/history/Index.vue'),
            meta: { title: 'Alert History', icon: 'history' },
          },
          {
            path: 'mute-rules',
            name: 'MuteRules',
            component: () => import('@/pages/alerts/mute/Index.vue'),
            meta: { title: 'Mute Rules', icon: 'mute' },
          },
          {
            path: 'inhibition-rules',
            name: 'InhibitionRules',
            component: () => import('@/pages/alerts/inhibition/Index.vue'),
            meta: { title: 'Inhibition Rules', icon: 'inhibition' },
          },
        ],
      },
      {
        path: 'notification',
        name: 'Notification',
        component: () => import('@/pages/notification/Index.vue'),
        meta: { title: 'Notification' },
      },
      {
        path: 'schedule',
        name: 'Schedule',
        component: () => import('@/pages/schedule/Index.vue'),
        meta: { title: 'On-Call Schedule', icon: 'calendar' },
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/pages/settings/Index.vue'),
        meta: { title: 'Settings', icon: 'settings', requiresRole: ['admin', 'team_lead'] },
      },
      // v2 routes
      {
        path: 'channels',
        name: 'Channels',
        component: () => import('@/pages/channels/Index.vue'),
        meta: { title: 'Channels' },
      },
      {
        path: 'channels/:id',
        name: 'ChannelDetail',
        component: () => import('@/pages/channels/Detail.vue'),
        meta: { title: 'Channel Detail' },
      },
      {
        path: 'incidents',
        name: 'Incidents',
        component: () => import('@/pages/incidents/Index.vue'),
        meta: { title: 'Incidents' },
      },
      {
        path: 'incidents/:id',
        name: 'IncidentDetail',
        component: () => import('@/pages/incidents/Detail.vue'),
        meta: { title: 'Incident Detail' },
      },
      {
        path: 'alerts-v2',
        name: 'AlertsV2',
        component: () => import('@/pages/alerts-v2/Index.vue'),
        meta: { title: 'Alert View' },
      },
      {
        path: 'alerts-v2/:id',
        name: 'AlertV2Detail',
        component: () => import('@/pages/alerts-v2/Detail.vue'),
        meta: { title: 'Alert Detail' },
      },
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
      next({ name: 'Dashboard', replace: true })
      return
    }
  }

  // Also handle query param for backwards compatibility
  const oidcTokenQuery = to.query.oidc_token as string | undefined
  if (oidcTokenQuery) {
    localStorage.setItem('token', oidcTokenQuery)
    next({ name: 'Dashboard', replace: true })
    return
  }

  const token = localStorage.getItem('token')

  if (to.meta.requiresAuth !== false && !token) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else if (to.name === 'Login' && token) {
    next({ name: 'Dashboard' })
  } else if (to.meta.requiresRole) {
    // Route-level role guard: prefer store, fall back to localStorage (pre-hydration)
    const authStore = useAuthStore()
    const role = authStore.user?.role || localStorage.getItem('user_role')
    const allowedRoles = to.meta.requiresRole as string[]
    if (role && !allowedRoles.includes(role)) {
      next({ name: 'Dashboard' })
    } else {
      next()
    }
  } else {
    next()
  }
})

export default router
