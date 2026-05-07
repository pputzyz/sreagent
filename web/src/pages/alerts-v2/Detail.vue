<script setup lang="ts">
import { ref, onMounted, computed, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage, NTag } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { alertV2Api } from '@/api'
import type { AlertV2, AlertEventV2 } from '@/types'
import { formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import { ArrowBackOutline, RefreshOutline } from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const route = useRoute()
const router = useRouter()

const alertId = computed(() => Number(route.params.id))
const alert = ref<AlertV2 | null>(null)
const events = ref<AlertEventV2[]>([])
const eventsTotal = ref(0)
const eventsPage = ref(1)
const eventsPageSize = ref(20)
const loading = ref(false)
const eventsLoading = ref(false)
const activeTab = ref('overview')

const severityTagType: Record<string, 'error' | 'warning' | 'info' | 'default'> = {
  critical: 'error', warning: 'warning', info: 'info',
  p0: 'error', p1: 'error', p2: 'warning', p3: 'warning', p4: 'info',
}

async function loadAlert() {
  loading.value = true
  try {
    const res = await alertV2Api.get(alertId.value)
    alert.value = res.data.data ?? null
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadEvents() {
  eventsLoading.value = true
  try {
    const res = await alertV2Api.listEvents(alertId.value, {
      page: eventsPage.value,
      page_size: eventsPageSize.value,
    })
    events.value = res.data.data?.list ?? []
    eventsTotal.value = res.data.data?.total ?? 0
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
  } finally {
    eventsLoading.value = false
  }
}

const eventColumns = computed(() => [
  {
    title: t('incident.severity'),
    key: 'event_severity',
    width: 90,
    render: (row: AlertEventV2) =>
      h(NTag, { type: severityTagType[row.event_severity] ?? 'default', size: 'small' },
        { default: () => row.event_severity.toUpperCase() }),
  },
  {
    title: t('common.status'),
    key: 'event_status',
    width: 100,
    render: (row: AlertEventV2) =>
      h(NTag, { type: row.event_status === 'firing' ? 'error' : 'success', size: 'small' },
        { default: () => row.event_status === 'firing' ? 'Firing' : 'Resolved' }),
  },
  {
    title: 'Value',
    key: 'value',
    width: 100,
    render: (row: AlertEventV2) => h('span', { style: 'font-family:monospace;font-size:12px' },
      row.value.toFixed(4)),
  },
  {
    title: 'Timestamp',
    key: 'timestamp',
    render: (row: AlertEventV2) => h('span', { style: 'font-size:12px' }, formatTime(row.timestamp)),
  },
  {
    title: 'Fingerprint',
    key: 'fingerprint',
    render: (row: AlertEventV2) => h('span', {
      style: 'font-family:monospace;font-size:11px;color:var(--sre-text-secondary)',
    }, row.fingerprint ? row.fingerprint.substring(0, 12) + '…' : '—'),
  },
])

onMounted(async () => {
  await loadAlert()
  await loadEvents()
})
</script>

<template>
  <div class="alert-detail">
    <PageHeader
      :title="alert?.title ?? t('alertV2.title')"
      :subtitle="alert ? `${alert.alert_key}` : ''"
    >
      <template #actions>
        <n-button quaternary @click="router.back()">
          <template #icon><n-icon :component="ArrowBackOutline" /></template>
          {{ t('common.back') }}
        </n-button>
        <n-button circle quaternary @click="loadAlert(); loadEvents()">
          <template #icon><n-icon :component="RefreshOutline" /></template>
        </n-button>
      </template>
    </PageHeader>

    <n-spin :show="loading">
      <div v-if="alert" class="detail-layout">

        <!-- Left: tabs -->
        <div class="detail-main">
          <n-card :bordered="false" class="tabs-card">
            <n-tabs v-model:value="activeTab" type="line" animated>

              <!-- Overview tab -->
              <n-tab-pane name="overview" :tab="'Overview'">
                <div class="badge-row">
                  <n-tag :type="alert.status === 'firing' ? 'error' : 'success'" size="medium">
                    {{ alert.status === 'firing' ? 'Firing' : 'Resolved' }}
                  </n-tag>
                  <n-tag :type="severityTagType[alert.severity] ?? 'default'" size="medium">
                    {{ alert.severity.toUpperCase() }}
                  </n-tag>
                </div>

                <n-descriptions :columns="2" label-placement="left" bordered size="small" style="margin-top:16px">
                  <n-descriptions-item :label="t('alertV2.alertKey')">
                    <span style="font-family:monospace;font-size:12px">{{ alert.alert_key }}</span>
                  </n-descriptions-item>
                  <n-descriptions-item label="Source">
                    {{ alert.source || '—' }}
                  </n-descriptions-item>
                  <n-descriptions-item :label="t('alertV2.firstFiredAt')">
                    {{ formatTime(alert.first_fired_at) }}
                  </n-descriptions-item>
                  <n-descriptions-item :label="t('alertV2.lastFiredAt')">
                    {{ formatTime(alert.last_fired_at) }}
                  </n-descriptions-item>
                  <n-descriptions-item v-if="alert.resolved_at" label="Resolved At">
                    {{ formatTime(alert.resolved_at) }}
                  </n-descriptions-item>
                  <n-descriptions-item :label="t('alertV2.fireCount')">
                    {{ alert.fire_count }}
                  </n-descriptions-item>
                  <n-descriptions-item :label="t('alertV2.eventCount')">
                    {{ alert.event_count }}
                  </n-descriptions-item>
                </n-descriptions>

                <!-- Labels -->
                <div v-if="alert.labels && Object.keys(alert.labels).length" style="margin-top:20px">
                  <div style="font-size:13px;font-weight:600;margin-bottom:8px;color:var(--sre-text-secondary)">Labels</div>
                  <n-space wrap>
                    <n-tag
                      v-for="(v, k) in alert.labels"
                      :key="k"
                      size="small"
                      style="font-family:monospace;font-size:11px"
                    >{{ k }}="{{ v }}"</n-tag>
                  </n-space>
                </div>

                <!-- Annotations -->
                <div v-if="alert.annotations && Object.keys(alert.annotations).length" style="margin-top:16px">
                  <div style="font-size:13px;font-weight:600;margin-bottom:8px;color:var(--sre-text-secondary)">Annotations</div>
                  <n-descriptions :columns="1" label-placement="left" size="small">
                    <n-descriptions-item v-for="(v, k) in alert.annotations" :key="k" :label="k">
                      {{ v }}
                    </n-descriptions-item>
                  </n-descriptions>
                </div>
              </n-tab-pane>

              <!-- Events tab -->
              <n-tab-pane name="events" :tab="t('alertV2.events')">
                <n-data-table
                  :loading="eventsLoading"
                  :columns="eventColumns"
                  :data="events"
                  :row-key="(row: AlertEventV2) => row.id"
                  size="small"
                />
                <div v-if="eventsTotal > eventsPageSize" class="pagination-row">
                  <n-pagination
                    v-model:page="eventsPage"
                    :page-count="Math.ceil(eventsTotal / eventsPageSize)"
                    @update:page="loadEvents"
                  />
                </div>
              </n-tab-pane>

            </n-tabs>
          </n-card>
        </div>

        <!-- Right: sidebar -->
        <div class="detail-sidebar">
          <n-card :bordered="false" class="info-card" title="Links">
            <n-descriptions :columns="1" label-placement="top" size="small">
              <n-descriptions-item :label="t('alertV2.linkedChannel')">
                <a
                  v-if="alert.channel"
                  style="cursor:pointer;color:var(--sre-primary)"
                  @click="router.push(`/channels/${alert.channel_id}`)"
                >{{ alert.channel.name }}</a>
                <span v-else>—</span>
              </n-descriptions-item>
              <n-descriptions-item :label="t('alertV2.linkedIncident')">
                <a
                  v-if="alert.incident"
                  style="cursor:pointer;color:var(--sre-primary)"
                  @click="router.push(`/incidents/${alert.incident_id}`)"
                >#{{ alert.incident_id }} {{ alert.incident.title }}</a>
                <span v-else>—</span>
              </n-descriptions-item>
            </n-descriptions>
          </n-card>

          <n-card v-if="alert.generator_url" :bordered="false" class="info-card" title="Generator" style="margin-top:12px">
            <a :href="alert.generator_url" target="_blank" style="font-size:12px;word-break:break-all">
              {{ alert.generator_url }}
            </a>
          </n-card>
        </div>
      </div>
    </n-spin>
  </div>
</template>

<style scoped>
.alert-detail { max-width: 1400px; }
.tabs-card { border-radius: 12px; }
.info-card { border-radius: 12px; }

.detail-layout {
  display: grid;
  grid-template-columns: 1fr 260px;
  gap: 16px;
  align-items: start;
}

.badge-row {
  display: flex;
  gap: 8px;
  margin-bottom: 4px;
}

.pagination-row {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
