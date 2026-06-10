import { ref, onMounted, onUnmounted } from 'vue'
import request from '@/api/request'
import axios from 'axios'

const INACTIVITY_TIMEOUT_MS = 30 * 60 * 1000 // 30 minutes
const HEARTBEAT_INTERVAL_MS = 60 * 1000       // 1 minute
const STARTED_AT_KEY = 'sre.server_started_at'

/**
 * Session Guard — only two jobs:
 * 1. Detect server restart (via /healthz started_at)
 * 2. Detect user inactivity (30 min no mouse/keyboard)
 *
 * Token refresh and 401 handling are delegated to request.ts interceptor.
 */
export function useSessionGuard() {
  const isOnline = ref(true)
  const sessionExpired = ref(false)
  const serverRestarted = ref(false)

  let heartbeatTimer: ReturnType<typeof setInterval> | null = null
  let inactivityTimer: ReturnType<typeof setTimeout> | null = null
  let consecutiveFailures = 0
  const MAX_FAILURES = 3
  let isTabVisible = true

  const activityEvents = ['mousedown', 'keydown', 'touchstart', 'scroll'] as const

  // ─── Inactivity Timer ────────────────────────────────────────
  function resetInactivityTimer() {
    if (inactivityTimer) clearTimeout(inactivityTimer)
    inactivityTimer = setTimeout(() => {
      if (isTabVisible) {
        console.warn('[session] Inactivity timeout (30 min)')
        sessionExpired.value = true
      }
      // If tab is hidden, don't timeout — reset on next visibility
    }, INACTIVITY_TIMEOUT_MS)
  }

  function handleActivity() {
    // Only count activity when tab is visible
    if (isTabVisible) {
      resetInactivityTimer()
    }
  }

  // ─── Health Check (server restart detection) ─────────────────
  async function checkHealth(): Promise<boolean> {
    try {
      // Use raw axios for /healthz — it's outside /api/v1 and needs no auth
      const res = await axios.get('/healthz', { timeout: 5000 })
      consecutiveFailures = 0
      isOnline.value = true

      const remoteStartedAt = res.data?.started_at
      if (remoteStartedAt) {
        const localStartedAt = localStorage.getItem(STARTED_AT_KEY)
        if (localStartedAt && localStartedAt !== remoteStartedAt) {
          console.warn('[session] Server restarted')
          serverRestarted.value = true
          sessionExpired.value = true
          localStorage.removeItem(STARTED_AT_KEY)
          return true
        }
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

  // ─── Visibility Change ───────────────────────────────────────
  function handleVisibilityChange() {
    if (document.visibilityState === 'visible') {
      isTabVisible = true
      resetInactivityTimer()
      // Only check server health on tab return — don't refresh token
      // (401 interceptor will handle refresh when the next API call fails)
      checkHealth()
    } else {
      isTabVisible = false
      // Pause inactivity timer while tab is hidden
      if (inactivityTimer) clearTimeout(inactivityTimer)
    }
  }

  // ─── Lifecycle ───────────────────────────────────────────────
  onMounted(() => {
    isTabVisible = document.visibilityState === 'visible'
    checkHealth()
    resetInactivityTimer()

    document.addEventListener('visibilitychange', handleVisibilityChange)
    for (const evt of activityEvents) {
      document.addEventListener(evt, handleActivity, { passive: true })
    }
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

  // ─── Public API ──────────────────────────────────────────────

  /** Call after successful login to sync server started_at */
  function acceptCurrentServer() {
    axios.get('/healthz', { timeout: 5000 }).then(res => {
      const ts = res.data?.started_at
      if (ts) localStorage.setItem(STARTED_AT_KEY, ts)
    }).catch(() => { /* ignore */ })
    sessionExpired.value = false
    serverRestarted.value = false
    consecutiveFailures = 0
    isTabVisible = true
    resetInactivityTimer()
  }

  /** Dismiss the modal and pause checks until next login */
  function dismiss() {
    sessionExpired.value = false
    serverRestarted.value = false
  }

  /** Force reconnect after login */
  function forceReconnect() {
    sessionExpired.value = false
    serverRestarted.value = false
    consecutiveFailures = 0
    acceptCurrentServer()
  }

  return {
    isOnline,
    sessionExpired,
    serverRestarted,
    acceptCurrentServer,
    forceReconnect,
    dismiss,
  }
}
