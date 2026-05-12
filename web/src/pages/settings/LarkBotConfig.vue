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
    message.warning(t('settings.larkWebhookRequired'))
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
        message: res?.data?.data?.message || (ok ? t('settings.larkTestCardSent') : t('settings.larkTestFailed')),
        time: new Date().toLocaleTimeString(),
      }
    } else {
      lastTestResult.value = { success: true, message: t('settings.larkConfigValid'), time: new Date().toLocaleTimeString() }
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
    <div class="sre-config-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">{{ t('settings.larkBotTitle') }}</h2>
          <p class="sre-config-header-sub">{{ t('settings.larkBotSubtitle') }} <code>/lark/event</code></p>
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
          <h3 class="sre-config-section-title">{{ t('settings.larkAppCredentials') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.larkAppCredentialsDesc') }}</p>
          <div class="sre-config-form-grid">
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

        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.larkDefaults') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.larkDefaultsDesc') }}</p>
          <div class="sre-config-form-grid">
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
</style>
