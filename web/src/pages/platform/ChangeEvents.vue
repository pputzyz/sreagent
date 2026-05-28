<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import {
  NButton, NIcon, NTag, NDrawer, NDrawerContent,
  NForm, NFormItem, NInput, NSelect, NSpace, NDataTable,
  NPagination, NEmpty, NDatePicker,
} from 'naive-ui'
import { AddOutline, SearchOutline, TrashOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import { changeEventApi } from '@/api/change-event'
import type { ChangeEvent, IngestChangeEventRequest } from '@/api/change-event'
import { getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const authStore = useAuthStore()

// ─── State ───
const events = ref<ChangeEvent[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const search = ref('')

// Filters
const filterService = ref<string | null>(null)
const filterEnvironment = ref<string | null>(null)
const filterSource = ref<string | null>(null)

// Drawer
const showDrawer = ref(false)
const saving = ref(false)

// Form
const form = ref<IngestChangeEventRequest>({
  source: '',
  change_type: 'deploy',
  service: '',
  environment: 'production',
  commit_sha: '',
  author: '',
  description: '',
  risk_level: 'low',
  metadata: {},
  timestamp: '',
})

// Metadata KV editor
const metadataStr = ref('{}')

// Date picker needs epoch ms (number), form.timestamp is ISO string
const timestampMs = ref<number>(Date.now())

function syncMetadata() {
  try {
    form.value.metadata = JSON.parse(metadataStr.value)
  } catch { /* keep previous value */ }
}

// ─── Options ───
const changeTypeOptions = computed(() => [
  { label: t('changeEvents.deploy'), value: 'deploy' },
  { label: t('changeEvents.config'), value: 'config' },
  { label: t('changeEvents.rollback'), value: 'rollback' },
  { label: t('changeEvents.scaling'), value: 'scaling' },
])

const environmentOptions = computed(() => [
  { label: t('changeEvents.production'), value: 'production' },
  { label: t('changeEvents.staging'), value: 'staging' },
  { label: t('changeEvents.development'), value: 'development' },
  { label: t('changeEvents.testing'), value: 'testing' },
])

const riskLevelOptions = computed(() => [
  { label: t('changeEvents.riskLow'), value: 'low' },
  { label: t('changeEvents.riskMedium'), value: 'medium' },
  { label: t('changeEvents.riskHigh'), value: 'high' },
  { label: t('changeEvents.riskCritical'), value: 'critical' },
])

// ─── Filtered events (client-side for search) ───
const filteredEvents = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return events.value
  return events.value.filter(e =>
    e.source.toLowerCase().includes(q) ||
    e.service.toLowerCase().includes(q) ||
    e.author.toLowerCase().includes(q) ||
    e.change_type.toLowerCase().includes(q) ||
    (e.commit_sha || '').toLowerCase().includes(q)
  )
})

// Unique values for filter dropdowns
const serviceOptions = computed(() => {
  const services = new Set(events.value.map(e => e.service).filter(Boolean))
  return Array.from(services).map(s => ({ label: s, value: s }))
})

const sourceOptions = computed(() => {
  const sources = new Set(events.value.map(e => e.source).filter(Boolean))
  return Array.from(sources).map(s => ({ label: s, value: s }))
})

// ─── API ───
async function fetchEvents() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (filterService.value) params.service = filterService.value
    if (filterEnvironment.value) params.environment = filterEnvironment.value
    if (filterSource.value) params.source = filterSource.value
    const resp = await changeEventApi.list(params as any)
    events.value = resp.data.data?.list || []
    total.value = resp.data.data?.total || 0
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

function handlePageChange(p: number) {
  page.value = p
  fetchEvents()
}

// ─── CRUD ───
function openCreate() {
  resetForm()
  showDrawer.value = true
}

function resetForm() {
  form.value = {
    source: '',
    change_type: 'deploy',
    service: '',
    environment: 'production',
    commit_sha: '',
    author: '',
    description: '',
    risk_level: 'low',
    metadata: {},
    timestamp: '',
  }
  metadataStr.value = '{}'
  timestampMs.value = Date.now()
}

async function handleSave() {
  if (!form.value.source?.trim()) {
    message.warning(t('common.required'))
    return
  }
  if (!form.value.service?.trim()) {
    message.warning(t('common.required'))
    return
  }
  syncMetadata()
  form.value.timestamp = new Date(timestampMs.value).toISOString()
  saving.value = true
  try {
    await changeEventApi.ingest(form.value)
    message.success(t('common.createSuccess'))
    showDrawer.value = false
    fetchEvents()
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    saving.value = false
  }
}

function confirmDelete(ev: ChangeEvent) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('common.confirmDeleteMsg'),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await changeEventApi.delete(ev.id)
        message.success(t('common.deleteSuccess'))
        fetchEvents()
      } catch (e: unknown) {
        message.error(getErrorMessage(e))
      }
    },
  })
}

// ─── Helpers ───
function riskLevelTag(level: string): 'success' | 'warning' | 'error' | 'default' {
  if (level === 'critical') return 'error'
  if (level === 'high') return 'warning'
  if (level === 'medium') return 'warning'
  return 'default'
}

function truncateSha(sha: string): string {
  if (!sha) return '-'
  return sha.length > 10 ? sha.substring(0, 10) + '...' : sha
}

function truncateTime(s: string | null): string {
  if (!s) return '-'
  return s.replace('T', ' ').substring(0, 19)
}

// ─── Columns ───
const columns = computed<DataTableColumns<ChangeEvent>>(() => [
  {
    title: t('changeEvents.source'),
    key: 'source',
    width: 120,
    render: (row) =>
      h(NTag, { size: 'small', bordered: false, type: 'info' }, () => row.source),
  },
  {
    title: t('changeEvents.changeType'),
    key: 'change_type',
    width: 110,
    render: (row) => row.change_type,
  },
  {
    title: t('changeEvents.service'),
    key: 'service',
    minWidth: 140,
    ellipsis: { tooltip: true },
  },
  {
    title: t('changeEvents.environment'),
    key: 'environment',
    width: 120,
    render: (row) =>
      h(NTag, { size: 'small', bordered: false }, () => row.environment),
  },
  {
    title: t('changeEvents.author'),
    key: 'author',
    width: 120,
    ellipsis: { tooltip: true },
    render: (row) => row.author || '-',
  },
  {
    title: t('changeEvents.commitSha'),
    key: 'commit_sha',
    width: 130,
    render: (row) =>
      row.commit_sha
        ? h('span', {
            style: 'font-family: var(--sre-font-mono, monospace); font-size: 12px;',
            title: row.commit_sha,
          }, truncateSha(row.commit_sha))
        : '-',
  },
  {
    title: t('changeEvents.riskLevel'),
    key: 'risk_level',
    width: 100,
    render: (row) =>
      h(NTag, { size: 'small', bordered: false, type: riskLevelTag(row.risk_level) }, () => row.risk_level),
  },
  {
    title: t('changeEvents.timestamp'),
    key: 'timestamp',
    width: 170,
    render: (row) => truncateTime(row.timestamp),
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 80,
    fixed: 'right',
    render: (row) =>
      h(NButton, {
        size: 'tiny',
        quaternary: true,
        type: 'error',
        onClick: () => confirmDelete(row),
      }, { default: () => t('common.delete'), icon: () => h(NIcon, { size: 14, component: TrashOutline }) }),
  },
])

// ─── Init ───
onMounted(fetchEvents)
</script>

<template>
  <div class="change-events-page">
    <PageHeader :title="t('changeEvents.title')" :subtitle="t('changeEvents.subtitle')">
      <template #actions>
        <n-button
          v-if="authStore.canManage"
          type="primary"
          size="small"
          @click="openCreate"
        >
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
        style="width: 220px"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select
        v-model:value="filterService"
        size="small"
        :options="serviceOptions"
        :placeholder="t('changeEvents.service')"
        clearable
        style="width: 160px"
        @update:value="() => { page = 1; fetchEvents() }"
      />
      <n-select
        v-model:value="filterEnvironment"
        size="small"
        :options="environmentOptions"
        :placeholder="t('changeEvents.environment')"
        clearable
        style="width: 160px"
        @update:value="() => { page = 1; fetchEvents() }"
      />
      <n-select
        v-model:value="filterSource"
        size="small"
        :options="sourceOptions"
        :placeholder="t('changeEvents.source')"
        clearable
        style="width: 160px"
        @update:value="() => { page = 1; fetchEvents() }"
      />
      <span class="count tnum">{{ filteredEvents.length }} / {{ events.length }}</span>
    </div>

    <n-empty
      v-if="!loading && events.length === 0"
      :description="t('common.noData')"
      style="padding: 60px 0"
    >
      <template #extra>
        <n-button v-if="authStore.canManage" type="primary" size="small" @click="openCreate">{{ t('common.create') }}</n-button>
      </template>
    </n-empty>

    <template v-else>
      <n-data-table
        :columns="columns"
        :data="filteredEvents"
        :loading="loading"
        :row-key="(row: ChangeEvent) => row.id"
        size="small"
        :bordered="false"
        striped
        :scroll-x="1100"
      />

      <div class="page-pagination" v-if="total > 0">
        <n-pagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :item-count="total"
          :page-sizes="[20, 50, 100]"
          show-size-picker
          @update:page="handlePageChange"
          @update:page-size="(ps: number) => { pageSize = ps; page = 1; fetchEvents() }"
        />
      </div>
    </template>

    <!-- ===== Ingest Drawer ===== -->
    <n-drawer v-model:show="showDrawer" :width="520">
      <n-drawer-content :title="t('common.create')">
        <n-form label-placement="top">
          <div style="display: flex; gap: 16px;">
            <n-form-item :label="t('changeEvents.source')" style="flex: 1;" required>
              <n-input v-model:value="form.source" :placeholder="t('changeEvents.sourcePlaceholder')" />
            </n-form-item>
            <n-form-item :label="t('changeEvents.changeType')" style="flex: 1;">
              <n-select v-model:value="form.change_type" :options="changeTypeOptions" />
            </n-form-item>
          </div>

          <div style="display: flex; gap: 16px;">
            <n-form-item :label="t('changeEvents.service')" style="flex: 1;" required>
              <n-input v-model:value="form.service" />
            </n-form-item>
            <n-form-item :label="t('changeEvents.environment')" style="flex: 1;">
              <n-select v-model:value="form.environment" :options="environmentOptions" />
            </n-form-item>
          </div>

          <div style="display: flex; gap: 16px;">
            <n-form-item :label="t('changeEvents.commitSha')" style="flex: 1;">
              <n-input v-model:value="form.commit_sha" />
            </n-form-item>
            <n-form-item :label="t('changeEvents.author')" style="flex: 1;">
              <n-input v-model:value="form.author" />
            </n-form-item>
          </div>

          <n-form-item :label="t('changeEvents.riskLevel')">
            <n-select v-model:value="form.risk_level" :options="riskLevelOptions" />
          </n-form-item>

          <n-form-item :label="t('common.description')">
            <n-input v-model:value="form.description" type="textarea" :rows="3" />
          </n-form-item>

          <n-form-item :label="t('changeEvents.timestamp')">
            <n-date-picker
              v-model:value="timestampMs"
              type="datetime"
              style="width: 100%"
              :actions="['confirm']"
            />
          </n-form-item>

          <n-form-item :label="t('common.labels') + ' (JSON)'">
            <n-input
              v-model:value="metadataStr"
              type="textarea"
              :rows="3"
              placeholder='{"key":"value"}'
              @blur="syncMetadata"
            />
          </n-form-item>
        </n-form>

        <template #footer>
          <div style="display: flex; justify-content: flex-end; gap: 8px;">
            <n-button @click="showDrawer = false">{{ t('common.cancel') }}</n-button>
            <n-button type="primary" :loading="saving" @click="handleSave">
              {{ t('common.create') }}
            </n-button>
          </div>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<style scoped>
.change-events-page {
  padding: 16px;
  max-width: 1400px;
}
.toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
  flex-wrap: wrap;
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
