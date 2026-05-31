import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { aiChatApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import type { ChatMessage } from '@/types'

export type ChatMode = 'alert' | 'general'

/** Reset module-level singleton state (call on logout) */
export function resetAIChat() {
  // Note: useAIChat uses instance-level state (inside the composable function),
  // so there are no module-level singletons to reset here.
  // This is a no-op export for consistent API surface.
}

export function useAIChat() {
  const { t } = useI18n()
  const messages = ref<ChatMessage[]>([])
  const loading = ref(false)
  const mode = ref<ChatMode>('general')
  const error = ref<string | null>(null)
  const lastFailedInput = ref<string | null>(null)

  // FE6-4: SSE streaming implementation plan
  // Current: waits for full response before displaying.
  // Plan:
  //  1. Backend: POST /api/v1/ai/chat/stream → SSE with "data: {chunk}" events
  //  2. Frontend: use fetch() with ReadableStream reader (not EventSource, since POST)
  //  3. Push partial content to the last assistant message as chunks arrive:
  //     - Create empty assistant message on stream start
  //     - Append each chunk to assistant message content
  //     - On stream end, mark message as complete
  //  4. Add AbortController support for cancel mid-stream
  //  5. Fallback: if stream fails, retry with regular POST
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
    } catch (e: unknown) {
      error.value = getErrorMessage(e) || t('ai.sendFailed')
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
    } catch (err) {
      console.error('Failed to load AI chat history:', err)
      error.value = getErrorMessage(err as Error) || t('ai.sendFailed')
    }
  }

  async function clearHistory() {
    try {
      await aiChatApi.clearHistory(mode.value)
      messages.value = []
    } catch (err) {
      console.error('Failed to clear AI chat history:', err)
      error.value = getErrorMessage(err as Error) || t('ai.sendFailed')
    }
  }

  async function switchMode(newMode: ChatMode) {
    if (mode.value === newMode) return
    mode.value = newMode
    messages.value = []
    lastFailedInput.value = null
    error.value = null
    await loadHistory()
  }

  return { messages, loading, mode, error, lastFailedInput, sendMessage, retryLast, loadHistory, clearHistory, switchMode }
}
