<script setup lang="ts">
import { computed, h } from 'vue'
import { useRouter } from 'vue-router'
import { NMenu, NIcon } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { ChevronBackOutline, ChevronForwardOutline } from '@vicons/ionicons5'
import type { MenuSection, AppKey } from '@/composables/useAppNav'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  sections: MenuSection[]
  activeKey: string
  collapsed: boolean
  appName: string
  pinned: boolean
  activeApp: AppKey
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
  <aside class="app-sidebar" :class="{ collapsed }" :data-app="activeApp">
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
  --sidebar-accent: var(--sre-primary);
  display: flex;
  flex-direction: column;
  width: 220px;
  height: 100%;
  background: var(--sre-bg-card);
  border-right: 2px solid var(--sre-border);
  flex-shrink: 0;
  transition: width 280ms var(--sre-ease-spring);
  overflow: hidden;
}

.app-sidebar[data-app="oncall"]   { --sidebar-accent: var(--sre-brand-oncall); }
.app-sidebar[data-app="alert"]    { --sidebar-accent: var(--sre-brand-alert); }
.app-sidebar[data-app="platform"] { --sidebar-accent: var(--sre-brand-platform); }

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

/* Naive UI menu overrides — colorful accent */
.sidebar-nav :deep(.n-menu-item) {
  border-radius: 10px;
  margin-bottom: 2px;
  transition:
    background var(--sre-duration-fast) var(--sre-ease-out),
    transform var(--sre-duration-fast) var(--sre-ease-spring);
}

.sidebar-nav :deep(.n-menu-item:hover) {
  background: color-mix(in srgb, var(--sidebar-accent) 8%, transparent);
  transform: translateX(2px);
}

.sidebar-nav :deep(.n-menu-item--selected) {
  background: color-mix(in srgb, var(--sidebar-accent) 12%, transparent) !important;
}

.sidebar-nav :deep(.n-menu-item-content--selected .n-menu-item-content__icon) {
  color: var(--sidebar-accent) !important;
}

.sidebar-nav :deep(.n-menu-item-content--selected .n-menu-item-content__label) {
  color: var(--sidebar-accent) !important;
  font-weight: 600;
}

/* Selected left indicator — gradient accent bar */
.sidebar-nav :deep(.n-menu-item--selected::before) {
  content: '';
  position: absolute;
  left: 4px;
  top: 8px;
  bottom: 8px;
  width: 3px;
  border-radius: 3px;
  background: linear-gradient(180deg, var(--sidebar-accent), color-mix(in srgb, var(--sidebar-accent) 60%, #fff));
  animation: sre-slide-in 300ms var(--sre-ease-spring);
}

@keyframes sre-slide-in {
  from { transform: scaleY(0); opacity: 0; }
  to   { transform: scaleY(1); opacity: 1; }
}

/* Group label */
.sidebar-nav :deep(.n-menu-group-label) {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-tertiary);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding: 12px 12px 4px;
}

/* Sidebar header */
.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 12px 8px;
  border-bottom: 2px solid var(--sre-border);
  min-height: 44px;
}

.sidebar-app-name {
  font-size: 13px;
  font-weight: 700;
  color: var(--sidebar-accent);
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  transition: color var(--sre-duration-base) var(--sre-ease-out);
}

/* Pin/collapse button — colorful gradient dot */
.sidebar-pin-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  flex-shrink: 0;
  transition:
    background var(--sre-duration-fast) var(--sre-ease-out),
    color var(--sre-duration-fast) var(--sre-ease-out),
    transform var(--sre-duration-fast) var(--sre-ease-spring);
}

.sidebar-pin-btn:hover {
  background: color-mix(in srgb, var(--sidebar-accent) 12%, transparent);
  color: var(--sidebar-accent);
  transform: scale(1.1);
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
