<script setup lang="ts">
import { computed, h } from 'vue'
import { useRouter } from 'vue-router'
import { NMenu, NIcon } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { ChevronBackOutline, ChevronForwardOutline } from '@vicons/ionicons5'
import type { MenuSection } from '@/composables/useAppNav'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  sections: MenuSection[]
  activeKey: string
  collapsed: boolean
  appName: string
  pinned: boolean
}>()

const emit = defineEmits<{
  'toggle-collapse': []
  navigate: [key: string]
}>()

const router = useRouter()
const { t } = useI18n()

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

</script>

<template>
  <aside class="app-sidebar" :class="{ collapsed }">
    <!-- Header with app name + collapse toggle -->
    <div class="sidebar-header">
      <transition name="fade">
        <span v-if="!collapsed" class="sidebar-app-name">{{ appName }}</span>
      </transition>
      <button class="sidebar-pin-btn" :title="pinned ? t('header.expandSidebar') : t('header.collapseSidebar')" @click="emit('toggle-collapse')">
        <n-icon :component="collapsed ? ChevronForwardOutline : ChevronBackOutline" :size="14" />
      </button>
    </div>

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

/* Sidebar header */
.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 12px 8px;
  border-bottom: 1px solid var(--sre-border);
  min-height: 44px;
}

.sidebar-app-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
}

.sidebar-pin-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--sre-radius-sm);
  background: transparent;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  flex-shrink: 0;
  transition:
    background var(--sre-duration-fast) var(--sre-ease-out),
    color var(--sre-duration-fast) var(--sre-ease-out);
}

.sidebar-pin-btn:hover {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
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
