<script setup lang="ts">
import { ref, computed, watch, h, inject, onMounted, onUnmounted } from 'vue'
import CommandPalette from '@/components/common/CommandPalette.vue'
import { useCommandPalette } from '@/composables/useCommandPalette'
import type { Ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NIcon, useMessage } from 'naive-ui'
import type { MenuOption, DropdownOption } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import { userNotifyConfigApi, authApi } from '@/api'
import type { UserNotifyConfig } from '@/types'
import {
  GridOutline,
  ServerOutline,
  AlertCircleOutline,
  CalendarOutline,
  SettingsOutline,
  LogOutOutline,
  NotificationsOutline,
  SunnyOutline,
  MoonOutline,
  ChevronDownOutline,
  PersonOutline,
  LockClosedOutline,
  EarthOutline,
  TimeOutline,
  ChevronBackOutline,
  ChevronForwardOutline,
  SearchOutline,
  LayersOutline,
  BugOutline,
  FlashOutline,
  GitNetworkOutline,
  BarChartOutline,
} from '@vicons/ionicons5'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { t, locale } = useI18n()
const message = useMessage()

// Sidebar collapse state
const collapsed = ref(JSON.parse(localStorage.getItem('sre-sider-collapsed') ?? 'false'))
watch(collapsed, v => localStorage.setItem('sre-sider-collapsed', JSON.stringify(v)))

function toggleCollapsed() {
  collapsed.value = !collapsed.value
}

const { open: openPalette } = useCommandPalette()
const appVersion = __APP_VERSION__

const isDark = inject<Ref<boolean>>('isDark', ref(true))
const toggleTheme = inject<() => void>('toggleTheme', () => {})

onMounted(() => {
  if (authStore.isLoggedIn && !authStore.user) {
    authStore.fetchProfile()
  }
})

// ===== Clock =====
const timeDisplay = ref('')
const dateDisplay = ref('')
const timezone = ref(localStorage.getItem('sre-timezone') || 'Asia/Shanghai')
const showTzPanel = ref(false)

const timezoneOptions = [
  { label: 'Asia/Shanghai', abbr: 'CST', value: 'Asia/Shanghai' },
  { label: 'UTC',           abbr: 'UTC', value: 'UTC' },
  { label: 'Asia/Tokyo',    abbr: 'JST', value: 'Asia/Tokyo' },
  { label: 'Asia/Singapore',abbr: 'SGT', value: 'Asia/Singapore' },
  { label: 'Europe/London', abbr: 'GMT', value: 'Europe/London' },
  { label: 'America/New_York', abbr: 'EST', value: 'America/New_York' },
  { label: 'America/Los_Angeles', abbr: 'PST', value: 'America/Los_Angeles' },
]

const tzAbbr = computed(() => {
  return timezoneOptions.find(o => o.value === timezone.value)?.abbr || timezone.value.split('/').pop() || 'TZ'
})

function updateClock() {
  const now = new Date()
  timeDisplay.value = now.toLocaleTimeString('en-GB', {
    timeZone: timezone.value,
    hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false,
  })
  dateDisplay.value = now.toLocaleDateString(locale.value === 'zh-CN' ? 'zh-CN' : 'en-US', {
    timeZone: timezone.value,
    year: 'numeric', month: 'short', day: '2-digit',
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

// ===== Menu — 3 collapsible parent menus (no type:'group') =====
function renderIcon(icon: any) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions = computed<MenuOption[]>(() => {
  const systemChildren: MenuOption[] = [
    { label: t('menu.integrations'), key: '/integrations', icon: renderIcon(GitNetworkOutline) },
    { label: t('menu.notification'), key: '/notification', icon: renderIcon(NotificationsOutline) },
    { label: t('menu.schedule'),     key: '/schedule',     icon: renderIcon(CalendarOutline) },
  ]
  if (authStore.canManage) {
    systemChildren.push({ label: t('menu.settings'), key: '/settings', icon: renderIcon(SettingsOutline) })
  }
  return [
    {
      label: t('menu.alertCenter'),
      key: 'alert-center',
      icon: renderIcon(AlertCircleOutline),
      children: [
        { label: t('menu.dashboard'),      key: '/dashboard',          icon: renderIcon(GridOutline) },
        { label: t('menu.alertRules'),      key: '/alerts/rules' },
        { label: t('menu.activeAlerts'),    key: '/alerts/events' },
        { label: t('menu.alertHistory'),    key: '/alerts/history' },
        { label: t('menu.muteRules'),       key: '/alerts/mute-rules' },
        { label: t('menu.inhibitionRules'), key: '/alerts/inhibition-rules' },
        { label: t('menu.datasources'),     key: '/datasources',       icon: renderIcon(ServerOutline) },
        { label: t('menu.dataQuery'),       key: '/query',             icon: renderIcon(SearchOutline) },
      ],
    },
    {
      label: t('menu.incidentMgmt'),
      key: 'incident-mgmt',
      icon: renderIcon(BugOutline),
      children: [
        { label: t('menu.incidentDashboard'), key: '/incident-dashboard', icon: renderIcon(BarChartOutline) },
        { label: t('menu.channels'),          key: '/channels',           icon: renderIcon(LayersOutline) },
        { label: t('menu.incidents'),         key: '/incidents',          icon: renderIcon(BugOutline) },
        { label: t('menu.alertsV2'),          key: '/alerts-v2',          icon: renderIcon(FlashOutline) },
      ],
    },
    {
      label: t('menu.systemConfig'),
      key: 'system-config',
      icon: renderIcon(SettingsOutline),
      children: systemChildren,
    },
  ]
})

// Expand all parent menus by default; user can collapse individual ones
const expandedKeys = ref<string[]>(['alert-center', 'incident-mgmt', 'system-config'])

function resolveActiveKey(p: string): string {
  if (p.startsWith('/incident-dashboard'))        return '/incident-dashboard'
  if (p.startsWith('/channels'))                  return '/channels'
  if (p.startsWith('/incidents'))                 return '/incidents'
  if (p.startsWith('/alerts-v2'))                 return '/alerts-v2'
  if (p.startsWith('/integrations'))              return '/integrations'
  if (p.startsWith('/query'))                     return '/query'
  if (p.startsWith('/datasources'))               return '/datasources'
  if (p.startsWith('/alerts/rules'))              return '/alerts/rules'
  if (p.startsWith('/alerts/events'))             return '/alerts/events'
  if (p.startsWith('/alerts/history'))            return '/alerts/history'
  if (p.startsWith('/alerts/mute-rules'))         return '/alerts/mute-rules'
  if (p.startsWith('/alerts/inhibition-rules'))   return '/alerts/inhibition-rules'
  if (p.startsWith('/notification'))              return '/notification'
  if (p.startsWith('/schedule'))                  return '/schedule'
  return p
}

const menuSelectedKey = ref(resolveActiveKey(route.path))
watch(
  () => route.path,
  (p) => { menuSelectedKey.value = resolveActiveKey(p) },
)

function handleMenuClick(key: string) {
  menuSelectedKey.value = ''
  router.push(key)
}

// ===== Language =====
const langOptions = [
  { label: '简体中文', value: 'zh-CN' },
  { label: 'English',  value: 'en' },
]
function handleLangChange(val: string) { locale.value = val; localStorage.setItem('locale', val) }

// ===== User (sidebar bottom) =====
const userDropdownOptions = computed<DropdownOption[]>(() => [
  { label: t('header.profile'),        key: 'profile',  icon: renderIcon(PersonOutline) },
  { label: t('header.changePassword'), key: 'password', icon: renderIcon(LockClosedOutline) },
  { type: 'divider', key: 'd1' },
  { label: t('header.logout'),         key: 'logout',   icon: renderIcon(LogOutOutline) },
])

async function handleUserDropdown(key: string) {
  if (key === 'logout') {
    authStore.logout()
    router.push('/login')
  } else if (key === 'profile') {
    profileTab.value = 'info'
    await openProfileModal()
  } else if (key === 'password') {
    profileTab.value = 'password'
    await openProfileModal()
  }
}

const userInitial  = computed(() => (authStore.user?.display_name || authStore.user?.username || 'U').charAt(0).toUpperCase())
const displayName  = computed(() => authStore.user?.display_name || authStore.user?.username || 'User')

function isImageAvatar(v: string | undefined | null): boolean {
  if (!v) return false
  return v.startsWith('data:image/') || v.startsWith('http://') || v.startsWith('https://') || v.startsWith('/')
}
const headerAvatar = computed(() => authStore.user?.avatar || '')
const headerAvatarIsImage = computed(() => isImageAvatar(headerAvatar.value))

// ===== Breadcrumb =====
const pageTitle = computed(() => {
  const p = route.path
  if (p === '/dashboard')                         return t('menu.dashboard')
  if (p === '/datasources')                       return t('menu.datasources')
  if (p.startsWith('/query'))                      return t('menu.dataQuery')
  if (p.startsWith('/alerts/rules'))              return t('menu.alertRules')
  if (p.startsWith('/alerts/events'))             return t('menu.activeAlerts')
  if (p.startsWith('/alerts/history'))            return t('menu.alertHistory')
  if (p.startsWith('/alerts/mute-rules'))         return t('menu.muteRules')
  if (p.startsWith('/alerts/inhibition-rules'))   return t('menu.inhibitionRules')
  if (p.startsWith('/notification'))              return t('menu.notification')
  if (p === '/schedule')                          return t('menu.schedule')
  if (p === '/settings')                          return t('menu.settings')
  return ''
})

// ===== Profile Modal =====
const showProfileModal = ref(false)
const profileTab = ref('info')
const profileSaving = ref(false)

const profileForm = ref({ display_name: '', email: '', phone: '', avatar: '' })

const presetAvatars = [
  '👤','🧑‍💻','👩‍💻','🧑‍🔧','👩‍🔧','🧑‍🚀','👩‍🚀','🧑‍🔬','👩‍🔬',
  '🧑‍💼','👩‍💼','🧑‍🎤','🧑‍🎨','🦊','🐺','🐧','🦅','🦁','🐯','🐻',
  '🐼','🦉','🦄','🐉','🤖','👾','🛰️','🚀','⚡','🔥','🌟','🌈',
]

const AVATAR_MAX_BYTES = 200 * 1024
const avatarFileInput = ref<HTMLInputElement | null>(null)

function triggerAvatarUpload() {
  avatarFileInput.value?.click()
}

function onAvatarFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!/^image\/(png|jpe?g|svg\+xml|webp)$/.test(file.type)) {
    message.error(t('profile.avatarInvalidType'))
    input.value = ''
    return
  }
  if (file.size > AVATAR_MAX_BYTES) {
    message.error(t('profile.avatarTooLarge'))
    input.value = ''
    return
  }
  const reader = new FileReader()
  reader.onload = () => { profileForm.value.avatar = String(reader.result || '') }
  reader.onerror = () => message.error(t('common.failed'))
  reader.readAsDataURL(file)
  input.value = ''
}

const pwdForm = ref({ old_password: '', new_password: '', confirm_password: '' })
const pwdError = ref('')

const userNotifyConfigs = ref<UserNotifyConfig[]>([])
const newNotifyConfig = ref<{ media_type: 'lark_personal' | 'email' | 'webhook'; config: string }>({ media_type: 'lark_personal', config: '' })

const mediaTypeOptions = computed(() => [
  { label: t('profile.larkPersonal'), value: 'lark_personal' },
  { label: t('profile.email'),        value: 'email' },
  { label: t('profile.webhook'),      value: 'webhook' },
])

const configHint = computed(() => {
  switch (newNotifyConfig.value.media_type) {
    case 'lark_personal': return t('profile.larkUserIdHint')
    case 'email':         return t('profile.emailHint')
    case 'webhook':       return t('profile.webhookHint')
    default:              return ''
  }
})

async function openProfileModal() {
  profileSaving.value = false
  pwdError.value = ''
  pwdForm.value = { old_password: '', new_password: '', confirm_password: '' }
  profileForm.value = {
    display_name: authStore.user?.display_name || '',
    email:        authStore.user?.email || '',
    phone:        authStore.user?.phone || '',
    avatar:       authStore.user?.avatar || '',
  }
  try {
    const cfgs = await userNotifyConfigApi.list()
    userNotifyConfigs.value = cfgs.data.data || []
  } catch {
    userNotifyConfigs.value = []
  }
  newNotifyConfig.value = { media_type: 'lark_personal', config: '' }
  showProfileModal.value = true
}

async function saveProfile() {
  profileSaving.value = true
  try {
    await authApi.updateMe(profileForm.value)
    await authStore.fetchProfile()
    message.success(t('profile.saved'))
  } catch (err: any) {
    message.error(err.message || t('common.failed'))
  } finally {
    profileSaving.value = false
  }
}

async function savePassword() {
  pwdError.value = ''
  if (pwdForm.value.new_password !== pwdForm.value.confirm_password) {
    pwdError.value = t('profile.passwordMismatch')
    return
  }
  profileSaving.value = true
  try {
    await authApi.changeMyPassword({ old_password: pwdForm.value.old_password, new_password: pwdForm.value.new_password })
    message.success(t('profile.passwordChanged'))
    pwdForm.value = { old_password: '', new_password: '', confirm_password: '' }
  } catch (err: any) {
    message.error(err.message || t('common.failed'))
  } finally {
    profileSaving.value = false
  }
}

async function addNotifyConfig() {
  if (!newNotifyConfig.value.config) return
  profileSaving.value = true
  try {
    await userNotifyConfigApi.upsert({ ...newNotifyConfig.value, is_enabled: true })
    const cfgs = await userNotifyConfigApi.list()
    userNotifyConfigs.value = cfgs.data.data || []
    newNotifyConfig.value = { media_type: 'lark_personal', config: '' }
    message.success(t('profile.notifyConfigSaved'))
  } catch (err: any) {
    message.error(err.message || t('common.failed'))
  } finally {
    profileSaving.value = false
  }
}

const larkOpenIdInput = ref('')
const larkBindSaving = ref(false)

async function saveLarkBind() {
  const openId = larkOpenIdInput.value.trim()
  if (!openId) return
  larkBindSaving.value = true
  try {
    await authApi.bindLark(openId)
    message.success(t('settings.larkBindSuccess'))
    larkOpenIdInput.value = ''
  } catch (err: any) {
    message.error(err.message || t('common.failed'))
  } finally {
    larkBindSaving.value = false
  }
}

async function removeNotifyConfig(mediaType: string) {
  try {
    await userNotifyConfigApi.deleteByType(mediaType)
    userNotifyConfigs.value = userNotifyConfigs.value.filter(c => c.media_type !== mediaType)
  } catch (err: any) {
    message.error(err.message || t('common.failed'))
  }
}

async function toggleNotifyConfig(cfg: UserNotifyConfig, enabled: boolean) {
  try {
    await userNotifyConfigApi.upsert({ ...cfg, is_enabled: enabled })
  } catch (err: any) {
    message.error(err.message || t('common.failed'))
  }
}
</script>

<template>
  <n-layout has-sider style="height: 100vh">

    <!-- ===== Glass Sidebar ===== -->
    <n-layout-sider
      class="sre-sider"
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="232"
      :collapsed="collapsed"
      :native-scrollbar="false"
    >
      <!-- Logo -->
      <div class="sider-logo" :class="{ collapsed }">
        <img src="/logo.svg" alt="SREAgent" class="logo-mark" />
        <transition name="fade">
          <span v-if="!collapsed" class="logo-text">
            <span class="gradient-text">SRE</span>Agent
          </span>
        </transition>
      </div>

      <!-- Navigation — collapsible parent-child menu -->
      <n-menu
        class="sre-menu"
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :indent="18"
        :options="menuOptions"
        :value="menuSelectedKey"
        :expanded-keys="expandedKeys"
        @update:value="handleMenuClick"
        @update:expanded-keys="(ks: string[]) => { expandedKeys = ks }"
      />

      <!-- Spacer -->
      <div class="sider-spacer" />

      <!-- Bottom: user profile + collapse toggle + version -->
      <div class="sider-bottom">
        <!-- User pill (moved from header) -->
        <n-dropdown :options="userDropdownOptions" trigger="click" @select="handleUserDropdown">
          <div class="sider-user-pill" :class="{ collapsed }">
            <div class="user-avatar" :class="{ 'user-avatar--image': headerAvatarIsImage, 'user-avatar--emoji': !!headerAvatar && !headerAvatarIsImage }">
              <img v-if="headerAvatarIsImage" :src="headerAvatar" alt="avatar" />
              <template v-else-if="headerAvatar">{{ headerAvatar }}</template>
              <template v-else>{{ userInitial }}</template>
            </div>
            <transition name="fade">
              <div v-if="!collapsed" class="sider-user-info">
                <span class="sider-user-name">{{ displayName }}</span>
                <span class="sider-user-role">{{ authStore.canManage ? 'Admin' : 'Member' }}</span>
              </div>
            </transition>
            <transition name="fade">
              <n-icon v-if="!collapsed" :component="ChevronDownOutline" :size="12" class="sider-user-chevron" />
            </transition>
          </div>
        </n-dropdown>

        <!-- Collapse toggle -->
        <div
          class="sider-collapse-toggle"
          :class="{ collapsed }"
          :title="collapsed ? t('header.expandSidebar') : t('header.collapseSidebar')"
          @click="toggleCollapsed"
        >
          <n-icon
            :component="collapsed ? ChevronForwardOutline : ChevronBackOutline"
            :size="14"
            class="collapse-icon"
          />
          <transition name="fade">
            <span v-if="!collapsed" class="collapse-label">{{ t('header.collapseSidebar') }}</span>
          </transition>
        </div>

        <transition name="fade">
          <div v-if="!collapsed" class="sider-version">v{{ appVersion }}</div>
        </transition>
      </div>
    </n-layout-sider>

    <CommandPalette />

    <!-- ===== Right: header + content ===== -->
    <n-layout>

      <!-- Header Bar (glass) -->
      <div class="header-bar">
        <div class="header-left">
          <span class="header-page-title">{{ pageTitle }}</span>
        </div>

        <div class="header-right">
          <!-- Clock pill -->
          <n-popover
            v-model:show="showTzPanel"
            trigger="click"
            placement="bottom-end"
            :show-arrow="false"
            style="padding: 0"
          >
            <template #trigger>
              <div class="clock-pill" :class="{ active: showTzPanel }">
                <n-icon :component="TimeOutline" :size="13" class="clock-icon" />
                <span class="clock-time">{{ timeDisplay }}</span>
                <span class="clock-sep">·</span>
                <span class="clock-date">{{ dateDisplay }}</span>
                <span class="clock-tz">{{ tzAbbr }}</span>
              </div>
            </template>
            <div class="tz-panel">
              <div class="tz-panel-title">
                <n-icon :component="EarthOutline" :size="14" />
                {{ t('header.timezone') }}
              </div>
              <div
                v-for="opt in timezoneOptions"
                :key="opt.value"
                class="tz-option"
                :class="{ selected: timezone === opt.value }"
                @click="selectTimezone(opt.value)"
              >
                <span class="tz-opt-abbr">{{ opt.abbr }}</span>
                <span class="tz-opt-label">{{ opt.label }}</span>
                <span v-if="timezone === opt.value" class="tz-opt-check">✓</span>
              </div>
            </div>
          </n-popover>

          <div class="header-sep" />

          <!-- ⌘K -->
          <div class="ctrl-btn ctrl-btn--search" @click="openPalette" title="⌘K">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
            </svg>
            <kbd class="cmd-shortcut">⌘K</kbd>
          </div>

          <div class="header-sep" />

          <!-- Language -->
          <n-popselect
            :value="locale"
            :options="langOptions"
            trigger="click"
            :render-label="(opt: any) => opt.label"
            @update:value="handleLangChange"
          >
            <div class="ctrl-btn">
              <n-icon :component="EarthOutline" :size="15" />
              <span class="ctrl-label">{{ locale === 'zh-CN' ? '中' : 'EN' }}</span>
            </div>
          </n-popselect>

          <!-- Theme toggle -->
          <div class="ctrl-btn" @click="toggleTheme" :title="isDark ? t('header.lightMode') : t('header.darkMode')">
            <n-icon :component="isDark ? SunnyOutline : MoonOutline" :size="16" />
          </div>
        </div>
      </div>

      <!-- Main content -->
      <n-layout-content class="sre-content" :native-scrollbar="false">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>

  <!-- ===== Profile Modal ===== -->
  <n-modal
    v-model:show="showProfileModal"
    :title="t('profile.title')"
    preset="card"
    style="width: 500px"
    :bordered="false"
    :segmented="{ content: true }"
  >
    <n-tabs v-model:value="profileTab" type="line" animated>

      <n-tab-pane name="info" :tab="t('profile.tabInfo')">
        <div class="avatar-section">
          <div class="avatar-current">
            <img v-if="isImageAvatar(profileForm.avatar)" :src="profileForm.avatar" alt="avatar" class="avatar-preview-img" />
            <span v-else class="avatar-preview">{{ profileForm.avatar || userInitial }}</span>
          </div>
          <div class="avatar-actions">
            <div class="avatar-grid">
              <span
                v-for="a in presetAvatars"
                :key="a"
                class="avatar-option"
                :class="{ selected: profileForm.avatar === a }"
                @click="profileForm.avatar = a"
              >{{ a }}</span>
            </div>
            <div class="avatar-upload-row">
              <n-button size="tiny" secondary @click="triggerAvatarUpload">
                📎 {{ t('profile.uploadAvatar') }}
              </n-button>
              <n-button
                v-if="profileForm.avatar"
                size="tiny"
                quaternary
                type="error"
                @click="profileForm.avatar = ''"
              >
                {{ t('profile.clearAvatar') }}
              </n-button>
              <input
                ref="avatarFileInput"
                type="file"
                accept="image/png,image/jpeg,image/svg+xml,image/webp"
                style="display: none"
                @change="onAvatarFileChange"
              />
            </div>
          </div>
        </div>

        <n-form label-placement="top" size="small" style="margin-top: 16px">
          <n-form-item :label="t('auth.username')">
            <n-input :value="authStore.user?.username" disabled />
          </n-form-item>
          <n-form-item :label="t('settings.displayName')">
            <n-input v-model:value="profileForm.display_name" :placeholder="t('settings.displayName')" />
          </n-form-item>
          <n-form-item :label="t('settings.email')">
            <n-input v-model:value="profileForm.email" :placeholder="t('settings.email')" />
          </n-form-item>
          <n-form-item :label="t('settings.phone') || '手机号'">
            <n-input v-model:value="profileForm.phone" placeholder="+86 138..." />
          </n-form-item>
        </n-form>

        <div class="modal-footer">
          <n-button type="primary" :loading="profileSaving" @click="saveProfile">{{ t('common.save') }}</n-button>
        </div>
      </n-tab-pane>

      <n-tab-pane name="password" :tab="t('profile.tabPassword')">
        <n-form label-placement="top" size="small">
          <n-form-item :label="t('profile.oldPassword')">
            <n-input v-model:value="pwdForm.old_password" type="password" show-password-on="click" />
          </n-form-item>
          <n-form-item :label="t('profile.newPassword')">
            <n-input v-model:value="pwdForm.new_password" type="password" show-password-on="click" />
          </n-form-item>
          <n-form-item :label="t('profile.confirmPassword')">
            <n-input
              v-model:value="pwdForm.confirm_password"
              type="password"
              show-password-on="click"
              :status="pwdError ? 'error' : undefined"
            />
            <template #feedback>
              <span v-if="pwdError" style="color: var(--sre-critical)">{{ pwdError }}</span>
            </template>
          </n-form-item>
        </n-form>

        <div class="modal-footer">
          <n-button type="primary" :loading="profileSaving" @click="savePassword">{{ t('profile.changePassword') }}</n-button>
        </div>
      </n-tab-pane>

      <n-tab-pane name="notify" :tab="t('profile.tabNotify')">
        <div class="notify-config-list">
          <div v-for="cfg in userNotifyConfigs" :key="cfg.media_type" class="notify-config-item">
            <div class="notify-config-info">
              <n-tag size="small" :type="cfg.media_type === 'lark_personal' ? 'success' : cfg.media_type === 'email' ? 'info' : 'default'">
                {{ cfg.media_type === 'lark_personal' ? t('profile.larkPersonal') : cfg.media_type === 'email' ? t('profile.email') : t('profile.webhook') }}
              </n-tag>
              <span class="notify-config-value">{{ cfg.config }}</span>
              <n-switch v-model:value="cfg.is_enabled" size="small" @update:value="(v: boolean) => toggleNotifyConfig(cfg, v)" />
            </div>
            <n-button size="tiny" quaternary type="error" @click="removeNotifyConfig(cfg.media_type)">{{ t('common.remove') }}</n-button>
          </div>
          <n-empty v-if="userNotifyConfigs.length === 0" :description="t('profile.noNotifyConfig')" style="padding: 20px 0" />
        </div>

        <n-divider>{{ t('profile.addNotify') }}</n-divider>
        <n-form label-placement="top" size="small">
          <n-form-item :label="t('profile.mediaType')">
            <n-select v-model:value="newNotifyConfig.media_type" :options="mediaTypeOptions" />
          </n-form-item>
          <n-form-item :label="t('profile.configValue')">
            <n-input v-model:value="newNotifyConfig.config" :placeholder="configHint" clearable />
          </n-form-item>
        </n-form>
        <div class="modal-footer">
          <n-button type="primary" :loading="profileSaving" @click="addNotifyConfig">{{ t('profile.addNotify') }}</n-button>
        </div>
      </n-tab-pane>

      <n-tab-pane name="lark" :tab="t('settings.larkBind')">
        <n-space vertical size="large" style="padding: 8px 0">
          <n-alert type="info" :title="t('settings.larkBind')" style="font-size:13px">
            {{ t('settings.larkBindHint') }}
          </n-alert>
          <n-form label-placement="top" size="small">
            <n-form-item :label="t('settings.larkOpenId')">
              <n-input
                v-model:value="larkOpenIdInput"
                :placeholder="t('settings.larkOpenId')"
                clearable
                style="max-width: 360px"
              />
            </n-form-item>
          </n-form>
          <n-button type="primary" :loading="larkBindSaving" :disabled="!larkOpenIdInput.trim()" @click="saveLarkBind">
            {{ t('settings.larkBind') }}
          </n-button>
        </n-space>
      </n-tab-pane>

    </n-tabs>
  </n-modal>
</template>

<style scoped>
/* ===== Glass Sidebar ===== */
.sre-sider {
  background: var(--sre-glass-bg);
  backdrop-filter: saturate(var(--sre-glass-saturate)) blur(var(--sre-glass-blur));
  -webkit-backdrop-filter: saturate(var(--sre-glass-saturate)) blur(var(--sre-glass-blur));
  border-right: 1px solid var(--sre-glass-border);
  position: relative;
  transition: width 280ms var(--sre-ease-spring) !important;
}

.sre-sider::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image: var(--sre-noise-url);
  opacity: 0.025;
  mix-blend-mode: overlay;
  pointer-events: none;
  z-index: 0;
}

.sider-logo {
  display: flex;
  align-items: center;
  gap: var(--sre-space-3);
  padding: 16px 18px;
  height: 60px;
  border-bottom: 1px solid var(--sre-glass-border);
  position: relative;
  z-index: 1;
}
.sider-logo.collapsed {
  justify-content: center;
  padding: 16px 12px;
}

.logo-mark {
  width: 32px;
  height: 32px;
  border-radius: var(--sre-radius-md);
  flex-shrink: 0;
  display: block;
  filter: drop-shadow(0 4px 16px rgba(16, 185, 129, 0.45));
}
.logo-text {
  font-size: var(--sre-fs-lg);
  font-weight: var(--sre-fw-semibold);
  letter-spacing: -0.01em;
  white-space: nowrap;
  color: var(--sre-text-primary);
}

/* ===== Menu ===== */
.sre-menu {
  padding: var(--sre-space-2);
  position: relative;
  z-index: 1;
  flex: 0 1 auto;
}

/* Selected menu item — left accent bar */
.sre-menu :deep(.n-menu-item-content--selected)::before {
  content: '';
  position: absolute;
  left: 0;
  top: 6px;
  bottom: 6px;
  width: 3px;
  border-radius: 0 3px 3px 0;
  background: var(--sre-gradient-brand);
}
.sre-menu :deep(.n-menu-item-content) {
  position: relative;
  overflow: visible;
  padding: 6px 10px !important;
  min-height: 34px;
}

/* Spacer — pushes bottom section down */
.sider-spacer { flex: 1; }

/* ===== Sidebar Bottom ===== */
.sider-bottom {
  display: flex;
  flex-direction: column;
  gap: var(--sre-space-2);
  padding: var(--sre-space-3);
  border-top: 1px solid var(--sre-glass-border);
  background: transparent;
  z-index: 1;
}

/* User pill in sidebar */
.sider-user-pill {
  display: flex;
  align-items: center;
  gap: var(--sre-space-2);
  padding: 8px 10px;
  border-radius: var(--sre-radius-md);
  cursor: pointer;
  user-select: none;
  transition: background var(--sre-duration-fast) var(--sre-ease-out);
}
.sider-user-pill:hover {
  background: var(--sre-bg-hover);
}
.sider-user-pill.collapsed {
  justify-content: center;
  padding: 8px;
}

.sider-user-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}
.sider-user-name {
  font-size: var(--sre-fs-md);
  font-weight: var(--sre-fw-medium);
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.2;
}
.sider-user-role {
  font-size: var(--sre-fs-2xs);
  color: var(--sre-text-tertiary);
  line-height: 1.3;
}
.sider-user-chevron {
  color: var(--sre-text-tertiary);
  flex-shrink: 0;
  transition: transform var(--sre-duration-fast) var(--sre-ease-out);
}
.sider-user-pill:hover .sider-user-chevron {
  transform: translateY(1px);
}

.user-avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--sre-gradient-brand);
  color: #fff;
  font-size: var(--sre-fs-sm);
  font-weight: var(--sre-fw-bold);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  overflow: hidden;
  box-shadow: 0 2px 8px -2px rgba(16, 185, 129, 0.40),
              inset 0 1px 0 rgba(255,255,255,0.2);
}
.user-avatar--emoji {
  font-size: 16px;
  font-weight: 400;
  line-height: 1;
  background: transparent;
  box-shadow: inset 0 0 0 1px var(--sre-border);
}
.user-avatar--image {
  background: transparent;
  box-shadow: inset 0 0 0 1px var(--sre-border);
}
.user-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.sider-collapse-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: var(--sre-radius-sm);
  cursor: pointer;
  transition: background var(--sre-duration-fast) var(--sre-ease-out),
              color var(--sre-duration-fast) var(--sre-ease-out);
  color: var(--sre-text-tertiary);
  user-select: none;
  white-space: nowrap;
  overflow: hidden;
}
.sider-collapse-toggle.collapsed {
  justify-content: center;
}
.sider-collapse-toggle:hover {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
}
.collapse-icon {
  flex-shrink: 0;
  transition: transform var(--sre-duration-base) var(--sre-ease-spring);
}
.collapse-label {
  font-size: var(--sre-fs-xs);
  font-weight: var(--sre-fw-medium);
}

.sider-version {
  text-align: center;
  font-size: var(--sre-fs-2xs);
  color: var(--sre-text-tertiary);
  letter-spacing: 0.05em;
  padding: 2px 0 0;
  opacity: 0.6;
}

/* ===== Header bar (glass) ===== */
.header-bar {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--sre-space-6);
  border-bottom: 1px solid var(--sre-glass-border);
  background: var(--sre-glass-bg);
  backdrop-filter: saturate(var(--sre-glass-saturate)) blur(var(--sre-glass-blur));
  -webkit-backdrop-filter: saturate(var(--sre-glass-saturate)) blur(var(--sre-glass-blur));
  flex-shrink: 0;
  position: sticky;
  top: 0;
  z-index: var(--sre-z-sticky);
  transition: background var(--sre-duration-slow) var(--sre-ease-out),
              border-color var(--sre-duration-slow) var(--sre-ease-out);
}
.header-left {
  display: flex;
  align-items: center;
}
.header-page-title {
  font-size: var(--sre-fs-lg);
  font-weight: var(--sre-fw-semibold);
  color: var(--sre-text-primary);
  letter-spacing: -0.005em;
  transition: opacity var(--sre-duration-fast) var(--sre-ease-out),
              transform var(--sre-duration-fast) var(--sre-ease-out);
}
.header-right {
  display: flex;
  align-items: center;
  gap: var(--sre-space-1);
}

.sre-content {
  padding: var(--sre-space-6);
  background: transparent;
}

.header-sep {
  width: 1px;
  height: 18px;
  background: var(--sre-border);
  margin: 0 var(--sre-space-2);
  opacity: 0.7;
}

/* ===== Clock Pill ===== */
.clock-pill {
  display: flex;
  align-items: center;
  gap: var(--sre-space-2);
  padding: 6px 12px;
  border-radius: var(--sre-radius-pill);
  cursor: pointer;
  user-select: none;
  transition: background var(--sre-duration-fast) var(--sre-ease-out),
              border-color var(--sre-duration-fast) var(--sre-ease-out),
              box-shadow var(--sre-duration-fast) var(--sre-ease-out);
  border: 1px solid var(--sre-border);
  background: var(--sre-bg-sunken);
}
.clock-pill:hover,
.clock-pill.active {
  background: var(--sre-primary-soft);
  border-color: var(--sre-primary-ring);
  box-shadow: 0 0 0 3px var(--sre-primary-soft);
}
.clock-icon {
  color: var(--sre-primary);
  flex-shrink: 0;
}
.clock-time {
  font-family: var(--sre-font-mono);
  font-size: var(--sre-fs-md);
  font-weight: var(--sre-fw-semibold);
  color: var(--sre-text-primary);
  letter-spacing: 0.6px;
  font-feature-settings: "tnum" 1;
}
.clock-sep {
  color: var(--sre-text-tertiary);
  font-size: var(--sre-fs-xs);
}
.clock-date {
  font-size: var(--sre-fs-xs);
  color: var(--sre-text-secondary);
  letter-spacing: 0.2px;
}
.clock-tz {
  font-size: var(--sre-fs-2xs);
  font-weight: var(--sre-fw-bold);
  color: var(--sre-primary);
  background: var(--sre-primary-soft);
  padding: 2px 7px;
  border-radius: var(--sre-radius-xs);
  letter-spacing: 0.6px;
  text-transform: uppercase;
}

/* ===== Timezone panel ===== */
.tz-panel {
  min-width: 240px;
  padding: var(--sre-space-2) 0;
  border-radius: var(--sre-radius-md);
}
.tz-panel-title {
  display: flex;
  align-items: center;
  gap: var(--sre-space-2);
  padding: var(--sre-space-2) var(--sre-space-4) var(--sre-space-2);
  font-size: var(--sre-fs-xs);
  font-weight: var(--sre-fw-semibold);
  color: var(--sre-text-tertiary);
  letter-spacing: 0.08em;
  text-transform: uppercase;
  border-bottom: 1px solid var(--sre-border);
  margin-bottom: var(--sre-space-1);
}
.tz-option {
  display: flex;
  align-items: center;
  gap: var(--sre-space-2);
  padding: 8px var(--sre-space-4);
  cursor: pointer;
  font-size: var(--sre-fs-md);
  transition: background var(--sre-duration-fast) var(--sre-ease-out);
  color: var(--sre-text-primary);
  border-radius: var(--sre-radius-sm);
  margin: 0 var(--sre-space-2);
}
.tz-option:hover { background: var(--sre-bg-hover); }
.tz-option.selected {
  color: var(--sre-primary);
  background: var(--sre-primary-soft);
}
.tz-opt-abbr {
  font-weight: var(--sre-fw-bold);
  font-size: var(--sre-fs-xs);
  width: 36px;
  color: var(--sre-primary);
  flex-shrink: 0;
  letter-spacing: 0.04em;
}
.tz-opt-label { flex: 1; }
.tz-opt-check {
  font-size: var(--sre-fs-sm);
  color: var(--sre-primary);
  font-weight: var(--sre-fw-bold);
}

/* ===== Control buttons ===== */
.ctrl-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--sre-space-1);
  padding: 7px 10px;
  min-height: 34px;
  border-radius: var(--sre-radius-md);
  cursor: pointer;
  color: var(--sre-text-secondary);
  transition: background var(--sre-duration-fast) var(--sre-ease-out),
              color var(--sre-duration-fast) var(--sre-ease-out),
              transform var(--sre-duration-fast) var(--sre-ease-spring);
  font-size: var(--sre-fs-md);
  font-weight: var(--sre-fw-medium);
}
.ctrl-btn:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
}
.ctrl-btn:active { transform: scale(0.95); }
.ctrl-label {
  font-size: var(--sre-fs-sm);
  font-weight: var(--sre-fw-semibold);
  letter-spacing: 0.4px;
}

/* ⌘K button */
.ctrl-btn--search {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  border-radius: var(--sre-radius-pill);
  border: 1px solid var(--sre-border);
  background: var(--sre-bg-sunken);
  color: var(--sre-text-tertiary);
  cursor: pointer;
  transition: background var(--sre-duration-fast) var(--sre-ease-out),
              border-color var(--sre-duration-fast) var(--sre-ease-out),
              color var(--sre-duration-fast) var(--sre-ease-out);
  font-size: var(--sre-fs-sm);
}
.ctrl-btn--search:hover {
  background: var(--sre-primary-soft);
  border-color: var(--sre-primary-ring);
  color: var(--sre-primary);
}
.cmd-shortcut {
  font-size: var(--sre-fs-2xs);
  padding: 1px 5px;
  border-radius: 4px;
  background: var(--sre-bg-elevated);
  border: 1px solid var(--sre-border-strong);
  color: var(--sre-text-muted);
  font-family: var(--sre-font-mono);
  pointer-events: none;
}

/* ===== Transitions ===== */
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* ===== Profile Modal ===== */
.avatar-section { display: flex; align-items: flex-start; gap: 16px; padding: 12px 0 4px; }
.avatar-current {
  width: 60px; height: 60px; border-radius: 14px; font-size: 30px;
  background: var(--sre-gradient-brand-soft);
  border: 2px solid var(--sre-primary-soft);
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  overflow: hidden;
}
.avatar-preview-img {
  width: 100%; height: 100%; object-fit: cover; display: block;
}
.avatar-actions {
  flex: 1; min-width: 0;
  display: flex; flex-direction: column; gap: 10px;
}
.avatar-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(32px, 1fr));
  gap: 6px;
  max-height: 120px;
  overflow-y: auto;
  padding: 2px;
}
.avatar-option {
  width: 32px; height: 32px; border-radius: 8px; font-size: 17px;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; border: 2px solid transparent;
  transition: border-color 0.2s, background 0.2s, transform 0.15s;
  background: var(--sre-bg-subtle);
}
.avatar-option:hover { background: var(--sre-bg-hover); transform: translateY(-1px); }
.avatar-option.selected { border-color: var(--sre-primary); background: var(--sre-primary-soft); }
.avatar-upload-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.modal-footer {
  display: flex; justify-content: flex-end;
  padding-top: 16px; margin-top: 4px;
  border-top: 1px solid var(--sre-border);
}

.notify-config-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 4px;
}
.notify-config-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  background: var(--sre-bg-subtle);
  border-radius: var(--sre-radius-sm);
}
.notify-config-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}
.notify-config-value {
  font-size: 12px;
  color: var(--sre-text-secondary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
