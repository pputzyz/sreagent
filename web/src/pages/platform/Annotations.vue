<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import {
  NButton, NIcon, NInput, NDrawer, NDrawerContent,
  NForm, NFormItem, NDatePicker, NDataTable, NPagination, NEmpty, NSpace,
} from 'naive-ui'
import { AddOutline, SearchOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import { annotationApi, type Annotation, type CreateAnnotationRequest } from '@/api/annotation'
import { getErrorMessage } from '@/utils/format'
import { useAuthStore } from '@/stores/auth'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const authStore = useAuthStore()

// --- State ---
const annotations = ref<Annotation[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const search = ref('')
const dashboardFilter = ref('')

// Drawer
const showDrawer = ref(false)
const drawerMode = ref<'create' | 'edit'>('create')
const editingId = ref<number | null>(null)
const saving = ref(false)

// Form
const form = ref({
  id: undefined as number | undefined,
  dashboard_id: '',
  time: new Date().toISOString(),
  text: '',
})

// --- Filtered (client-side for search) ---
const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return annotations.value
  return annotations.value.filter(a =>
    a.text.toLowerCase().includes(q) ||
    (a.dashboard_name || '').toLowerCase().includes(q)
  )
})

// --- API ---
async function fetchAnnotations() {
  loading.value = true
  try {
    const resp = await annotationApi.list({
      page: page.value,
      page_size: pageSize.value,
      dashboard_id: dashboardFilter.value ? Number(dashboardFilter.value) : undefined,
    })
    annotations.value = resp.data.data?.list || []
    total.value = resp.data.data?.total || 0
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

function handlePageChange(p: number) {
  page.value = p
  fetchAnnotations()
}

function handlePageSizeChange(ps: number) {
  pageSize.value = ps
  page.value = 1
  fetchAnnotations()
}

let dashboardFilterTimer: ReturnType<typeof setTimeout> | null = null
function handleDashboardFilter() {
  if (dashboardFilterTimer) clearTimeout(dashboardFilterTimer)
  dashboardFilterTimer = setTimeout(() => {
    page.value = 1
    fetchAnnotations()
  }, 300)
}

onBeforeUnmount(() => {
  if (dashboardFilterTimer) clearTimeout(dashboardFilterTimer)
})

// --- CRUD ---
function openCreate() {
  drawerMode.value = 'create'
  editingId.value = null
  resetForm()
  showDrawer.value = true
}

function openEdit(annotation: Annotation) {
  drawerMode.value = 'edit'
  editingId.value = annotation.id
  fillForm(annotation)
  showDrawer.value = true
}

function resetForm() {
  form.value = {
    id: undefined,
    dashboard_id: '',
    time: new Date().toISOString(),
    text: '',
  }
}

function fillForm(annotation: Annotation) {
  form.value = {
    id: annotation.id,
    dashboard_id: String(annotation.dashboard_id),
    time: annotation.time,
    text: annotation.text,
  }
}

function handleTimeUpdate(value: number | null) {
  if (value) {
    form.value.time = new Date(value).toISOString()
  }
}

async function handleSave() {
  if (!form.value.dashboard_id) {
    message.warning(t('annotations.dashboardIdRequired'))
    return
  }
  if (!form.value.text?.trim()) {
    message.warning(t('annotations.contentRequired'))
    return
  }
  saving.value = true
  try {
    const payload: CreateAnnotationRequest = {
      dashboard_id: Number(form.value.dashboard_id),
      time: form.value.time,
      text: form.value.text,
    }
    if (drawerMode.value === 'edit' && editingId.value) {
      await annotationApi.update(editingId.value, payload)
      message.success(t('annotations.updateSuccess'))
    } else {
      await annotationApi.create(payload)
      message.success(t('annotations.createSuccess'))
    }
    showDrawer.value = false
    fetchAnnotations()
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    saving.value = false
  }
}

function confirmDelete(annotation: Annotation) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('annotations.confirmDelete', { id: annotation.id }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await annotationApi.delete(annotation.id)
        message.success(t('annotations.deleteSuccess'))
        fetchAnnotations()
      } catch (e: unknown) {
        message.error(getErrorMessage(e))
      }
    },
  })
}

// --- Columns ---
const columns = computed<DataTableColumns<Annotation>>(() => [
  {
    title: t('annotations.dashboardId'),
    key: 'dashboard_id',
    width: 120,
    render: (row) => row.dashboard_name
      ? h('span', {}, row.dashboard_name)
      : h('span', { class: 'tnum' }, String(row.dashboard_id)),
  },
  {
    title: t('annotations.time'),
    key: 'time',
    width: 180,
    render: (row) => row.time ? new Date(row.time).toLocaleString() : '-',
  },
  {
    title: t('annotations.content'),
    key: 'text',
    minWidth: 250,
    ellipsis: { tooltip: true },
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
    width: 140,
    fixed: 'right',
    render: (row) =>
      h(NSpace, { size: 'small' }, () => [
        ...(authStore.canManage ? [
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
        ] : []),
      ]),
  },
])

// --- Init ---
onMounted(fetchAnnotations)
</script>

<template>
  <div class="annotations-page">
    <PageHeader :title="t('annotations.title')" :subtitle="t('annotations.subtitle')">
      <template #actions>
        <n-button v-if="authStore.canManage" type="primary" size="small" @click="openCreate">
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
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-input
        v-model:value="dashboardFilter"
        size="small"
        :placeholder="t('annotations.filterDashboardId')"
        clearable
        style="width: 180px"
        @update:value="handleDashboardFilter"
      />
      <span class="count tnum">{{ search.trim() ? `${filtered.length} ${t('common.filtered')}` : total }}</span>
    </div>

    <n-empty v-if="!loading && annotations.length === 0" :description="t('common.noData')" style="padding: 60px 0">
      <template v-if="authStore.canManage" #extra>
        <n-button type="primary" size="small" @click="openCreate">{{ t('common.create') }}</n-button>
      </template>
    </n-empty>

    <template v-else>
      <n-data-table
        :columns="columns"
        :data="filtered"
        :loading="loading"
        :row-key="(row: Annotation) => row.id"
        size="small"
        :bordered="false"
        striped
        :scroll-x="900"
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
    <n-drawer v-model:show="showDrawer" :width="480">
      <n-drawer-content :title="drawerMode === 'edit' ? t('common.edit') : t('common.create')">
        <n-form label-placement="top">
          <n-form-item :label="t('annotations.dashboardId')" required>
            <n-input
              v-model:value="form.dashboard_id"
              :placeholder="t('annotations.dashboardIdPlaceholder')"
            />
          </n-form-item>

          <n-form-item :label="t('annotations.time')">
            <n-date-picker
              :value="new Date(form.time).getTime()"
              type="datetime"
              style="width: 100%"
              @update:value="handleTimeUpdate"
            />
          </n-form-item>

          <n-form-item :label="t('annotations.content')" required>
            <n-input
              v-model:value="form.text"
              type="textarea"
              :rows="6"
              :placeholder="t('annotations.contentPlaceholder')"
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
.annotations-page {
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
