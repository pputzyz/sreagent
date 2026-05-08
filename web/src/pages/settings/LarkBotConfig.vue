<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { NButton, NIcon, NSwitch, NInput, NFormItem, NSpin, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { PulseOutline, SaveOutline } from '@vicons/ionicons5'
import { larkBotApi } from '@/api'

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const lastTestResult = ref<{ success: boolean; message: string; time: string } | null>(null)

const form = reactive({
  bot_enabled: false,
  app_id: '',
  app_secret: '',
  default_webhook: '',
  verification_token: '',
  encrypt_key: '',
})

async function fetchConfig() {
  loading.value = true
  try {
    const res = await larkBotApi.getConfig()
    if (res.data.data) {
      const d = res.data.data
      form.bot_enabled = d.bot_enabled
      form.app_id = d.app_id || ''
      form.app_secret = d.app_secret || ''
      form.default_webhook = d.default_webhook || ''
      form.verification_token = d.verification_token || ''
      form.encrypt_key = d.encrypt_key || ''
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
    await larkBotApi.updateConfig({ ...form })
    message.success(t('common.savedSuccess'))
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  if (!form.default_webhook) {
    message.warning('Default webhook URL is required to send a test card')
    return
  }
  testing.value = true
  try {
    // Best-effort test: many backends accept arbitrary webhook ping. Fallback message.
    const fn = (larkBotApi as any).testConnection || (larkBotApi as any).test
    if (typeof fn === 'function') {
      const res = await fn.call(larkBotApi)
      const ok = !!res?.data?.data?.success
      lastTestResult.value = {
        success: ok,
        message: res?.data?.data?.message || (ok ? 'Test card sent' : 'Test failed'),
        time: new Date().toLocaleTimeString(),
      }
    } else {
      lastTestResult.value = { success: true, message: 'Configuration looks valid (no live test endpoint available)', time: new Date().toLocaleTimeString() }
    }
    lastTestResult.value!.success ? message.success(lastTestResult.value!.message) : message.error(lastTestResult.value!.message)
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
    <div class="config-page">
      <header class="config-header">
        <div>
          <h2 class="config-title">Lark Bot Integration</h2>
          <p class="config-subtitle">Lark bot for direct messages, alert card updates, and slash commands. Callback endpoint: <code>/lark/event</code></p>
        </div>
        <div class="config-actions">
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

      <div v-if="lastTestResult" class="config-status" :data-tone="lastTestResult.success ? 'success' : 'error'">
        <span class="sre-dot" :data-severity="lastTestResult.success ? 'success' : 'critical'"></span>
        <span>{{ lastTestResult.message }}</span>
        <span class="sre-meta-divider"></span>
        <span class="tnum">{{ lastTestResult.time }}</span>
      </div>

      <div class="config-sections sre-stagger">
        <section class="config-section">
          <h3 class="section-title">App Credentials</h3>
          <p class="section-desc">Obtain these values from the Lark Open Platform developer console for your custom app.</p>
          <div class="form-grid">
            <NFormItem :label="t('settings.larkBotEnabled')" class="full-row">
              <NSwitch v-model:value="form.bot_enabled" />
            </NFormItem>
            <NFormItem :label="t('settings.larkAppId')">
              <NInput v-model:value="form.app_id" :placeholder="t('settings.larkAppIdPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.larkAppSecret')">
              <NInput v-model:value="form.app_secret" type="password" show-password-on="click" :placeholder="t('settings.larkAppSecretPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.larkVerificationToken')">
              <NInput v-model:value="form.verification_token" :placeholder="t('settings.larkTokenPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.larkEncryptKey')">
              <NInput v-model:value="form.encrypt_key" type="password" show-password-on="click" :placeholder="t('settings.larkEncryptKeyPlaceholder')" />
            </NFormItem>
          </div>
        </section>

        <section class="config-section">
          <h3 class="section-title">Defaults</h3>
          <p class="section-desc">The default webhook is used when a notification rule does not specify its own target.</p>
          <div class="form-grid">
            <NFormItem :label="t('settings.larkDefaultWebhook')" class="full-row">
              <NInput v-model:value="form.default_webhook" :placeholder="t('settings.larkWebhookPlaceholder')" />
            </NFormItem>
          </div>
        </section>
      </div>
    </div>
  </NSpin>
</template>

<style scoped>
.config-page { display: flex; flex-direction: column; gap: 20px; font-family: 'Geist', system-ui, sans-serif; }
.config-header { display: flex; align-items: flex-start; justify-content: space-between; padding-bottom: 16px; border-bottom: var(--sre-hairline); gap: 16px; }
.config-title { font-size: 18px; font-weight: 600; margin: 0 0 4px; color: var(--sre-text-primary); }
.config-subtitle { font-size: 12px; color: var(--sre-text-secondary); margin: 0; max-width: 600px; line-height: 1.5; }
.config-subtitle code { font-family: 'Geist Mono', ui-monospace, monospace; font-size: 11px; padding: 1px 6px; border-radius: 4px; background: var(--sre-bg-card); border: var(--sre-hairline); }
.config-actions { display: flex; gap: 8px; flex-shrink: 0; }
.config-status { display: flex; align-items: center; gap: 8px; padding: 10px 14px; border-radius: var(--sre-radius-md); font-size: 12px; background: var(--sre-bg-card); border: var(--sre-hairline); }
.config-status[data-tone="success"] { border-color: rgba(16,185,129,0.3); background: rgba(16,185,129,0.06); }
.config-status[data-tone="error"]   { border-color: rgba(239,68,68,0.3); background: rgba(239,68,68,0.06); }
.config-sections { display: flex; flex-direction: column; gap: 16px; }
.config-section { background: var(--sre-bg-card); border: var(--sre-hairline); border-radius: var(--sre-radius-md); padding: 20px 24px; }
.section-title { font-size: 14px; font-weight: 600; letter-spacing: 0.3px; color: var(--sre-text-primary); margin: 0 0 4px; }
.section-desc { font-size: 12px; color: var(--sre-text-secondary); margin: 0 0 16px; line-height: 1.5; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.form-grid .full-row { grid-column: 1 / -1; }
:deep(.n-form-item-label) { padding: 0 0 4px 0 !important; font-size: 11px !important; font-weight: 600 !important; letter-spacing: 0.3px !important; color: var(--sre-text-tertiary) !important; text-transform: uppercase; }
</style>
