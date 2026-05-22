<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { NButton, NIcon, NSwitch, NSelect, NInput, NFormItem, NSpin, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { PulseOutline, SaveOutline } from '@vicons/ionicons5'
import { oidcSettingsApi } from '@/api'
import { getErrorMessage } from '@/utils/format'

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

const defaultRoleOptions = computed(() => [
  { label: t('settings.admin'), value: 'admin' },
  { label: t('settings.teamLead'), value: 'team_lead' },
  { label: t('settings.member'), value: 'member' },
  { label: t('settings.viewerName'), value: 'viewer' },
])

// Inline validation
const urlPattern = /^https:\/\/.+/i
const issuerError = computed(() => {
  if (!form.issuer_url) return ''
  return urlPattern.test(form.issuer_url) ? '' : t('settings.oidcInvalidUrl')
})
const roleMappingError = computed(() => {
  if (!form.role_mapping || !form.role_mapping.trim()) return ''
  try {
    const parsed = JSON.parse(form.role_mapping)
    return (typeof parsed === 'object' && parsed !== null && !Array.isArray(parsed)) ? '' : t('settings.oidcInvalidJson')
  } catch { return t('settings.oidcInvalidJson') }
})
const canSave = computed(() => !issuerError.value && !roleMappingError.value)

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
      form.username_claim = (d as Record<string, unknown>).username_claim as string || 'preferred_username'
      form.email_claim = (d as Record<string, unknown>).email_claim as string || 'email'
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
    await oidcSettingsApi.updateConfig({ ...form })
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  if (!form.issuer_url) {
    message.warning(t('settings.oidcIssuerUrlRequired'))
    return
  }
  testing.value = true
  try {
    const fn = (oidcSettingsApi as Record<string, unknown>).testConnection as ((url: string) => Promise<unknown>) || (oidcSettingsApi as Record<string, unknown>).discover as ((url: string) => Promise<unknown>)
    if (typeof fn === 'function') {
      const res = await fn.call(oidcSettingsApi, form.issuer_url) as { data?: { data?: { success?: boolean; message?: string } } }
      const ok = !!res?.data?.data?.success
      lastTestResult.value = {
        success: ok,
        message: res?.data?.data?.message || (ok ? t('settings.oidcDiscoveryFetched') : t('settings.oidcDiscoveryFailed')),
        time: new Date().toLocaleTimeString(),
      }
    } else {
      const url = form.issuer_url.replace(/\/$/, '') + '/.well-known/openid-configuration'
      const r = await fetch(url, { method: 'GET' })
      lastTestResult.value = {
        success: r.ok,
        message: r.ok ? `${t('settings.oidcDiscoveryOk')} (${r.status})` : `${t('settings.oidcDiscoveryFailed')} (${r.status})`,
        time: new Date().toLocaleTimeString(),
      }
    }
    lastTestResult.value!.success ? message.success(lastTestResult.value!.message) : message.error(lastTestResult.value!.message)
  } catch (err: unknown) {
    lastTestResult.value = { success: false, message: getErrorMessage(err), time: new Date().toLocaleTimeString() }
    message.error(getErrorMessage(err))
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
          <h2 class="sre-config-header-title">{{ t('settings.oidcConfig') }}</h2>
          <p class="sre-config-header-sub">{{ t('settings.oidcSubtitle') }}</p>
        </div>
        <div class="sre-config-header-actions">
          <NButton size="small" quaternary :loading="testing" @click="testConnection">
            <template #icon><NIcon :component="PulseOutline" /></template>
            {{ t('common.test') }}
          </NButton>
          <NButton type="primary" size="small" :loading="saving" :disabled="!canSave" @click="save">
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
          <h3 class="sre-config-section-title">{{ t('settings.oidcProviderSection') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.oidcProviderDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.oidcEnabled')" class="full-row">
              <NSwitch v-model:value="form.enabled" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcIssuerUrl')" class="full-row" :validation-status="issuerError ? 'error' : undefined" :feedback="issuerError || t('settings.oidcIssuerUrlHelp')">
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

        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.oidcClaimMapping') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.oidcClaimMappingDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.oidcUsernameClaim')">
              <NInput v-model:value="form.username_claim" :placeholder="t('settings.oidcUsernameClaimPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcEmailClaim')">
              <NInput v-model:value="form.email_claim" :placeholder="t('settings.oidcEmailClaimPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcRoleClaim')" class="full-row" :feedback="t('settings.oidcRoleClaimHelp')">
              <NInput v-model:value="form.role_claim" :placeholder="t('settings.oidcRoleClaimPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcRoleMapping')" class="full-row" :validation-status="roleMappingError ? 'error' : undefined" :feedback="roleMappingError || (t('settings.oidcRoleMappingHelp') + ' ' + t('settings.oidcRoleMappingExample'))">
              <NInput v-model:value="form.role_mapping" type="textarea" :rows="3" :placeholder="t('settings.oidcRoleMappingPlaceholder')" />
            </NFormItem>
          </div>
        </section>

        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.oidcBehavior') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.oidcBehaviorDesc') }}</p>
          <div class="sre-config-form-grid">
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
</style>
