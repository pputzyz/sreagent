<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { NButton, NIcon, NSwitch, NInput, NInputNumber, NFormItem, NSpin, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { PulseOutline, SaveOutline } from '@vicons/ionicons5'
import { smtpSettingsApi } from '@/api'
import { getErrorMessage } from '@/utils/format'

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const testTo = ref('')
const lastTestResult = ref<{ success: boolean; message: string; time: string } | null>(null)

const form = reactive({
  enabled: false,
  smtp_host: '',
  smtp_port: 587,
  smtp_tls: true,
  username: '',
  password: '',
  from: '',
  from_name: '',
})

async function fetchConfig() {
  loading.value = true
  try {
    const res = await smtpSettingsApi.getConfig()
    if (res.data.data) {
      const d = res.data.data
      form.enabled = d.enabled
      form.smtp_host = d.smtp_host || ''
      form.smtp_port = d.smtp_port || 587
      form.smtp_tls = d.smtp_tls ?? true
      form.username = d.username || ''
      form.password = d.password || ''
      form.from = d.from || ''
      form.from_name = (d as Record<string, unknown>).from_name as string || ''
    }
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await smtpSettingsApi.updateConfig({ ...form })
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  if (!testTo.value) {
    message.warning(t('smtp.enterTestEmail'))
    return
  }
  testing.value = true
  try {
    const res = await smtpSettingsApi.testConnection(testTo.value)
    const msg = res.data.data?.message || t('common.success')
    lastTestResult.value = { success: true, message: msg, time: new Date().toLocaleTimeString() }
    message.success(msg)
  } catch (err: unknown) {
    const errMsg = getErrorMessage(err)
    lastTestResult.value = { success: false, message: errMsg, time: new Date().toLocaleTimeString() }
    message.error(errMsg)
  } finally {
    testing.value = false
  }
}

function handlePasswordFocus() {
  if (form.password === '********') form.password = ''
}

onMounted(fetchConfig)
</script>

<template>
  <NSpin :show="loading">
    <div class="sre-config-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">{{ t('settings.smtpTitle') }}</h2>
          <p class="sre-config-header-sub">{{ t('settings.smtpSubtitle') }}</p>
        </div>
        <div class="sre-config-header-actions">
          <NButton size="small" :loading="testing" :disabled="!testTo" @click="testConnection">
            <template #icon><NIcon :component="PulseOutline" /></template>
            {{ t('smtp.sendTest') }}
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
          <h3 class="sre-config-section-title">{{ t('settings.smtpServerSection') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.smtpServerDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('smtp.enabled')" class="full-row">
              <NSwitch v-model:value="form.enabled" />
            </NFormItem>
            <NFormItem :label="t('smtp.host')">
              <NInput v-model:value="form.smtp_host" :placeholder="t('smtp.hostPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('smtp.port')">
              <NInputNumber v-model:value="form.smtp_port" :min="1" :max="65535" style="width: 100%" />
            </NFormItem>
            <NFormItem :label="t('smtp.tls')">
              <NSwitch v-model:value="form.smtp_tls" />
            </NFormItem>
            <NFormItem :label="t('smtp.username')">
              <NInput v-model:value="form.username" :placeholder="t('smtp.usernamePlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('smtp.password')" class="full-row">
              <NInput
                v-model:value="form.password"
                type="password"
                show-password-on="click"
                :placeholder="form.password === '********' ? t('smtp.passwordMasked') : t('smtp.passwordPlaceholder')"
                @focus="handlePasswordFocus"
              />
            </NFormItem>
          </div>
        </section>

        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.smtpSenderSection') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.smtpSenderDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('smtp.from')">
              <NInput v-model:value="form.from" :placeholder="t('smtp.fromPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.smtpFromName')">
              <NInput v-model:value="form.from_name" :placeholder="t('settings.smtpFromNamePlaceholder')" />
            </NFormItem>
          </div>
        </section>

        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.smtpTestDelivery') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.smtpTestDeliveryDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('smtp.testRecipient')" class="full-row">
              <NInput v-model:value="testTo" :placeholder="t('smtp.testRecipientPlaceholder')" />
            </NFormItem>
          </div>
        </section>
      </div>
    </div>
  </NSpin>
</template>

<style scoped>
</style>
