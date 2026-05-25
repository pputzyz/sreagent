<script setup lang="ts">
import { computed } from 'vue'
import type { PanelConfig } from '@/types/dashboard'
import PanelCard from '@/components/query/PanelCard.vue'

const props = defineProps<{
  panel: PanelConfig
  timeRange: { start: number; end: number }
}>()

// Use a computed to pass a reactive shallow copy for live preview
const previewPanel = computed(() => ({
  ...props.panel,
  targets: props.panel.targets || [],
}))
</script>

<template>
  <div class="panel-preview">
    <div class="preview-label">Preview</div>
    <div class="preview-container">
      <PanelCard :panel="previewPanel" :time-range="timeRange" />
    </div>
  </div>
</template>

<style scoped>
.panel-preview {
  border-top: 1px solid var(--sre-border);
  padding-top: 12px;
  margin-top: 12px;
}
.preview-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  margin-bottom: 8px;
}
.preview-container {
  height: 250px;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--sre-border);
}
</style>
