<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { alertEventApi } from '@/api/alert'
import type { AlertEvent, AlertEventFilter } from '@/types'
import { getErrorMessage, relTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'

const router = useRouter()
const message = useMessage()
const { t } = useI18n()

const filter = ref<string>('firing')
const alerts = ref<AlertEvent[]>([])
const loading = ref(false)
const loadError = ref(false)

const emptyTitle = computed(() => {
  const map: Record<string, string> = {
    firing: t('myAlerts.emptyFiring'),
    assigned: t('myAlerts.emptyAssigned'),
    acknowledged: t('myAlerts.emptyAcked'),
    resolved: t('myAlerts.emptyResolved'),
    closed: t('myAlerts.emptyClosed'),
    all: t('myAlerts.emptyAll'),
  }
  return map[filter.value] || t('myAlerts.emptyAll')
})

const emptyHint = computed(() => {
  const map: Record<string, string> = {
    firing: t('myAlerts.emptyFiringHint'),
    assigned: t('myAlerts.emptyAssignedHint'),
    acknowledged: t('myAlerts.emptyAckedHint'),
    resolved: t('myAlerts.emptyResolvedHint'),
    closed: t('myAlerts.emptyClosedHint'),
    all: t('myAlerts.emptyAllHint'),
  }
  return map[filter.value] || t('myAlerts.emptyAllHint')
})

async function refresh() {
  loading.value = true
  loadError.value = false
  try {
    const params: AlertEventFilter = { view_mode: 'mine', page: 1, page_size: 100 }
    if (filter.value !== 'all') {
      params.status = [filter.value]
    }
    const r = await alertEventApi.list(params)
    alerts.value = r.data?.data?.list || []
  } catch {
    loadError.value = true
  } finally {
    loading.value = false
  }
}

async function handleAck(alert: AlertEvent) {
  try {
    await alertEventApi.acknowledge(alert.id)
    message.success(t('myAlerts.ackedSuccess'))
    await refresh()
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('myAlerts.ackError'))
  }
}

async function handleResolve(alert: AlertEvent) {
  try {
    await alertEventApi.resolve(alert.id)
    message.success(t('myAlerts.resolvedSuccess'))
    await refresh()
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('myAlerts.resolveError'))
  }
}

function goDetail(alert: AlertEvent) {
  router.push(`/alert/events/${alert.id}`)
}

function severityType(sev: string) {
  const map: Record<string, 'error' | 'warning' | 'info' | 'default'> = { critical: 'error', warning: 'warning', info: 'info' }
  return map[sev] || 'default'
}

function severityLabel(sev: string) {
  const map: Record<string, string> = {
    critical: t('alert.critical'),
    warning: t('alert.warning'),
    info: t('alert.info'),
  }
  return map[sev] || sev
}

function statusType(status: string) {
  const map: Record<string, 'error' | 'warning' | 'info' | 'default'> = { firing: 'error', assigned: 'warning', acknowledged: 'info', resolved: 'default', closed: 'default' }
  return map[status] || 'default'
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    firing: t('alert.firing'),
    assigned: t('alert.assigned'),
    acknowledged: t('alert.acknowledged'),
    resolved: t('alert.resolved'),
    closed: t('alert.closed'),
  }
  return map[status] || status
}

watch(filter, refresh)
onMounted(refresh)
</script>

<template>
  <div class="my-alerts-page">
    <PageHeader :title="t('myAlerts.title')" :subtitle="t('myAlerts.subtitle')">
      <template #actions>
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
    </PageHeader>

    <n-divider />

    <n-spin :show="loading">
      <div v-if="loadError" class="my-alerts-error">
        <n-result status="warning" :title="t('myAlerts.loadError')" :description="t('myAlerts.loadErrorDetail')">
          <template #footer>
            <n-button size="small" @click="refresh">{{ t('common.retry') }}</n-button>
          </template>
        </n-result>
      </div>

      <div v-else-if="!loading && !alerts.length" class="my-alerts-empty">
        <n-empty :description="emptyTitle">
          <template #extra>
            <p class="my-alerts-empty-hint">{{ emptyHint }}</p>
            <n-button size="small" type="primary" @click="router.push('/alert/events')">
              {{ t('myAlerts.viewAll') || t('myAlerts.all') }}
            </n-button>
          </template>
        </n-empty>
      </div>

      <n-list bordered>
        <n-list-item v-for="alert in alerts" :key="alert.id">
          <template #prefix>
            <n-tag :type="severityType(alert.severity)" round>
              {{ severityLabel(alert.severity) }}
            </n-tag>
          </template>

          <n-thing
            :title="alert.alert_name"
            :description="alert.annotations?.summary || alert.annotations?.description || ''"
          >
            <template #header-extra>
              <n-text depth="3">{{ relTime(alert.fired_at, t) }}</n-text>
            </template>

            <n-space size="small">
              <n-tag v-if="alert.source" size="small">source={{ alert.source }}</n-tag>
              <n-tag :type="statusType(alert.status)" size="small">
                {{ statusLabel(alert.status) }}
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

<style scoped>
.my-alerts-page {
  padding: 16px;
}
.my-alerts-error,
.my-alerts-empty {
  padding: 48px 20px;
}
.my-alerts-empty-hint {
  margin: 0;
  font-size: 12px;
  color: var(--sre-text-muted);
}
</style>
