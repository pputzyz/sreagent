<script setup lang="ts">
import { ref, watch, nextTick, onMounted, computed } from 'vue'
import { NDrawer, NDrawerContent, NButton, NIcon, NInput, NPopconfirm } from 'naive-ui'
import { Trash2, Send, RefreshCw, Maximize2, Minimize2 } from 'lucide-vue-next'
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
const isFullscreen = ref(false)

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
}

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
    :width="isFullscreen ? '100vw' : 420"
    :min-width="isFullscreen ? undefined : 320"
    :max-width="isFullscreen ? undefined : 800"
    placement="right"
    :resizable="!isFullscreen"
    :class="{ 'chat-drawer-fullscreen': isFullscreen }"
    @update:show="handleClose"
  >
    <n-drawer-content :native-scrollbar="false">
      <template #header>
        <div class="chat-header">
          <span class="chat-title">{{ t('ai.chatTitle') }}</span>
          <div class="chat-header-actions">
            <n-button
              quaternary
              size="small"
              circle
              :title="isFullscreen ? t('ai.exitFullscreen') : t('ai.fullscreen')"
              @click="toggleFullscreen"
            >
              <template #icon>
                <n-icon :component="isFullscreen ? Minimize2 : Maximize2" />
              </template>
            </n-button>
            <n-popconfirm @positive-click="clearHistory">
              <template #trigger>
                <n-button
                  quaternary
                  size="small"
                  circle
                  :title="t('ai.clear')"
                >
                  <template #icon>
                    <n-icon :component="Trash2" />
                  </template>
                </n-button>
              </template>
              {{ t('ai.clearConfirm') }}
            </n-popconfirm>
          </div>
        </div>
      </template>

      <div class="chat-body">
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
                <n-icon :component="RefreshCw" />
              </template>
              {{ t('ai.retry') }}
            </n-button>
          </div>

          <div v-if="messages.length === 0 && !loading" class="chat-empty">
            <div class="chat-empty-icon">
              <svg width="48" height="48" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
                <rect x="4" y="4" width="40" height="40" rx="12" fill="var(--sre-primary-soft)" stroke="var(--sre-primary)" stroke-width="1.5"/>
                <path d="M24 14l2 6h6l-5 4 2 6-5-4-5 4 2-6-5-4h6z" fill="var(--sre-primary)" opacity="0.8"/>
                <circle cx="16" cy="16" r="2" fill="var(--sre-primary)" opacity="0.4"/>
                <circle cx="32" cy="14" r="1.5" fill="var(--sre-primary)" opacity="0.3"/>
                <circle cx="34" cy="32" r="1.5" fill="var(--sre-primary)" opacity="0.3"/>
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

        <div class="chat-footer">
          <div class="chat-input-wrap">
            <n-input
              v-model:value="inputText"
              type="textarea"
              :placeholder="t('ai.inputPlaceholder')"
              :autosize="{ minRows: 2, maxRows: 6 }"
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
              <n-icon :component="Send" />
            </template>
          </n-button>
        </div>
      </div>
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

.chat-body {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.chat-messages {
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
  min-height: 0;
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
  padding: 12px 0 4px;
  flex-shrink: 0;
  border-top: 1px solid var(--sre-border);
  background: var(--sre-bg-card);
}

.chat-input-wrap {
  flex: 1;
  min-width: 0;
  background: var(--sre-bg-elevated);
  border: 1.5px solid var(--sre-border);
  border-radius: var(--sre-radius-md);
  transition: border-color var(--sre-duration-fast) var(--sre-ease-out);
}

.chat-input-wrap:focus-within {
  border-color: var(--sre-primary);
  box-shadow: 0 0 0 3px var(--sre-primary-soft);
}

.chat-input :deep(.n-input-wrapper) {
  padding: 10px 14px;
  background: transparent !important;
}

.chat-input :deep(.n-input__textarea) {
  min-height: 52px;
  line-height: 1.6;
}

.chat-input :deep(.n-input--textarea .n-input__border),
.chat-input :deep(.n-input--textarea .n-input__state-border) {
  display: none;
}

.chat-send-btn {
  flex-shrink: 0;
  height: 44px;
  width: 44px;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.chat-drawer-fullscreen :deep(.n-drawer-content) {
  width: 100vw !important;
  max-width: 100vw !important;
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
