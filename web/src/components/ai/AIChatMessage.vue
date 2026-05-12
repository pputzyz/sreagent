<script setup lang="ts">
import type { ChatMessage } from '@/types'

defineProps<{
  message: ChatMessage
}>()
</script>

<template>
  <div class="chat-msg" :class="message.role === 'user' ? 'chat-msg--user' : 'chat-msg--assistant'">
    <div class="chat-bubble">
      <div class="chat-content">{{ message.content }}</div>
    </div>
  </div>
</template>

<style scoped>
.chat-msg {
  display: flex;
  margin-bottom: 12px;
}

.chat-msg--user {
  justify-content: flex-end;
}

.chat-msg--assistant {
  justify-content: flex-start;
}

.chat-bubble {
  max-width: 80%;
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.6;
  word-break: break-word;
  white-space: pre-wrap;
}

.chat-msg--user .chat-bubble {
  background: var(--sre-primary);
  color: var(--sre-text-inverse, #fff);
  border-bottom-right-radius: 4px;
}

.chat-msg--assistant .chat-bubble {
  background: var(--sre-bg-elevated);
  color: var(--sre-text-primary);
  border: 1px solid var(--sre-border);
  border-bottom-left-radius: 4px;
}

.chat-content {
  margin: 0;
}
</style>
