<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'
import {
  NButton, NIcon, NInput, NSelect, NTag, NSwitch, NDrawer, NDrawerContent,
  NForm, NFormItem, NInputNumber, NCollapse, NCollapseItem, NSpace, NDataTable,
  NPagination, NEmpty, NTooltip,
} from 'naive-ui'
import {
  AddOutline, SearchOutline, StarOutline, Star,
} from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import { llmConfigApi, type LLMConfig, type CreateLLMConfigRequest, type LLMExtraConfig } from '@/api/llm-config'
import { getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

// --- State ---
const configs = ref<LLMConfig[]>([])
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
const testingId = ref<number | null>(null)

// Form
const form = ref<CreateLLMConfigRequest & { id?: number }>({
  name: '',
  provider: 'openai',
  api_url: '',
  api_key: '',
  model: '',
  extra_config: '{}',
  enabled: true,
  is_default: false,
  description: '',
})

// Extra config parsed
const extraForm = ref<LLMExtraConfig>({
  timeout_seconds: 30,
  skip_tls_verify: false,
  proxy: '',
  temperature: 0.7,
  max_tokens: 4096,
})

// --- Options ---
const providerOptions = computed(() => [
  { label: 'OpenAI', value: 'openai' },
  { label: 'Azure OpenAI', value: 'azure' },
  { label: 'Ollama', value: 'ollama' },
  { label: 'Anthropic', value: 'anthropic' },
  { label: t('llmConfigs.providers.custom'), value: 'custom' },
])

function getProviderLabel(provider: string): string {
  const opt = providerOptions.value.find(o => o.value === provider)
  return opt?.label || provider
}

function getProviderColor(provider: string): string {
  const map: Record<string, string> = {
    openai: 'success',
    azure: 'info',
    ollama: 'warning',
    anthropic: 'error',
    custom: 'default',
  }
  return map[provider] || 'default'
}

// --- Filtered ---
const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return configs.value
  return configs.value.filter(c =>
    c.name.toLowerCase().includes(q) ||
    c.model.toLowerCase().includes(q) ||
    c.provider.toLowerCase().includes(q) ||
    (c.description || '').toLowerCase().includes(q)
  )
})

// --- API ---
async function fetchConfigs() {
  loading.value = true
  try {
    const resp = await llmConfigApi.list({ page: page.value, page_size: pageSize.value })
    configs.value = resp.data.data?.list || []
    total.value = resp.data.data?.total || 0
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

function handlePageChange(p: number) {
  page.value = p
  fetchConfigs()
}

function handlePageSizeChange(ps: number) {
  pageSize.value = ps
  page.value = 1
  fetchConfigs()
}

// --- CRUD ---
function openCreate() {
  drawerMode.value = 'create'
  editingId.value = null
  resetForm()
  showDrawer.value = true
}

function openEdit(config: LLMConfig) {
  drawerMode.value = 'edit'
  editingId.value = config.id
  fillForm(config)
  showDrawer.value = true
}

function resetForm() {
  form.value = {
    name: '',
    provider: 'openai',
    api_url: '',
    api_key: '',
    model: '',
    extra_config: '{}',
    enabled: true,
    is_default: false,
    description: '',
  }
  extraForm.value = {
    timeout_seconds: 30,
    skip_tls_verify: false,
    proxy: '',
    temperature: 0.7,
    max_tokens: 4096,
  }
}

function fillForm(config: LLMConfig) {
  form.value = {
    name: config.name,
    provider: config.provider,
    api_url: config.api_url,
    api_key: config.api_key,
    model: config.model,
    extra_config: config.extra_config,
    enabled: config.enabled,
    is_default: config.is_default,
    description: config.description,
  }
  try {
    const parsed = JSON.parse(config.extra_config || '{}')
    extraForm.value = {
      timeout_seconds: parsed.timeout_seconds ?? 30,
      skip_tls_verify: parsed.skip_tls_verify ?? false,
      proxy: parsed.proxy ?? '',
      temperature: parsed.temperature ?? 0.7,
      max_tokens: parsed.max_tokens ?? 4096,
    }
  } catch {
    extraForm.value = {
      timeout_seconds: 30,
      skip_tls_verify: false,
      proxy: '',
      temperature: 0.7,
      max_tokens: 4096,
    }
  }
}

async function handleSave() {
  if (!form.value.name?.trim()) {
    message.warning(t('llmConfigs.nameRequired'))
    return
  }
  if (!form.value.api_url?.trim()) {
    message.warning(t('llmConfigs.apiUrlRequired'))
    return
  }
  if (!form.value.model?.trim()) {
    message.warning(t('llmConfigs.modelRequired'))
    return
  }
  saving.value = true
  try {
    const payload: CreateLLMConfigRequest = {
      name: form.value.name,
      provider: form.value.provider,
      api_url: form.value.api_url,
      api_key: form.value.api_key,
      model: form.value.model,
      extra_config: JSON.stringify(extraForm.value),
      enabled: form.value.enabled,
      is_default: form.value.is_default,
      description: form.value.description,
    }
    if (drawerMode.value === 'edit' && editingId.value) {
      await llmConfigApi.update(editingId.value, payload)
      message.success(t('llmConfigs.updateSuccess'))
    } else {
      await llmConfigApi.create(payload)
      message.success(t('llmConfigs.createSuccess'))
    }
    showDrawer.value = false
    fetchConfigs()
  } catch (e: unknown) {
    message.error(getErrorMessage(e))
  } finally {
    saving.value = false
  }
}

function confirmDelete(config: LLMConfig) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('llmConfigs.confirmDelete', { name: config.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await llmConfigApi.delete(config.id)
        message.success(t('llmConfigs.deleteSuccess'))
        fetchConfigs()
      } catch (e: unknown) {
        message.error(getErrorMessage(e))
      }
    },
  })
}

async function handleTest(config: LLMConfig) {
  testingId.value = config.id
  try {
    const resp = await llmConfigApi.testConnection(config.id)
    const result = resp.data.data
    if (result?.success) {
      const latency = result.latency_ms ? ` (${result.latency_ms}ms)` : ''
      message.success(`${t('llmConfigs.testSuccess')}${latency}`)
    } else {
      message.warning(`${t('llmConfigs.testFailed')}: ${result?.message || ''}`)
    }
  } catch (e: unknown) {
    message.error(`${t('llmConfigs.testFailed')}: ${getErrorMessage(e)}`)
  } finally {
    testingId.value = null
  }
}

// --- Columns ---
const columns = computed<DataTableColumns<LLMConfig>>(() => [
  {
    title: t('llmConfigs.name'),
    key: 'name',
    minWidth: 160,
    ellipsis: { tooltip: true },
    render: (row) =>
      h('div', { style: 'display: flex; align-items: center; gap: 6px;' }, [
        row.is_default
          ? h(NTooltip, {}, {
              trigger: () => h(NIcon, { size: 14, color: '#f0a020', component: Star }),
              default: () => t('llmConfigs.isDefault'),
            })
          : null,
        h('a', {
          style: 'color: var(--sre-primary); cursor: pointer; text-decoration: none;',
          onClick: () => openEdit(row),
        }, row.name),
      ]),
  },
  {
    title: t('llmConfigs.provider'),
    key: 'provider',
    width: 130,
    render: (row) =>
      h(NTag, {
        size: 'small',
        type: getProviderColor(row.provider) as any,
        bordered: false,
      }, () => getProviderLabel(row.provider)),
  },
  {
    title: t('llmConfigs.model'),
    key: 'model',
    minWidth: 150,
    ellipsis: { tooltip: true },
    render: (row) => row.model || '-',
  },
  {
    title: t('llmConfigs.enabled'),
    key: 'enabled',
    width: 90,
    render: (row) =>
      h(NSwitch, {
        value: row.enabled,
        size: 'small',
        disabled: true,
      }),
  },
  {
    title: t('llmConfigs.isDefault'),
    key: 'is_default',
    width: 90,
    render: (row) =>
      row.is_default
        ? h(NIcon, { size: 16, color: '#f0a020', component: Star })
        : h(NIcon, { size: 16, color: '#ccc', component: StarOutline }),
  },
  {
    title: t('common.description'),
    key: 'description',
    minWidth: 150,
    ellipsis: { tooltip: true },
    render: (row) => row.description || '-',
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
          loading: testingId.value === row.id,
          onClick: () => handleTest(row),
        }, () => t('llmConfigs.testConnection')),
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
onMounted(fetchConfigs)
</script>

<template>
  <div class="llm-configs-page">
    <PageHeader :title="t('llmConfigs.title')" :subtitle="t('llmConfigs.subtitle')">
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
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <span class="count tnum">{{ filtered.length }} / {{ configs.length }}</span>
    </div>

    <n-empty v-if="!loading && configs.length === 0" :description="t('common.noData')" style="padding: 60px 0">
      <template #extra>
        <n-button type="primary" size="small" @click="openCreate">{{ t('common.create') }}</n-button>
      </template>
    </n-empty>

    <template v-else>
      <n-data-table
        :columns="columns"
        :data="filtered"
        :loading="loading"
        :row-key="(row: LLMConfig) => row.id"
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
    <n-drawer v-model:show="showDrawer" :width="520">
      <n-drawer-content :title="drawerMode === 'edit' ? t('common.edit') : t('common.create')">
        <n-form label-placement="top">
          <n-form-item :label="t('llmConfigs.name')" required>
            <n-input
              v-model:value="form.name"
              :placeholder="t('llmConfigs.namePlaceholder')"
            />
          </n-form-item>

          <n-form-item :label="t('llmConfigs.provider')">
            <n-select
              v-model:value="form.provider"
              :options="providerOptions"
            />
          </n-form-item>

          <n-form-item :label="t('llmConfigs.apiKey')">
            <n-input
              v-model:value="form.api_key"
              type="password"
              show-password-on="click"
              :placeholder="t('llmConfigs.apiKeyPlaceholder')"
            />
          </n-form-item>

          <n-form-item :label="t('llmConfigs.apiUrl')" required>
            <n-input
              v-model:value="form.api_url"
              :placeholder="t('llmConfigs.apiUrlPlaceholder')"
            />
          </n-form-item>

          <n-form-item :label="t('llmConfigs.model')" required>
            <n-input
              v-model:value="form.model"
              :placeholder="t('llmConfigs.modelPlaceholder')"
            />
          </n-form-item>

          <div style="display: flex; gap: 16px;">
            <n-form-item :label="t('llmConfigs.enabled')" style="flex: 1;">
              <n-switch v-model:value="form.enabled" />
            </n-form-item>
            <n-form-item :label="t('llmConfigs.isDefault')" style="flex: 1;">
              <n-switch v-model:value="form.is_default" />
            </n-form-item>
          </div>

          <n-form-item :label="t('common.description')">
            <n-input
              v-model:value="form.description"
              type="textarea"
              :rows="2"
              :placeholder="t('llmConfigs.descriptionPlaceholder')"
            />
          </n-form-item>

          <!-- Advanced Settings -->
          <n-collapse>
            <n-collapse-item :title="t('llmConfigs.advanced')" name="advanced">
              <n-form-item :label="t('llmConfigs.timeout')">
                <n-input-number
                  v-model:value="extraForm.timeout_seconds"
                  :min="1"
                  :max="300"
                  style="width: 100%"
                >
                  <template #suffix>{{ t('common.seconds') }}</template>
                </n-input-number>
              </n-form-item>

              <n-form-item :label="t('llmConfigs.skipTls')">
                <n-switch v-model:value="extraForm.skip_tls_verify" />
              </n-form-item>

              <n-form-item :label="t('llmConfigs.proxy')">
                <n-input
                  v-model:value="extraForm.proxy"
                  placeholder="http://proxy:8080"
                />
              </n-form-item>

              <n-form-item :label="t('llmConfigs.temperature')">
                <n-input-number
                  v-model:value="extraForm.temperature"
                  :min="0"
                  :max="2"
                  :step="0.1"
                  style="width: 100%"
                />
              </n-form-item>

              <n-form-item :label="t('llmConfigs.maxTokens')">
                <n-input-number
                  v-model:value="extraForm.max_tokens"
                  :min="1"
                  :max="128000"
                  style="width: 100%"
                />
              </n-form-item>
            </n-collapse-item>
          </n-collapse>
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
.llm-configs-page {
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
