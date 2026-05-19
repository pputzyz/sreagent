<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, NButton, NIcon, NRadioGroup, NRadioButton, NSpin, NEmpty, NTag } from 'naive-ui'
import { useRouter } from 'vue-router'
import { CheckmarkDoneOutline, TrashOutline, NotificationsOutline } from '@vicons/ionicons5'
import { notificationCenterApi } from '@/api'
import type { UserNotification } from '@/api/center'
import { getErrorMessage, formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

const loading = ref(false)
const items = ref<UserNotification[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const filter = ref<'all' | 'unread' | 'read'>('all')

async function fetchList() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    if (filter.value === 'unread') params.is_read = false
    if (filter.value === 'read') params.is_read = true
    const { data } = await notificationCenterApi.list(params)
    items.value = data.data.list || []
    total.value = data.data.total || 0
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function handleMarkRead(item: UserNotification) {
  try {
    await notificationCenterApi.markRead(item.id)
    item.is_read = true
    if (item.link) router.push(item.link)
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

async function handleMarkAllRead() {
  try {
    await notificationCenterApi.markAllRead()
    message.success(t('notification.markAllReadSuccess'))
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

async function handleDelete(id: number) {
  try {
    await notificationCenterApi.delete(id)
    items.value = items.value.filter(i => i.id !== id)
    total.value--
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

function typeColor(type: string): 'error' | 'warning' | 'info' | 'default' {
  if (type === 'alert') return 'error'
  if (type === 'incident') return 'warning'
  if (type === 'todo') return 'info'
  return 'default'
}

onMounted(fetchList)
</script>

<template>
  <div class="notif-center">
    <PageHeader :title="t('notification.centerTitle')" :subtitle="t('notification.centerSubtitle')">
      <template #actions>
        <NButton size="small" quaternary @click="fetchList">
          <template #icon><NIcon :component="NotificationsOutline" /></template>
          {{ t('common.refresh') }}
        </NButton>
        <NButton size="small" @click="handleMarkAllRead">
          <template #icon><NIcon :component="CheckmarkDoneOutline" /></template>
          {{ t('notification.markAllRead') }}
        </NButton>
      </template>
    </PageHeader>

    <div class="notif-filter">
      <NRadioGroup v-model:value="filter" size="small" @update:value="fetchList">
        <NRadioButton value="all">{{ t('common.all') }}</NRadioButton>
        <NRadioButton value="unread">{{ t('notification.unread') }}</NRadioButton>
        <NRadioButton value="read">{{ t('notification.read') }}</NRadioButton>
      </NRadioGroup>
    </div>

    <NSpin :show="loading">
      <div v-if="items.length === 0 && !loading" class="notif-empty">
        <NEmpty :description="t('notification.noNotifications')" />
      </div>
      <div v-else class="notif-list">
        <div
          v-for="item in items"
          :key="item.id"
          class="notif-item sre-row-card"
          :class="{ unread: !item.is_read }"
          @click="handleMarkRead(item)"
        >
          <div class="notif-main">
            <div class="notif-head">
              <NTag :type="typeColor(item.type)" size="small" :bordered="false">{{ item.type }}</NTag>
              <span class="notif-title">{{ item.title }}</span>
            </div>
            <div v-if="item.content" class="notif-content">{{ item.content }}</div>
            <div class="notif-meta tnum">{{ formatTime(item.created_at) }}</div>
          </div>
          <div class="notif-actions" @click.stop>
            <NButton size="tiny" quaternary @click="handleDelete(item.id)">
              <template #icon><NIcon :component="TrashOutline" :size="14" /></template>
            </NButton>
          </div>
        </div>
      </div>
    </NSpin>
  </div>
</template>

<style scoped>
.notif-center { font-family: var(--sre-font-sans); }
.notif-filter { margin: 12px 0 16px; }
.notif-list { display: flex; flex-direction: column; gap: 6px; }
.notif-empty { padding: 60px 0; text-align: center; }
.notif-item {
  display: flex; align-items: flex-start; gap: 12px;
  padding: 12px 14px; cursor: pointer;
}
.notif-item.unread { border-left: 3px solid var(--sre-primary); }
.notif-main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 4px; }
.notif-head { display: flex; align-items: center; gap: 8px; }
.notif-title { font-size: 14px; font-weight: 600; color: var(--sre-text-primary); }
.notif-content { font-size: 12px; color: var(--sre-text-secondary); }
.notif-meta { font-size: 11px; color: var(--sre-text-tertiary); }
.notif-actions { flex-shrink: 0; }
</style>
