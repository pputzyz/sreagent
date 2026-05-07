<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage, NTag, NButton, NSpace } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { incidentApi, alertV2Api } from '@/api'
import type { Incident, IncidentTimeline, AlertV2 } from '@/types'
import { formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import { ArrowBackOutline } from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const route = useRoute()
const router = useRouter()

const incidentId = computed(() => Number(route.params.id))
const incident = ref<Incident | null>(null)
const timeline = ref<IncidentTimeline[]>([])
const relatedAlerts = ref<AlertV2[]>([])
const loading = ref(false)
const activeTab = ref('overview')
const commentText = ref('')
const submittingComment = ref(false)

const severityTagType: Record<string, 'error' | 'warning' | 'info'> = {
  critical: 'error', warning: 'warning', info: 'info',
}
const statusTagType: Record<string, 'error' | 'warning' | 'success' | 'default'> = {
  triggered: 'error', processing: 'warning', closed: 'success',
}
const statusLabel: Record<string, string> = {
  triggered: 'incident.statusTriggered',
  processing: 'incident.statusProcessing',
  closed: 'incident.statusClosed',
}
const severityLabel: Record<string, string> = {
  critical: 'incident.severityCritical',
  warning: 'incident.severityWarning',
  info: 'incident.severityInfo',
}

async function load() {
  loading.value = true
  try {
    const [incRes, tlRes, alertRes] = await Promise.all([
      incidentApi.get(incidentId.value),
      incidentApi.getTimeline(incidentId.value),
      alertV2Api.list({ incident_id: incidentId.value, page: 1, page_size: 50 }),
    ])
    incident.value = incRes.data.data ?? null
    timeline.value = tlRes.data.data ?? []
    relatedAlerts.value = alertRes.data.data?.list ?? []
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function doAction(action: 'acknowledge' | 'close' | 'reopen' | 'escalate') {
  try {
    await incidentApi[action](incidentId.value)
    message.success(t('common.success'))
    await load()
  } catch (e: any) {
    message.error(e?.message ?? t('common.failed'))
  }
}

async function submitComment() {
  if (!commentText.value.trim()) return
  submittingComment.value = true
  try {
    await incidentApi.addComment(incidentId.value, commentText.value)
    commentText.value = ''
    const tlRes = await incidentApi.getTimeline(incidentId.value)
    timeline.value = tlRes.data.data ?? []
  } catch (e: any) {
    message.error(e?.message ?? t('common.failed'))
  } finally {
    submittingComment.value = false
  }
}

const alertColumns = computed(() => [
  {
    title: t('incident.severity'),
    key: 'severity',
    width: 90,
    render: (row: AlertV2) =>
      h(NTag, { type: (severityTagType as any)[row.severity] ?? 'default', size: 'small' },
        { default: () => row.severity.toUpperCase() }),
  },
  {
    title: t('common.name'),
    key: 'title',
    render: (row: AlertV2) => h('span', {}, row.title),
  },
  {
    title: t('common.status'),
    key: 'status',
    width: 100,
    render: (row: AlertV2) =>
      h(NTag, { type: row.status === 'firing' ? 'error' : 'success', size: 'small' },
        { default: () => row.status === 'firing' ? 'Firing' : 'Resolved' }),
  },
  {
    title: t('alertV2.lastFiredAt'),
    key: 'last_fired_at',
    render: (row: AlertV2) => h('span', { style: 'font-size:12px' }, formatTime(row.last_fired_at)),
  },
  {
    title: t('alertV2.fireCount'),
    key: 'fire_count',
    width: 90,
    render: (row: AlertV2) => h('span', {}, String(row.fire_count)),
  },
])

onMounted(load)
</script>

<template>
  <div class="incident-detail">
    <PageHeader
      :title="incident ? `#${incident.id} ${incident.title}` : t('incident.title')"
      :subtitle="incident?.channel?.name ?? ''"
    >
      <template #actions>
        <n-button quaternary @click="router.back()">
          <template #icon><n-icon :component="ArrowBackOutline" /></template>
          {{ t('common.back') }}
        </n-button>
      </template>
    </PageHeader>

    <n-spin :show="loading">
      <div v-if="incident" class="detail-layout">
        <!-- Left: main content -->
        <div class="detail-main">
          <!-- Action bar -->
          <n-card :bordered="false" class="action-card">
            <n-space>
              <n-tag :type="statusTagType[incident.status] ?? 'default'" size="medium">
                {{ t(statusLabel[incident.status] ?? incident.status) }}
              </n-tag>
              <n-tag :type="severityTagType[incident.severity] ?? 'default'" size="medium">
                {{ t(severityLabel[incident.severity] ?? incident.severity) }}
              </n-tag>
              <n-button
                v-if="incident.status === 'triggered'"
                type="primary" size="small"
                @click="doAction('acknowledge')"
              >{{ t('incident.acknowledge') }}</n-button>
              <n-button
                v-if="incident.status !== 'closed'"
                size="small"
                @click="doAction('close')"
              >{{ t('incident.close') }}</n-button>
              <n-button
                v-if="incident.status === 'closed'"
                size="small"
                @click="doAction('reopen')"
              >{{ t('incident.reopen') }}</n-button>
              <n-button size="small" @click="doAction('escalate')">
                {{ t('incident.escalate') }}
              </n-button>
            </n-space>
          </n-card>

          <!-- Tabs -->
          <n-card :bordered="false" class="tabs-card">
            <n-tabs v-model:value="activeTab" type="line" animated>

              <!-- Overview -->
              <n-tab-pane name="overview" :tab="'Overview'">
                <n-descriptions :columns="2" label-placement="left" bordered size="small">
                  <n-descriptions-item :label="t('incident.triggeredAt')">
                    {{ formatTime(incident.triggered_at) }}
                  </n-descriptions-item>
                  <n-descriptions-item v-if="incident.acknowledged_at" :label="t('incident.acknowledgedAt')">
                    {{ formatTime(incident.acknowledged_at) }}
                  </n-descriptions-item>
                  <n-descriptions-item v-if="incident.resolved_at" :label="t('incident.resolvedAt')">
                    {{ formatTime(incident.resolved_at) }}
                  </n-descriptions-item>
                  <n-descriptions-item v-if="incident.closed_at" :label="t('incident.closedAt')">
                    {{ formatTime(incident.closed_at) }}
                  </n-descriptions-item>
                  <n-descriptions-item :label="t('incident.alertCount')">
                    {{ incident.alert_count }}
                  </n-descriptions-item>
                  <n-descriptions-item :label="t('incident.assignee')">
                    {{ incident.assigned_user?.display_name ?? incident.assigned_user?.username ?? '—' }}
                  </n-descriptions-item>
                </n-descriptions>

                <div v-if="incident.description" style="margin-top:16px">
                  <p style="color:var(--sre-text-secondary);font-size:13px">{{ incident.description }}</p>
                </div>
              </n-tab-pane>

              <!-- Related Alerts -->
              <n-tab-pane name="alerts" :tab="t('alertV2.title')">
                <n-data-table
                  :columns="alertColumns"
                  :data="relatedAlerts"
                  :row-key="(row: AlertV2) => row.id"
                  size="small"
                />
              </n-tab-pane>

              <!-- Timeline -->
              <n-tab-pane name="timeline" :tab="t('incident.timeline')">
                <div class="timeline-list">
                  <div v-if="timeline.length === 0" style="text-align:center;padding:24px">
                    <n-empty :description="t('incident.noTimeline')" />
                  </div>
                  <div v-for="entry in timeline" :key="entry.id" class="timeline-entry">
                    <div class="tl-dot" />
                    <div class="tl-content">
                      <span class="tl-action">{{ entry.action }}</span>
                      <span v-if="entry.actor" class="tl-actor"> · {{ entry.actor.display_name ?? entry.actor.username }}</span>
                      <span class="tl-time"> · {{ formatTime(entry.created_at) }}</span>
                      <p v-if="entry.content" class="tl-text">{{ entry.content }}</p>
                    </div>
                  </div>
                </div>

                <!-- Add comment -->
                <div class="comment-box">
                  <n-input
                    v-model:value="commentText"
                    type="textarea"
                    :rows="2"
                    :placeholder="t('incident.commentPlaceholder')"
                  />
                  <n-button
                    type="primary" size="small"
                    :loading="submittingComment"
                    style="margin-top:8px"
                    @click="submitComment"
                  >{{ t('incident.addComment') }}</n-button>
                </div>
              </n-tab-pane>

            </n-tabs>
          </n-card>
        </div>

        <!-- Right: info sidebar -->
        <div class="detail-sidebar">
          <n-card :bordered="false" class="info-card" title="Info">
            <n-descriptions :columns="1" label-placement="top" size="small">
              <n-descriptions-item :label="t('incident.channel')">
                <a
                  v-if="incident.channel"
                  style="cursor:pointer;color:var(--sre-primary)"
                  @click="$router.push(`/channels/${incident.channel_id}`)"
                >{{ incident.channel.name }}</a>
                <span v-else>—</span>
              </n-descriptions-item>
              <n-descriptions-item :label="t('incident.severity')">
                <n-tag :type="severityTagType[incident.severity] ?? 'default'" size="small">
                  {{ t(severityLabel[incident.severity] ?? incident.severity) }}
                </n-tag>
              </n-descriptions-item>
              <n-descriptions-item :label="t('incident.status')">
                <n-tag :type="statusTagType[incident.status] ?? 'default'" size="small">
                  {{ t(statusLabel[incident.status] ?? incident.status) }}
                </n-tag>
              </n-descriptions-item>
            </n-descriptions>
          </n-card>
        </div>
      </div>
    </n-spin>
  </div>
</template>

<style scoped>
.incident-detail { max-width: 1400px; }

.action-card { border-radius: 12px; margin-bottom: 16px; }
.tabs-card { border-radius: 12px; }
.info-card { border-radius: 12px; }

.detail-layout {
  display: grid;
  grid-template-columns: 1fr 260px;
  gap: 16px;
  align-items: start;
}

.timeline-list { margin-bottom: 16px; }

.timeline-entry {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
}

.tl-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--sre-primary);
  margin-top: 5px;
  flex-shrink: 0;
}

.tl-content { flex: 1; }
.tl-action { font-weight: 600; font-size: 13px; }
.tl-actor, .tl-time { font-size: 12px; color: var(--sre-text-secondary); }
.tl-text { font-size: 13px; margin: 4px 0 0; color: var(--sre-text-primary); }

.comment-box {
  border-top: 1px solid var(--sre-border);
  padding-top: 12px;
}
</style>
