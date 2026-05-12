<script setup lang="ts">
import { ref, watch, nextTick, onMounted, computed } from 'vue'
import { NDrawer, NDrawerContent, NButton, NIcon, NInput, NPopconfirm } from 'naive-ui'
import { TrashOutline, SendOutline, RefreshOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useAIChat } from '@/composables/useAIChat'
import AIChatMessage from './AIChatMessage.vue'

const props = defineProps<{
  show: boolean
  alertContext?: string
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
}>()

const { t } = useI18n()
const { messages, loading, error, lastFailedInput, sendMessage, retryLast, loadHistory, clearHistory } = useAIChat()

const inputText = ref('')
const messageListRef = ref<HTMLElement | null>(null)

const suggestedPrompts = computed(() => [
  t('ai.suggestGeneral1'),
  t('ai.suggestGeneral2'),
  t('ai.suggestGeneral3'),
])

watch(() => props.show, (val) => {
  if (val) {
    loadHistory()
  }
})

watch(messages, () => {
  nextTick(() => scrollToBottom())
}, { deep: true })

function scrollToBottom() {
  if (messageListRef.value) {
    messageListRef.value.scrollTop = messageListRef.value.scrollHeight
  }
}

async function handleSend() {
  if (loading.value) return
  const text = inputText.value.trim()
  if (!text) return
  inputText.value = ''
  await sendMessage(text, props.alertContext)
}

function handleRetry() {
  retryLast()
}

function handleSuggestion(prompt: string) {
  inputText.value = prompt
  handleSend()
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

function handleClose() {
  emit('update:show', false)
}

onMounted(() => {
  if (props.show) loadHistory()
})
</script>

<template>
  <n-drawer
    :show="show"
    :width="400"
    placement="right"
    @update:show="handleClose"
  >
    <n-drawer-content :title="t('ai.chatTitle')" closable>
      <template #header>
        <div class="chat-header">
          <span class="chat-title">{{ t('ai.chatTitle') }}</span>
          <div class="chat-header-actions">
            <n-popconfirm @positive-click="clearHistory">
              <template #trigger>
                <n-button
                  quaternary
                  size="small"
                  circle
                  :title="t('ai.clear')"
                >
                  <template #icon>
                    <n-icon :component="TrashOutline" />
                  </template>
                </n-button>
              </template>
              {{ t('ai.clearConfirm') }}
            </n-popconfirm>
          </div>
        </div>
      </template>

      <div ref="messageListRef" class="chat-messages">
        <AIChatMessage
          v-for="(msg, idx) in messages"
          :key="msg.id || idx"
          :message="msg"
        />

        <div v-if="loading" class="chat-loading">
          <div class="chat-loading-dots">
            <span /><span /><span />
          </div>
          <span class="chat-loading-label">{{ t('ai.thinking') }}</span>
        </div>

        <div v-if="error" class="chat-error">
          <span>{{ error }}</span>
          <n-button size="tiny" quaternary @click="handleRetry">
            <template #icon>
              <n-icon :component="RefreshOutline" />
            </template>
            {{ t('ai.retry') }}
          </n-button>
        </div>

        <div v-if="messages.length === 0 && !loading" class="chat-empty">
          <div class="chat-empty-icon">
            <svg width="48" height="48" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="24" cy="24" r="20" fill="var(--sre-primary-soft)"/>
              <circle cx="24" cy="24" r="16" fill="var(--sre-bg-elevated)" stroke="var(--sre-primary)" stroke-width="1.5"/>
              <circle cx="18" cy="21" r="2.5" fill="var(--sre-primary)"/>
              <circle cx="30" cy="21" r="2.5" fill="var(--sre-primary)"/>
              <circle cx="19" cy="20" r="1" fill="white"/>
              <circle cx="31" cy="20" r="1" fill="white"/>
              <path d="M19 29 Q24 33 29 29" stroke="var(--sre-primary)" stroke-width="1.5" fill="none" stroke-linecap="round"/>
              <circle cx="24" cy="10" r="3" fill="var(--sre-primary)" opacity="0.6"/>
              <line x1="24" y1="7" x2="24" y2="4" stroke="var(--sre-primary)" stroke-width="1.5" stroke-linecap="round" opacity="0.4"/>
            </svg>
          </div>
          <div class="chat-empty-title">{{ t('ai.emptyTitle') }}</div>
          <div class="chat-empty-text">{{ t('ai.emptyHint') }}</div>
          <div class="chat-suggestions">
            <button
              v-for="(prompt, i) in suggestedPrompts"
              :key="i"
              class="chat-suggestion"
              @click="handleSuggestion(prompt)"
            >
              <span class="chat-suggestion-icon">💡</span>
              {{ prompt }}
            </button>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="chat-footer">
          <div class="chat-input-wrap">
            <n-input
              v-model:value="inputText"
              type="textarea"
              :placeholder="t('ai.inputPlaceholder')"
              :autosize="{ minRows: 2, maxRows: 5 }"
              @keydown="handleKeydown"
              class="chat-input"
            />
          </div>
          <n-button
            type="primary"
            :loading="loading"
            :disabled="!inputText.trim()"
            @click="handleSend"
            class="chat-send-btn"
          >
            <template #icon>
              <n-icon :component="SendOutline" />
            </template>
          </n-button>
        </div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<style scoped>
.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 12px;
}

.chat-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--sre-text-primary);
  white-space: nowrap;
}

.chat-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.chat-messages {
  display: flex;
  flex-direction: column;
  min-height: 100%;
  padding: 4px 0;
}

.chat-loading {
  display: flex;
  justify-content: flex-start;
  margin-bottom: 12px;
}

.chat-loading-dots {
  display: flex;
  gap: 4px;
  padding: 10px 14px;
  background: var(--sre-bg-elevated);
  border: 1px solid var(--sre-border);
  border-radius: 12px;
  border-bottom-left-radius: 4px;
}

.chat-loading-dots span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--sre-text-tertiary);
  animation: dot-pulse 1.4s ease-in-out infinite;
}

.chat-loading-dots span:nth-child(2) {
  animation-delay: 0.2s;
}

.chat-loading-dots span:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes dot-pulse {
  0%, 80%, 100% { opacity: 0.3; transform: scale(0.8); }
  40% { opacity: 1; transform: scale(1); }
}

.chat-loading-label {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  margin-left: 8px;
  opacity: 0;
  animation: chat-label-fade 600ms var(--sre-ease-out) 800ms forwards;
}

@keyframes chat-label-fade {
  from { opacity: 0; transform: translateX(-4px); }
  to { opacity: 1; transform: translateX(0); }
}

.chat-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  margin-bottom: 12px;
  font-size: 12px;
  color: var(--sre-critical);
  background: var(--sre-critical-soft);
  border-radius: var(--sre-radius-sm);
}

.chat-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 200px;
  gap: 12px;
}

.chat-empty-icon {
  font-size: 40px;
  line-height: 1;
  animation: chat-empty-wave 2.5s var(--sre-ease-out) infinite;
  transform-origin: bottom center;
}

@keyframes chat-empty-wave {
  0%, 100% { transform: rotate(0deg); }
  15% { transform: rotate(12deg); }
  30% { transform: rotate(-8deg); }
  45% { transform: rotate(6deg); }
  60% { transform: rotate(0deg); }
}

.chat-empty-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--sre-text-primary);
}

.chat-empty-text {
  color: var(--sre-text-tertiary);
  font-size: 13px;
}

.chat-suggestions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  max-width: 300px;
}

.chat-suggestion {
  padding: 8px 12px;
  font-size: 12px;
  color: var(--sre-text-secondary);
  background: var(--sre-bg-elevated);
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-sm);
  cursor: pointer;
  text-align: left;
  transition:
    background var(--sre-duration-fast) var(--sre-ease-out),
    border-color var(--sre-duration-fast) var(--sre-ease-out);
}

.chat-suggestion:hover {
  background: var(--sre-bg-hover);
  border-color: var(--sre-primary);
  color: var(--sre-text-primary);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.chat-suggestion:active {
  transform: translateY(0);
  box-shadow: none;
}

.chat-suggestion-icon {
  margin-right: 4px;
  font-size: 13px;
}

.chat-footer {
  display: flex;
  gap: 10px;
  align-items: flex-end;
  padding: 4px 0;
}

.chat-input-wrap {
  flex: 1;
  min-width: 0;
}

.chat-input :deep(.n-input-wrapper) {
  padding: 8px 12px;
}

.chat-input :deep(.n-input__textarea) {
  min-height: 40px;
  line-height: 1.5;
}

.chat-send-btn {
  flex-shrink: 0;
  height: 40px;
  width: 40px;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

@media (prefers-reduced-motion: reduce) {
  .chat-empty-icon,
  .chat-loading-label {
    animation: none;
  }
  .chat-suggestion:hover {
    transform: none;
    box-shadow: none;
  }
}
</style>
