<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { NIcon, NTooltip, NPopover, NAvatar } from 'naive-ui'
import { FlashOutline, AlertCircleOutline, SettingsOutline, PersonOutline, KeyOutline, LogOutOutline } from '@vicons/ionicons5'
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
const avatarError = ref(false)

interface RailItem {
  key: AppKey
  icon: typeof FlashOutline
  label: string
  desc: string
}

const topItems: RailItem[] = [
  { key: 'oncall', icon: FlashOutline, label: 'On-Call', desc: t('rail.oncallDesc') },
  { key: 'alert', icon: AlertCircleOutline, label: 'Alert', desc: t('rail.alertDesc') },
]

const userInitial = computed(() =>
  (authStore.user?.display_name || authStore.user?.username || 'U').charAt(0).toUpperCase(),
)

const displayName = computed(() =>
  authStore.user?.display_name || authStore.user?.username || 'User',
)

const userRole = computed(() =>
  authStore.canManage ? 'Admin' : 'Member',
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
            <div class="rail-icon-circle" :data-app="item.key">
              <n-icon :component="item.icon" :size="20" color="#fff" />
            </div>
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
      <!-- User avatar with colorful ring -->
      <n-popover
        trigger="click"
        placement="right"
        :show-arrow="false"
        v-model:show="showUserMenu"
      >
        <template #trigger>
          <button class="rail-avatar-btn" :class="{ active: showUserMenu }">
            <div class="avatar-ring">
              <n-avatar
                v-if="authStore.user?.avatar && !avatarError"
                :src="authStore.user.avatar"
                :size="30"
                round
                @error="avatarError = true"
              />
              <n-avatar v-else :size="30" round :style="{ fontSize: '12px', fontWeight: 700 }">{{ userInitial }}</n-avatar>
            </div>
          </button>
        </template>

        <div class="user-popover">
          <div class="user-popover-header">
            <n-avatar :size="32" round>{{ userInitial }}</n-avatar>
            <div class="user-popover-info">
              <div class="user-popover-name">{{ displayName }}</div>
              <div class="user-popover-role">{{ userRole }}</div>
            </div>
          </div>
          <div class="user-popover-divider" />
          <div class="user-popover-item" @click="goToProfile">
            <n-icon :component="PersonOutline" :size="16" />
            <span>{{ t('header.profile') }}</span>
          </div>
          <div class="user-popover-item" @click="handleChangePassword">
            <n-icon :component="KeyOutline" :size="16" />
            <span>{{ t('header.changePassword') }}</span>
          </div>
          <div class="user-popover-divider" />
          <div class="user-popover-item user-popover-item--danger" @click="handleLogout">
            <n-icon :component="LogOutOutline" :size="16" />
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
            <div class="rail-icon-circle" data-app="platform">
              <n-icon :component="SettingsOutline" :size="20" color="#fff" />
            </div>
          </button>
        </template>
        <div class="rail-tooltip-content">
          <div class="rail-tooltip-title">Platform</div>
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
  border-right: 2px solid var(--sre-border);
  flex-shrink: 0;
  box-shadow: 2px 0 8px rgba(0,0,0,0.04);
}

.rail-top,
.rail-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 12px 8px;
}

.rail-spacer {
  flex: 1;
}

/* Icon button — transparent container */
.rail-icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  border: none;
  border-radius: 14px;
  background: transparent;
  cursor: pointer;
  padding: 0;
  transition:
    transform var(--sre-duration-base) var(--sre-ease-spring),
    background var(--sre-duration-base) var(--sre-ease-out);
}

.rail-icon-btn:hover {
  background: var(--sre-bg-hover);
}

.rail-icon-btn:hover .rail-icon-circle {
  transform: scale(1.12);
  box-shadow: 0 4px 14px rgba(0,0,0,0.18);
}

.rail-icon-btn:active .rail-icon-circle {
  animation: sre-jelly 400ms ease-in-out;
}

.rail-icon-btn.active .rail-icon-circle {
  box-shadow: 0 0 0 3px rgba(255,107,107,0.25), 0 4px 14px rgba(0,0,0,0.15);
  animation: sre-float 3s ease-in-out infinite;
}

/* Icon circle — colorful gradient background */
.rail-icon-circle {
  width: 34px;
  height: 34px;
  border-radius: 11px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition:
    transform var(--sre-duration-base) var(--sre-ease-spring),
    box-shadow var(--sre-duration-base) var(--sre-ease-out);
  box-shadow: 0 2px 8px rgba(0,0,0,0.10);
}

.rail-icon-circle[data-app="oncall"]   { background: var(--sre-gradient-oncall); }
.rail-icon-circle[data-app="alert"]    { background: var(--sre-gradient-alert); }
.rail-icon-circle[data-app="platform"] { background: var(--sre-gradient-platform); }

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

/* User avatar with gradient ring */
.rail-avatar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  border: none;
  border-radius: 50%;
  background: transparent;
  cursor: pointer;
  padding: 0;
  transition: transform var(--sre-duration-base) var(--sre-ease-spring);
}

.rail-avatar-btn:hover {
  transform: scale(1.08);
}

.rail-avatar-btn:active {
  animation: sre-jelly 400ms ease-in-out;
}

.avatar-ring {
  padding: 2px;
  border-radius: 50%;
  background: var(--sre-gradient-rainbow);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: box-shadow var(--sre-duration-base) var(--sre-ease-out);
}

.rail-avatar-btn:hover .avatar-ring {
  box-shadow: 0 0 12px rgba(255,107,107,0.3);
}

.rail-avatar-btn.active .avatar-ring {
  box-shadow: 0 0 16px rgba(255,107,107,0.4);
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
  border-radius: 8px;
  margin: 0 4px;
  transition:
    background var(--sre-duration-fast) var(--sre-ease-out),
    color var(--sre-duration-fast) var(--sre-ease-out),
    transform var(--sre-duration-fast) var(--sre-ease-spring);
}

.user-popover-item:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
  transform: translateX(2px);
}

.user-popover-item--danger {
  color: var(--sre-critical);
}

.user-popover-item--danger:hover {
  background: var(--sre-critical-soft);
  color: var(--sre-critical);
}
</style>
