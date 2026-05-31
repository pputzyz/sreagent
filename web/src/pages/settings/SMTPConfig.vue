<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NButton, NIcon, NSwitch, NInput, NInputNumber, NFormItem, NSpin, NSpace } from 'naive-ui'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { PulseOutline, SaveOutline } from '@vicons/ionicons5'
import { smtpSettingsApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import { useConfigForm } from '@/composables'

const message = useMessage()
const { t } = useI18n()

const { form, loading, isDirty, save, load, markSaved } = useConfigForm({
  load: () => smtpSettingsApi.getConfig().then(r => r.data.data),
  save: (f) => smtpSettingsApi.updateConfig({ ...f }),
})

// Per-section saving state
const savingServer = ref(false)
const savingSender = ref(false)

// Test state
const testing = ref(false)
const testTo = ref('')
const lastTestResult = ref<{ success: boolean; message: string; time: string } | null>(null)

async function saveServer() {
  if (form.enabled) {
    if (!form.smtp_host?.trim()) { message.warning(t('settings.smtpHostRequired')); return }
    if (!form.smtp_port) { message.warning(t('settings.smtpPortRequired')); return }
    if (!form.username?.trim()) { message.warning(t('settings.smtpUsernameRequired')); return }
    if (!form.password?.trim() || form.password === '********') { message.warning(t('settings.smtpPasswordRequired')); return }
  }
  savingServer.value = true
  try {
    await smtpSettingsApi.updateConfig({ ...form })
    markSaved()
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { savingServer.value = false }
}

async function saveSender() {
  if (form.enabled && !form.from?.trim()) {
    message.warning(t('settings.smtpFromRequired'))
    return
  }
  savingSender.value = true
  try {
    await smtpSettingsApi.updateConfig({ ...form })
    markSaved()
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { savingSender.value = false }
}

async function testConnection() {
  if (!testTo.value) {
    message.warning(t('smtp.enterTestEmail'))
    return
  }
  if (isDirty.value) {
    const ok = await save()
    if (!ok) return
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

onMounted(() => load())
</script>

<template>
  <NSpin :show="loading">
    <div class="sre-config-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">{{ t('settings.smtpTitle') }}</h2>
          <p class="sre-config-header-sub">{{ t('settings.smtpSubtitle') }}</p>
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
            <NFormItem :label="t('smtp.host')" :required="form.enabled">
              <NInput v-model:value="form.smtp_host" :placeholder="t('smtp.hostPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('smtp.port')" :required="form.enabled">
              <NInputNumber v-model:value="form.smtp_port" :min="1" :max="65535" style="width: 100%" />
            </NFormItem>
            <NFormItem :label="t('smtp.tls')">
              <NSwitch v-model:value="form.smtp_tls" />
            </NFormItem>
            <NFormItem :label="t('smtp.username')" :required="form.enabled">
              <NInput v-model:value="form.username" :placeholder="t('smtp.usernamePlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('smtp.password')" class="full-row" :required="form.enabled">
              <NInput
                v-model:value="form.password"
                type="password"
                show-password-on="click"
                :placeholder="form.password === '********' ? t('smtp.passwordMasked') : t('smtp.passwordPlaceholder')"
                @focus="handlePasswordFocus"
              />
            </NFormItem>
          </div>
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingServer" @click="saveServer">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
          </div>
        </section>

        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.smtpSenderSection') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.smtpSenderDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('smtp.from')" :required="form.enabled" class="full-row">
              <NInput v-model:value="form.from" :placeholder="t('smtp.fromPlaceholder')" />
            </NFormItem>
          </div>
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingSender" @click="saveSender">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
          </div>
        </section>

        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.smtpTestDelivery') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.smtpTestDeliveryDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('smtp.testRecipient')" class="full-row">
              <NSpace :size="8">
                <NInput v-model:value="testTo" :placeholder="t('smtp.testRecipientPlaceholder')" style="flex: 1" />
                <NButton type="primary" size="small" :loading="testing" :disabled="!testTo" @click="testConnection">
                  <template #icon><NIcon :component="PulseOutline" /></template>
                  {{ t('smtp.sendTest') }}
                </NButton>
              </NSpace>
            </NFormItem>
          </div>
        </section>
      </div>
    </div>
  </NSpin>
</template>

<style scoped>
.section-footer {
  display: flex; justify-content: flex-end;
  padding-top: 16px; margin-top: 16px;
  border-top: var(--sre-hairline);
}
</style>
