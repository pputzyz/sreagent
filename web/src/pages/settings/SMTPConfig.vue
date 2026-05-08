<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { NButton, NIcon, NSwitch, NInput, NInputNumber, NFormItem, NSpin, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { PulseOutline, SaveOutline } from '@vicons/ionicons5'
import { smtpSettingsApi } from '@/api'

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
      form.from_name = (d as any).from_name || ''
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
    await smtpSettingsApi.updateConfig({ ...form })
    message.success(t('common.success'))
  } catch (err: any) {
    message.error(err.message)
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
  } catch (err: any) {
    lastTestResult.value = { success: false, message: err.message, time: new Date().toLocaleTimeString() }
    message.error(err.message)
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
    <div class="config-page">
      <header class="config-header">
        <div>
          <h2 class="config-title">SMTP Email</h2>
          <p class="config-subtitle">Outbound email server used by escalation policies and direct notifications.</p>
        </div>
        <div class="config-actions">
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

      <div v-if="lastTestResult" class="config-status" :data-tone="lastTestResult.success ? 'success' : 'error'">
        <span class="sre-dot" :data-severity="lastTestResult.success ? 'success' : 'critical'"></span>
        <span>{{ lastTestResult.message }}</span>
        <span class="sre-meta-divider"></span>
        <span class="tnum">{{ lastTestResult.time }}</span>
      </div>

      <div class="config-sections sre-stagger">
        <section class="config-section">
          <h3 class="section-title">Server</h3>
          <p class="section-desc">Connection details and SMTP authentication. STARTTLS is recommended on port 587.</p>
          <div class="form-grid">
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

        <section class="config-section">
          <h3 class="section-title">Sender</h3>
          <p class="section-desc">Identity used in the From header. The display name appears in most email clients.</p>
          <div class="form-grid">
            <NFormItem :label="t('smtp.from')">
              <NInput v-model:value="form.from" :placeholder="t('smtp.fromPlaceholder')" />
            </NFormItem>
            <NFormItem label="From Name">
              <NInput v-model:value="form.from_name" placeholder="SREAgent Alerts" />
            </NFormItem>
          </div>
        </section>

        <section class="config-section">
          <h3 class="section-title">Test Delivery</h3>
          <p class="section-desc">Send a real test message using the configuration above. Save first if you have unsaved changes.</p>
          <div class="form-grid">
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
.config-page { display: flex; flex-direction: column; gap: 20px; font-family: 'Geist', system-ui, sans-serif; }
.config-header { display: flex; align-items: flex-start; justify-content: space-between; padding-bottom: 16px; border-bottom: var(--sre-hairline); gap: 16px; }
.config-title { font-size: 18px; font-weight: 600; margin: 0 0 4px; color: var(--sre-text-primary); }
.config-subtitle { font-size: 12px; color: var(--sre-text-secondary); margin: 0; max-width: 600px; line-height: 1.5; }
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
