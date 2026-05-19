<script setup lang="ts">
import { NButton, NIcon, NPopconfirm } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { CloseOutline } from '@vicons/ionicons5'

defineProps<{
  selectedCount: number
  loading: boolean
}>()

const emit = defineEmits<{
  batchEnable: []
  batchDisable: []
  batchDelete: []
  clearSelection: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="batch-bar" role="toolbar" :aria-label="t('alert.batchActions')">
    <span class="batch-count tnum">{{ selectedCount }} {{ t('common.selected', { count: selectedCount }) }}</span>
    <n-button size="small" secondary :loading="loading" @click="emit('batchEnable')">
      {{ t('common.enabled') }}
    </n-button>
    <n-button size="small" secondary :loading="loading" @click="emit('batchDisable')">
      {{ t('common.disabled') }}
    </n-button>
    <n-popconfirm @positive-click="emit('batchDelete')">
      <template #trigger>
        <n-button size="small" tertiary type="error" :loading="loading">
          {{ t('common.delete') }}
        </n-button>
      </template>
      {{ t('alert.batchDeleteConfirm', { count: selectedCount }) }}
    </n-popconfirm>
    <div class="batch-spacer"></div>
    <n-button size="small" quaternary circle @click="emit('clearSelection')">
      <template #icon><n-icon :component="CloseOutline" /></template>
    </n-button>
  </div>
</template>

<style scoped>
.batch-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--sre-primary-soft);
  border-radius: 8px;
  margin-bottom: 16px;
}

.batch-count {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-primary);
}

.batch-spacer {
  flex: 1;
}
</style>
