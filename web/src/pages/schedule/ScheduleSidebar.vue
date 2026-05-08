<script setup lang="ts">
import type { Schedule, User } from '@/types'
import { useI18n } from 'vue-i18n'

defineProps<{
  schedules: Schedule[]
  loading: boolean
  selectedId: number | null
  onCallMap: Record<number, User | null>
}>()

const emit = defineEmits<{
  select: [schedule: Schedule]
  create: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="schedule-sidebar">
    <div class="sidebar-header">
      <span class="sidebar-title">{{ t('schedule.title') }}</span>
      <n-button size="small" type="primary" @click="emit('create')">+ {{ t('schedule.newSchedule') }}</n-button>
    </div>

    <n-spin :show="loading" style="min-height: 100px">
      <div class="schedule-list">
        <div
          v-for="s in schedules"
          :key="s.id"
          class="schedule-item"
          :class="{ active: selectedId === s.id }"
          @click="emit('select', s)"
        >
          <div class="schedule-item-name">{{ s.name }}</div>
          <div class="schedule-item-meta">
            <span v-if="onCallMap[s.id]" class="oncall-user">
              <span class="oncall-dot" />
              {{ onCallMap[s.id]?.display_name || onCallMap[s.id]?.username }}
            </span>
            <span v-else class="no-oncall">{{ t('schedule.noOneOnCall') }}</span>
            <n-tag
              :type="s.is_enabled ? 'success' : 'default'"
              size="tiny"
              :bordered="false"
              style="margin-left: auto; flex-shrink: 0"
            >
              {{ s.is_enabled ? t('common.active') : t('common.disabled') }}
            </n-tag>
          </div>
        </div>

        <n-empty v-if="!loading && schedules.length === 0" :description="t('schedule.noSchedules')" style="padding: 40px 0" />
      </div>
    </n-spin>
  </div>
</template>

<style scoped>
.schedule-sidebar {
  width: 280px;
  flex-shrink: 0;
  border-right: 1px solid var(--sre-border);
  display: flex;
  flex-direction: column;
  background: var(--sre-bg-card);
  overflow: hidden;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid var(--sre-border);
  flex-shrink: 0;
}

.sidebar-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
}

.schedule-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.schedule-item {
  padding: 10px 16px;
  cursor: pointer;
  border-left: 3px solid transparent;
  transition: background var(--sre-duration-fast) var(--sre-ease-out);
}

.schedule-item:hover {
  background: var(--sre-bg-hover);
}

.schedule-item.active {
  background: var(--sre-success-soft);
  border-left-color: var(--sre-primary);
}

.schedule-item-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-text-primary);
  margin-bottom: 4px;
}

.schedule-item-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
}

.oncall-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--sre-success);
  margin-right: 4px;
}

.oncall-user {
  color: var(--sre-text-secondary);
  display: flex;
  align-items: center;
}

.no-oncall {
  color: var(--sre-text-secondary);
  opacity: 0.5;
}
</style>
