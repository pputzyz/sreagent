<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, h } from 'vue'
import {
  NButton, NIcon, NSwitch, NDropdown, NInput, NSpin,
  NModal, NForm, NFormItem, NSpace, useMessage, useDialog,
} from 'naive-ui'
import type { FormInst } from 'naive-ui'
import {
  AddOutline, EllipsisHorizontalOutline, CreateOutline, TrashOutline, SearchOutline, SparklesOutline,
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { inhibitionRuleApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import type { InhibitionRule } from '@/types'
import type { RuleGenerateResult, MuteRuleGenerateResult } from '@/types/ai-module'
import { useAIModule } from '@/composables'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import AIGenerateModal from '@/components/alert-rule/AIGenerateModal.vue'
import LabelMatcherEditor from '@/components/common/LabelMatcherEditor.vue'
import type { LabelMatcher } from '@/components/common/LabelMatcherEditor.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const auth = useAuthStore()
const canManage = computed(() => auth.canManage)

const { isEnabled: isAIModuleEnabled, loadModules } = useAIModule()

const list = shallowRef<InhibitionRule[]>([])
const loading = ref(false)
const searchKeyword = ref('')

// ─── AI Inhibition Generation ───
const showAIModal = ref(false)

function openAIGenerate() {
  showAIModal.value = true
}

async function handleAIGenerated(result: RuleGenerateResult | MuteRuleGenerateResult) {
  const inhibResult = result as RuleGenerateResult
  try {
    const sourceMatch: Record<string, string> = {}
    if (inhibResult.source_labels) {
      for (const label of inhibResult.source_labels) {
        sourceMatch[label] = label === 'alertname' ? (inhibResult.source_value || '') : '=~.*'
      }
    }
    const targetMatch: Record<string, string> = {}
    if (inhibResult.target_labels) {
      for (const label of inhibResult.target_labels) {
        targetMatch[label] = '=~.*'
      }
    }
    await inhibitionRuleApi.create({
      name: inhibResult.name,
      description: inhibResult.description,
      source_match: sourceMatch,
      target_match: targetMatch,
      equal_labels: (inhibResult.equal_labels || []).join(','),
      is_enabled: true,
    })
    message.success(t('common.createSuccess'))
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

const filteredList = computed(() => {
  const kw = searchKeyword.value.trim().toLowerCase()
  if (!kw) return list.value
  return list.value.filter(r =>
    r.name.toLowerCase().includes(kw) ||
    (r.description || '').toLowerCase().includes(kw)
  )
})

async function fetchList() {
  loading.value = true
  try {
    const res = await inhibitionRuleApi.list({ page: 1, page_size: 200 })
    list.value = res.data.data.list || []
  } catch {
    message.error(t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}
onMounted(() => {
  fetchList()
  loadModules()
})

// ---- helpers ----
function recordToMatchers(record: Record<string, string>): LabelMatcher[] {
  return Object.entries(record || {}).map(([key, raw]) => {
    for (const op of ['!=', '=~', '!~'] as const) {
      if (raw.startsWith(op)) return { key, op, value: raw.slice(op.length) }
    }
    return { key, op: '=', value: raw }
  })
}
function matchersToRecord(matchers: LabelMatcher[]): Record<string, string> {
  return Object.fromEntries(matchers.map(m => {
    const v = m.op === '=' ? m.value : `${m.op}${m.value}`
    return [m.key, v]
  }))
}
function formatMatcher(k: string, raw: string): string {
  for (const op of ['!=', '=~', '!~'] as const) {
    if (raw.startsWith(op)) return `${k}${op}${raw.slice(op.length)}`
  }
  return `${k}=${raw}`
}
function matchEntries(rec: Record<string, string>): string[] {
  return Object.entries(rec || {}).map(([k, v]) => formatMatcher(k, v))
}
function equalArr(s: string): string[] {
  return (s || '').split(',').map(x => x.trim()).filter(Boolean)
}
function relTime(timeStr: string): string {
  const ms = new Date(timeStr).getTime()
  if (Number.isNaN(ms)) return '-'
  const diff = Date.now() - ms
  const m = Math.round(diff / 60000)
  if (m < 1) return t('inhibition.justNow')
  if (m < 60) return t('inhibition.minAgo', { n: m })
  const hr = Math.round(m / 60)
  if (hr < 24) return t('inhibition.hourAgo', { n: hr })
  return t('inhibition.dayAgo', { n: Math.round(hr / 24) })
}

// ---- modal ----
interface InhibitionForm {
  name: string
  description: string
  source_matchers: LabelMatcher[]
  target_matchers: LabelMatcher[]
  equal_labels: string
  is_enabled: boolean
}

const modalVisible = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInst | null>(null)
const defaultForm = (): InhibitionForm => ({
  name: '', description: '',
  source_matchers: [], target_matchers: [],
  equal_labels: '', is_enabled: true,
})
const formData = ref<InhibitionForm>(defaultForm())

function getHitCount(rule: InhibitionRule): number {
  return (rule as InhibitionRule & { hit_count?: number }).hit_count || 0
}

function getLastHitAt(rule: InhibitionRule): string | undefined {
  return (rule as InhibitionRule & { last_hit_at?: string }).last_hit_at
}

function openCreate() {
  editingId.value = null
  formData.value = defaultForm()
  modalVisible.value = true
}
function openEdit(row: InhibitionRule) {
  editingId.value = row.id
  formData.value = {
    name: row.name, description: row.description,
    source_matchers: recordToMatchers(row.source_match ?? {}),
    target_matchers: recordToMatchers(row.target_match ?? {}),
    equal_labels: row.equal_labels, is_enabled: row.is_enabled,
  }
  modalVisible.value = true
}

async function handleSave() {
  try { await formRef.value?.validate() } catch { return }
  saving.value = true
  try {
    const payload = {
      name: formData.value.name,
      description: formData.value.description,
      source_match: matchersToRecord(formData.value.source_matchers),
      target_match: matchersToRecord(formData.value.target_matchers),
      equal_labels: formData.value.equal_labels,
      is_enabled: formData.value.is_enabled,
    }
    if (editingId.value) {
      await inhibitionRuleApi.update(editingId.value, payload)
      message.success(t('common.updateSuccess'))
    } else {
      await inhibitionRuleApi.create(payload)
      message.success(t('common.createSuccess'))
    }
    modalVisible.value = false
    fetchList()
  } catch {
    message.error(t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

function handleDelete(row: InhibitionRule) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: `${t('common.confirmDeleteMsg')} "${row.name}"?`,
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await inhibitionRuleApi.delete(row.id)
        message.success(t('common.deleteSuccess'))
        fetchList()
      } catch { message.error(t('common.deleteFailed')) }
    },
  })
}

async function toggle(row: InhibitionRule) {
  try {
    await inhibitionRuleApi.update(row.id, { is_enabled: !row.is_enabled })
    fetchList()
  } catch { message.error(t('common.saveFailed')) }
}

function rowActions(_r: InhibitionRule) {
  return [
    { label: t('common.edit'), key: 'edit', icon: () => h(NIcon, { component: CreateOutline }) },
    { label: t('common.delete'), key: 'delete', icon: () => h(NIcon, { component: TrashOutline }) },
  ]
}
function handleAction(key: string, row: InhibitionRule) {
  if (key === 'edit') openEdit(row)
  if (key === 'delete') handleDelete(row)
}
function goEdit(row: InhibitionRule) { if (canManage.value) openEdit(row) }
</script>

<template>
  <div class="inhib-page sre-stagger">
    <PageHeader :title="t('inhibition.title')" :subtitle="t('inhibition.description')">
      <template #actions>
        <NButton v-if="canManage && isAIModuleEnabled('rule_gen')" size="small" secondary @click="openAIGenerate">
          <template #icon><NIcon :component="SparklesOutline" /></template>
          {{ t('alert.aiGenerate') }}
        </NButton>
        <NButton v-if="canManage" type="primary" @click="openCreate">
          <template #icon><NIcon :component="AddOutline" /></template>
          {{ t('inhibition.createRule') }}
        </NButton>
      </template>
    </PageHeader>

    <div class="inhib-toolbar">
      <NInput v-model:value="searchKeyword" size="small" :placeholder="t('common.search')" clearable class="inhib-search">
        <template #prefix><NIcon :component="SearchOutline" /></template>
      </NInput>
    </div>

    <LoadingSkeleton v-if="loading && filteredList.length === 0" :rows="6" variant="row" />
    <EmptyState
      v-else-if="!loading && filteredList.length === 0"
      :icon="AddOutline"
      :title="t('inhibition.noRules')"
      :description="t('inhibition.description')"
      :primary-text="t('inhibition.createRule')"
      @primary="openCreate"
    />

    <NSpin v-else :show="loading">
      <div class="inhib-list">
        <div
          v-for="rule in filteredList" :key="rule.id"
          class="sre-row-card inhib-row sre-lift"
          :data-dim="!rule.is_enabled || undefined"
          @click="goEdit(rule)"
        >
          <div class="inhib-main">
            <div class="inhib-headline">
              <span class="sre-dot" :data-severity="rule.is_enabled ? 'success' : 'muted'"></span>
              <span class="inhib-status">{{ rule.is_enabled ? t('inhibition.statusEnabled') : t('inhibition.statusDisabled') }}</span>
              <span class="inhib-name">{{ rule.name }}</span>
              <span class="inhib-hit-badge" :data-active="getHitCount(rule) > 0 || undefined">
                {{ t('inhibition.hits', { n: getHitCount(rule) }) }}
              </span>
            </div>
            <div v-if="rule.description" class="inhib-desc">{{ rule.description }}</div>
            <div class="inhib-row-config">
              <span class="sre-label-eyebrow">{{ t('inhibition.sourceLabel') }}</span>
              <span v-for="m in matchEntries(rule.source_match)" :key="'s-' + m" class="mono-chip">{{ m }}</span>
              <span v-if="!matchEntries(rule.source_match).length" class="muted">—</span>
            </div>
            <div class="inhib-row-config">
              <span class="sre-label-eyebrow">{{ t('inhibition.targetLabel') }}</span>
              <span v-for="m in matchEntries(rule.target_match)" :key="'t-' + m" class="mono-chip">{{ m }}</span>
              <span v-if="!matchEntries(rule.target_match).length" class="muted">—</span>
            </div>
            <div v-if="equalArr(rule.equal_labels).length" class="inhib-row-config">
              <span class="sre-label-eyebrow">{{ t('inhibition.equalLabel') }}</span>
              <span class="mono-chip">{{ equalArr(rule.equal_labels).join(', ') }}</span>
            </div>
            <div class="inhib-footer tnum">
              <span>{{ t('inhibition.hits', { n: getHitCount(rule) }) }}</span>
              <template v-if="getLastHitAt(rule)">
                <span class="sre-meta-divider"></span>
                <span>{{ t('inhibition.lastHit') }}{{ relTime(getLastHitAt(rule)!) }}</span>
              </template>
              <template v-else-if="rule.updated_at">
                <span class="sre-meta-divider"></span>
                <span>{{ t('inhibition.updatedAt') }}{{ relTime(rule.updated_at) }}</span>
              </template>
            </div>
          </div>
          <div class="inhib-actions" @click.stop>
            <NSwitch :value="rule.is_enabled" size="small" :disabled="!canManage" @update:value="toggle(rule)" />
            <NDropdown v-if="canManage" :options="rowActions(rule)" trigger="click" @select="(k: string) => handleAction(k, rule)">
              <NButton quaternary circle size="small">
                <template #icon><NIcon :component="EllipsisHorizontalOutline" /></template>
              </NButton>
            </NDropdown>
          </div>
        </div>
      </div>
    </NSpin>

    <!-- Create / Edit Modal -->
    <NModal
      v-model:show="modalVisible"
      :title="editingId ? t('inhibition.editRule') : t('inhibition.createRule')"
      preset="card" class="inhib-modal" :mask-closable="false" :bordered="false"
    >
      <NForm ref="formRef" :model="formData" label-placement="top">
        <NFormItem :label="t('inhibition.name')" path="name" :rule="{ required: true, message: t('common.required') }">
          <NInput v-model:value="formData.name" :placeholder="t('inhibition.name')" />
        </NFormItem>
        <NFormItem :label="t('common.description')">
          <NInput v-model:value="formData.description" type="textarea" :rows="2" />
        </NFormItem>
        <NFormItem :label="t('inhibition.sourceMatch')">
          <LabelMatcherEditor v-model:modelValue="formData.source_matchers" :add-label="t('inhibition.addLabel')" />
        </NFormItem>
        <NFormItem :label="t('inhibition.targetMatch')">
          <LabelMatcherEditor v-model:modelValue="formData.target_matchers" :add-label="t('inhibition.addLabel')" />
        </NFormItem>
        <NFormItem :label="t('inhibition.equalLabels')" :feedback="t('inhibition.equalLabelsHint')">
          <NInput v-model:value="formData.equal_labels" :placeholder="t('inhibitionMgmt.equalLabelsPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('inhibition.isEnabled')">
          <NSwitch v-model:value="formData.is_enabled" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="modalVisible = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="saving" @click="handleSave">{{ t('common.save') }}</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- AI Generate Inhibition Modal -->
    <AIGenerateModal
      v-model:visible="showAIModal"
      rule-type="inhibition"
      @generated="handleAIGenerated"
    />
  </div>
</template>

<style scoped>
.inhib-page { max-width: 1280px; font-family: var(--sre-font-sans); }

.inhib-toolbar { margin: 12px 0 14px; }

.inhib-list { display: flex; flex-direction: column; gap: 8px; }
.inhib-empty { padding: 60px 0; text-align: center; }

.inhib-row {
  padding: 14px 18px; gap: 12px; cursor: pointer;
  display: flex; align-items: flex-start;
}
.inhib-row[data-dim] { opacity: 0.55; }

.inhib-main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 6px; }
.inhib-headline {
  display: flex; align-items: center; gap: 8px;
  font-size: 14px; font-weight: 600;
}
.inhib-status {
  font-size: 11px; font-weight: 600; color: var(--sre-text-secondary);
  text-transform: uppercase; letter-spacing: 0.6px;
}
.inhib-name { color: var(--sre-text-primary); }
.inhib-hit-badge {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-tertiary);
  background: var(--sre-bg-elevated);
  border: var(--sre-hairline);
  border-radius: 10px;
  padding: 1px 8px;
  margin-left: auto;
  white-space: nowrap;
}
.inhib-hit-badge[data-active] {
  color: var(--sre-error, #e88080);
  border-color: var(--sre-error, #e88080);
  background: rgba(232, 128, 128, 0.08);
}
.inhib-desc { font-size: 12px; color: var(--sre-text-secondary); }

.inhib-row-config {
  display: flex; align-items: center; gap: 6px; flex-wrap: wrap;
  font-size: 12px; color: var(--sre-text-tertiary);
}
.mono-chip {
  font-family: var(--sre-font-mono); font-size: 11px;
  background: var(--sre-bg-elevated); border-radius: 4px;
  padding: 2px 6px; color: var(--sre-text-secondary);
  border: var(--sre-hairline);
}
.muted { color: var(--sre-text-tertiary); }

.inhib-footer {
  display: flex; align-items: center;
  font-size: 12px; color: var(--sre-text-tertiary);
}

.inhib-actions {
  display: flex; align-items: center; gap: 6px; flex-shrink: 0;
}

/* AI Generate Modal */
.ai-gen-form { display: flex; flex-direction: column; gap: 14px; }
.ai-gen-field { display: flex; flex-direction: column; gap: 6px; }
.ai-gen-label { font-size: 13px; font-weight: 500; color: var(--sre-text-secondary); }
.ai-gen-preview {
  margin-top: 20px; padding: 16px;
  background: var(--sre-bg-elevated, rgba(255,255,255,0.04));
  border: var(--sre-hairline); border-radius: 8px;
}
.ai-gen-preview-header { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
.ai-gen-preview-title { font-size: 15px; font-weight: 600; color: var(--sre-text-primary); }
.ai-gen-confidence { font-size: 12px; font-family: var(--sre-font-mono, monospace); color: var(--sre-text-tertiary); margin-left: auto; }
.ai-gen-desc { font-size: 13px; color: var(--sre-text-secondary); margin-bottom: 8px; }
.ai-gen-meta { font-size: 12px; color: var(--sre-text-tertiary); margin-bottom: 4px; display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.ai-gen-meta-label { font-weight: 600; color: var(--sre-text-secondary); }
</style>
