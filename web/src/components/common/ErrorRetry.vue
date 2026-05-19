<script setup lang="ts">
/**
 * Error state with retry button.
 * Use when a data fetch or action fails and the user should be able to retry.
 *
 * Usage:
 *   <ErrorRetry :error="errorMsg" :loading="loading" @retry="fetchData" />
 */
import { NIcon, NButton } from 'naive-ui'
import { AlertCircleOutline, RefreshOutline } from '@vicons/ionicons5'

defineProps<{
  error: string
  loading?: boolean
}>()

defineEmits<{
  retry: []
}>()
</script>

<template>
  <div class="error-retry">
    <div class="error-retry-icon">
      <NIcon :component="AlertCircleOutline" />
    </div>
    <p class="error-retry-msg">{{ error }}</p>
    <NButton
      size="small"
      type="primary"
      :loading="loading"
      @click="$emit('retry')"
    >
      <template #icon>
        <NIcon :component="RefreshOutline" />
      </template>
      {{ $t?.('common.retry') || 'Retry' }}
    </NButton>
  </div>
</template>

<style scoped>
.error-retry {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 48px 24px;
  gap: 12px;
}
.error-retry-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--sre-critical-soft, rgba(208, 48, 80, 0.12));
  border: 1px solid var(--sre-critical-soft, rgba(208, 48, 80, 0.2));
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: var(--sre-critical, #d03050);
}
.error-retry-msg {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin: 0;
  max-width: 400px;
  line-height: 1.5;
}
</style>
