<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { NIcon, NBadge } from 'naive-ui'
import { NotificationsOutline } from '@vicons/ionicons5'
import { notificationCenterApi } from '@/api'

const router = useRouter()
const unreadCount = ref(0)
let interval: ReturnType<typeof setInterval> | null = null

async function fetchCount() {
  try {
    const { data } = await notificationCenterApi.unreadCount()
    unreadCount.value = data.data?.count || 0
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
  if (interval) clearInterval(interval)
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
