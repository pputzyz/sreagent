<script setup lang="ts">
import { ref, computed, watch, inject, onMounted, onUnmounted } from 'vue'
import type { Ref } from 'vue'
import { NIcon, NPopover, NPopselect } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useAppNav } from '@/composables/useAppNav'
import { useCommandPalette } from '@/composables/useCommandPalette'
import { useAuthStore } from '@/stores/auth'
import AppRail from '@/layouts/AppRail.vue'
import AppSidebar from '@/layouts/AppSidebar.vue'
import CommandPalette from '@/components/common/CommandPalette.vue'
import ChangePasswordModal from '@/components/common/ChangePasswordModal.vue'
import AIChatButton from '@/components/ai/AIChatButton.vue'
import AIChatPanel from '@/components/ai/AIChatPanel.vue'
import PetCorner from '@/components/pet/PetCorner.vue'
import { useRouter } from 'vue-router'
import { TimeOutline, EarthOutline, SunnyOutline, MoonOutline } from '@vicons/ionicons5'
import { MessageCircle, PawPrint } from 'lucide-vue-next'
import { usePetStore } from '@/stores/pet'

const { t, locale } = useI18n()
const authStore = useAuthStore()
const { activeApp, switchApp, menuSections, activeMenuKey, pageTitle } = useAppNav()
const { open: openPalette, registerAction } = useCommandPalette()

const router = useRouter()
const petStore = usePetStore()

const isDark = inject<Ref<boolean>>('isDark', ref(true))
const toggleTheme = inject<() => void>('toggleTheme', () => {})

// ===== Profile fetch =====
onMounted(() => {
  if (authStore.isLoggedIn && !authStore.user) authStore.fetchProfile()

  // ── Command palette actions ──────────────────────────────────
  registerAction({
    id: 'act-theme',
    label: t('command.toggleDarkMode'),
    hint: t('command.action'),
    icon: 'contrast-outline',
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
const collapsed = ref(safeParse(localStorage.getItem('sre-sider-collapsed'), false))
watch(collapsed, v => localStorage.setItem('sre-sider-collapsed', JSON.stringify(v)))
const showPasswordModal = ref(false)
const showAIChat = ref(false)
const pinned = ref(false)

function toggleAIChat() {
  showAIChat.value = !showAIChat.value
}

function openPetChat() {
  showAIChat.value = true
}

function toggleCollapse() {
  collapsed.value = !collapsed.value
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
onMounted(() => { updateClock(); clockInterval = setInterval(updateClock, 1000) })
onUnmounted(() => clearInterval(clockInterval))

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
    <a href="#main-content" class="skip-to-content">{{ t('a11y.skipToContent') }}</a>
    <!-- ===== Top Bar ===== -->
    <header class="topbar">
      <div class="topbar-start">
        <!-- Logo -->
        <router-link to="/dashboard" class="topbar-logo">
          <img src="/logo.svg" alt="SREAgent" class="logo-img" />
          <span class="logo-label">SREAgent</span>
        </router-link>

        <div class="topbar-sep" />
      </div>

      <div class="topbar-end">
        <!-- Clock -->
        <n-popover v-model:show="showTzPanel" trigger="click" placement="bottom-end" :show-arrow="false" style="padding:0">
          <template #trigger>
            <button v-ripple class="topbar-btn topbar-clock" :class="{ active: showTzPanel }">
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
              @click="selectTimezone(opt.value)"
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

        <!-- Lang -->
        <n-popselect :value="locale" :options="langOptions" trigger="click" :render-label="(o: any) => o.label" @update:value="handleLangChange">
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
    <div class="app-body">
      <div class="nav-zone" @mouseenter="handleNavEnter" @mouseleave="handleNavLeave">
        <AppRail :active-app="activeApp" @switch="switchApp" @change-password="showPasswordModal = true" />
        <AppSidebar
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
        <div class="main-header">
          <h1 class="main-title">{{ pageTitle }}</h1>
          <div class="main-actions">
            <slot name="actions" />
          </div>
        </div>
        <div class="main-content" :data-app="activeApp">
          <router-view v-slot="{ Component, route }">
            <Transition name="page" mode="out-in">
              <component :is="Component" :key="route.path" />
            </Transition>
          </router-view>
        </div>
      </main>
    </div>

    <ChangePasswordModal v-model:show="showPasswordModal" />
    <CommandPalette />

    <!-- AI Chat floating button + drawer -->
    <AIChatButton :active="showAIChat" @click="toggleAIChat()" />
    <AIChatPanel v-model:show="showAIChat" />

    <!-- Floating Ask AI button -->
    <button class="float-ai-btn" @click="toggleAIChat()" :title="t('ai.askAI')">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z" fill="white"/>
        <path d="M19 13l.75 2.25L22 16l-2.25.75L19 19l-.75-2.25L16 16l2.25-.75L19 13z" fill="white" opacity="0.7"/>
        <path d="M5 13l.75 2.25L8 16l-2.25.75L5 19l-.75-2.25L2 16l2.25-.75L5 13z" fill="white" opacity="0.7"/>
      </svg>
      <span class="float-ai-label">{{ t('ai.askAI') }}</span>
    </button>

    <!-- Floating Pet button with level/attribute summary -->
    <div class="float-pet-wrap">
      <button class="float-pet-btn" @click="router.push('/pet')" :title="t('pet.viewDetail')">
        <PawPrint :size="20" color="white" :stroke-width="2" />
      </button>
      <div v-if="petStore.pet" class="float-pet-summary">
        <span class="float-pet-name">{{ petStore.pet.name }}</span>
        <span class="float-pet-level">Lv.{{ petStore.pet.level }}</span>
        <div class="float-pet-bars">
          <div class="float-pet-bar" :title="`${t('pet.hunger')}: ${petStore.hungerPercent}%`">
            <div class="float-pet-bar-fill hunger" :style="{ width: `${petStore.hungerPercent}%` }" />
          </div>
          <div class="float-pet-bar" :title="`${t('pet.mood')}: ${petStore.moodPercent}%`">
            <div class="float-pet-bar-fill mood" :style="{ width: `${petStore.moodPercent}%` }" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ============================================================
   App Shell — v5.0 Clean Neutral
   ============================================================ */
.app-shell { display:flex; flex-direction:column; height:100vh; overflow:hidden; }

/* ===== Top Bar ===== */
.topbar {
  display:flex; align-items:center; justify-content:space-between;
  height:var(--sre-topbar-h); padding:0 16px; flex-shrink:0;
  background:var(--sre-bg-card);
  border-bottom:1px solid var(--sre-border);
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
.logo-label { font-size:15px; font-weight:600; color:var(--sre-text-primary); letter-spacing:-0.01em; white-space:nowrap; }

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
.nav-zone { display:flex; height:100%; flex-shrink:0; }

/* ===== Main Content ===== */
.main { flex:1; min-width:0; display:flex; flex-direction:column; overflow:hidden; }

.main-header {
  display:flex; align-items:center; justify-content:space-between;
  padding:14px 20px; flex-shrink:0;
  border-bottom:1px solid var(--sre-border);
  background:var(--sre-bg-card);
}
.main-title {
  font-size:18px; font-weight:600; color:var(--sre-text-primary);
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
.skip-to-content:focus {
  top: 0;
}

/* Page transition animation */
.page-enter-active {
  animation: sre-page-enter 300ms var(--sre-ease-out) both;
}
.page-leave-active {
  animation: sre-page-enter 200ms var(--sre-ease-out) reverse both;
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
  background: linear-gradient(135deg, #FF6B6B, #FF8E8E);
  color: white;
  cursor: pointer;
  box-shadow: 0 4px 16px rgba(255, 107, 107, 0.3);
  transition: transform 200ms var(--sre-ease-out), box-shadow 200ms var(--sre-ease-out);
  font-size: 14px;
  font-weight: 600;
}
.float-ai-btn:hover {
  transform: translateY(-2px) scale(1.02);
  box-shadow: 0 6px 24px rgba(255, 107, 107, 0.4);
}
.float-ai-btn:active {
  transform: scale(0.97);
}

.float-pet-wrap {
  position: fixed;
  bottom: 88px;
  right: 24px;
  z-index: var(--sre-z-tooltip);
  display: flex;
  align-items: center;
  gap: 10px;
  flex-direction: row-reverse;
}

.float-pet-btn {
  width: 48px; height: 48px;
  border-radius: 50%;
  border: none;
  background: linear-gradient(135deg, #A855F7, #C084FC);
  color: white;
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  box-shadow: 0 4px 12px rgba(168, 85, 247, 0.3);
  transition: transform 200ms var(--sre-ease-out), box-shadow 200ms var(--sre-ease-out);
  flex-shrink: 0;
}
.float-pet-btn:hover {
  transform: scale(1.08);
  box-shadow: 0 6px 20px rgba(168, 85, 247, 0.4);
}

.float-pet-summary {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-md);
  padding: 6px 10px;
  display: flex;
  flex-direction: column;
  gap: 3px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  opacity: 0;
  transform: translateX(8px);
  transition: opacity 200ms var(--sre-ease-out), transform 200ms var(--sre-ease-out);
  pointer-events: none;
  min-width: 100px;
}

.float-pet-wrap:hover .float-pet-summary {
  opacity: 1;
  transform: translateX(0);
  pointer-events: auto;
}

.float-pet-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-primary);
  white-space: nowrap;
}

.float-pet-level {
  font-size: 11px;
  font-weight: 700;
  color: var(--sre-lavender);
}

.float-pet-bars {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 2px;
}

.float-pet-bar {
  height: 4px;
  background: var(--sre-bg-sunken);
  border-radius: 2px;
  overflow: hidden;
}

.float-pet-bar-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 500ms var(--sre-ease-out);
}

.float-pet-bar-fill.hunger {
  background: var(--sre-coral);
}

.float-pet-bar-fill.mood {
  background: var(--sre-amber);
}

.float-ai-label {
  font-size: 13px;
  font-weight: 600;
}

/* ===== Responsive ===== */
@media (max-width: 768px) {
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
  .float-pet-wrap {
    bottom: 76px;
    right: 16px;
  }
  .float-pet-summary {
    display: none;
  }
}
</style>
