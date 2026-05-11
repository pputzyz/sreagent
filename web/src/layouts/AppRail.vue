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
            v-ripple
            class="rail-icon-btn"
            :class="{ active: activeApp === item.key }"
            @click="emit('switch', item.key)"
          >
            <n-icon :component="item.icon" :size="22" />
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
      <!-- User avatar with popover menu -->
      <n-popover
        trigger="click"
        placement="right"
        :show-arrow="false"
        v-model:show="showUserMenu"
      >
        <template #trigger>
          <button class="rail-avatar-btn" :class="{ active: showUserMenu }">
            <n-avatar
              v-if="authStore.user?.avatar && !avatarError"
              :src="authStore.user.avatar"
              :size="28"
              round
              @error="avatarError = true"
            />
            <n-avatar v-else :size="28" round :style="{ fontSize: '12px', fontWeight: 700 }">{{ userInitial }}</n-avatar>
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
            v-ripple
            class="rail-icon-btn"
            :class="{ active: activeApp === 'platform' }"
            @click="emit('switch', 'platform')"
          >
            <n-icon :component="SettingsOutline" :size="22" />
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
  width: 48px;
  height: 100%;
  background: var(--sre-bg-base);
  border-right: 1px solid var(--sre-border);
  flex-shrink: 0;
}

.rail-top,
.rail-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 6px;
}

.rail-spacer {
  flex: 1;
}

.rail-icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  transition:
    background var(--sre-duration-base) var(--sre-ease-out),
    color var(--sre-duration-base) var(--sre-ease-out),
    box-shadow var(--sre-duration-base) var(--sre-ease-out);
}

.rail-icon-btn:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
  box-shadow: var(--sre-shadow-xs);
}

.rail-icon-btn:active {
  transform: scale(0.92);
  transition-duration: 80ms;
}

.rail-icon-btn.active {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
  box-shadow: inset 3px 0 0 var(--sre-primary);
}

/* Rail tooltip content */
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

/* User avatar button */
.rail-avatar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: none;
  border-radius: 50%;
  background: transparent;
  cursor: pointer;
  padding: 0;
  transition: box-shadow var(--sre-duration-base) var(--sre-ease-out);
}

.rail-avatar-btn:hover {
  box-shadow: 0 0 0 2px var(--sre-primary-ring);
}

.rail-avatar-btn.active {
  box-shadow: 0 0 0 2px var(--sre-primary);
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
  transition:
    background var(--sre-duration-fast) var(--sre-ease-out),
    color var(--sre-duration-fast) var(--sre-ease-out);
  border-radius: 0;
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
