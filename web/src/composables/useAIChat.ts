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

  // FE6-7: Chat history pagination state
  const historyPageSize = 50
  const hasMoreHistory = ref(false)
  const loadingMore = ref(false)

  // FE6-4: BLOCKED — SSE streaming requires backend SSE endpoint (POST /api/v1/ai/chat/stream).
  // Cannot implement frontend streaming without backend support.
  // When backend is ready, plan:
  //  1. Use fetch() with ReadableStream reader (not EventSource, since POST)
  //  2. Push partial content to the last assistant message as chunks arrive
  //  3. Add AbortController support for cancel mid-stream
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

  // FE6-7: Chat history pagination — loads most recent messages first,
  // with loadMore() to prepend older pages.
  async function loadHistory() {
    try {
      const resp = await aiChatApi.getHistory(mode.value)
      const all = resp.data.data || []
      // Show only the last page of messages initially
      if (all.length > historyPageSize) {
        messages.value = all.slice(-historyPageSize)
        hasMoreHistory.value = true
      } else {
        messages.value = all
        hasMoreHistory.value = false
      }
    } catch (err) {
      console.error('Failed to load AI chat history:', err)
      error.value = getErrorMessage(err as Error) || t('ai.sendFailed')
    }
  }

  // FE6-7: Load older messages by re-fetching full history and prepending older ones
  async function loadMore() {
    if (loadingMore.value || !hasMoreHistory.value) return
    loadingMore.value = true
    try {
      const resp = await aiChatApi.getHistory(mode.value)
      const all = resp.data.data || []
      // Determine how many older messages to prepend
      const currentCount = messages.value.length
      const olderEnd = all.length - currentCount
      if (olderEnd <= 0) {
        hasMoreHistory.value = false
        return
      }
      const olderStart = Math.max(0, olderEnd - historyPageSize)
      const olderMessages = all.slice(olderStart, olderEnd)
      messages.value = [...olderMessages, ...messages.value]
      hasMoreHistory.value = olderStart > 0
    } catch (err) {
      console.error('Failed to load more chat history:', err)
    } finally {
      loadingMore.value = false
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

  return {
    messages, loading, mode, error, lastFailedInput,
    hasMoreHistory, loadingMore,
    sendMessage, retryLast, loadHistory, loadMore, clearHistory, switchMode,
  }
}
