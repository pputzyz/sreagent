<script setup lang="ts">
import { ref, shallowRef, reactive, computed, onMounted } from 'vue'
import {
  useMessage, useDialog, NButton, NIcon, NSwitch, NDropdown,
  NRadioGroup, NRadioButton, NInput, NSelect, NSpin, NModal, NForm,
  NFormItem, NGrid, NGi, NDatePicker, NTimePicker, NCheckboxGroup, NCheckbox,
  NSpace, NDivider,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import {
  AddOutline, RefreshOutline, EyeOutline, EllipsisHorizontalOutline,
  CreateOutline, TrashOutline, SearchOutline, SparklesOutline,
} from '@vicons/ionicons5'
import { muteRuleApi } from '@/api'
import type { MuteRule } from '@/types'
import type { RuleGenerateResult, MuteRuleGenerateResult } from '@/types/ai-module'
import { getErrorMessage } from '@/utils/format'
import { formatTime } from '@/utils/format'
import LabelMatcherEditor from '@/components/common/LabelMatcherEditor.vue'
import type { LabelMatcher } from '@/components/common/LabelMatcherEditor.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import AIGenerateModal from '@/components/alert-rule/AIGenerateModal.vue'
import {
  ruleType, isActiveNow, isFuture, isExpired, statusToSev, statusKey,
  getHitCount, remainingMin, relTimeFuture, describePeriodic,
} from './utils'

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

const loading = ref(false)
const rules = shallowRef<MuteRule[]>([])
const statusFilter = ref<'all' | 'active' | 'future' | 'expired' | 'disabled'>('all')
const searchKeyword = ref('')
const typeFilter = ref<'all' | 'once' | 'periodic'>('all')

// ---------- type helpers (imported from ./utils) ----------
function statusText(r: MuteRule): string {
  return t(statusKey(r))
}

function dayMapForPeriodic(): Record<string, string> {
  return {
    '0': t('mute.sunday'), '1': t('mute.monday'), '2': t('mute.tuesday'),
    '3': t('mute.wednesday'), '4': t('mute.thursday'), '5': t('mute.friday'),
    '6': t('mute.saturday'),
  }
}

function describePeriodicLocalized(r: MuteRule): string {
  return describePeriodic(r, dayMapForPeriodic(), t('mute.daily'))
}

// ---------- filter ----------
const filteredRules = computed(() => {
  const kw = searchKeyword.value.trim().toLowerCase()
  return rules.value.filter(r => {
    if (statusFilter.value === 'active' && !isActiveNow(r)) return false
    if (statusFilter.value === 'future' && !isFuture(r)) return false
    if (statusFilter.value === 'expired' && !isExpired(r)) return false
    if (statusFilter.value === 'disabled' && r.is_enabled) return false
    if (typeFilter.value !== 'all' && ruleType(r) !== typeFilter.value) return false
    if (kw && !r.name.toLowerCase().includes(kw) && !(r.description || '').toLowerCase().includes(kw)) return false
    return true
  })
})

const typeOptions = [
  { label: () => t('mute.allTypes'), value: 'all' },
  { label: () => t('mute.oneTime'), value: 'once' },
  { label: () => t('mute.periodic'), value: 'periodic' },
]

// ---------- API ----------
async function fetchRules() {
  loading.value = true
  try {
    const { data } = await muteRuleApi.list({ page: 1, page_size: 200 })
    rules.value = data.data.list || []
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function toggle(rule: MuteRule) {
  try {
    await muteRuleApi.toggle(rule.id, !rule.is_enabled)
    message.success(rule.is_enabled ? t('mute.disabledSuccess') : t('mute.enabledSuccess'))
    fetchRules()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

async function handleDelete(id: number) {
  try {
    await muteRuleApi.delete(id)
    message.success(t('mute.deleted'))
    fetchRules()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

function rowActions(_rule: MuteRule) {
  return [
    { label: t('common.edit'), key: 'edit', icon: () => h(NIcon, { component: CreateOutline }) },
    { label: t('common.delete'), key: 'delete', icon: () => h(NIcon, { component: TrashOutline }) },
  ]
}

import { h } from 'vue'

function handleAction(key: string, rule: MuteRule) {
  if (key === 'edit') openEdit(rule)
  if (key === 'delete') {
    dialog.warning({
      title: t('common.confirmDelete'),
      content: t('common.confirmDeleteMsg'),
      positiveText: t('common.confirmDelete'),
      negativeText: t('common.cancel'),
      onPositiveClick: () => handleDelete(rule.id),
    })
  }
}

// ---------- inline preview ----------
const expandedRuleId = ref<number | null>(null)
const previewLoading = ref(false)
interface PreviewAlert { alert_name: string; severity: string; labels: Record<string, string>; firing_at: string }
const previewItems = ref<PreviewAlert[]>([])

function isPreviewOpen(rule: MuteRule): boolean {
  return expandedRuleId.value === rule.id
}

async function togglePreview(rule: MuteRule) {
  if (expandedRuleId.value === rule.id) {
    expandedRuleId.value = null
    previewItems.value = []
    return
  }
  expandedRuleId.value = rule.id
  previewLoading.value = true
  try {
    const { data } = await muteRuleApi.previewOne(rule.id)
    previewItems.value = data.data?.matched_alerts || []
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    previewLoading.value = false
  }
}

// ---------- modal ----------
const showModal = ref(false)
const modalTitle = ref('')
const editingId = ref<number | null>(null)
const saving = ref(false)

const defaultForm = {
  name: '', description: '',
  match_labels: [] as LabelMatcher[],
  severities: [] as string[],
  start_time: null as number | null, end_time: null as number | null,
  periodic_start: '', periodic_end: '',
  days_of_week: [] as string[],
  timezone: 'Asia/Shanghai',
  rule_ids: '', is_enabled: true,
}
const form = reactive({ ...defaultForm })

const severityOptions = [
  { label: () => t('alert.critical'), value: 'critical' },
  { label: () => t('alert.warning'), value: 'warning' },
  { label: () => t('alert.info'), value: 'info' },
]
const daysOfWeekOptions = [
  { label: () => t('mute.monday'), value: '1' }, { label: () => t('mute.tuesday'), value: '2' },
  { label: () => t('mute.wednesday'), value: '3' }, { label: () => t('mute.thursday'), value: '4' },
  { label: () => t('mute.friday'), value: '5' }, { label: () => t('mute.saturday'), value: '6' },
  { label: () => t('mute.sunday'), value: '0' },
]
const timezoneOptions = [
  { label: 'Asia/Shanghai (CST)', value: 'Asia/Shanghai' },
  { label: 'Asia/Tokyo (JST)', value: 'Asia/Tokyo' },
  { label: 'America/New_York (EST)', value: 'America/New_York' },
  { label: 'Europe/London (GMT)', value: 'Europe/London' },
  { label: 'UTC', value: 'UTC' },
]

function parseSev(s: string) { return !s ? [] : s.split(',').map(x => x.trim()).filter(Boolean) }
function parseDays(s: string) { return !s ? [] : s.split(',').map(x => x.trim()).filter(Boolean) }

function resetForm() { Object.assign(form, defaultForm, { match_labels: [], severities: [], days_of_week: [] }) }

function openCreate() {
  editingId.value = null
  modalTitle.value = t('mute.create')
  resetForm()
  showModal.value = true
}

function openEdit(rule: MuteRule) {
  editingId.value = rule.id
  modalTitle.value = t('mute.edit')
  Object.assign(form, {
    name: rule.name, description: rule.description || '',
    match_labels: Object.entries(rule.match_labels || {}).map(([key, raw]) => {
      for (const op of ['!=', '=~', '!~'] as const) {
        if (raw.startsWith(op)) return { key, op, value: raw.slice(op.length) }
      }
      return { key, op: '=' as const, value: raw }
    }),
    severities: parseSev(rule.severities),
    start_time: rule.start_time ? new Date(rule.start_time).getTime() : null,
    end_time: rule.end_time ? new Date(rule.end_time).getTime() : null,
    periodic_start: rule.periodic_start || '', periodic_end: rule.periodic_end || '',
    days_of_week: parseDays(rule.days_of_week),
    timezone: rule.timezone || 'Asia/Shanghai',
    rule_ids: rule.rule_ids || '', is_enabled: rule.is_enabled,
  })
  showModal.value = true
}

function goEdit(rule: MuteRule) { openEdit(rule) }

async function handleSave() {
  if (!form.name.trim()) { message.warning(t('mute.nameRequired')); return }
  saving.value = true
  try {
    const payload: Partial<MuteRule> = {
      name: form.name, description: form.description,
      match_labels: Object.fromEntries(form.match_labels.map(m => {
        const v = m.op === '=' ? m.value : `${m.op}${m.value}`
        return [m.key, v]
      })),
      severities: form.severities.join(','),
      start_time: form.start_time ? new Date(form.start_time).toISOString() : null,
      end_time: form.end_time ? new Date(form.end_time).toISOString() : null,
      periodic_start: form.periodic_start, periodic_end: form.periodic_end,
      days_of_week: form.days_of_week.join(','),
      timezone: form.timezone, rule_ids: form.rule_ids, is_enabled: form.is_enabled,
    }
    if (editingId.value) {
      await muteRuleApi.update(editingId.value, payload)
      message.success(t('mute.updated'))
    } else {
      await muteRuleApi.create(payload)
      message.success(t('mute.created'))
    }
    showModal.value = false
    fetchRules()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

onMounted(fetchRules)

// ---------- AI Mute Generation ----------
const showAIModal = ref(false)

function openAIGenerate() {
  showAIModal.value = true
}

function handleAIGenerated(result: RuleGenerateResult | MuteRuleGenerateResult) {
  const muteResult = result as MuteRuleGenerateResult
  editingId.value = null
  modalTitle.value = t('mute.create')
  resetForm()
  form.name = muteResult.name || ''
  form.description = muteResult.description || ''
  form.match_labels = Object.entries(muteResult.match_labels || {}).map(([key, value]) => ({
    key, op: '=' as const, value: String(value),
  }))
  form.severities = muteResult.severities || []
  if (muteResult.periodic_start) form.periodic_start = muteResult.periodic_start
  if (muteResult.periodic_end) form.periodic_end = muteResult.periodic_end
  form.days_of_week = muteResult.days_of_week || []
  if (muteResult.start_time) form.start_time = new Date(muteResult.start_time).getTime()
  if (muteResult.end_time) form.end_time = new Date(muteResult.end_time).getTime()
  if (muteResult.timezone) form.timezone = muteResult.timezone
  showModal.value = true
}
</script>

<template>
  <div class="mute-page sre-stagger">
    <PageHeader :title="t('mute.title')" :subtitle="t('mute.subtitle')">
      <template #actions>
        <NButton quaternary @click="fetchRules" :loading="loading">
          <template #icon><NIcon :component="RefreshOutline" /></template>
          {{ t('common.refresh') }}
        </NButton>
        <NButton quaternary type="info" @click="openAIGenerate">
          <template #icon><NIcon :component="SparklesOutline" /></template>
          {{ t('mute.aiGenerate') }}
        </NButton>
        <NButton type="primary" @click="openCreate">
          <template #icon><NIcon :component="AddOutline" /></template>
          {{ t('mute.create') }}
        </NButton>
      </template>
    </PageHeader>

    <div class="mute-toolbar">
      <NRadioGroup v-model:value="statusFilter" size="small">
        <NRadioButton value="all">{{ t('common.all') }}</NRadioButton>
        <NRadioButton value="active">{{ t('common.active') }}</NRadioButton>
        <NRadioButton value="future">{{ t('mute.schedule') }}</NRadioButton>
        <NRadioButton value="expired">{{ t('common.expired') }}</NRadioButton>
        <NRadioButton value="disabled">{{ t('common.disabled') }}</NRadioButton>
      </NRadioGroup>
      <div class="mute-filters">
        <NInput v-model:value="searchKeyword" size="small" :placeholder="t('common.search')" clearable class="mute-search">
          <template #prefix><NIcon :component="SearchOutline" /></template>
        </NInput>
        <NSelect v-model:value="typeFilter" size="small" :options="typeOptions" class="mute-type-select" />
      </div>
    </div>

    <LoadingSkeleton v-if="loading && filteredRules.length === 0" :rows="6" variant="row" />
    <EmptyState
      v-else-if="!loading && filteredRules.length === 0"
      :icon="AddOutline"
      :title="t('mute.noData')"
      :description="t('mute.subtitle')"
      :primary-text="t('mute.createFirst')"
      @primary="openCreate"
    />
    <NSpin v-else :show="loading">
      <div class="mute-list">
        <div v-for="rule in filteredRules" :key="rule.id" class="mute-rule-group">
          <div
            class="sre-row-card mute-row sre-lift"
            :data-dim="!rule.is_enabled || undefined"
            @click="goEdit(rule)"
          >
            <div class="mute-main">
              <div class="mute-headline">
                <span class="sre-dot" :data-severity="statusToSev(rule)"></span>
                <span class="mute-status-label">{{ statusText(rule) }}</span>
                <span class="mute-name">{{ rule.name }}</span>
              </div>
              <div v-if="Object.keys(rule.match_labels || {}).length" class="mute-match">
                <span class="sre-label-eyebrow">{{ t('mute.matchLabel') }}</span>
                <span v-for="(v, k) in rule.match_labels" :key="k" class="mute-chip">{{ k }}={{ v }}</span>
              </div>
              <div class="mute-schedule">
                <span class="sre-label-eyebrow">{{ t('mute.schedule') }}</span>
                <span v-if="ruleType(rule) === 'once'">{{ t('mute.oneTime') }} {{ formatTime(rule.start_time) }} → {{ formatTime(rule.end_time) }}</span>
                <span v-else-if="ruleType(rule) === 'periodic'">{{ t('mute.periodic') }} {{ describePeriodicLocalized(rule) }}</span>
                <span v-else class="muted">{{ t('mute.noSchedule') }}</span>
              </div>
              <div class="mute-footer tnum">
                <span>{{ t('mute.hits', { n: getHitCount(rule) }) }}</span>
                <span class="sre-meta-divider"></span>
                <span v-if="!rule.is_enabled">{{ t('common.disabled') }}</span>
                <span v-else-if="isActiveNow(rule)">{{ t('mute.statusActive') }} · {{ t('mute.remaining', { n: remainingMin(rule) }) }}</span>
                <span v-else-if="isFuture(rule)">{{ t('mute.startsIn', { n: relTimeFuture(rule.start_time) }) }}</span>
                <span v-else-if="isExpired(rule)">{{ t('common.expired') }}</span>
                <span v-else>{{ t('common.idle') }}</span>
              </div>
            </div>
            <div class="mute-actions" @click.stop>
              <NButton size="tiny" quaternary :type="isPreviewOpen(rule) ? 'primary' : 'default'" @click="togglePreview(rule)">
                <template #icon><NIcon :component="EyeOutline" /></template>
                {{ t('mute.previewBtn') }}
              </NButton>
              <NSwitch :value="rule.is_enabled" size="small" @update:value="toggle(rule)" />
              <NDropdown :options="rowActions(rule)" trigger="click" @select="(k: string) => handleAction(k, rule)">
                <NButton quaternary circle size="small">
                  <template #icon><NIcon :component="EllipsisHorizontalOutline" /></template>
                </NButton>
              </NDropdown>
            </div>
          </div>
          <!-- Inline preview panel -->
          <div v-if="isPreviewOpen(rule)" class="mute-inline-preview">
            <div class="mute-inline-preview-header">
              <span class="sre-label-eyebrow">{{ t('tooltip.willBeMuted') }}</span>
            </div>
            <NSpin :show="previewLoading">
              <div v-if="!previewLoading && previewItems.length === 0" class="mute-preview-empty">
                <EmptyState
                  :icon="AddOutline"
                  :title="t('mute.previewNoMatch')"
                  size="sm"
                />
              </div>
              <div v-else class="mute-preview-list">
                <div
                  v-for="(ev, idx) in previewItems" :key="idx"
                  class="sre-row-card mute-preview-row"
                  :data-severity="(ev.severity || 'info').toLowerCase()"
                >
                  <div class="mute-preview-main">
                    <div class="mute-preview-head">
                      <span class="sre-dot" :data-severity="(ev.severity || 'info').toLowerCase()"></span>
                      <span class="mute-preview-sev">{{ (ev.severity || 'info').toUpperCase() }}</span>
                      <span class="mute-preview-name">{{ ev.alert_name }}</span>
                    </div>
                    <div v-if="ev.labels && Object.keys(ev.labels).length" class="mute-preview-desc">
                      <span v-for="(v, k) in ev.labels" :key="k" class="mute-chip">{{ k }}={{ v }}</span>
                    </div>
                    <div class="mute-preview-meta tnum">
                      <span v-if="ev.firing_at">{{ formatTime(ev.firing_at) }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </NSpin>
          </div>
        </div>
      </div>
    </NSpin>


    <!-- Create/Edit Modal -->
    <NModal v-model:show="showModal" preset="card" :title="modalTitle" style="width: 720px" :bordered="false">
      <NForm label-placement="top">
        <NGrid :x-gap="12" :cols="2">
          <NGi>
            <NFormItem :label="t('mute.name')" required>
              <NInput v-model:value="form.name" :placeholder="t('mute.name')" />
            </NFormItem>
          </NGi>
          <NGi>
            <NFormItem :label="t('mute.severities')">
              <NSelect v-model:value="form.severities" :options="severityOptions" multiple />
            </NFormItem>
          </NGi>
        </NGrid>
        <NFormItem :label="t('mute.description')">
          <NInput v-model:value="form.description" type="textarea" :rows="2" />
        </NFormItem>
        <NFormItem :label="t('mute.matchLabels')">
          <LabelMatcherEditor v-model:modelValue="form.match_labels" :add-label="t('mute.addLabel')" />
        </NFormItem>
        <NDivider class="mute-divider">{{ t('mute.oneTimeMute') }}</NDivider>
        <NGrid :x-gap="12" :cols="2">
          <NGi>
            <NFormItem :label="t('mute.startTime')">
              <NDatePicker v-model:value="form.start_time" type="datetime" clearable class="mute-input-full" />
            </NFormItem>
          </NGi>
          <NGi>
            <NFormItem :label="t('mute.endTime')">
              <NDatePicker v-model:value="form.end_time" type="datetime" clearable class="mute-input-full" />
            </NFormItem>
          </NGi>
        </NGrid>
        <NDivider class="mute-divider">{{ t('mute.periodicMute') }}</NDivider>
        <NGrid :x-gap="12" :cols="2">
          <NGi>
            <NFormItem :label="t('mute.periodicStart')">
              <NTimePicker v-model:formatted-value="form.periodic_start" value-format="HH:mm" format="HH:mm" clearable class="mute-input-full" />
            </NFormItem>
          </NGi>
          <NGi>
            <NFormItem :label="t('mute.periodicEnd')">
              <NTimePicker v-model:formatted-value="form.periodic_end" value-format="HH:mm" format="HH:mm" clearable class="mute-input-full" />
            </NFormItem>
          </NGi>
        </NGrid>
        <NFormItem :label="t('mute.daysOfWeek')">
          <NCheckboxGroup v-model:value="form.days_of_week">
            <NSpace>
              <NCheckbox v-for="day in daysOfWeekOptions" :key="day.value" :value="day.value" :label="day.label()" />
            </NSpace>
          </NCheckboxGroup>
        </NFormItem>
        <NGrid :x-gap="12" :cols="2">
          <NGi>
            <NFormItem :label="t('mute.timezone')">
              <NSelect v-model:value="form.timezone" :options="timezoneOptions" filterable />
            </NFormItem>
          </NGi>
          <NGi>
            <NFormItem :label="t('mute.ruleIds')">
              <NInput v-model:value="form.rule_ids" :placeholder="t('muteMgmt.ruleIdsPlaceholder')" />
            </NFormItem>
          </NGi>
        </NGrid>
        <NFormItem :label="t('common.status')">
          <NSwitch v-model:value="form.is_enabled" />
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace justify="end">
          <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="saving" @click="handleSave">
            {{ editingId ? t('common.update') : t('common.create') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- AI Generate Modal -->
    <AIGenerateModal
      v-model:visible="showAIModal"
      rule-type="mute"
      @generated="handleAIGenerated"
    />
  </div>
</template>

<style scoped>
.mute-page { max-width: 1280px; font-family: var(--sre-font-sans); }

.mute-toolbar {
  display: flex; align-items: center; justify-content: space-between;
  margin: 12px 0 14px; gap: 12px; flex-wrap: wrap;
}
.mute-filters { display: flex; align-items: center; gap: 8px; }
.mute-search { width: 220px; }
.mute-type-select { width: 140px; }

.mute-list { display: flex; flex-direction: column; gap: 8px; }
.mute-divider { margin: 12px 0; }
.mute-input-full { width: 100%; }
.mute-preview-empty { padding: 40px 0; text-align: center; }

.mute-inline-preview {
  margin-top: -4px;
  padding: 12px 18px 14px;
  border-top: 1px dashed var(--sre-border);
  background: var(--sre-bg-elevated);
  border-radius: 0 0 var(--sre-radius-lg, 8px) var(--sre-radius-lg, 8px);
}
.mute-inline-preview-header {
  margin-bottom: 8px;
}

.mute-row {
  padding: 14px 18px; gap: 12px; cursor: pointer;
  display: flex; align-items: flex-start;
}
.mute-row[data-dim] { opacity: 0.55; }

.mute-main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 6px; }
.mute-headline {
  display: flex; align-items: center; gap: 8px;
  font-size: 14px; font-weight: 600;
}
.mute-status-label {
  font-size: 11px; font-weight: 600; color: var(--sre-text-secondary);
  text-transform: uppercase; letter-spacing: 0.6px;
}
.mute-name { color: var(--sre-text-primary); }

.mute-match, .mute-schedule {
  display: flex; align-items: center; gap: 6px; flex-wrap: wrap;
  font-size: 12px; color: var(--sre-text-tertiary);
}
.mute-chip {
  font-family: var(--sre-font-mono); font-size: 11px;
  background: var(--sre-bg-elevated); border-radius: 4px;
  padding: 2px 6px; color: var(--sre-text-secondary);
  border: var(--sre-hairline);
}

.mute-footer {
  display: flex; align-items: center;
  font-size: 12px; color: var(--sre-text-tertiary);
}
.muted { color: var(--sre-text-tertiary); }

.mute-actions {
  display: flex; align-items: center; gap: 6px; flex-shrink: 0;
}

.mute-preview-list { display: flex; flex-direction: column; gap: 8px; }
.mute-preview-row { padding: 12px 14px; }
.mute-preview-main { display: flex; flex-direction: column; gap: 4px; }
.mute-preview-head { display: flex; align-items: center; gap: 8px; font-size: 13px; font-weight: 600; }
.mute-preview-sev {
  font-size: 11px; font-weight: 600; color: var(--sre-text-secondary);
  text-transform: uppercase; letter-spacing: 0.6px;
}
.mute-preview-name { color: var(--sre-text-primary); }
.mute-preview-desc { font-size: 12px; color: var(--sre-text-secondary); }
.mute-preview-meta {
  font-size: 11px; color: var(--sre-text-tertiary);
  display: flex; align-items: center;
}
</style>
