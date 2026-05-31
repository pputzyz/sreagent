<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, h, watch, type Ref } from 'vue'
import { useMessage, useDialog, NDropdown, NPagination } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { notifyRuleApi, notifyMediaApi } from '@/api'
import type { NotifyRule, NotifyMedia } from '@/types'
import { getErrorMessage } from '@/utils/format'
import { useCrudPage } from '@/composables/useCrudPage'
import type { CrudApiModule } from '@/composables/useCrudPage'
import { AddOutline, SearchOutline, FilterOutline, FlaskOutline } from '@vicons/ionicons5'
import EmptyState from '@/components/common/EmptyState.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import LabelMatcherEditor from '@/components/common/LabelMatcherEditor.vue'
import type { LabelMatcher } from '@/components/common/LabelMatcherEditor.vue'
import { recordToMatchers, matchersToRecord } from '@/utils/label-matcher'

interface RuleForm {
  name: string
  description: string
  severities: string[]
  match_labels: LabelMatcher[]
  pipeline: string
  notify_configs: string
  repeat_interval: number
  callback_url: string
  is_enabled: boolean
}

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

const crud = useCrudPage<NotifyRule>({
  api: notifyRuleApi as unknown as CrudApiModule<NotifyRule>,
  defaultForm: () => ({
    name: '', description: '', severities: [] as string[],
    match_labels: [] as LabelMatcher[],
    pipeline: '[]', notify_configs: '[]',
    repeat_interval: 3600, callback_url: '', is_enabled: true,
  } as unknown as Partial<NotifyRule>),
  i18nKeys: {
    created: 'notifyRule.created',
    updated: 'notifyRule.updated',
    deleted: 'notifyRule.deleted',
    deleteConfirm: 'notifyRule.deleteConfirm',
    createTitle: 'notifyRule.create',
    editTitle: 'notifyRule.edit',
  },
  rowToForm: (row) => ({
    name: row.name, description: row.description,
    severities: (row.severities || '').split(',').filter(Boolean),
    match_labels: recordToMatchers(row.match_labels),
    pipeline: row.pipeline || '[]',
    notify_configs: row.notify_configs || '[]',
    repeat_interval: row.repeat_interval,
    callback_url: row.callback_url || '',
    is_enabled: row.is_enabled,
  } as unknown as Partial<NotifyRule>),
  formToPayload: (form) => {
    const f = form as unknown as RuleForm
    return {
      name: form.name, description: form.description,
      severities: (f.severities || []).join(','),
      match_labels: matchersToRecord(f.match_labels || []),
      pipeline: f.pipeline,
      notify_configs: f.notify_configs,
      repeat_interval: f.repeat_interval,
      callback_url: f.callback_url,
      is_enabled: form.is_enabled,
    }
  },
  validate: (form) => {
    if (!form.name?.trim()) return t('notifyRule.nameRequired')
    const f = form as unknown as RuleForm
    try { JSON.parse(f.pipeline) } catch { return t('notifyRule.pipeline') + ': ' + t('notifyRule.invalidJson') }
    try { JSON.parse(f.notify_configs) } catch { return t('notifyRule.notifyConfigs') + ': ' + t('notifyRule.invalidJson') }
    return null
  },
  pageSize: 100,
})

const {
  loading,
  items: rules,
  total,
  page,
  pageSize,
  search,
  showModal,
  modalTitle,
  editingId,
  saving,
  fetchList,
  openCreate,
  openEdit,
  handleSave,
  confirmDelete,
} = crud
const form = crud.form as unknown as Ref<RuleForm>

const severityOptions = computed(() => [
  { label: t('alert.critical'), value: 'critical' },
  { label: t('alert.warning'), value: 'warning' },
  { label: t('alert.info'), value: 'info' },
])

// --- Structured editors for notify_configs and pipeline ---
const allMediaOptions = ref<Array<{ label: string; value: number; type: string }>>([])

interface NotifyConfigItem {
  media_id: number | null
  type: string
}

interface PipelineItem {
  pipeline_id: number | null
}

const notifyConfigItems = ref<NotifyConfigItem[]>([])
const pipelineItems = ref<PipelineItem[]>([])

async function loadMediaOptions() {
  try {
    const res = await notifyMediaApi.list({ page: 1, page_size: 500 })
    allMediaOptions.value = (res.data.data.list || []).map((m: NotifyMedia) => ({
      label: m.name,
      value: m.id,
      type: m.type,
    }))
  } catch { /* ignore */ }
}

function parseNotifyConfigs(json: string): NotifyConfigItem[] {
  try {
    const arr = JSON.parse(json || '[]')
    if (!Array.isArray(arr)) return []
    return arr.map((c: Record<string, unknown>) => ({
      media_id: (c.media_id as number) || null,
      type: (c.type as string) || '',
    }))
  } catch { return [] }
}

function serializeNotifyConfigs(items: NotifyConfigItem[]): string {
  return JSON.stringify(items.filter(c => c.media_id != null).map(c => {
    const opt = allMediaOptions.value.find(m => m.value === c.media_id)
    return { media_id: c.media_id, type: opt?.type || c.type }
  }))
}

function parsePipeline(json: string): PipelineItem[] {
  try {
    const arr = JSON.parse(json || '[]')
    if (!Array.isArray(arr)) return []
    return arr.map((p: Record<string, unknown>) => ({
      pipeline_id: (p.pipeline_id as number) || (typeof p === 'number' ? p as number : null),
    }))
  } catch { return [] }
}

function serializePipeline(items: PipelineItem[]): string {
  return JSON.stringify(items.filter(p => p.pipeline_id != null).map(p => ({ pipeline_id: p.pipeline_id })))
}

function addNotifyConfig() {
  notifyConfigItems.value.push({ media_id: null, type: '' })
}

function removeNotifyConfig(idx: number) {
  notifyConfigItems.value.splice(idx, 1)
}

function onMediaSelect(idx: number, mediaId: number) {
  const opt = allMediaOptions.value.find(m => m.value === mediaId)
  if (opt) notifyConfigItems.value[idx].type = opt.type
}

function addPipeline() {
  pipelineItems.value.push({ pipeline_id: null })
}

function removePipeline(idx: number) {
  pipelineItems.value.splice(idx, 1)
}

// Sync structured items back to form JSON strings
watch(notifyConfigItems, (items) => {
  form.value.notify_configs = serializeNotifyConfigs(items)
}, { deep: true })

watch(pipelineItems, (items) => {
  form.value.pipeline = serializePipeline(items)
}, { deep: true })

// Load media options on mount and parse form when modal opens
watch(showModal, (open) => {
  if (open) {
    loadMediaOptions()
    notifyConfigItems.value = parseNotifyConfigs(form.value.notify_configs)
    pipelineItems.value = parsePipeline(form.value.pipeline)
  }
})

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return rules.value
  return rules.value.filter(r =>
    r.name.toLowerCase().includes(q) ||
    (r.description || '').toLowerCase().includes(q),
  )
})

function severityDot(s: string) {
  return s === 'critical' ? 'critical' : s === 'warning' ? 'warning' : 'info'
}

function summarizeMedia(r: NotifyRule): string[] {
  try {
    const arr = JSON.parse(r.notify_configs || '[]')
    if (!Array.isArray(arr)) return []
    return arr.map((c: Record<string, unknown>) => (c.media_name || c.name || c.type || 'media') as string).slice(0, 4)
  } catch { return [] }
}

async function toggleEnabled(row: NotifyRule, val: boolean) {
  try {
    await notifyRuleApi.update(row.id, { is_enabled: val })
    rules.value = rules.value.map(r => r.id === row.id ? { ...r, is_enabled: val } : r)
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
}

function rowMenu(row: NotifyRule) {
  return [
    { label: t('common.edit'), key: 'edit' },
    { type: 'divider', key: 'd1' },
    { label: t('common.delete'), key: 'delete', props: { style: 'color: var(--sre-danger, #ef4444)' } },
  ]
}
function onRowMenu(key: string, row: NotifyRule) {
  if (key === 'edit') {
    openEdit(row)
  } else if (key === 'delete') {
    dialog.warning({
      title: t('common.confirmDelete'),
      content: t('notifyRule.deleteConfirm'),
      positiveText: t('common.confirm'),
      negativeText: t('common.cancel'),
      onPositiveClick: () => confirmDelete(row.id),
    })
  }
}
const RowMenu = (row: NotifyRule) => h(NDropdown, {
  trigger: 'click', options: rowMenu(row),
  onSelect: (k: string) => onRowMenu(k, row),
}, { default: () => h('button', { class: 'sre-icon-btn', 'aria-label': t('common.actions') }, h('span', { class: 'sre-dots' })) })

// --- Test Rule ---
const showTestModal = ref(false)
const testingRule = ref<NotifyRule | null>(null)
const testLoading = ref(false)
const testAlertName = ref('Test Alert')
const testSeverity = ref<string>('critical')
const testMediaId = ref<number | null>(null)
const testResults = ref<Array<{ media_id: number; media_name: string; status: string; error?: string }>>([])
const mediaOptions = ref<Array<{ label: string; value: number }>>([])

const testSeverityOptions = computed(() => [
  { label: t('alert.critical'), value: 'critical' },
  { label: t('alert.warning'), value: 'warning' },
  { label: t('alert.info'), value: 'info' },
])

async function openTestModal(rule: NotifyRule) {
  testingRule.value = rule
  testAlertName.value = 'Test Alert'
  testSeverity.value = 'critical'
  testMediaId.value = null
  testResults.value = []
  showTestModal.value = true
  // Load media options
  try {
    const res = await notifyMediaApi.list({ page: 1, page_size: 100 })
    mediaOptions.value = (res.data.data.list || []).map((m: NotifyMedia) => ({
      label: m.name,
      value: m.id,
    }))
  } catch { /* ignore */ }
}

async function handleTestRule() {
  if (!testingRule.value) return
  testLoading.value = true
  testResults.value = []
  try {
    const data: { alert_name?: string; severity?: string; media_id?: number } = {}
    if (testAlertName.value.trim()) data.alert_name = testAlertName.value.trim()
    if (testSeverity.value) data.severity = testSeverity.value
    if (testMediaId.value) data.media_id = testMediaId.value
    const res = await notifyRuleApi.test(testingRule.value.id, data)
    testResults.value = res.data.data || []
    if (testResults.value.length === 0) {
      message.success(t('notifyRule.testSent'))
    }
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('notifyRule.testFailed'))
  } finally {
    testLoading.value = false
  }
}

onMounted(fetchList)
</script>

<template>
  <div class="rules-page">
    <PageHeader :title="t('notifyRule.title')" :subtitle="t('notifyRule.subtitle')">
      <template #actions>
        <n-button type="primary" size="small" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('notifyRule.create') }}
        </n-button>
      </template>
    </PageHeader>

    <div class="toolbar">
      <n-input v-model:value="search" size="small" :placeholder="t('common.search')" clearable style="width: 240px">
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <span class="count tnum">{{ filtered.length }} / {{ rules.length }}</span>
    </div>

    <LoadingSkeleton v-if="loading" :rows="4" variant="row" />

    <EmptyState
      v-else-if="filtered.length === 0"
      :icon="FilterOutline"
      :title="t('notifyRule.noData')"
      :description="t('notifyRule.subtitle')"
      :primary-text="t('notifyRule.create')"
      @primary="openCreate"
    />

    <ul v-else class="row-list sre-stagger">
      <li v-for="r in filtered" :key="r.id" class="sre-notify-card sre-lift">
        <div class="row-l1">
          <span class="sre-dot" :class="r.is_enabled ? 'on' : 'off'"></span>
          <span class="row-name">{{ r.name }}</span>
          <div class="severities">
            <span v-for="s in (r.severities || '').split(',').filter(Boolean)" :key="s"
              class="sev-chip" :data-sev="severityDot(s)">{{ t('severity.' + s) }}</span>
          </div>
          <div class="row-actions">
            <n-button size="tiny" quaternary @click="openTestModal(r)">
              <template #icon><n-icon :component="FlaskOutline" /></template>
              {{ t('common.test') }}
            </n-button>
            <n-switch :value="r.is_enabled" size="small" :aria-label="r.is_enabled ? t('common.disable') : t('common.enable')" @update:value="(v: boolean) => toggleEnabled(r, v)" />
            <component :is="RowMenu(r)" />
          </div>
        </div>
        <div class="row-l2">
          <template v-for="(v, k) in (r.match_labels || {})" :key="k">
            <code class="label-chip">{{ k }}={{ v }}</code>
          </template>
          <span v-if="!Object.keys(r.match_labels || {}).length" class="muted">{{ t('common.noMatchLabels') || '—' }}</span>
          <span class="arrow">→</span>
          <span v-for="m in summarizeMedia(r)" :key="m" class="media-chip">{{ m }}</span>
        </div>
        <div class="row-l3">
          <span class="meta tnum">{{ t('notifyRule.repeatDisplay', { n: r.repeat_interval }) }}</span>
          <span class="sre-meta-divider">·</span>
          <span class="meta" v-if="r.description">{{ r.description }}</span>
        </div>
      </li>
    </ul>

    <div v-if="total > pageSize" class="pagination-wrap">
      <n-pagination
        v-model:page="page"
        :page-size="pageSize"
        :item-count="total"
        :page-slot="7"
        @update:page="fetchList"
      />
    </div>

    <n-modal v-model:show="showModal" preset="card" :title="modalTitle" :bordered="false" class="rules-modal">
      <n-form label-placement="top">
        <n-grid :x-gap="12" :cols="2">
          <n-gi>
            <n-form-item :label="t('notifyRule.name')" required>
              <n-input v-model:value="form.name" :placeholder="t('notifyRule.namePlaceholder')" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('common.enabled')">
              <n-switch v-model:value="form.is_enabled" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-form-item :label="t('notifyRule.description')">
          <n-input v-model:value="form.description" :placeholder="t('notifyRule.description')" />
        </n-form-item>

        <n-form-item :label="t('notifyRule.severities')">
          <n-select v-model:value="form.severities" :options="severityOptions" multiple
            :placeholder="t('common.selectSeverities')" />
        </n-form-item>

        <n-form-item :label="t('notifyRule.matchLabels')">
          <LabelMatcherEditor v-model:modelValue="form.match_labels" :add-label="t('notifyRule.addLabel')" />
        </n-form-item>

        <n-form-item :label="t('notifyRule.notifyConfigs')">
          <div style="width: 100%; display: flex; flex-direction: column; gap: 8px;">
            <div v-for="(item, idx) in notifyConfigItems" :key="idx" style="display: flex; align-items: center; gap: 8px;">
              <n-select
                :value="item.media_id"
                :options="allMediaOptions"
                :placeholder="t('notifyRule.selectMedia')"
                filterable
                style="flex: 1"
                @update:value="(v: number) => { notifyConfigItems[idx].media_id = v; onMediaSelect(idx, v) }"
              />
              <n-tag v-if="item.type" size="small" :bordered="false">{{ item.type }}</n-tag>
              <button class="sre-icon-btn" style="color: var(--sre-danger)" @click="removeNotifyConfig(idx)" :aria-label="t('common.delete')">
                <span class="sre-dots">&times;</span>
              </button>
            </div>
            <n-button dashed size="small" @click="addNotifyConfig" style="align-self: flex-start">
              {{ t('notifyRule.addMedia') }}
            </n-button>
          </div>
        </n-form-item>

        <n-form-item :label="t('notifyRule.pipeline')">
          <div style="width: 100%; display: flex; flex-direction: column; gap: 8px;">
            <div v-for="(item, idx) in pipelineItems" :key="idx" style="display: flex; align-items: center; gap: 8px;">
              <n-input-number
                :value="item.pipeline_id"
                :placeholder="t('notifyRule.pipelineIdPlaceholder')"
                :min="1"
                style="flex: 1"
                @update:value="(v: number | null) => pipelineItems[idx].pipeline_id = v"
              />
              <button class="sre-icon-btn" style="color: var(--sre-danger)" @click="removePipeline(idx)" :aria-label="t('common.delete')">
                <span class="sre-dots">&times;</span>
              </button>
            </div>
            <n-button dashed size="small" @click="addPipeline" style="align-self: flex-start">
              {{ t('notifyRule.addPipeline') }}
            </n-button>
          </div>
        </n-form-item>

        <n-grid :x-gap="12" :cols="2">
          <n-gi>
            <n-form-item :label="t('notifyRule.repeatInterval')">
              <n-input-number v-model:value="form.repeat_interval" :min="0" style="width: 100%" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('notifyRule.callbackUrl')">
              <n-input v-model:value="form.callback_url" :placeholder="t('notifyRule.callbackUrlPlaceholder')" />
            </n-form-item>
          </n-gi>
        </n-grid>
      </n-form>

      <template #action>
        <n-space justify="end">
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="handleSave">
            {{ editingId ? t('common.update') : t('common.create') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Test Rule Modal -->
    <n-modal v-model:show="showTestModal" preset="card" :title="t('notifyRule.testTitle')" :bordered="false" class="test-modal">
      <n-form label-placement="top">
        <n-form-item :label="t('notifyRule.testAlertName')">
          <n-input v-model:value="testAlertName" :placeholder="t('notifyRule.testAlertNamePlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('notifyRule.testSeverity')">
          <n-select v-model:value="testSeverity" :options="testSeverityOptions" />
        </n-form-item>
        <n-form-item :label="t('notifyRule.testChannel')">
          <n-select
            v-model:value="testMediaId"
            :options="mediaOptions"
            :placeholder="t('notifyRule.testChannelPlaceholder')"
            clearable
          />
        </n-form-item>
      </n-form>

      <!-- Test Results -->
      <div v-if="testResults.length > 0" class="test-results">
        <div class="test-results-title">{{ t('notifyRule.testResults') }}</div>
        <div v-for="r in testResults" :key="r.media_id" class="test-result-item">
          <span class="test-result-name">{{ r.media_name }}</span>
          <n-tag :type="r.status === 'success' ? 'success' : 'error'" size="small">
            {{ r.status === 'success' ? t('common.success') : t('common.failed') }}
          </n-tag>
          <span v-if="r.error" class="test-result-error">{{ r.error }}</span>
        </div>
      </div>

      <template #action>
        <n-space justify="end">
          <n-button @click="showTestModal = false">{{ t('common.close') }}</n-button>
          <n-button type="primary" :loading="testLoading" @click="handleTestRule">
            {{ t('notifyRule.testSend') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.rules-page { font-family: var(--sre-font-sans); max-width: 1400px; }

.sub-header {
  display: flex; align-items: flex-end; justify-content: space-between;
  padding-bottom: 14px; border-bottom: 1px solid var(--sre-hairline, rgba(255,255,255,0.06));
  margin-bottom: 14px;
}
.sub-title { font: 600 18px/1.2 var(--sre-font-sans), sans-serif; margin: 0; letter-spacing: -0.01em; }
.sub-sub { font-size: 12px; color: var(--sre-text-secondary, #888); margin: 4px 0 0; }

.toolbar { display: flex; gap: 8px; align-items: center; margin-bottom: 12px; }
.count { font-size: 12px; color: var(--sre-text-secondary, #888); margin-left: auto; font-variant-numeric: tabular-nums; }

.loading { padding: 60px 20px; text-align: center; color: var(--sre-text-secondary, #888); }

.row-list { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 6px; }

.row-l1 { display: flex; align-items: center; gap: 10px; }
.row-name { font: 600 14px/1.3 var(--sre-font-sans), sans-serif; letter-spacing: -0.005em; }

.severities { display: flex; gap: 4px; flex-wrap: wrap; }
/* .sev-chip styles are in global.css */

.row-actions { margin-left: auto; display: flex; align-items: center; gap: 6px; }

.row-l2 { padding-left: 18px; display: flex; flex-wrap: wrap; gap: 4px; align-items: center; }
.label-chip, .media-chip {
  font: 11px/1 var(--sre-font-mono), monospace; padding: 3px 6px; border-radius: 4px;
  background: var(--sre-bg-hover); color: var(--sre-text-secondary, #aaa);
}
.media-chip { background: var(--sre-accent-soft); color: var(--sre-accent); }
.arrow { color: var(--sre-text-secondary, #666); margin: 0 4px; font-size: 12px; }
.muted { color: var(--sre-text-secondary, #666); font-size: 12px; }

.row-l3 { padding-left: 18px; display: flex; gap: 6px; align-items: center; }
.meta { font-size: 12px; color: var(--sre-text-secondary, #888); }

.rules-modal { width: 600px; }
.test-modal { width: 480px; }
.test-results { margin-top: 16px; padding-top: 12px; border-top: 1px solid var(--sre-hairline, rgba(255,255,255,0.06)); }
.test-results-title { font-size: 13px; font-weight: 600; margin-bottom: 8px; color: var(--sre-text-primary); }
.test-result-item { display: flex; align-items: center; gap: 8px; padding: 6px 0; }
.test-result-name { font-size: 13px; color: var(--sre-text-primary); min-width: 120px; }
.test-result-error { font-size: 12px; color: var(--sre-danger, #ef4444); flex: 1; }

.pagination-wrap {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}
</style>
