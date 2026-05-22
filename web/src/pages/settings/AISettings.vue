<script setup lang="ts">
import { ref, reactive, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton, NIcon, NSwitch, NAlert, NCard, NDivider, NSpin,
  NSpace, NTag, NSelect, NInput, NModal, NForm, NFormItem,
  NPopconfirm, NStatistic, NTabs, NTabPane, NInputNumber, NDataTable, useMessage,
} from 'naive-ui'
import { PulseOutline, SaveOutline, SparklesOutline, AddOutline, TrashOutline, CreateOutline, StarOutline, SettingsOutline } from '@vicons/ionicons5'
import { aiApi, aiModuleApi, alertRuleApi } from '@/api'
import type { AIModuleConfig, AIProvider, AIProvidersConfig, AIGlobalConfig } from '@/types/ai-module'
import { getErrorMessage } from '@/utils/format'
import { useConfigForm } from '@/composables'

const { t } = useI18n()
const message = useMessage()

// ─── Active tab ───
const activeTab = ref('providers')

// ─── Providers config (manual — complex CRUD, not a simple form) ───
const providersLoading = ref(false)
const providersSaving = ref(false)
const providersConfig = ref<AIProvidersConfig | null>(null)

// ─── Modules config (via useConfigForm — switches auto-save) ───
const modulesForm = useConfigForm({
  load: () => aiModuleApi.getModules().then(r => r.data.data),
  save: (f) => aiModuleApi.updateModules(f as AIModuleConfig),
  autoSaveKeys: ['platform', 'chat', 'rule_gen', 'analysis', 'agent'],
})

// ─── Global config (via useConfigForm — switch auto-save) ───
const globalForm = useConfigForm({
  load: () => aiApi.getGlobal().then(r => r.data.data ?? {
    retry_max: 3,
    context_max_chars: 8000,
    default_temperature: 0.7,
    default_max_tokens: 2000,
    monthly_token_budget: 0,
    data_masking_enabled: true,
  }),
  save: (f) => aiApi.saveGlobal(f as AIGlobalConfig),
  autoSaveKeys: ['data_masking_enabled'],
})

// ─── Shared testing state ───
const testing = ref(false)
const testingProvider = ref<string | null>(null)

// ─── Label validation preview ───
const previewLoading = ref(false)
const showPreviewModal = ref(false)
const previewResult = ref<{ total: number; passing: number; failing: number; samples: Array<{ rule_id: number; rule_name: string; pass: boolean; issues?: string[] }> } | null>(null)

// ─── Provider modal ───
const showModal = ref(false)
const editingIndex = ref<number>(-1)
const providerForm = reactive<AIProvider>({
  key: '',
  provider: 'openai',
  api_key: '',
  base_url: '',
  model: '',
  enabled: true,
})

const providerOptions = [
  { label: 'OpenAI', value: 'openai' },
  { label: 'Azure OpenAI', value: 'azure' },
  { label: 'Ollama (Local)', value: 'ollama' },
  { label: 'Anthropic Claude', value: 'anthropic' },
  { label: 'Custom / Compatible', value: 'custom' },
]

const moduleLabels = computed<Record<keyof AIModuleConfig, { name: string; description: string }>>(() => ({
  platform: {
    name: t('aiSettings.modulePlatform'),
    description: t('aiSettings.modulePlatformDesc'),
  },
  chat: {
    name: t('aiSettings.moduleChat'),
    description: t('aiSettings.moduleChatDesc'),
  },
  rule_gen: {
    name: t('aiSettings.moduleRuleGen'),
    description: t('aiSettings.moduleRuleGenDesc'),
  },
  analysis: {
    name: t('aiSettings.moduleAnalysis'),
    description: t('aiSettings.moduleAnalysisDesc'),
  },
  agent: {
    name: t('aiSettings.moduleAgent'),
    description: t('aiSettings.moduleAgentDesc'),
  },
}))

const moduleKeys: (keyof AIModuleConfig)[] = ['platform', 'chat', 'rule_gen', 'analysis', 'agent']

// ─── Computed ───
const providerSelectOptions = computed(() => {
  if (!providersConfig.value?.providers) return []
  return providersConfig.value.providers.map(p => ({
    label: `${p.key} (${p.model || p.provider})`,
    value: p.key,
  }))
})

const hasProviders = computed(() => {
  return (providersConfig.value?.providers?.length ?? 0) > 0
})

const defaultProviderHealthy = computed(() => {
  const def = providersConfig.value?.providers.find((p: AIProvider) => p.key === providersConfig.value?.default_provider)
  return def?.enabled === true
})

const providerColumns = computed(() => [
  {
    title: 'Key',
    key: 'key',
    width: 140,
    render: (row: AIProvider) => h('span', { style: 'font-family: var(--sre-font-mono, monospace); font-size: 13px; font-weight: 600;' }, [
      row.key,
      providersConfig.value?.default_provider === row.key
        ? h(NTag, { type: 'warning', size: 'tiny', bordered: false, style: 'margin-left: 6px;' }, () => t('aiSettings.default'))
        : null,
    ]),
  },
  {
    title: t('common.type'),
    key: 'provider',
    width: 140,
    render: (row: AIProvider) => providerTypeLabel(row.provider),
  },
  {
    title: t('aiSettings.model'),
    key: 'model',
    width: 160,
    render: (row: AIProvider) => h('span', { style: 'font-family: var(--sre-font-mono, monospace); font-size: 12px;' }, row.model || '-'),
  },
  {
    title: '',
    key: 'actions',
    width: 140,
    render: (row: AIProvider) => {
      const idx = providersConfig.value!.providers.indexOf(row)
      return h('div', { style: 'display: flex; gap: 2px;' }, [
        h(NButton, { text: true, size: 'small', loading: testingProvider.value === row.key, onClick: () => handleTestProvider(row.key) }, { icon: () => h(NIcon, { component: PulseOutline }) }),
        h(NButton, { text: true, size: 'small', disabled: providersConfig.value!.default_provider === row.key, onClick: () => setDefaultProvider(row.key) }, { icon: () => h(NIcon, { component: StarOutline }) }),
        h(NButton, { text: true, size: 'small', onClick: () => openEditProvider(idx) }, { icon: () => h(NIcon, { component: CreateOutline }) }),
        h(NPopconfirm, { onPositiveClick: () => deleteProvider(idx) }, {
          trigger: () => h(NButton, { text: true, size: 'small', type: 'error' }, { icon: () => h(NIcon, { component: TrashOutline }) }),
          default: () => t('aiSettings.deleteProviderConfirm', { key: row.key }),
        }),
      ])
    },
  },
])

// ─── Fetch providers ───
async function fetchProviders() {
  providersLoading.value = true
  try {
    const res = await aiApi.getProviders()
    providersConfig.value = res.data.data ?? { default_provider: '', providers: [] }
  } catch {
    providersConfig.value = { default_provider: '', providers: [] }
  } finally {
    providersLoading.value = false
  }
}

// ─── Provider CRUD ───
function openAddProvider() {
  editingIndex.value = -1
  Object.assign(providerForm, {
    key: '',
    provider: 'openai',
    api_key: '',
    base_url: '',
    model: '',
    enabled: true,
  })
  showModal.value = true
}

function openEditProvider(index: number) {
  if (!providersConfig.value) return
  const p = providersConfig.value.providers[index]
  editingIndex.value = index
  Object.assign(providerForm, {
    key: p.key,
    provider: p.provider,
    api_key: p.api_key,
    base_url: p.base_url,
    model: p.model,
    enabled: p.enabled,
  })
  showModal.value = true
}

function handleProviderSave() {
  if (!providersConfig.value) return

  if (!providerForm.key.trim()) {
    message.error(t('aiSettings.providerKeyRequired'))
    return
  }

  const entry: AIProvider = {
    key: providerForm.key.trim(),
    provider: providerForm.provider,
    api_key: providerForm.api_key,
    base_url: providerForm.base_url,
    model: providerForm.model,
    enabled: providerForm.enabled,
  }

  if (editingIndex.value >= 0) {
    providersConfig.value.providers[editingIndex.value] = entry
  } else {
    if (providersConfig.value.providers.some(p => p.key === entry.key)) {
      message.error(t('aiSettings.providerKeyDuplicate'))
      return
    }
    providersConfig.value.providers.push(entry)
    if (providersConfig.value.providers.length === 1) {
      providersConfig.value.default_provider = entry.key
    }
  }

  showModal.value = false
}

function deleteProvider(index: number) {
  if (!providersConfig.value) return
  const key = providersConfig.value.providers[index].key
  providersConfig.value.providers.splice(index, 1)
  if (providersConfig.value.default_provider === key) {
    providersConfig.value.default_provider = providersConfig.value.providers[0]?.key ?? ''
  }
}

function setDefaultProvider(key: string) {
  if (!providersConfig.value) return
  providersConfig.value.default_provider = key
}

async function handleSaveProviders() {
  if (!providersConfig.value) return
  providersSaving.value = true
  try {
    await aiApi.saveProviders(providersConfig.value)
    message.success(t('aiSettings.providerSaved'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    providersSaving.value = false
  }
}

// ─── Module config helpers ───
function toggleModule(key: keyof AIModuleConfig, val: boolean) {
  if (!modulesForm.form[key]) return
  modulesForm.form[key].enabled = val
}

function setModuleProvider(key: keyof AIModuleConfig, providerKey: string) {
  if (!modulesForm.form[key]) return
  modulesForm.form[key].provider_key = providerKey
}

// ─── Label validation preview ───
async function handlePreviewImpact() {
  previewLoading.value = true
  try {
    const res = await alertRuleApi.labelValidationPreview(20)
    previewResult.value = res.data.data ?? null
    showPreviewModal.value = true
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    previewLoading.value = false
  }
}

// ─── Test connection ───
async function handleTestDefault() {
  testing.value = true
  try {
    const res = await aiApi.testConnection()
    const ok = !!res.data.data?.success
    ok
      ? message.success(res.data.data?.message || t('aiSettings.testSuccess'))
      : message.error(res.data.data?.message || t('aiSettings.testFailed'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    testing.value = false
  }
}

async function handleTestProvider(key: string) {
  testingProvider.value = key
  try {
    const res = await aiApi.testProvider(key)
    message.success(res.data.data?.message || t('aiSettings.testSuccess'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    testingProvider.value = null
  }
}

// ─── Provider label ───
function providerTypeLabel(p: string) {
  const map: Record<string, string> = {
    openai: 'OpenAI',
    azure: 'Azure OpenAI',
    ollama: 'Ollama (Local)',
    anthropic: 'Anthropic Claude',
    custom: 'Custom / Compatible',
  }
  return map[p] || p
}

onMounted(() => {
  fetchProviders()
  modulesForm.load()
  globalForm.load()
})
</script>

<template>
  <NSpin :show="providersLoading && modulesForm.loading.value">
    <div class="sre-config-page ai-settings-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">
            <n-icon :component="SparklesOutline" :size="20" style="margin-right: 8px; vertical-align: -3px;" />
            {{ t('aiSettings.title') }}
          </h2>
          <p class="sre-config-header-sub">{{ t('aiSettings.subtitle') }}</p>
        </div>
        <div class="sre-config-header-actions">
          <n-button v-if="activeTab === 'providers'" type="primary" size="small" :loading="providersSaving" @click="handleSaveProviders">
            <template #icon><n-icon :component="SaveOutline" /></template>
            {{ t('common.save') }}
          </n-button>
          <n-button v-else-if="activeTab === 'modules'" type="primary" size="small" :loading="modulesForm.saving.value" @click="modulesForm.save">
            <template #icon><n-icon :component="SaveOutline" /></template>
            {{ t('common.save') }}
          </n-button>
          <n-button v-else-if="activeTab === 'global'" type="primary" size="small" :loading="globalForm.saving.value" @click="globalForm.save">
            <template #icon><n-icon :component="SaveOutline" /></template>
            {{ t('common.save') }}
          </n-button>
        </div>
      </header>

      <n-tabs v-model:value="activeTab" type="line" animated>
        <!-- Tab 1: Providers -->
        <n-tab-pane name="providers" :tab="t('aiSettings.providersTitle')">
          <section class="sre-config-section">
            <div class="section-header-row">
              <p class="sre-config-section-desc" style="margin: 0">{{ t('aiSettings.providersDesc') }}</p>
              <n-space :size="8">
                <n-button size="small" quaternary :loading="testing" @click="handleTestDefault">
                  <template #icon><n-icon :component="PulseOutline" /></template>
                  {{ t('aiSettings.testDefault') }}
                </n-button>
                <n-button size="small" @click="openAddProvider">
                  <template #icon><n-icon :component="AddOutline" /></template>
                  {{ t('aiSettings.addProvider') }}
                </n-button>
              </n-space>
            </div>

            <n-alert v-if="hasProviders && !defaultProviderHealthy" type="error" class="mb-4">
              {{ t('aiSettings.defaultProviderUnhealthy') }}
            </n-alert>

            <n-data-table
              v-if="hasProviders"
              :columns="providerColumns"
              :data="providersConfig!.providers"
              :row-class-name="(row: AIProvider) => row.enabled ? '' : 'provider-row-disabled'"
              size="small"
              :bordered="false"
              style="margin-bottom: 8px"
            />
            <n-alert
              v-else-if="!providersLoading"
              type="warning"
              :bordered="false"
            >
              {{ t('aiSettings.noProvidersWarning') }}
            </n-alert>
          </section>
        </n-tab-pane>

        <!-- Tab 2: Modules -->
        <n-tab-pane name="modules" :tab="t('aiSettings.moduleConfigTitle')">
          <section class="sre-config-section">
            <div class="section-header-row">
              <p class="sre-config-section-desc" style="margin: 0">{{ t('aiSettings.moduleConfigDesc') }}</p>
              <n-space :size="8">
                <n-tag v-if="providersConfig?.default_provider" size="small" :bordered="false" type="info">
                  {{ t('aiSettings.default') }}: {{ providersConfig.default_provider }}
                </n-tag>
                <n-button size="small" quaternary :loading="previewLoading" @click="handlePreviewImpact">
                  <template #icon><n-icon :component="PulseOutline" /></template>
                  {{ t('aiSettings.previewImpact') }}
                </n-button>
              </n-space>
            </div>

            <div v-if="modulesForm.form.platform" class="module-list">
              <div
                v-for="key in moduleKeys"
                :key="key"
                class="module-item"
                :class="{ disabled: !modulesForm.form[key]?.enabled }"
              >
                <div class="module-info">
                  <div class="module-name">
                    {{ moduleLabels[key].name }}
                    <n-tag v-if="modulesForm.form[key]?.enabled" type="success" size="tiny" :bordered="false">{{ t('common.enabled') }}</n-tag>
                    <n-tag v-else size="tiny" :bordered="false">{{ t('common.disabled') }}</n-tag>
                  </div>
                  <div class="module-desc">{{ moduleLabels[key].description }}</div>
                  <div class="module-provider-row" v-if="hasProviders">
                    <span class="module-provider-label">{{ t('aiSettings.providerLabel') }}</span>
                    <n-select
                      :value="modulesForm.form[key]?.provider_key || ''"
                      :options="[{ label: t('aiSettings.default'), value: '' }, ...providerSelectOptions]"
                      size="tiny"
                      style="width: 240px"
                      :placeholder="t('aiSettings.default')"
                      @update:value="(val: string) => setModuleProvider(key, val)"
                    />
                  </div>
                </div>
                <n-switch
                  :value="modulesForm.form[key]?.enabled"
                  @update:value="(val: boolean) => toggleModule(key, val)"
                />
              </div>
            </div>
            <div v-else-if="!modulesForm.loading.value" class="ai-info-empty">
              {{ t('aiSettings.loadModuleFailed') }}
            </div>
          </section>
        </n-tab-pane>

        <!-- Tab 3: Global Settings -->
        <n-tab-pane name="global" :tab="t('aiSettings.globalTab')">
          <div class="global-config-section">
            <h3 class="sre-config-section-title">{{ t('aiSettings.globalTab') }}</h3>
            <p class="sre-config-section-desc">{{ t('aiSettings.globalDesc') }}</p>

            <n-spin :show="globalForm.loading.value">
              <n-form label-placement="left" label-width="180" style="max-width: 560px; margin-top: 16px;">
                <n-form-item :label="t('aiSettings.retryMax')">
                  <n-input-number v-model:value="globalForm.form.retry_max" :min="0" :max="10" style="width: 100%" />
                </n-form-item>
                <n-form-item :label="t('aiSettings.contextMaxChars')">
                  <n-input-number v-model:value="globalForm.form.context_max_chars" :min="1000" :max="100000" :step="1000" style="width: 100%" />
                </n-form-item>
                <n-form-item :label="t('aiSettings.defaultTemperature')">
                  <n-input-number v-model:value="globalForm.form.default_temperature" :min="0" :max="2" :step="0.1" :precision="1" style="width: 100%" />
                </n-form-item>
                <n-form-item :label="t('aiSettings.defaultMaxTokens')">
                  <n-input-number v-model:value="globalForm.form.default_max_tokens" :min="100" :max="32000" :step="100" style="width: 100%" />
                </n-form-item>
                <n-form-item :label="t('aiSettings.monthlyTokenBudget')">
                  <n-input-number v-model:value="globalForm.form.monthly_token_budget" :min="0" :step="100000" style="width: 100%" />
                  <span class="form-hint">{{ t('aiSettings.monthlyTokenBudgetHint') }}</span>
                </n-form-item>
                <n-form-item :label="t('aiSettings.dataMasking')">
                  <div>
                    <n-switch v-model:value="globalForm.form.data_masking_enabled" />
                    <p class="form-desc">{{ t('aiSettings.dataMaskingDesc') }}</p>
                  </div>
                </n-form-item>
              </n-form>
            </n-spin>
          </div>
        </n-tab-pane>
      </n-tabs>

      <!-- Provider Add/Edit Modal -->
      <n-modal
        v-model:show="showModal"
        preset="card"
        :title="editingIndex >= 0 ? t('aiSettings.editProvider') : t('aiSettings.addProvider')"
        style="max-width: 520px"
        :bordered="false"
        :segmented="{ content: true, footer: true }"
      >
        <n-form label-placement="left" label-width="100">
          <n-form-item :label="t('aiSettings.providerKey')" required>
            <n-input
              v-model:value="providerForm.key"
              :placeholder="t('aiSettings.keyPlaceholder')"
              :disabled="editingIndex >= 0"
            />
          </n-form-item>
          <n-form-item :label="t('aiSettings.providerType')">
            <n-select v-model:value="providerForm.provider" :options="providerOptions" />
          </n-form-item>
          <n-form-item :label="t('aiSettings.apiKey')">
            <n-input
              v-model:value="providerForm.api_key"
              type="password"
              show-password-on="click"
              :placeholder="t('aiSettings.apiKeyPlaceholder')"
            />
          </n-form-item>
          <n-form-item :label="t('aiSettings.baseUrl')">
            <n-input
              v-model:value="providerForm.base_url"
              :placeholder="t('aiSettings.baseUrlPlaceholder')"
            />
          </n-form-item>
          <n-form-item :label="t('aiSettings.model')">
            <n-input
              v-model:value="providerForm.model"
              :placeholder="t('aiSettings.modelPlaceholder')"
            />
          </n-form-item>
          <n-form-item :label="t('common.enabled')">
            <n-switch v-model:value="providerForm.enabled" />
          </n-form-item>
        </n-form>
        <template #footer>
          <n-space justify="end">
            <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
            <n-button type="primary" @click="handleProviderSave">
              {{ editingIndex >= 0 ? t('common.update') : t('common.add') }}
            </n-button>
          </n-space>
        </template>
      </n-modal>

      <!-- Label Validation Preview Modal -->
      <n-modal
        v-model:show="showPreviewModal"
        preset="card"
        :title="t('aiSettings.previewTitle')"
        style="max-width: 640px"
        :bordered="false"
        :segmented="{ content: true, footer: true }"
      >
        <div v-if="previewResult" class="preview-stats">
          <n-statistic :label="t('aiSettings.totalRules')" :value="previewResult.total" />
          <n-statistic :label="t('aiSettings.passing')" :value="previewResult.passing">
            <template #suffix><n-tag type="success" size="tiny" :bordered="false">{{ t('aiSettings.pass') }}</n-tag></template>
          </n-statistic>
          <n-statistic :label="t('aiSettings.failing')" :value="previewResult.failing">
            <template #suffix><n-tag type="warning" size="tiny" :bordered="false">{{ t('aiSettings.fail') }}</n-tag></template>
          </n-statistic>
        </div>
        <n-divider v-if="previewResult && previewResult.samples.length > 0" />
        <div v-if="previewResult && previewResult.samples.length > 0" class="preview-samples">
          <div class="preview-samples-title">{{ t('aiSettings.sampleFailingRules') }}</div>
          <div v-for="sample in previewResult.samples" :key="sample.rule_id" class="preview-sample-item">
            <div class="preview-sample-name">
              <n-tag :type="sample.pass ? 'success' : 'warning'" size="tiny" :bordered="false">
                {{ sample.pass ? t('aiSettings.pass') : t('aiSettings.fail') }}
              </n-tag>
              {{ sample.rule_name }}
            </div>
            <div v-if="sample.issues && sample.issues.length > 0" class="preview-sample-issues">
              <div v-for="(issue, i) in sample.issues" :key="i" class="preview-sample-issue">{{ issue }}</div>
            </div>
          </div>
        </div>
        <div v-else-if="previewResult && previewResult.failing === 0" class="ai-info-empty">
          {{ t('aiSettings.allRulesPass') }}
        </div>
        <template #footer>
          <n-space justify="end">
            <n-button @click="showPreviewModal = false">{{ t('common.close') }}</n-button>
          </n-space>
        </template>
      </n-modal>
    </div>
  </NSpin>
</template>

<style scoped>
.ai-settings-page {
  max-width: 880px;
}

.section-header-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

/* Provider Table */
:deep(.provider-row-disabled) {
  opacity: 0.55;
}

/* Module List */
.module-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.module-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  transition: opacity 200ms ease, border-color 200ms ease;
}
.module-item.disabled {
  opacity: 0.65;
}
.module-item:hover {
  border-color: var(--sre-primary-ring, var(--sre-border-strong));
}
.module-info {
  flex: 1;
  min-width: 0;
}
.module-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}
.module-desc {
  font-size: 12px;
  color: var(--sre-text-secondary);
  line-height: 1.5;
}
.module-provider-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}
.module-provider-label {
  font-size: 12px;
  color: var(--sre-text-secondary);
  font-weight: 500;
  white-space: nowrap;
}

.ai-info-empty {
  font-size: 13px;
  color: var(--sre-text-tertiary);
  padding: 16px;
  text-align: center;
}

/* Preview Impact Modal */
.preview-stats {
  display: flex;
  gap: 32px;
  justify-content: center;
}
.preview-samples-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  margin-bottom: 8px;
}
.preview-sample-item {
  padding: 8px 12px;
  border: 1px solid var(--sre-border);
  border-radius: 6px;
  margin-bottom: 6px;
}
.preview-sample-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--sre-text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
}
.preview-sample-issues {
  margin-top: 4px;
  padding-left: 52px;
}
.preview-sample-issue {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  line-height: 1.6;
}
.mb-4 {
  margin-bottom: 16px;
}
.form-desc {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin-top: 4px;
  line-height: 1.5;
}
</style>
