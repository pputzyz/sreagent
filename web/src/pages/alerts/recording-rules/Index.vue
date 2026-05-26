<script setup lang="ts">
import { ref, reactive, onMounted, watch, computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import { NButton, NSpace, NSwitch, NTag, NIcon, NForm, NFormItem, NInput, NSelect, NAlert } from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import { recordingRuleApi, type RecordingRule } from '@/api/recording'
import { datasourceApi } from '@/api'
import { useFilterMemory, usePermissions } from '@/composables'
import PageHeader from '@/components/common/PageHeader.vue'
import PromQLEditor from '@/components/query/PromQLEditor.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const { hasPerm } = usePermissions()

// State
const rules = ref<RecordingRule[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const selectedIds = ref<(string | number)[]>([])

// Filters with memory
const filterMemory = useFilterMemory('recording-rules')
const searchQuery = ref(filterMemory.restore('query', ''))
const filterDisabled = ref<number | null>(filterMemory.restore('disabled', null))
const filterGroupId = ref<number | null>(filterMemory.restore('group_id', null))
const filterDatasourceIds = ref<number[]>(filterMemory.restore('datasource_ids', []))
filterMemory.bindRefs({ query: searchQuery, disabled: filterDisabled, group_id: filterGroupId, datasource_ids: filterDatasourceIds })

// Datasource options for display and filter
const datasourceOptions = ref<{ label: string; value: number }[]>([])
const prometheusDatasources = computed(() => datasourceOptions.value)

// Edit dialog
const showEditModal = ref(false)
const editMode = ref<'create' | 'edit' | 'clone'>('create')
const editId = ref<number | null>(null)

// Form state
const form = reactive({
  name: '',
  prom_ql: '',
  datasource_ids: [] as number[],
  cron_pattern: '@every 60s',
  disabled: 0,
  append_tags: [] as string[],
  note: '',
})

// Import
const showImportModal = ref(false)
const importJson = ref('')
const importResults = ref<Record<string, string> | null>(null)

// Batch update
const showBatchUpdateModal = ref(false)
const batchUpdateField = ref<string>('datasource_ids')
const batchUpdateValue = ref<any>(null)
const batchUpdateLoading = ref(false)

const canWrite = computed(() => hasPerm('rules.write'))

// Name validation (Nightingale pattern: letters, digits, underscores, colons)
const namePattern = /^[0-9a-zA-Z_:]+$/

async function fetchRules() {
  loading.value = true
  try {
    const resp = await recordingRuleApi.list({
      page: page.value,
      page_size: pageSize.value,
      group_id: filterGroupId.value || undefined,
      query: searchQuery.value || undefined,
      disabled: filterDisabled.value !== null ? filterDisabled.value : undefined,
    })
    let list = resp.data.data?.list || []
    // Client-side datasource filter
    if (filterDatasourceIds.value.length > 0) {
      list = list.filter((r) => {
        if (!r.datasource_ids || r.datasource_ids.length === 0 || r.datasource_ids.includes(0)) return true
        return r.datasource_ids.some((id) => filterDatasourceIds.value.includes(id))
      })
    }
    rules.value = list
    total.value = resp.data.data?.total || 0
  } catch (e: any) {
    message.error(e.message || 'Failed to load recording rules')
  } finally {
    loading.value = false
  }
}

async function fetchDatasources() {
  try {
    const resp = await datasourceApi.list({ page: 1, page_size: 500 })
    const all = (resp.data.data?.list || [])
      .filter((ds: any) => ds.type === 'prometheus' || ds.type === 'victoriametrics')
    datasourceOptions.value = [
      { label: t('recording.allDatasources'), value: 0 },
      ...all.map((ds: any) => ({ label: ds.name, value: ds.id })),
    ]
  } catch {}
}

function handlePageChange(p: number) {
  page.value = p
  fetchRules()
}

function handlePageSizeChange(ps: number) {
  pageSize.value = ps
  page.value = 1
  fetchRules()
}

function openCreate() {
  editMode.value = 'create'
  editId.value = null
  resetForm()
  showEditModal.value = true
}

function openEdit(rule: RecordingRule) {
  editMode.value = 'edit'
  editId.value = rule.id
  fillForm(rule)
  showEditModal.value = true
}

function openClone(rule: RecordingRule) {
  editMode.value = 'clone'
  editId.value = null
  fillForm(rule)
  form.name = `${form.name}_copy`
  showEditModal.value = true
}

function resetForm() {
  form.name = ''
  form.prom_ql = ''
  form.datasource_ids = []
  form.cron_pattern = '@every 60s'
  form.disabled = 0
  form.append_tags = []
  form.note = ''
}

function fillForm(rule: RecordingRule) {
  form.name = rule.name
  form.prom_ql = rule.prom_ql
  form.datasource_ids = [...(rule.datasource_ids || [])]
  form.cron_pattern = rule.cron_pattern || '@every 60s'
  form.disabled = rule.disabled
  form.append_tags = [...(rule.append_tags || [])]
  form.note = rule.note || ''
}

// Validate PromQL against datasource (Nightingale pattern)
async function validatePromql(): Promise<boolean> {
  if (!form.prom_ql) return false
  const dsId = form.datasource_ids.length > 0 && form.datasource_ids[0] !== 0
    ? form.datasource_ids[0]
    : null
  if (!dsId) return true // Skip validation if $all or no datasource
  try {
    const resp = await datasourceApi.query(dsId, { expression: form.prom_ql })
    const data = resp.data?.data as any
    if (data?.error) {
      message.error(t('recording.promqlValidationError', { error: data.error }))
      return false
    }
    return true
  } catch (e: any) {
    message.error(t('recording.promqlValidationError', { error: e.message || 'Unknown error' }))
    return false
  }
}

async function handleSave() {
  // Name validation
  if (!form.name) {
    message.warning(t('recording.nameAndPromqlRequired'))
    return
  }
  if (!namePattern.test(form.name)) {
    message.error(t('recording.nameInvalid'))
    return
  }
  if (!form.prom_ql) {
    message.warning(t('recording.nameAndPromqlRequired'))
    return
  }

  // PromQL validation (Nightingale: validate before save)
  const valid = await validatePromql()
  if (!valid) return

  try {
    if (editMode.value === 'edit' && editId.value) {
      await recordingRuleApi.update(editId.value, { ...form })
      message.success(t('common.savedSuccess'))
    } else {
      await recordingRuleApi.create({
        group_id: filterGroupId.value || 0,
        ...form,
      })
      message.success(t('common.createSuccess'))
    }
    showEditModal.value = false
    fetchRules()
  } catch (e: any) {
    message.error(e.message || t('common.saveFailed'))
  }
}

async function handleDelete(rule: RecordingRule) {
  dialog.warning({
    title: t('common.confirm'),
    content: t('recording.confirmDelete', { name: rule.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await recordingRuleApi.delete(rule.id)
        message.success(t('common.deleteSuccess'))
        fetchRules()
      } catch (e: any) {
        message.error(e.message || t('common.deleteFailed'))
      }
    },
  })
}

async function handleBatchDelete() {
  if (!selectedIds.value.length) return
  const ids = selectedIds.value.map((id: string | number) => Number(id))
  dialog.warning({
    title: t('common.confirm'),
    content: t('recording.confirmBatchDelete', { count: ids.length }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await recordingRuleApi.batchDelete(filterGroupId.value || 0, ids)
        message.success(t('common.deleteSuccess'))
        selectedIds.value = []
        fetchRules()
      } catch (e: any) {
        message.error(e.message || t('common.deleteFailed'))
      }
    },
  })
}

async function handleToggleDisabled(rule: RecordingRule) {
  try {
    await recordingRuleApi.update(rule.id, {
      name: rule.name,
      prom_ql: rule.prom_ql,
      disabled: rule.disabled ? 0 : 1,
      datasource_ids: rule.datasource_ids,
      cron_pattern: rule.cron_pattern,
      append_tags: rule.append_tags,
      note: rule.note,
    })
    message.success(rule.disabled ? t('recording.enabled') : t('recording.disabled'))
    fetchRules()
  } catch (e: any) {
    message.error(e.message || t('common.updateFailed'))
  }
}

function handleExport() {
  const data = selectedIds.value.length
    ? rules.value.filter((r) => selectedIds.value.includes(r.id))
    : rules.value
  const exportData = data.map((r) => ({
    name: r.name,
    prom_ql: r.prom_ql,
    datasource_ids: r.datasource_ids,
    cron_pattern: r.cron_pattern,
    disabled: r.disabled,
    append_tags: r.append_tags,
    note: r.note,
  }))
  const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'recording-rules.json'
  a.click()
  URL.revokeObjectURL(url)
}

async function handleImport() {
  try {
    const parsed = JSON.parse(importJson.value)
    if (!Array.isArray(parsed)) {
      message.error(t('recording.importFormatError'))
      return
    }
    const resp = await recordingRuleApi.batchCreate(filterGroupId.value || 0, parsed)
    importResults.value = resp.data.data || {}
    fetchRules()
  } catch (e: any) {
    message.error(e.message || 'Import failed')
  }
}

// Batch update (Nightingale pattern)
async function handleBatchUpdate() {
  if (!selectedIds.value.length) return
  const ids = selectedIds.value.map((id) => Number(id))
  const fields: Record<string, any> = {}

  if (batchUpdateField.value === 'datasource_ids') {
    if (!batchUpdateValue.value || batchUpdateValue.value.length === 0) return
    fields.datasource_ids = batchUpdateValue.value
  } else if (batchUpdateField.value === 'cron_pattern') {
    if (!batchUpdateValue.value) return
    fields.cron_pattern = batchUpdateValue.value
  } else if (batchUpdateField.value === 'disabled') {
    fields.disabled = batchUpdateValue.value ? 0 : 1
  } else if (batchUpdateField.value === 'append_tags') {
    fields.append_tags = batchUpdateValue.value || []
  }

  batchUpdateLoading.value = true
  try {
    await recordingRuleApi.updateFields(ids, fields)
    message.success(t('recording.batchUpdateSuccess'))
    showBatchUpdateModal.value = false
    selectedIds.value = []
    batchUpdateValue.value = null
    fetchRules()
  } catch (e: any) {
    message.error(e.message || t('common.updateFailed'))
  } finally {
    batchUpdateLoading.value = false
  }
}

function openBatchUpdate() {
  batchUpdateField.value = 'datasource_ids'
  batchUpdateValue.value = null
  showBatchUpdateModal.value = true
}

function handleSelectionChange(keys: (string | number)[]) {
  selectedIds.value = keys
}

// Tag input helpers
const tagInput = ref('')
function addTag() {
  const tag = tagInput.value.trim()
  if (tag && !form.append_tags.includes(tag)) {
    form.append_tags.push(tag)
  }
  tagInput.value = ''
}
function removeTag(idx: number) {
  form.append_tags.splice(idx, 1)
}

// Datasource name display
function dsNames(ids: number[]): string {
  if (!ids || ids.length === 0) return t('recording.allDatasources')
  if (ids.includes(0)) return t('recording.allDatasources')
  return ids
    .map((id) => {
      const ds = datasourceOptions.value.find((d) => d.value === id)
      return ds ? ds.label : `ID:${id}`
    })
    .join(', ')
}

const columns = computed<DataTableColumns<RecordingRule>>(() => {
  const cols: DataTableColumns<RecordingRule> = [
    { type: 'selection' },
    {
      title: t('recording.name'),
      key: 'name',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: (row) =>
        h('a', {
          style: 'color: var(--sre-primary); cursor: pointer; text-decoration: none;',
          onClick: () => openEdit(row),
        }, row.name),
    },
    {
      title: t('recording.datasource'),
      key: 'datasource_ids',
      minWidth: 150,
      ellipsis: { tooltip: true },
      render: (row) => {
        const ids = row.datasource_ids || []
        if (ids.length === 0 || ids.includes(0)) {
          return h(NTag, { size: 'small', type: 'info', bordered: false }, () => t('recording.allDatasources'))
        }
        return h(NSpace, { size: 4, wrap: true }, () =>
          ids.map((id) => {
            const ds = datasourceOptions.value.find((d) => d.value === id)
            return h(NTag, { size: 'small', bordered: false }, () => ds ? ds.label : `ID:${id}`)
          })
        )
      },
    },
    {
      title: t('recording.cronPattern'),
      key: 'cron_pattern',
      width: 140,
    },
    {
      title: t('recording.appendTags'),
      key: 'append_tags',
      minWidth: 150,
      ellipsis: { tooltip: true },
      render: (row) =>
        row.append_tags?.length
          ? h(NSpace, { size: 4, wrap: true }, () =>
              row.append_tags.map((tag) =>
                h(NTag, { size: 'small', type: 'warning', bordered: false }, () => tag)
              )
            )
          : '-',
    },
    {
      title: t('recording.updatedAt'),
      key: 'updated_at',
      width: 170,
      render: (row) => new Date(row.updated_at).toLocaleString(),
    },
    {
      title: t('recording.enabled'),
      key: 'disabled',
      width: 80,
      render: (row) =>
        h(NSwitch, {
          value: row.disabled === 0,
          onUpdateValue: () => handleToggleDisabled(row),
          disabled: !canWrite.value,
          size: 'small',
        }),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 180,
      render: (row) =>
        h(NSpace, { size: 'small' }, () => [
          h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => openEdit(row) }, () => t('common.edit')),
          h(NButton, { size: 'tiny', quaternary: true, onClick: () => openClone(row) }, () => t('common.duplicate')),
          h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleDelete(row) }, () => t('common.delete')),
        ]),
    },
  ]
  return cols
})

// Watch filters
watch([searchQuery, filterDisabled, filterGroupId, filterDatasourceIds], () => {
  page.value = 1
  fetchRules()
})

onMounted(() => {
  fetchRules()
  fetchDatasources()
})
</script>

<template>
  <div class="recording-rules-page">
    <PageHeader :title="t('menu.recordingRules')" />

    <div class="page-toolbar">
      <div class="toolbar-left">
        <NInput
          v-model:value="searchQuery"
          :placeholder="t('recording.searchPlaceholder')"
          clearable
          size="small"
          style="width: 240px;"
        />
        <NSelect
          v-model:value="filterDatasourceIds"
          :placeholder="t('recording.selectDatasource')"
          :options="prometheusDatasources"
          multiple
          filterable
          clearable
          size="small"
          style="width: 200px;"
        />
        <NSelect
          v-model:value="filterDisabled"
          :placeholder="t('recording.statusFilter')"
          :options="[
            { label: t('recording.enabled'), value: 0 },
            { label: t('recording.disabled'), value: 1 },
          ]"
          clearable
          size="small"
          style="width: 140px;"
        />
      </div>
      <div class="toolbar-right" v-if="canWrite">
        <NButton size="small" type="primary" @click="openCreate">
          <template #icon><NIcon><AddOutline /></NIcon></template>
          {{ t('recording.create') }}
        </NButton>
        <NButton size="small" @click="showImportModal = true">
          {{ t('recording.import') }}
        </NButton>
        <NButton size="small" @click="handleExport">
          {{ t('recording.export') }}
        </NButton>
        <NButton
          v-if="selectedIds.length"
          size="small"
          type="warning"
          @click="openBatchUpdate"
        >
          {{ t('recording.batchUpdate') }} ({{ selectedIds.length }})
        </NButton>
        <NButton
          v-if="selectedIds.length"
          size="small"
          type="error"
          @click="handleBatchDelete"
        >
          {{ t('common.delete') }} ({{ selectedIds.length }})
        </NButton>
      </div>
    </div>

    <NDataTable
      :columns="columns"
      :data="rules"
      :loading="loading"
      :row-key="(row: RecordingRule) => row.id"
      :checked-row-keys="selectedIds"
      @update:checked-row-keys="handleSelectionChange"
      size="small"
      :bordered="false"
      striped
    />

    <div class="page-pagination" v-if="total > 0">
      <NPagination
        v-model:page="page"
        v-model:page-size="pageSize"
        :item-count="total"
        :page-sizes="[20, 50, 100]"
        show-size-picker
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </div>

    <!-- Create/Edit Modal -->
    <NModal
      v-model:show="showEditModal"
      preset="card"
      :title="editMode === 'edit' ? t('recording.editRule') : editMode === 'clone' ? t('recording.cloneRule') : t('recording.createRule')"
      style="width: 720px; max-height: 80vh; overflow-y: auto;"
    >
      <NForm label-placement="left" label-width="120px">
        <NFormItem :label="t('recording.name')" required :validation-status="form.name && !namePattern.test(form.name) ? 'error' : undefined" :feedback="form.name && !namePattern.test(form.name) ? t('recording.nameInvalid') : undefined">
          <NInput v-model:value="form.name" :placeholder="t('recording.namePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('recording.note')">
          <NInput v-model:value="form.note" type="textarea" :placeholder="t('recording.notePlaceholder')" :rows="2" />
        </NFormItem>
        <NFormItem :label="t('recording.datasource')">
          <NSelect
            v-model:value="form.datasource_ids"
            :options="datasourceOptions"
            multiple
            filterable
            :placeholder="t('recording.allDatasources')"
          />
        </NFormItem>
        <NFormItem :label="t('recording.promql')" required>
          <PromQLEditor
            :model-value="form.prom_ql"
            :datasource-id="form.datasource_ids.length && form.datasource_ids[0] !== 0 ? form.datasource_ids[0] : null"
            :placeholder="t('recording.promqlPlaceholder')"
            style="width: 100%; min-height: 100px; border: 1px solid var(--n-border-color); border-radius: 3px;"
            @update:model-value="form.prom_ql = $event"
          />
        </NFormItem>
        <NFormItem :label="t('recording.cronPattern')">
          <NInput v-model:value="form.cron_pattern" placeholder="@every 60s" />
        </NFormItem>
        <NFormItem :label="t('recording.appendTags')">
          <div style="width: 100%;">
            <div style="display: flex; flex-wrap: wrap; gap: 4px; margin-bottom: 8px;">
              <NTag
                v-for="(tag, idx) in form.append_tags"
                :key="idx"
                size="small"
                closable
                @close="removeTag(idx)"
              >
                {{ tag }}
              </NTag>
            </div>
            <NInput
              v-model:value="tagInput"
              size="small"
              placeholder="key=value"
              @keyup.enter="addTag"
              style="width: 200px;"
            />
          </div>
        </NFormItem>
        <NFormItem :label="t('recording.enabled')">
          <NSwitch :value="form.disabled === 0" @update-value="(v: boolean) => form.disabled = v ? 0 : 1" />
        </NFormItem>
      </NForm>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 8px;">
          <NButton @click="showEditModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" @click="handleSave">{{ t('common.save') }}</NButton>
        </div>
      </template>
    </NModal>

    <!-- Import Modal -->
    <NModal
      v-model:show="showImportModal"
      preset="card"
      :title="t('recording.import')"
      style="width: 600px;"
    >
      <NInput
        v-model:value="importJson"
        type="textarea"
        :autosize="{ minRows: 8, maxRows: 16 }"
        :placeholder="t('recording.importPlaceholder')"
      />
      <div v-if="importResults" style="margin-top: 12px;">
        <NAlert type="info" :title="t('recording.importResults')">
          <div v-for="(err, name) in importResults" :key="name" style="font-size: 13px;">
            <span :style="{ color: err ? 'var(--sre-error)' : 'var(--sre-success)' }">
              {{ name }}: {{ err || t('common.success') }}
            </span>
          </div>
        </NAlert>
      </div>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 8px;">
          <NButton @click="showImportModal = false; importResults = null">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" @click="handleImport">{{ t('recording.import') }}</NButton>
        </div>
      </template>
    </NModal>

    <!-- Batch Update Modal (Nightingale pattern) -->
    <NModal
      v-model:show="showBatchUpdateModal"
      preset="card"
      :title="t('recording.batchUpdateTitle')"
      style="width: 520px;"
    >
      <NAlert type="info" style="margin-bottom: 16px;">{{ t('recording.batchUpdateHint') }}</NAlert>
      <NForm label-placement="left" label-width="100px">
        <NFormItem :label="t('recording.batchUpdateField')">
          <NSelect
            v-model:value="batchUpdateField"
            :options="[
              { label: t('recording.datasource'), value: 'datasource_ids' },
              { label: t('recording.cronPattern'), value: 'cron_pattern' },
              { label: t('recording.enabled'), value: 'disabled' },
              { label: t('recording.appendTags'), value: 'append_tags' },
            ]"
          />
        </NFormItem>
        <NFormItem :label="t('recording.batchUpdateValue')">
          <NSelect
            v-if="batchUpdateField === 'datasource_ids'"
            v-model:value="batchUpdateValue"
            :options="datasourceOptions"
            multiple
            filterable
          />
          <NInput
            v-else-if="batchUpdateField === 'cron_pattern'"
            v-model:value="batchUpdateValue"
            placeholder="@every 60s"
          />
          <NSwitch
            v-else-if="batchUpdateField === 'disabled'"
            :value="batchUpdateValue === null ? true : batchUpdateValue"
            @update-value="(v: boolean) => batchUpdateValue = v"
          />
          <NInput
            v-else-if="batchUpdateField === 'append_tags'"
            v-model:value="batchUpdateValue"
            placeholder="tag1,tag2,tag3"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 8px;">
          <NButton @click="showBatchUpdateModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="batchUpdateLoading" @click="handleBatchUpdate">
            {{ t('recording.batchUpdate') }}
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.recording-rules-page {
  padding: 16px;
}
.page-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  gap: 12px;
}
.toolbar-left {
  display: flex;
  gap: 8px;
  align-items: center;
}
.toolbar-right {
  display: flex;
  gap: 8px;
  align-items: center;
}
.page-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
