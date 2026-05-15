<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { NTooltip, NPopover } from 'naive-ui'
import { Zap, Bell, Settings, User, KeyRound, LogOut } from 'lucide-vue-next'
import UserAvatar from '@/components/common/UserAvatar.vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import type { AppKey } from '@/composables/useAppNav'

defineProps<{
  activeApp: AppKey
}>()

const emit = defineEmits<{
  switch: [app: AppKey]
  'change-password': []
}>()

const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()

const showUserMenu = ref(false)

const avatarPreset = computed(() => {
  const uid = authStore.user?.id
  if (!uid) return undefined
  return localStorage.getItem(`sre-avatar-preset-${uid}`) || undefined
})

interface RailItem {
  key: AppKey
  icon: typeof Zap
  label: string
  desc: string
  colorClass: string
}

const topItems: RailItem[] = [
  { key: 'oncall', icon: Zap, label: t('rail.oncall'), desc: t('rail.oncallDesc'), colorClass: 'nav-icon-oncall' },
  { key: 'alert', icon: Bell, label: t('rail.alert'), desc: t('rail.alertDesc'), colorClass: 'nav-icon-alert' },
]

const userInitial = computed(() =>
  (authStore.user?.display_name || authStore.user?.username || 'U').charAt(0).toUpperCase(),
)

const displayName = computed(() =>
  authStore.user?.display_name || authStore.user?.username || t('header.defaultUser'),
)

const userRole = computed(() =>
  authStore.canManage ? t('settings.admin') : t('settings.member'),
)

function goToProfile() {
  showUserMenu.value = false
  router.push('/platform/profile')
}

function handleChangePassword() {
  showUserMenu.value = false
  emit('change-password')
}

function handleLogout() {
  showUserMenu.value = false
  authStore.logout()
  router.push('/login')
}
</script>

<template>
  <aside class="app-rail">
    <div class="rail-top">
      <n-tooltip
        v-for="item in topItems"
        :key="item.key"
        placement="right"
        :show-arrow="false"
      >
        <template #trigger>
          <button
            class="rail-icon-btn"
            :class="{ active: activeApp === item.key }"
            :data-app="item.key"
            @click="emit('switch', item.key)"
          >
            <div class="rail-icon-circle" :class="item.colorClass">
              <component :is="item.icon" :size="18" color="white" :stroke-width="2" />
            </div>
            <span class="rail-dot" />
          </button>
        </template>
        <div class="rail-tooltip-content">
          <div class="rail-tooltip-title">{{ item.label }}</div>
          <div class="rail-tooltip-desc">{{ item.desc }}</div>
        </div>
      </n-tooltip>
    </div>

    <div class="rail-spacer" />

    <div class="rail-bottom">
      <!-- User avatar -->
      <n-popover
        trigger="click"
        placement="right"
        :show-arrow="false"
        v-model:show="showUserMenu"
      >
        <template #trigger>
          <button class="rail-avatar-btn" :class="{ active: showUserMenu }">
            <UserAvatar
              :src="authStore.user?.avatar || undefined"
              :preset-id="avatarPreset"
              :name="displayName"
              :size="30"
            />
          </button>
        </template>

        <div class="user-popover">
          <div class="user-popover-header">
            <UserAvatar
              :src="authStore.user?.avatar || undefined"
              :preset-id="avatarPreset"
              :name="displayName"
              :size="32"
            />
            <div class="user-popover-info">
              <div class="user-popover-name">{{ displayName }}</div>
              <div class="user-popover-role">{{ userRole }}</div>
            </div>
          </div>
          <div class="user-popover-divider" />
          <div class="user-popover-item" @click="goToProfile">
            <User :size="16" />
            <span>{{ t('header.profile') }}</span>
          </div>
          <div class="user-popover-item" @click="handleChangePassword">
            <KeyRound :size="16" />
            <span>{{ t('header.changePassword') }}</span>
          </div>
          <div class="user-popover-divider" />
          <div class="user-popover-item user-popover-item--danger" @click="handleLogout">
            <LogOut :size="16" />
            <span>{{ t('header.logout') }}</span>
          </div>
        </div>
      </n-popover>

      <!-- Platform settings icon -->
      <n-tooltip
        placement="right"
        :show-arrow="false"
      >
        <template #trigger>
          <button
            class="rail-icon-btn"
            :class="{ active: activeApp === 'platform' }"
            data-app="platform"
            @click="emit('switch', 'platform')"
          >
            <div class="rail-icon-circle nav-icon-platform">
              <Settings :size="18" color="white" :stroke-width="2" />
            </div>
            <span class="rail-dot" />
          </button>
        </template>
        <div class="rail-tooltip-content">
          <div class="rail-tooltip-title">{{ t('rail.platform') }}</div>
          <div class="rail-tooltip-desc">{{ t('rail.platformDesc') }}</div>
        </div>
      </n-tooltip>
    </div>
  </aside>
</template>

<style scoped>
.app-rail {
  display: flex;
  flex-direction: column;
  width: 56px;
  height: 100%;
  background: var(--sre-bg-card);
  border-right: 1px solid var(--sre-border);
  flex-shrink: 0;
}

.rail-top,
.rail-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 12px 8px;
}

.rail-spacer {
  flex: 1;
}

/* Icon button */
.rail-icon-btn {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: none;
  border-radius: var(--sre-radius-md);
  background: transparent;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  padding: 0;
  transition:
    background var(--sre-duration-fast) var(--sre-ease-out),
    color var(--sre-duration-fast) var(--sre-ease-out),
    transform 200ms var(--sre-ease-spring);
}

.rail-icon-btn:hover {
  background: color-mix(in srgb, var(--sre-bg-hover) 100%, transparent);
  color: var(--sre-text-secondary);
  transform: translateY(-1px);
}

.rail-icon-btn.active {
  background: var(--sre-bg-active);
  color: var(--sre-text-primary);
}

.rail-icon-btn:active {
  transform: translateY(0) scale(0.97);
  transition-duration: 80ms;
}

/* Colored icon circles */
.rail-icon-circle {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition:
    transform 300ms var(--sre-ease-spring),
    box-shadow 300ms var(--sre-ease-out);
}

.rail-icon-btn:hover .rail-icon-circle {
  transform: scale(1.1);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.rail-icon-btn:active .rail-icon-circle {
  transform: scale(0.92);
}

.rail-icon-btn.active .rail-icon-circle {
  box-shadow: 0 0 0 2px var(--sre-primary-ring);
}

/* Per-app active glow rings */
.rail-icon-btn[data-app="oncall"].active .rail-icon-circle {
  box-shadow: 0 0 0 3px rgba(244, 63, 94, 0.25), 0 4px 12px rgba(244, 63, 94, 0.12);
}

.rail-icon-btn[data-app="alert"].active .rail-icon-circle {
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.25), 0 4px 12px rgba(59, 130, 246, 0.12);
}

.rail-icon-btn[data-app="platform"].active .rail-icon-circle {
  box-shadow: 0 0 0 3px rgba(139, 92, 246, 0.25), 0 4px 12px rgba(139, 92, 246, 0.12);
}

.nav-icon-oncall { background: var(--sre-brand-oncall); }
.nav-icon-alert { background: var(--sre-brand-alert); }
.nav-icon-platform { background: var(--sre-brand-platform); }

/* Colored dot indicator — only visible when active */
.rail-dot {
  position: absolute;
  right: 6px;
  top: 6px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  opacity: 0;
  transition: opacity var(--sre-duration-fast) var(--sre-ease-out);
}

.rail-icon-btn.active .rail-dot {
  opacity: 1;
}

.rail-icon-btn[data-app="oncall"] .rail-dot   { background: var(--sre-brand-oncall); }
.rail-icon-btn[data-app="alert"] .rail-dot    { background: var(--sre-brand-alert); }
.rail-icon-btn[data-app="platform"] .rail-dot { background: var(--sre-brand-platform); }

/* Rail tooltip */
.rail-tooltip-content {
  padding: 2px 0;
}

.rail-tooltip-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
  line-height: 1.3;
}

.rail-tooltip-desc {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  line-height: 1.3;
  margin-top: 2px;
}

/* User avatar */
.rail-avatar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: none;
  border-radius: var(--sre-radius-md);
  background: transparent;
  cursor: pointer;
  padding: 0;
  transition: background var(--sre-duration-fast) var(--sre-ease-out);
}

.rail-avatar-btn:hover {
  background: var(--sre-bg-hover);
}

.rail-avatar-btn.active {
  background: var(--sre-bg-active);
}

/* User popover */
.user-popover {
  min-width: 200px;
  padding: 4px 0;
}

.user-popover-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
}

.user-popover-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.user-popover-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
  line-height: 1.3;
}

.user-popover-role {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  line-height: 1.3;
}

.user-popover-divider {
  height: 1px;
  background: var(--sre-border);
  margin: 4px 0;
}

.user-popover-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  font-size: 13px;
  color: var(--sre-text-secondary);
  cursor: pointer;
  border-radius: var(--sre-radius-sm);
  margin: 0 4px;
  transition:
    background var(--sre-duration-fast) var(--sre-ease-out),
    color var(--sre-duration-fast) var(--sre-ease-out);
}

.user-popover-item:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
}

.user-popover-item--danger {
  color: var(--sre-critical);
}

.user-popover-item--danger:hover {
  background: var(--sre-critical-soft);
  color: var(--sre-critical);
}
</style>
