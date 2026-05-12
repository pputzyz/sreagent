<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'
import type { ChatMessage } from '@/types'

const props = defineProps<{
  message: ChatMessage
}>()

const renderedContent = computed(() => {
  if (props.message.role === 'assistant') {
    return marked.parse(props.message.content, { breaks: true }) as string
  }
  return ''
})
</script>

<template>
  <div
    class="chat-msg"
    :class="[
      message.role === 'user' ? 'chat-msg--user' : 'chat-msg--assistant',
      { 'chat-msg--failed': message._failed },
    ]"
  >
    <div class="chat-bubble">
      <div v-if="message.role === 'assistant'" class="chat-content chat-markdown" v-html="renderedContent" />
      <div v-else class="chat-content">{{ message.content }}</div>
      <div v-if="message._failed" class="chat-failed-hint">&#x26A0;</div>
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
}

.chat-msg--user .chat-bubble {
  background: var(--sre-primary);
  color: var(--sre-text-inverse);
  border-bottom-right-radius: 4px;
  white-space: pre-wrap;
}

.chat-msg--assistant .chat-bubble {
  background: var(--sre-bg-elevated);
  color: var(--sre-text-primary);
  border: 1px solid var(--sre-border);
  border-bottom-left-radius: 4px;
}

.chat-msg--failed .chat-bubble {
  opacity: 0.7;
  border: 1px solid var(--sre-critical);
}

.chat-content {
  margin: 0;
}

.chat-failed-hint {
  font-size: 11px;
  color: var(--sre-critical);
  margin-top: 4px;
}

/* Markdown styles for assistant messages */
.chat-markdown :deep(p) {
  margin: 0 0 8px;
}

.chat-markdown :deep(p:last-child) {
  margin-bottom: 0;
}

.chat-markdown :deep(code) {
  font-family: var(--sre-font-mono);
  font-size: 12px;
  padding: 1px 4px;
  background: var(--sre-bg-hover);
  border-radius: 3px;
}

.chat-markdown :deep(pre) {
  margin: 8px 0;
  padding: 10px;
  background: var(--sre-bg-page);
  border-radius: var(--sre-radius-sm);
  overflow-x: auto;
}

.chat-markdown :deep(pre code) {
  padding: 0;
  background: none;
}

.chat-markdown :deep(ul),
.chat-markdown :deep(ol) {
  margin: 4px 0;
  padding-left: 20px;
}

.chat-markdown :deep(li) {
  margin: 2px 0;
}

.chat-markdown :deep(blockquote) {
  margin: 8px 0;
  padding: 4px 12px;
  border-left: 3px solid var(--sre-primary);
  color: var(--sre-text-secondary);
}

.chat-markdown :deep(h1),
.chat-markdown :deep(h2),
.chat-markdown :deep(h3) {
  margin: 12px 0 4px;
  font-weight: 600;
}

.chat-markdown :deep(h1) { font-size: 16px; }
.chat-markdown :deep(h2) { font-size: 14px; }
.chat-markdown :deep(h3) { font-size: 13px; }

.chat-markdown :deep(table) {
  border-collapse: collapse;
  margin: 8px 0;
  font-size: 12px;
}

.chat-markdown :deep(th),
.chat-markdown :deep(td) {
  padding: 4px 8px;
  border: 1px solid var(--sre-border);
}

.chat-markdown :deep(th) {
  background: var(--sre-bg-hover);
  font-weight: 600;
}
</style>
