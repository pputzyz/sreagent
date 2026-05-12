<script setup lang="ts">
import { NIcon } from 'naive-ui'
import { ChatbubbleEllipsesOutline } from '@vicons/ionicons5'

defineProps<{
  active?: boolean
}>()

defineEmits<{
  click: []
}>()
</script>

<template>
  <button
    class="ai-chat-btn"
    :class="{ 'ai-chat-btn--active': active }"
    aria-label="AI Chat"
    @click="$emit('click')"
  >
    <n-icon :component="ChatbubbleEllipsesOutline" :size="24" />
  </button>
</template>

<style scoped>
.ai-chat-btn {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: var(--sre-z-float, 100);
  width: 48px;
  height: 48px;
  border: none;
  border-radius: 50%;
  background: var(--sre-primary);
  color: var(--sre-text-inverse, #fff);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
  transition:
    transform var(--sre-duration-fast) var(--sre-ease-out),
    box-shadow var(--sre-duration-fast) var(--sre-ease-out);
}

.ai-chat-btn:hover {
  transform: scale(1.08);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
}

.ai-chat-btn--active {
  background: var(--sre-primary-hover, var(--sre-primary));
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
}
</style>
