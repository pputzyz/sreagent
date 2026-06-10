import { ref, onMounted, onUnmounted } from 'vue'
import axios from 'axios'

const INACTIVITY_TIMEOUT_MS = 30 * 60 * 1000 // 30 minutes
const HEARTBEAT_INTERVAL_MS = 60 * 1000       // 1 minute
const STARTED_AT_KEY = 'sre.server_started_at'

/**
 * Session health guard:
 * 1. Server restart detection — compares /healthz started_at with stored value
 * 2. Inactivity timeout — 30 min no user activity → force logout
 * 3. Visibility change — re-validate when user returns to tab
 * 4. Periodic heartbeat — every 1 min
 */
export function useSessionGuard() {
  const isOnline = ref(true)
  const sessionExpired = ref(false)
  const serverRestarted = ref(false)

  let heartbeatTimer: ReturnType<typeof setInterval> | null = null
  let inactivityTimer: ReturnType<typeof setTimeout> | null = null
  let consecutiveFailures = 0
  const MAX_FAILURES = 3
  let alertDismissed = false // prevent re-triggering after user dismisses

  const activityEvents = ['mousedown', 'keydown', 'touchstart', 'scroll']

  function resetInactivityTimer() {
    if (inactivityTimer) clearTimeout(inactivityTimer)
    inactivityTimer = setTimeout(() => {
      console.warn('[session] Inactivity timeout — forcing logout')
      sessionExpired.value = true
    }, INACTIVITY_TIMEOUT_MS)
  }

  async function checkHealth(): Promise<boolean> {
    // Skip if user already dismissed the alert (prevents infinite loop)
    if (alertDismissed) return true

    try {
      const res = await axios.get('/healthz', { timeout: 5000 })
      consecutiveFailures = 0
      isOnline.value = true

      // Detect server restart
      const remoteStartedAt = res.data?.started_at
      if (remoteStartedAt) {
        const localStartedAt = localStorage.getItem(STARTED_AT_KEY)
        if (localStartedAt && localStartedAt !== remoteStartedAt) {
          console.warn('[session] Server restarted — forcing logout')
          serverRestarted.value = true
          sessionExpired.value = true
          // Clear stored value to prevent re-triggering on next check
          localStorage.removeItem(STARTED_AT_KEY)
          return true
        }
        // Store current server's started_at
        localStorage.setItem(STARTED_AT_KEY, remoteStartedAt)
      }

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
    // Skip if alert is already showing (user hasn't acted yet)
    if (sessionExpired.value) return

    const token = localStorage.getItem('token')
    if (!token) {
      sessionExpired.value = true
      return
    }

    const healthy = await checkHealth()
    if (!healthy) return
    if (sessionExpired.value) return

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
    }
  }

  /** Call this after successful login to accept the current server's started_at */
  function acceptCurrentServer() {
    alertDismissed = false
    // Fetch current started_at and store it
    axios.get('/healthz', { timeout: 5000 }).then(res => {
      const remoteStartedAt = res.data?.started_at
      if (remoteStartedAt) {
        localStorage.setItem(STARTED_AT_KEY, remoteStartedAt)
      }
    }).catch(() => { /* ignore */ })
  }

  function handleVisibilityChange() {
    if (document.visibilityState === 'visible') {
      validateSession()
    }
  }

  function handleActivity() {
    resetInactivityTimer()
  }

  onMounted(() => {
    // Initial checks
    checkHealth()
    resetInactivityTimer()

    // Visibility change
    document.addEventListener('visibilitychange', handleVisibilityChange)

    // User activity tracking
    for (const evt of activityEvents) {
      document.addEventListener(evt, handleActivity, { passive: true })
    }

    // Periodic heartbeat
    heartbeatTimer = setInterval(checkHealth, HEARTBEAT_INTERVAL_MS)
  })

  onUnmounted(() => {
    if (heartbeatTimer) clearInterval(heartbeatTimer)
    if (inactivityTimer) clearTimeout(inactivityTimer)
    document.removeEventListener('visibilitychange', handleVisibilityChange)
    for (const evt of activityEvents) {
      document.removeEventListener(evt, handleActivity)
    }
  })

  function forceReconnect() {
    sessionExpired.value = false
    serverRestarted.value = false
    alertDismissed = false
    consecutiveFailures = 0
    validateSession()
  }

  /** Dismiss the alert without re-triggering */
  function dismiss() {
    sessionExpired.value = false
    serverRestarted.value = false
    alertDismissed = true
  }

  return {
    isOnline,
    sessionExpired,
    serverRestarted,
    validateSession,
    forceReconnect,
    acceptCurrentServer,
    dismiss,
  }
}
