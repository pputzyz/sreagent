<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, h } from 'vue'
import {
  NButton, NIcon, NSwitch, NDropdown, NInput, NEmpty, NSpin,
  NModal, NForm, NFormItem, NSpace, useMessage, useDialog,
} from 'naive-ui'
import type { FormInst } from 'naive-ui'
import {
  AddOutline, EllipsisHorizontalOutline, CreateOutline, TrashOutline, SearchOutline,
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { inhibitionRuleApi } from '@/api'
import type { InhibitionRule } from '@/types'
import PageHeader from '@/components/common/PageHeader.vue'
import LabelMatcherEditor from '@/components/common/LabelMatcherEditor.vue'
import type { LabelMatcher } from '@/components/common/LabelMatcherEditor.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const auth = useAuthStore()
const canManage = computed(() => auth.canManage)

const list = shallowRef<InhibitionRule[]>([])
const loading = ref(false)
const searchKeyword = ref('')

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
onMounted(fetchList)

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
function relTime(t: string): string {
  const ms = new Date(t).getTime()
  if (Number.isNaN(ms)) return '-'
  const diff = Date.now() - ms
  const m = Math.round(diff / 60000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m}m ago`
  const hr = Math.round(m / 60)
  if (hr < 24) return `${hr}h ago`
  return `${Math.round(hr / 24)}d ago`
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
    <PageHeader title="Inhibition Rules" subtitle="Suppress target alerts when source alert is firing">
      <template #actions>
        <NButton v-if="canManage" type="primary" @click="openCreate">
          <template #icon><NIcon :component="AddOutline" /></template>
          New Rule
        </NButton>
      </template>
    </PageHeader>

    <div class="inhib-toolbar">
      <NInput v-model:value="searchKeyword" size="small" placeholder="Search by name" clearable style="width: 260px">
        <template #prefix><NIcon :component="SearchOutline" /></template>
      </NInput>
    </div>

    <NSpin :show="loading">
      <div v-if="!loading && filteredList.length === 0" class="inhib-empty">
        <NEmpty description="No inhibition rules">
          <template #extra>
            <NButton v-if="canManage" type="primary" size="small" @click="openCreate">Create your first rule</NButton>
          </template>
        </NEmpty>
      </div>

      <div v-else class="inhib-list">
        <div
          v-for="rule in filteredList" :key="rule.id"
          class="sre-row-card inhib-row sre-lift"
          :data-dim="!rule.is_enabled || undefined"
          @click="goEdit(rule)"
        >
          <div class="inhib-main">
            <div class="inhib-headline">
              <span class="sre-dot" :data-severity="rule.is_enabled ? 'success' : 'muted'"></span>
              <span class="inhib-status">{{ rule.is_enabled ? 'ENABLED' : 'DISABLED' }}</span>
              <span class="inhib-name">{{ rule.name }}</span>
            </div>
            <div v-if="rule.description" class="inhib-desc">{{ rule.description }}</div>
            <div class="inhib-row-config">
              <span class="sre-label-eyebrow">Source</span>
              <span v-for="m in matchEntries(rule.source_match)" :key="'s-' + m" class="mono-chip">{{ m }}</span>
              <span v-if="!matchEntries(rule.source_match).length" class="muted">—</span>
            </div>
            <div class="inhib-row-config">
              <span class="sre-label-eyebrow">Target</span>
              <span v-for="m in matchEntries(rule.target_match)" :key="'t-' + m" class="mono-chip">{{ m }}</span>
              <span v-if="!matchEntries(rule.target_match).length" class="muted">—</span>
            </div>
            <div v-if="equalArr(rule.equal_labels).length" class="inhib-row-config">
              <span class="sre-label-eyebrow">Equal</span>
              <span class="mono-chip">{{ equalArr(rule.equal_labels).join(', ') }}</span>
            </div>
            <div class="inhib-footer tnum">
              <span>{{ ((rule as any).hit_count) || 0 }} hits</span>
              <template v-if="(rule as any).last_hit_at">
                <span class="sre-meta-divider"></span>
                <span>last {{ relTime((rule as any).last_hit_at) }}</span>
              </template>
              <template v-else-if="rule.updated_at">
                <span class="sre-meta-divider"></span>
                <span>updated {{ relTime(rule.updated_at) }}</span>
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
      preset="card" style="width: 640px" :mask-closable="false" :bordered="false"
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
          <NInput v-model:value="formData.equal_labels" placeholder="alertname,namespace" />
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
  </div>
</template>

<style scoped>
.inhib-page { max-width: 1280px; }

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
.inhib-desc { font-size: 12px; color: var(--sre-text-secondary); }

.inhib-row-config {
  display: flex; align-items: center; gap: 6px; flex-wrap: wrap;
  font-size: 12px; color: var(--sre-text-tertiary);
}
.mono-chip {
  font-family: var(--sre-font-mono); font-size: 11px;
  background: var(--sre-bg-elevated); border-radius: 4px;
  padding: 2px 6px; color: var(--sre-text-secondary);
  border: 1px solid var(--sre-hairline);
}
.muted { color: var(--sre-text-tertiary); }

.inhib-footer {
  display: flex; align-items: center;
  font-size: 12px; color: var(--sre-text-tertiary);
}

.inhib-actions {
  display: flex; align-items: center; gap: 6px; flex-shrink: 0;
}
</style>
