import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { aiChatApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import type { ChatMessage } from '@/types'

export type ChatMode = 'alert' | 'general'

/** Module-level abort controller shared across useAIChat instances */
const moduleAbortController = { current: null as AbortController | null }

/** Reset module-level singleton state (call on logout) */
export function resetAIChat() {
  if (moduleAbortController.current) {
    moduleAbortController.current.abort()
    moduleAbortController.current = null
  }
}

export function useAIChat() {
  const { t } = useI18n()
  const messages = ref<ChatMessage[]>([])
  const loading = ref(false)
  const mode = ref<ChatMode>('general')
  const error = ref<string | null>(null)
  const lastFailedInput = ref<string | null>(null)
  const streamingContent = ref<string>('')
  const streaming = ref(false)
  let abortController: AbortController | null = null
  // Sync local reference with module-level ref for resetAIChat()
  moduleAbortController.current = abortController

  // FE6-7: Chat history pagination state
  const historyPageSize = 50
  const hasMoreHistory = ref(false)
  const loadingMore = ref(false)

  /** Cancel an in-progress stream */
  function cancelStream() {
    if (abortController) {
      abortController.abort()
      abortController = null
    }
  }

  /**
   * Send a message using SSE streaming.
   * Uses fetch() + ReadableStream (not EventSource, since we need POST with body).
   * Falls back to synchronous chat if streaming fails.
   */
  async function sendMessageStream(text: string, context?: string): Promise<boolean> {
    const token = localStorage.getItem('token')
    if (!token) return false

    abortController = new AbortController()
    moduleAbortController.current = abortController
    streaming.value = true
    streamingContent.value = ''

    // Push an empty assistant message that we'll fill incrementally
    const assistantMsg: ChatMessage = {
      mode: mode.value,
      role: 'assistant',
      content: '',
      created_at: new Date().toISOString(),
    }
    messages.value.push(assistantMsg)

    try {
      const resp = await fetch('/api/v1/ai/chat/stream', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ mode: mode.value, message: text, context }),
        signal: abortController.signal,
      })

      if (!resp.ok) {
        // Remove the empty assistant message on non-OK response
        messages.value.pop()
        return false
      }

      const reader = resp.body?.getReader()
      if (!reader) {
        messages.value.pop()
        return false
      }

      const decoder = new TextDecoder()
      let buffer = ''
      let fullContent = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        // Keep the last incomplete line in the buffer
        buffer = lines.pop() || ''

        for (const line of lines) {
          const trimmed = line.trim()
          if (!trimmed || !trimmed.startsWith('data: ')) continue
          const data = trimmed.slice(6)

          if (data === '[DONE]') {
            break
          }

          try {
            const parsed = JSON.parse(data)
            if (parsed.error) {
              error.value = parsed.error
              break
            }
            if (parsed.content) {
              fullContent += parsed.content
              streamingContent.value = fullContent
              // Update the assistant message in-place
              const lastMsg = messages.value[messages.value.length - 1]
              if (lastMsg && lastMsg.role === 'assistant') {
                lastMsg.content = fullContent
              }
            }
          } catch {
            // Skip malformed JSON chunks
          }
        }
      }

      // If we got no content at all, treat as failure
      if (!fullContent) {
        messages.value.pop()
        return false
      }

      return true
    } catch (e: unknown) {
      // AbortError means user cancelled — keep whatever content we have
      if (e instanceof DOMException && e.name === 'AbortError') {
        if (!streamingContent.value) {
          messages.value.pop()
        }
        return !!streamingContent.value
      }
      // Remove empty assistant message on other errors
      if (!streamingContent.value) {
        messages.value.pop()
      }
      return false
    } finally {
      streaming.value = false
      streamingContent.value = ''
      abortController = null
    }
  }

  /**
   * Send a message with streaming first, falling back to synchronous chat.
   */
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
      // Try streaming first
      const streamOk = await sendMessageStream(text, context)
      if (streamOk) return

      // Fallback to synchronous chat
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
    cancelStream()
    mode.value = newMode
    messages.value = []
    lastFailedInput.value = null
    error.value = null
    await loadHistory()
  }

  return {
    messages, loading, mode, error, lastFailedInput,
    hasMoreHistory, loadingMore, streamingContent, streaming,
    sendMessage, sendMessageStream, cancelStream,
    retryLast, loadHistory, loadMore, clearHistory, switchMode,
  }
}
