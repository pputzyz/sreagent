<template>
  <div class="es-patterns-page">
    <n-card :bordered="false">
      <template #header>
        <div class="card-header">
          <span>{{ t('esPatterns.title') }}</span>
          <n-button type="primary" @click="openCreate">
            {{ t('esPatterns.create') }}
          </n-button>
        </div>
      </template>

      <n-data-table
        :columns="columns"
        :data="patterns"
        :loading="loading"
        :row-key="(row: ESIndexPattern) => row.id"
        :bordered="false"
        striped
      />
    </n-card>

    <!-- Create/Edit Drawer -->
    <n-drawer v-model:show="drawerVisible" :width="640" placement="right">
      <n-drawer-content :title="isEdit ? t('esPatterns.edit') : t('esPatterns.create')">
        <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="140">
          <n-form-item :label="t('esPatterns.datasource')" path="datasource_id">
            <n-select
              v-model:value="form.datasource_id"
              :options="dsOptions"
              :placeholder="t('esPatterns.datasourcePlaceholder')"
              filterable
            />
          </n-form-item>
          <n-form-item :label="t('esPatterns.name')" path="name">
            <n-input v-model:value="form.name" :placeholder="t('esPatterns.namePlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('esPatterns.timeField')">
            <n-input v-model:value="form.time_field" placeholder="@timestamp" />
          </n-form-item>
          <n-form-item :label="t('esPatterns.crossCluster')">
            <n-switch v-model:value="form.cross_cluster_enabled" />
          </n-form-item>
          <n-form-item :label="t('esPatterns.hideSystemIndices')">
            <n-switch v-model:value="form.allow_hide_system_indices" />
          </n-form-item>
          <n-form-item :label="t('esPatterns.note')">
            <n-input v-model:value="form.note" type="textarea" :rows="3" :placeholder="t('esPatterns.notePlaceholder')" />
          </n-form-item>
        </n-form>

        <template #footer>
          <n-space>
            <n-button @click="drawerVisible = false">{{ t('common.cancel') }}</n-button>
            <n-button type="primary" :loading="saving" @click="handleSave">
              {{ t('common.save') }}
            </n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton, NCard, NDataTable, NDrawer, NDrawerContent, NForm, NFormItem,
  NInput, NSpace, NSwitch, NSelect, NPopconfirm, NTag, useMessage,
} from 'naive-ui'
import type { DataTableColumns, FormRules } from 'naive-ui'
import { esIndexPatternApi, datasourceApi } from '@/api'
import type { ESIndexPattern, CreateESIndexPatternRequest } from '@/api/es-index-pattern'
import type { DataSource } from '@/types'

const { t } = useI18n()
const message = useMessage()

// --- State ---
const patterns = ref<ESIndexPattern[]>([])
const loading = ref(false)
const drawerVisible = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const formRef = ref()
const currentEditId = ref<number | null>(null)

const datasources = ref<DataSource[]>([])
const dsOptions = ref<{ label: string; value: number }[]>([])

const form = ref<CreateESIndexPatternRequest>({
  datasource_id: 0,
  name: '',
  time_field: '@timestamp',
  allow_hide_system_indices: false,
  cross_cluster_enabled: false,
  note: '',
})

const rules: FormRules = {
  datasource_id: [{ required: true, type: 'number', min: 1, message: t('esPatterns.datasourceRequired'), trigger: 'change' }],
  name: [{ required: true, message: t('esPatterns.nameRequired'), trigger: 'blur' }],
}

// --- Columns ---
const columns: DataTableColumns<ESIndexPattern> = [
  {
    title: t('esPatterns.datasource'),
    key: 'datasource_id',
    width: 150,
    render: (row) => {
      const ds = datasources.value.find(d => d.id === row.datasource_id)
      return h(NTag, { type: 'info', size: 'small', bordered: false }, { default: () => ds?.name || `#${row.datasource_id}` })
    },
  },
  {
    title: t('esPatterns.name'),
    key: 'name',
    ellipsis: { tooltip: true },
  },
  {
    title: t('esPatterns.timeField'),
    key: 'time_field',
    width: 150,
  },
  {
    title: t('esPatterns.crossCluster'),
    key: 'cross_cluster_enabled',
    width: 100,
    render: (row) => row.cross_cluster_enabled ? h(NTag, { type: 'success', size: 'small', bordered: false }, { default: () => t('common.yes') }) : null,
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 160,
    render: (row) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'small', onClick: () => openEdit(row) }, { default: () => t('common.edit') }),
        h(NPopconfirm, { onPositiveClick: () => handleDelete(row.id) }, {
          trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('common.delete') }),
          default: () => t('common.confirmDelete'),
        }),
      ],
    }),
  },
]

// --- Methods ---
async function loadPatterns() {
  loading.value = true
  try {
    const res = await esIndexPatternApi.list()
    patterns.value = res.data.data || []
  } catch {
    message.error(t('esPatterns.loadError'))
  } finally {
    loading.value = false
  }
}

async function loadDatasources() {
  try {
    const res = await datasourceApi.list({ page: 1, page_size: 1000 })
    const list = res.data.data?.list || []
    datasources.value = list.filter((d: DataSource) => d.type === 'elasticsearch' && d.is_enabled)
    dsOptions.value = datasources.value.map(d => ({ label: d.name, value: d.id }))
  } catch {
    // silent
  }
}

function openCreate() {
  isEdit.value = false
  currentEditId.value = null
  form.value = { datasource_id: 0, name: '', time_field: '@timestamp', allow_hide_system_indices: false, cross_cluster_enabled: false, note: '' }
  drawerVisible.value = true
}

function openEdit(row: ESIndexPattern) {
  isEdit.value = true
  currentEditId.value = row.id
  form.value = {
    datasource_id: row.datasource_id,
    name: row.name,
    time_field: row.time_field,
    allow_hide_system_indices: row.allow_hide_system_indices,
    cross_cluster_enabled: row.cross_cluster_enabled,
    note: row.note,
  }
  drawerVisible.value = true
}

async function handleSave() {
  try {
    await formRef.value?.validate()
  } catch { return }

  saving.value = true
  try {
    if (isEdit.value && currentEditId.value) {
      await esIndexPatternApi.update(currentEditId.value, form.value)
      message.success(t('esPatterns.updateSuccess'))
    } else {
      await esIndexPatternApi.create(form.value)
      message.success(t('esPatterns.createSuccess'))
    }
    drawerVisible.value = false
    await loadPatterns()
  } catch {
    message.error(t('esPatterns.saveError'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await esIndexPatternApi.delete(id)
    message.success(t('esPatterns.deleteSuccess'))
    await loadPatterns()
  } catch {
    message.error(t('esPatterns.deleteError'))
  }
}

onMounted(() => {
  loadPatterns()
  loadDatasources()
})
</script>

<style scoped>
.es-patterns-page {
  padding: 16px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
