# Phase 1: Frontend Navigation Refactor — 三栏布局 + 应用分区

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current top-tab + sidebar navigation with a three-column layout (icon rail + menu sidebar + content area) supporting three apps (On-Call, Alert, Platform).

**Architecture:** Extract navigation logic from MainLayout.vue into a `useAppNav` composable. Build new layout components (AppShell, AppRail, AppSidebar). Restructure routes under `/oncall/`, `/alert/`, `/platform/` prefixes. Add backward-compatible redirects for all old routes. No backend changes.

**Tech Stack:** Vue 3, Naive UI, Vue Router, Pinia, Ionicons5

---

## File Structure

### New files to create

| File | Responsibility |
|------|---------------|
| `web/src/composables/useAppNav.ts` | Navigation state: active app, menu sections, route resolution |
| `web/src/layouts/AppShell.vue` | Three-column layout shell (rail + sidebar + content) |
| `web/src/layouts/AppRail.vue` | Left icon rail (48px) — app switcher |
| `web/src/layouts/AppSidebar.vue` | Left menu sidebar (220px) — current app's menu |

### Files to modify

| File | Change |
|------|--------|
| `web/src/router/index.ts` | Restructure routes under app prefixes, add redirects |
| `web/src/composables/useCommandPalette.ts` | Update route list for new paths |
| `web/src/App.vue` | No changes needed (already renders `<router-view />`) |

### Files to delete (after migration verified)

| File | Reason |
|------|--------|
| `web/src/layouts/MainLayout.vue` | Replaced by AppShell + AppRail + AppSidebar |

---

### Task 1: Create `useAppNav` Composable

**Files:**
- Create: `web/src/composables/useAppNav.ts`

Extract all navigation logic from MainLayout.vue into a reusable composable. This is the foundation for all layout components.

- [ ] **Step 1: Create the composable file**

```ts
// web/src/composables/useAppNav.ts
import { ref, computed, watch, inject } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import type { Ref } from 'vue'

export type AppKey = 'oncall' | 'alert' | 'platform'

export interface MenuItem {
  label: string
  key: string        // route path
  icon?: any
  children?: MenuItem[]
  show?: boolean     // permission-based visibility
}

export interface MenuSection {
  label?: string
  items: MenuItem[]
}

export function useAppNav() {
  const route = useRoute()
  const router = useRouter()
  const { t } = useI18n()
  const authStore = useAuthStore()

  // --- Active App ---
  const activeApp = ref<AppKey>(resolveAppFromPath(route.path))

  function resolveAppFromPath(path: string): AppKey {
    if (path.startsWith('/oncall')) return 'oncall'
    if (path.startsWith('/alert')) return 'alert'
    if (path.startsWith('/platform')) return 'platform'
    // Legacy route mapping
    if (path.startsWith('/incident-dashboard') || path.startsWith('/channels') ||
        path.startsWith('/incidents') || path.startsWith('/alerts-v2')) return 'oncall'
    if (path.startsWith('/alerts') || path.startsWith('/datasources') ||
        path.startsWith('/query') || path.startsWith('/dashboards') ||
        path.startsWith('/notification')) return 'alert'
    if (path.startsWith('/schedule') || path.startsWith('/integrations')) return 'oncall'
    if (path.startsWith('/settings')) return 'platform'
    return 'oncall' // default
  }

  // --- App Config ---
  const appConfig: Record<AppKey, { defaultRoute: string }> = {
    oncall: { defaultRoute: '/oncall/overview' },
    alert: { defaultRoute: '/alert/overview' },
    platform: { defaultRoute: '/platform/profile' },
  }

  function switchApp(app: AppKey) {
    activeApp.value = app
    const config = appConfig[app]
    // Only navigate if current route doesn't belong to this app
    if (resolveAppFromPath(route.path) !== app) {
      router.push(config.defaultRoute)
    }
  }

  // --- Menu Sections per App ---
  const menuSections = computed<MenuSection[]>(() => {
    switch (activeApp.value) {
      case 'oncall':
        return [
          {
            items: [
              { label: t('menu.overview'), key: '/oncall/overview' },
            ],
          },
          {
            items: [
              { label: t('menu.channels'), key: '/oncall/spaces' },
              { label: t('menu.incidents'), key: '/oncall/incidents' },
              { label: '状态页面', key: '/oncall/status-page' },
              { label: '故障复盘', key: '/oncall/postmortems' },
            ],
          },
          {
            items: [
              { label: t('menu.integrations'), key: '/oncall/integrations', show: authStore.canManage },
              { label: t('menu.schedule'), key: '/oncall/schedule', show: authStore.canManage },
            ],
          },
          {
            label: '配置中心',
            items: [
              { label: '通道通知规则', key: '/oncall/config/notify-rules' },
              { label: '路由规则', key: '/oncall/config/routing-rules' },
              { label: '业务分组', key: '/oncall/config/biz-groups' },
              { label: '订阅规则', key: '/oncall/config/subscribe-rules' },
            ],
          },
        ]
      case 'alert':
        return [
          {
            items: [
              { label: t('menu.overview'), key: '/alert/overview' },
            ],
          },
          {
            label: '告警',
            items: [
              { label: t('menu.alertRules'), key: '/alert/rules' },
              { label: t('menu.activeAlerts'), key: '/alert/events' },
              { label: t('menu.alertHistory'), key: '/alert/history' },
              { label: '静默与抑制', key: '/alert/suppression' },
            ],
          },
          {
            label: '数据',
            items: [
              { label: t('menu.datasources'), key: '/alert/datasources' },
              { label: t('menu.dataQuery'), key: '/alert/explore' },
              { label: '仪表盘', key: '/alert/dashboards' },
            ],
          },
          {
            label: '通知',
            items: [
              { label: '通知策略', key: '/alert/notify/policies' },
              { label: '消息模板', key: '/alert/notify/templates' },
              { label: '通知渠道', key: '/alert/notify/channels' },
              { label: '订阅管理', key: '/alert/notify/subscriptions' },
            ],
          },
        ]
      case 'platform':
        return [
          {
            items: [
              { label: '个人中心', key: '/platform/profile' },
            ],
          },
          {
            label: '组织管理',
            items: [
              { label: '成员管理', key: '/platform/org/members', show: authStore.isAdmin },
              { label: '团队管理', key: '/platform/org/teams', show: authStore.canManage },
              { label: '角色权限', key: '/platform/org/roles' },
              { label: '单点登录', key: '/platform/org/sso', show: authStore.isAdmin },
            ],
          },
          {
            items: [
              { label: '审计日志', key: '/platform/audit', show: authStore.isAdmin },
            ],
          },
          {
            label: '系统设置',
            items: [
              { label: '邮件服务', key: '/platform/settings/smtp', show: authStore.isAdmin },
              { label: '飞书机器人', key: '/platform/settings/lark', show: authStore.isAdmin },
              { label: 'AI 配置', key: '/platform/settings/ai', show: authStore.isAdmin },
              { label: '安全设置', key: '/platform/settings/security', show: authStore.isAdmin },
            ],
          },
        ]
    }
  })

  // --- Active Menu Key ---
  const activeMenuKey = computed(() => {
    const path = route.path
    // Find matching menu item (longest prefix match)
    const allItems = menuSections.value.flatMap(s => s.items)
    let bestMatch = ''
    for (const item of allItems) {
      if (path.startsWith(item.key) && item.key.length > bestMatch.length) {
        bestMatch = item.key
      }
      if (item.children) {
        for (const child of item.children) {
          if (path.startsWith(child.key) && child.key.length > bestMatch.length) {
            bestMatch = child.key
          }
        }
      }
    }
    return bestMatch || path
  })

  // --- Page Title ---
  const pageTitle = computed(() => {
    const allItems = menuSections.value.flatMap(s => s.items)
    const item = allItems.find(m => m.key === activeMenuKey.value)
    return item?.label || ''
  })

  // --- Watch route changes ---
  watch(() => route.path, (path) => {
    activeApp.value = resolveAppFromPath(path)
  })

  return {
    activeApp,
    switchApp,
    menuSections,
    activeMenuKey,
    pageTitle,
  }
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd web && npx vue-tsc --noEmit`
Expected: PASS (composable has no template dependencies)

- [ ] **Step 3: Commit**

```bash
git add web/src/composables/useAppNav.ts
git commit -m "feat(nav): add useAppNav composable — navigation state for 3-app layout"
```

---

### Task 2: Create `AppRail` Component

**Files:**
- Create: `web/src/layouts/AppRail.vue`

The left icon rail (48px wide) with app icons and platform management gear.

- [ ] **Step 1: Create the component**

```vue
<!-- web/src/layouts/AppRail.vue -->
<script setup lang="ts">
import { h, inject } from 'vue'
import { NIcon, NTooltip } from 'naive-ui'
import type { Ref } from 'vue'
import type { AppKey } from '@/composables/useAppNav'
import {
  FlashOutline, AlertCircleOutline, SettingsOutline,
  SunnyOutline, MoonOutline,
} from '@vicons/ionicons5'

const props = defineProps<{ activeApp: AppKey }>()
const emit = defineEmits<{ (e: 'switch', app: AppKey): void }>()

const isDark = inject<Ref<boolean>>('isDark', ref(true))

const apps: { key: AppKey; icon: any; label: string }[] = [
  { key: 'oncall', icon: FlashOutline, label: 'On-Call' },
  { key: 'alert', icon: AlertCircleOutline, label: 'Alert' },
]

const bottomItems: { key: AppKey | 'theme'; icon: any; label: string }[] = [
  { key: 'platform', icon: SettingsOutline, label: 'Platform' },
]
</script>

<template>
  <div class="app-rail">
    <div class="rail-top">
      <n-tooltip v-for="app in apps" :key="app.key" placement="right" :show-arrow="false">
        <template #trigger>
          <button
            class="rail-icon"
            :class="{ active: activeApp === app.key }"
            @click="emit('switch', app.key)"
          >
            <n-icon :component="app.icon" :size="20" />
          </button>
        </template>
        {{ app.label }}
      </n-tooltip>
    </div>

    <div class="rail-bottom">
      <n-tooltip v-for="item in bottomItems" :key="item.key" placement="right" :show-arrow="false">
        <template #trigger>
          <button
            class="rail-icon"
            :class="{ active: activeApp === item.key }"
            @click="emit('switch', item.key as AppKey)"
          >
            <n-icon :component="item.icon" :size="20" />
          </button>
        </template>
        {{ item.label }}
      </n-tooltip>
    </div>
  </div>
</template>

<style scoped>
.app-rail {
  width: 48px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
  background: var(--sre-bg-base);
  border-right: 1px solid var(--sre-border);
}

.rail-top,
.rail-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.rail-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  border: none;
  background: transparent;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--sre-duration-base) var(--sre-ease-out);
}

.rail-icon:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
}

.rail-icon.active {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/layouts/AppRail.vue
git commit -m "feat(nav): add AppRail component — left icon rail for app switching"
```

---

### Task 3: Create `AppSidebar` Component

**Files:**
- Create: `web/src/layouts/AppSidebar.vue`

The menu sidebar (220px) that shows menu sections for the current app. Supports multi-level collapsible menus.

- [ ] **Step 1: Create the component**

```vue
<!-- web/src/layouts/AppSidebar.vue -->
<script setup lang="ts">
import { computed, inject, ref, h } from 'vue'
import { useRouter } from 'vue-router'
import { NMenu, NIcon } from 'naive-ui'
import type { Ref } from 'vue'
import type { MenuSection } from '@/composables/useAppNav'
import { useAuthStore } from '@/stores/auth'
import {
  ChevronBackOutline, ChevronForwardOutline,
} from '@vicons/ionicons5'

const props = defineProps<{
  sections: MenuSection[]
  activeKey: string
  collapsed: boolean
}>()

const emit = defineEmits<{
  (e: 'update:collapsed', value: boolean): void
  (e: 'navigate', key: string): void
}>()

const router = useRouter()
const authStore = useAuthStore()

// Convert MenuSection[] to Naive UI MenuOption[]
const menuOptions = computed(() => {
  return props.sections
    .filter(s => !s.items.every(i => i.show === false))
    .map(section => {
      const visibleItems = section.items.filter(i => i.show !== false)
      if (section.label) {
        return {
          type: 'group' as const,
          label: section.label,
          children: visibleItems.map(item => ({
            label: item.label,
            key: item.key,
          })),
        }
      }
      return visibleItems.map(item => ({
        label: item.label,
        key: item.key,
      }))
    })
    .flat()
})

function handleMenuUpdate(key: string) {
  emit('navigate', key)
  router.push(key)
}

const userInitial = computed(() =>
  (authStore.user?.display_name || authStore.user?.username || 'U').charAt(0).toUpperCase()
)
const displayName = computed(() =>
  authStore.user?.display_name || authStore.user?.username || 'User'
)
</script>

<template>
  <div class="app-sidebar" :class="{ collapsed }">
    <nav class="sidebar-nav">
      <n-menu
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions"
        :value="activeKey"
        :indent="16"
        @update:value="handleMenuUpdate"
      />
    </nav>

    <div class="sidebar-bottom">
      <div class="sidebar-user" :class="{ collapsed }">
        <div class="user-avatar">
          {{ userInitial }}
        </div>
        <transition name="fade">
          <div v-if="!collapsed" class="user-meta">
            <span class="user-name">{{ displayName }}</span>
            <span class="user-role">{{ authStore.canManage ? 'Admin' : 'Member' }}</span>
          </div>
        </transition>
      </div>

      <button class="sidebar-collapse" @click="emit('update:collapsed', !collapsed)">
        <n-icon :component="collapsed ? ChevronForwardOutline : ChevronBackOutline" :size="14" />
        <transition name="fade">
          <span v-if="!collapsed">收起</span>
        </transition>
      </button>
    </div>
  </div>
</template>

<style scoped>
.app-sidebar {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--sre-bg-card);
  border-right: 1px solid var(--sre-border);
  transition: width 280ms var(--sre-ease-spring);
  overflow: hidden;
}

.app-sidebar.collapsed {
  width: 64px;
}

.sidebar-nav {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px;
}

.sidebar-bottom {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  border-top: 1px solid var(--sre-border);
}

.sidebar-user {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--sre-radius-md);
  cursor: pointer;
  transition: background var(--sre-duration-fast) var(--sre-ease-out);
}

.sidebar-user:hover {
  background: var(--sre-bg-hover);
}

.sidebar-user.collapsed {
  justify-content: center;
  padding: 8px;
}

.user-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
  font-size: 12px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.user-meta {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.user-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-role {
  font-size: 10px;
  color: var(--sre-text-tertiary);
}

.sidebar-collapse {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border: none;
  border-radius: var(--sre-radius-sm);
  background: transparent;
  color: var(--sre-text-tertiary);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: background var(--sre-duration-fast) var(--sre-ease-out);
}

.sidebar-collapse:hover {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/layouts/AppSidebar.vue
git commit -m "feat(nav): add AppSidebar component — menu sidebar with collapsible sections"
```

---

### Task 4: Create `AppShell` Layout

**Files:**
- Create: `web/src/layouts/AppShell.vue`

The root layout that composes AppRail + AppSidebar + content area. Replaces MainLayout.vue.

- [ ] **Step 1: Create the component**

```vue
<!-- web/src/layouts/AppShell.vue -->
<script setup lang="ts">
import { ref, inject, watch, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { Ref } from 'vue'
import { NPopselect, NPopover, NIcon } from 'naive-ui'
import { useAppNav } from '@/composables/useAppNav'
import { useCommandPalette } from '@/composables/useCommandPalette'
import { useAuthStore } from '@/stores/auth'
import AppRail from '@/layouts/AppRail.vue'
import AppSidebar from '@/layouts/AppSidebar.vue'
import CommandPalette from '@/components/common/CommandPalette.vue'
import {
  TimeOutline, EarthOutline, SunnyOutline, MoonOutline, SearchOutline,
} from '@vicons/ionicons5'

const route = useRoute()
const router = useRouter()
const { t, locale } = useI18n()
const authStore = useAuthStore()
const { open: openPalette } = useCommandPalette()
const appVersion = __APP_VERSION__

const isDark = inject<Ref<boolean>>('isDark', ref(true))
const toggleTheme = inject<() => void>('toggleTheme', () => {})

const { activeApp, switchApp, menuSections, activeMenuKey, pageTitle } = useAppNav()

// Sidebar collapsed state
const sidebarCollapsed = ref(JSON.parse(localStorage.getItem('sre-sider-collapsed') ?? 'false'))
watch(sidebarCollapsed, v => localStorage.setItem('sre-sider-collapsed', JSON.stringify(v)))

// Clock
const timeDisplay = ref('')
const timezone = ref(localStorage.getItem('sre-timezone') || 'Asia/Shanghai')
const showTzPanel = ref(false)
const timezoneOptions = [
  { label: 'Asia/Shanghai', abbr: 'CST', value: 'Asia/Shanghai' },
  { label: 'UTC', abbr: 'UTC', value: 'UTC' },
  { label: 'Asia/Tokyo', abbr: 'JST', value: 'Asia/Tokyo' },
  { label: 'America/New_York', abbr: 'EST', value: 'America/New_York' },
]
const tzAbbr = computed(() => timezoneOptions.find(o => o.value === timezone.value)?.abbr || 'TZ')

function updateClock() {
  timeDisplay.value = new Date().toLocaleTimeString('en-GB', {
    timeZone: timezone.value, hour: '2-digit', minute: '2-digit', hour12: false,
  })
}
let clockInterval: ReturnType<typeof setInterval>
onMounted(() => { updateClock(); clockInterval = setInterval(updateClock, 1000) })
onUnmounted(() => clearInterval(clockInterval))

function selectTimezone(val: string) {
  timezone.value = val
  localStorage.setItem('sre-timezone', val)
  showTzPanel.value = false
  updateClock()
}

// Language
const langOptions = computed(() => [
  { label: t('language.zh'), value: 'zh-CN' },
  { label: t('language.en'), value: 'en' },
])
function handleLangChange(val: string) {
  locale.value = val
  localStorage.setItem('locale', val)
}

// Navigation
function handleNavigate(key: string) {
  router.push(key)
}
</script>

<template>
  <div class="app-shell">
    <!-- Top Bar -->
    <header class="topbar">
      <div class="topbar-start">
        <router-link to="/oncall/overview" class="topbar-logo">
          <img src="/logo.svg" alt="SREAgent" class="logo-img" />
          <span class="logo-label"><span class="gradient-text">SRE</span>Agent</span>
        </router-link>
      </div>

      <div class="topbar-end">
        <!-- Clock -->
        <n-popover v-model:show="showTzPanel" trigger="click" placement="bottom-end" :show-arrow="false" style="padding:0">
          <template #trigger>
            <button class="topbar-btn topbar-clock" :class="{ active: showTzPanel }">
              <n-icon :component="TimeOutline" :size="14" />
              <span class="clock-text">{{ timeDisplay }}</span>
              <span class="clock-tz">{{ tzAbbr }}</span>
            </button>
          </template>
          <div class="tz-panel">
            <div class="tz-panel-title">{{ t('header.timezone') }}</div>
            <div v-for="opt in timezoneOptions" :key="opt.value" class="tz-item" :class="{ selected: timezone === opt.value }" @click="selectTimezone(opt.value)">
              <span class="tz-abbr">{{ opt.abbr }}</span>
              <span class="tz-label">{{ opt.label }}</span>
              <span v-if="timezone === opt.value" class="tz-check">&#10003;</span>
            </div>
          </div>
        </n-popover>

        <!-- Search -->
        <button class="topbar-btn topbar-kbd" @click="openPalette" title="⌘K">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
          <kbd>⌘K</kbd>
        </button>

        <!-- Language -->
        <n-popselect :value="locale" :options="langOptions" trigger="click" @update:value="handleLangChange">
          <button class="topbar-btn">
            <n-icon :component="EarthOutline" :size="15" />
            <span>{{ locale === 'zh-CN' ? '中' : 'EN' }}</span>
          </button>
        </n-popselect>

        <!-- Theme -->
        <button class="topbar-btn" @click="toggleTheme" :title="isDark ? t('header.lightMode') : t('header.darkMode')">
          <n-icon :component="isDark ? SunnyOutline : MoonOutline" :size="16" />
        </button>
      </div>
    </header>

    <!-- Body -->
    <div class="app-body">
      <AppRail :active-app="activeApp" @switch="switchApp" />
      <AppSidebar
        :sections="menuSections"
        :active-key="activeMenuKey"
        :collapsed="sidebarCollapsed"
        @update:collapsed="sidebarCollapsed = $event"
        @navigate="handleNavigate"
      />
      <main class="main">
        <div class="main-header">
          <h1 class="main-title">{{ pageTitle }}</h1>
          <div class="main-actions">
            <slot name="actions" />
          </div>
        </div>
        <div class="main-content">
          <router-view />
        </div>
      </main>
    </div>

    <CommandPalette />
  </div>
</template>

<style scoped>
.app-shell { display: flex; flex-direction: column; height: 100vh; overflow: hidden; }

/* Top Bar */
.topbar {
  display: flex; align-items: center; justify-content: space-between;
  height: var(--sre-topbar-h); padding: 0 16px; flex-shrink: 0;
  background: var(--sre-bg-base);
  border-bottom: 1px solid var(--sre-border);
  z-index: var(--sre-z-sticky);
}

.topbar-start { display: flex; align-items: center; gap: 0; }
.topbar-end { display: flex; align-items: center; gap: 4px; }

.topbar-logo {
  display: flex; align-items: center; gap: 8px; text-decoration: none;
  padding: 4px 8px 4px 0; border-radius: var(--sre-radius-sm);
  transition: opacity var(--sre-duration-fast) var(--sre-ease-out);
}
.topbar-logo:hover { opacity: 0.8; }
.logo-img { width: 24px; height: 24px; border-radius: var(--sre-radius-sm); }
.logo-label { font-size: 15px; font-weight: 600; color: var(--sre-text-primary); letter-spacing: -0.01em; white-space: nowrap; }

.topbar-btn {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 6px 9px; min-height: 32px; border: none; border-radius: var(--sre-radius-sm);
  background: transparent; color: var(--sre-text-secondary);
  font-size: 12px; font-weight: 500; font-family: var(--sre-font-sans);
  cursor: pointer; white-space: nowrap;
  transition: background var(--sre-duration-fast) var(--sre-ease-out),
             color var(--sre-duration-fast) var(--sre-ease-out);
}
.topbar-btn:hover { background: var(--sre-bg-hover); color: var(--sre-text-primary); }

.topbar-clock {
  border: 1px solid var(--sre-border); border-radius: var(--sre-radius-pill);
  padding: 5px 11px; gap: 6px; background: var(--sre-bg-card);
}
.topbar-clock:hover, .topbar-clock.active {
  background: var(--sre-primary-soft); border-color: var(--sre-primary-ring);
}
.clock-text { font-family: var(--sre-font-mono); font-size: 13px; font-weight: 600; color: var(--sre-text-primary); font-feature-settings: "tnum" 1; }
.clock-tz { font-size: 10px; font-weight: 700; color: var(--sre-primary); background: var(--sre-primary-soft); padding: 1px 6px; border-radius: 4px; }

.topbar-kbd kbd { font-size: 10px; padding: 1px 5px; border-radius: 4px; background: var(--sre-bg-elevated); border: 1px solid var(--sre-border-strong); color: var(--sre-text-muted); font-family: var(--sre-font-mono); }

/* Timezone Panel */
.tz-panel { min-width: 220px; padding: 4px 0; }
.tz-panel-title { display: flex; align-items: center; gap: 8px; padding: 8px 16px 6px; font-size: 11px; font-weight: 600; color: var(--sre-text-tertiary); letter-spacing: 0.06em; text-transform: uppercase; border-bottom: 1px solid var(--sre-border); margin-bottom: 4px; }
.tz-item { display: flex; align-items: center; gap: 8px; padding: 7px 16px; cursor: pointer; font-size: 13px; color: var(--sre-text-primary); transition: background var(--sre-duration-fast) var(--sre-ease-out); margin: 0 4px; border-radius: 6px; }
.tz-item:hover { background: var(--sre-bg-hover); }
.tz-item.selected { color: var(--sre-primary); background: var(--sre-primary-soft); }
.tz-abbr { font-weight: 700; font-size: 11px; width: 32px; color: var(--sre-primary); flex-shrink: 0; }
.tz-label { flex: 1; }
.tz-check { font-weight: 700; color: var(--sre-primary); font-size: 12px; }

/* Body */
.app-body { display: flex; flex: 1; min-height: 0; }

/* Main */
.main { flex: 1; min-width: 0; display: flex; flex-direction: column; overflow: hidden; }

.main-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 20px; flex-shrink: 0;
  border-bottom: 1px solid var(--sre-border);
  background: var(--sre-bg-base);
}
.main-title {
  font-size: 18px; font-weight: 600; color: var(--sre-text-primary);
  letter-spacing: -0.01em; margin: 0; line-height: 1.2;
}
.main-actions { display: flex; align-items: center; gap: 8px; }
.main-content { flex: 1; overflow-y: auto; padding: 20px; }
</style>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/layouts/AppShell.vue
git commit -m "feat(nav): add AppShell layout — three-column shell (rail + sidebar + content)"
```

---

### Task 5: Restructure Routes

**Files:**
- Modify: `web/src/router/index.ts`

Replace the flat route list with app-prefixed routes. Add backward-compatible redirects for all old routes.

- [ ] **Step 1: Read the current router file**

Read `web/src/router/index.ts` to understand the full current structure.

- [ ] **Step 2: Replace the route definitions**

Replace the authenticated routes (children of the root `/` path) with the new app-prefixed structure. The key changes:

1. Change the root layout component from `MainLayout` to `AppShell`
2. Add new routes under `/oncall/`, `/alert/`, `/platform/` prefixes
3. Add redirect entries for all old routes
4. Keep `meta.requiresRole` for permission-gated routes

The new route structure:

```ts
{
  path: '/',
  component: () => import('@/layouts/AppShell.vue'),
  meta: { requiresAuth: true },
  children: [
    // Root redirect
    { path: '', redirect: '/oncall/overview' },

    // ===== On-Call =====
    { path: 'oncall', redirect: '/oncall/overview' },
    {
      path: 'oncall/overview',
      component: () => import('@/pages/dashboard/IncidentDashboard.vue'),
      meta: { title: '概览' },
    },
    {
      path: 'oncall/spaces',
      component: () => import('@/pages/channels/Index.vue'),
      meta: { title: '协作空间', requiresRole: ['admin', 'team_lead', 'member'] },
    },
    {
      path: 'oncall/spaces/:id',
      component: () => import('@/pages/channels/Detail.vue'),
      meta: { title: '协作空间', requiresRole: ['admin', 'team_lead', 'member'] },
    },
    {
      path: 'oncall/incidents',
      component: () => import('@/pages/incidents/Index.vue'),
      meta: { title: '故障列表' },
    },
    {
      path: 'oncall/incidents/:id',
      component: () => import('@/pages/incidents/Detail.vue'),
      meta: { title: '故障详情' },
    },
    {
      path: 'oncall/status-page',
      component: () => import('@/pages/oncall/StatusPage.vue'),
      meta: { title: '状态页面' },
    },
    {
      path: 'oncall/postmortems',
      component: () => import('@/pages/incidents/PostMortems.vue'),
      meta: { title: '故障复盘' },
    },
    {
      path: 'oncall/integrations',
      component: () => import('@/pages/integrations/Index.vue'),
      meta: { title: '集成中心', requiresRole: ['admin', 'team_lead'] },
    },
    {
      path: 'oncall/schedule',
      component: () => import('@/pages/schedule/Index.vue'),
      meta: { title: '值班管理', requiresRole: ['admin', 'team_lead'] },
    },
    {
      path: 'oncall/config/notify-rules',
      component: () => import('@/pages/notification/Rules.vue'),
      meta: { title: '通道通知规则' },
    },
    {
      path: 'oncall/config/routing-rules',
      component: () => import('@/pages/integrations/RoutingRules.vue'),
      meta: { title: '路由规则' },
    },
    {
      path: 'oncall/config/biz-groups',
      component: () => import('@/pages/settings/BizGroups.vue'),
      meta: { title: '业务分组' },
    },
    {
      path: 'oncall/config/subscribe-rules',
      component: () => import('@/pages/notification/Subscribe.vue'),
      meta: { title: '订阅规则' },
    },

    // ===== Alert =====
    { path: 'alert', redirect: '/alert/overview' },
    {
      path: 'alert/overview',
      component: () => import('@/pages/dashboard/Index.vue'),
      meta: { title: '概览' },
    },
    {
      path: 'alert/rules',
      component: () => import('@/pages/alerts/rules/Index.vue'),
      meta: { title: '告警规则' },
    },
    {
      path: 'alert/events',
      component: () => import('@/pages/alerts/events/Index.vue'),
      meta: { title: '活跃告警' },
    },
    {
      path: 'alert/events/:id',
      component: () => import('@/pages/alerts/events/Detail.vue'),
      meta: { title: '告警详情' },
    },
    {
      path: 'alert/history',
      component: () => import('@/pages/alerts/history/Index.vue'),
      meta: { title: '告警历史' },
    },
    {
      path: 'alert/suppression',
      component: () => import('@/pages/alerts/mute/Index.vue'),
      meta: { title: '静默规则' },
    },
    {
      path: 'alert/suppression/inhibition',
      component: () => import('@/pages/alerts/inhibition/Index.vue'),
      meta: { title: '抑制规则' },
    },
    {
      path: 'alert/datasources',
      component: () => import('@/pages/datasources/Index.vue'),
      meta: { title: '数据源' },
    },
    {
      path: 'alert/explore',
      component: () => import('@/pages/explore/Index.vue'),
      meta: { title: '数据查询' },
    },
    {
      path: 'alert/dashboards',
      component: () => import('@/pages/dashboard-v2/Index.vue'),
      meta: { title: '仪表盘' },
    },
    {
      path: 'alert/dashboards/:id',
      component: () => import('@/pages/dashboard-v2/View.vue'),
      meta: { title: '仪表盘' },
    },
    {
      path: 'alert/notify/policies',
      component: () => import('@/pages/notification/Index.vue'),
      meta: { title: '通知策略' },
    },
    {
      path: 'alert/notify/templates',
      component: () => import('@/pages/notification/Templates.vue'),
      meta: { title: '消息模板' },
    },
    {
      path: 'alert/notify/channels',
      component: () => import('@/pages/notification/Media.vue'),
      meta: { title: '通知渠道' },
    },
    {
      path: 'alert/notify/subscriptions',
      component: () => import('@/pages/notification/Subscribe.vue'),
      meta: { title: '订阅管理' },
    },

    // ===== Platform =====
    { path: 'platform', redirect: '/platform/profile' },
    {
      path: 'platform/profile',
      component: () => import('@/pages/platform/Profile.vue'),
      meta: { title: '个人中心' },
    },
    {
      path: 'platform/org/members',
      component: () => import('@/pages/settings/TeamManagement.vue'),
      meta: { title: '成员管理', requiresRole: ['admin'] },
    },
    {
      path: 'platform/org/teams',
      component: () => import('@/pages/settings/TeamManagement.vue'),
      meta: { title: '团队管理', requiresRole: ['admin', 'team_lead'] },
    },
    {
      path: 'platform/org/roles',
      component: () => import('@/pages/platform/Roles.vue'),
      meta: { title: '角色权限' },
    },
    {
      path: 'platform/org/sso',
      component: () => import('@/pages/settings/SSO.vue'),
      meta: { title: '单点登录', requiresRole: ['admin'] },
    },
    {
      path: 'platform/audit',
      component: () => import('@/pages/settings/AuditLogs.vue'),
      meta: { title: '审计日志', requiresRole: ['admin'] },
    },
    {
      path: 'platform/settings/smtp',
      component: () => import('@/pages/settings/SMTP.vue'),
      meta: { title: '邮件服务', requiresRole: ['admin'] },
    },
    {
      path: 'platform/settings/lark',
      component: () => import('@/pages/settings/LarkBot.vue'),
      meta: { title: '飞书机器人', requiresRole: ['admin'] },
    },
    {
      path: 'platform/settings/ai',
      component: () => import('@/pages/settings/AI.vue'),
      meta: { title: 'AI 配置', requiresRole: ['admin'] },
    },
    {
      path: 'platform/settings/security',
      component: () => import('@/pages/settings/Security.vue'),
      meta: { title: '安全设置', requiresRole: ['admin'] },
    },

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
}
```

- [ ] **Step 3: Verify the router compiles**

Run: `cd web && npx vue-tsc --noEmit`
Expected: May have errors for missing page components (StatusPage, Roles, Profile, etc.). Create stub pages in Task 6.

- [ ] **Step 4: Commit**

```bash
git add web/src/router/index.ts
git commit -m "feat(nav): restructure routes under /oncall, /alert, /platform prefixes with legacy redirects"
```

---

### Task 6: Create Stub Pages for New Routes

**Files:**
- Create: `web/src/pages/oncall/StatusPage.vue`
- Create: `web/src/pages/incidents/PostMortems.vue`
- Create: `web/src/pages/platform/Profile.vue`
- Create: `web/src/pages/platform/Roles.vue`
- Create: `web/src/pages/settings/SSO.vue`
- Create: `web/src/pages/settings/SMTP.vue`
- Create: `web/src/pages/settings/LarkBot.vue`
- Create: `web/src/pages/settings/AI.vue`
- Create: `web/src/pages/settings/Security.vue`
- Create: `web/src/pages/settings/AuditLogs.vue`
- Create: `web/src/pages/settings/BizGroups.vue`
- Create: `web/src/pages/notification/Templates.vue`
- Create: `web/src/pages/notification/Rules.vue`
- Create: `web/src/pages/integrations/RoutingRules.vue`

These are stub pages that either import existing components or show a placeholder. Most of these pages already exist under different paths — they just need to be re-exported or created as thin wrappers.

- [ ] **Step 1: Create stub pages for genuinely new pages**

For pages that don't exist yet (StatusPage, Roles), create minimal stubs:

```vue
<!-- web/src/pages/oncall/StatusPage.vue -->
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
</script>

<template>
  <div class="page-container">
    <div class="content-card" style="text-align: center; padding: 80px 20px;">
      <h2 style="color: var(--sre-text-primary); margin-bottom: 8px;">状态页面</h2>
      <p style="color: var(--sre-text-secondary);">公开/内部服务状态页 — 即将上线</p>
    </div>
  </div>
</template>
```

```vue
<!-- web/src/pages/platform/Roles.vue -->
<script setup lang="ts">
</script>

<template>
  <div class="page-container">
    <div class="content-card" style="text-align: center; padding: 80px 20px;">
      <h2 style="color: var(--sre-text-primary); margin-bottom: 8px;">角色权限</h2>
      <p style="color: var(--sre-text-secondary);">RBAC 权限矩阵 — 即将上线</p>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Create re-export pages for existing functionality**

For pages that already exist but need to be accessible from new routes, create thin wrapper components:

```vue
<!-- web/src/pages/platform/Profile.vue -->
<script setup lang="ts">
// Profile page — wraps existing user profile functionality
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { authApi, userNotifyConfigApi } from '@/api'
import { NForm, NFormItem, NInput, NButton, NTabs, NTabPane, NMessage } from 'naive-ui'
import type { UserNotifyConfig } from '@/types'

const { t } = useI18n()
const authStore = useAuthStore()
const message = useMessage()

const profileForm = ref({ display_name: '', email: '', phone: '' })
const saving = ref(false)

onMounted(() => {
  profileForm.value = {
    display_name: authStore.user?.display_name || '',
    email: authStore.user?.email || '',
    phone: authStore.user?.phone || '',
  }
})

async function saveProfile() {
  saving.value = true
  try {
    await authApi.updateMe(profileForm.value)
    await authStore.fetchProfile()
    message.success(t('profile.saved'))
  } catch (err: any) {
    message.error(err.message || t('common.failed'))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="page-container">
    <div class="content-card">
      <h3 style="margin: 0 0 16px; color: var(--sre-text-primary);">基本信息</h3>
      <n-form label-placement="top" size="small">
        <n-form-item :label="t('auth.username')">
          <n-input :value="authStore.user?.username" disabled />
        </n-form-item>
        <n-form-item :label="t('settings.displayName')">
          <n-input v-model:value="profileForm.display_name" />
        </n-form-item>
        <n-form-item :label="t('settings.email')">
          <n-input v-model:value="profileForm.email" />
        </n-form-item>
        <n-form-item :label="t('settings.phone')">
          <n-input v-model:value="profileForm.phone" placeholder="+86 138..." />
        </n-form-item>
      </n-form>
      <div style="margin-top: 16px;">
        <n-button type="primary" :loading="saving" @click="saveProfile">{{ t('common.save') }}</n-button>
      </div>
    </div>
  </div>
</template>
```

For other pages that are just re-exports of existing functionality, create simple redirect pages or import the existing component:

```vue
<!-- web/src/pages/settings/SSO.vue -->
<script setup lang="ts">
// Re-uses the settings page's OIDC section
// In Phase 2, this will be a standalone page
import SettingsIndex from '@/pages/settings/Index.vue'
</script>

<template>
  <SettingsIndex />
</template>
```

**Note:** For pages like SMTP, LarkBot, AI, Security that are currently part of the monolithic Settings page, the simplest Phase 1 approach is to render the full Settings page and let the user navigate within it. Phase 2 will split these into standalone pages.

- [ ] **Step 3: Verify all routes resolve**

Run: `cd web && npx vue-tsc --noEmit`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/
git commit -m "feat(nav): create stub and wrapper pages for new route structure"
```

---

### Task 7: Update Command Palette

**Files:**
- Modify: `web/src/composables/useCommandPalette.ts`

Update the hardcoded route list to use new paths.

- [ ] **Step 1: Read the current file**

Read `web/src/composables/useCommandPalette.ts` to find the route registration.

- [ ] **Step 2: Update route paths**

Replace all old route paths with new app-prefixed paths:

```ts
// Old paths → New paths
'/dashboard'           → '/oncall/overview'
'/channels'            → '/oncall/spaces'
'/incidents'           → '/oncall/incidents'
'/incident-dashboard'  → '/oncall/overview'
'/integrations'        → '/oncall/integrations'
'/schedule'            → '/oncall/schedule'
'/alerts/rules'        → '/alert/rules'
'/alerts/events'       → '/alert/events'
'/alerts/history'      → '/alert/history'
'/alerts/mute-rules'   → '/alert/suppression'
'/alerts/inhibition-rules' → '/alert/suppression/inhibition'
'/datasources'         → '/alert/datasources'
'/query'               → '/alert/explore'
'/notification'        → '/alert/notify/policies'
'/settings'            → '/platform/profile'
```

- [ ] **Step 3: Commit**

```bash
git add web/src/composables/useCommandPalette.ts
git commit -m "feat(nav): update command palette routes to new app-prefixed paths"
```

---

### Task 8: Update i18n Menu Keys

**Files:**
- Modify: `web/src/i18n/zh-CN.ts` (or equivalent locale file)
- Modify: `web/src/i18n/en.ts` (or equivalent locale file)

Add new menu translation keys for the restructured navigation.

- [ ] **Step 1: Read current locale files**

Find the i18n files and check the existing `menu.*` keys.

- [ ] **Step 2: Add new keys**

```ts
// Add to menu section
'menu.overview': '概览',
'menu.channels': '协作空间',
'menu.incidents': '故障列表',
'menu.statusPage': '状态页面',
'menu.postmortems': '故障复盘',
'menu.integrations': '集成中心',
'menu.schedule': '值班管理',
'menu.notifyPolicies': '通知策略',
'menu.notifyTemplates': '消息模板',
'menu.notifyChannels': '通知渠道',
'menu.subscriptions': '订阅管理',
'menu.profile': '个人中心',
'menu.members': '成员管理',
'menu.teams': '团队管理',
'menu.roles': '角色权限',
'menu.sso': '单点登录',
'menu.audit': '审计日志',
'menu.smtp': '邮件服务',
'menu.larkBot': '飞书机器人',
'menu.aiConfig': 'AI 配置',
'menu.security': '安全设置',
```

- [ ] **Step 3: Commit**

```bash
git add web/src/i18n/
git commit -m "feat(nav): add i18n keys for new navigation structure"
```

---

### Task 9: Delete MainLayout.vue and Verify

**Files:**
- Delete: `web/src/layouts/MainLayout.vue`

- [ ] **Step 1: Verify no other files import MainLayout**

```bash
grep -r "MainLayout" web/src/ --include="*.vue" --include="*.ts"
```

Expected: Only the router file should reference it (which we already changed to AppShell).

- [ ] **Step 2: Delete MainLayout.vue**

```bash
rm web/src/layouts/MainLayout.vue
```

- [ ] **Step 3: Verify typecheck passes**

```bash
cd web && npx vue-tsc --noEmit
```

- [ ] **Step 4: Verify build passes**

```bash
cd web && npx vite build
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(nav): remove MainLayout.vue, replaced by AppShell + AppRail + AppSidebar"
```

---

### Task 10: Version Bump and Tag

**Files:**
- Modify: `CLAUDE.md`
- Modify: `MODULES.md`
- Modify: `web/package.json`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Bump version to v4.0.0**

Update version in all files.

- [ ] **Step 2: Add CHANGELOG entry**

```markdown
## [v4.0.0] — 2026-05-XX

### Changed — 应用架构重构：三栏布局 + 三应用分区

**导航重构：**
- 顶部 Tab（Monitor/Incident/System）改为左侧图标栏（On-Call/Alert/Platform）
- 新增三栏布局：图标栏（48px）+ 菜单栏（220px）+ 内容区
- 菜单栏支持多级折叠
- 个人中心从弹窗改为独立页面

**路由重组：**
- 新增 `/oncall/` 前缀：故障响应相关页面
- 新增 `/alert/` 前缀：告警引擎 + 通知管道页面
- 新增 `/platform/` 前缀：平台管理页面
- 所有旧路由添加向后兼容重定向

**新增页面：**
- 状态页面（On-Call）— 即将上线
- 角色权限（Platform）— 即将上线
- 个人中心（Platform）— 独立页面替代弹窗
```

- [ ] **Step 3: Commit, tag, push**

```bash
git add CLAUDE.md MODULES.md web/package.json CHANGELOG.md
git commit -m "chore: bump version to v4.0.0"
git tag v4.0.0
git push origin main --tags
```
