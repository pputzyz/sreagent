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
  color: var(--sre-text-inverse);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--sre-shadow-md);
  transition:
    transform var(--sre-duration-fast) var(--sre-ease-out),
    box-shadow var(--sre-duration-fast) var(--sre-ease-out);
}

.ai-chat-btn:hover {
  transform: scale(1.08);
  box-shadow: var(--sre-shadow-lg);
}

.ai-chat-btn--active {
  background: var(--sre-primary-hover, var(--sre-primary));
  box-shadow: var(--sre-shadow-lg);
}
</style>
