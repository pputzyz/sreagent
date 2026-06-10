import { ref, onMounted, onUnmounted } from 'vue'
import axios from 'axios'

/**
 * Session health guard — detects stale sessions and server unreachability.
 *
 * - visibilitychange: re-validates session when user returns to tab
 * - periodic heartbeat: every 5 min, pings /healthz
 * - exposes reactive state for UI banners
 */
export function useSessionGuard() {
  const isOnline = ref(true)
  const sessionExpired = ref(false)

  let heartbeatTimer: ReturnType<typeof setInterval> | null = null
  let visibilityHandler: (() => void) | null = null
  let consecutiveFailures = 0
  const MAX_FAILURES = 3

  async function checkHealth(): Promise<boolean> {
    try {
      await axios.get('/healthz', { timeout: 5000 })
      consecutiveFailures = 0
      isOnline.value = true
      return true
    } catch {
      consecutiveFailures++
      if (consecutiveFailures >= MAX_FAILURES) {
        isOnline.value = false
      }
      return false
    }
  }

  async function validateSession() {
    const token = localStorage.getItem('token')
    if (!token) {
      sessionExpired.value = true
      return
    }

    // First check if server is reachable
    const healthy = await checkHealth()
    if (!healthy) return

    try {
      const res = await axios.post('/api/v1/auth/refresh', { token }, { timeout: 10000 })
      const newToken = res.data?.data?.token
      if (newToken) {
        localStorage.setItem('token', newToken)
        sessionExpired.value = false
        isOnline.value = true
        consecutiveFailures = 0
      } else {
        sessionExpired.value = true
      }
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 401 || status === 403) {
        sessionExpired.value = true
      }
      // Network errors — server unreachable, handled by checkHealth
    }
  }

  function handleVisibilityChange() {
    if (document.visibilityState === 'visible') {
      validateSession()
    }
  }

  onMounted(() => {
    // Initial health check
    checkHealth()

    // Listen for tab visibility changes
    visibilityHandler = handleVisibilityChange
    document.addEventListener('visibilitychange', visibilityHandler)

    // Periodic heartbeat (every 5 minutes)
    heartbeatTimer = setInterval(checkHealth, 5 * 60 * 1000)
  })

  onUnmounted(() => {
    if (heartbeatTimer) clearInterval(heartbeatTimer)
    if (visibilityHandler) {
      document.removeEventListener('visibilitychange', visibilityHandler)
    }
  })

  function forceReconnect() {
    sessionExpired.value = false
    consecutiveFailures = 0
    validateSession()
  }

  return {
    isOnline,
    sessionExpired,
    validateSession,
    forceReconnect,
  }
}
