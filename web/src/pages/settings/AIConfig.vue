<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { NButton, NIcon, NSwitch, NSelect, NInput, NInputNumber, NFormItem, NSpin, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { PulseOutline, SaveOutline } from '@vicons/ionicons5'
import { aiApi } from '@/api'

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const lastTestResult = ref<{ success: boolean; message: string; time: string } | null>(null)

const form = reactive({
  enabled: false,
  provider: 'openai',
  api_key: '',
  base_url: '',
  model: '',
  temperature: 0.3,
  max_tokens: 1024,
  system_prompt: '',
})

const providerOptions = [
  { label: 'OpenAI', value: 'openai' },
  { label: 'Azure OpenAI', value: 'azure' },
  { label: 'Ollama (Local)', value: 'ollama' },
  { label: 'Custom / Compatible', value: 'custom' },
]

async function fetchConfig() {
  loading.value = true
  try {
    const res = await aiApi.getConfig()
    if (res.data.data) {
      const d = res.data.data
      form.enabled = d.enabled
      form.provider = d.provider || 'openai'
      form.api_key = d.api_key || ''
      form.base_url = d.base_url || ''
      form.model = d.model || ''
      form.temperature = (d as any).temperature ?? 0.3
      form.max_tokens = (d as any).max_tokens ?? 1024
      form.system_prompt = (d as any).system_prompt || ''
    }
  } catch (err: any) {
    message.error(err.message)
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await aiApi.updateConfig({ ...form })
    message.success(t('common.savedSuccess'))
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  testing.value = true
  try {
    const res = await aiApi.testConnection()
    const ok = !!res.data.data?.success
    lastTestResult.value = {
      success: ok,
      message: res.data.data?.message || (ok ? t('settings.aiTestSuccess') : t('settings.aiTestFailed')),
      time: new Date().toLocaleTimeString(),
    }
    ok ? message.success(t('settings.aiTestSuccess')) : message.error(lastTestResult.value.message)
  } catch (err: any) {
    lastTestResult.value = { success: false, message: err.message, time: new Date().toLocaleTimeString() }
    message.error(err.message)
  } finally {
    testing.value = false
  }
}

onMounted(fetchConfig)
</script>

<template>
  <NSpin :show="loading">
    <div class="sre-config-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">AI Configuration</h2>
          <p class="sre-config-header-sub">Configure language model providers (OpenAI / Azure / Ollama / Custom) for root-cause analysis and alert summarization.</p>
        </div>
        <div class="sre-config-header-actions">
          <NButton size="small" :loading="testing" @click="testConnection">
            <template #icon><NIcon :component="PulseOutline" /></template>
            {{ t('common.test') }}
          </NButton>
          <NButton type="primary" size="small" :loading="saving" @click="save">
            <template #icon><NIcon :component="SaveOutline" /></template>
            {{ t('common.save') }}
          </NButton>
        </div>
      </header>

      <div v-if="lastTestResult" class="sre-config-status" :data-tone="lastTestResult.success ? 'success' : 'error'">
        <span class="sre-dot" :data-severity="lastTestResult.success ? 'success' : 'critical'"></span>
        <span>{{ lastTestResult.message }}</span>
        <span class="sre-meta-divider"></span>
        <span class="tnum">{{ lastTestResult.time }}</span>
      </div>

      <div class="config-sections sre-stagger">
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">Provider</h3>
          <p class="sre-config-section-desc">Select the upstream LLM and supply credentials. The base URL is optional for OpenAI defaults.</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.aiEnabled')" class="full-row">
              <NSwitch v-model:value="form.enabled" />
            </NFormItem>
            <NFormItem :label="t('settings.aiProvider')">
              <NSelect v-model:value="form.provider" :options="providerOptions" />
            </NFormItem>
            <NFormItem :label="t('settings.aiModel')">
              <NInput v-model:value="form.model" :placeholder="t('settings.aiModelPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.aiBaseUrl')" class="full-row">
              <NInput v-model:value="form.base_url" :placeholder="t('settings.aiBaseUrlPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.aiApiKey')" class="full-row">
              <NInput v-model:value="form.api_key" type="password" show-password-on="click" :placeholder="t('settings.aiApiKeyPlaceholder')" />
            </NFormItem>
          </div>
        </section>

        <section class="sre-config-section">
          <h3 class="sre-config-section-title">Behavior</h3>
          <p class="sre-config-section-desc">Tune sampling and the system prompt used as context for every analysis request.</p>
          <div class="sre-config-form-grid">
            <NFormItem label="Temperature">
              <NInputNumber v-model:value="form.temperature" :min="0" :max="2" :step="0.1" style="width: 100%" />
            </NFormItem>
            <NFormItem label="Max Tokens">
              <NInputNumber v-model:value="form.max_tokens" :min="64" :max="32000" :step="64" style="width: 100%" />
            </NFormItem>
            <NFormItem label="System Prompt" class="full-row">
              <NInput v-model:value="form.system_prompt" type="textarea" :rows="4" placeholder="You are an expert SRE assistant..." />
            </NFormItem>
          </div>
        </section>
      </div>
    </div>
  </NSpin>
</template>

<style scoped>
</style>
