<script setup lang="ts">
import { computed, h } from 'vue'
import { useRouter } from 'vue-router'
import { NMenu, NIcon } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { ChevronBackOutline, ChevronForwardOutline } from '@vicons/ionicons5'
import type { MenuSection } from '@/composables/useAppNav'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  sections: MenuSection[]
  activeKey: string
  collapsed: boolean
}>()

const emit = defineEmits<{
  'update:collapsed': [value: boolean]
  navigate: [key: string]
}>()

const router = useRouter()
const authStore = useAuthStore()
const appVersion = __APP_VERSION__

// ===== Convert MenuSection[] → Naive UI MenuOption[] =====

const menuOptions = computed<MenuOption[]>(() => {
  const result: MenuOption[] = []

  for (const section of props.sections) {
    // Filter out items where show === false
    const visibleItems = section.items
      .filter(item => item.show !== false)
      .map(item => ({
        label: item.label,
        key: item.key,
        icon: item.icon ? () => h(NIcon, null, { default: () => h(item.icon!) }) : undefined,
        children: item.children
          ?.filter(child => child.show !== false)
          .map(child => ({
            label: child.label,
            key: child.key,
            icon: child.icon ? () => h(NIcon, null, { default: () => h(child.icon!) }) : undefined,
          })),
      }))

    if (visibleItems.length === 0) continue

    if (section.label) {
      result.push({
        type: 'group',
        label: section.label,
        children: visibleItems,
      })
    } else {
      result.push(...visibleItems)
    }
  }

  return result
})

// ===== Menu click handler =====

function handleMenuUpdate(key: string) {
  emit('navigate', key)
  router.push(key)
}

// ===== Collapse toggle =====

function toggleCollapse() {
  emit('update:collapsed', !props.collapsed)
}

// ===== User info =====

const userInitial = computed(() =>
  (authStore.user?.display_name || authStore.user?.username || 'U').charAt(0).toUpperCase(),
)

const displayName = computed(() =>
  authStore.user?.display_name || authStore.user?.username || 'User',
)

const userRole = computed(() =>
  authStore.canManage ? 'Admin' : 'Member',
)
</script>

<template>
  <aside class="app-sidebar" :class="{ collapsed }">
    <!-- Nav area -->
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

    <!-- Spacer -->
    <div class="sidebar-spacer" />

    <!-- User area -->
    <div class="sidebar-bottom">
      <div class="sidebar-user">
        <div class="user-avatar">{{ userInitial }}</div>
        <transition name="fade">
          <div v-if="!collapsed" class="user-meta">
            <span class="user-name">{{ displayName }}</span>
            <span class="user-role">{{ userRole }}</span>
          </div>
        </transition>
      </div>

      <button class="sidebar-collapse-btn" @click="toggleCollapse">
        <n-icon :component="collapsed ? ChevronForwardOutline : ChevronBackOutline" :size="14" />
        <transition name="fade">
          <span v-if="!collapsed" class="collapse-label">收起侧栏</span>
        </transition>
      </button>

      <transition name="fade">
        <div v-if="!collapsed" class="sidebar-version">v{{ appVersion }}</div>
      </transition>
    </div>
  </aside>
</template>

<style scoped>
.app-sidebar {
  display: flex;
  flex-direction: column;
  width: 220px;
  height: 100%;
  background: var(--sre-bg-card);
  border-right: 1px solid var(--sre-border);
  flex-shrink: 0;
  transition: width 280ms var(--sre-ease-spring);
  overflow: hidden;
}

.app-sidebar.collapsed {
  width: 64px;
}

/* Nav area — scrollable */
.sidebar-nav {
  flex: 0 1 auto;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px;
}

/* Spacer */
.sidebar-spacer {
  flex: 1;
}

/* Bottom user area */
.sidebar-bottom {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  border-top: 1px solid var(--sre-border);
}

/* User row */
.sidebar-user {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--sre-radius-md);
  min-height: 40px;
}

.app-sidebar.collapsed .sidebar-user {
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
  line-height: 1.2;
}

.user-role {
  font-size: 10px;
  color: var(--sre-text-tertiary);
  line-height: 1.3;
}

/* Collapse button */
.sidebar-collapse-btn {
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
  font-family: var(--sre-font-sans);
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  transition:
    background var(--sre-duration-fast) var(--sre-ease-out),
    color var(--sre-duration-fast) var(--sre-ease-out);
}

.app-sidebar.collapsed .sidebar-collapse-btn {
  justify-content: center;
}

.sidebar-collapse-btn:hover {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
}

/* Version */
.sidebar-version {
  text-align: center;
  font-size: 10px;
  color: var(--sre-text-tertiary);
  opacity: 0.5;
}

/* Fade transition */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
