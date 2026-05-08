<script setup lang="ts">
import { ref, shallowRef, reactive, computed, onMounted } from 'vue'
import { useMessage, NIcon, NButton, NDropdown } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { messageTemplateApi } from '@/api'
import type { MessageTemplate } from '@/types'
import {
  AddOutline, SearchOutline, EllipsisHorizontalOutline,
  CreateOutline, EyeOutline, TrashOutline, DocumentTextOutline,
} from '@vicons/ionicons5'

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const templates = shallowRef<MessageTemplate[]>([])
const search = ref('')
const typeFilter = ref<string>('')

// Modal state
const showModal = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)

// Preview state
const showPreviewModal = ref(false)
const previewLoading = ref(false)
const previewResult = ref('')

const form = reactive({
  name: '',
  description: '',
  type: 'text' as 'text' | 'html' | 'markdown' | 'lark_card',
  content: '',
})

const typeOptions = computed(() => [
  { label: t('template.text'), value: 'text' },
  { label: t('template.html'), value: 'html' },
  { label: t('template.markdown'), value: 'markdown' },
  { label: t('template.larkCard'), value: 'lark_card' },
])

const filterTypeOptions = computed(() => [
  { label: t('common.all'), value: '' },
  ...typeOptions.value,
])

// Type chip color mapping (subtle, not big NTag)
function typeChipClass(type: string) {
  const map: Record<string, string> = {
    text: 'text',
    html: 'html',
    markdown: 'markdown',
    lark_card: 'lark',
  }
  return map[type] || 'text'
}

const filtered = computed(() => {
  let list = templates.value
  if (typeFilter.value) list = list.filter(t => t.type === typeFilter.value)
  if (search.value.trim()) {
    const q = search.value.toLowerCase()
    list = list.filter(t =>
      t.name.toLowerCase().includes(q) ||
      (t.description?.toLowerCase().includes(q))
    )
  }
  return list
})

async function fetchData() {
  loading.value = true
  try {
    const { data } = await messageTemplateApi.list({ page: 1, page_size: 100 })
    templates.value = data.data.list || []
  } catch (err: any) {
    message.error(err.message)
  } finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, { name: '', description: '', type: 'text', content: '' })
}

function openCreate() {
  editingId.value = null
  resetForm()
  showModal.value = true
}

function openEdit(row: MessageTemplate) {
  editingId.value = row.id
  Object.assign(form, {
    name: row.name,
    description: row.description,
    type: row.type,
    content: row.content || '',
  })
  showModal.value = true
}

async function handleSave() {
  if (!form.name.trim()) {
    message.warning(t('template.nameRequired'))
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.name,
      description: form.description,
      type: form.type,
      content: form.content,
    }
    if (editingId.value) {
      await messageTemplateApi.update(editingId.value, payload)
      message.success(t('template.updated'))
    } else {
      await messageTemplateApi.create(payload)
      message.success(t('template.created'))
    }
    showModal.value = false
    fetchData()
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await messageTemplateApi.delete(id)
    message.success(t('template.deleted'))
    fetchData()
  } catch (err: any) {
    message.error(err.message)
  }
}

async function handlePreview(row: MessageTemplate) {
  previewLoading.value = true
  previewResult.value = ''
  showPreviewModal.value = true
  try {
    const { data } = await messageTemplateApi.preview({ content: row.content, type: row.type })
    previewResult.value = data.data.rendered
  } catch (err: any) {
    previewResult.value = ''
    message.error(t('template.previewFailed') + ': ' + err.message)
  } finally {
    previewLoading.value = false
  }
}

async function handlePreviewFromForm() {
  previewLoading.value = true
  previewResult.value = ''
  showPreviewModal.value = true
  try {
    const { data } = await messageTemplateApi.preview({ content: form.content, type: form.type })
    previewResult.value = data.data.rendered
  } catch (err: any) {
    previewResult.value = ''
    message.error(t('template.previewFailed') + ': ' + err.message)
  } finally {
    previewLoading.value = false
  }
}

function rowActions(row: MessageTemplate) {
  const actions = [
    { key: 'edit', label: t('common.edit'), icon: () => h(NIcon, { component: CreateOutline }) },
    { key: 'preview', label: t('template.preview'), icon: () => h(NIcon, { component: EyeOutline }) },
  ]
  if (!row.is_builtin) {
    actions.push({
      key: 'delete',
      label: t('common.delete'),
      icon: () => h(NIcon, { component: TrashOutline }),
    } as any)
  }
  return actions
}

function handleAction(key: string, row: MessageTemplate) {
  if (key === 'edit') openEdit(row)
  else if (key === 'preview') handlePreview(row)
  else if (key === 'delete') {
    if (window.confirm(t('template.deleteConfirm') || 'Delete?')) handleDelete(row.id)
  }
}

// Truncate template content preview
function contentPreview(content: string | null | undefined): string {
  if (!content) return '—'
  const trimmed = content.trim().replace(/\s+/g, ' ')
  return trimmed.length > 80 ? trimmed.slice(0, 80) + '…' : trimmed
}

const availableVariables = '{{.AlertName}} {{.Severity}} {{.Status}} {{.Labels}} {{.Annotations}} {{.FiredAt}} {{.Value}} {{.Duration}} {{.RuleName}}'

import { h } from 'vue'

onMounted(fetchData)
</script>

<template>
  <div class="tmpl-page">
    <header class="tmpl-header">
      <div>
        <h2 class="tmpl-title">{{ t('template.title') }}</h2>
        <p class="tmpl-subtitle">{{ t('template.subtitle') }}</p>
      </div>
      <n-button type="primary" size="small" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        {{ t('template.create') }}
      </n-button>
    </header>

    <div class="toolbar">
      <n-input
        v-model:value="search" size="small" clearable
        :placeholder="t('common.search')" style="width: 240px"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select
        v-model:value="typeFilter" size="small"
        :options="filterTypeOptions" style="width: 140px"
      />
      <span class="count tnum">{{ filtered.length }} / {{ templates.length }}</span>
    </div>

    <div v-if="loading" class="empty-state">{{ t('common.loading') }}…</div>
    <div v-else-if="filtered.length === 0" class="empty-state">
      <n-icon :component="DocumentTextOutline" size="40" />
      <p>{{ t('template.noData') }}</p>
      <n-button type="primary" size="small" @click="openCreate">{{ t('template.create') }}</n-button>
    </div>

    <div v-else class="tmpl-list sre-stagger">
      <div
        v-for="tpl in filtered" :key="tpl.id"
        class="tmpl-row"
        @click="openEdit(tpl)"
      >
        <div class="tmpl-main">
          <div class="tmpl-head">
            <span class="tmpl-name">{{ tpl.name }}</span>
            <span class="tmpl-type-chip" :data-type="typeChipClass(tpl.type)">{{ tpl.type }}</span>
            <span v-if="tpl.is_builtin" class="tmpl-builtin">{{ t('template.builtin') }}</span>
          </div>
          <div v-if="tpl.description" class="tmpl-desc">{{ tpl.description }}</div>
          <code class="tmpl-content">{{ contentPreview(tpl.content) }}</code>
        </div>
        <div class="tmpl-actions" @click.stop>
          <n-dropdown :options="rowActions(tpl)" trigger="click" @select="handleAction($event, tpl)">
            <n-button quaternary circle size="small">
              <template #icon><n-icon :component="EllipsisHorizontalOutline" /></template>
            </n-button>
          </n-dropdown>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <n-modal v-model:show="showModal" preset="card" :title="editingId ? t('template.edit') : t('template.create')" style="width: 600px" :bordered="false">
      <n-form label-placement="top">
        <n-grid :x-gap="12" :cols="2">
          <n-gi>
            <n-form-item :label="t('template.name')" required>
              <n-input v-model:value="form.name" placeholder="e.g. default-alert-template" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('template.type')">
              <n-select v-model:value="form.type" :options="typeOptions" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-form-item :label="t('template.description')">
          <n-input v-model:value="form.description" :placeholder="t('template.description')" />
        </n-form-item>

        <n-collapse>
          <n-collapse-item :title="t('template.availableVariables')" name="variables">
            <n-code :code="availableVariables" language="text" style="font-size: 12px" />
          </n-collapse-item>
        </n-collapse>

        <n-form-item :label="t('template.content')" style="margin-top: 12px">
          <n-input
            v-model:value="form.content" type="textarea" :rows="12"
            :placeholder="t('common.enterContent')"
            style="font-family: var(--sre-font-mono); font-size: 12px"
          />
        </n-form-item>
      </n-form>

      <template #action>
        <n-space justify="end">
          <n-button @click="handlePreviewFromForm" :loading="previewLoading">{{ t('template.preview') }}</n-button>
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="handleSave">
            {{ editingId ? t('common.update') : t('common.create') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Preview Modal -->
    <n-modal v-model:show="showPreviewModal" preset="card" :title="t('template.previewResult')" style="width: 600px" :bordered="false">
      <n-spin :show="previewLoading">
        <div class="preview-pane">
          <pre v-if="previewResult" class="preview-content">{{ previewResult }}</pre>
          <n-empty v-else-if="!previewLoading" :description="t('common.noPreview')" style="padding: 20px 0" />
        </div>
      </n-spin>
      <template #action>
        <n-space justify="end">
          <n-button @click="showPreviewModal = false">{{ t('common.close') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.tmpl-page { display: flex; flex-direction: column; gap: 16px; }

.tmpl-header {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: 16px;
}
.tmpl-title { font-size: 18px; font-weight: 600; margin: 0 0 4px; color: var(--sre-text-primary); }
.tmpl-subtitle { font-size: 12px; color: var(--sre-text-secondary); margin: 0; }

.toolbar {
  display: flex; align-items: center; gap: 8px;
  padding-bottom: 12px;
  border-bottom: var(--sre-hairline);
}
.count {
  margin-left: auto;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  font-variant-numeric: tabular-nums;
}

.empty-state {
  display: flex; flex-direction: column; align-items: center;
  gap: 12px;
  padding: 64px 0;
  color: var(--sre-text-tertiary);
}
.empty-state p { margin: 0; font-size: 14px; }

.tmpl-list { display: flex; flex-direction: column; gap: 6px; }

.tmpl-row {
  display: flex; align-items: stretch; gap: 14px;
  padding: 14px 18px;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
  cursor: pointer;
  transition: background var(--sre-duration-fast) var(--sre-ease-out),
              border-color var(--sre-duration-fast) var(--sre-ease-out);
}
.tmpl-row:hover {
  background: var(--sre-bg-hover);
  border-color: rgba(255,255,255,0.14);
}

.tmpl-main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 6px; }

.tmpl-head { display: flex; align-items: center; gap: 8px; }
.tmpl-name { font-size: 14px; font-weight: 600; color: var(--sre-text-primary); }
.tmpl-type-chip {
  font-size: 11px; font-weight: 500;
  padding: 2px 8px; border-radius: 4px;
  letter-spacing: 0.3px;
  font-family: var(--sre-font-mono);
}
.tmpl-type-chip[data-type="text"]     { background: var(--sre-bg-elevated); color: var(--sre-text-secondary); }
.tmpl-type-chip[data-type="html"]     { background: var(--sre-info-soft); color: var(--sre-info); }
.tmpl-type-chip[data-type="markdown"] { background: var(--sre-primary-soft); color: var(--sre-primary); }
.tmpl-type-chip[data-type="lark"]     { background: rgba(99,102,241,0.14); color: #818cf8; }
.tmpl-builtin {
  font-size: 11px; padding: 2px 8px; border-radius: 4px;
  background: var(--sre-bg-elevated);
  color: var(--sre-text-tertiary);
  letter-spacing: 0.3px;
  text-transform: uppercase;
}

.tmpl-desc {
  font-size: 12px;
  color: var(--sre-text-secondary);
  line-height: 1.5;
}

.tmpl-content {
  font-family: var(--sre-font-mono); font-size: 11px;
  background: var(--sre-bg-elevated);
  border-radius: 4px;
  padding: 4px 8px;
  color: var(--sre-text-tertiary);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  display: block;
}

.tmpl-actions {
  display: flex; align-items: center; flex-shrink: 0;
}

.preview-pane {
  background: var(--sre-bg-elevated);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
  min-height: 120px;
  padding: 16px;
}
.preview-content {
  font-family: var(--sre-font-mono);
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
  color: var(--sre-text-primary);
  line-height: 1.6;
}
</style>
