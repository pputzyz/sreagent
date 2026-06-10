<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, h, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import {
  NButton, NSpace, NDataTable, NInput, NSelect, NDrawer, NDrawerContent,
  NForm, NFormItem, NTag, NSwitch, NIcon, NInputNumber, NEmpty, NPagination,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { AddOutline, TrashOutline, CopyOutline, CreateOutline, SearchOutline } from '@vicons/ionicons5'
import { savedViewApi } from '@/api/saved-views'
import type { SavedViewApiItem } from '@/api/saved-views'
import { usePaginatedList, usePermissions } from '@/composables'
import { getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const { hasPerm } = usePermissions()

// Filters
const searchQuery = ref('')
const filterTab = ref<string | null>(null)
const filterPublic = ref<string | null>(null)

// Pagination & data
const {
  loading,
  items: views,
  total,
  page,
  pageSize,
  fetchList,
  refresh,
} = usePaginatedList<SavedViewApiItem>({
  apiFn: savedViewApi.list,
  pageSize: 20,
  extraParams: () => {
    const params: Record<string, unknown> = {}
    if (filterTab.value) params.tab = filterTab.value
    if (filterPublic.value === 'true') params.is_public = true
    else if (filterPublic.value === 'false') params.is_public = false
    return params
  },
  onError: (err: unknown) => {
    message.error(getErrorMessage(err))
  },
})

// Debounced search
let searchTimer: ReturnType<typeof setTimeout> | null = null
watch(searchQuery, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { page.value = 1; fetchList() }, 300)
})
watch([filterTab, filterPublic], () => {
  page.value = 1
  fetchList()
})

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

// Drawer state
const showDrawer = ref(false)
const drawerMode = ref<'create' | 'edit'>('create')
const saving = ref(false)
const form = ref({
  name: '',
  description: '',
  tab: 'metrics' as string,
  datasource_id: null as number | null,
  expression: '',
  query_config: '',
  is_public: false,
})

// Tab options
const tabOptions = computed(() => [
  { label: t('common.all'), value: '' },
  { label: t('savedViews.metrics'), value: 'metrics' },
  { label: t('savedViews.logs'), value: 'logs' },
])

const publicOptions = computed(() => [
  { label: t('common.all'), value: '' },
  { label: t('savedViews.isPublic'), value: 'true' },
  { label: t('common.no'), value: 'false' },
])

const tabSelectOptions = computed(() => [
  { label: t('savedViews.metrics'), value: 'metrics' },
  { label: t('savedViews.logs'), value: 'logs' },
])

// Table columns
const columns = computed<DataTableColumns<SavedViewApiItem>>(() => [
  {
    title: t('common.name'),
    key: 'name',
    ellipsis: { tooltip: true },
    minWidth: 180,
  },
  {
    title: t('savedViews.tab'),
    key: 'tab',
    width: 100,
    render: (row) => h(NTag, {
      type: row.tab === 'metrics' ? 'success' : 'info',
      size: 'small',
      bordered: false,
      round: true,
    }, { default: () => row.tab === 'metrics' ? t('savedViews.metrics') : t('savedViews.logs') }),
  },
  {
    title: t('datasource.title'),
    key: 'datasource_id',
    width: 100,
    render: (row) => h('span', { class: 'tnum' }, row.datasource_id || '—'),
  },
  {
    title: t('savedViews.expression'),
    key: 'expression',
    ellipsis: { tooltip: true },
    minWidth: 200,
    render: (row) => h('span', { style: 'font-family: var(--sre-font-mono, monospace); font-size: 12px; color: var(--sre-text-secondary)' }, row.expression),
  },
  {
    title: t('savedViews.isPublic'),
    key: 'is_public',
    width: 80,
    render: (row) => h(NTag, {
      type: row.is_public ? 'success' : 'default',
      size: 'small',
      bordered: false,
      round: true,
    }, { default: () => row.is_public ? t('common.yes') : t('common.no') }),
  },
  {
    title: t('common.createdAt'),
    key: 'created_at',
    width: 160,
    render: (row) => h('span', { class: 'tnum', style: 'font-size: 12px; color: var(--sre-text-tertiary)' }, row.created_at ? new Date(row.created_at).toLocaleString() : '—'),
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 120,
    fixed: 'right',
    render: (row) => h(NSpace, { size: 'small', justify: 'center' }, {
      default: () => [
        h(NButton, {
          quaternary: true,
          circle: true,
          size: 'small',
          title: t('common.edit'),
          disabled: !hasPerm('rules.manage'),
          onClick: (e: MouseEvent) => { e.stopPropagation(); openEdit(row) },
        }, { icon: () => h(NIcon, { component: CreateOutline }) }),
        h(NButton, {
          quaternary: true,
          circle: true,
          size: 'small',
          title: t('savedViews.copy'),
          disabled: !hasPerm('rules.manage'),
          onClick: (e: MouseEvent) => { e.stopPropagation(); handleCopy(row) },
        }, { icon: () => h(NIcon, { component: CopyOutline }) }),
        h(NButton, {
          quaternary: true,
          circle: true,
          size: 'small',
          title: t('common.delete'),
          type: 'error',
          disabled: !hasPerm('rules.manage'),
          onClick: (e: MouseEvent) => { e.stopPropagation(); handleDelete(row) },
        }, { icon: () => h(NIcon, { component: TrashOutline }) }),
      ],
    }),
  },
])

// Drawer operations
function resetForm() {
  form.value = {
    name: '',
    description: '',
    tab: 'metrics',
    datasource_id: null,
    expression: '',
    query_config: '',
    is_public: false,
  }
}

function openCreate() {
  resetForm()
  drawerMode.value = 'create'
  showDrawer.value = true
}

function openEdit(row: SavedViewApiItem) {
  form.value = {
    name: row.name,
    description: row.description || '',
    tab: row.tab,
    datasource_id: row.datasource_id || null,
    expression: row.expression,
    query_config: row.query_config || '',
    is_public: row.is_public,
  }
  drawerMode.value = 'edit'
  editingId.value = row.id
  showDrawer.value = true
}

const editingId = ref<number | null>(null)

async function handleSave() {
  if (!form.value.name.trim()) {
    message.warning(t('alert.nameRequired'))
    return
  }
  if (!form.value.expression.trim()) {
    message.warning(t('alert.expressionRequired'))
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      description: form.value.description.trim() || undefined,
      tab: form.value.tab,
      datasource_id: form.value.datasource_id || undefined,
      expression: form.value.expression.trim(),
      query_config: form.value.query_config.trim() || undefined,
      is_public: form.value.is_public,
    }
    if (drawerMode.value === 'create') {
      await savedViewApi.create(payload as any)
      message.success(t('common.createSuccess'))
    } else {
      await savedViewApi.update(editingId.value!, payload as any)
      message.success(t('common.updateSuccess'))
    }
    showDrawer.value = false
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function handleCopy(row: SavedViewApiItem) {
  try {
    await savedViewApi.copy(row.id)
    message.success(t('common.copied'))
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

function handleDelete(row: SavedViewApiItem) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('common.confirmDeleteMsg'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await savedViewApi.delete(row.id)
        message.success(t('common.deleteSuccess'))
        fetchList()
      } catch (err: unknown) {
        message.error(getErrorMessage(err))
      }
    },
  })
}

onMounted(() => {
  fetchList()
})
</script>

<template>
  <div class="saved-views-page">
    <PageHeader :title="t('savedViews.title')" :subtitle="t('savedViews.subtitle')">
      <template #actions>
        <n-button v-if="hasPerm('rules.manage')" size="small" type="primary" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('common.create') }}
        </n-button>
      </template>
    </PageHeader>

    <!-- Toolbar -->
    <div class="toolbar">
      <n-input
        v-model:value="searchQuery"
        size="small"
        :placeholder="t('common.search')"
        clearable
        class="toolbar-search"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select
        v-model:value="filterTab"
        size="small"
        :options="tabOptions"
        :placeholder="t('savedViews.tab')"
        clearable
        class="toolbar-select"
      />
      <n-select
        v-model:value="filterPublic"
        size="small"
        :options="publicOptions"
        :placeholder="t('savedViews.isPublic')"
        clearable
        class="toolbar-select"
      />
    </div>

    <!-- Empty state -->
    <NEmpty
      v-if="!loading && views.length === 0"
      :description="t('common.noData')"
      style="margin-top: 80px"
    />

    <!-- Data table -->
    <n-data-table
      v-else
      :columns="columns"
      :data="views"
      :loading="loading"
      :bordered="false"
      :single-line="false"
      size="small"
      :row-key="(row: SavedViewApiItem) => row.id"
      style="margin-top: 8px"
    />

    <!-- Pagination -->
    <div v-if="total > 0" class="pagination-wrap">
      <n-pagination
        v-model:page="page"
        :page-size="pageSize"
        :item-count="total"
        :page-slot="7"
        @update:page="fetchList"
      />
    </div>

    <!-- Create/Edit Drawer -->
    <n-drawer v-model:show="showDrawer" :width="480" placement="right">
      <n-drawer-content :title="drawerMode === 'create' ? t('common.create') : t('common.edit')">
        <n-form label-placement="top" size="small">
          <n-form-item :label="t('common.name')" required>
            <n-input
              v-model:value="form.name"
              :placeholder="t('placeholder.viewName')"
              maxlength="200"
              show-count
            />
          </n-form-item>
          <n-form-item :label="t('common.description')">
            <n-input
              v-model:value="form.description"
              type="textarea"
              :placeholder="t('placeholder.viewDescription')"
              maxlength="500"
              show-count
              :rows="2"
            />
          </n-form-item>
          <n-form-item :label="t('savedViews.tab')" required>
            <n-select
              v-model:value="form.tab"
              :options="tabSelectOptions"
            />
          </n-form-item>
          <n-form-item :label="t('datasource.title')">
            <n-input-number
              v-model:value="form.datasource_id"
              :placeholder="t('datasource.title')"
              :min="0"
              style="width: 100%"
            />
          </n-form-item>
          <n-form-item :label="t('savedViews.expression')" required>
            <n-input
              v-model:value="form.expression"
              type="textarea"
              :placeholder="t('savedViews.expression')"
              :rows="4"
              style="font-family: var(--sre-font-mono, monospace)"
            />
          </n-form-item>
          <n-form-item :label="t('common.labels') + ' (JSON)'">
            <n-input
              v-model:value="form.query_config"
              type="textarea"
              :placeholder="'{}'"
              :rows="3"
              style="font-family: var(--sre-font-mono, monospace)"
            />
          </n-form-item>
          <n-form-item :label="t('savedViews.isPublic')">
            <n-switch v-model:value="form.is_public" />
          </n-form-item>
        </n-form>
        <template #footer>
          <n-space justify="end">
            <n-button size="small" @click="showDrawer = false">{{ t('common.cancel') }}</n-button>
            <n-button size="small" type="primary" :loading="saving" @click="handleSave">
              {{ t('common.save') }}
            </n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<style scoped>
.saved-views-page {
  max-width: 1400px;
  font-family: var(--sre-font-sans);
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 0;
  margin-bottom: 4px;
}

.toolbar-search {
  width: 240px;
}

.toolbar-select {
  width: 160px;
}

.pagination-wrap {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.tnum {
  font-variant-numeric: tabular-nums;
}
</style>
