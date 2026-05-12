<script setup lang="ts">
import { ref, watch, nextTick, onMounted } from 'vue'
import { NDrawer, NDrawerContent, NSelect, NButton, NIcon, NInput } from 'naive-ui'
import { TrashOutline, SendOutline } from '@vicons/ionicons5'
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
const { messages, loading, mode, error, sendMessage, loadHistory, clearHistory, switchMode } = useAIChat()

const inputText = ref('')
const messageListRef = ref<HTMLElement | null>(null)

const modeOptions = [
  { label: t('ai.alertMode'), value: 'alert' },
  { label: t('ai.generalMode'), value: 'general' },
  { label: t('ai.petMode'), value: 'pet' },
]

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
            <n-button
              quaternary
              size="small"
              circle
              :title="t('ai.clear')"
              @click="clearHistory"
            >
              <template #icon>
                <n-icon :component="TrashOutline" />
              </template>
            </n-button>
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

        <div v-if="error" class="chat-error">{{ error }}</div>

        <div v-if="messages.length === 0 && !loading" class="chat-empty">
          {{ t('ai.inputPlaceholder') }}
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
  padding: 8px 12px;
  margin-bottom: 12px;
  font-size: 12px;
  color: var(--sre-critical);
  background: var(--sre-critical-soft);
  border-radius: var(--sre-radius-sm);
}

.chat-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 200px;
  color: var(--sre-text-tertiary);
  font-size: 13px;
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
