<script setup lang="ts">
import { onMounted, computed, shallowRef, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { alertV2Api } from '@/api'
import type { AlertV2, AlertEventV2 } from '@/types'
import { formatTime } from '@/utils/format'
import {
  ArrowBackOutline,
  RefreshOutline,
  VolumeOffOutline,
} from '@vicons/ionicons5'
import QuickSilenceModal from '@/components/noise/QuickSilenceModal.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'

const { t } = useI18n()
const message = useMessage()
const route = useRoute()
const router = useRouter()

const alertId = computed(() => Number(route.params.id))
const alert = shallowRef<AlertV2 | null>(null)
const events = shallowRef<AlertEventV2[]>([])
const eventsTotal = ref(0)
const eventsPage = ref(1)
const eventsPageSize = ref(50)
const loading = ref(false)
const eventsLoading = ref(false)
const activeTab = ref('overview')
const showQuickSilence = ref(false)

const SEVERITY_LABEL: Record<string, string> = {
  critical: 'Critical', warning: 'Warning', info: 'Info',
  p0: 'P0', p1: 'P1', p2: 'P2', p3: 'P3', p4: 'P4',
}

function severityLabel(s?: string) {
  if (!s) return ''
  return SEVERITY_LABEL[s] ?? s.toUpperCase()
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
    const list = res.data.data?.list ?? []
    // sort desc by timestamp
    events.value = [...list].sort((a, b) =>
      new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
    eventsTotal.value = res.data.data?.total ?? 0
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
  } finally {
    eventsLoading.value = false
  }
}

const labelEntries = computed(() =>
  alert.value?.labels ? Object.entries(alert.value.labels) : [])
const annotationEntries = computed(() =>
  alert.value?.annotations ? Object.entries(alert.value.annotations) : [])

onMounted(async () => {
  await loadAlert()
  await loadEvents()
})

function refreshAll() {
  loadAlert()
  loadEvents()
}
</script>

<template>
  <div class="alert-detail">
    <!-- Header -->
    <div class="detail-header">
      <div class="header-top">
        <n-button quaternary circle size="small" @click="router.back()">
          <template #icon><n-icon :component="ArrowBackOutline" /></template>
        </n-button>
        <h1 class="detail-title">{{ alert?.title ?? t('alertV2.title') }}</h1>
        <div class="header-actions">
          <n-button
            v-if="alert"
            size="small"
            type="warning"
            @click="showQuickSilence = true"
          >
            <template #icon><n-icon :component="VolumeOffOutline" /></template>
            快速静默
          </n-button>
          <n-button circle quaternary size="small" @click="refreshAll">
            <template #icon><n-icon :component="RefreshOutline" /></template>
          </n-button>
        </div>
      </div>

      <div
        v-if="alert"
        class="header-sub sre-row-card"
        :data-severity="alert.severity"
        data-static
      >
        <span class="sre-dot" :data-severity="alert.severity"></span>
        <span class="hsub-sev">{{ severityLabel(alert.severity) }}</span>
        <span class="sre-meta-divider"></span>
        <span class="hsub-status">{{ alert.status }}</span>
        <span class="sre-meta-divider"></span>
        <span class="hsub-key">
          <span class="hsub-key-label">alert_key:</span>
          <code>{{ alert.alert_key }}</code>
        </span>
      </div>
    </div>

    <LoadingSkeleton v-if="loading && !alert" :rows="6" variant="row" />
    <n-spin v-else :show="loading">
      <div v-if="alert" class="detail-layout sre-fadein">

        <!-- LEFT: tabs -->
        <div class="detail-main">
          <n-tabs v-model:value="activeTab" type="line" animated>

            <!-- Overview -->
            <n-tab-pane name="overview" tab="Overview">
              <div class="ov-section" v-if="annotationEntries.length">
                <div class="sre-label-eyebrow">Description</div>
                <div class="ov-desc">
                  <div
                    v-for="[k, v] in annotationEntries"
                    :key="k"
                    class="annot-row"
                  >
                    <span class="annot-key">{{ k }}</span>
                    <span class="annot-val">{{ v }}</span>
                  </div>
                </div>
              </div>

              <div class="ov-section" v-if="labelEntries.length">
                <div class="sre-label-eyebrow">Labels</div>
                <div class="chip-row">
                  <span
                    v-for="[k, v] in labelEntries"
                    :key="k"
                    class="label-chip"
                  ><span class="lc-k">{{ k }}</span><span class="lc-eq">=</span><span class="lc-v">{{ v }}</span></span>
                </div>
              </div>

              <div class="ov-section">
                <div class="sre-label-eyebrow">Fire Summary</div>
                <div class="summary-grid">
                  <div class="sg-item">
                    <div class="sg-label">Fire count</div>
                    <div class="sg-value tnum">{{ alert.fire_count }}</div>
                  </div>
                  <div class="sg-item">
                    <div class="sg-label">Event count</div>
                    <div class="sg-value tnum">{{ alert.event_count }}</div>
                  </div>
                  <div class="sg-item">
                    <div class="sg-label">First fired</div>
                    <div class="sg-value tnum">{{ formatTime(alert.first_fired_at) }}</div>
                  </div>
                  <div class="sg-item">
                    <div class="sg-label">Last fired</div>
                    <div class="sg-value tnum">{{ formatTime(alert.last_fired_at) }}</div>
                  </div>
                </div>
              </div>
            </n-tab-pane>

            <!-- Events -->
            <n-tab-pane name="events" :tab="t('alertV2.events') || 'Events'">
              <n-spin :show="eventsLoading">
                <div v-if="!events.length" class="ev-empty">No events</div>
                <div v-else class="event-list">
                  <div
                    v-for="ev in events"
                    :key="ev.id"
                    class="event-row"
                    :data-status="ev.event_status"
                  >
                    <span class="sre-dot" :data-severity="ev.event_severity"></span>
                    <span class="ev-time tnum">{{ formatTime(ev.timestamp) }}</span>
                    <span class="ev-value tnum">{{ ev.value.toFixed(4) }}</span>
                    <span class="ev-status" :data-status="ev.event_status">
                      {{ ev.event_status }}
                    </span>
                    <code v-if="ev.fingerprint" class="ev-fp">
                      {{ ev.fingerprint.substring(0, 12) }}…
                    </code>
                  </div>
                </div>
                <div v-if="eventsTotal > eventsPageSize" class="pagination">
                  <n-pagination
                    v-model:page="eventsPage"
                    :page-count="Math.ceil(eventsTotal / eventsPageSize)"
                    @update:page="loadEvents"
                  />
                </div>
              </n-spin>
            </n-tab-pane>
          </n-tabs>
        </div>

        <!-- RIGHT: sidebar -->
        <aside class="detail-sidebar">
          <section class="side-card">
            <div class="sre-label-eyebrow">Key info</div>
            <dl class="kv-list">
              <div class="kv-item">
                <dt>Severity</dt>
                <dd>
                  <span class="sre-dot" :data-severity="alert.severity"></span>
                  {{ severityLabel(alert.severity) }}
                </dd>
              </div>
              <div class="kv-item">
                <dt>Status</dt>
                <dd>{{ alert.status }}</dd>
              </div>
              <div class="kv-item">
                <dt>First fired</dt>
                <dd class="tnum">{{ formatTime(alert.first_fired_at) }}</dd>
              </div>
              <div class="kv-item">
                <dt>Last fired</dt>
                <dd class="tnum">{{ formatTime(alert.last_fired_at) }}</dd>
              </div>
              <div v-if="alert.resolved_at" class="kv-item">
                <dt>Resolved</dt>
                <dd class="tnum">{{ formatTime(alert.resolved_at) }}</dd>
              </div>
              <div class="kv-item">
                <dt>Event count</dt>
                <dd class="tnum">{{ alert.event_count }}</dd>
              </div>
              <div class="kv-item">
                <dt>Fire count</dt>
                <dd class="tnum">{{ alert.fire_count }}</dd>
              </div>
              <div v-if="alert.source" class="kv-item">
                <dt>Source</dt>
                <dd>{{ alert.source }}</dd>
              </div>
              <div v-if="alert.channel" class="kv-item">
                <dt>Channel</dt>
                <dd>
                  <a class="link" @click="router.push(`/channels/${alert.channel_id}`)">
                    {{ alert.channel.name }}
                  </a>
                </dd>
              </div>
              <div v-if="alert.incident" class="kv-item">
                <dt>Incident</dt>
                <dd>
                  <a class="link" @click="router.push(`/incidents/${alert.incident_id}`)">
                    #{{ alert.incident_id }}
                  </a>
                </dd>
              </div>
              <div v-if="alert.generator_url" class="kv-item">
                <dt>Generator</dt>
                <dd>
                  <a class="link" :href="alert.generator_url" target="_blank">
                    Open ↗
                  </a>
                </dd>
              </div>
            </dl>
          </section>

          <section v-if="labelEntries.length" class="side-card">
            <div class="sre-label-eyebrow">Labels</div>
            <div class="side-labels">
              <div v-for="[k, v] in labelEntries" :key="k" class="side-label-row">
                <span class="slr-k">{{ k }}</span>
                <span class="slr-v">{{ v }}</span>
              </div>
            </div>
          </section>
        </aside>
      </div>
    </n-spin>

    <QuickSilenceModal
      v-model:show="showQuickSilence"
      :labels="alert?.labels ?? {}"
      :title="alert?.title"
    />
  </div>
</template>

<style scoped>
.alert-detail { max-width: 1400px; }

/* Header */
.detail-header { margin-bottom: 16px; }
.header-top {
  display: flex;
  align-items: center;
  gap: 8px;
}
.detail-title {
  font-family: var(--sre-font-sans);
  font-size: 22px;
  font-weight: 600;
  margin: 0;
  color: var(--sre-text-primary);
  line-height: 1.3;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.header-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.header-sub {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--sre-text-secondary);
  cursor: default;
}
.hsub-sev {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.6px;
}
.hsub-status { text-transform: capitalize; }
.hsub-key { display: inline-flex; align-items: center; gap: 6px; }
.hsub-key-label { color: var(--sre-text-tertiary); }
.hsub-key code {
  font-family: var(--sre-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 12px;
  background: var(--sre-bg-elevated);
  border-radius: 4px;
  padding: 2px 6px;
  color: var(--sre-text-secondary);
}

/* Layout */
.detail-layout {
  display: grid;
  grid-template-columns: 1fr 280px;
  gap: 16px;
  align-items: start;
}
.detail-main { min-width: 0; }

/* Overview sections */
.ov-section { margin-bottom: 24px; }
.ov-section .sre-label-eyebrow { margin-bottom: 8px; }

.ov-desc {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-hairline, var(--sre-border));
  border-radius: 8px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.annot-row {
  display: flex;
  gap: 8px;
  font-size: 13px;
  line-height: 1.5;
}
.annot-key {
  color: var(--sre-text-tertiary);
  font-weight: 500;
  min-width: 100px;
  flex-shrink: 0;
}
.annot-val { color: var(--sre-text-primary); }

.chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.label-chip {
  display: inline-flex;
  align-items: center;
  font-family: var(--sre-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 11px;
  background: var(--sre-bg-elevated);
  border: 1px solid var(--sre-hairline, var(--sre-border));
  border-radius: 4px;
  padding: 2px 6px;
}
.lc-k { color: var(--sre-text-tertiary); }
.lc-eq { color: var(--sre-text-tertiary); margin: 0 1px; }
.lc-v { color: var(--sre-text-primary); }

.summary-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 8px;
}
.sg-item {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-hairline, var(--sre-border));
  border-radius: 8px;
  padding: 10px 12px;
}
.sg-label {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.6px;
  margin-bottom: 4px;
}
.sg-value { font-size: 13px; color: var(--sre-text-primary); font-weight: 500; }

/* Events */
.event-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.event-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-hairline, var(--sre-border));
  border-radius: 6px;
  font-size: 12px;
}
.event-row[data-status="resolved"] { opacity: 0.7; }
.ev-time { color: var(--sre-text-secondary); min-width: 160px; }
.ev-value {
  font-family: var(--sre-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  color: var(--sre-text-primary);
  min-width: 80px;
}
.ev-status {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: 2px 8px;
  border-radius: 4px;
}
.ev-status[data-status="firing"] {
  color: var(--sre-critical);
  background: var(--sre-critical-soft);
}
.ev-status[data-status="resolved"] {
  color: var(--sre-success);
  background: var(--sre-success-soft);
}
.ev-fp {
  font-family: var(--sre-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 11px;
  color: var(--sre-text-tertiary);
  margin-left: auto;
}
.ev-empty {
  padding: 40px;
  text-align: center;
  color: var(--sre-text-tertiary);
  font-size: 13px;
}

/* Sidebar */
.detail-sidebar {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.side-card {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-hairline, var(--sre-border));
  border-radius: 10px;
  padding: 16px;
}
.side-card .sre-label-eyebrow { margin-bottom: 12px; }

.kv-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 0;
}
.kv-item {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  font-size: 12px;
  margin: 0;
}
.kv-item dt {
  color: var(--sre-text-tertiary);
  flex-shrink: 0;
}
.kv-item dd {
  margin: 0;
  color: var(--sre-text-primary);
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  justify-content: flex-end;
}
.link {
  color: var(--sre-primary);
  cursor: pointer;
  text-decoration: none;
}
.link:hover { text-decoration: underline; }

.side-labels {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.side-label-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 8px;
  background: var(--sre-bg-elevated);
  border-radius: 4px;
  font-family: var(--sre-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 11px;
}
.slr-k { color: var(--sre-text-tertiary); }
.slr-v { color: var(--sre-text-primary); word-break: break-all; }

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

@media (max-width: 1024px) {
  .detail-layout { grid-template-columns: 1fr; }
}
</style>
