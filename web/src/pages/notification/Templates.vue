<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, h, type Ref } from 'vue'
import { useMessage, useDialog, NIcon, NButton, NDropdown } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { messageTemplateApi } from '@/api'
import type { MessageTemplate } from '@/types'
import { getErrorMessage } from '@/utils/format'
import { useCrudPage } from '@/composables/useCrudPage'
import type { CrudApiModule } from '@/composables/useCrudPage'
import {
  AddOutline, SearchOutline, EllipsisHorizontalOutline,
  CreateOutline, EyeOutline, TrashOutline, DocumentTextOutline,
} from '@vicons/ionicons5'
import EmptyState from '@/components/common/EmptyState.vue'
import PageHeader from '@/components/common/PageHeader.vue'

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

interface TemplateForm {
  name: string
  description: string
  type: 'text' | 'html' | 'markdown' | 'lark_card'
  content: string
}

const crud = useCrudPage<MessageTemplate>({
  api: messageTemplateApi as unknown as CrudApiModule<MessageTemplate>,
  defaultForm: () => ({
    name: '', description: '',
    type: 'text' as 'text' | 'html' | 'markdown' | 'lark_card',
    content: '',
  } as unknown as Partial<MessageTemplate>),
  i18nKeys: {
    created: 'template.created',
    updated: 'template.updated',
    deleted: 'template.deleted',
    deleteConfirm: 'template.deleteConfirm',
    createTitle: 'template.create',
    editTitle: 'template.edit',
  },
  rowToForm: (row) => ({
    name: row.name, description: row.description,
    type: row.type, content: row.content || '',
  } as unknown as Partial<MessageTemplate>),
  formToPayload: (form) => ({
    name: form.name, description: form.description,
    type: form.type, content: form.content,
  }),
  validate: (form) => {
    if (!form.name?.trim()) return t('template.nameRequired')
    const f = form as unknown as TemplateForm
    if (!f.content?.trim()) return t('template.contentRequired')
    return null
  },
  pageSize: 100,
})

const {
  loading,
  items: templates,
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
const form = crud.form as Ref<TemplateForm>

const typeFilter = ref<string>('')

// Preview state
const showPreviewModal = ref(false)
const previewLoading = ref(false)
const previewResult = ref('')

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

async function handlePreview(row: MessageTemplate) {
  previewLoading.value = true
  previewResult.value = ''
  showPreviewModal.value = true
  try {
    const { data } = await messageTemplateApi.preview({ content: row.content, type: row.type })
    previewResult.value = data.data.rendered
  } catch (err: unknown) {
    previewResult.value = ''
    message.error(t('template.previewFailed') + ': ' + getErrorMessage(err))
  } finally {
    previewLoading.value = false
  }
}

async function handlePreviewFromForm() {
  previewLoading.value = true
  previewResult.value = ''
  showPreviewModal.value = true
  try {
    const { data } = await messageTemplateApi.preview({ content: form.value.content, type: form.value.type })
    previewResult.value = data.data.rendered
  } catch (err: unknown) {
    previewResult.value = ''
    message.error(t('template.previewFailed') + ': ' + getErrorMessage(err))
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
    })
  }
  return actions
}

function handleAction(key: string, row: MessageTemplate) {
  if (key === 'edit') openEdit(row)
  else if (key === 'preview') handlePreview(row)
  else if (key === 'delete') {
    dialog.warning({
      title: t('common.confirmDelete'),
      content: t('template.deleteConfirm'),
      positiveText: t('common.confirm'),
      negativeText: t('common.cancel'),
      onPositiveClick: () => confirmDelete(row.id),
    })
  }
}

// Truncate template content preview
function contentPreview(content: string | null | undefined): string {
  if (!content) return '—'
  const trimmed = content.trim().replace(/\s+/g, ' ')
  return trimmed.length > 80 ? trimmed.slice(0, 80) + '...' : trimmed
}

const availableVariables = '{{.AlertName}} {{.Severity}} {{.Status}} {{.Labels}} {{.Annotations}} {{.FiredAt}} {{.Value}} {{.Duration}} {{.RuleName}}'

const templateVars = [
  { name: '{{.AlertName}}', desc: 'template.varAlertName' },
  { name: '{{.Severity}}', desc: 'template.varSeverity' },
  { name: '{{.Status}}', desc: 'template.varStatus' },
  { name: '{{.Labels.xxx}}', desc: 'template.varLabels' },
  { name: '{{.Summary}}', desc: 'template.varSummary' },
  { name: '{{.FiredAt}}', desc: 'template.varFiredAt' },
  { name: '{{.Value}}', desc: 'template.varValue' },
  { name: '{{.Duration}}', desc: 'template.varDuration' },
  { name: '{{.RuleName}}', desc: 'template.varRuleName' },
  { name: '{{.Annotations}}', desc: 'template.varAnnotations' },
]

async function copyVar(name: string) {
  try {
    await navigator.clipboard.writeText(name)
    message.success(t('common.copied'))
  } catch {
    message.error(t('common.copyFailed'))
  }
}

onMounted(fetchList)
</script>

<template>
  <div class="tmpl-page">
    <PageHeader :title="t('template.title')" :subtitle="t('template.subtitle')">
      <template #actions>
        <n-button type="primary" size="small" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('template.create') }}
        </n-button>
      </template>
    </PageHeader>

    <div class="toolbar">
      <n-input
        v-model:value="search" size="small" clearable
        :placeholder="t('common.search')" class="tmpl-search-input"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select
        v-model:value="typeFilter" size="small"
        :options="filterTypeOptions" class="tmpl-type-select"
      />
      <span class="count tnum">{{ filtered.length }} / {{ templates.length }}</span>
    </div>

    <div v-if="loading" class="tmpl-loading">{{ t('common.loading') }}</div>
    <EmptyState
      v-else-if="filtered.length === 0"
      :icon="DocumentTextOutline"
      :title="t('template.noData')"
      :primary-text="t('template.create')"
      @primary="openCreate"
    />

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
    <n-modal v-model:show="showModal" preset="card" :title="modalTitle" :bordered="false" class="tmpl-modal">
      <n-form label-placement="top">
        <n-grid :x-gap="12" :cols="2">
          <n-gi>
            <n-form-item :label="t('template.name')" required>
              <n-input v-model:value="form.name" :placeholder="t('templateMgmt.namePlaceholder')" />
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

        <div class="var-section">
          <div class="var-title">{{ t('template.availableVariables') }}</div>
          <div class="var-grid">
            <div
              v-for="v in templateVars"
              :key="v.name"
              class="var-chip"
              :title="t(v.desc)"
              @click="copyVar(v.name)"
            >
              <code class="var-name">{{ v.name }}</code>
              <span class="var-desc">{{ t(v.desc) }}</span>
            </div>
          </div>
          <div class="var-hint">{{ t('template.clickToCopy') }}</div>
        </div>

        <n-form-item :label="t('template.content')" class="tmpl-content-field">
          <n-input
            v-model:value="form.content" type="textarea" :rows="12"
            :placeholder="t('common.enterContent')"
            class="tmpl-content-input"
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
    <n-modal v-model:show="showPreviewModal" preset="card" :title="t('template.previewResult')" :bordered="false" class="tmpl-modal">
      <n-spin :show="previewLoading">
        <div class="preview-pane">
          <pre v-if="previewResult" class="preview-content">{{ previewResult }}</pre>
          <EmptyState
            v-else-if="!previewLoading"
            :title="t('common.noPreview')"
            size="sm"
          />
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

.tmpl-loading {
  display: flex; flex-direction: column; align-items: center;
  padding: 64px 0;
  color: var(--sre-text-tertiary);
  font-size: 14px;
}

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
  border-color: var(--sre-border-strong);
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
.tmpl-type-chip[data-type="lark"]     { background: var(--sre-info-soft); color: var(--sre-info); }
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

/* Toolbar */
.tmpl-search-input { width: 240px; }
.tmpl-type-select { width: 140px; }

/* Modal */
.tmpl-modal { width: 600px; }
.tmpl-content-field { margin-top: 12px; }
.tmpl-content-input { font-family: var(--sre-font-mono); font-size: 12px; }

/* Variable hint section */
.var-section {
  margin-bottom: 14px;
  padding: 12px 14px;
  background: var(--sre-bg-elevated, rgba(255,255,255,0.03));
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
}
.var-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  margin-bottom: 10px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.var-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.var-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: var(--sre-bg-card, rgba(0,0,0,0.12));
  border: 1px solid var(--sre-border);
  border-radius: 6px;
  cursor: pointer;
  transition: border-color 120ms, background 120ms;
}
.var-chip:hover {
  border-color: var(--sre-primary);
  background: var(--sre-primary-soft);
}
.var-name {
  font-family: var(--sre-font-mono, monospace);
  font-size: 11px;
  color: var(--sre-primary);
  white-space: nowrap;
}
.var-desc {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  white-space: nowrap;
}
.var-hint {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  margin-top: 8px;
}
</style>
