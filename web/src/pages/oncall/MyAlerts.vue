<template>
  <div class="my-alerts-page">
    <n-page-header :title="t('myAlerts.title')" :subtitle="t('myAlerts.subtitle')">
      <template #extra>
        <n-space>
          <n-radio-group v-model:value="filter" size="small">
            <n-radio-button value="firing">{{ t('myAlerts.pending') }}</n-radio-button>
            <n-radio-button value="assigned">{{ t('myAlerts.assigned') }}</n-radio-button>
            <n-radio-button value="acknowledged">{{ t('myAlerts.acked') }}</n-radio-button>
            <n-radio-button value="all">{{ t('myAlerts.all') }}</n-radio-button>
          </n-radio-group>
          <n-button @click="refresh" :loading="loading">{{ t('myAlerts.refresh') }}</n-button>
        </n-space>
      </template>
    </n-page-header>

    <n-divider />

    <n-spin :show="loading">
      <n-empty v-if="!loading && !alerts.length" :description="t('myAlerts.noAlerts')" />

      <n-list bordered>
        <n-list-item v-for="alert in alerts" :key="alert.id">
          <template #prefix>
            <n-tag :type="severityType(alert.severity)" round>
              {{ alert.severity }}
            </n-tag>
          </template>

          <n-thing
            :title="alert.alert_name"
            :description="alert.summary || ''"
          >
            <template #header-extra>
              <n-text depth="3">{{ formatTime(alert.fired_at) }}</n-text>
            </template>

            <n-space size="small">
              <n-tag v-if="alert.source" size="small">source={{ alert.source }}</n-tag>
              <n-tag :type="statusType(alert.status)" size="small">
                {{ alert.status }}
              </n-tag>
            </n-space>

            <template #footer>
              <n-space>
                <n-button size="small" type="primary" @click="handleAck(alert)" v-if="alert.status === 'firing' || alert.status === 'assigned'">
                  {{ t('myAlerts.ack') }}
                </n-button>
                <n-button size="small" @click="handleResolve(alert)" v-if="alert.status !== 'resolved' && alert.status !== 'closed'">
                  {{ t('myAlerts.resolve') }}
                </n-button>
                <n-button size="small" tertiary @click="goDetail(alert)">
                  {{ t('myAlerts.detail') }}
                </n-button>
              </n-space>
            </template>
          </n-thing>
        </n-list-item>
      </n-list>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { alertEventApi } from '@/api/alert'

const router = useRouter()
const message = useMessage()
const { t } = useI18n()

const filter = ref<string>('firing')
const alerts = ref<any[]>([])
const loading = ref(false)

async function refresh() {
  loading.value = true
  try {
    const params: any = { view_mode: 'mine', page_size: 100 }
    if (filter.value !== 'all') {
      params.status = filter.value
    }
    const r = await alertEventApi.list(params)
    alerts.value = r.data?.items || []
  } catch (e: any) {
    message.error(t('myAlerts.loadError') + (e?.message || ''))
  } finally {
    loading.value = false
  }
}

async function handleAck(alert: any) {
  try {
    await alertEventApi.acknowledge(alert.id)
    message.success(t('myAlerts.ackedSuccess'))
    await refresh()
  } catch (e: any) {
    message.error(t('myAlerts.ackError') + (e?.message || ''))
  }
}

async function handleResolve(alert: any) {
  try {
    await alertEventApi.resolve(alert.id)
    message.success(t('myAlerts.resolvedSuccess'))
    await refresh()
  } catch (e: any) {
    message.error(t('myAlerts.resolveError') + (e?.message || ''))
  }
}

function goDetail(alert: any) {
  router.push(`/alert/events/${alert.id}`)
}

function severityType(sev: string) {
  return ({ critical: 'error', warning: 'warning', info: 'info' } as any)[sev] || 'default'
}

function statusType(status: string) {
  return ({ firing: 'error', assigned: 'warning', acknowledged: 'info', resolved: 'success', closed: 'default' } as any)[status] || 'default'
}

function formatTime(t: string) {
  if (!t) return '-'
  const date = new Date(t)
  const diffMs = Date.now() - date.getTime()
  const min = Math.floor(diffMs / 60000)
  if (min < 1) return t('myAlerts.justNow')
  if (min < 60) return t('myAlerts.minutesAgo', { n: min })
  const hr = Math.floor(min / 60)
  if (hr < 24) return t('myAlerts.hoursAgo', { n: hr })
  return date.toLocaleString()
}

watch(filter, refresh)
onMounted(refresh)
</script>

<style scoped>
.my-alerts-page {
  padding: 16px;
}
</style>
