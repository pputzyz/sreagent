<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import {
  NButton, NIcon, NInput, NSelect, NTag, NDrawer, NDrawerContent,
  NForm, NFormItem, NInputNumber, NDataTable, NPagination, NEmpty, NSpace,
} from 'naive-ui'
import {
  AddOutline, SearchOutline, ThumbsUpOutline,
} from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import { knowledgeApi, type KnowledgeDocument, type CreateKnowledgeRequest } from '@/api/knowledge'
import { getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

// --- State ---
const docs = ref<KnowledgeDocument[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const search = ref('')
const sourceFilter = ref<string | null>(null)

// Drawer
const showDrawer = ref(false)
const drawerMode = ref<'create' | 'edit'>('create')
const editingId = ref<number | null>(null)
const saving = ref(false)

// Form
const form = ref<CreateKnowledgeRequest & { id?: number }>({
  title: '',
  content: '',
  source: 'sop',
  tags: [],
})

const tagsInput = ref('')

// --- Options ---
const sourceOptions = computed(() => [
  { label: t('knowledge.source.sop'), value: 'sop' },
  { label: t('knowledge.source.incidentCase'), value: 'incident_case' },
  { label: t('knowledge.source.runbook'), value: 'runbook' },
  { label: t('knowledge.source.template'), value: 'template' },
  { label: t('knowledge.source.wiki'), value: 'wiki' },
])

const filterSourceOptions = computed(() => [
  { label: t('common.all'), value: '' },
  ...sourceOptions.value,
])

function getSourceLabel(source: string): string {
  const opt = sourceOptions.value.find(o => o.value === source)
  return opt?.label || source
}

function getSourceColor(source: string): string {
  const map: Record<string, string> = {
    sop: 'success',
    incident_case: 'error',
    runbook: 'info',
    template: 'warning',
    wiki: 'default',
  }
  return map[source] || 'default'
}

// --- API ---
async function fetchDocs() {
  loading.value = true
  try {
    const resp = await knowledgeApi.list({
      page: page.value,
      page_size: pageSize.value,
      source: sourceFilter.value || undefined,
      search: search.value.trim() || undefined,
    })
    docs.value = resp.data.data?.list || []
    total.value = resp.data.data?.total || 0
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

function handlePageChange(p: number) {
  page.value = p
  fetchDocs()
}

function handlePageSizeChange(ps: number) {
  pageSize.value = ps
  page.value = 1
  fetchDocs()
}

function handleSearch() {
  page.value = 1
  fetchDocs()
}

function handleSourceFilterChange(val: string) {
  sourceFilter.value = val || null
  page.value = 1
  fetchDocs()
}

// --- CRUD ---
function openCreate() {
  drawerMode.value = 'create'
  editingId.value = null
  resetForm()
  showDrawer.value = true
}

function openEdit(doc: KnowledgeDocument) {
  drawerMode.value = 'edit'
  editingId.value = doc.id
  fillForm(doc)
  showDrawer.value = true
}

function resetForm() {
  form.value = {
    title: '',
    content: '',
    source: 'sop',
    tags: [],
  }
  tagsInput.value = ''
}

function fillForm(doc: KnowledgeDocument) {
  form.value = {
    title: doc.title,
    content: doc.content,
    source: doc.source,
    tags: doc.tags || [],
  }
  tagsInput.value = (doc.tags || []).join(', ')
}

function parseTags() {
  form.value.tags = tagsInput.value
    .split(',')
    .map(s => s.trim())
    .filter(Boolean)
}

async function handleSave() {
  if (!form.value.title?.trim()) {
    message.warning(t('knowledge.titleRequired'))
    return
  }
  if (!form.value.content?.trim()) {
    message.warning(t('knowledge.contentRequired'))
    return
  }
  parseTags()
  saving.value = true
  try {
    const payload: CreateKnowledgeRequest = {
      title: form.value.title,
      content: form.value.content,
      source: form.value.source,
      tags: form.value.tags,
    }
    if (drawerMode.value === 'edit' && editingId.value) {
      await knowledgeApi.update(editingId.value, payload)
      message.success(t('knowledge.updateSuccess'))
    } else {
      await knowledgeApi.create(payload)
      message.success(t('knowledge.createSuccess'))
    }
    showDrawer.value = false
    fetchDocs()
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    saving.value = false
  }
}

function confirmDelete(doc: KnowledgeDocument) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('knowledge.confirmDelete', { title: doc.title }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await knowledgeApi.delete(doc.id)
        message.success(t('knowledge.deleteSuccess'))
        fetchDocs()
      } catch (e: unknown) {
        message.error(getErrorMessage(e))
      }
    },
  })
}

async function handleHelpful(doc: KnowledgeDocument) {
  try {
    await knowledgeApi.markHelpful(doc.id)
    doc.helpful_count++
    message.success(t('knowledge.helpfulThanks'))
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  }
}

// --- Columns ---
const columns = computed<DataTableColumns<KnowledgeDocument>>(() => [
  {
    title: t('knowledge.title'),
    key: 'title',
    minWidth: 200,
    ellipsis: { tooltip: true },
    render: (row) =>
      h('a', {
        style: 'color: var(--sre-primary); cursor: pointer; text-decoration: none;',
        onClick: () => openEdit(row),
      }, row.title),
  },
  {
    title: t('knowledge.source.label'),
    key: 'source',
    width: 130,
    render: (row) =>
      h(NTag, {
        size: 'small',
        type: getSourceColor(row.source) as any,
        bordered: false,
      }, () => getSourceLabel(row.source)),
  },
  {
    title: t('knowledge.tags'),
    key: 'tags',
    minWidth: 150,
    render: (row) => {
      if (!row.tags || row.tags.length === 0) return '-'
      return h(NSpace, { size: 4 }, () =>
        row.tags.slice(0, 3).map(tag =>
          h(NTag, { size: 'tiny', bordered: false, round: true }, () => tag)
        )
      )
    },
  },
  {
    title: t('knowledge.helpfulCount'),
    key: 'helpful_count',
    width: 100,
    render: (row) =>
      h('div', { style: 'display: flex; align-items: center; gap: 4px;' }, [
        h(NIcon, { size: 14, component: ThumbsUpOutline }),
        h('span', { class: 'tnum' }, String(row.helpful_count)),
      ]),
  },
  {
    title: t('common.createdAt'),
    key: 'created_at',
    width: 170,
    render: (row) => row.created_at ? new Date(row.created_at).toLocaleString() : '-',
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 200,
    fixed: 'right',
    render: (row) =>
      h(NSpace, { size: 'small' }, () => [
        h(NButton, {
          size: 'tiny',
          quaternary: true,
          onClick: () => handleHelpful(row),
        }, {
          icon: () => h(NIcon, { size: 14, component: ThumbsUpOutline }),
          default: () => t('knowledge.helpful'),
        }),
        h(NButton, {
          size: 'tiny',
          quaternary: true,
          type: 'primary',
          onClick: () => openEdit(row),
        }, () => t('common.edit')),
        h(NButton, {
          size: 'tiny',
          quaternary: true,
          type: 'error',
          onClick: () => confirmDelete(row),
        }, () => t('common.delete')),
      ]),
  },
])

// --- Init ---
onMounted(fetchDocs)
</script>

<template>
  <div class="knowledge-page">
    <PageHeader :title="t('knowledge.title')" :subtitle="t('knowledge.subtitle')">
      <template #actions>
        <n-button type="primary" size="small" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('common.create') }}
        </n-button>
      </template>
    </PageHeader>

    <div class="toolbar">
      <n-input
        v-model:value="search"
        size="small"
        :placeholder="t('common.search')"
        clearable
        style="width: 240px"
        @update:value="handleSearch"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select
        :value="sourceFilter"
        :options="filterSourceOptions"
        size="small"
        clearable
        :placeholder="t('knowledge.source.label')"
        style="width: 160px"
        @update:value="handleSourceFilterChange"
      />
      <span class="count tnum">{{ docs.length }} / {{ total }}</span>
    </div>

    <n-empty v-if="!loading && docs.length === 0" :description="t('common.noData')" style="padding: 60px 0">
      <template #extra>
        <n-button type="primary" size="small" @click="openCreate">{{ t('common.create') }}</n-button>
      </template>
    </n-empty>

    <template v-else>
      <n-data-table
        :columns="columns"
        :data="docs"
        :loading="loading"
        :row-key="(row: KnowledgeDocument) => row.id"
        size="small"
        :bordered="false"
        striped
        :scroll-x="1000"
      />

      <div class="page-pagination" v-if="total > 0">
        <n-pagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :item-count="total"
          :page-sizes="[20, 50, 100]"
          show-size-picker
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </template>

    <!-- Create/Edit Drawer -->
    <n-drawer v-model:show="showDrawer" :width="560">
      <n-drawer-content :title="drawerMode === 'edit' ? t('common.edit') : t('common.create')">
        <n-form label-placement="top">
          <n-form-item :label="t('knowledge.title')" required>
            <n-input
              v-model:value="form.title"
              :placeholder="t('knowledge.titlePlaceholder')"
            />
          </n-form-item>

          <n-form-item :label="t('knowledge.source.label')">
            <n-select
              v-model:value="form.source"
              :options="sourceOptions"
            />
          </n-form-item>

          <n-form-item :label="t('knowledge.tags')">
            <n-input
              v-model:value="tagsInput"
              :placeholder="t('knowledge.tagsPlaceholder')"
              @blur="parseTags"
            />
          </n-form-item>

          <n-form-item :label="t('knowledge.content')" required>
            <n-input
              v-model:value="form.content"
              type="textarea"
              :rows="12"
              :placeholder="t('knowledge.contentPlaceholder')"
            />
          </n-form-item>
        </n-form>

        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 8px;">
            <n-button @click="showDrawer = false">{{ t('common.cancel') }}</n-button>
            <n-button type="primary" :loading="saving" @click="handleSave">
              {{ drawerMode === 'edit' ? t('common.update') : t('common.create') }}
            </n-button>
          </div>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<style scoped>
.knowledge-page {
  padding: 16px;
  max-width: 1400px;
}
.toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}
.count {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin-left: auto;
  font-variant-numeric: tabular-nums;
}
.page-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
