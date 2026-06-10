<script setup lang="ts">
import { ref, computed, watch, inject, onMounted, onUnmounted, onErrorCaptured, nextTick } from 'vue'
import type { Ref } from 'vue'
import { NIcon, NPopover, NPopselect, NResult, NModal, NDrawer, NDrawerContent, NAlert } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useAppNav } from '@/composables/useAppNav'
import { useCommandPalette } from '@/composables/useCommandPalette'
import { useSessionGuard } from '@/composables/useSessionGuard'
import { useAuthStore } from '@/stores/auth'
import AppRail from '@/layouts/AppRail.vue'
import AppSidebar from '@/layouts/AppSidebar.vue'
import CommandPalette from '@/components/common/CommandPalette.vue'
import ChangePasswordModal from '@/components/common/ChangePasswordModal.vue'
import NotificationBell from '@/components/common/NotificationBell.vue'
import AIChatButton from '@/components/ai/AIChatButton.vue'
import AIChatPanel from '@/components/ai/AIChatPanel.vue'
import { useRouter } from 'vue-router'
import { TimeOutline, EarthOutline, SunnyOutline, MoonOutline, HelpOutline, MenuOutline } from '@vicons/ionicons5'

const { t, locale } = useI18n()
const authStore = useAuthStore()
const { activeApp, switchApp, menuSections, activeMenuKey, pageTitle } = useAppNav()
const { open: openPalette, registerAction } = useCommandPalette()

const router = useRouter()

// ===== Session Guard =====
const { isOnline, sessionExpired, serverRestarted, acceptCurrentServer, dismiss } = useSessionGuard()
const showSessionExpiredModal = ref(false)
const sessionRedirectCountdown = ref(10)
let countdownTimer: ReturnType<typeof setInterval> | null = null

watch(sessionExpired, (expired) => {
  if (expired) {
    showSessionExpiredModal.value = true
    sessionRedirectCountdown.value = 10
    countdownTimer = setInterval(() => {
      sessionRedirectCountdown.value--
      if (sessionRedirectCountdown.value <= 0) {
        if (countdownTimer) clearInterval(countdownTimer)
        doSessionRedirect()
      }
    }, 1000)
  }
})

const sessionExpiredTitle = computed(() => {
  if (serverRestarted.value) return t('session.serverRestarted')
  return t('session.sessionExpired')
})
const sessionExpiredDesc = computed(() => {
  if (serverRestarted.value) return t('session.serverRestartedDesc')
  return t('session.expiredDesc')
})

function doSessionRedirect() {
  if (countdownTimer) { clearInterval(countdownTimer); countdownTimer = null }
  showSessionExpiredModal.value = false
  // Clear started_at so re-login accepts the current server
  localStorage.removeItem('sre.server_started_at')
  authStore.logout()
  router.push({ name: 'Login', query: { redirect: router.currentRoute.value.fullPath } })
}

function dismissSessionExpired() {
  if (countdownTimer) { clearInterval(countdownTimer); countdownTimer = null }
  showSessionExpiredModal.value = false
  dismiss()
}

// ===== Error Boundary =====
// Key-based re-render: incrementing the key forces Vue to destroy and re-create
// the entire router-view subtree, giving the failed component a clean slate.
const routerViewKey = ref(0)
const capturedError = ref<Error | null>(null)
onErrorCaptured((err, instance, info) => {
  console.error('[AppShell] Error captured:', err, info)
  capturedError.value = err
  return false // prevent further propagation
})
async function resetError() {
  capturedError.value = null
  routerViewKey.value++
  await nextTick()
}

// FE2-5: Copy error details to clipboard
const copiedOk = ref(false)
async function copyErrorDetails() {
  if (!capturedError.value) return
  const text = [
    `Error: ${capturedError.value.message}`,
    `Stack: ${capturedError.value.stack || '(no stack trace)'}`,
    `URL: ${window.location.href}`,
    `Time: ${new Date().toISOString()}`,
  ].join('\n')
  try {
    await navigator.clipboard.writeText(text)
    copiedOk.value = true
    setTimeout(() => { copiedOk.value = false }, 2000)
  } catch {
    // Fallback for older browsers
    const ta = document.createElement('textarea')
    ta.value = text
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
    copiedOk.value = true
    setTimeout(() => { copiedOk.value = false }, 2000)
  }
}

const isDark = inject<Ref<boolean>>('isDark', ref(true))
const toggleTheme = inject<() => void>('toggleTheme', () => {})

// ===== Profile fetch =====
onMounted(() => {
  if (authStore.isLoggedIn && !authStore.user) authStore.fetchProfile()

  // Sync server started_at after login to prevent false restart detection
  if (authStore.isLoggedIn) acceptCurrentServer()

  // ── Command palette actions ──────────────────────────────────
  registerAction({
    id: 'act-theme',
    label: t('command.toggleDarkMode'),
    hint: t('command.action'),
    icon: 'contrast-outline',
    category: 'settings',
    action: toggleTheme,
  })
  registerAction({
    id: 'act-switch-oncall',
    label: t('command.switchToOncall'),
    hint: t('command.action'),
    icon: 'grid-outline',
    action: () => switchApp('oncall'),
  })
  registerAction({
    id: 'act-switch-alert',
    label: t('command.switchToAlert'),
    hint: t('command.action'),
    icon: 'flash-outline',
    action: () => switchApp('alert'),
  })
  registerAction({
    id: 'act-switch-platform',
    label: t('command.switchToPlatform'),
    hint: t('command.action'),
    icon: 'settings-outline',
    action: () => switchApp('platform'),
  })
})

// ===== Sidebar collapsed =====
function safeParse(json: string | null, fallback: boolean): boolean {
  try { return json ? JSON.parse(json) : fallback } catch { return fallback }
}
const collapsed = ref(false)
watch(collapsed, v => localStorage.setItem('sre-sider-collapsed', JSON.stringify(v)))
const showPasswordModal = ref(false)
const showAIChat = ref(false)
const showShortcuts = ref(false)
const showMobileNav = ref(false)
const pinned = ref(true)
watch(pinned, v => localStorage.setItem('sre-sider-pinned', JSON.stringify(v)))

function toggleAIChat() {
  showAIChat.value = !showAIChat.value
}

function toggleCollapse() {
  collapsed.value = !collapsed.value
}

function renderLangLabel(option: { label?: string }): string {
  return option.label ?? ''
}

// Hover expand: when collapsed, hovering the nav zone temporarily expands it
let hoverTimeout: ReturnType<typeof setTimeout> | null = null

function handleNavEnter() {
  if (!collapsed.value) return
  if (hoverTimeout) { clearTimeout(hoverTimeout); hoverTimeout = null }
  pinned.value = true
}

function handleNavLeave() {
  if (!collapsed.value) return
  hoverTimeout = setTimeout(() => { pinned.value = false }, 200)
}

const activeAppLabel = computed(() => {
  const labels: Record<string, string> = { oncall: t('rail.oncall'), alert: t('rail.alert'), platform: t('rail.platform') }
  return labels[activeApp.value] || 'SREAgent'
})

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

// FE8-6: Keyboard shortcut help overlay (Ctrl+? / Cmd+?)
const shortcuts = computed(() => [
  { keys: 'Ctrl + K / Cmd + K', desc: t('commandPalette.openHint') || 'Open command palette' },
  { keys: 'Ctrl + ? / Cmd + ?', desc: t('shortcuts.showHelp') || 'Show keyboard shortcuts' },
  { keys: 'Escape', desc: t('shortcuts.closeOverlay') || 'Close overlay / modal' },
])

function handleGlobalKeydown(e: KeyboardEvent) {
  // Ctrl+? or Cmd+? — toggle shortcut help
  if ((e.ctrlKey || e.metaKey) && e.key === '?') {
    e.preventDefault()
    showShortcuts.value = !showShortcuts.value
  }
}

onMounted(() => {
  updateClock()
  clockInterval = setInterval(updateClock, 1000)
  document.addEventListener('keydown', handleGlobalKeydown)
})
onUnmounted(() => {
  clearInterval(clockInterval)
  document.removeEventListener('keydown', handleGlobalKeydown)
  if (hoverTimeout) { clearTimeout(hoverTimeout); hoverTimeout = null }
})

function selectTimezone(val: string) {
  timezone.value = val
  localStorage.setItem('sre-timezone', val)
  showTzPanel.value = false
  updateClock()
}

// ===== Language =====
const langOptions = computed(() => [
  { label: t('language.zh'), value: 'zh-CN' },
  { label: t('language.en'), value: 'en' },
])
function handleLangChange(val: string) { locale.value = val; localStorage.setItem('locale', val) }
</script>

<template>
  <div class="app-shell">
    <!-- Error Boundary fallback -->
    <div v-if="capturedError" class="error-boundary">
      <NResult status="error" :title="t('error.renderError')" :description="capturedError.message">
        <template #footer>
          <div class="error-actions">
            <button class="error-reset-btn" @click="resetError">{{ t('common.retry') }}</button>
            <button class="error-copy-btn" @click="copyErrorDetails">{{ copiedOk ? (t('common.copied') || 'Copied!') : (t('error.copyDetails') || 'Copy error details') }}</button>
          </div>
        </template>
      </NResult>
    </div>
    <template v-else>
    <a href="#main-content" class="skip-to-content">{{ t('a11y.skipToContent') }}</a>

    <!-- ===== Connection Status Banner ===== -->
    <Transition name="banner">
      <NAlert
        v-if="!isOnline && !sessionExpired"
        type="warning"
        :bordered="false"
        :show-icon="true"
        class="connection-banner"
      >
        {{ t('session.serverUnreachable') }}
        <template #header>
          {{ t('session.connectionLost') }}
        </template>
      </NAlert>
    </Transition>

    <!-- ===== Top Bar ===== -->
    <header class="topbar">
      <div class="topbar-start">
        <!-- Logo -->
        <router-link to="/" class="topbar-logo">
          <img src="/logo.svg" alt="SREAgent" class="logo-img" />
          <span class="logo-label">SREAgent</span>
        </router-link>

        <div class="topbar-sep" />
      </div>

      <!-- Mobile hamburger (hidden on desktop via CSS) -->
      <button class="topbar-btn mobile-hamburger" @click="showMobileNav = true" :aria-label="t('common.menu') || 'Menu'">
        <n-icon :component="MenuOutline" :size="20" />
      </button>

      <div class="topbar-end">
        <!-- Notification Bell — aria-live for screen readers -->
        <div aria-live="polite" aria-atomic="true" class="topbar-notification-area">
          <NotificationBell />
        </div>

        <!-- Clock -->
        <n-popover v-model:show="showTzPanel" trigger="click" placement="bottom-end" :show-arrow="false" style="padding:0">
          <template #trigger>
            <button v-ripple class="topbar-btn topbar-clock" :class="{ active: showTzPanel }" :aria-label="t('header.timezone')">
              <n-icon :component="TimeOutline" :size="14" />
              <span class="clock-text">{{ timeDisplay }}</span>
              <span class="clock-tz">{{ tzAbbr }}</span>
            </button>
          </template>
          <div class="tz-panel">
            <div class="tz-panel-title">{{ t('header.timezone') }}</div>
            <div
              v-for="opt in timezoneOptions"
              :key="opt.value"
              class="tz-item"
              :class="{ selected: timezone === opt.value }"
              role="option"
              tabindex="0"
              @click="selectTimezone(opt.value)"
              @keydown.enter="selectTimezone(opt.value)"
            >
              <span class="tz-abbr">{{ opt.abbr }}</span>
              <span class="tz-label">{{ opt.label }}</span>
              <span v-if="timezone === opt.value" class="tz-check">&#10003;</span>
            </div>
          </div>
        </n-popover>

        <!-- ⌘K -->
        <button v-ripple class="topbar-btn topbar-kbd" @click="openPalette" title="⌘K" :aria-label="t('common.search')">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
          <kbd>⌘K</kbd>
        </button>

        <!-- Shortcuts help (FE8-6) -->
        <button v-ripple class="topbar-btn" @click="showShortcuts = true" :title="t('shortcuts.title') || 'Keyboard Shortcuts'" :aria-label="t('shortcuts.title') || 'Keyboard Shortcuts'">
          <n-icon :component="HelpOutline" :size="15" />
        </button>

        <!-- Lang -->
        <n-popselect :value="locale" :options="langOptions" trigger="click" :render-label="renderLangLabel" @update:value="handleLangChange">
          <button v-ripple class="topbar-btn" :aria-label="t('language.switch')">
            <n-icon :component="EarthOutline" :size="15" />
            <span>{{ locale === 'zh-CN' ? '中' : 'EN' }}</span>
          </button>
        </n-popselect>

        <!-- Theme -->
        <button v-ripple class="topbar-btn" @click="toggleTheme" :title="isDark ? t('header.lightMode') : t('header.darkMode')" :aria-label="isDark ? t('header.lightMode') : t('header.darkMode')">
          <n-icon :component="isDark ? SunnyOutline : MoonOutline" :size="16" />
        </button>
      </div>
    </header>

    <!-- ===== Body ===== -->
    <div class="app-body" :class="{ 'is-home': activeApp === 'home' }">
      <div class="nav-zone" :class="{ 'nav-zone--home': activeApp === 'home' }" @mouseenter="handleNavEnter" @mouseleave="handleNavLeave">
        <AppRail :active-app="activeApp" @switch="switchApp" @change-password="showPasswordModal = true" />
        <AppSidebar
          v-if="activeApp !== 'home'"
          :sections="menuSections"
          :active-key="activeMenuKey"
          :collapsed="collapsed"
          :pinned="pinned"
          :app-name="activeAppLabel"
          :active-app="activeApp"
          @toggle-collapse="toggleCollapse"
        />
      </div>

      <!-- Main -->
      <main id="main-content" class="main">
        <div v-if="activeApp !== 'home'" class="main-header">
          <h1 class="main-title">{{ pageTitle }}</h1>
          <div class="main-actions">
            <slot name="actions" />
          </div>
        </div>
        <div class="main-content" :data-app="activeApp" aria-live="polite" aria-atomic="false">
          <router-view v-slot="{ Component, route }">
            <Transition name="page" mode="out-in">
              <component :is="Component" :key="route.path + ':' + routerViewKey" />
            </Transition>
          </router-view>
        </div>
      </main>
    </div>

    <!-- Mobile Navigation Drawer -->
    <NDrawer v-model:show="showMobileNav" placement="left" :width="280">
      <NDrawerContent :title="activeAppLabel" :native-scrollbar="false">
        <div class="mobile-nav-list">
          <template v-for="section in menuSections" :key="section.label">
            <div class="mobile-nav-section-title">{{ section.label }}</div>
            <router-link
              v-for="item in section.items"
              :key="item.key"
              :to="item.key"
              class="mobile-nav-item"
              :class="{ active: activeMenuKey === item.key }"
              @click="showMobileNav = false"
            >
              <n-icon v-if="item.icon" :component="item.icon" :size="16" />
              <span>{{ item.label }}</span>
            </router-link>
          </template>
        </div>
      </NDrawerContent>
    </NDrawer>

    <ChangePasswordModal v-model:show="showPasswordModal" />
    <CommandPalette />

    <!-- Keyboard Shortcuts Help (FE8-6) -->
    <NModal v-model:show="showShortcuts" preset="card" :title="t('shortcuts.title') || 'Keyboard Shortcuts'" style="max-width: 480px">
      <div class="shortcuts-list">
        <div v-for="s in shortcuts" :key="s.keys" class="shortcut-row">
          <kbd class="shortcut-keys">{{ s.keys }}</kbd>
          <span class="shortcut-desc">{{ s.desc }}</span>
        </div>
      </div>
    </NModal>

    <!-- Session Expired Modal -->
    <NModal v-model:show="showSessionExpiredModal" preset="card" :title="sessionExpiredTitle" style="max-width: 420px" :closable="false" :mask-closable="false">
      <div style="text-align: center; padding: 12px 0;">
        <div style="font-size: 48px; margin-bottom: 16px;">🔐</div>
        <p style="font-size: 14px; color: var(--sre-text-secondary); margin-bottom: 8px;">
          {{ sessionExpiredDesc }}
        </p>
        <p style="font-size: 13px; color: var(--sre-text-tertiary);">
          {{ t('session.autoRedirect') }}: <strong style="color: var(--sre-primary); font-size: 16px;">{{ sessionRedirectCountdown }}s</strong>
        </p>
      </div>
      <template #action>
        <div style="display: flex; justify-content: center; gap: 12px;">
          <button class="session-btn session-btn-secondary" @click="dismissSessionExpired">
            {{ t('session.stayHere') }}
          </button>
          <button class="session-btn session-btn-primary" @click="doSessionRedirect">
            {{ t('session.goLogin') }}
          </button>
        </div>
      </template>
    </NModal>

    <!-- AI Chat floating button + drawer -->
    <AIChatButton :active="showAIChat" @click="toggleAIChat()" />
    <AIChatPanel v-model:show="showAIChat" />

    <!-- Floating Ask AI button -->
    <button class="float-ai-btn" @click="toggleAIChat()" :title="t('ai.askAI')" :aria-label="t('ai.askAI')">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z" fill="white"/>
        <path d="M19 13l.75 2.25L22 16l-2.25.75L19 19l-.75-2.25L16 16l2.25-.75L19 13z" fill="white" opacity="0.7"/>
        <path d="M5 13l.75 2.25L8 16l-2.25.75L5 19l-.75-2.25L2 16l2.25-.75L5 13z" fill="white" opacity="0.7"/>
      </svg>
      <span class="float-ai-label">{{ t('ai.askAI') }}</span>
    </button>
    </template><!-- /v-else -->
  </div>
</template>

<style scoped>
/* ============================================================
   App Shell — v6.0 Warm Vibrant
   ============================================================ */
.app-shell { display:flex; flex-direction:column; height:100vh; overflow:hidden; }

/* ===== Connection Banner ===== */
.connection-banner {
  flex-shrink: 0;
  border-radius: 0;
}
.banner-enter-active {
  transition: all 300ms var(--sre-ease-out);
}
.banner-leave-active {
  transition: all 200ms var(--sre-ease-out);
}
.banner-enter-from, .banner-leave-to {
  opacity: 0;
  transform: translateY(-100%);
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
  margin-top: 0;
  margin-bottom: 0;
}

/* ===== Top Bar ===== */
.topbar {
  display:flex; align-items:center; justify-content:space-between;
  height:var(--sre-topbar-h); padding:0 16px; flex-shrink:0;
  background:var(--sre-bg-card);
  border-bottom:1px solid var(--sre-border);
  box-shadow: 0 1px 0 0 rgba(13, 148, 136, 0.06);
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
.logo-img { width:24px; height:24px; border-radius:var(--sre-radius-sm); }
.logo-label { font-size:15px; font-weight:600; font-family:var(--sre-font-display); color:var(--sre-text-primary); letter-spacing:-0.01em; white-space:nowrap; }

.topbar-sep {
  width:1px; height:20px; background:var(--sre-border); margin:0 12px; opacity:0.6;
}

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
  padding:5px 11px; gap:6px; background:var(--sre-bg-card);
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
.tz-panel-title { display:flex; align-items:center; gap:8px; padding:8px 16px 6px; font-size:11px; font-weight:600; color:var(--sre-text-tertiary); letter-spacing:0.06em; text-transform:uppercase; border-bottom:1px solid var(--sre-border); margin-bottom:4px; }
.tz-item { display:flex; align-items:center; gap:8px; padding:7px 16px; cursor:pointer; font-size:13px; color:var(--sre-text-primary); transition:background var(--sre-duration-fast) var(--sre-ease-out); margin:0 4px; border-radius:6px; }
.tz-item:hover { background:var(--sre-bg-hover); }
.tz-item.selected { color:var(--sre-primary); background:var(--sre-primary-soft); }
.tz-abbr { font-weight:700; font-size:11px; width:32px; color:var(--sre-primary); flex-shrink:0; }
.tz-label { flex:1; }
.tz-check { font-weight:700; color:var(--sre-primary); font-size:12px; }

/* ===== App Body ===== */
.app-body { display:flex; flex:1; min-height:0; }
.app-body.is-home .main-content { padding:0; background:var(--sre-bg-page); }
.nav-zone { display:flex; height:100%; flex-shrink:0; }
.nav-zone--home { width:auto; }

/* ===== Main Content ===== */
.main { flex:1; min-width:0; display:flex; flex-direction:column; overflow:hidden; }

.main-header {
  display:flex; align-items:center; justify-content:space-between;
  padding:14px 20px; flex-shrink:0;
  border-bottom:1px solid var(--sre-border);
  background:var(--sre-bg-card);
}
.main-title {
  font-size:18px; font-weight:600; font-family:var(--sre-font-display); color:var(--sre-text-primary);
  letter-spacing:-0.01em; margin:0; line-height:1.2;
}
.main-actions { display:flex; align-items:center; gap:8px; }

.main-content {
  flex:1; overflow-y:auto; padding:20px;
  background: var(--sre-bg-page);
}

/* Skip-to-content link */
.skip-to-content {
  position: absolute;
  top: -40px;
  left: 0;
  background: var(--sre-primary);
  color: var(--sre-text-inverse);
  padding: 8px 16px;
  z-index: 100;
  transition: top 0.2s;
}
.skip-to-content:focus,
.skip-to-content:focus-visible {
  top: 0;
  outline: 2px solid var(--sre-text-inverse);
  outline-offset: 2px;
}

/* Page transition animation — smooth fade + scale */
.page-enter-active {
  transition: opacity 280ms var(--sre-ease-out), transform 280ms var(--sre-ease-out);
}
.page-leave-active {
  transition: opacity 180ms var(--sre-ease-out), transform 180ms var(--sre-ease-out);
}
.page-enter-from {
  opacity: 0;
  transform: scale(0.985) translateY(4px);
}
.page-leave-to {
  opacity: 0;
  transform: scale(0.99) translateY(-2px);
}

/* ===== Floating Buttons ===== */
.float-ai-btn {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: var(--sre-z-tooltip);
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  border-radius: 28px;
  border: none;
  background: linear-gradient(135deg, #0D9488, #14B8A6);
  color: white;
  cursor: pointer;
  box-shadow: 0 4px 16px rgba(13, 148, 136, 0.3);
  transition: transform 200ms var(--sre-ease-out), box-shadow 200ms var(--sre-ease-out);
  font-size: 14px;
  font-weight: 600;
}
.float-ai-btn:hover {
  transform: translateY(-2px) scale(1.02);
  box-shadow: 0 6px 24px rgba(13, 148, 136, 0.4);
}
.float-ai-btn:active {
  transform: scale(0.97);
}

.float-ai-label {
  font-size: 13px;
  font-weight: 600;
}

/* ===== Error Boundary ===== */
.error-boundary {
  display: flex; align-items: center; justify-content: center;
  height: 100vh; padding: 40px;
  background: var(--sre-bg-page);
}
.error-reset-btn {
  padding: 8px 24px; border: none; border-radius: var(--sre-radius-sm);
  background: var(--sre-primary); color: #fff;
  font-size: 14px; font-weight: 600; cursor: pointer;
  transition: opacity var(--sre-duration-fast) var(--sre-ease-out);
}
.error-reset-btn:hover { opacity: 0.85; }
.error-actions { display: flex; gap: 8px; justify-content: center; flex-wrap: wrap; }
.error-copy-btn {
  padding: 8px 24px; border: 1px solid var(--sre-border); border-radius: var(--sre-radius-sm);
  background: transparent; color: var(--sre-text-secondary);
  font-size: 14px; font-weight: 500; cursor: pointer; font-family: var(--sre-font-sans);
  transition: background var(--sre-duration-fast) var(--sre-ease-out), color var(--sre-duration-fast) var(--sre-ease-out);
}
.error-copy-btn:hover { background: var(--sre-bg-hover); color: var(--sre-primary); }

/* ===== Session Expired Modal ===== */
.session-btn {
  padding: 8px 24px;
  border: none;
  border-radius: var(--sre-radius-sm);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity var(--sre-duration-fast) var(--sre-ease-out);
  font-family: var(--sre-font-sans);
}
.session-btn:hover { opacity: 0.85; }
.session-btn-primary {
  background: var(--sre-primary);
  color: #fff;
}
.session-btn-secondary {
  background: transparent;
  color: var(--sre-text-secondary);
  border: 1px solid var(--sre-border);
}
.session-btn-secondary:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
}

/* ===== Keyboard Shortcuts (FE8-6) ===== */
.shortcuts-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.shortcut-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.shortcut-keys {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  padding: 3px 8px;
  border-radius: 4px;
  background: var(--sre-bg-elevated);
  border: 1px solid var(--sre-border);
  color: var(--sre-text-primary);
  white-space: nowrap;
}
.shortcut-desc {
  font-size: 13px;
  color: var(--sre-text-secondary);
}

/* Mobile hamburger — hidden on desktop */
.mobile-hamburger {
  display: none;
}

/* Mobile nav drawer */
.mobile-nav-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.mobile-nav-section-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--sre-text-tertiary);
  padding: 12px 8px 4px;
  margin-top: 8px;
}
.mobile-nav-section-title:first-child {
  margin-top: 0;
}
.mobile-nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 14px;
  color: var(--sre-text-secondary);
  text-decoration: none;
  transition: background 120ms ease, color 120ms ease;
}
.mobile-nav-item:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
}
.mobile-nav-item.active {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
  font-weight: 600;
}

/* ===== Responsive ===== */
@media (max-width: 768px) {
  .mobile-hamburger {
    display: inline-flex;
  }
  .nav-zone {
    display: none;
  }
  .main-content {
    padding: 12px;
  }
  .main-header {
    padding: 10px 12px;
  }
  .topbar-sep {
    display: none;
  }
  .logo-label {
    display: none;
  }
  .topbar-kbd {
    display: none;
  }
  .float-ai-btn {
    bottom: 16px;
    right: 16px;
    padding: 10px 16px;
  }
  .float-ai-label {
    display: none;
  }
}
</style>
