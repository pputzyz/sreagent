<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import {
  NButton, NIcon, NSwitch, NAlert, NCard, NDivider, NSpin,
  NSpace, NTag, NSelect, NInput, NModal, NForm, NFormItem,
  NPopconfirm, NStatistic, useMessage,
} from 'naive-ui'
import { PulseOutline, SaveOutline, SparklesOutline, AddOutline, TrashOutline, CreateOutline, StarOutline } from '@vicons/ionicons5'
import { aiApi, aiModuleApi, alertRuleApi } from '@/api'
import type { AIModuleConfig, AIProvider, AIProvidersConfig } from '@/types/ai-module'
import { getErrorMessage } from '@/utils/format'

const message = useMessage()

// ─── Providers config ───
const providersLoading = ref(false)
const providersConfig = ref<AIProvidersConfig | null>(null)

// ─── Module config ───
const moduleLoading = ref(false)
const saving = ref(false)
const testing = ref(false)
const testingProvider = ref<string | null>(null)
const modules = ref<AIModuleConfig | null>(null)

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
  { label: 'Custom / Compatible', value: 'custom' },
]

const moduleLabels: Record<keyof AIModuleConfig, { name: string; description: string }> = {
  platform: {
    name: '平台智能助手',
    description: '全局 AI 助手浮窗，支持自然语言问答、告警上下文对话',
  },
  chat: {
    name: 'AI 对话',
    description: '告警详情页的 AI 对话面板，支持告警分析和通用问答模式',
  },
  rule_gen: {
    name: '规则生成',
    description: '基于自然语言描述自动生成 PromQL/MetricsQL 告警规则表达式',
  },
  analysis: {
    name: '告警分析',
    description: '告警事件的 AI 根因分析报告和 SOP 建议生成',
  },
  agent: {
    name: 'AI Agent',
    description: '自主告警处理 Agent，支持自动诊断、关联分析和处理建议',
  },
}

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

// ─── Fetch module config ───
async function fetchModules() {
  moduleLoading.value = true
  try {
    const res = await aiModuleApi.getModules()
    modules.value = res.data.data
  } catch {
    modules.value = null
  } finally {
    moduleLoading.value = false
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
    message.error('Provider key is required')
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
    // Edit existing
    providersConfig.value.providers[editingIndex.value] = entry
  } else {
    // Check duplicate key
    if (providersConfig.value.providers.some(p => p.key === entry.key)) {
      message.error('Provider key already exists')
      return
    }
    providersConfig.value.providers.push(entry)
    // If first provider, set as default
    if (providersConfig.value.providers.length === 1) {
      providersConfig.value.default_provider = entry.key
    }
  }

  showModal.value = false
  saveProvidersConfig()
}

function deleteProvider(index: number) {
  if (!providersConfig.value) return
  const key = providersConfig.value.providers[index].key
  providersConfig.value.providers.splice(index, 1)
  // Clear default if deleted
  if (providersConfig.value.default_provider === key) {
    providersConfig.value.default_provider = providersConfig.value.providers[0]?.key ?? ''
  }
  saveProvidersConfig()
}

function setDefaultProvider(key: string) {
  if (!providersConfig.value) return
  providersConfig.value.default_provider = key
  saveProvidersConfig()
}

async function saveProvidersConfig() {
  if (!providersConfig.value) return
  try {
    await aiApi.saveProviders(providersConfig.value)
    message.success('Provider configuration saved')
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

// ─── Module config ───
function toggleModule(key: keyof AIModuleConfig, val: boolean) {
  if (!modules.value) return
  modules.value[key].enabled = val
}

function setModuleProvider(key: keyof AIModuleConfig, providerKey: string) {
  if (!modules.value) return
  modules.value[key].provider_key = providerKey
}

// ─── Save modules ───
async function handleSave() {
  if (!modules.value) return
  saving.value = true
  try {
    await aiModuleApi.updateModules(modules.value)
    message.success('AI module configuration saved')
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
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
      ? message.success(res.data.data?.message || 'Connection test successful')
      : message.error(res.data.data?.message || 'Connection test failed')
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
    message.success(res.data.data?.message || 'Connection test successful')
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
    custom: 'Custom / Compatible',
  }
  return map[p] || p
}

onMounted(() => {
  fetchProviders()
  fetchModules()
})
</script>

<template>
  <NSpin :show="providersLoading && moduleLoading">
    <div class="sre-config-page ai-settings-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">
            <n-icon :component="SparklesOutline" :size="20" style="margin-right: 8px; vertical-align: -3px;" />
            AI Configuration
          </h2>
          <p class="sre-config-header-sub">Manage AI providers, module assignments, and connections</p>
        </div>
        <div class="sre-config-header-actions">
          <n-button size="small" :loading="previewLoading" @click="handlePreviewImpact">
            <template #icon><n-icon :component="PulseOutline" /></template>
            Preview Impact
          </n-button>
          <n-button size="small" :loading="testing" @click="handleTestDefault">
            <template #icon><n-icon :component="PulseOutline" /></template>
            Test Default
          </n-button>
          <n-button type="primary" size="small" :loading="saving" @click="handleSave">
            <template #icon><n-icon :component="SaveOutline" /></template>
            Save Modules
          </n-button>
        </div>
      </header>

      <!-- Warning: No providers configured -->
      <n-alert
        v-if="!providersLoading && !hasProviders"
        type="warning"
        :bordered="false"
        style="margin-bottom: 20px"
      >
        No AI providers configured. Add a provider below to enable AI features.
      </n-alert>

      <div class="config-sections sre-stagger">
        <!-- Section 1: Providers Manager -->
        <section class="sre-config-section">
          <div class="section-header-row">
            <div>
              <h3 class="sre-config-section-title">AI Providers</h3>
              <p class="sre-config-section-desc">Configure multiple AI providers. Each module can use a different provider.</p>
            </div>
            <n-button size="small" @click="openAddProvider">
              <template #icon><n-icon :component="AddOutline" /></template>
              Add Provider
            </n-button>
          </div>

          <div v-if="hasProviders" class="providers-grid">
            <div
              v-for="(provider, idx) in providersConfig!.providers"
              :key="provider.key"
              class="provider-card"
              :class="{ 'is-default': providersConfig!.default_provider === provider.key, disabled: !provider.enabled }"
            >
              <div class="provider-card-header">
                <div class="provider-card-title">
                  <span class="provider-key">{{ provider.key }}</span>
                  <n-tag v-if="providersConfig!.default_provider === provider.key" type="warning" size="tiny" :bordered="false">
                    Default
                  </n-tag>
                  <n-tag :type="provider.enabled ? 'success' : 'default'" size="tiny" :bordered="false">
                    {{ provider.enabled ? 'Enabled' : 'Disabled' }}
                  </n-tag>
                </div>
                <div class="provider-card-actions">
                  <n-button text size="small" @click="handleTestProvider(provider.key)" :loading="testingProvider === provider.key">
                    <template #icon><n-icon :component="PulseOutline" /></template>
                  </n-button>
                  <n-button text size="small" @click="setDefaultProvider(provider.key)" :disabled="providersConfig!.default_provider === provider.key">
                    <template #icon><n-icon :component="StarOutline" /></template>
                  </n-button>
                  <n-button text size="small" @click="openEditProvider(idx)">
                    <template #icon><n-icon :component="CreateOutline" /></template>
                  </n-button>
                  <n-popconfirm @positive-click="deleteProvider(idx)">
                    <template #trigger>
                      <n-button text size="small" type="error">
                        <template #icon><n-icon :component="TrashOutline" /></template>
                      </n-button>
                    </template>
                    Delete provider "{{ provider.key }}"?
                  </n-popconfirm>
                </div>
              </div>
              <div class="provider-card-body">
                <div class="provider-detail">
                  <span class="provider-detail-label">Type</span>
                  <span class="provider-detail-value">{{ providerTypeLabel(provider.provider) }}</span>
                </div>
                <div class="provider-detail">
                  <span class="provider-detail-label">Model</span>
                  <span class="provider-detail-value mono">{{ provider.model || '-' }}</span>
                </div>
                <div class="provider-detail full-row">
                  <span class="provider-detail-label">Base URL</span>
                  <span class="provider-detail-value mono">{{ provider.base_url || 'Default' }}</span>
                </div>
              </div>
            </div>
          </div>
          <div v-else-if="!providersLoading" class="ai-info-empty">
            No providers configured yet. Click "Add Provider" to get started.
          </div>
        </section>

        <n-divider />

        <!-- Section 2: Module Toggles -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">Module Configuration</h3>
          <p class="sre-config-section-desc">Control each AI module and assign a specific provider</p>

          <div v-if="modules" class="module-list">
            <div
              v-for="key in moduleKeys"
              :key="key"
              class="module-item"
              :class="{ disabled: !modules[key].enabled }"
            >
              <div class="module-info">
                <div class="module-name">
                  {{ moduleLabels[key].name }}
                  <n-tag v-if="modules[key].enabled" type="success" size="tiny" :bordered="false">Enabled</n-tag>
                  <n-tag v-else size="tiny" :bordered="false">Disabled</n-tag>
                </div>
                <div class="module-desc">{{ moduleLabels[key].description }}</div>
                <div class="module-provider-row" v-if="hasProviders">
                  <span class="module-provider-label">Provider:</span>
                  <n-select
                    :value="modules[key].provider_key || ''"
                    :options="[{ label: 'Default', value: '' }, ...providerSelectOptions]"
                    size="tiny"
                    style="width: 240px"
                    placeholder="Default"
                    @update:value="(val: string) => setModuleProvider(key, val)"
                  />
                </div>
              </div>
              <n-switch
                :value="modules[key].enabled"
                @update:value="(val: boolean) => toggleModule(key, val)"
              />
            </div>
          </div>
          <div v-else-if="!moduleLoading" class="ai-info-empty">
            Failed to load module configuration
          </div>
        </section>
      </div>

      <!-- Provider Add/Edit Modal -->
      <n-modal
        v-model:show="showModal"
        preset="card"
        :title="editingIndex >= 0 ? 'Edit Provider' : 'Add Provider'"
        style="max-width: 520px"
        :bordered="false"
        :segmented="{ content: true, footer: true }"
      >
        <n-form label-placement="left" label-width="100">
          <n-form-item label="Key" required>
            <n-input
              v-model:value="providerForm.key"
              placeholder="e.g. openai-main"
              :disabled="editingIndex >= 0"
            />
          </n-form-item>
          <n-form-item label="Provider Type">
            <n-select v-model:value="providerForm.provider" :options="providerOptions" />
          </n-form-item>
          <n-form-item label="API Key">
            <n-input
              v-model:value="providerForm.api_key"
              type="password"
              show-password-on="click"
              placeholder="Enter API key"
            />
          </n-form-item>
          <n-form-item label="Base URL">
            <n-input
              v-model:value="providerForm.base_url"
              placeholder="https://api.openai.com/v1"
            />
          </n-form-item>
          <n-form-item label="Model">
            <n-input
              v-model:value="providerForm.model"
              placeholder="e.g. gpt-4o"
            />
          </n-form-item>
          <n-form-item label="Enabled">
            <n-switch v-model:value="providerForm.enabled" />
          </n-form-item>
        </n-form>
        <template #footer>
          <n-space justify="end">
            <n-button @click="showModal = false">Cancel</n-button>
            <n-button type="primary" @click="handleProviderSave">
              {{ editingIndex >= 0 ? 'Update' : 'Add' }}
            </n-button>
          </n-space>
        </template>
      </n-modal>

      <!-- Label Validation Preview Modal -->
      <n-modal
        v-model:show="showPreviewModal"
        preset="card"
        title="Label Validation Impact"
        style="max-width: 640px"
        :bordered="false"
        :segmented="{ content: true, footer: true }"
      >
        <div v-if="previewResult" class="preview-stats">
          <n-statistic label="Total Rules" :value="previewResult.total" />
          <n-statistic label="Passing" :value="previewResult.passing">
            <template #suffix><n-tag type="success" size="tiny" :bordered="false">Pass</n-tag></template>
          </n-statistic>
          <n-statistic label="Failing" :value="previewResult.failing">
            <template #suffix><n-tag type="warning" size="tiny" :bordered="false">Fail</n-tag></template>
          </n-statistic>
        </div>
        <n-divider v-if="previewResult && previewResult.samples.length > 0" />
        <div v-if="previewResult && previewResult.samples.length > 0" class="preview-samples">
          <div class="preview-samples-title">Sample Failing Rules</div>
          <div v-for="sample in previewResult.samples" :key="sample.rule_id" class="preview-sample-item">
            <div class="preview-sample-name">
              <n-tag :type="sample.pass ? 'success' : 'warning'" size="tiny" :bordered="false">
                {{ sample.pass ? 'Pass' : 'Fail' }}
              </n-tag>
              {{ sample.rule_name }}
            </div>
            <div v-if="sample.issues && sample.issues.length > 0" class="preview-sample-issues">
              <div v-for="(issue, i) in sample.issues" :key="i" class="preview-sample-issue">{{ issue }}</div>
            </div>
          </div>
        </div>
        <div v-else-if="previewResult && previewResult.failing === 0" class="ai-info-empty">
          All rules pass label validation.
        </div>
        <template #footer>
          <n-space justify="end">
            <n-button @click="showPreviewModal = false">Close</n-button>
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

/* Provider Cards */
.providers-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.provider-card {
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  background: var(--sre-bg-card);
  transition: opacity 200ms ease, border-color 200ms ease;
}
.provider-card.disabled {
  opacity: 0.6;
}
.provider-card.is-default {
  border-color: var(--sre-warning, #f0a020);
}
.provider-card:hover {
  border-color: var(--sre-primary-ring, var(--sre-border-strong));
}
.provider-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px 8px;
}
.provider-card-title {
  display: flex;
  align-items: center;
  gap: 8px;
}
.provider-key {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  font-family: var(--sre-font-mono, monospace);
}
.provider-card-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}
.provider-card-body {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 24px;
  padding: 4px 16px 12px;
}
.provider-detail {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.provider-detail.full-row {
  grid-column: 1 / -1;
}
.provider-detail-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.provider-detail-value {
  font-size: 13px;
  color: var(--sre-text-primary);
  font-weight: 500;
}
.provider-detail-value.mono {
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
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
  color: var(--sre-text-tertiary);
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
  color: var(--sre-text-tertiary);
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
</style>
