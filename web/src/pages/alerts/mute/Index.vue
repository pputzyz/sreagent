<script setup lang="ts">
import { ref, shallowRef, reactive, computed, onMounted } from 'vue'
import {
  useMessage, NButton, NIcon, NSwitch, NDropdown, NDrawer, NDrawerContent,
  NRadioGroup, NRadioButton, NInput, NSelect, NEmpty, NSpin, NModal, NForm,
  NFormItem, NGrid, NGi, NDatePicker, NTimePicker, NCheckboxGroup, NCheckbox,
  NSpace, NDivider,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import {
  AddOutline, RefreshOutline, EyeOutline, EllipsisHorizontalOutline,
  CreateOutline, TrashOutline, SearchOutline,
} from '@vicons/ionicons5'
import { muteRuleApi } from '@/api'
import type { MuteRule } from '@/types'
import { formatTime } from '@/utils/format'
import LabelMatcherEditor from '@/components/common/LabelMatcherEditor.vue'
import type { LabelMatcher } from '@/components/common/LabelMatcherEditor.vue'
import PageHeader from '@/components/common/PageHeader.vue'

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const rules = shallowRef<MuteRule[]>([])
const statusFilter = ref<'all' | 'active' | 'future' | 'expired' | 'disabled'>('all')
const searchKeyword = ref('')
const typeFilter = ref<'all' | 'once' | 'periodic'>('all')

// ---------- type helpers ----------
function ruleType(r: MuteRule): 'once' | 'periodic' | 'unknown' {
  if (r.start_time && r.end_time) return 'once'
  if (r.periodic_start && r.periodic_end) return 'periodic'
  return 'unknown'
}

function toMs(t: string | null | undefined): number | null {
  if (!t) return null
  const ms = new Date(t).getTime()
  return Number.isNaN(ms) ? null : ms
}

function isActiveNow(r: MuteRule): boolean {
  if (!r.is_enabled) return false
  const now = Date.now()
  if (ruleType(r) === 'once') {
    const s = toMs(r.start_time), e = toMs(r.end_time)
    return !!(s && e && now >= s && now <= e)
  }
  if (ruleType(r) === 'periodic') {
    // best-effort; just check current time within HH:mm window and weekday
    const d = new Date()
    const cur = d.getHours() * 60 + d.getMinutes()
    const [sh, sm] = (r.periodic_start || '0:0').split(':').map(Number)
    const [eh, em] = (r.periodic_end || '0:0').split(':').map(Number)
    const s = sh * 60 + sm, e = eh * 60 + em
    const inWindow = s <= e ? (cur >= s && cur <= e) : (cur >= s || cur <= e)
    if (!inWindow) return false
    const days = (r.days_of_week || '').split(',').map(x => x.trim()).filter(Boolean)
    if (days.length === 0) return true
    return days.includes(String(d.getDay()))
  }
  return false
}

function isFuture(r: MuteRule): boolean {
  if (!r.is_enabled) return false
  if (ruleType(r) !== 'once') return false
  const s = toMs(r.start_time)
  return !!(s && s > Date.now())
}

function isExpired(r: MuteRule): boolean {
  if (ruleType(r) !== 'once') return false
  const e = toMs(r.end_time)
  return !!(e && e < Date.now())
}

function statusToSev(r: MuteRule): string {
  if (!r.is_enabled) return 'muted'
  if (isActiveNow(r)) return 'success'
  if (isFuture(r)) return 'info'
  if (isExpired(r)) return 'muted'
  return 'info'
}

function statusText(r: MuteRule): string {
  if (!r.is_enabled) return 'DISABLED'
  if (isActiveNow(r)) return 'ACTIVE'
  if (isFuture(r)) return 'SCHEDULED'
  if (isExpired(r)) return 'EXPIRED'
  return 'IDLE'
}

function remainingMin(r: MuteRule): number {
  const e = toMs(r.end_time)
  if (!e) return 0
  return Math.max(0, Math.round((e - Date.now()) / 60000))
}

function relTimeFuture(t: string | null): string {
  const ms = toMs(t)
  if (!ms) return '-'
  const diff = ms - Date.now()
  const m = Math.round(diff / 60000)
  if (m < 60) return `${m}m`
  const h = Math.round(m / 60)
  if (h < 24) return `${h}h`
  return `${Math.round(h / 24)}d`
}

function describePeriodic(r: MuteRule): string {
  const days = (r.days_of_week || '').split(',').map(x => x.trim()).filter(Boolean)
  const dayMap: Record<string, string> = { '0': 'Sun', '1': 'Mon', '2': 'Tue', '3': 'Wed', '4': 'Thu', '5': 'Fri', '6': 'Sat' }
  const dayLabel = days.length ? days.map(d => dayMap[d] || d).join('/') : 'Daily'
  return `${dayLabel} ${r.periodic_start} - ${r.periodic_end} (${r.timezone || 'UTC'})`
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
  { label: 'All Types', value: 'all' },
  { label: 'One-time', value: 'once' },
  { label: 'Periodic', value: 'periodic' },
]

// ---------- API ----------
async function fetchRules() {
  loading.value = true
  try {
    const { data } = await muteRuleApi.list({ page: 1, page_size: 200 })
    rules.value = data.data.list || []
  } catch (err: any) {
    message.error(err.message)
  } finally {
    loading.value = false
  }
}

async function toggle(rule: MuteRule) {
  try {
    await muteRuleApi.update(rule.id, { is_enabled: !rule.is_enabled })
    message.success(rule.is_enabled ? t('mute.disabledSuccess') : t('mute.enabledSuccess'))
    fetchRules()
  } catch (err: any) {
    message.error(err.message)
  }
}

async function handleDelete(id: number) {
  try {
    await muteRuleApi.delete(id)
    message.success(t('mute.deleted'))
    fetchRules()
  } catch (err: any) {
    message.error(err.message)
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
  if (key === 'delete') handleDelete(rule.id)
}

// ---------- preview drawer ----------
const showPreview = ref(false)
const previewLoading = ref(false)
const previewRuleName = ref('')
const previewItems = ref<any[]>([])

async function previewHits(rule: MuteRule) {
  previewRuleName.value = rule.name
  showPreview.value = true
  previewLoading.value = true
  try {
    const { data } = await muteRuleApi.preview()
    const all = (data.data || []) as Array<{ rule_id: number; rule_name: string; matched_alerts: any[] }>
    const target = all.find(x => x.rule_id === rule.id)
    previewItems.value = target?.matched_alerts || []
  } catch (err: any) {
    message.error(err.message)
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
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

function severityOf(ev: any): string {
  return (ev.severity || ev.level || 'info').toLowerCase()
}

onMounted(fetchRules)
</script>

<template>
  <div class="mute-page sre-stagger">
    <PageHeader title="Mute Rules" subtitle="Suppress alert notifications during specific time windows">
      <template #actions>
        <NButton quaternary @click="fetchRules" :loading="loading">
          <template #icon><NIcon :component="RefreshOutline" /></template>
          {{ t('common.refresh') }}
        </NButton>
        <NButton type="primary" @click="openCreate">
          <template #icon><NIcon :component="AddOutline" /></template>
          New Rule
        </NButton>
      </template>
    </PageHeader>

    <div class="mute-toolbar">
      <NRadioGroup v-model:value="statusFilter" size="small">
        <NRadioButton value="all">All</NRadioButton>
        <NRadioButton value="active">Active</NRadioButton>
        <NRadioButton value="future">Scheduled</NRadioButton>
        <NRadioButton value="expired">Expired</NRadioButton>
        <NRadioButton value="disabled">Disabled</NRadioButton>
      </NRadioGroup>
      <div class="mute-filters">
        <NInput v-model:value="searchKeyword" size="small" placeholder="Search by name" clearable style="width: 220px">
          <template #prefix><NIcon :component="SearchOutline" /></template>
        </NInput>
        <NSelect v-model:value="typeFilter" size="small" :options="typeOptions" style="width: 140px" />
      </div>
    </div>

    <NSpin :show="loading">
      <div v-if="!loading && filteredRules.length === 0" class="mute-empty">
        <NEmpty description="No mute rules">
          <template #extra>
            <NButton type="primary" size="small" @click="openCreate">Create your first rule</NButton>
          </template>
        </NEmpty>
      </div>

      <div v-else class="mute-list">
        <div
          v-for="rule in filteredRules" :key="rule.id"
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
              <span class="sre-label-eyebrow">Match</span>
              <span v-for="(v, k) in rule.match_labels" :key="k" class="mute-chip">{{ k }}={{ v }}</span>
            </div>
            <div class="mute-schedule">
              <span class="sre-label-eyebrow">Schedule</span>
              <span v-if="ruleType(rule) === 'once'">Once {{ formatTime(rule.start_time) }} → {{ formatTime(rule.end_time) }}</span>
              <span v-else-if="ruleType(rule) === 'periodic'">Periodic {{ describePeriodic(rule) }}</span>
              <span v-else class="muted">No schedule</span>
            </div>
            <div class="mute-footer tnum">
              <span>{{ ((rule as any).hit_count) || 0 }} hits</span>
              <span class="sre-meta-divider"></span>
              <span v-if="!rule.is_enabled">Disabled</span>
              <span v-else-if="isActiveNow(rule)">Active · {{ remainingMin(rule) }}m remaining</span>
              <span v-else-if="isFuture(rule)">Starts in {{ relTimeFuture(rule.start_time) }}</span>
              <span v-else-if="isExpired(rule)">Expired</span>
              <span v-else>Idle</span>
            </div>
          </div>
          <div class="mute-actions" @click.stop>
            <NButton size="tiny" quaternary @click="previewHits(rule)">
              <template #icon><NIcon :component="EyeOutline" /></template>
              Preview
            </NButton>
            <NSwitch :value="rule.is_enabled" size="small" @update:value="toggle(rule)" />
            <NDropdown :options="rowActions(rule)" trigger="click" @select="(k: string) => handleAction(k, rule)">
              <NButton quaternary circle size="small">
                <template #icon><NIcon :component="EllipsisHorizontalOutline" /></template>
              </NButton>
            </NDropdown>
          </div>
        </div>
      </div>
    </NSpin>

    <!-- Preview Drawer -->
    <NDrawer v-model:show="showPreview" :width="480" placement="right">
      <NDrawerContent :title="`Will be muted — ${previewRuleName}`" closable>
        <NSpin :show="previewLoading">
          <div v-if="!previewLoading && previewItems.length === 0" class="mute-empty">
            <NEmpty description="No alerts currently match this rule" />
          </div>
          <div v-else class="mute-preview-list">
            <div
              v-for="ev in previewItems" :key="ev.id"
              class="sre-row-card mute-preview-row"
              :data-severity="severityOf(ev)"
            >
              <div class="mute-preview-main">
                <div class="mute-preview-head">
                  <span class="sre-dot" :data-severity="severityOf(ev)"></span>
                  <span class="mute-preview-sev">{{ severityOf(ev).toUpperCase() }}</span>
                  <span class="mute-preview-name">{{ ev.alert_name || ev.name || `#${ev.id}` }}</span>
                </div>
                <div v-if="ev.summary || ev.description" class="mute-preview-desc">
                  {{ ev.summary || ev.description }}
                </div>
                <div class="mute-preview-meta tnum">
                  <span>#{{ ev.id }}</span>
                  <span v-if="ev.fired_at" class="sre-meta-divider"></span>
                  <span v-if="ev.fired_at">{{ formatTime(ev.fired_at) }}</span>
                </div>
              </div>
            </div>
          </div>
        </NSpin>
      </NDrawerContent>
    </NDrawer>

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
        <NDivider style="margin: 12px 0">{{ t('mute.oneTimeMute') }}</NDivider>
        <NGrid :x-gap="12" :cols="2">
          <NGi>
            <NFormItem :label="t('mute.startTime')">
              <NDatePicker v-model:value="form.start_time" type="datetime" clearable style="width: 100%" />
            </NFormItem>
          </NGi>
          <NGi>
            <NFormItem :label="t('mute.endTime')">
              <NDatePicker v-model:value="form.end_time" type="datetime" clearable style="width: 100%" />
            </NFormItem>
          </NGi>
        </NGrid>
        <NDivider style="margin: 12px 0">{{ t('mute.periodicMute') }}</NDivider>
        <NGrid :x-gap="12" :cols="2">
          <NGi>
            <NFormItem :label="t('mute.periodicStart')">
              <NTimePicker v-model:formatted-value="form.periodic_start" value-format="HH:mm" format="HH:mm" clearable style="width: 100%" />
            </NFormItem>
          </NGi>
          <NGi>
            <NFormItem :label="t('mute.periodicEnd')">
              <NTimePicker v-model:formatted-value="form.periodic_end" value-format="HH:mm" format="HH:mm" clearable style="width: 100%" />
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
              <NInput v-model:value="form.rule_ids" placeholder="1,2,3" />
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
  </div>
</template>

<style scoped>
.mute-page { max-width: 1280px; }

.mute-toolbar {
  display: flex; align-items: center; justify-content: space-between;
  margin: 12px 0 14px; gap: 12px; flex-wrap: wrap;
}
.mute-filters { display: flex; align-items: center; gap: 8px; }

.mute-list { display: flex; flex-direction: column; gap: 8px; }
.mute-empty { padding: 60px 0; text-align: center; }

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
  border: 1px solid var(--sre-hairline);
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
