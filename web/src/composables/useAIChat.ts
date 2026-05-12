import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { aiChatApi } from '@/api'
import type { ChatMessage } from '@/types'

export type ChatMode = 'alert' | 'general' | 'pet'

export function useAIChat() {
  const { t } = useI18n()
  const messages = ref<ChatMessage[]>([])
  const loading = ref(false)
  const mode = ref<ChatMode>('general')
  const error = ref<string | null>(null)
  const lastFailedInput = ref<string | null>(null)

  async function sendMessage(text: string, context?: string) {
    if (!text.trim() || loading.value) return
    const userMsg: ChatMessage = {
      mode: mode.value,
      role: 'user',
      content: text,
      context,
      created_at: new Date().toISOString(),
      _failed: false,
    }
    messages.value.push(userMsg)
    loading.value = true
    error.value = null
    lastFailedInput.value = null
    try {
      const resp = await aiChatApi.send({ mode: mode.value, message: text, context })
      messages.value.push({
        mode: mode.value,
        role: 'assistant',
        content: resp.data.data.reply,
        created_at: new Date().toISOString(),
      })
    } catch (e: any) {
      error.value = e?.message || t('ai.sendFailed')
      lastFailedInput.value = text
      const lastMsg = messages.value[messages.value.length - 1]
      if (lastMsg && lastMsg.role === 'user') {
        lastMsg._failed = true
      }
    } finally {
      loading.value = false
    }
  }

  async function retryLast() {
    if (!lastFailedInput.value) return
    const text = lastFailedInput.value
    lastFailedInput.value = null
    await sendMessage(text)
  }

  async function loadHistory() {
    try {
      const resp = await aiChatApi.getHistory(mode.value)
      messages.value = resp.data.data || []
    } catch { /* silent */ }
  }

  async function clearHistory() {
    try {
      await aiChatApi.clearHistory(mode.value)
      messages.value = []
    } catch { /* ignore */ }
  }

  function switchMode(newMode: ChatMode) {
    if (mode.value === newMode) return
    mode.value = newMode
    messages.value = []
    lastFailedInput.value = null
    error.value = null
    loadHistory()
  }

  return { messages, loading, mode, error, lastFailedInput, sendMessage, retryLast, loadHistory, clearHistory, switchMode }
}
