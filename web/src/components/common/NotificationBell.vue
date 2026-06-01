<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { NIcon, NBadge } from 'naive-ui'
import { NotificationsOutline } from '@vicons/ionicons5'
import { notificationCenterApi } from '@/api'

const router = useRouter()
const unreadCount = ref(0)
const previousCount = ref(0)
let interval: ReturnType<typeof setInterval> | null = null
let audioCtx: AudioContext | null = null
let alive = true

// FE6-8: Notification sound using Web Audio API
// Respects user preference in localStorage: sre.notification.sound ('on' | 'off', default 'on')
function isSoundEnabled(): boolean {
  return localStorage.getItem('sre.notification.sound') !== 'off'
}

async function playNotificationSound() {
  if (!isSoundEnabled()) return
  try {
    if (!audioCtx) audioCtx = new AudioContext()
    if (audioCtx.state === 'suspended') await audioCtx.resume()
    const ctx = audioCtx
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.type = 'sine'
    osc.frequency.setValueAtTime(880, ctx.currentTime)
    osc.frequency.setValueAtTime(1100, ctx.currentTime + 0.08)
    gain.gain.setValueAtTime(0.15, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.3)
    osc.start(ctx.currentTime)
    osc.stop(ctx.currentTime + 0.3)
  } catch {
    // Audio not available (e.g., autoplay policy blocked)
  }
}

async function fetchCount() {
  try {
    const { data } = await notificationCenterApi.unreadCount()
    if (!alive) return
    const newCount = data.data?.count || 0
    // Play sound when count increases (skip first load)
    if (previousCount.value > 0 && newCount > previousCount.value) {
      playNotificationSound()
    }
    previousCount.value = newCount
    unreadCount.value = newCount
  } catch {
    // ignore
  }
}

function handleClick() {
  router.push('/notifications')
}

onMounted(() => {
  fetchCount()
  interval = setInterval(fetchCount, 30000) // poll every 30s
})

onUnmounted(() => {
  alive = false
  if (interval) clearInterval(interval)
  if (audioCtx) {
    audioCtx.close().catch(() => {})
    audioCtx = null
  }
})
</script>

<template>
  <button
    v-ripple
    class="topbar-btn"
    @click="handleClick"
    :title="$t('notification.centerTitle')"
    :aria-label="unreadCount > 0 ? `${$t('notification.centerTitle')} (${unreadCount})` : $t('notification.centerTitle')"
  >
    <NBadge :value="unreadCount" :max="99" :offset="[-4, -2]" :show="unreadCount > 0">
      <NIcon :component="NotificationsOutline" :size="16" />
    </NBadge>
  </button>
</template>
