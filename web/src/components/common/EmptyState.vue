<script setup lang="ts">
/**
 * Shared empty state component — FlashCat operational refinement.
 * Use whenever a list/table has zero items.
 *
 * Examples:
 *   <EmptyState
 *     :icon="ShieldCheckmarkOutline"
 *     title="All quiet"
 *     description="No active alerts firing"
 *   />
 *   <EmptyState
 *     :icon="LayersOutline"
 *     title="No channels yet"
 *     description="Create a channel to start aggregating incidents"
 *     :primary-text="t('common.create')"
 *     @primary="onCreate"
 *     :secondary-text="'View docs'"
 *     @secondary="openDocs"
 *   />
 */
import { type Component } from 'vue'
import { NIcon, NButton } from 'naive-ui'

defineProps<{
  icon?: Component
  title: string
  description?: string
  primaryText?: string
  secondaryText?: string
  size?: 'sm' | 'md' | 'lg'  // default md
  variant?: 'default' | 'success' | 'warning' | 'critical' | 'info'  // tone for icon
}>()

defineEmits<{
  primary: []
  secondary: []
}>()
</script>

<template>
  <div class="empty-state" :data-size="size || 'md'" :data-variant="variant || 'default'">
    <div v-if="icon" class="empty-icon">
      <NIcon :component="icon" />
    </div>
    <h3 class="empty-title">{{ title }}</h3>
    <p v-if="description" class="empty-desc">{{ description }}</p>
    <div v-if="primaryText || secondaryText" class="empty-actions">
      <NButton v-if="primaryText" type="primary" size="small" @click="$emit('primary')">
        {{ primaryText }}
      </NButton>
      <NButton v-if="secondaryText" quaternary size="small" @click="$emit('secondary')">
        {{ secondaryText }}
      </NButton>
    </div>
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 64px 24px;
  color: var(--sre-text-secondary);
}
.empty-state[data-size="sm"] { padding: 32px 16px; }
.empty-state[data-size="lg"] { padding: 96px 32px; }

.empty-icon {
  width: 56px; height: 56px;
  border-radius: 50%;
  background: var(--sre-bg-elevated);
  border: 1px solid var(--sre-border);
  display: flex; align-items: center; justify-content: center;
  font-size: 28px;
  color: var(--sre-text-tertiary);
  margin-bottom: 16px;
}
.empty-state[data-size="sm"] .empty-icon { width: 40px; height: 40px; font-size: 20px; margin-bottom: 12px; }
.empty-state[data-size="lg"] .empty-icon { width: 72px; height: 72px; font-size: 36px; margin-bottom: 20px; }

.empty-state[data-variant="success"]  .empty-icon { color: var(--sre-primary);  background: var(--sre-primary-soft);  border-color: rgba(24,160,88,0.18); }
.empty-state[data-variant="warning"]  .empty-icon { color: var(--sre-warning);  background: var(--sre-warning-soft); border-color: rgba(245,158,11,0.18); }
.empty-state[data-variant="critical"] .empty-icon { color: var(--sre-critical); background: var(--sre-critical-soft); border-color: rgba(239,68,68,0.18); }
.empty-state[data-variant="info"]     .empty-icon { color: var(--sre-info);     background: var(--sre-info-soft);    border-color: rgba(59,130,246,0.18); }

.empty-title {
  font-size: 15px; font-weight: 600;
  color: var(--sre-text-primary);
  margin: 0 0 6px;
  letter-spacing: -0.2px;
}
.empty-state[data-size="sm"] .empty-title { font-size: 13px; }
.empty-state[data-size="lg"] .empty-title { font-size: 18px; }

.empty-desc {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin: 0 0 20px;
  max-width: 360px;
  line-height: 1.5;
}
.empty-state[data-size="sm"] .empty-desc { font-size: 12px; margin-bottom: 12px; }
.empty-state[data-size="lg"] .empty-desc { font-size: 14px; max-width: 480px; }

.empty-actions {
  display: flex; align-items: center; gap: 8px;
}
</style>
