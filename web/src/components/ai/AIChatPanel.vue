<script setup lang="ts">
import { ref, watch, nextTick, onMounted, computed } from 'vue'
import { NDrawer, NDrawerContent, NSelect, NButton, NIcon, NInput, NPopconfirm } from 'naive-ui'
import { TrashOutline, SendOutline, RefreshOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useAIChat } from '@/composables/useAIChat'
import type { ChatMode } from '@/composables/useAIChat'
import AIChatMessage from './AIChatMessage.vue'

const props = defineProps<{
  show: boolean
  alertContext?: string
  initialMode?: ChatMode
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
}>()

const { t } = useI18n()
const { messages, loading, mode, error, lastFailedInput, sendMessage, retryLast, loadHistory, clearHistory, switchMode } = useAIChat()

const inputText = ref('')
const messageListRef = ref<HTMLElement | null>(null)

const modeOptions = [
  { label: t('ai.alertMode'), value: 'alert' },
  { label: t('ai.generalMode'), value: 'general' },
  { label: t('ai.petMode'), value: 'pet' },
]

const suggestedPrompts = computed(() => {
  const prompts: Record<ChatMode, string[]> = {
    alert: [
      t('ai.suggestAlert1'),
      t('ai.suggestAlert2'),
      t('ai.suggestAlert3'),
    ],
    general: [
      t('ai.suggestGeneral1'),
      t('ai.suggestGeneral2'),
      t('ai.suggestGeneral3'),
    ],
    pet: [
      t('ai.suggestPet1'),
      t('ai.suggestPet2'),
      t('ai.suggestPet3'),
    ],
  }
  return prompts[mode.value] || prompts.general
})

watch(() => props.show, (val) => {
  if (val) {
    if (props.initialMode && props.initialMode !== mode.value) {
      switchMode(props.initialMode)
    }
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

function handleModeChange(val: ChatMode) {
  switchMode(val)
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
            <n-select
              :value="mode"
              :options="modeOptions"
              size="small"
              style="width: 120px"
              @update:value="handleModeChange"
            />
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
          <div class="chat-empty-text">{{ t('ai.emptyHint') }}</div>
          <div class="chat-suggestions">
            <button
              v-for="(prompt, i) in suggestedPrompts"
              :key="i"
              class="chat-suggestion"
              @click="handleSuggestion(prompt)"
            >
              {{ prompt }}
            </button>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="chat-footer">
          <n-input
            v-model:value="inputText"
            type="textarea"
            :placeholder="t('ai.inputPlaceholder')"
            :autosize="{ minRows: 1, maxRows: 4 }"
            @keydown="handleKeydown"
          />
          <n-button
            type="primary"
            :loading="loading"
            :disabled="!inputText.trim()"
            @click="handleSend"
          >
            <template #icon>
              <n-icon :component="SendOutline" />
            </template>
            {{ t('ai.send') }}
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
  gap: 16px;
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
}

.chat-footer {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.chat-footer :deep(.n-input) {
  flex: 1;
}
</style>
