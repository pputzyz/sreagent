<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { alertEventApi, userApi, aiApi } from '@/api'
import type { AlertEvent, AlertTimeline, User } from '@/types'
import { getErrorMessage } from '@/utils/format'
import { formatTime, formatDuration } from '@/utils/format'
import { getStatusLabelKey } from '@/utils/alert'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import {
  ArrowBackOutline,
  RefreshOutline,
  EllipsisHorizontal,
  ChatbubbleOutline,
  CopyOutline,
  OpenOutline,
  SparklesOutline,
  BookOutline,
} from '@vicons/ionicons5'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const { t } = useI18n()

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push('/alert/events')
}

const event = shallowRef<AlertEvent | null>(null)
const timeline = ref<AlertTimeline[]>([])
const commentText = ref('')
const loading = ref(false)
const activeTab = ref<'overview' | 'timeline' | 'ai'>('overview')

const eventId = computed(() => Number(route.params.id))
watch(eventId, () => { if (eventId.value) fetchEvent() })

// ── Silence modal ──
const showSilenceModal = ref(false)
const silenceDuration = ref(60)
const silenceReason = ref('')
const silenceSaving = ref(false)
const lastTriggerEl = ref<HTMLElement | null>(null)
const silenceDurationOptions = [
  { label: '30m', value: 30 }, { label: '1h', value: 60 },
  { label: '2h', value: 120 }, { label: '6h', value: 360 },
  { label: '12h', value: 720 }, { label: '24h', value: 1440 },
]

// ── Assign modal ──
const showAssignModal = ref(false)
const assignUserId = ref<number | null>(null)
const assignNote = ref('')
const assignSaving = ref(false)
const users = ref<User[]>([])
const userOptions = computed(() =>
  users.value.map(u => ({ label: `${u.display_name} (${u.username})`, value: u.id }))
)

// ── Severity helpers ──
const severityKey = computed(() => {
  const s = event.value?.severity ?? 'info'
  if (s === 'p0' || s === 'critical') return 'critical'
  if (s === 'p1' || s === 'p2' || s === 'warning') return 'warning'
  return 'info'
})
const severityLabel = computed(() => (event.value?.severity ?? '').toString().toUpperCase())

// ── Status helpers ──
const statusKey = computed(() => event.value?.status ?? 'firing')
const statusLabel = computed(() => (event.value ? t(getStatusLabelKey(event.value.status)) : ''))

// ── Duration (live-updating every second) ──
const timeTick = ref(0)
let tickTimer: ReturnType<typeof setInterval> | null = null

const eventDuration = computed(() => {
  void timeTick.value // depend on tick for re-evaluation
  if (!event.value) return '—'
  const firedAt = new Date(event.value.fired_at).getTime()
  if (event.value.status === 'resolved' || event.value.status === 'closed') {
    const end = event.value.resolved_at
      ? new Date(event.value.resolved_at).getTime()
      : (event.value.closed_at ? new Date(event.value.closed_at).getTime() : Date.now())
    return formatDuration(Math.floor((end - firedAt) / 1000))
  }
  return formatDuration(Math.floor((Date.now() - firedAt) / 1000))
})

// ── Action guards ──
const canAck = computed(() => event.value?.status === 'firing')
const canAssign = computed(() => event.value?.status === 'firing' || event.value?.status === 'acknowledged' || event.value?.status === 'assigned')
const canSilence = computed(() => event.value?.status === 'firing' || event.value?.status === 'acknowledged' || event.value?.status === 'assigned')
const canResolve = computed(() => event.value && event.value.status !== 'resolved' && event.value.status !== 'closed')
const canClose = computed(() => event.value?.status !== 'closed')

// ── Timeline label ──
function getTimelineLabel(action: string): string {
  const map: Record<string, string> = {
    created: t('alert.created'),
    acknowledged: t('alert.acknowledged'),
    assigned: t('alert.assigned'),
    resolved: t('alert.resolved'),
    closed: t('common.close'),
    commented: t('alert.commented'),
    silenced: t('alert.silenced'),
    escalated: t('alert.escalated'),
    notified: t('alert.notified'),
    reopened: t('alert.reopened'),
  }
  return map[action] ?? action
}

function timelineDotSeverity(action: string): string {
  switch (action) {
    case 'created':
    case 'escalated': return 'critical'
    case 'acknowledged':
    case 'silenced': return 'warning'
    case 'resolved': return 'success'
    case 'assigned':
    case 'notified': return 'info'
    default: return 'muted'
  }
}

// ── Copy ──
function copyText(value: string, hint = t('common.copied')) {
  navigator.clipboard.writeText(value).then(() => message.success(hint)).catch(() => {
    message.error(t('common.copyFailed') || 'Copy failed')
  })
}

// ── API ──
async function fetchEvent() {
  loading.value = true
  try {
    const { data } = await alertEventApi.get(eventId.value)
    event.value = data.data
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
  finally { loading.value = false }
}

async function fetchTimeline() {
  try {
    const { data } = await alertEventApi.getTimeline(eventId.value)
    timeline.value = data.data || []
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
}

async function fetchUsers() {
  try {
    const { data } = await userApi.list({ page: 1, page_size: 200, is_active: true })
    users.value = data.data.list || []
  } catch { message.error(t('common.loadFailed')) }
}

async function refreshAll() {
  await Promise.all([fetchEvent(), fetchTimeline()])
}

const actionLoading = ref(false)

async function handleAck() {
  actionLoading.value = true
  try {
    await alertEventApi.acknowledge(eventId.value)
    message.success(t('alert.alertAcknowledged'))
    refreshAll()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
  finally { actionLoading.value = false }
}

async function handleResolve() {
  actionLoading.value = true
  try {
    await alertEventApi.resolve(eventId.value, { resolution: t('alert.manuallyResolved') })
    message.success(t('alert.alertResolved'))
    refreshAll()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
  finally { actionLoading.value = false }
}

async function handleClose() {
  actionLoading.value = true
  try {
    await alertEventApi.close(eventId.value)
    message.success(t('alert.alertClosed'))
    refreshAll()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
  finally { actionLoading.value = false }
}

function openSilenceModal() {
  lastTriggerEl.value = document.activeElement as HTMLElement
  silenceDuration.value = 60; silenceReason.value = ''; showSilenceModal.value = true
}

async function handleSilence() {
  if (!silenceReason.value.trim()) { message.warning(t('alert.silenceReasonPlaceholder')); return }
  silenceSaving.value = true
  try {
    await alertEventApi.silence(eventId.value, { duration_minutes: silenceDuration.value, reason: silenceReason.value })
    message.success(t('alert.silenceSuccess'))
    showSilenceModal.value = false; refreshAll()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
  finally { silenceSaving.value = false }
}

function openAssignModal() {
  lastTriggerEl.value = document.activeElement as HTMLElement
  assignUserId.value = null; assignNote.value = ''; showAssignModal.value = true
  if (users.value.length === 0) fetchUsers()
}

async function handleAssign() {
  if (!assignUserId.value) { message.warning(t('alert.selectUser')); return }
  assignSaving.value = true
  try {
    await alertEventApi.assign(eventId.value, { assign_to: assignUserId.value, note: assignNote.value || undefined })
    message.success(t('alert.assignSuccess'))
    showAssignModal.value = false; refreshAll()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
  finally { assignSaving.value = false }
}

const commentLoading = ref(false)
async function handleComment() {
  if (!commentText.value.trim()) return
  commentLoading.value = true
  try {
    await alertEventApi.comment(eventId.value, { note: commentText.value })
    commentText.value = ''
    message.success(t('alert.commentAdded'))
    fetchTimeline()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
  finally { commentLoading.value = false }
}

// ── Dropdown overflow actions ──
const moreOptions = computed(() => {
  const opts: { label: string; key: string; disabled?: boolean }[] = []
  opts.push({ label: t('alert.assign'), key: 'assign', disabled: !canAssign.value })
  opts.push({ label: t('alert.silence'), key: 'silence', disabled: !canSilence.value })
  opts.push({ label: t('alert.generateReport'), key: 'ai' })
  opts.push({ label: t('alert.aiSuggestSOP'), key: 'sop' })
  return opts
})
function handleMore(key: string) {
  if (key === 'assign') openAssignModal()
  else if (key === 'silence') openSilenceModal()
  else if (key === 'ai') { activeTab.value = 'ai'; if (!aiReport.value) generateAIReport() }
  else if (key === 'sop') { activeTab.value = 'ai'; if (!sopReport.value) generateSOP() }
}

// ── AI ──
const aiReport = ref<string | null>(null)
const aiReportLoading = ref(false)
const aiReportError = ref('')

async function generateAIReport() {
  aiReportLoading.value = true; aiReportError.value = ''
  try {
    const res = await aiApi.generateReport(eventId.value)
    const d = res.data.data
    aiReport.value = typeof d === 'string' ? d : (d as any)?.report ?? JSON.stringify(d)
  } catch (err: unknown) { aiReportError.value = getErrorMessage(err) || t('alert.aiReportError') }
  finally { aiReportLoading.value = false }
}

const sopReport = ref<string | null>(null)
const sopLoading = ref(false)
const sopError = ref('')

// ── Root Cause Analysis ──
interface RootCauseResult {
  summary: string
  severity: string
  probable_causes: string[]
  impact: string
  recommended_steps: string[]
  root_cause_hint: string
}
const rcaResult = ref<RootCauseResult | null>(null)
const rcaLoading = ref(false)
const rcaError = ref('')

async function generateRootCauseAnalysis() {
  rcaLoading.value = true; rcaError.value = ''
  try {
    const res = await aiApi.analyzeAlert(eventId.value)
    rcaResult.value = res.data.data ?? null
  } catch (err: unknown) { rcaError.value = getErrorMessage(err) || t('alert.aiReportError') }
  finally { rcaLoading.value = false }
}

async function generateSOP() {
  sopLoading.value = true; sopError.value = ''
  try {
    const res = await aiApi.suggestSOP(eventId.value)
    const d = res.data.data
    sopReport.value = typeof d === 'string' ? d : (d as any)?.sop ?? JSON.stringify(d)
  } catch (err: unknown) { sopError.value = getErrorMessage(err) || t('alert.aiReportError') }
  finally { sopLoading.value = false }
}

// ── Related rule ──
function gotoRule() {
  if (event.value?.rule_id) router.push(`/alert/rules/${event.value.rule_id}`)
}

onMounted(() => {
  fetchEvent(); fetchTimeline()
  tickTimer = setInterval(() => { timeTick.value++ }, 1000)
})

onUnmounted(() => {
  if (tickTimer) { clearInterval(tickTimer); tickTimer = null }
})
</script>

<template>
  <div class="evt-page" v-if="event">
    <!-- ═══ HEADER ═══ -->
    <header class="evt-header sre-stagger">
      <div class="evt-header-top">
        <n-button quaternary circle size="small" @click="goBack">
          <template #icon><n-icon :component="ArrowBackOutline" /></template>
        </n-button>
        <div class="evt-header-title-block">
          <h1 class="evt-title">{{ event.alert_name }}</h1>
          <div class="evt-subtitle sre-row-card" :data-severity="severityKey">
            <span class="sre-dot" :data-severity="severityKey" />
            <span class="evt-sev-label">{{ severityLabel }}</span>
            <span class="sre-meta-divider" />
            <span class="evt-status" :data-status="statusKey">{{ statusLabel }}</span>
            <span class="sre-meta-divider" />
            <span class="evt-id tnum">#{{ event.id }}</span>
            <span class="sre-meta-divider" />
            <span class="evt-fired tnum">{{ formatTime(event.fired_at) }}</span>
            <span class="sre-meta-divider" />
            <span class="evt-duration tnum">{{ eventDuration }}</span>
          </div>
        </div>
        <div class="evt-header-actions">
          <n-button quaternary size="small" @click="refreshAll">
            <template #icon><n-icon :component="RefreshOutline" /></template>
            {{ t('common.refresh') }}
          </n-button>
          <n-button v-if="event.rule_id" quaternary size="small" @click="gotoRule">
            <template #icon><n-icon :component="OpenOutline" /></template>
            {{ t('alert.rule') }}
          </n-button>
        </div>
      </div>

      <!-- ═══ ACTION BAR ═══ -->
      <div class="evt-action-bar">
        <n-space :size="8" align="center">
          <template v-if="statusKey === 'firing'">
            <n-button type="primary" size="small" :loading="actionLoading" @click="handleAck">{{ t('alert.acknowledge') }}</n-button>
            <n-button size="small" secondary :loading="actionLoading" @click="handleResolve" v-if="canResolve">{{ t('alert.resolve') }}</n-button>
            <n-button size="small" secondary :loading="actionLoading" @click="handleClose" v-if="canClose">{{ t('common.close') }}</n-button>
          </template>
          <template v-else-if="statusKey === 'acknowledged' || statusKey === 'assigned'">
            <n-button type="primary" size="small" :loading="actionLoading" @click="handleResolve" v-if="canResolve">{{ t('alert.resolve') }}</n-button>
            <n-button size="small" secondary :loading="actionLoading" @click="handleClose" v-if="canClose">{{ t('common.close') }}</n-button>
            <n-button size="small" quaternary @click="openAssignModal" v-if="canAssign">{{ t('alert.assign') }}</n-button>
          </template>
          <template v-else-if="statusKey === 'resolved'">
            <n-button size="small" secondary :loading="actionLoading" @click="handleClose" v-if="canClose">{{ t('common.close') }}</n-button>
          </template>

          <n-button v-if="canSilence" type="warning" size="small" ghost @click="openSilenceModal">
            {{ t('alert.silence') }}
          </n-button>

          <n-dropdown :options="moreOptions" trigger="click" @select="handleMore" placement="bottom-end">
            <n-button quaternary size="small">
              <template #icon><n-icon :component="EllipsisHorizontal" /></template>
            </n-button>
          </n-dropdown>
        </n-space>
      </div>
    </header>

    <!-- ═══ MAIN GRID ═══ -->
    <div class="evt-grid sre-stagger">
      <!-- ── Left column ── -->
      <main class="evt-main">
        <n-tabs v-model:value="activeTab" type="line" animated size="medium">
          <!-- Overview -->
          <n-tab-pane name="overview" :tab="t('alert.overview')">
            <section class="evt-section">
              <div class="sre-label-eyebrow">{{ t('alert.summary') }}</div>
              <div class="evt-summary">
                <template v-if="event.annotations?.summary || event.annotations?.description">
                  <p v-if="event.annotations.summary" class="evt-summary-text">{{ event.annotations.summary }}</p>
                  <p v-if="event.annotations.description" class="evt-summary-text muted">{{ event.annotations.description }}</p>
                </template>
                <p v-else class="evt-summary-text muted">{{ event.alert_name }}</p>
              </div>
            </section>

            <section class="evt-section" v-if="event.labels && Object.keys(event.labels).length">
              <div class="sre-label-eyebrow">{{ t('alert.labels') }}</div>
              <div class="evt-chips">
                <button
                  v-for="(value, key) in event.labels"
                  :key="key"
                  class="evt-chip"
                  @click="copyText(`${key}=${value}`, t('tooltip.copiedKey', { key: `${key}` }))"
                  :title="`${key}=${value}`"
                >
                  <span class="evt-chip-key">{{ key }}</span>
                  <span class="evt-chip-eq">=</span>
                  <span class="evt-chip-val">{{ value }}</span>
                </button>
              </div>
            </section>

            <section class="evt-section" v-if="event.annotations && Object.keys(event.annotations).length">
              <div class="sre-label-eyebrow">{{ t('alert.annotations') }}</div>
              <dl class="evt-kv">
                <template v-for="(value, key) in event.annotations" :key="key">
                  <dt>{{ key }}</dt>
                  <dd>{{ value }}</dd>
                </template>
              </dl>
            </section>

            <section class="evt-section" v-if="event.rule">
              <div class="sre-label-eyebrow">{{ t('alert.rule') }}</div>
              <div class="evt-rule-card">
                <div class="evt-rule-head">
                  <div class="evt-rule-name">{{ event.rule.name }}</div>
                  <n-button quaternary size="tiny" @click="gotoRule">
                    <template #icon><n-icon :component="OpenOutline" /></template>
                    {{ t('common.open') }}
                  </n-button>
                </div>
                <pre v-if="event.rule.expression" class="evt-rule-expr">{{ event.rule.expression }}</pre>
              </div>
            </section>
          </n-tab-pane>

          <!-- Timeline -->
          <n-tab-pane name="timeline" :tab="t('alert.timeline')">
            <section class="evt-section">
              <ol class="evt-timeline" v-if="timeline.length" :aria-label="t('alert.timeline')">
                <li v-for="item in timeline" :key="item.id" class="evt-tl-item">
                  <span class="evt-tl-dot sre-dot" :data-severity="timelineDotSeverity(item.action)" />
                  <div class="evt-tl-body">
                    <div class="evt-tl-row">
                      <span class="evt-tl-action">{{ getTimelineLabel(item.action) }}</span>
                      <span v-if="item.operator" class="evt-tl-operator">
                        {{ item.operator.display_name || item.operator.username }}
                      </span>
                      <span class="evt-tl-time tnum">{{ formatTime(item.created_at) }}</span>
                    </div>
                    <p v-if="item.note" class="evt-tl-note">{{ item.note }}</p>
                  </div>
                </li>
              </ol>
              <n-empty v-else size="small" />

              <div class="evt-comment-box">
                <n-input
                  v-model:value="commentText"
                  type="textarea"
                  :placeholder="t('alert.addCommentPlaceholder')"
                  :rows="2"
                />
                <div class="evt-comment-actions">
                  <n-button type="primary" size="small" :loading="commentLoading" :disabled="!commentText.trim()" @click="handleComment">
                    <template #icon><n-icon :component="ChatbubbleOutline" /></template>
                    {{ t('alert.addComment') }}
                  </n-button>
                </div>
              </div>
            </section>
          </n-tab-pane>

          <!-- AI -->
          <n-tab-pane name="ai" :tab="t('alert.aiAnalysis')">
            <section class="evt-section">
              <div class="evt-ai-head">
                <div class="sre-label-eyebrow">
                  <n-icon :component="SparklesOutline" :size="12" />
                  {{ t('alert.aiAnalysis') }}
                </div>
                <n-button quaternary size="tiny" @click="generateAIReport">
                  {{ aiReport ? t('alert.regenerateReport') : t('alert.generateReport') }}
                </n-button>
              </div>
              <n-spin :show="aiReportLoading">
                <p v-if="!aiReport && !aiReportError && !aiReportLoading" class="evt-ai-hint">
                  {{ t('alert.aiAnalysisHint') }}
                </p>
                <n-alert v-if="aiReportError" type="error" :bordered="false" size="small">
                  {{ aiReportError }}
                </n-alert>
                <div v-if="aiReport" class="evt-ai-report">
                  <pre class="evt-ai-text">{{ aiReport }}</pre>
                </div>
              </n-spin>
            </section>

            <section class="evt-section">
              <div class="evt-ai-head">
                <div class="sre-label-eyebrow">
                  <n-icon :component="BookOutline" :size="12" />
                  {{ t('alert.aiSuggestSOP') }}
                </div>
                <n-button quaternary size="tiny" @click="generateSOP">
                  {{ sopReport ? t('alert.regenerateReport') : t('alert.aiSuggestSOP') }}
                </n-button>
              </div>
              <n-spin :show="sopLoading">
                <n-alert v-if="sopError" type="error" :bordered="false" size="small">{{ sopError }}</n-alert>
                <div v-if="sopReport" class="evt-ai-report">
                  <pre class="evt-ai-text">{{ sopReport }}</pre>
                </div>
              </n-spin>
            </section>

            <!-- Root Cause Analysis -->
            <section class="evt-section">
              <div class="evt-ai-head">
                <div class="sre-label-eyebrow">
                  <n-icon :component="SparklesOutline" :size="12" />
                  {{ t('alert.rootCauseAnalysis') }}
                </div>
                <n-button quaternary size="tiny" @click="generateRootCauseAnalysis">
                  {{ rcaResult ? t('alert.regenerateReport') : t('alert.rootCauseAnalysis') }}
                </n-button>
              </div>
              <n-spin :show="rcaLoading">
                <n-alert v-if="rcaError" type="error" :bordered="false" size="small">{{ rcaError }}</n-alert>
                <div v-if="rcaResult" class="evt-ai-report">
                  <div v-if="rcaResult.summary" class="rca-section">
                    <strong>{{ t('alert.summary') }}:</strong>
                    <p>{{ rcaResult.summary }}</p>
                  </div>
                  <div v-if="rcaResult.probable_causes?.length" class="rca-section">
                    <strong>{{ t('alert.probableCauses') }}:</strong>
                    <ul>
                      <li v-for="(cause, i) in rcaResult.probable_causes" :key="i">{{ cause }}</li>
                    </ul>
                  </div>
                  <div v-if="rcaResult.impact" class="rca-section">
                    <strong>{{ t('alert.impact') }}:</strong>
                    <p>{{ rcaResult.impact }}</p>
                  </div>
                  <div v-if="rcaResult.recommended_steps?.length" class="rca-section">
                    <strong>{{ t('alert.recommendedSteps') }}:</strong>
                    <ol>
                      <li v-for="(step, i) in rcaResult.recommended_steps" :key="i">{{ step }}</li>
                    </ol>
                  </div>
                  <div v-if="rcaResult.root_cause_hint" class="rca-section">
                    <strong>{{ t('alert.rootCauseHint') }}:</strong>
                    <p>{{ rcaResult.root_cause_hint }}</p>
                  </div>
                </div>
              </n-spin>
            </section>
          </n-tab-pane>
        </n-tabs>
      </main>

      <!-- ── Right sidebar ── -->
      <aside class="evt-aside">
        <!-- Key Info -->
        <div class="evt-aside-card">
          <div class="sre-label-eyebrow">{{ t('alert.keyInfo') }}</div>
          <dl class="evt-meta">
            <div class="evt-meta-row">
              <dt>{{ t('alert.severity') }}</dt>
              <dd><span class="sre-dot" :data-severity="severityKey" />{{ severityLabel }}</dd>
            </div>
            <div class="evt-meta-row">
              <dt>{{ t('alert.status') }}</dt>
              <dd>{{ statusLabel }}</dd>
            </div>
            <div class="evt-meta-row" v-if="event.rule">
              <dt>{{ t('alert.rule') }}</dt>
              <dd><a class="evt-link" @click="gotoRule">{{ event.rule.name }}</a></dd>
            </div>
            <div class="evt-meta-row">
              <dt>{{ t('alert.source') }}</dt>
              <dd>{{ event.source || '—' }}</dd>
            </div>
            <div class="evt-meta-row">
              <dt>{{ t('alert.firedAt') }}</dt>
              <dd class="tnum">{{ formatTime(event.fired_at) }}</dd>
            </div>
            <div class="evt-meta-row" v-if="event.acked_at">
              <dt>{{ t('alert.ackedAt') }}</dt>
              <dd class="tnum">{{ formatTime(event.acked_at) }}</dd>
            </div>
            <div class="evt-meta-row" v-if="event.resolved_at">
              <dt>{{ t('alert.resolvedAt') }}</dt>
              <dd class="tnum">{{ formatTime(event.resolved_at) }}</dd>
            </div>
            <div class="evt-meta-row" v-if="event.closed_at">
              <dt>{{ t('alert.closedAt') }}</dt>
              <dd class="tnum">{{ formatTime(event.closed_at) }}</dd>
            </div>
            <div class="evt-meta-row">
              <dt>{{ t('alert.fireCount') }}</dt>
              <dd class="tnum evt-fire-count">×{{ event.fire_count }}</dd>
            </div>
            <div class="evt-meta-row" v-if="event.silenced_until">
              <dt>{{ t('alert.silence') }}</dt>
              <dd class="tnum evt-silenced">{{ formatTime(event.silenced_until) }}</dd>
            </div>
          </dl>
        </div>

        <!-- Responders -->
        <div class="evt-aside-card" v-if="event.acked_by_user || event.assigned_to_user || event.oncall_user">
          <div class="sre-label-eyebrow">{{ t('common.responders') }}</div>
          <ul class="evt-responders">
            <li v-if="event.acked_by_user">
              <span class="evt-resp-avatar">{{ (event.acked_by_user.display_name || '?').charAt(0).toUpperCase() }}</span>
              <div class="evt-resp-info">
                <div class="evt-resp-name">{{ event.acked_by_user.display_name || event.acked_by_user.username }}</div>
                <div class="evt-resp-role">{{ t('alert.acknowledged') }}</div>
              </div>
            </li>
            <li v-if="event.assigned_to_user">
              <span class="evt-resp-avatar">{{ (event.assigned_to_user.display_name || '?').charAt(0).toUpperCase() }}</span>
              <div class="evt-resp-info">
                <div class="evt-resp-name">{{ event.assigned_to_user.display_name || event.assigned_to_user.username }}</div>
                <div class="evt-resp-role">{{ t('alert.assignedTo') }}</div>
              </div>
            </li>
            <li v-if="event.oncall_user">
              <span class="evt-resp-avatar">{{ (event.oncall_user.display_name || '?').charAt(0).toUpperCase() }}</span>
              <div class="evt-resp-info">
                <div class="evt-resp-name">{{ event.oncall_user.display_name || event.oncall_user.username }}</div>
                <div class="evt-resp-role">{{ t('alert.oncallUser') }}</div>
              </div>
            </li>
          </ul>
        </div>

        <!-- Labels (compact) -->
        <div class="evt-aside-card" v-if="event.labels && Object.keys(event.labels).length">
          <div class="sre-label-eyebrow">{{ t('alert.labels') }}</div>
          <ul class="evt-aside-labels">
            <li
              v-for="(value, key) in event.labels"
              :key="key"
              @click="copyText(`${key}=${value}`, t('tooltip.copiedKey', { key: `${key}` }))"
              :title="t('tooltip.clickToCopy', { key: `${key}=${value}` })"
            >
              <span class="evt-aside-k">{{ key }}</span>
              <span class="evt-aside-v">{{ value }}</span>
            </li>
          </ul>
        </div>

        <!-- Related -->
        <div class="evt-aside-card">
          <div class="sre-label-eyebrow">{{ t('alert.related') }}</div>
          <ul class="evt-related">
            <li v-if="event.rule_id">
              <span>{{ t('alert.rule') }}</span>
              <a class="evt-link" @click="gotoRule">{{ event.rule?.name || `#${event.rule_id}` }}</a>
            </li>
            <li v-if="event.generator_url">
              <span>{{ t('alert.source') }}</span>
              <a class="evt-link" :href="event.generator_url" target="_blank" rel="noopener">↗ {{ t('tooltip.view') }}</a>
            </li>
            <li>
              <span>{{ t('alert.fingerprint') }}</span>
              <code class="evt-fp" @click="copyText(event.fingerprint, t('alert.fingerprintCopied'))">
                {{ event.fingerprint.slice(0, 12) }}…
                <n-icon :component="CopyOutline" :size="10" />
              </code>
            </li>
          </ul>
        </div>
      </aside>
    </div>

    <!-- Silence Modal -->
    <n-modal v-model:show="showSilenceModal" preset="card" :title="t('alert.silence')" style="width: 480px" :bordered="false" @after-leave="lastTriggerEl?.focus()">
      <n-form label-placement="top">
        <n-form-item :label="t('alert.silenceDuration')">
          <n-radio-group v-model:value="silenceDuration">
            <n-space>
              <n-radio-button v-for="opt in silenceDurationOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</n-radio-button>
            </n-space>
          </n-radio-group>
        </n-form-item>
        <n-form-item :label="t('alert.silenceReason')">
          <n-input v-model:value="silenceReason" type="textarea" :placeholder="t('alert.silenceReasonPlaceholder')" :rows="3" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space justify="end">
          <n-button @click="showSilenceModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="warning" :loading="silenceSaving" @click="handleSilence">{{ t('common.confirm') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Assign Modal -->
    <n-modal v-model:show="showAssignModal" preset="card" :title="t('alert.assign')" style="width: 480px" :bordered="false" @after-leave="lastTriggerEl?.focus()">
      <n-form label-placement="top">
        <n-form-item :label="t('alert.assignTo')">
          <n-select v-model:value="assignUserId" :options="userOptions" :placeholder="t('alert.selectUser')" filterable />
        </n-form-item>
        <n-form-item :label="t('alert.assignNote')">
          <n-input v-model:value="assignNote" type="textarea" :placeholder="t('alert.assignNotePlaceholder')" :rows="3" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space justify="end">
          <n-button @click="showAssignModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="assignSaving" @click="handleAssign">{{ t('common.confirm') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>

  <LoadingSkeleton v-else-if="loading" :rows="6" variant="row" />
</template>

<style scoped>
.evt-page {
  max-width: 1400px;
  font-family: var(--sre-font-sans);
  letter-spacing: -0.005em;
}

/* ── Header ── */
.evt-header {
  margin-bottom: 16px;
}
.evt-header-top {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.evt-header-title-block {
  flex: 1;
  min-width: 0;
}
.evt-title {
  font-family: var(--sre-font-sans);
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--sre-text-primary);
  margin: 0 0 8px 0;
  line-height: 1.25;
  word-break: break-word;
}
.evt-subtitle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px 4px 10px;
  border-radius: 8px;
  font-size: 12px;
  color: var(--sre-text-secondary);
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
}
.evt-sev-label {
  font-weight: 600;
  color: var(--sre-text-primary);
  letter-spacing: 0.04em;
  font-size: 11px;
}
.evt-status {
  font-weight: 500;
  color: var(--sre-text-primary);
  text-transform: lowercase;
}
.evt-id, .evt-fired, .evt-duration {
  color: var(--sre-text-secondary);
}
.evt-header-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

/* Action bar */
.evt-action-bar {
  margin-top: 14px;
  padding-top: 14px;
  border-top: var(--sre-hairline);
}

/* ── Grid ── */
.evt-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  gap: 16px;
  align-items: flex-start;
}
@media (max-width: 1024px) {
  .evt-grid { grid-template-columns: 1fr; }
}

.evt-main {
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: 12px;
  padding: 4px 16px 16px;
  min-width: 0;
}

.evt-section {
  padding: 14px 0;
  border-bottom: var(--sre-hairline);
}
.evt-section:last-child { border-bottom: 0; }

/* Summary */
.evt-summary {
  margin-top: 8px;
}
.evt-summary-text {
  font-size: 14px;
  line-height: 1.6;
  color: var(--sre-text-primary);
  margin: 0 0 6px;
}
.evt-summary-text.muted { color: var(--sre-text-secondary); font-size: 13px; }

/* Chips */
.evt-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}
.evt-chip {
  display: inline-flex;
  align-items: center;
  font-family: var(--sre-font-mono);
  font-size: 12px;
  border: var(--sre-hairline);
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  overflow: hidden;
  transition: border-color 0.15s, background 0.15s;
  max-width: 360px;
  padding: 0;
}
.evt-chip:hover {
  border-color: var(--sre-primary-ring, var(--sre-text-primary));
  background: var(--sre-primary-soft);
}
.evt-chip-key { padding: 3px 7px; color: var(--sre-text-secondary); }
.evt-chip-eq  { padding: 3px 1px; color: var(--sre-text-secondary); opacity: 0.6; }
.evt-chip-val {
  padding: 3px 7px;
  color: var(--sre-text-primary);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 220px;
}

/* Annotations KV */
.evt-kv { display: grid; grid-template-columns: 140px 1fr; gap: 6px 12px; margin: 8px 0 0; }
.evt-kv dt {
  font-family: var(--sre-font-mono);
  font-size: 12px;
  color: var(--sre-text-secondary);
  padding-top: 1px;
}
.evt-kv dd {
  font-size: 13px;
  color: var(--sre-text-primary);
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.55;
}

/* Rule card */
.evt-rule-card {
  margin-top: 8px;
  border: var(--sre-hairline);
  border-radius: 8px;
  padding: 10px 12px;
}
.evt-rule-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.evt-rule-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
}
.evt-rule-expr {
  margin: 8px 0 0;
  padding: 8px 10px;
  background: var(--sre-bg-elevated);
  border-radius: 6px;
  font-family: var(--sre-font-mono);
  font-size: 11.5px;
  color: var(--sre-text-primary);
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  line-height: 1.5;
}

/* Timeline */
.evt-timeline {
  list-style: none;
  margin: 8px 0 0;
  padding: 0;
  position: relative;
}
.evt-timeline::before {
  content: '';
  position: absolute;
  left: 5px;
  top: 6px;
  bottom: 6px;
  width: 1px;
  background: var(--sre-hairline);
}
.evt-tl-item {
  position: relative;
  padding: 4px 0 14px 22px;
}
.evt-tl-dot {
  position: absolute !important;
  left: 0;
  top: 8px;
  width: 11px;
  height: 11px;
}
.evt-tl-body { min-width: 0; }
.evt-tl-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.evt-tl-action {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-primary);
}
.evt-tl-operator {
  font-size: 12px;
  color: var(--sre-text-secondary);
  background: var(--sre-bg-hover);
  padding: 1px 6px;
  border-radius: 4px;
}
.evt-tl-time {
  font-size: 11px;
  color: var(--sre-text-secondary);
  margin-left: auto;
}
.evt-tl-note {
  margin: 4px 0 0;
  padding: 6px 9px;
  background: var(--sre-bg-subtle);
  border-radius: 6px;
  font-size: 12px;
  color: var(--sre-text-secondary);
  white-space: pre-wrap;
  line-height: 1.55;
}

/* Comment */
.evt-comment-box {
  margin-top: 14px;
  padding-top: 12px;
  border-top: var(--sre-hairline);
}
.evt-comment-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

/* AI */
.evt-ai-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}
.evt-ai-hint {
  font-size: 12px;
  color: var(--sre-text-secondary);
  text-align: center;
  padding: 14px 0;
  margin: 0;
  line-height: 1.6;
}
.evt-ai-report { font-size: 13px; color: var(--sre-text-primary); }
.rca-section { margin-bottom: 12px; }
.rca-section strong { display: block; margin-bottom: 4px; color: var(--sre-text-secondary); font-size: 12px; text-transform: uppercase; letter-spacing: 0.03em; }
.rca-section p { margin: 0; line-height: 1.6; }
.rca-section ul, .rca-section ol { margin: 4px 0 0; padding-left: 20px; }
.rca-section li { margin-bottom: 4px; line-height: 1.5; }
.evt-ai-summary {
  margin: 0 0 12px;
  padding: 10px 12px;
  background: var(--sre-bg-subtle);
  border-radius: 8px;
  line-height: 1.6;
  white-space: pre-wrap;
}
.evt-ai-block { margin-bottom: 12px; }
.evt-ai-block ul, .evt-ai-block ol {
  margin: 4px 0 0; padding-left: 18px; line-height: 1.7; color: var(--sre-text-primary);
}
.evt-ai-block p { margin: 4px 0 0; line-height: 1.6; }
.evt-sop-title { font-size: 14px; font-weight: 600; margin: 0 0 8px; color: var(--sre-text-primary); }

/* ── Aside ── */
.evt-aside {
  display: flex;
  flex-direction: column;
  gap: 12px;
  position: sticky;
  top: 16px;
}
.evt-aside-card {
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: 12px;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.evt-meta { margin: 0; display: flex; flex-direction: column; gap: 7px; }
.evt-meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
}
.evt-meta-row dt {
  color: var(--sre-text-secondary);
  flex-shrink: 0;
}
.evt-meta-row dd {
  margin: 0;
  color: var(--sre-text-primary);
  font-weight: 500;
  text-align: right;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  word-break: break-all;
}
.evt-fire-count { color: var(--sre-critical); font-weight: 700; }
.evt-silenced { color: var(--sre-aurora-3); }

.evt-link {
  color: var(--sre-info);
  cursor: pointer;
  text-decoration: none;
}
.evt-link:hover { text-decoration: underline; }

/* Responders */
.evt-responders { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.evt-responders li { display: flex; align-items: center; gap: 10px; }
.evt-resp-avatar {
  width: 28px; height: 28px;
  border-radius: 50%;
  background: var(--sre-primary-soft);
  color: var(--sre-text-primary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
  border: var(--sre-hairline);
}
.evt-resp-info { min-width: 0; flex: 1; }
.evt-resp-name { font-size: 12.5px; font-weight: 600; color: var(--sre-text-primary); }
.evt-resp-role { font-size: 11px; color: var(--sre-text-secondary); margin-top: 1px; }

/* Aside labels */
.evt-aside-labels { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 4px; }
.evt-aside-labels li {
  display: flex; align-items: center; gap: 6px;
  font-family: var(--sre-font-mono);
  font-size: 11.5px;
  cursor: pointer;
  padding: 2px 0;
  border-radius: 4px;
}
.evt-aside-labels li:hover { background: var(--sre-bg-subtle); }
.evt-aside-k { color: var(--sre-text-secondary); }
.evt-aside-v {
  color: var(--sre-text-primary);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Related */
.evt-related { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 7px; }
.evt-related li {
  display: flex; justify-content: space-between; align-items: center;
  font-size: 12px;
  gap: 8px;
}
.evt-related li > span:first-child { color: var(--sre-text-secondary); }
.evt-fp {
  font-family: var(--sre-font-mono);
  font-size: 11px;
  color: var(--sre-text-secondary);
  background: var(--sre-bg-hover);
  padding: 2px 6px;
  border-radius: 4px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.evt-fp:hover { color: var(--sre-text-primary); }

/* Tabular nums */
.tnum {
  font-variant-numeric: tabular-nums;
  font-feature-settings: 'tnum';
}
</style>
