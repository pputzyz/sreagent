<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { NButton, NIcon, NSwitch, NSelect, NInput, NFormItem, NSpin, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { PulseOutline, SaveOutline } from '@vicons/ionicons5'
import { oidcSettingsApi } from '@/api'

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const lastTestResult = ref<{ success: boolean; message: string; time: string } | null>(null)

const form = reactive({
  enabled: false,
  issuer_url: '',
  client_id: '',
  client_secret: '',
  redirect_url: '',
  scopes: 'openid,profile,email',
  role_claim: 'realm_access.roles',
  role_mapping: '',
  default_role: 'viewer',
  auto_provision: true,
  username_claim: 'preferred_username',
  email_claim: 'email',
})

const defaultRoleOptions = [
  { label: 'admin', value: 'admin' },
  { label: 'team_lead', value: 'team_lead' },
  { label: 'member', value: 'member' },
  { label: 'viewer', value: 'viewer' },
]

async function fetchConfig() {
  loading.value = true
  try {
    const res = await oidcSettingsApi.getConfig()
    if (res.data.data) {
      const d = res.data.data
      form.enabled = d.enabled
      form.issuer_url = d.issuer_url || ''
      form.client_id = d.client_id || ''
      form.client_secret = d.client_secret || ''
      form.redirect_url = d.redirect_url || ''
      form.scopes = d.scopes || 'openid,profile,email'
      form.role_claim = d.role_claim || 'realm_access.roles'
      form.role_mapping = d.role_mapping || ''
      form.default_role = d.default_role || 'viewer'
      form.auto_provision = d.auto_provision
      form.username_claim = (d as any).username_claim || 'preferred_username'
      form.email_claim = (d as any).email_claim || 'email'
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
    await oidcSettingsApi.updateConfig({ ...form })
    message.success(t('common.savedSuccess'))
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  if (!form.issuer_url) {
    message.warning('Issuer URL is required')
    return
  }
  testing.value = true
  try {
    const fn = (oidcSettingsApi as any).testConnection || (oidcSettingsApi as any).discover
    if (typeof fn === 'function') {
      const res = await fn.call(oidcSettingsApi, form.issuer_url)
      const ok = !!res?.data?.data?.success
      lastTestResult.value = {
        success: ok,
        message: res?.data?.data?.message || (ok ? 'Discovery document fetched' : 'Discovery failed'),
        time: new Date().toLocaleTimeString(),
      }
    } else {
      const url = form.issuer_url.replace(/\/$/, '') + '/.well-known/openid-configuration'
      const r = await fetch(url, { method: 'GET' })
      lastTestResult.value = {
        success: r.ok,
        message: r.ok ? `Discovery OK (${r.status})` : `Discovery failed (${r.status})`,
        time: new Date().toLocaleTimeString(),
      }
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
          <h2 class="config-title">SSO / OIDC</h2>
          <p class="config-subtitle">Single sign-on via Keycloak or any OIDC-compliant provider. Changes apply on next login.</p>
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

      <div class="config-status" data-tone="warning" style="border-color: rgba(245,158,11,0.3); background: rgba(245,158,11,0.06);">
        <span class="sre-dot" data-severity="warning"></span>
        <span>{{ t('settings.oidcRestartWarning') }}</span>
      </div>

      <div class="config-sections sre-stagger">
        <section class="config-section">
          <h3 class="section-title">Provider</h3>
          <p class="section-desc">Issuer discovery URL and OAuth2 client credentials registered with the IdP.</p>
          <div class="form-grid">
            <NFormItem :label="t('settings.oidcEnabled')" class="full-row">
              <NSwitch v-model:value="form.enabled" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcIssuerUrl')" class="full-row">
              <NInput v-model:value="form.issuer_url" :placeholder="t('settings.oidcIssuerUrlPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcClientId')">
              <NInput v-model:value="form.client_id" :placeholder="t('settings.oidcClientIdPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcClientSecret')">
              <NInput v-model:value="form.client_secret" type="password" show-password-on="click" :placeholder="t('settings.oidcClientSecretPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcRedirectUrl')" class="full-row">
              <NInput v-model:value="form.redirect_url" :placeholder="t('settings.oidcRedirectUrlPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcScopes')" class="full-row">
              <NInput v-model:value="form.scopes" :placeholder="t('settings.oidcScopesPlaceholder')" />
            </NFormItem>
          </div>
        </section>

        <section class="config-section">
          <h3 class="section-title">Claim Mapping</h3>
          <p class="section-desc">Map ID-token claims to user fields and translate provider roles into SREAgent roles.</p>
          <div class="form-grid">
            <NFormItem label="Username Claim">
              <NInput v-model:value="form.username_claim" placeholder="preferred_username" />
            </NFormItem>
            <NFormItem label="Email Claim">
              <NInput v-model:value="form.email_claim" placeholder="email" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcRoleClaim')" class="full-row">
              <NInput v-model:value="form.role_claim" :placeholder="t('settings.oidcRoleClaimPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcRoleMapping')" class="full-row">
              <NInput v-model:value="form.role_mapping" type="textarea" :rows="3" :placeholder="t('settings.oidcRoleMappingPlaceholder')" />
            </NFormItem>
          </div>
        </section>

        <section class="config-section">
          <h3 class="section-title">Behavior</h3>
          <p class="section-desc">Control how unknown users are handled when they authenticate for the first time.</p>
          <div class="form-grid">
            <NFormItem :label="t('settings.oidcAutoProvision')">
              <NSwitch v-model:value="form.auto_provision" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcDefaultRole')">
              <NSelect v-model:value="form.default_role" :options="defaultRoleOptions" />
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
