<script setup lang="ts">
import { reactive, ref, shallowRef, computed, onMounted, h } from 'vue'
import { useMessage, NDropdown } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { notifyRuleApi } from '@/api'
import type { NotifyRule } from '@/types'
import { getErrorMessage } from '@/utils/format'
import { AddOutline, SearchOutline, FilterOutline } from '@vicons/ionicons5'
import LabelMatcherEditor from '@/components/common/LabelMatcherEditor.vue'
import type { LabelMatcher } from '@/components/common/LabelMatcherEditor.vue'

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const rules = shallowRef<NotifyRule[]>([])
const showModal = ref(false)
const modalTitle = ref('')
const editingId = ref<number | null>(null)
const saving = ref(false)
const search = ref('')

const form = reactive({
  name: '',
  description: '',
  severities: [] as string[],
  match_labels: [] as LabelMatcher[],
  pipeline: '[]',
  notify_configs: '[]',
  repeat_interval: 3600,
  callback_url: '',
  is_enabled: true,
})

const severityOptions = computed(() => [
  { label: t('alert.critical'), value: 'critical' },
  { label: t('alert.warning'), value: 'warning' },
  { label: t('alert.info'), value: 'info' },
])

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

async function fetchData() {
  loading.value = true
  try {
    const { data } = await notifyRuleApi.list({ page: 1, page_size: 100 })
    rules.value = data.data.list || []
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { loading.value = false }
}

function resetForm() {
  Object.assign(form, {
    name: '', description: '', severities: [], match_labels: [],
    pipeline: '[]', notify_configs: '[]', repeat_interval: 3600,
    callback_url: '', is_enabled: true,
  })
}

function openCreate() {
  editingId.value = null
  modalTitle.value = t('notifyRule.create')
  resetForm()
  showModal.value = true
}

function openEdit(row: NotifyRule) {
  editingId.value = row.id
  modalTitle.value = t('notifyRule.edit')
  Object.assign(form, {
    name: row.name,
    description: row.description,
    severities: (row.severities || '').split(',').filter(Boolean),
    match_labels: Object.entries(row.match_labels || {}).map(([key, raw]) => {
      for (const op of ['!=', '=~', '!~'] as const) {
        if (raw.startsWith(op)) return { key, op, value: raw.slice(op.length) }
      }
      return { key, op: '=' as const, value: raw }
    }),
    pipeline: row.pipeline || '[]',
    notify_configs: row.notify_configs || '[]',
    repeat_interval: row.repeat_interval,
    callback_url: row.callback_url || '',
    is_enabled: row.is_enabled,
  })
  showModal.value = true
}

async function handleSave() {
  if (!form.name.trim()) { message.warning(t('notifyRule.nameRequired')); return }
  try { JSON.parse(form.pipeline) } catch { message.warning(t('notifyRule.pipeline') + ': ' + t('notifyRule.invalidJson')); return }
  try { JSON.parse(form.notify_configs) } catch { message.warning(t('notifyRule.notifyConfigs') + ': ' + t('notifyRule.invalidJson')); return }

  saving.value = true
  try {
    const payload = {
      name: form.name,
      description: form.description,
      severities: form.severities.join(','),
      match_labels: Object.fromEntries(form.match_labels.map(m => {
        const v = m.op === '=' ? m.value : `${m.op}${m.value}`
        return [m.key, v]
      })),
      pipeline: form.pipeline,
      notify_configs: form.notify_configs,
      repeat_interval: form.repeat_interval,
      callback_url: form.callback_url,
      is_enabled: form.is_enabled,
    }
    if (editingId.value) {
      await notifyRuleApi.update(editingId.value, payload)
      message.success(t('notifyRule.updated'))
    } else {
      await notifyRuleApi.create(payload)
      message.success(t('notifyRule.created'))
    }
    showModal.value = false
    fetchData()
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { saving.value = false }
}

async function handleDelete(id: number) {
  try {
    await notifyRuleApi.delete(id)
    message.success(t('notifyRule.deleted'))
    fetchData()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
}

async function toggleEnabled(row: NotifyRule, val: boolean) {
  try {
    await notifyRuleApi.update(row.id, { ...row, is_enabled: val })
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
  if (key === 'edit') openEdit(row)
  else if (key === 'delete' && confirm(t('notifyRule.deleteConfirm'))) handleDelete(row.id)
}
const RowMenu = (row: NotifyRule) => h(NDropdown, {
  trigger: 'click', options: rowMenu(row),
  onSelect: (k: string) => onRowMenu(k, row),
}, { default: () => h('button', { class: 'sre-icon-btn' }, h('span', { class: 'sre-dots' })) })

onMounted(fetchData)
</script>

<template>
  <div class="rules-page">
    <header class="sub-header">
      <div>
        <h2 class="sub-title">{{ t('notifyRule.title') }}</h2>
        <p class="sub-sub">{{ t('notifyRule.subtitle') }}</p>
      </div>
      <n-button type="primary" size="small" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        {{ t('notifyRule.create') }}
      </n-button>
    </header>

    <div class="toolbar">
      <n-input v-model:value="search" size="small" :placeholder="t('common.search')" clearable style="width: 240px">
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <span class="count tnum">{{ filtered.length }} / {{ rules.length }}</span>
    </div>

    <div v-if="loading" class="loading">{{ t('common.loading') }}…</div>

    <div v-else-if="filtered.length === 0" class="empty">
      <n-icon :component="FilterOutline" size="36" />
      <div class="empty-text">{{ t('notifyRule.noData') }}</div>
      <n-button type="primary" size="small" @click="openCreate">{{ t('notifyRule.create') }}</n-button>
    </div>

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
            <n-switch :value="r.is_enabled" size="small" @update:value="(v: boolean) => toggleEnabled(r, v)" />
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

        <n-form-item :label="t('notifyRule.pipeline')">
          <n-input v-model:value="form.pipeline" type="textarea" :rows="4"
            :placeholder="t('notifyRule.pipelineHint')"
            style="font-family: var(--sre-font-mono); font-size: 12px" />
        </n-form-item>

        <n-form-item :label="t('notifyRule.notifyConfigs')">
          <n-input v-model:value="form.notify_configs" type="textarea" :rows="4"
            :placeholder="t('notifyRule.notifyConfigsHint')"
            style="font-family: var(--sre-font-mono); font-size: 12px" />
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

.loading, .empty { padding: 60px 20px; text-align: center; color: var(--sre-text-secondary, #888); }
.empty { display: flex; flex-direction: column; gap: 12px; align-items: center; }
.empty-text { font-size: 13px; }

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
</style>
