<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage, NTag, NButton, NSpace } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { incidentApi, alertV2Api, userApi } from '@/api'
import type { Incident, IncidentTimeline, AlertV2, User } from '@/types'
import { formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import {
  ArrowBackOutline, SparklesOutline, VolumeOffOutline,
  TimeOutline, GitMergeOutline, PersonOutline,
} from '@vicons/ionicons5'
import QuickSilenceModal from '@/components/noise/QuickSilenceModal.vue'
import { MdEditor } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'

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

// Quick silence
const showQuickSilence = ref(false)

// Post-mortem
const postMortem = ref<any | null>(null)
const pmLoading = ref(false)
const pmSaving = ref(false)
const pmAiLoading = ref(false)

// ---- Snooze modal ----
const showSnooze = ref(false)
const snoozeLoading = ref(false)
const snoozeDuration = ref<number | null>(null) // minutes preset
const snoozeCustomUntil = ref('')
const snoozePresets = [
  { label: '15 分钟', minutes: 15 },
  { label: '30 分钟', minutes: 30 },
  { label: '1 小时', minutes: 60 },
  { label: '2 小时', minutes: 120 },
  { label: '4 小时', minutes: 240 },
  { label: '自定义', minutes: -1 },
]

async function doSnooze() {
  let until: string
  if (snoozeDuration.value === -1) {
    if (!snoozeCustomUntil.value) {
      message.warning('请选择暂缓结束时间')
      return
    }
    until = new Date(snoozeCustomUntil.value).toISOString()
  } else if (snoozeDuration.value) {
    const d = new Date()
    d.setMinutes(d.getMinutes() + snoozeDuration.value)
    until = d.toISOString()
  } else {
    message.warning('请选择暂缓时长')
    return
  }
  snoozeLoading.value = true
  try {
    await incidentApi.snooze(incidentId.value, until)
    message.success('暂缓成功')
    showSnooze.value = false
    snoozeDuration.value = null
    snoozeCustomUntil.value = ''
    await load()
  } catch (e: any) {
    message.error(e?.message ?? '操作失败')
  } finally {
    snoozeLoading.value = false
  }
}

// ---- Merge modal ----
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
  } catch (e: any) {
    message.error(e?.message ?? '搜索失败')
  } finally {
    mergeSearchLoading.value = false
  }
}

async function doMerge() {
  if (!mergeTargetId.value) {
    message.warning('请选择目标故障')
    return
  }
  mergeLoading.value = true
  try {
    await incidentApi.merge(incidentId.value, mergeTargetId.value)
    message.success('合并成功，当前故障已并入目标故障')
    showMerge.value = false
    router.push(`/incidents/${mergeTargetId.value}`)
  } catch (e: any) {
    message.error(e?.message ?? '操作失败')
  } finally {
    mergeLoading.value = false
  }
}

// ---- Reassign modal ----
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
          (u.display_name?.toLowerCase().includes(q))
        )
      : allUsers
  } catch (e: any) {
    message.error(e?.message ?? '搜索失败')
  } finally {
    reassignSearchLoading.value = false
  }
}

async function doReassign() {
  if (!reassignUserId.value) {
    message.warning('请选择处理人')
    return
  }
  reassignLoading.value = true
  try {
    await incidentApi.reassign(incidentId.value, reassignUserId.value)
    message.success('重新分派成功')
    showReassign.value = false
    reassignUserId.value = null
    await load()
  } catch (e: any) {
    message.error(e?.message ?? '操作失败')
  } finally {
    reassignLoading.value = false
  }
}

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

// --- Post-mortem ---
async function loadPostMortem() {
  pmLoading.value = true
  try {
    const res = await incidentApi.getPostMortem(incidentId.value)
    postMortem.value = res.data.data ?? null
  } catch {
    postMortem.value = null
  } finally {
    pmLoading.value = false
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
  } catch (e: any) {
    message.error(e?.message ?? t('common.saveFailed'))
  } finally {
    pmSaving.value = false
  }
}

async function publishPostMortem() {
  pmSaving.value = true
  try {
    const res = await incidentApi.publishPostMortem(incidentId.value)
    postMortem.value = res.data.data
    message.success(t('postMortem.published'))
  } catch (e: any) {
    message.error(e?.message ?? t('common.failed'))
  } finally {
    pmSaving.value = false
  }
}

async function aiGeneratePostMortem() {
  pmAiLoading.value = true
  try {
    const res = await incidentApi.aiGeneratePostMortem(incidentId.value)
    postMortem.value = res.data.data
    message.success('AI 初稿已生成')
  } catch (e: any) {
    message.error(e?.message ?? 'AI 生成失败')
  } finally {
    pmAiLoading.value = false
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

onMounted(async () => {
  await load()
  await loadPostMortem()
  await searchUsers()
})
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
            <n-space wrap>
              <n-tag :type="statusTagType[incident.status] ?? 'default'" size="medium">
                {{ t(statusLabel[incident.status] ?? incident.status) }}
              </n-tag>
              <n-tag :type="severityTagType[incident.severity] ?? 'default'" size="medium">
                {{ t(severityLabel[incident.severity] ?? incident.severity) }}
              </n-tag>

              <!-- Acknowledge -->
              <n-button
                v-if="incident.status === 'triggered'"
                type="primary" size="small"
                @click="doAction('acknowledge')"
              >{{ t('incident.acknowledge') }}</n-button>

              <!-- Close -->
              <n-button
                v-if="incident.status !== 'closed'"
                size="small"
                @click="doAction('close')"
              >{{ t('incident.close') }}</n-button>

              <!-- Reopen -->
              <n-button
                v-if="incident.status === 'closed'"
                size="small"
                @click="doAction('reopen')"
              >{{ t('incident.reopen') }}</n-button>

              <!-- Escalate -->
              <n-button size="small" @click="doAction('escalate')">
                {{ t('incident.escalate') }}
              </n-button>

              <!-- Snooze -->
              <n-button
                v-if="incident.status !== 'closed'"
                size="small"
                @click="showSnooze = true"
              >
                <template #icon><n-icon :component="TimeOutline" /></template>
                暂缓
              </n-button>

              <!-- Reassign -->
              <n-button size="small" @click="showReassign = true">
                <template #icon><n-icon :component="PersonOutline" /></template>
                重新分派
              </n-button>

              <!-- Merge -->
              <n-button size="small" @click="showMerge = true">
                <template #icon><n-icon :component="GitMergeOutline" /></template>
                合并故障
              </n-button>

              <!-- Quick Silence -->
              <n-button size="small" type="warning" @click="showQuickSilence = true">
                <template #icon><n-icon :component="VolumeOffOutline" /></template>
                快速静默
              </n-button>
            </n-space>
          </n-card>

          <!-- Tabs -->
          <n-card :bordered="false" class="tabs-card">
            <n-tabs v-model:value="activeTab" type="line" animated>

              <!-- Overview -->
              <n-tab-pane name="overview" tab="Overview">
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

              <!-- Post-mortem tab -->
              <n-tab-pane name="postmortem" :tab="t('postMortem.tab')">
                <n-spin :show="pmLoading">
                  <div v-if="postMortem" class="pm-container">
                    <!-- Toolbar -->
                    <div class="pm-toolbar">
                      <div class="pm-meta">
                        <n-tag :type="postMortem.status === 'published' ? 'success' : 'default'" size="small">
                          {{ postMortem.status === 'published' ? t('postMortem.published') : t('postMortem.draft') }}
                        </n-tag>
                        <span v-if="postMortem.updated_at" style="font-size:11px;color:var(--sre-text-secondary)">
                          {{ t('postMortem.lastUpdated') }}: {{ formatTime(postMortem.updated_at) }}
                        </span>
                      </div>
                      <n-space size="small">
                        <n-button
                          size="small"
                          :loading="pmAiLoading"
                          @click="aiGeneratePostMortem"
                        >
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

                    <!-- Title -->
                    <n-input
                      v-model:value="postMortem.title"
                      size="small"
                      style="margin-bottom:12px;font-weight:600"
                    />

                    <!-- Markdown editor with preview -->
                    <MdEditor
                      v-model="postMortem.content"
                      :preview="true"
                      :toolbars-exclude="['github']"
                      language="zh-CN"
                      style="height: 500px; border-radius: 8px"
                    />
                  </div>
                  <n-empty v-else :description="t('postMortem.noPostMortem')" style="padding:40px 0">
                    <template #extra>
                      <n-button type="primary" size="small" @click="loadPostMortem">
                        {{ t('common.create') }}
                      </n-button>
                    </template>
                  </n-empty>
                </n-spin>
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
              <n-descriptions-item v-if="incident.assigned_user" :label="t('incident.assignee')">
                {{ incident.assigned_user.display_name ?? incident.assigned_user.username }}
              </n-descriptions-item>
            </n-descriptions>
          </n-card>
        </div>
      </div>
    </n-spin>

    <!-- Quick Silence Modal -->
    <QuickSilenceModal
      v-model:show="showQuickSilence"
      :labels="incident?.labels ?? {}"
      :title="incident?.title"
      @created="load"
    />

    <!-- Snooze Modal -->
    <n-modal
      v-model:show="showSnooze"
      title="暂缓故障"
      preset="card"
      style="width: 400px"
      :bordered="false"
    >
      <div class="snooze-presets">
        <n-button
          v-for="p in snoozePresets"
          :key="p.minutes"
          :type="snoozeDuration === p.minutes ? 'primary' : 'default'"
          size="small"
          @click="snoozeDuration = p.minutes"
        >{{ p.label }}</n-button>
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
          <n-button @click="showSnooze = false">取消</n-button>
          <n-button type="primary" :loading="snoozeLoading" @click="doSnooze">确认暂缓</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Merge Modal -->
    <n-modal
      v-model:show="showMerge"
      title="合并到目标故障"
      preset="card"
      style="width: 520px"
      :bordered="false"
    >
      <p style="font-size:13px;color:var(--sre-text-secondary);margin-bottom:12px">
        将当前故障 <strong>#{{ incident?.id }}</strong> 的所有告警并入另一个故障，当前故障将被关闭。
      </p>
      <n-input-group>
        <n-input
          v-model:value="mergeSearch"
          placeholder="搜索故障 ID 或标题…"
          @keydown.enter="searchMergeIncidents"
        />
        <n-button :loading="mergeSearchLoading" @click="searchMergeIncidents">搜索</n-button>
      </n-input-group>
      <n-list v-if="mergeResults.length" style="margin-top:12px;max-height:240px;overflow-y:auto">
        <n-list-item
          v-for="inc in mergeResults"
          :key="inc.id"
          :class="{ 'selected-item': mergeTargetId === inc.id }"
          style="cursor:pointer;padding:8px 12px;border-radius:6px"
          @click="mergeTargetId = inc.id"
        >
          <n-space align="center">
            <n-tag :type="inc.severity === 'critical' ? 'error' : inc.severity === 'warning' ? 'warning' : 'info'" size="tiny">
              {{ inc.severity.toUpperCase() }}
            </n-tag>
            <span style="font-size:13px"><strong>#{{ inc.id }}</strong> {{ inc.title }}</span>
          </n-space>
        </n-list-item>
      </n-list>
      <n-empty v-else-if="mergeSearch && !mergeSearchLoading" description="无匹配故障" style="padding:16px 0" />
      <template #footer>
        <n-space justify="end">
          <n-button @click="showMerge = false">取消</n-button>
          <n-popconfirm @positive-click="doMerge">
            <template #trigger>
              <n-button type="error" :loading="mergeLoading" :disabled="!mergeTargetId">
                确认合并
              </n-button>
            </template>
            合并后当前故障将关闭，操作不可撤销，确认？
          </n-popconfirm>
        </n-space>
      </template>
    </n-modal>

    <!-- Reassign Modal -->
    <n-modal
      v-model:show="showReassign"
      title="重新分派"
      preset="card"
      style="width: 440px"
      :bordered="false"
    >
      <n-input
        v-model:value="reassignSearch"
        placeholder="搜索用户名或姓名…"
        clearable
        style="margin-bottom:12px"
        @update:value="searchUsers"
      />
      <n-spin :show="reassignSearchLoading">
        <n-list style="max-height:260px;overflow-y:auto">
          <n-list-item
            v-for="u in reassignUsers"
            :key="u.id"
            :class="{ 'selected-item': reassignUserId === u.id }"
            style="cursor:pointer;padding:8px 12px;border-radius:6px"
            @click="reassignUserId = u.id"
          >
            <n-space align="center">
              <n-avatar size="small" round>
                {{ (u.display_name || u.username).charAt(0).toUpperCase() }}
              </n-avatar>
              <div>
                <div style="font-size:13px;font-weight:500">{{ u.display_name || u.username }}</div>
                <div style="font-size:11px;color:var(--sre-text-secondary)">{{ u.username }}</div>
              </div>
            </n-space>
          </n-list-item>
        </n-list>
      </n-spin>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showReassign = false">取消</n-button>
          <n-button type="primary" :loading="reassignLoading" :disabled="!reassignUserId" @click="doReassign">
            确认分派
          </n-button>
        </n-space>
      </template>
    </n-modal>
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

/* Post-mortem */
.pm-container { display: flex; flex-direction: column; gap: 10px; }

.pm-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--sre-border);
}

.pm-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Snooze presets */
.snooze-presets {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

/* Selected list item */
.selected-item {
  background: var(--sre-primary-alpha, rgba(99,102,241,0.1));
  outline: 1px solid var(--sre-primary);
}
</style>
