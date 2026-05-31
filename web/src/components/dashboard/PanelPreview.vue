<script setup lang="ts">
import { computed, ref } from 'vue'
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

// FE4-10: Fullscreen mode using the Fullscreen API
const containerRef = ref<HTMLElement | null>(null)
const isFullscreen = ref(false)

function toggleFullscreen() {
  if (!containerRef.value) return
  if (!document.fullscreenElement) {
    containerRef.value.requestFullscreen().catch(() => {})
  } else {
    document.exitFullscreen().catch(() => {})
  }
}

function onFullscreenChange() {
  isFullscreen.value = !!document.fullscreenElement
}

import { onMounted, onUnmounted } from 'vue'
onMounted(() => document.addEventListener('fullscreenchange', onFullscreenChange))
onUnmounted(() => document.removeEventListener('fullscreenchange', onFullscreenChange))
</script>

<template>
  <div ref="containerRef" class="panel-preview" :class="{ 'panel-fullscreen': isFullscreen }">
    <div class="preview-header">
      <span class="preview-label">Preview</span>
      <button class="fullscreen-btn" :title="isFullscreen ? 'Exit fullscreen' : 'Fullscreen'" @click="toggleFullscreen">
        <svg v-if="!isFullscreen" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3H5a2 2 0 00-2 2v3m18 0V5a2 2 0 00-2-2h-3m0 18h3a2 2 0 002-2v-3M3 16v3a2 2 0 002 2h3"/></svg>
        <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 14h3a2 2 0 012 2v3m4-5h3a2 2 0 002-2V9M15 3v3a2 2 0 002 2h3M4 10V7a2 2 0 012-2h3"/></svg>
      </button>
    </div>
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
.panel-preview.panel-fullscreen {
  background: var(--sre-bg-page);
  padding: 16px;
  display: flex;
  flex-direction: column;
}
.panel-preview.panel-fullscreen .preview-container {
  flex: 1;
  height: auto;
}
.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.preview-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
}
.fullscreen-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.fullscreen-btn:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-primary);
}
.preview-container {
  height: 250px;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--sre-border);
}
</style>
