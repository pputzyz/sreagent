<script setup lang="ts">
import { NIcon, NTooltip } from 'naive-ui'
import { FlashOutline, AlertCircleOutline, SettingsOutline } from '@vicons/ionicons5'
import type { AppKey } from '@/composables/useAppNav'
import MascotFox from '@/components/common/MascotFox.vue'

defineProps<{
  activeApp: AppKey
}>()

const emit = defineEmits<{
  switch: [app: AppKey]
}>()

interface RailItem {
  key: AppKey
  icon: typeof FlashOutline
  label: string
}

const topItems: RailItem[] = [
  { key: 'oncall', icon: FlashOutline, label: 'On-Call' },
  { key: 'alert', icon: AlertCircleOutline, label: 'Alert' },
]

const bottomItems: RailItem[] = [
  { key: 'platform', icon: SettingsOutline, label: 'Platform' },
]
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
            <n-icon :component="item.icon" :size="20" />
          </button>
        </template>
        {{ item.label }}
      </n-tooltip>
    </div>

    <div class="rail-spacer" />

    <div class="rail-mascot">
      <MascotFox />
    </div>

    <div class="rail-bottom">
      <n-tooltip
        v-for="item in bottomItems"
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
            <n-icon :component="item.icon" :size="20" />
          </button>
        </template>
        {{ item.label }}
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

.rail-mascot {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px 0;
}

.rail-icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  transition:
    background var(--sre-duration-base) var(--sre-ease-out),
    color var(--sre-duration-base) var(--sre-ease-out);
}

.rail-icon-btn:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
}

.rail-icon-btn.active {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
}
</style>
