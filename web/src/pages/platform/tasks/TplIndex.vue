<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useMessage, useDialog } from 'naive-ui'
import {
  NButton, NIcon, NInput, NTag, NDrawer, NDrawerContent,
  NForm, NFormItem, NInputNumber, NSelect, NDataTable, NPagination,
  NEmpty, NSpace, NTooltip,
} from 'naive-ui'
import {
  AddOutline, SearchOutline, PlayOutline, FlashOutline,
} from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import { taskTplApi, taskApi, type TaskTpl, type CreateTaskTplRequest } from '@/api/task'
import { getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const router = useRouter()

// --- State ---
const tpls = ref<TaskTpl[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const search = ref('')

// Drawer
const showDrawer = ref(false)
const drawerMode = ref<'create' | 'edit'>('create')
const editingId = ref<number | null>(null)
const saving = ref(false)

// Execute modal
const showExecuteModal = ref(false)
const executingTpl = ref<TaskTpl | null>(null)
const executeHosts = ref('')
const executing = ref(false)

// Form
const form = ref<CreateTaskTplRequest>({
  name: '',
  script: '',
  args: '',
  batch: 0,
  tolerance: 0,
  timeout: 60,
  account: '',
  pause: '',
  hosts: '[]',
  tags: '[]',
  note: '',
})

// --- Filtered ---
const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return tpls.value
  return tpls.value.filter(tpl =>
    tpl.name.toLowerCase().includes(q) ||
    (tpl.note || '').toLowerCase().includes(q) ||
    (tpl.create_by || '').toLowerCase().includes(q)
  )
})

// --- Helpers ---
function parseJsonArray(str: string): string[] {
  try {
    const arr = JSON.parse(str || '[]')
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
}

// --- API ---
async function fetchTpls() {
  loading.value = true
  try {
    const resp = await taskTplApi.list({ keyword: search.value, page: page.value, page_size: pageSize.value })
    tpls.value = resp.data.data?.list || []
    total.value = resp.data.data?.total || 0
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

function handlePageChange(p: number) {
  page.value = p
  fetchTpls()
}

function handlePageSizeChange(ps: number) {
  pageSize.value = ps
  page.value = 1
  fetchTpls()
}

// --- CRUD ---
function openCreate() {
  drawerMode.value = 'create'
  editingId.value = null
  resetForm()
  showDrawer.value = true
}

function openEdit(tpl: TaskTpl) {
  drawerMode.value = 'edit'
  editingId.value = tpl.id
  fillForm(tpl)
  showDrawer.value = true
}

function resetForm() {
  form.value = {
    name: '',
    script: '',
    args: '',
    batch: 0,
    tolerance: 0,
    timeout: 60,
    account: '',
    pause: '',
    hosts: '[]',
    tags: '[]',
    note: '',
  }
}

function fillForm(tpl: TaskTpl) {
  form.value = {
    name: tpl.name,
    script: tpl.script,
    args: tpl.args,
    batch: tpl.batch,
    tolerance: tpl.tolerance,
    timeout: tpl.timeout,
    account: tpl.account,
    pause: tpl.pause,
    hosts: tpl.hosts || '[]',
    tags: tpl.tags || '[]',
    note: tpl.note,
  }
}

async function handleSave() {
  if (!form.value.name?.trim()) {
    message.warning(t('taskTpl.nameRequired'))
    return
  }
  if (!form.value.script?.trim()) {
    message.warning(t('taskTpl.scriptRequired'))
    return
  }
  saving.value = true
  try {
    if (drawerMode.value === 'edit' && editingId.value) {
      await taskTplApi.update(editingId.value, form.value)
      message.success(t('common.updateSuccess'))
    } else {
      await taskTplApi.create(form.value)
      message.success(t('common.createSuccess'))
    }
    showDrawer.value = false
    fetchTpls()
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    saving.value = false
  }
}

function confirmDelete(tpl: TaskTpl) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('taskTpl.confirmDelete', { name: tpl.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await taskTplApi.delete(tpl.id)
        message.success(t('common.deleteSuccess'))
        fetchTpls()
      } catch (e: unknown) {
        message.error(getErrorMessage(e))
      }
    },
  })
}

// --- Execute ---
function openExecute(tpl: TaskTpl) {
  executingTpl.value = tpl
  const hosts = parseJsonArray(tpl.hosts)
  executeHosts.value = hosts.join('\n')
  showExecuteModal.value = true
}

async function handleExecute() {
  if (!executingTpl.value) return
  const hosts = executeHosts.value.split('\n').map(h => h.trim()).filter(Boolean)
  if (hosts.length === 0) {
    message.warning(t('taskTpl.hostsRequired'))
    return
  }
  executing.value = true
  try {
    await taskApi.execute({ tpl_id: executingTpl.value.id, hosts })
    message.success(t('taskTpl.executeSuccess'))
    showExecuteModal.value = false
    router.push('/platform/tasks')
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    executing.value = false
  }
}

// --- Columns ---
const columns = computed<DataTableColumns<TaskTpl>>(() => [
  {
    title: t('taskTpl.name'),
    key: 'name',
    minWidth: 160,
    ellipsis: { tooltip: true },
    render: (row) =>
      h('a', {
        style: 'color: var(--sre-primary); cursor: pointer; text-decoration: none;',
        onClick: () => openEdit(row),
      }, row.name),
  },
  {
    title: t('taskTpl.script'),
    key: 'script',
    minWidth: 200,
    ellipsis: { tooltip: true },
    render: (row) => {
      const preview = (row.script || '').substring(0, 80)
      return h('code', { style: 'font-size: 12px; color: var(--sre-text-secondary);' }, preview + (row.script.length > 80 ? '...' : ''))
    },
  },
  {
    title: t('taskTpl.batch'),
    key: 'batch',
    width: 80,
    render: (row) => row.batch === 0 ? t('taskTpl.allAtOnce') : String(row.batch),
  },
  {
    title: t('taskTpl.timeout'),
    key: 'timeout',
    width: 80,
    render: (row) => `${row.timeout}s`,
  },
  {
    title: t('taskTpl.account'),
    key: 'account',
    width: 100,
    ellipsis: { tooltip: true },
    render: (row) => row.account || '-',
  },
  {
    title: t('taskTpl.tags'),
    key: 'tags',
    minWidth: 150,
    render: (row) => {
      const tags = parseJsonArray(row.tags)
      if (tags.length === 0) return '-'
      return h(NSpace, { size: 4 }, () =>
        tags.map(tag => h(NTag, { size: 'small', bordered: false, type: 'info' }, () => tag))
      )
    },
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 220,
    fixed: 'right',
    render: (row) =>
      h(NSpace, { size: 'small' }, () => [
        h(NButton, {
          size: 'tiny',
          quaternary: true,
          type: 'primary',
          onClick: () => openExecute(row),
        }, {
          default: () => t('taskTpl.execute'),
          icon: () => h(NIcon, { size: 14 }, { default: () => h(PlayOutline) }),
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
onMounted(fetchTpls)
</script>

<template>
  <div class="task-tpl-page">
    <PageHeader :title="t('taskTpl.title')" :subtitle="t('taskTpl.subtitle')">
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
        style="width: 260px"
        @update:value="fetchTpls"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <span class="count tnum">{{ total }} {{ t('taskTpl.items') }}</span>
    </div>

    <n-empty v-if="!loading && tpls.length === 0" :description="t('taskTpl.noData')" style="padding: 60px 0">
      <template #extra>
        <n-button type="primary" size="small" @click="openCreate">{{ t('common.create') }}</n-button>
      </template>
    </n-empty>

    <template v-else>
      <n-data-table
        :columns="columns"
        :data="filtered"
        :loading="loading"
        :row-key="(row: TaskTpl) => row.id"
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
      <n-drawer-content :title="drawerMode === 'edit' ? t('taskTpl.editTitle') : t('taskTpl.createTitle')">
        <n-form label-placement="top">
          <n-form-item :label="t('taskTpl.name')" required>
            <n-input v-model:value="form.name" :placeholder="t('taskTpl.namePlaceholder')" />
          </n-form-item>

          <n-form-item :label="t('taskTpl.script')" required>
            <n-input
              v-model:value="form.script"
              type="textarea"
              :rows="8"
              :placeholder="t('taskTpl.scriptPlaceholder')"
              style="font-family: monospace;"
            />
          </n-form-item>

          <n-form-item :label="t('taskTpl.args')">
            <n-input v-model:value="form.args" :placeholder="t('taskTpl.argsPlaceholder')" />
          </n-form-item>

          <div style="display: flex; gap: 16px;">
            <n-form-item :label="t('taskTpl.batch')" style="flex: 1;">
              <n-input-number v-model:value="form.batch" :min="0" :placeholder="t('taskTpl.batchPlaceholder')" style="width: 100%" />
            </n-form-item>
            <n-form-item :label="t('taskTpl.tolerance')" style="flex: 1;">
              <n-input-number v-model:value="form.tolerance" :min="0" style="width: 100%" />
            </n-form-item>
            <n-form-item :label="t('taskTpl.timeout')" style="flex: 1;">
              <n-input-number v-model:value="form.timeout" :min="1" :max="3600" style="width: 100%">
                <template #suffix>{{ t('common.seconds') }}</template>
              </n-input-number>
            </n-form-item>
          </div>

          <div style="display: flex; gap: 16px;">
            <n-form-item :label="t('taskTpl.account')" style="flex: 1;">
              <n-input v-model:value="form.account" :placeholder="t('taskTpl.accountPlaceholder')" />
            </n-form-item>
            <n-form-item :label="t('taskTpl.pause')" style="flex: 1;">
              <n-input v-model:value="form.pause" :placeholder="t('taskTpl.pausePlaceholder')" />
            </n-form-item>
          </div>

          <n-form-item :label="t('taskTpl.hosts')">
            <n-input
              v-model:value="form.hosts"
              type="textarea"
              :rows="3"
              :placeholder="t('taskTpl.hostsPlaceholder')"
              style="font-family: monospace;"
            />
          </n-form-item>

          <n-form-item :label="t('taskTpl.tags')">
            <n-input
              v-model:value="form.tags"
              :placeholder="t('taskTpl.tagsPlaceholder')"
            />
          </n-form-item>

          <n-form-item :label="t('taskTpl.note')">
            <n-input
              v-model:value="form.note"
              type="textarea"
              :rows="2"
              :placeholder="t('taskTpl.notePlaceholder')"
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

    <!-- Execute Modal -->
    <n-drawer v-model:show="showExecuteModal" :width="480">
      <n-drawer-content :title="t('taskTpl.executeTitle')">
        <n-form label-placement="top">
          <n-form-item :label="t('taskTpl.template')">
            <n-input :value="executingTpl?.name" disabled />
          </n-form-item>
          <n-form-item :label="t('taskTpl.executeHosts')">
            <n-input
              v-model:value="executeHosts"
              type="textarea"
              :rows="6"
              :placeholder="t('taskTpl.executeHostsPlaceholder')"
              style="font-family: monospace;"
            />
          </n-form-item>
        </n-form>

        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 8px;">
            <n-button @click="showExecuteModal = false">{{ t('common.cancel') }}</n-button>
            <n-button type="primary" :loading="executing" @click="handleExecute">
              <template #icon><n-icon :component="FlashOutline" /></template>
              {{ t('taskTpl.execute') }}
            </n-button>
          </div>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<style scoped>
.task-tpl-page {
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
