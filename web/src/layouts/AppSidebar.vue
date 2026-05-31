<script setup lang="ts">
import { computed, h } from 'vue'
import { useRouter } from 'vue-router'
import { NMenu, NIcon } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { ChevronBackOutline, ChevronForwardOutline } from '@vicons/ionicons5'
import type { MenuSection, AppKey } from '@/composables/useAppNav'
import { iconColorMap } from '@/composables/useAppNav'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  sections: MenuSection[]
  activeKey: string
  collapsed: boolean
  pinned: boolean
  appName: string
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
        icon: item.icon
          ? () => h(NIcon, { color: item.iconColor || iconColorMap.get(item.icon!) || 'var(--sre-text-tertiary)' }, { default: () => h(item.icon!) })
          : undefined,
        children: item.children
          ?.filter(child => child.show !== false)
          .map(child => ({
            label: child.label,
            key: child.key,
            icon: child.icon
              ? () => h(NIcon, { color: child.iconColor || iconColorMap.get(child.icon!) || 'var(--sre-text-tertiary)' }, { default: () => h(child.icon!) })
              : undefined,
          })),
      }))

    if (visibleItems.length === 0) continue

    if (section.label) {
      result.push({
        type: 'submenu',
        label: section.label,
        key: `section-${section.label}`,
        children: visibleItems,
      })
    } else {
      result.push(...visibleItems)
    }
  }

  return result
})

// Expand all sub-menus by default
const defaultExpandedKeys = computed(() => {
  return menuOptions.value
    .filter(item => item.type === 'submenu' && item.children?.length)
    .map(item => item.key as string)
})

// ===== Menu click handler =====

function handleMenuUpdate(key: string) {
  emit('navigate', key)
  router.push(key)
}

</script>

<template>
  <aside class="app-sidebar" :class="{ collapsed, pinned: collapsed && pinned }" :data-app="activeApp">
    <!-- Header with app name + collapse toggle -->
    <div class="sidebar-header">
      <transition name="fade">
        <span v-if="!collapsed" class="sidebar-app-name">{{ appName }}</span>
      </transition>
      <button class="sidebar-pin-btn" :title="pinned ? t('header.expandSidebar') : (collapsed ? t('header.expandSidebar') : t('header.collapseSidebar'))" @click="emit('toggle-collapse')">
        <n-icon :component="collapsed ? ChevronForwardOutline : ChevronBackOutline" :size="14" />
      </button>
    </div>

    <!-- Nav area — hidden when collapsed to prevent stacking -->
    <nav class="sidebar-nav" :class="{ 'nav-hidden': collapsed && !pinned }" aria-label="Main navigation">
      <div class="sidebar-nav-inner">
        <n-menu
          :collapsed="false"
          :collapsed-width="64"
          :collapsed-icon-size="22"
          :options="menuOptions"
          :value="activeKey"
          :default-expanded-keys="defaultExpandedKeys"
          :indent="16"
          @update:value="handleMenuUpdate"
        />
      </div>
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
  border-right: 1px solid var(--sre-border);
  flex-shrink: 0;
  transition: width 280ms var(--sre-ease-out);
  overflow: hidden;
}

.app-sidebar[data-app="oncall"]   { --sidebar-accent: var(--sre-brand-oncall); --sidebar-accent-soft: rgba(244, 63, 94, 0.08); }
.app-sidebar[data-app="alert"]    { --sidebar-accent: var(--sre-brand-alert); --sidebar-accent-soft: rgba(59, 130, 246, 0.08); }
.app-sidebar[data-app="platform"] { --sidebar-accent: var(--sre-brand-platform); --sidebar-accent-soft: rgba(139, 92, 246, 0.08); }

.app-sidebar.collapsed {
  width: 64px;
}

.app-sidebar.collapsed.pinned {
  width: 220px;
}

/* Nav area — scrollable, hidden when collapsed */
.sidebar-nav {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px;
  opacity: 1;
  transition: opacity 180ms var(--sre-ease-out) 100ms;
}

.sidebar-nav.nav-hidden {
  opacity: 0;
  pointer-events: none;
  padding: 0;
  transition-delay: 0ms;
}

.sidebar-nav-inner {
  min-width: 200px;
}

/* Naive UI menu overrides — clean accent */
.sidebar-nav :deep(.n-menu-item) {
  border-radius: 8px;
  margin-bottom: 2px;
  transition: background var(--sre-duration-fast) var(--sre-ease-out);
}

.sidebar-nav :deep(.n-menu-item-content__icon) {
  transition: color 180ms var(--sre-ease-out), transform 180ms var(--sre-ease-out);
}

.sidebar-nav :deep(.n-menu-item:hover) {
  background: var(--sidebar-accent-soft, rgba(13, 148, 136, 0.06));
}

.sidebar-nav :deep(.n-menu-item .n-menu-item-content__label) {
  transition: transform 200ms var(--sre-ease-out), color 200ms var(--sre-ease-out);
}

.sidebar-nav :deep(.n-menu-item:hover .n-menu-item-content__label) {
  color: var(--sre-text-primary);
  transform: translateX(2px);
}

.sidebar-nav :deep(.n-menu-item:hover .n-menu-item-content__icon) {
  color: var(--sidebar-accent) !important;
  transform: scale(1.05);
}

.sidebar-nav :deep(.n-menu-item--selected) {
  background: var(--sidebar-accent-soft, rgba(13, 148, 136, 0.08)) !important;
}

.sidebar-nav :deep(.n-menu-item-content--selected .n-menu-item-content__icon) {
  color: var(--sidebar-accent) !important;
}

.sidebar-nav :deep(.n-menu-item-content--selected .n-menu-item-content__label) {
  color: var(--sidebar-accent) !important;
  font-weight: 600;
}

/* Selected left indicator — solid accent bar */
.sidebar-nav :deep(.n-menu-item--selected::before) {
  content: '';
  position: absolute;
  left: 4px;
  top: 8px;
  bottom: 8px;
  width: 3.5px;
  border-radius: 3px;
  background: var(--sidebar-accent);
  box-shadow: 0 0 6px color-mix(in srgb, var(--sidebar-accent) 30%, transparent);
  animation: sidebar-indicator-enter 300ms var(--sre-ease-spring);
}

/* Group label */
.sidebar-nav :deep(.n-menu-group-label) {
  font-family: var(--sre-font-display);
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-tertiary);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding: 12px 12px 4px;
}

/* Submenu (collapsible groups) */
.sidebar-nav :deep(.n-submenu) {
  margin-bottom: 2px;
}

.sidebar-nav :deep(.n-submenu-children) {
  padding-left: 0;
}

.sidebar-nav :deep(.n-submenu > .n-menu-item-content) {
  font-family: var(--sre-font-display);
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-tertiary);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding: 10px 12px 4px;
  cursor: pointer;
  user-select: none;
}

.sidebar-nav :deep(.n-submenu > .n-menu-item-content:hover) {
  color: var(--sre-text-secondary);
  background: transparent;
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
  font-family: var(--sre-font-display);
  font-size: 13px;
  font-weight: 700;
  color: var(--sidebar-accent);
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  transition: color var(--sre-duration-base) var(--sre-ease-out);
}

/* Pin/collapse button */
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
    transform 200ms var(--sre-ease-spring),
    background var(--sre-duration-fast) var(--sre-ease-out),
    color var(--sre-duration-fast) var(--sre-ease-out);
}

.sidebar-pin-btn:hover {
  transform: scale(1.08);
  background: var(--sre-bg-hover);
  color: var(--sidebar-accent);
}

.sidebar-pin-btn:active {
  transform: scale(0.92);
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

/* Indicator enter animation */
@keyframes sidebar-indicator-enter {
  0% { transform: scaleY(0); opacity: 0; }
  100% { transform: scaleY(1); opacity: 1; }
}

/* Ensure indicator scales from top */
.sidebar-nav :deep(.n-menu-item--selected::before) {
  transform-origin: top;
}
</style>
