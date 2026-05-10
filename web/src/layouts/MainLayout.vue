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
  GridOutline, ServerOutline, AlertCircleOutline, CalendarOutline,
  SettingsOutline, LogOutOutline, NotificationsOutline, NotificationsOffOutline,
  SunnyOutline, MoonOutline, ChevronDownOutline, PersonOutline,
  LockClosedOutline, EarthOutline, TimeOutline, ChevronBackOutline,
  ChevronForwardOutline, SearchOutline, LayersOutline, BugOutline,
  FlashOutline, ShieldCheckmarkOutline, GitNetworkOutline, BarChartOutline,
  PulseOutline, CubeOutline,
} from '@vicons/ionicons5'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { t, locale } = useI18n()
const message = useMessage()
const { open: openPalette } = useCommandPalette()
const appVersion = __APP_VERSION__

const isDark = inject<Ref<boolean>>('isDark', ref(true))
const toggleTheme = inject<() => void>('toggleTheme', () => {})

onMounted(() => {
  if (authStore.isLoggedIn && !authStore.user) authStore.fetchProfile()
})

// ===== Types =====
type TabKey = 'monitor' | 'incident' | 'system'

// ===== Collapsed =====
const collapsed = ref(JSON.parse(localStorage.getItem('sre-sider-collapsed') ?? 'false'))
watch(collapsed, v => localStorage.setItem('sre-sider-collapsed', JSON.stringify(v)))

// ===== Clock =====
const timeDisplay = ref('')
const timezone = ref(localStorage.getItem('sre-timezone') || 'Asia/Shanghai')
const showTzPanel = ref(false)
const timezoneOptions = [
  { label: 'Asia/Shanghai', abbr: 'CST', value: 'Asia/Shanghai' },
  { label: 'UTC', abbr: 'UTC', value: 'UTC' },
  { label: 'Asia/Tokyo', abbr: 'JST', value: 'Asia/Tokyo' },
  { label: 'Asia/Singapore', abbr: 'SGT', value: 'Asia/Singapore' },
  { label: 'Europe/London', abbr: 'GMT', value: 'Europe/London' },
  { label: 'America/New_York', abbr: 'EST', value: 'America/New_York' },
  { label: 'America/Los_Angeles', abbr: 'PST', value: 'America/Los_Angeles' },
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
function selectTimezone(val: string) { timezone.value = val; localStorage.setItem('sre-timezone', val); showTzPanel.value = false; updateClock() }

// ===== Menu =====
function renderIcon(icon: any) { return () => h(NIcon, null, { default: () => h(icon) }) }

const menuSections = computed<{ label?: string; items: MenuOption[] }[]>(() => {
  switch (activeTab.value) {
    case 'incident':
      return [{
        items: [
          { label: t('menu.incidentDashboard'), key: '/incident-dashboard', icon: renderIcon(BarChartOutline) },
          { label: t('menu.channels'), key: '/channels', icon: renderIcon(LayersOutline) },
          { label: t('menu.incidents'), key: '/incidents', icon: renderIcon(BugOutline) },
          { label: t('menu.alertsV2'), key: '/alerts-v2', icon: renderIcon(FlashOutline) },
        ],
      }]
    case 'system':
      return [{
        items: (() => {
          const items: MenuOption[] = [
            { label: t('menu.notification'), key: '/notification', icon: renderIcon(NotificationsOutline) },
            { label: t('menu.integrations'), key: '/integrations', icon: renderIcon(GitNetworkOutline) },
            { label: t('menu.schedule'), key: '/schedule', icon: renderIcon(CalendarOutline) },
          ]
          if (authStore.canManage) items.push({ label: t('menu.settings'), key: '/settings', icon: renderIcon(SettingsOutline) })
          return items
        })(),
      }]
    default:
      return [
        {
          label: t('menu.overview'),
          items: [
            { label: t('menu.dashboard'), key: '/dashboard', icon: renderIcon(GridOutline) },
          ],
        },
        {
          label: t('menu.alerts'),
          items: [
            { label: t('menu.alertRules'), key: '/alerts/rules', icon: renderIcon(AlertCircleOutline) },
            { label: t('menu.activeAlerts'), key: '/alerts/events', icon: renderIcon(PulseOutline) },
            { label: t('menu.alertHistory'), key: '/alerts/history', icon: renderIcon(TimeOutline) },
          ],
        },
        {
          label: t('menu.policies'),
          items: [
            { label: t('menu.muteRules'), key: '/alerts/mute-rules', icon: renderIcon(NotificationsOffOutline) },
            { label: t('menu.inhibitionRules'), key: '/alerts/inhibition-rules', icon: renderIcon(ShieldCheckmarkOutline) },
          ],
        },
        {
          label: t('menu.data'),
          items: [
            { label: t('menu.datasources'), key: '/datasources', icon: renderIcon(ServerOutline) },
            { label: t('menu.dataQuery'), key: '/query', icon: renderIcon(SearchOutline) },
          ],
        },
      ]
  }
})

const flatMenuOptions = computed<MenuOption[]>(() => {
  return menuSections.value.flatMap(s => s.items)
})

// ===== Tab =====
const activeTab = ref<TabKey>('monitor')

function resolveTabFromPath(p: string): TabKey {
  if (p.startsWith('/incident-dashboard') || p.startsWith('/channels') ||
      p.startsWith('/incidents') || p.startsWith('/alerts-v2')) return 'incident'
  if (p.startsWith('/integrations') || p.startsWith('/notification') ||
      p.startsWith('/schedule') || p.startsWith('/settings')) return 'system'
  return 'monitor'
}

const tabFirstRoute: Record<TabKey, string> = { monitor: '/dashboard', incident: '/incident-dashboard', system: '/notification' }

function switchTab(tab: TabKey) {
  activeTab.value = tab
  const first = tabFirstRoute[tab]
  const currentInTab = resolveTabFromPath(route.path) === tab
  if (!currentInTab && first) router.push(first)
}

const topTabs = computed(() => [
  { key: 'monitor' as TabKey, label: t('menu.monitorAlert'), icon: AlertCircleOutline },
  { key: 'incident' as TabKey, label: t('menu.incidentMgmt'), icon: BugOutline },
  { key: 'system' as TabKey, label: t('menu.systemConfig'), icon: SettingsOutline },
])

// ===== Active menu key =====
function resolveActiveKey(p: string): string {
  if (p.startsWith('/incident-dashboard')) return '/incident-dashboard'
  if (p.startsWith('/channels')) return '/channels'
  if (p.startsWith('/incidents')) return '/incidents'
  if (p.startsWith('/alerts-v2')) return '/alerts-v2'
  if (p.startsWith('/integrations')) return '/integrations'
  if (p.startsWith('/query')) return '/query'
  if (p.startsWith('/datasources')) return '/datasources'
  if (p.startsWith('/alerts/rules')) return '/alerts/rules'
  if (p.startsWith('/alerts/events')) return '/alerts/events'
  if (p.startsWith('/alerts/history')) return '/alerts/history'
  if (p.startsWith('/alerts/mute-rules')) return '/alerts/mute-rules'
  if (p.startsWith('/alerts/inhibition-rules')) return '/alerts/inhibition-rules'
  if (p.startsWith('/notification')) return '/notification'
  if (p.startsWith('/schedule')) return '/schedule'
  return p
}

const menuSelectedKey = ref(resolveActiveKey(route.path))

watch(() => route.path, (p) => {
  activeTab.value = resolveTabFromPath(p)
  menuSelectedKey.value = resolveActiveKey(p)
})

function handleMenuClick(key: string) {
  menuSelectedKey.value = ''
  router.push(key)
}

// ===== Page title =====
const pageTitle = computed(() => {
  const item = flatMenuOptions.value.find(m => m.key === resolveActiveKey(route.path))
  return item?.label as string || ''
})

// ===== Language =====
const langOptions = computed(() => [
  { label: t('language.zh'), value: 'zh-CN' },
  { label: t('language.en'), value: 'en' },
])
function handleLangChange(val: string) { locale.value = val; localStorage.setItem('locale', val) }

// ===== User =====
const userDropdownOptions = computed<DropdownOption[]>(() => [
  { label: t('header.profile'), key: 'profile', icon: renderIcon(PersonOutline) },
  { label: t('header.changePassword'), key: 'password', icon: renderIcon(LockClosedOutline) },
  { type: 'divider', key: 'd1' },
  { label: t('header.logout'), key: 'logout', icon: renderIcon(LogOutOutline) },
])

async function handleUserDropdown(key: string) {
  if (key === 'logout') { authStore.logout(); router.push('/login') }
  else if (key === 'profile') { profileTab.value = 'info'; await openProfileModal() }
  else if (key === 'password') { profileTab.value = 'password'; await openProfileModal() }
}

const userInitial = computed(() => (authStore.user?.display_name || authStore.user?.username || 'U').charAt(0).toUpperCase())
const displayName = computed(() => authStore.user?.display_name || authStore.user?.username || 'User')

function isImageAvatar(v: string | undefined | null): boolean {
  if (!v) return false
  return v.startsWith('data:image/') || v.startsWith('http://') || v.startsWith('https://') || v.startsWith('/')
}
const headerAvatar = computed(() => authStore.user?.avatar || '')
const headerAvatarIsImage = computed(() => isImageAvatar(headerAvatar.value))

// ===== Profile Modal =====
const showProfileModal = ref(false)
const profileTab = ref('info')
const profileSaving = ref(false)
const profileForm = ref({ display_name: '', email: '', phone: '', avatar: '' })
const presetAvatars = ['👤','🧑‍💻','👩‍💻','🧑‍🔧','👩‍🔧','🧑‍🚀','👩‍🚀','🧑‍🔬','👩‍🔬','🧑‍💼','👩‍💼','🧑‍🎤','🧑‍🎨','🦊','🐺','🐧','🦅','🦁','🐯','🐻','🐼','🦉','🦄','🐉','🤖','👾','🛰️','🚀','⚡','🔥','🌟','🌈']
const AVATAR_MAX_BYTES = 200 * 1024
const avatarFileInput = ref<HTMLInputElement | null>(null)
function triggerAvatarUpload() { avatarFileInput.value?.click() }
function onAvatarFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!/^image\/(png|jpe?g|svg\+xml|webp)$/.test(file.type)) { message.error(t('profile.avatarInvalidType')); input.value = ''; return }
  if (file.size > AVATAR_MAX_BYTES) { message.error(t('profile.avatarTooLarge')); input.value = ''; return }
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
  { label: t('profile.email'), value: 'email' },
  { label: t('profile.webhook'), value: 'webhook' },
])
const configHint = computed(() => {
  switch (newNotifyConfig.value.media_type) {
    case 'lark_personal': return t('profile.larkUserIdHint')
    case 'email': return t('profile.emailHint')
    case 'webhook': return t('profile.webhookHint')
    default: return ''
  }
})

async function openProfileModal() {
  profileSaving.value = false; pwdError.value = ''
  pwdForm.value = { old_password: '', new_password: '', confirm_password: '' }
  profileForm.value = { display_name: authStore.user?.display_name || '', email: authStore.user?.email || '', phone: authStore.user?.phone || '', avatar: authStore.user?.avatar || '' }
  try { const cfgs = await userNotifyConfigApi.list(); userNotifyConfigs.value = cfgs.data.data || [] } catch { userNotifyConfigs.value = [] }
  newNotifyConfig.value = { media_type: 'lark_personal', config: '' }
  showProfileModal.value = true
}

async function saveProfile() {
  profileSaving.value = true
  try { await authApi.updateMe(profileForm.value); await authStore.fetchProfile(); message.success(t('profile.saved')) }
  catch (err: any) { message.error(err.message || t('common.failed')) }
  finally { profileSaving.value = false }
}

async function savePassword() {
  pwdError.value = ''
  if (pwdForm.value.new_password !== pwdForm.value.confirm_password) { pwdError.value = t('profile.passwordMismatch'); return }
  profileSaving.value = true
  try { await authApi.changeMyPassword({ old_password: pwdForm.value.old_password, new_password: pwdForm.value.new_password }); message.success(t('profile.passwordChanged')); pwdForm.value = { old_password: '', new_password: '', confirm_password: '' } }
  catch (err: any) { message.error(err.message || t('common.failed')) }
  finally { profileSaving.value = false }
}

async function addNotifyConfig() {
  if (!newNotifyConfig.value.config) return
  profileSaving.value = true
  try { await userNotifyConfigApi.upsert({ ...newNotifyConfig.value, is_enabled: true }); const cfgs = await userNotifyConfigApi.list(); userNotifyConfigs.value = cfgs.data.data || []; newNotifyConfig.value = { media_type: 'lark_personal', config: '' }; message.success(t('profile.notifyConfigSaved')) }
  catch (err: any) { message.error(err.message || t('common.failed')) }
  finally { profileSaving.value = false }
}

const larkOpenIdInput = ref('')
const larkBindSaving = ref(false)
async function saveLarkBind() {
  const openId = larkOpenIdInput.value.trim(); if (!openId) return
  larkBindSaving.value = true
  try { await authApi.bindLark(openId); message.success(t('settings.larkBindSuccess')); larkOpenIdInput.value = '' }
  catch (err: any) { message.error(err.message || t('common.failed')) }
  finally { larkBindSaving.value = false }
}

async function removeNotifyConfig(mediaType: string) {
  try { await userNotifyConfigApi.deleteByType(mediaType); userNotifyConfigs.value = userNotifyConfigs.value.filter(c => c.media_type !== mediaType) }
  catch (err: any) { message.error(err.message || t('common.failed')) }
}

async function toggleNotifyConfig(cfg: UserNotifyConfig, enabled: boolean) {
  try { await userNotifyConfigApi.upsert({ ...cfg, is_enabled: enabled }) }
  catch (err: any) { message.error(err.message || t('common.failed')) }
}
</script>

<template>
  <div class="app-shell">
    <!-- ===== Top Bar ===== -->
    <header class="topbar">
      <div class="topbar-start">
        <!-- Logo -->
        <router-link to="/dashboard" class="topbar-logo">
          <img src="/logo.svg" alt="SREAgent" class="logo-img" />
          <span class="logo-label"><span class="gradient-text">SRE</span>Agent</span>
        </router-link>

        <div class="topbar-sep" />

        <!-- Module Tabs -->
        <nav class="topbar-tabs">
          <button
            v-for="tab in topTabs"
            :key="tab.key"
            class="topbar-tab"
            :class="{ active: activeTab === tab.key }"
            @click="switchTab(tab.key)"
          >
            <n-icon :component="tab.icon" :size="18" />
            <span>{{ tab.label }}</span>
          </button>
        </nav>
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

        <!-- ⌘K -->
        <button class="topbar-btn topbar-kbd" @click="openPalette" title="⌘K">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
          <kbd>⌘K</kbd>
        </button>

        <!-- Lang -->
        <n-popselect :value="locale" :options="langOptions" trigger="click" :render-label="(o: any) => o.label" @update:value="handleLangChange">
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

    <!-- ===== Body ===== -->
    <div class="app-body">
      <!-- Sidebar -->
      <aside class="sidebar" :class="{ collapsed }">
        <nav class="sidebar-nav">
          <template v-for="(section, si) in menuSections" :key="si">
            <div v-if="section.label" class="sidebar-section-label">
              <transition name="fade">
                <span v-if="!collapsed">{{ section.label }}</span>
              </transition>
            </div>
            <n-menu
              :collapsed="collapsed"
              :collapsed-width="64"
              :collapsed-icon-size="22"
              :options="section.items"
              :value="menuSelectedKey"
              :indent="16"
              @update:value="handleMenuClick"
            />
            <div v-if="si < menuSections.length - 1" class="sidebar-section-gap" />
          </template>
        </nav>

        <div class="sidebar-spacer" />

        <!-- User -->
        <div class="sidebar-bottom">
          <n-dropdown :options="userDropdownOptions" trigger="click" @select="handleUserDropdown">
            <div class="sidebar-user" :class="{ collapsed }">
              <div class="user-avatar" :class="{ 'avatar-img': headerAvatarIsImage, 'avatar-emoji': !!headerAvatar && !headerAvatarIsImage }">
                <img v-if="headerAvatarIsImage" :src="headerAvatar" alt="" />
                <template v-else-if="headerAvatar">{{ headerAvatar }}</template>
                <template v-else>{{ userInitial }}</template>
              </div>
              <transition name="fade">
                <div v-if="!collapsed" class="user-meta">
                  <span class="user-name">{{ displayName }}</span>
                  <span class="user-role">{{ authStore.canManage ? t('role.admin') : t('role.member') }}</span>
                </div>
              </transition>
              <transition name="fade">
                <n-icon v-if="!collapsed" :component="ChevronDownOutline" :size="12" class="user-chevron" />
              </transition>
            </div>
          </n-dropdown>

          <button class="sidebar-collapse" :class="{ collapsed }" @click="collapsed = !collapsed">
            <n-icon :component="collapsed ? ChevronForwardOutline : ChevronBackOutline" :size="14" />
            <transition name="fade">
              <span v-if="!collapsed">{{ t('header.collapseSidebar') }}</span>
            </transition>
          </button>

          <transition name="fade">
            <div v-if="!collapsed" class="sidebar-version">v{{ appVersion }}</div>
          </transition>
        </div>
      </aside>

      <!-- Main -->
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

  <!-- Profile Modal -->
  <n-modal v-model:show="showProfileModal" :title="t('profile.title')" preset="card" style="width:500px" :bordered="false" :segmented="{ content: true }">
    <n-tabs v-model:value="profileTab" type="line" animated>
      <n-tab-pane name="info" :tab="t('profile.tabInfo')">
        <div class="avatar-section">
          <div class="avatar-current">
            <img v-if="isImageAvatar(profileForm.avatar)" :src="profileForm.avatar" alt="" class="avatar-preview-img" />
            <span v-else class="avatar-preview">{{ profileForm.avatar || userInitial }}</span>
          </div>
          <div class="avatar-actions">
            <div class="avatar-grid">
              <span v-for="a in presetAvatars" :key="a" class="avatar-option" :class="{ selected: profileForm.avatar === a }" @click="profileForm.avatar = a">{{ a }}</span>
            </div>
            <div class="avatar-upload-row">
              <n-button size="tiny" secondary @click="triggerAvatarUpload">📎 {{ t('profile.uploadAvatar') }}</n-button>
              <n-button v-if="profileForm.avatar" size="tiny" quaternary type="error" @click="profileForm.avatar = ''">{{ t('profile.clearAvatar') }}</n-button>
              <input ref="avatarFileInput" type="file" accept="image/png,image/jpeg,image/svg+xml,image/webp" style="display:none" @change="onAvatarFileChange" />
            </div>
          </div>
        </div>
        <n-form label-placement="top" size="small" style="margin-top:16px">
          <n-form-item :label="t('auth.username')"><n-input :value="authStore.user?.username" disabled /></n-form-item>
          <n-form-item :label="t('settings.displayName')"><n-input v-model:value="profileForm.display_name" /></n-form-item>
          <n-form-item :label="t('settings.email')"><n-input v-model:value="profileForm.email" /></n-form-item>
          <n-form-item :label="t('settings.phone')"><n-input v-model:value="profileForm.phone" placeholder="+86 138..." /></n-form-item>
        </n-form>
        <div class="modal-footer"><n-button type="primary" :loading="profileSaving" @click="saveProfile">{{ t('common.save') }}</n-button></div>
      </n-tab-pane>
      <n-tab-pane name="password" :tab="t('profile.tabPassword')">
        <n-form label-placement="top" size="small">
          <n-form-item :label="t('profile.oldPassword')"><n-input v-model:value="pwdForm.old_password" type="password" show-password-on="click" /></n-form-item>
          <n-form-item :label="t('profile.newPassword')"><n-input v-model:value="pwdForm.new_password" type="password" show-password-on="click" /></n-form-item>
          <n-form-item :label="t('profile.confirmPassword')"><n-input v-model:value="pwdForm.confirm_password" type="password" show-password-on="click" :status="pwdError ? 'error' : undefined" />
            <template #feedback><span v-if="pwdError" style="color:var(--sre-critical)">{{ pwdError }}</span></template>
          </n-form-item>
        </n-form>
        <div class="modal-footer"><n-button type="primary" :loading="profileSaving" @click="savePassword">{{ t('profile.changePassword') }}</n-button></div>
      </n-tab-pane>
      <n-tab-pane name="notify" :tab="t('profile.tabNotify')">
        <div class="notify-config-list">
          <div v-for="cfg in userNotifyConfigs" :key="cfg.media_type" class="notify-config-item">
            <div class="notify-config-info">
              <n-tag size="small" :type="cfg.media_type === 'lark_personal' ? 'success' : cfg.media_type === 'email' ? 'info' : 'default'">{{ cfg.media_type === 'lark_personal' ? t('profile.larkPersonal') : cfg.media_type === 'email' ? t('profile.email') : t('profile.webhook') }}</n-tag>
              <span class="notify-config-value">{{ cfg.config }}</span>
              <n-switch v-model:value="cfg.is_enabled" size="small" @update:value="(v: boolean) => toggleNotifyConfig(cfg, v)" />
            </div>
            <n-button size="tiny" quaternary type="error" @click="removeNotifyConfig(cfg.media_type)">{{ t('common.remove') }}</n-button>
          </div>
          <n-empty v-if="userNotifyConfigs.length === 0" :description="t('profile.noNotifyConfig')" style="padding:20px 0" />
        </div>
        <n-divider>{{ t('profile.addNotify') }}</n-divider>
        <n-form label-placement="top" size="small">
          <n-form-item :label="t('profile.mediaType')"><n-select v-model:value="newNotifyConfig.media_type" :options="mediaTypeOptions" /></n-form-item>
          <n-form-item :label="t('profile.configValue')"><n-input v-model:value="newNotifyConfig.config" :placeholder="configHint" clearable /></n-form-item>
        </n-form>
        <div class="modal-footer"><n-button type="primary" :loading="profileSaving" @click="addNotifyConfig">{{ t('profile.addNotify') }}</n-button></div>
      </n-tab-pane>
      <n-tab-pane name="lark" :tab="t('settings.larkBind')">
        <n-space vertical size="large" style="padding:8px 0">
          <n-alert type="info" :title="t('settings.larkBind')" style="font-size:13px">{{ t('settings.larkBindHint') }}</n-alert>
          <n-form label-placement="top" size="small">
            <n-form-item :label="t('settings.larkOpenId')"><n-input v-model:value="larkOpenIdInput" :placeholder="t('settings.larkOpenId')" clearable style="max-width:360px" /></n-form-item>
          </n-form>
          <n-button type="primary" :loading="larkBindSaving" :disabled="!larkOpenIdInput.trim()" @click="saveLarkBind">{{ t('settings.larkBind') }}</n-button>
        </n-space>
      </n-tab-pane>
    </n-tabs>
  </n-modal>
</template>

<style scoped>
/* ============================================================
   App Shell
   ============================================================ */
.app-shell { display:flex; flex-direction:column; height:100vh; overflow:hidden; }

/* ===== Top Bar ===== */
.topbar {
  display:flex; align-items:center; justify-content:space-between;
  height:52px; padding:0 16px; flex-shrink:0;
  background:var(--sre-glass-bg);
  backdrop-filter:saturate(var(--sre-glass-saturate)) blur(var(--sre-glass-blur));
  -webkit-backdrop-filter:saturate(var(--sre-glass-saturate)) blur(var(--sre-glass-blur));
  border-bottom:1px solid var(--sre-glass-border);
  z-index:var(--sre-z-sticky);
}

.topbar-start { display:flex; align-items:center; gap:0; }
.topbar-end { display:flex; align-items:center; gap:4px; }

.topbar-logo {
  display:flex; align-items:center; gap:8px; text-decoration:none;
  padding:4px 8px 4px 0; border-radius:var(--sre-radius-sm);
  transition:opacity var(--sre-duration-fast) var(--sre-ease-out);
}
.topbar-logo:hover { opacity:0.8; }
.logo-img { width:26px; height:26px; border-radius:6px; filter:drop-shadow(0 4px 12px rgba(16,185,129,0.45)); }
.logo-label { font-size:15px; font-weight:600; color:var(--sre-text-primary); letter-spacing:-0.01em; white-space:nowrap; }

.topbar-sep {
  width:1px; height:22px; background:var(--sre-border); margin:0 12px; opacity:0.6;
}

/* Top Tabs */
.topbar-tabs { display:flex; align-items:center; gap:2px; }

.topbar-tab {
  display:inline-flex; align-items:center; gap:7px;
  padding:7px 14px; border:none; border-radius:var(--sre-radius-md);
  background:transparent; color:var(--sre-text-secondary);
  font-size:13px; font-weight:500; font-family:var(--sre-font-sans);
  cursor:pointer; white-space:nowrap; user-select:none;
  transition:all var(--sre-duration-base) var(--sre-ease-out);
  position:relative;
}
.topbar-tab:hover { background:var(--sre-bg-hover); color:var(--sre-text-primary); }
.topbar-tab.active {
  background:var(--sre-primary-soft);
  color:var(--sre-primary);
  font-weight:600;
  box-shadow:0 0 0 1px rgba(16,185,129,0.20), 0 4px 12px -2px rgba(16,185,129,0.15);
}
.topbar-tab:active { transform:scale(0.97); transition-duration:80ms; }

/* Topbar utility buttons */
.topbar-btn {
  display:inline-flex; align-items:center; gap:5px;
  padding:6px 9px; min-height:32px; border:none; border-radius:var(--sre-radius-sm);
  background:transparent; color:var(--sre-text-secondary);
  font-size:12px; font-weight:500; font-family:var(--sre-font-sans);
  cursor:pointer; white-space:nowrap;
  transition:background var(--sre-duration-fast) var(--sre-ease-out),
             color var(--sre-duration-fast) var(--sre-ease-out);
}
.topbar-btn:hover { background:var(--sre-bg-hover); color:var(--sre-text-primary); }

.topbar-clock {
  border:1px solid var(--sre-border); border-radius:var(--sre-radius-pill);
  padding:5px 11px; gap:6px; background:var(--sre-bg-sunken);
}
.topbar-clock:hover, .topbar-clock.active {
  background:var(--sre-primary-soft); border-color:var(--sre-primary-ring);
}
.clock-text { font-family:var(--sre-font-mono); font-size:13px; font-weight:600; color:var(--sre-text-primary); font-feature-settings:"tnum" 1; }
.clock-tz { font-size:10px; font-weight:700; color:var(--sre-primary); background:var(--sre-primary-soft); padding:1px 6px; border-radius:4px; }

.topbar-kbd { gap:5px; }
.topbar-kbd kbd { font-size:10px; padding:1px 5px; border-radius:4px; background:var(--sre-bg-elevated); border:1px solid var(--sre-border-strong); color:var(--sre-text-muted); font-family:var(--sre-font-mono); }

/* ===== Timezone Panel ===== */
.tz-panel { min-width:220px; padding:4px 0; }
.tz-panel-title { display:flex; align-items:center; gap:8px; padding:8px 16px 6px; font-size:11px; font-weight:600; color:var(--sre-text-tertiary); letter-spacing:0.08em; text-transform:uppercase; border-bottom:1px solid var(--sre-border); margin-bottom:4px; }
.tz-item { display:flex; align-items:center; gap:8px; padding:7px 16px; cursor:pointer; font-size:13px; color:var(--sre-text-primary); transition:background var(--sre-duration-fast) var(--sre-ease-out); margin:0 4px; border-radius:6px; }
.tz-item:hover { background:var(--sre-bg-hover); }
.tz-item.selected { color:var(--sre-primary); background:var(--sre-primary-soft); }
.tz-abbr { font-weight:700; font-size:11px; width:32px; color:var(--sre-primary); flex-shrink:0; }
.tz-label { flex:1; }
.tz-check { font-weight:700; color:var(--sre-primary); font-size:12px; }

/* ===== App Body ===== */
.app-body { display:flex; flex:1; min-height:0; }

/* ===== Sidebar ===== */
.sidebar {
  width:232px; flex-shrink:0; display:flex; flex-direction:column;
  background:var(--sre-glass-bg);
  backdrop-filter:saturate(var(--sre-glass-saturate)) blur(var(--sre-glass-blur));
  -webkit-backdrop-filter:saturate(var(--sre-glass-saturate)) blur(var(--sre-glass-blur));
  border-right:1px solid var(--sre-glass-border);
  transition:width 280ms var(--sre-ease-spring);
  overflow:hidden; position:relative;
}
.sidebar.collapsed { width:64px; }

.sidebar-nav { flex:0 1 auto; overflow-y:auto; overflow-x:hidden; padding:8px; position:relative; z-index:1; }

/* Section labels */
.sidebar-section-label {
  display:flex; align-items:center; gap:8px;
  padding:10px 12px 4px;
  font-size:10px; font-weight:700; letter-spacing:0.08em;
  text-transform:uppercase; color:var(--sre-text-tertiary);
  white-space:nowrap; overflow:hidden;
}
.sidebar-section-label::before {
  content:''; width:4px; height:4px; border-radius:50%;
  background:var(--sre-text-tertiary); flex-shrink:0;
  transition:background var(--sre-duration-fast) var(--sre-ease-out);
}
.sidebar-section-gap { height:6px; }

/* Active menu accent */
:deep(.n-menu .n-menu-item-content) {
  position:relative; overflow:visible; min-height:36px; margin:1px 0;
}
:deep(.n-menu .n-menu-item-content--selected::before) {
  content:''; position:absolute; left:2px; top:7px; bottom:7px;
  width:3px; border-radius:0 3px 3px 0; background:var(--sre-gradient-brand);
}

.sidebar-spacer { flex:1; }

/* User area */
.sidebar-bottom {
  display:flex; flex-direction:column; gap:6px; padding:8px;
  border-top:1px solid var(--sre-glass-border); z-index:1;
}

.sidebar-user {
  display:flex; align-items:center; gap:10px;
  padding:8px 10px; border-radius:var(--sre-radius-md);
  cursor:pointer; user-select:none;
  border:1px solid transparent;
  transition:all var(--sre-duration-fast) var(--sre-ease-out);
}
.sidebar-user:hover {
  background:var(--sre-bg-hover);
  border-color:var(--sre-border);
}
.sidebar-user.collapsed { justify-content:center; padding:8px; }

.user-avatar {
  width:30px; height:30px; border-radius:50%;
  background:var(--sre-gradient-brand); color:#fff;
  font-size:12px; font-weight:700;
  display:flex; align-items:center; justify-content:center;
  flex-shrink:0; overflow:hidden;
  box-shadow:0 2px 8px -2px rgba(16,185,129,0.40), inset 0 1px 0 rgba(255,255,255,0.2);
}
.user-avatar.avatar-emoji { font-size:16px; font-weight:400; background:transparent; box-shadow:inset 0 0 0 1px var(--sre-border); }
.user-avatar.avatar-img { background:transparent; box-shadow:inset 0 0 0 1px var(--sre-border); }
.user-avatar img { width:100%; height:100%; object-fit:cover; }
.user-meta { display:flex; flex-direction:column; min-width:0; flex:1; }
.user-name { font-size:13px; font-weight:500; color:var(--sre-text-primary); overflow:hidden; text-overflow:ellipsis; white-space:nowrap; line-height:1.2; }
.user-role { font-size:10px; color:var(--sre-text-tertiary); line-height:1.3; }
.user-chevron { color:var(--sre-text-tertiary); flex-shrink:0; transition:transform var(--sre-duration-fast) var(--sre-ease-out); }
.sidebar-user:hover .user-chevron { transform:translateY(1px); }

.sidebar-collapse {
  display:flex; align-items:center; gap:8px;
  padding:6px 10px; border:none; border-radius:var(--sre-radius-sm);
  background:transparent; color:var(--sre-text-tertiary);
  font-size:11px; font-weight:500; font-family:var(--sre-font-sans);
  cursor:pointer; white-space:nowrap; overflow:hidden;
  transition:all var(--sre-duration-fast) var(--sre-ease-out);
}
.sidebar-collapse.collapsed { justify-content:center; }
.sidebar-collapse:hover { background:var(--sre-primary-soft); color:var(--sre-primary); }

.sidebar-version { text-align:center; font-size:10px; color:var(--sre-text-tertiary); opacity:0.5; }

/* ===== Main Content ===== */
.main { flex:1; min-width:0; display:flex; flex-direction:column; overflow:hidden; }

.main-header {
  display:flex; align-items:center; justify-content:space-between;
  padding:14px 24px; flex-shrink:0;
  border-bottom:1px solid var(--sre-glass-border);
  background:var(--sre-glass-bg);
  backdrop-filter:saturate(var(--sre-glass-saturate)) blur(var(--sre-glass-blur));
  -webkit-backdrop-filter:saturate(var(--sre-glass-saturate)) blur(var(--sre-glass-blur));
}
.main-title {
  font-size:18px; font-weight:600; color:var(--sre-text-primary);
  letter-spacing:-0.01em; margin:0; line-height:1.2;
}
.main-actions { display:flex; align-items:center; gap:8px; }

.main-content { flex:1; overflow-y:auto; padding:24px; }

/* ===== Transitions ===== */
.fade-enter-active, .fade-leave-active { transition:opacity 0.2s; }
.fade-enter-from, .fade-leave-to { opacity:0; }

/* ===== Profile Modal ===== */
.avatar-section { display:flex; align-items:flex-start; gap:16px; padding:12px 0 4px; }
.avatar-current { width:60px; height:60px; border-radius:14px; font-size:30px; background:var(--sre-gradient-brand-soft); border:2px solid var(--sre-primary-soft); display:flex; align-items:center; justify-content:center; flex-shrink:0; overflow:hidden; }
.avatar-preview-img { width:100%; height:100%; object-fit:cover; }
.avatar-actions { flex:1; min-width:0; display:flex; flex-direction:column; gap:10px; }
.avatar-grid { display:grid; grid-template-columns:repeat(auto-fill, minmax(32px, 1fr)); gap:6px; max-height:120px; overflow-y:auto; padding:2px; }
.avatar-option { width:32px; height:32px; border-radius:8px; font-size:17px; display:flex; align-items:center; justify-content:center; cursor:pointer; border:2px solid transparent; transition:border-color 0.2s, background 0.2s, transform 0.15s; background:var(--sre-bg-subtle); }
.avatar-option:hover { background:var(--sre-bg-hover); transform:translateY(-1px); }
.avatar-option.selected { border-color:var(--sre-primary); background:var(--sre-primary-soft); }
.avatar-upload-row { display:flex; align-items:center; gap:8px; }
.modal-footer { display:flex; justify-content:flex-end; padding-top:16px; margin-top:4px; border-top:1px solid var(--sre-border); }
.notify-config-list { display:flex; flex-direction:column; gap:8px; margin-bottom:4px; }
.notify-config-item { display:flex; align-items:center; justify-content:space-between; gap:8px; padding:8px 10px; background:var(--sre-bg-subtle); border-radius:var(--sre-radius-sm); }
.notify-config-info { display:flex; align-items:center; gap:8px; flex:1; min-width:0; }
.notify-config-value { font-size:12px; color:var(--sre-text-secondary); flex:1; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
</style>
