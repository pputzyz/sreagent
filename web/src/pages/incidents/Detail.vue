<script setup lang="ts">
import { ref, shallowRef, onMounted, computed, h, inject } from 'vue'
import type { Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage, NIcon, NDataTable, NTag } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { incidentApi, alertV2Api } from '@/api'
import { changeEventApi } from '@/api/change-event'
import type { ChangeEvent } from '@/api/change-event'
import type { Incident, IncidentTimeline, AlertV2, PostMortem, DispatchLog } from '@/types'
import { getErrorMessage } from '@/utils/format'
import SnoozeModal from '@/components/incident/SnoozeModal.vue'
import MergeModal from '@/components/incident/MergeModal.vue'
import ReassignModal from '@/components/incident/ReassignModal.vue'
import { formatTime } from '@/utils/format'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import {
  ArrowBackOutline, SparklesOutline, VolumeOffOutline,
  TimeOutline, GitMergeOutline, PersonOutline,
  EllipsisHorizontal, RefreshOutline, ArrowUpCircleOutline,
  AlertCircleOutline, GitPullRequestOutline, PlayOutline,
} from '@vicons/ionicons5'
import QuickSilenceModal from '@/components/noise/QuickSilenceModal.vue'
import { MdEditor } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'

const { t, locale } = useI18n()
const message = useMessage()
const route = useRoute()
const router = useRouter()

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push('/incident')
}

const incidentId = computed(() => Number(route.params.id))
const incident = shallowRef<Incident | null>(null)
const timeline = shallowRef<IncidentTimeline[]>([])
const relatedAlerts = shallowRef<AlertV2[]>([])
const dispatchLogs = shallowRef<DispatchLog[]>([])
const loading = ref(false)
const activeTab = ref('overview')
const commentText = ref('')
const submittingComment = ref(false)

// Quick silence
const showQuickSilence = ref(false)

// Post-mortem
const postMortem = ref<PostMortem | null>(null)
const pmLoading = ref(false)
const pmSaving = ref(false)
const pmAiLoading = ref(false)

// Modal visibility (consumed by extracted components)
const showSnooze = ref(false)
const showMerge = ref(false)
const showReassign = ref(false)

// Related changes
const relatedChanges = shallowRef<ChangeEvent[]>([])
const changesLoading = ref(false)

function onSnoozeDone() { load() }
function onMergeDone(targetId: number) { router.push(`/oncall/incidents/${targetId}`) }
function onReassignDone() { load() }

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

function relTime(ts?: string): string {
  if (!ts) return '—'
  const diff = Date.now() - new Date(ts).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 1) return t('common.justNow')
  if (m < 60) return t('incident.relMAgo', { n: m })
  const h = Math.floor(m / 60)
  if (h < 24) return t('incident.relHAgo', { n: h })
  const d = Math.floor(h / 24)
  return t('incident.relDAgo', { n: d })
}

function durationStr(start?: string, end?: string): string {
  if (!start) return '—'
  const s = new Date(start).getTime()
  const e = end ? new Date(end).getTime() : Date.now()
  let sec = Math.max(0, Math.floor((e - s) / 1000))
  const h = Math.floor(sec / 3600); sec -= h * 3600
  const m = Math.floor(sec / 60); sec -= m * 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${sec}s`
  return `${sec}s`
}

async function load() {
  loading.value = true
  try {
    const [incRes, tlRes, alertRes, dlRes] = await Promise.all([
      incidentApi.get(incidentId.value),
      incidentApi.getTimeline(incidentId.value),
      alertV2Api.list({ incident_id: incidentId.value, page: 1, page_size: 50 }),
      incidentApi.getDispatchLogs(incidentId.value),
    ])
    incident.value = incRes.data.data ?? null
    timeline.value = tlRes.data.data ?? []
    relatedAlerts.value = alertRes.data.data?.list ?? []
    dispatchLogs.value = dlRes.data.data ?? []
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.loadFailed'))
  } finally { loading.value = false }
}

async function loadRelatedChanges() {
  changesLoading.value = true
  try {
    const res = await changeEventApi.list({ page: 1, page_size: 50 })
    relatedChanges.value = res.data.data?.list ?? []
  } catch {
    relatedChanges.value = []
  } finally { changesLoading.value = false }
}

async function doAction(action: 'acknowledge' | 'close' | 'reopen' | 'escalate') {
  try {
    await incidentApi[action](incidentId.value)
    message.success(t('common.success'))
    await load()
  } catch (e: unknown) { message.error(getErrorMessage(e) || t('common.failed')) }
}

async function submitComment() {
  if (!commentText.value.trim()) return
  submittingComment.value = true
  try {
    await incidentApi.addComment(incidentId.value, commentText.value)
    commentText.value = ''
    const tlRes = await incidentApi.getTimeline(incidentId.value)
    timeline.value = tlRes.data.data ?? []
  } catch (e: unknown) { message.error(getErrorMessage(e) || t('common.failed')) } finally { submittingComment.value = false }
}

async function loadPostMortem() {
  pmLoading.value = true
  try {
    const res = await incidentApi.getPostMortem(incidentId.value)
    postMortem.value = res.data.data ?? null
  } catch { postMortem.value = null } finally { pmLoading.value = false }
}

function initPostMortem() {
  postMortem.value = {
    id: 0,
    incident_id: incidentId.value,
    title: '',
    content: '',
    status: 'draft',
    published_at: null,
    created_at: '',
    updated_at: '',
  }
}

async function savePostMortem() {
  if (!postMortem.value) return
  pmSaving.value = true
  try {
    const res = await incidentApi.updatePostMortem(incidentId.value, {
      title: postMortem.value.title,
      content: postMortem.value.content,
    })
    postMortem.value = res.data.data
    message.success(t('common.savedSuccess'))
  } catch (e: unknown) { message.error(getErrorMessage(e) || t('common.saveFailed')) } finally { pmSaving.value = false }
}

async function publishPostMortem() {
  pmSaving.value = true
  try {
    const res = await incidentApi.publishPostMortem(incidentId.value)
    postMortem.value = res.data.data
    message.success(t('postMortem.published'))
  } catch (e: unknown) { message.error(getErrorMessage(e) || t('common.failed')) } finally { pmSaving.value = false }
}

async function aiGeneratePostMortem() {
  pmAiLoading.value = true
  try {
    const res = await incidentApi.aiGeneratePostMortem(incidentId.value)
    postMortem.value = res.data.data
    message.success(t('incident.aiDraftGenerated'))
  } catch (e: unknown) { message.error(getErrorMessage(e) || t('incident.aiGenerateFailed')) } finally { pmAiLoading.value = false }
}

const moreActionOptions = computed(() => {
  const opts = [
    { label: t('incident.escalate'), key: 'escalate', icon: () => h(NIcon, { component: ArrowUpCircleOutline }) },
    { label: t('incident.reassign'), key: 'reassign', icon: () => h(NIcon, { component: PersonOutline }) },
    { label: t('incident.mergeIncident'), key: 'merge', icon: () => h(NIcon, { component: GitMergeOutline }) },
    { label: t('incident.quickSilence'), key: 'silence', icon: () => h(NIcon, { component: VolumeOffOutline }) },
  ]
  if (incident.value && incident.value.status !== 'closed') {
    opts.splice(1, 0, { label: t('incident.snooze'), key: 'snooze', icon: () => h(NIcon, { component: TimeOutline }) })
  }
  return opts
})

function handleMoreAction(key: string) {
  switch (key) {
    case 'escalate': doAction('escalate'); break
    case 'snooze': showSnooze.value = true; break
    case 'reassign': showReassign.value = true; break
    case 'merge': showMerge.value = true; break
    case 'silence': showQuickSilence.value = true; break
  }
}

// Dispatch log table columns
const dispatchLogColumns = [
  { title: 'ID', key: 'id', width: 60 },
  {
    title: t('incident.dispatchStatus'),
    key: 'status',
    width: 80,
    render: (row: DispatchLog) => {
      const typeMap: Record<string, 'success' | 'warning' | 'default' | 'error'> = { sent: 'success', pending: 'warning', skipped: 'default', failed: 'error' }
      return h(NTag, { type: typeMap[row.status] || 'default', size: 'small', bordered: false }, () => row.status)
    },
  },
  { title: t('incident.dispatchPolicy'), key: 'dispatch_policy_id', width: 80 },
  { title: t('incident.dispatchAttempt'), key: 'attempt', width: 70 },
  {
    title: t('incident.dispatchNote'),
    key: 'note',
    ellipsis: { tooltip: true },
    render: (row: DispatchLog) => row.note || '—',
  },
  {
    title: t('incident.dispatchTime'),
    key: 'created_at',
    width: 160,
    render: (row: DispatchLog) => formatTime(row.created_at),
  },
]

// B4/B5: MdEditor computed properties from i18n locale + injected theme
const isDark = inject<Ref<boolean>>('isDark', ref(false))
const mdLanguage = computed(() => locale.value === 'zh-CN' ? 'zh-CN' : 'en-US')
const mdTheme = computed(() => isDark.value ? 'dark' : 'light')

// B8/B9: Map timeline entry.action to i18n labels
const ACTION_LABEL_MAP: Record<string, string> = {
  acknowledge: 'incident.acknowledge',
  close: 'incident.close',
  reopen: 'incident.reopen',
  assign: 'incident.assignTo',
  escalate: 'incident.escalate',
  resolve: 'alert.resolve',
  comment: 'incident.comment',
  created: 'alert.created',
  notified: 'alert.notified',
}

// Timeline dot color by action type
const ACTION_COLOR_MAP: Record<string, string> = {
  acknowledge: 'ack',   // blue
  resolve: 'resolve',   // green
  escalate: 'escalate', // orange
  comment: 'comment',   // gray
  close: 'resolve',
  reopen: 'escalate',
  assign: 'ack',
  created: 'default',
  notified: 'default',
}
function timelineActionColor(action: string): string {
  return ACTION_COLOR_MAP[action] || 'default'
}
function actionLabel(action: string): string {
  return t(ACTION_LABEL_MAP[action] ?? action)
}

onMounted(async () => {
  await load()
  await loadPostMortem()
  await loadRelatedChanges()
})
</script>

<template>
  <div class="incident-detail">
    <!-- Header -->
    <header class="detail-header">
      <div class="header-top">
        <div class="header-left">
          <n-button quaternary circle size="small" @click="goBack">
            <template #icon><n-icon :component="ArrowBackOutline" /></template>
          </n-button>
          <span v-if="incident" class="incident-id tnum">#{{ incident.id }}</span>
          <h1 class="incident-title">{{ incident?.title ?? t('incident.title') }}</h1>
        </div>
        <div class="header-right">
          <n-button quaternary circle size="small" :loading="loading" @click="load">
            <template #icon><n-icon :component="RefreshOutline" /></template>
          </n-button>
          <n-dropdown
            v-if="incident"
            trigger="click"
            :options="moreActionOptions"
            @select="handleMoreAction"
          >
            <n-button quaternary size="small">
              <template #icon><n-icon :component="EllipsisHorizontal" /></template>
              {{ t('common.actions') }}
            </n-button>
          </n-dropdown>
        </div>
      </div>

      <div
        v-if="incident"
        class="header-stripe sre-row-card"
        :data-severity="incident.severity"
      >
        <span class="sre-dot" :data-severity="incident.severity" />
        <span class="stripe-text">{{ t(severityLabel[incident.severity] ?? incident.severity) }}</span>
        <span class="sre-meta-divider" />
        <span class="stripe-text">{{ t(statusLabel[incident.status] ?? incident.status) }}</span>
        <span class="sre-meta-divider" />
        <span class="stripe-text">{{ incident.channel?.name ?? '—' }}</span>
        <span class="sre-meta-divider" />
        <span class="stripe-text tnum">{{ durationStr(incident.triggered_at, incident.closed_at) }}</span>
      </div>
    </header>

    <LoadingSkeleton v-if="loading && !incident" :rows="8" variant="row" />
    <n-spin v-else :show="loading">
      <div v-if="incident" class="detail-layout sre-fadein">
        <!-- LEFT MAIN -->
        <div class="detail-main">
          <!-- Action bar -->
          <div class="action-bar">
            <n-button
              v-if="incident.status === 'triggered'"
              type="primary" size="small" @click="doAction('acknowledge')"
            >{{ t('incident.acknowledge') }}</n-button>

            <n-button
              v-if="incident.status !== 'closed'"
              size="small" @click="doAction('close')"
            >{{ t('incident.close') }}</n-button>

            <n-button
              v-if="incident.status === 'closed'"
              size="small" @click="doAction('reopen')"
            >{{ t('incident.reopen') }}</n-button>

            <n-button
              v-if="incident.status !== 'closed'"
              size="small" tertiary @click="showSnooze = true"
            >
              <template #icon><n-icon :component="TimeOutline" /></template>
              {{ t('incident.snooze') }}
            </n-button>

            <n-button size="small" tertiary @click="router.push({ path: '/platform/diagnostic-workflows', query: { incident_id: String(incident.id) } })">
              <template #icon><n-icon :component="PlayOutline" /></template>
              {{ t('incident.startDiagnosis') }}
            </n-button>

            <n-button size="small" tertiary @click="showQuickSilence = true">
              <template #icon><n-icon :component="VolumeOffOutline" /></template>
              {{ t('incident.quickSilence') }}
            </n-button>
          </div>

          <!-- Tabs -->
          <n-tabs v-model:value="activeTab" type="line" animated class="detail-tabs">

            <!-- Overview -->
            <n-tab-pane name="overview" :tab="t('common.overview')">
              <div class="overview-grid">
                <div class="ov-row">
                  <div class="sre-label-eyebrow">{{ t('incident.triggeredAt') }}</div>
                  <div class="ov-value tnum">{{ formatTime(incident.triggered_at) }}</div>
                </div>
                <div class="ov-row">
                  <div class="sre-label-eyebrow">{{ t('incident.acknowledgedAt') }}</div>
                  <div class="ov-value tnum">{{ incident.acknowledged_at ? formatTime(incident.acknowledged_at) : '—' }}</div>
                </div>
                <div class="ov-row">
                  <div class="sre-label-eyebrow">{{ t('incident.resolvedAt') }}</div>
                  <div class="ov-value tnum">{{ incident.resolved_at ? formatTime(incident.resolved_at) : '—' }}</div>
                </div>
                <div class="ov-row">
                  <div class="sre-label-eyebrow">{{ t('incident.closedAt') }}</div>
                  <div class="ov-value tnum">{{ incident.closed_at ? formatTime(incident.closed_at) : '—' }}</div>
                </div>
                <div class="ov-row">
                  <div class="sre-label-eyebrow">{{ t('incident.alertCount') }}</div>
                  <div class="ov-value tnum">{{ incident.alert_count }}</div>
                </div>
                <div class="ov-row">
                  <div class="sre-label-eyebrow">{{ t('incident.assignee') }}</div>
                  <div class="ov-value">
                    {{ incident.assigned_user?.display_name ?? incident.assigned_user?.username ?? '—' }}
                  </div>
                </div>
              </div>

              <div v-if="incident.description" class="ov-description">
                {{ incident.description }}
              </div>

              <div v-if="incident.labels && Object.keys(incident.labels).length" class="ov-labels">
                <div class="sre-label-eyebrow labels-heading">{{ t('incident.labels') }}</div>
                <div class="label-chips">
                  <span v-for="(v, k) in incident.labels" :key="k" class="label-chip">
                    {{ k }}={{ v }}
                  </span>
                </div>
              </div>
            </n-tab-pane>

            <!-- Related alerts -->
            <n-tab-pane name="alerts" :tab="t('alertV2.title')">
              <EmptyState
                v-if="!relatedAlerts.length"
                :icon="AlertCircleOutline"
                :title="t('common.noData')"
                :description="t('incident.noEvents')"
                size="sm"
              />
              <div v-else class="alert-rows">
                <div
                  v-for="a in relatedAlerts" :key="a.id"
                  class="sre-row-card alert-row"
                  :data-severity="a.severity"
                  @click="router.push(`/alert/events/${a.id}`)"
                >
                  <span class="sre-dot" :data-severity="a.severity" />
                  <span class="alert-title">{{ a.title }}</span>
                  <span class="sre-meta-divider" />
                  <span class="alert-meta tnum">{{ relTime(a.last_fired_at) }}</span>
                  <span class="sre-meta-divider" />
                  <span class="alert-meta tnum">{{ t('incident.nxFired', { n: a.fire_count }) }}</span>
                  <span class="alert-status" :class="`s-${a.status}`">{{ a.status }}</span>
                </div>
              </div>
            </n-tab-pane>

            <!-- Timeline -->
            <n-tab-pane name="timeline" :tab="t('incident.timeline')">
              <EmptyState
                v-if="!timeline.length"
                :icon="TimeOutline"
                :title="t('incident.noTimeline')"
                :description="t('incident.noEvents')"
                size="sm"
              />
              <ol v-else class="tl-list">
                <li v-for="entry in timeline" :key="entry.id" class="tl-item">
                  <span class="tl-dot" :data-action="timelineActionColor(entry.action)" />
                  <div class="tl-body">
                    <div class="tl-line">
                      <span class="tl-action">{{ actionLabel(entry.action) }}</span>
                      <span class="tl-time tnum">{{ relTime(entry.created_at) }}</span>
                    </div>
                    <div v-if="entry.actor || entry.content" class="tl-sub">
                      <span v-if="entry.actor">{{ t('incident.by', { name: entry.actor.display_name ?? entry.actor.username }) }}</span>
                      <span v-if="entry.actor && entry.content" class="sre-meta-divider" />
                      <span v-if="entry.content">{{ entry.content }}</span>
                    </div>
                  </div>
                </li>
              </ol>

              <div class="comment-box">
                <textarea
                  v-model="commentText"
                  class="comment-input"
                  rows="3"
                  :placeholder="t('incident.commentPlaceholder')"
                />
                <div class="comment-actions">
                  <n-button
                    type="primary" size="small"
                    :loading="submittingComment"
                    :disabled="!commentText.trim()"
                    @click="submitComment"
                  >{{ t('incident.addComment') }}</n-button>
                </div>
              </div>
            </n-tab-pane>

            <!-- Post-mortem -->
            <n-tab-pane name="postmortem" :tab="t('postMortem.tab')">
              <n-spin :show="pmLoading">
                <div v-if="postMortem" class="pm-container">
                  <div class="pm-toolbar">
                    <div class="pm-meta">
                      <n-tag :type="postMortem.status === 'published' ? 'success' : 'default'" size="small" :bordered="false">
                        {{ postMortem.status === 'published' ? t('postMortem.published') : t('postMortem.draft') }}
                      </n-tag>
                      <span v-if="postMortem.updated_at" class="pm-updated tnum">
                        {{ t('postMortem.lastUpdated') }} · {{ formatTime(postMortem.updated_at) }}
                      </span>
                    </div>
                    <n-space size="small">
                      <n-button size="small" tertiary :loading="pmAiLoading" @click="aiGeneratePostMortem">
                        <template #icon><n-icon :component="SparklesOutline" /></template>
                        {{ pmAiLoading ? t('postMortem.generating') : t('postMortem.aiGenerate') }}
                      </n-button>
                      <n-button size="small" type="primary" :loading="pmSaving" @click="savePostMortem">
                        {{ t('common.save') }}
                      </n-button>
                      <n-popconfirm
                        v-if="postMortem.status !== 'published'"
                        @positive-click="publishPostMortem"
                      >
                        <template #trigger>
                          <n-button size="small" type="success" :loading="pmSaving">
                            {{ t('postMortem.publish') }}
                          </n-button>
                        </template>
                        {{ t('postMortem.publishConfirm') }}
                      </n-popconfirm>
                    </n-space>
                  </div>

                  <input
                    v-model="postMortem.title"
                    class="pm-title-input"
                    type="text"
                    :placeholder="t('postMortem.titlePlaceholder')"
                  />

                  <MdEditor
                    v-model="postMortem.content"
                    :preview="true"
                    :toolbars-exclude="['github']"
                    :language="mdLanguage"
                    :theme="mdTheme"
                    class="pm-editor"
                  />
                </div>
                <div v-else class="empty-state pm-empty">
                  <p class="pm-empty-text">{{ t('postMortem.noPostMortem') }}</p>
                  <n-button type="primary" size="small" @click="initPostMortem">
                    {{ t('common.create') }}
                  </n-button>
                </div>
              </n-spin>
            </n-tab-pane>

            <!-- Dispatch Logs -->
            <n-tab-pane name="dispatch-logs" :tab="t('incident.dispatchLogs')">
              <EmptyState
                v-if="!dispatchLogs.length"
                :icon="TimeOutline"
                :title="t('incident.dispatchLogEmpty')"
                size="sm"
              />
              <n-data-table
                v-else
                :columns="dispatchLogColumns"
                :data="dispatchLogs"
                :bordered="false"
                size="small"
                striped
              />
            </n-tab-pane>

            <!-- Related Changes -->
            <n-tab-pane name="changes" :tab="t('incident.relatedChanges')">
              <n-spin :show="changesLoading">
                <EmptyState
                  v-if="!changesLoading && relatedChanges.length === 0"
                  :icon="GitPullRequestOutline"
                  :title="t('incident.noRelatedChanges')"
                  size="sm"
                />
                <div v-else class="alert-rows">
                  <div
                    v-for="ev in relatedChanges" :key="ev.id"
                    class="sre-row-card alert-row"
                  >
                    <span class="sre-dot" data-severity="info" />
                    <span class="alert-title">{{ ev.description || ev.service }}</span>
                    <span class="sre-meta-divider" />
                    <n-tag size="small" :bordered="false">{{ ev.source }}</n-tag>
                    <span class="sre-meta-divider" />
                    <n-tag size="small" :bordered="false">{{ ev.change_type }}</n-tag>
                    <span class="sre-meta-divider" />
                    <span class="alert-meta tnum">{{ ev.author }}</span>
                    <span class="sre-meta-divider" />
                    <span class="alert-meta tnum">{{ formatTime(ev.timestamp || ev.created_at) }}</span>
                  </div>
                </div>
              </n-spin>
            </n-tab-pane>

          </n-tabs>
        </div>

        <!-- RIGHT SIDEBAR -->
        <aside class="detail-sidebar">
          <section class="side-card">
            <div class="sre-label-eyebrow card-eyebrow">{{ t('incident.keyInfo') }}</div>
            <dl class="kv-list">
              <div class="kv-row">
                <dt>{{ t('incident.channel') }}</dt>
                <dd>
                  <a v-if="incident.channel" class="kv-link" @click="router.push(`/oncall/spaces/${incident.channel_id}`)">
                    {{ incident.channel.name }}
                  </a>
                  <span v-else>—</span>
                </dd>
              </div>
              <div class="kv-row">
                <dt>{{ t('incident.severity') }}</dt>
                <dd class="kv-flex">
                  <span class="sre-dot" :data-severity="incident.severity" />
                  {{ t(severityLabel[incident.severity] ?? incident.severity) }}
                </dd>
              </div>
              <div class="kv-row">
                <dt>{{ t('common.status') }}</dt>
                <dd>{{ t(statusLabel[incident.status] ?? incident.status) }}</dd>
              </div>
              <div class="kv-row">
                <dt>{{ t('incident.triggeredAt') }}</dt>
                <dd class="tnum">{{ formatTime(incident.triggered_at) }}</dd>
              </div>
              <div class="kv-row">
                <dt>{{ t('incident.acknowledgedAt') }}</dt>
                <dd class="tnum">{{ incident.acknowledged_at ? formatTime(incident.acknowledged_at) : '—' }}</dd>
              </div>
              <div class="kv-row">
                <dt>{{ t('incident.assignee') }}</dt>
                <dd>{{ incident.assigned_user?.display_name ?? incident.assigned_user?.username ?? '—' }}</dd>
              </div>
              <div class="kv-row">
                <dt>{{ t('incident.alertCount') }}</dt>
                <dd class="tnum">{{ incident.alert_count }}</dd>
              </div>
              <div class="kv-row">
                <dt>{{ t('incident.duration') }}</dt>
                <dd class="tnum">{{ durationStr(incident.triggered_at, incident.closed_at) }}</dd>
              </div>
            </dl>
          </section>

          <section class="side-card">
            <div class="sre-label-eyebrow card-eyebrow">{{ t('incident.timelineBrief') }}</div>
            <ol v-if="timeline.length" class="brief-list">
              <li v-for="e in timeline.slice(0, 5)" :key="e.id" class="brief-item">
                <span class="brief-dot" :data-action="timelineActionColor(e.action)" />
                <div class="brief-body">
                  <div class="brief-action">{{ actionLabel(e.action) }}</div>
                  <div class="brief-meta tnum">{{ relTime(e.created_at) }}</div>
                </div>
              </li>
            </ol>
            <p v-else class="brief-empty">{{ t('incident.noEvents') }}</p>
          </section>
        </aside>
      </div>
    </n-spin>

    <!-- Quick Silence Modal -->
    <QuickSilenceModal
      v-model:show="showQuickSilence"
      :labels="incident?.labels ?? {}"
      :title="incident?.title"
      @created="load"
    />

    <!-- Extracted Modals -->
    <SnoozeModal
      v-model:show="showSnooze"
      :incident-id="incidentId"
      @done="onSnoozeDone"
    />

    <MergeModal
      v-model:show="showMerge"
      :incident-id="incidentId"
      @done="onMergeDone"
    />

    <ReassignModal
      v-model:show="showReassign"
      :incident-id="incidentId"
      @done="onReassignDone"
    />
  </div>
</template>

<style scoped>
.incident-detail {
  max-width: 1400px;
  font-family: var(--sre-font-sans);
}

/* Header */
.detail-header {
  margin-bottom: 16px;
}
.header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 6px;
}
.incident-id {
  font-size: 13px;
  color: var(--sre-text-tertiary);
  font-weight: 500;
  font-family: var(--sre-font-mono);
}
.incident-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--sre-text-primary);
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.header-stripe {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: var(--sre-radius-sm);
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  font-size: 12px;
}
.stripe-text {
  color: var(--sre-text-secondary);
  font-weight: 500;
}

/* Layout */
.detail-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  gap: 16px;
  align-items: start;
}
.detail-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Action bar */
.action-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 14px;
  border: var(--sre-hairline);
  background: var(--sre-bg-card);
  border-radius: var(--sre-radius-md);
}

/* Tabs */
.detail-tabs {
  border: var(--sre-hairline);
  background: var(--sre-bg-card);
  border-radius: var(--sre-radius-md);
  padding: 4px 16px 16px;
}
.detail-tabs :deep(.n-tabs-tab) {
  text-transform: uppercase;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
}

/* Overview */
.overview-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px 24px;
  padding: 8px 0 4px;
}
.ov-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.ov-value {
  font-size: 13px;
  color: var(--sre-text-primary);
  line-height: 1.5;
}
.ov-description {
  margin-top: 18px;
  padding-top: 14px;
  border-top: var(--sre-hairline);
  font-size: 13px;
  line-height: 1.6;
  color: var(--sre-text-secondary);
}
.ov-labels {
  margin-top: 18px;
  padding-top: 14px;
  border-top: var(--sre-hairline);
}
.labels-heading {
  margin-bottom: 8px;
  color: var(--sre-text-tertiary);
}
.label-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.label-chip {
  font-family: var(--sre-font-mono);
  font-size: 11px;
  padding: 2px 6px;
  background: var(--sre-bg-elevated);
  border-radius: 4px;
  color: var(--sre-text-secondary);
}

/* Alerts list */
.alert-rows {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-top: 4px;
}
.alert-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-sm);
  cursor: pointer;
  transition: background 120ms ease;
}
.alert-row:hover { background: var(--sre-bg-hover); }
.alert-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-text-primary);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.alert-meta { font-size: 12px; color: var(--sre-text-secondary); }
.alert-status {
  font-size: 11px;
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 0.05em;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--sre-bg-elevated);
}
.alert-status.s-firing { color: var(--sre-critical); background: var(--sre-critical-soft); }
.alert-status.s-resolved { color: var(--sre-success); background: var(--sre-success-soft); }

/* Timeline */
.tl-list {
  list-style: none;
  margin: 0;
  padding: 8px 0 0;
  position: relative;
}
.tl-list::before {
  content: '';
  position: absolute;
  left: 5px;
  top: 14px;
  bottom: 14px;
  width: 1px;
  background: var(--sre-border);
}
.tl-item {
  position: relative;
  display: flex;
  gap: 14px;
  padding: 8px 0;
}
.tl-dot {
  position: relative;
  z-index: 1;
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: var(--sre-primary);
  margin-top: 4px;
  flex-shrink: 0;
  box-shadow: 0 0 0 3px var(--sre-bg-card);
}
/* Timeline dot colors by action type */
.tl-dot[data-action="ack"],
.brief-dot[data-action="ack"]      { background: var(--sre-info, #3b82f6); }
.tl-dot[data-action="resolve"],
.brief-dot[data-action="resolve"]  { background: var(--sre-success, #22c55e); }
.tl-dot[data-action="escalate"],
.brief-dot[data-action="escalate"] { background: var(--sre-warning, #f59e0b); }
.tl-dot[data-action="comment"],
.brief-dot[data-action="comment"]  { background: var(--sre-text-tertiary, #94a3b8); }
.tl-body { flex: 1; min-width: 0; }
.tl-line {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}
.tl-action {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
}
.tl-time {
  font-size: 11px;
  color: var(--sre-text-tertiary);
}
.tl-sub {
  margin-top: 3px;
  font-size: 12px;
  color: var(--sre-text-secondary);
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.comment-box {
  margin-top: 16px;
  padding-top: 14px;
  border-top: var(--sre-hairline);
}
.comment-input {
  width: 100%;
  background: var(--sre-bg-elevated);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-sm);
  padding: 10px 12px;
  color: var(--sre-text-primary);
  font-family: inherit;
  font-size: 13px;
  line-height: 1.5;
  resize: vertical;
  outline: none;
  transition: border-color 120ms ease;
}
.comment-input:focus {
  border-color: var(--sre-primary);
}
.comment-input:focus-visible {
  outline: 2px solid var(--sre-primary);
  outline-offset: 1px;
}
.comment-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

/* Post-mortem */
.pm-container { display: flex; flex-direction: column; gap: 12px; padding-top: 6px; }
.pm-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 12px;
  border-bottom: var(--sre-hairline);
  gap: 12px;
}
.pm-meta { display: flex; align-items: center; gap: 10px; }
.pm-updated { font-size: 11px; color: var(--sre-text-tertiary); }
.pm-title-input {
  width: 100%;
  background: transparent;
  border: none;
  outline: none;
  font-family: inherit;
  font-size: 18px;
  font-weight: 600;
  color: var(--sre-text-primary);
  padding: 4px 0;
  border-bottom: var(--sre-hairline);
}
.pm-title-input:focus {
  border-bottom-color: var(--sre-primary);
}
.pm-title-input:focus-visible {
  outline: 2px solid var(--sre-primary);
  outline-offset: 2px;
}
.pm-editor {
  height: 520px;
  border-radius: 6px;
}
.pm-empty { padding: 60px 0; text-align: center; }
.pm-empty-text { color: var(--sre-text-secondary); margin-bottom: 16px; font-size: 13px; }

/* Empty state */
.empty-state { padding: 32px 0; text-align: center; }

/* Sidebar */
.detail-sidebar {
  display: flex;
  flex-direction: column;
  gap: 16px;
  position: sticky;
  top: 16px;
}
.side-card {
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
  padding: 16px;
}
.card-eyebrow { margin-bottom: 12px; color: var(--sre-text-tertiary); }
.kv-list { margin: 0; display: flex; flex-direction: column; gap: 10px; }
.kv-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.kv-row dt {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--sre-text-tertiary);
  margin: 0;
}
.kv-row dd {
  margin: 0;
  font-size: 13px;
  color: var(--sre-text-primary);
  text-align: right;
}
.kv-flex { display: inline-flex; align-items: center; gap: 6px; }
.kv-link { color: var(--sre-primary); cursor: pointer; }
.kv-link:hover { text-decoration: underline; }

.brief-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 10px; }
.brief-item { display: flex; gap: 10px; align-items: flex-start; }
.brief-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: var(--sre-primary);
  margin-top: 7px;
  flex-shrink: 0;
}
.brief-body { flex: 1; min-width: 0; }
.brief-action {
  font-size: 12px;
  color: var(--sre-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.brief-meta { font-size: 11px; color: var(--sre-text-tertiary); margin-top: 2px; }
.brief-empty { font-size: 12px; color: var(--sre-text-tertiary); margin: 0; }

/* Fade in */
.sre-fadein {
  animation: sre-fadein 200ms ease-out;
}
@keyframes sre-fadein {
  from { opacity: 0; }
  to { opacity: 1; }
}

@media (max-width: 980px) {
  .detail-layout { grid-template-columns: 1fr; }
  .detail-sidebar { position: static; }
  .overview-grid { grid-template-columns: 1fr; }
}
</style>
