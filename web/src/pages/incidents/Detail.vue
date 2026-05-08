<script setup lang="ts">
import { ref, shallowRef, onMounted, computed, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { incidentApi, alertV2Api, userApi } from '@/api'
import type { Incident, IncidentTimeline, AlertV2, User } from '@/types'
import { formatTime } from '@/utils/format'
import {
  ArrowBackOutline, SparklesOutline, VolumeOffOutline,
  TimeOutline, GitMergeOutline, PersonOutline,
  EllipsisHorizontal, RefreshOutline, ArrowUpCircleOutline,
} from '@vicons/ionicons5'
import QuickSilenceModal from '@/components/noise/QuickSilenceModal.vue'
import { MdEditor } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'

const { t } = useI18n()
const message = useMessage()
const route = useRoute()
const router = useRouter()

const incidentId = computed(() => Number(route.params.id))
const incident = shallowRef<Incident | null>(null)
const timeline = shallowRef<IncidentTimeline[]>([])
const relatedAlerts = shallowRef<AlertV2[]>([])
const loading = ref(false)
const activeTab = ref('overview')
const commentText = ref('')
const submittingComment = ref(false)

// Quick silence
const showQuickSilence = ref(false)

// Post-mortem
const postMortem = ref<any | null>(null)
const pmLoading = ref(false)
const pmSaving = ref(false)
const pmAiLoading = ref(false)

// Snooze
const showSnooze = ref(false)
const snoozeLoading = ref(false)
const snoozeDuration = ref<number | null>(null)
const snoozeCustomUntil = ref('')
const snoozePresets = computed(() => [
  { label: '15m', minutes: 15 },
  { label: '30m', minutes: 30 },
  { label: '1h', minutes: 60 },
  { label: '2h', minutes: 120 },
  { label: '4h', minutes: 240 },
  { label: t('query.timeCustom'), minutes: -1 },
])

async function doSnooze() {
  let until: string
  if (snoozeDuration.value === -1) {
    if (!snoozeCustomUntil.value) { message.warning(t('incident.selectSnoozeEnd')); return }
    until = new Date(snoozeCustomUntil.value).toISOString()
  } else if (snoozeDuration.value) {
    const d = new Date()
    d.setMinutes(d.getMinutes() + snoozeDuration.value)
    until = d.toISOString()
  } else {
    message.warning(t('incident.selectSnoozeDuration')); return
  }
  snoozeLoading.value = true
  try {
    await incidentApi.snooze(incidentId.value, until)
    message.success(t('incident.snoozeSuccess'))
    showSnooze.value = false
    snoozeDuration.value = null
    snoozeCustomUntil.value = ''
    await load()
  } catch (e: any) { message.error(e?.message ?? t('incident.opFailed')) } finally { snoozeLoading.value = false }
}

// Merge
const showMerge = ref(false)
const mergeLoading = ref(false)
const mergeSearch = ref('')
const mergeSearchLoading = ref(false)
const mergeResults = ref<Incident[]>([])
const mergeTargetId = ref<number | null>(null)

async function searchMergeIncidents() {
  if (!mergeSearch.value.trim()) return
  mergeSearchLoading.value = true
  try {
    const res = await incidentApi.list({ query: mergeSearch.value, page: 1, page_size: 10 })
    mergeResults.value = (res.data.data?.list ?? []).filter((i: Incident) => i.id !== incidentId.value)
  } catch (e: any) { message.error(e?.message ?? t('incident.searchFailed')) } finally { mergeSearchLoading.value = false }
}

async function doMerge() {
  if (!mergeTargetId.value) { message.warning(t('incident.selectTargetIncident')); return }
  mergeLoading.value = true
  try {
    await incidentApi.merge(incidentId.value, mergeTargetId.value)
    message.success(t('incident.mergeSuccess'))
    showMerge.value = false
    router.push(`/incidents/${mergeTargetId.value}`)
  } catch (e: any) { message.error(e?.message ?? t('incident.opFailed')) } finally { mergeLoading.value = false }
}

// Reassign
const showReassign = ref(false)
const reassignLoading = ref(false)
const reassignSearch = ref('')
const reassignSearchLoading = ref(false)
const reassignUsers = ref<User[]>([])
const reassignUserId = ref<number | null>(null)

async function searchUsers() {
  reassignSearchLoading.value = true
  try {
    const res = await userApi.list({ page: 1, page_size: 50 })
    const allUsers: User[] = res.data.data?.list ?? []
    const q = reassignSearch.value.toLowerCase()
    reassignUsers.value = q
      ? allUsers.filter(u =>
          (u.username?.toLowerCase().includes(q)) ||
          (u.display_name?.toLowerCase().includes(q)))
      : allUsers
  } catch (e: any) { message.error(e?.message ?? t('incident.searchFailed')) } finally { reassignSearchLoading.value = false }
}

async function doReassign() {
  if (!reassignUserId.value) { message.warning(t('incident.selectAssignee')); return }
  reassignLoading.value = true
  try {
    await incidentApi.reassign(incidentId.value, reassignUserId.value)
    message.success(t('incident.reassignSuccess'))
    showReassign.value = false
    reassignUserId.value = null
    await load()
  } catch (e: any) { message.error(e?.message ?? t('incident.opFailed')) } finally { reassignLoading.value = false }
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

function relTime(ts?: string): string {
  if (!ts) return '—'
  const diff = Date.now() - new Date(ts).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  const d = Math.floor(h / 24)
  return `${d}d ago`
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
  } finally { loading.value = false }
}

async function doAction(action: 'acknowledge' | 'close' | 'reopen' | 'escalate') {
  try {
    await incidentApi[action](incidentId.value)
    message.success(t('common.success'))
    await load()
  } catch (e: any) { message.error(e?.message ?? t('common.failed')) }
}

async function submitComment() {
  if (!commentText.value.trim()) return
  submittingComment.value = true
  try {
    await incidentApi.addComment(incidentId.value, commentText.value)
    commentText.value = ''
    const tlRes = await incidentApi.getTimeline(incidentId.value)
    timeline.value = tlRes.data.data ?? []
  } catch (e: any) { message.error(e?.message ?? t('common.failed')) } finally { submittingComment.value = false }
}

async function loadPostMortem() {
  pmLoading.value = true
  try {
    const res = await incidentApi.getPostMortem(incidentId.value)
    postMortem.value = res.data.data ?? null
  } catch { postMortem.value = null } finally { pmLoading.value = false }
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
  } catch (e: any) { message.error(e?.message ?? t('common.saveFailed')) } finally { pmSaving.value = false }
}

async function publishPostMortem() {
  pmSaving.value = true
  try {
    const res = await incidentApi.publishPostMortem(incidentId.value)
    postMortem.value = res.data.data
    message.success(t('postMortem.published'))
  } catch (e: any) { message.error(e?.message ?? t('common.failed')) } finally { pmSaving.value = false }
}

async function aiGeneratePostMortem() {
  pmAiLoading.value = true
  try {
    const res = await incidentApi.aiGeneratePostMortem(incidentId.value)
    postMortem.value = res.data.data
    message.success(t('incident.aiDraftGenerated'))
  } catch (e: any) { message.error(e?.message ?? t('incident.aiGenerateFailed')) } finally { pmAiLoading.value = false }
}

const moreActionOptions = computed(() => {
  const opts: any[] = [
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

onMounted(async () => {
  await load()
  await loadPostMortem()
  await searchUsers()
})
</script>

<template>
  <div class="incident-detail">
    <!-- Header -->
    <header class="detail-header">
      <div class="header-top">
        <div class="header-left">
          <n-button quaternary circle size="small" @click="router.back()">
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

    <n-spin :show="loading">
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

            <n-button size="small" tertiary @click="showQuickSilence = true">
              <template #icon><n-icon :component="VolumeOffOutline" /></template>
              {{ t('incident.quickSilence') }}
            </n-button>
          </div>

          <!-- Tabs -->
          <n-tabs v-model:value="activeTab" type="line" animated class="detail-tabs">

            <!-- Overview -->
            <n-tab-pane name="overview" tab="Overview">
              <div class="overview-grid">
                <div class="ov-row">
                  <div class="sre-label-eyebrow">Triggered</div>
                  <div class="ov-value tnum">{{ formatTime(incident.triggered_at) }}</div>
                </div>
                <div class="ov-row">
                  <div class="sre-label-eyebrow">Acknowledged</div>
                  <div class="ov-value tnum">{{ incident.acknowledged_at ? formatTime(incident.acknowledged_at) : '—' }}</div>
                </div>
                <div class="ov-row">
                  <div class="sre-label-eyebrow">Resolved</div>
                  <div class="ov-value tnum">{{ incident.resolved_at ? formatTime(incident.resolved_at) : '—' }}</div>
                </div>
                <div class="ov-row">
                  <div class="sre-label-eyebrow">Closed</div>
                  <div class="ov-value tnum">{{ incident.closed_at ? formatTime(incident.closed_at) : '—' }}</div>
                </div>
                <div class="ov-row">
                  <div class="sre-label-eyebrow">Alert count</div>
                  <div class="ov-value tnum">{{ incident.alert_count }}</div>
                </div>
                <div class="ov-row">
                  <div class="sre-label-eyebrow">Assignee</div>
                  <div class="ov-value">
                    {{ incident.assigned_user?.display_name ?? incident.assigned_user?.username ?? '—' }}
                  </div>
                </div>
              </div>

              <div v-if="incident.description" class="ov-description">
                {{ incident.description }}
              </div>

              <div v-if="incident.labels && Object.keys(incident.labels).length" class="ov-labels">
                <div class="sre-label-eyebrow" style="margin-bottom:8px">Labels</div>
                <div class="label-chips">
                  <span v-for="(v, k) in incident.labels" :key="k" class="label-chip">
                    {{ k }}={{ v }}
                  </span>
                </div>
              </div>
            </n-tab-pane>

            <!-- Related alerts -->
            <n-tab-pane name="alerts" :tab="t('alertV2.title')">
              <div v-if="!relatedAlerts.length" class="empty-state">
                <n-empty :description="t('common.noData')" />
              </div>
              <div v-else class="alert-rows">
                <div
                  v-for="a in relatedAlerts" :key="a.id"
                  class="sre-row-card alert-row"
                  :data-severity="a.severity"
                  @click="router.push(`/alerts/${a.id}`)"
                >
                  <span class="sre-dot" :data-severity="a.severity" />
                  <span class="alert-title">{{ a.title }}</span>
                  <span class="sre-meta-divider" />
                  <span class="alert-meta tnum">{{ relTime(a.last_fired_at) }}</span>
                  <span class="sre-meta-divider" />
                  <span class="alert-meta tnum">{{ a.fire_count }}× fired</span>
                  <span class="alert-status" :class="`s-${a.status}`">{{ a.status }}</span>
                </div>
              </div>
            </n-tab-pane>

            <!-- Timeline -->
            <n-tab-pane name="timeline" :tab="t('incident.timeline')">
              <div v-if="!timeline.length" class="empty-state">
                <n-empty :description="t('incident.noTimeline')" />
              </div>
              <ol v-else class="tl-list">
                <li v-for="entry in timeline" :key="entry.id" class="tl-item">
                  <span class="tl-dot" />
                  <div class="tl-body">
                    <div class="tl-line">
                      <span class="tl-action">{{ entry.action }}</span>
                      <span class="tl-time tnum">{{ relTime(entry.created_at) }}</span>
                    </div>
                    <div v-if="entry.actor || entry.content" class="tl-sub">
                      <span v-if="entry.actor">by {{ entry.actor.display_name ?? entry.actor.username }}</span>
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
                    placeholder="Post-mortem title…"
                  />

                  <MdEditor
                    v-model="postMortem.content"
                    :preview="true"
                    :toolbars-exclude="['github']"
                    language="zh-CN"
                    theme="dark"
                    style="height: 520px; border-radius: 8px"
                  />
                </div>
                <div v-else class="empty-state pm-empty">
                  <p class="pm-empty-text">No post-mortem yet</p>
                  <n-button type="primary" size="small" @click="loadPostMortem">
                    {{ t('common.create') }}
                  </n-button>
                </div>
              </n-spin>
            </n-tab-pane>

          </n-tabs>
        </div>

        <!-- RIGHT SIDEBAR -->
        <aside class="detail-sidebar">
          <section class="side-card">
            <div class="sre-label-eyebrow card-eyebrow">Key info</div>
            <dl class="kv-list">
              <div class="kv-row">
                <dt>Channel</dt>
                <dd>
                  <a v-if="incident.channel" class="kv-link" @click="router.push(`/channels/${incident.channel_id}`)">
                    {{ incident.channel.name }}
                  </a>
                  <span v-else>—</span>
                </dd>
              </div>
              <div class="kv-row">
                <dt>Severity</dt>
                <dd class="kv-flex">
                  <span class="sre-dot" :data-severity="incident.severity" />
                  {{ t(severityLabel[incident.severity] ?? incident.severity) }}
                </dd>
              </div>
              <div class="kv-row">
                <dt>Status</dt>
                <dd>{{ t(statusLabel[incident.status] ?? incident.status) }}</dd>
              </div>
              <div class="kv-row">
                <dt>Triggered</dt>
                <dd class="tnum">{{ formatTime(incident.triggered_at) }}</dd>
              </div>
              <div class="kv-row">
                <dt>Acked</dt>
                <dd class="tnum">{{ incident.acknowledged_at ? formatTime(incident.acknowledged_at) : '—' }}</dd>
              </div>
              <div class="kv-row">
                <dt>Assignee</dt>
                <dd>{{ incident.assigned_user?.display_name ?? incident.assigned_user?.username ?? '—' }}</dd>
              </div>
              <div class="kv-row">
                <dt>Alerts</dt>
                <dd class="tnum">{{ incident.alert_count }}</dd>
              </div>
              <div class="kv-row">
                <dt>Duration</dt>
                <dd class="tnum">{{ durationStr(incident.triggered_at, incident.closed_at) }}</dd>
              </div>
            </dl>
          </section>

          <section class="side-card">
            <div class="sre-label-eyebrow card-eyebrow">Timeline brief</div>
            <ol v-if="timeline.length" class="brief-list">
              <li v-for="e in timeline.slice(0, 5)" :key="e.id" class="brief-item">
                <span class="brief-dot" />
                <div class="brief-body">
                  <div class="brief-action">{{ e.action }}</div>
                  <div class="brief-meta tnum">{{ relTime(e.created_at) }}</div>
                </div>
              </li>
            </ol>
            <p v-else class="brief-empty">No events yet</p>
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

    <!-- Snooze -->
    <n-modal
      v-model:show="showSnooze"
      :title="t('incident.snoozeIncident')"
      preset="card"
      style="width: 420px"
      :bordered="false"
    >
      <div class="snooze-presets">
        <button
          v-for="p in snoozePresets" :key="p.minutes"
          class="preset-btn" :class="{ active: snoozeDuration === p.minutes }"
          @click="snoozeDuration = p.minutes"
        >{{ p.label }}</button>
      </div>
      <div v-if="snoozeDuration === -1" style="margin-top:12px">
        <n-date-picker
          v-model:formatted-value="snoozeCustomUntil"
          type="datetime"
          :is-date-disabled="(ts: number) => ts < Date.now()"
          style="width:100%"
        />
      </div>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showSnooze = false">{{ t('incident.cancelBtn') }}</n-button>
          <n-button type="primary" :loading="snoozeLoading" @click="doSnooze">{{ t('incident.confirmSnooze') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Merge -->
    <n-modal
      v-model:show="showMerge"
      :title="t('incident.mergeToTarget')"
      preset="card"
      style="width: 540px"
      :bordered="false"
    >
      <p class="modal-hint">
        {{ t('incident.mergeDescription') }}
      </p>
      <n-input-group>
        <n-input
          v-model:value="mergeSearch"
          :placeholder="t('incident.searchIncidentHint')"
          @keydown.enter="searchMergeIncidents"
        />
        <n-button :loading="mergeSearchLoading" @click="searchMergeIncidents">{{ t('incident.searchBtn') }}</n-button>
      </n-input-group>
      <div v-if="mergeResults.length" class="picker-list">
        <div
          v-for="inc in mergeResults" :key="inc.id"
          class="picker-row sre-row-card"
          :class="{ selected: mergeTargetId === inc.id }"
          :data-severity="inc.severity"
          @click="mergeTargetId = inc.id"
        >
          <span class="sre-dot" :data-severity="inc.severity" />
          <span class="tnum incident-id-small">#{{ inc.id }}</span>
          <span class="picker-title">{{ inc.title }}</span>
        </div>
      </div>
      <n-empty v-else-if="mergeSearch && !mergeSearchLoading" :description="t('incident.noMatchingIncident')" style="padding:16px 0" />
      <template #footer>
        <n-space justify="end">
          <n-button @click="showMerge = false">{{ t('incident.cancelBtn') }}</n-button>
          <n-popconfirm @positive-click="doMerge">
            <template #trigger>
              <n-button type="error" :loading="mergeLoading" :disabled="!mergeTargetId">
                {{ t('incident.confirmMerge') }}
              </n-button>
            </template>
            {{ t('incident.confirmMergeMsg') }}
          </n-popconfirm>
        </n-space>
      </template>
    </n-modal>

    <!-- Reassign -->
    <n-modal
      v-model:show="showReassign"
      :title="t('incident.reassign')"
      preset="card"
      style="width: 460px"
      :bordered="false"
    >
      <n-input
        v-model:value="reassignSearch"
        :placeholder="t('incident.searchUserHint')"
        clearable
        style="margin-bottom:12px"
        @update:value="searchUsers"
      />
      <n-spin :show="reassignSearchLoading">
        <div class="picker-list">
          <div
            v-for="u in reassignUsers" :key="u.id"
            class="picker-row user-row"
            :class="{ selected: reassignUserId === u.id }"
            @click="reassignUserId = u.id"
          >
            <n-avatar size="small" round>
              {{ (u.display_name || u.username).charAt(0).toUpperCase() }}
            </n-avatar>
            <div class="user-meta">
              <div class="user-name">{{ u.display_name || u.username }}</div>
              <div class="user-handle">{{ u.username }}</div>
            </div>
          </div>
        </div>
      </n-spin>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showReassign = false">{{ t('incident.cancelBtn') }}</n-button>
          <n-button type="primary" :loading="reassignLoading" :disabled="!reassignUserId" @click="doReassign">
            {{ t('incident.confirmReassign') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
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
.alert-status.s-firing { color: #ef4444; background: rgba(239,68,68,0.12); }
.alert-status.s-resolved { color: #22c55e; background: rgba(34,197,94,0.12); }

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
.comment-input:focus { border-color: var(--sre-primary); }
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
.pm-title-input:focus { border-bottom-color: var(--sre-primary); }
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

/* Modals */
.modal-hint { font-size: 13px; color: var(--sre-text-secondary); margin: 0 0 12px; line-height: 1.5; }
.snooze-presets { display: flex; flex-wrap: wrap; gap: 6px; }
.preset-btn {
  background: var(--sre-bg-elevated);
  border: var(--sre-hairline);
  color: var(--sre-text-secondary);
  font-family: inherit;
  font-size: 12px;
  padding: 6px 12px;
  border-radius: var(--sre-radius-sm);
  cursor: pointer;
  transition: all 120ms ease;
}
.preset-btn:hover { color: var(--sre-text-primary); border-color: var(--sre-border-strong); }
.preset-btn.active {
  background: var(--sre-primary-soft);
  border-color: var(--sre-primary);
  color: var(--sre-primary);
}

.picker-list {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 280px;
  overflow-y: auto;
}
.picker-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: var(--sre-radius-sm);
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  cursor: pointer;
  transition: background 120ms ease;
}
.picker-row:hover { background: var(--sre-bg-hover); }
.picker-row.selected {
  background: var(--sre-primary-soft);
  border-color: var(--sre-primary);
}
.incident-id-small {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  font-weight: 500;
}
.picker-title {
  font-size: 13px;
  color: var(--sre-text-primary);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-row { padding: 10px 12px; }
.user-meta { display: flex; flex-direction: column; gap: 1px; }
.user-name { font-size: 13px; font-weight: 500; color: var(--sre-text-primary); }
.user-handle { font-size: 11px; color: var(--sre-text-tertiary); font-family: var(--sre-font-mono); }

/* Fade in */
.sre-fadein {
  animation: sre-fadein 200ms ease-out;
}
@keyframes sre-fadein {
  from { opacity: 0; transform: translateY(2px); }
  to { opacity: 1; transform: none; }
}

@media (max-width: 980px) {
  .detail-layout { grid-template-columns: 1fr; }
  .detail-sidebar { position: static; }
  .overview-grid { grid-template-columns: 1fr; }
}
</style>
