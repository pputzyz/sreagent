<script setup lang="ts">
import { MessageCircle } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  active?: boolean
}>()

const emit = defineEmits<{
  click: []
}>()
</script>

<template>
  <button
    class="ai-chat-fab"
    :class="{ 'ai-chat-fab--active': active }"
    :aria-label="t('ai.askAI')"
    @click="emit('click')"
  >
    <MessageCircle :size="20" color="white" :stroke-width="2" />
    <span class="ai-chat-fab-label">{{ t('ai.askAI') }}</span>
  </button>
</template>

<style scoped>
.ai-chat-fab {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: var(--sre-z-float, 100);
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border-radius: 24px;
  border: none;
  background: linear-gradient(135deg, #F59E0B, #FBBF24);
  color: white;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  box-shadow: 0 4px 12px rgba(245, 158, 11, 0.3);
  transition:
    transform var(--sre-duration-fast) var(--sre-ease-out),
    box-shadow var(--sre-duration-fast) var(--sre-ease-out);
}

.ai-chat-fab:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(245, 158, 11, 0.4);
}

.ai-chat-fab:active {
  transform: scale(0.97);
}

.ai-chat-fab--active {
  box-shadow: 0 6px 20px rgba(245, 158, 11, 0.5);
}

.ai-chat-fab:not(.ai-chat-fab--active) {
  animation: ai-breathe 3s ease-in-out infinite;
}

@keyframes ai-breathe {
  0%, 100% { box-shadow: 0 4px 12px rgba(245, 158, 11, 0.3); }
  50% { box-shadow: 0 4px 20px rgba(245, 158, 11, 0.55); }
}

@media (prefers-reduced-motion: reduce) {
  .ai-chat-fab:hover {
    transform: none;
  }
  .ai-chat-fab:active {
    transform: none;
  }
}
</style>
