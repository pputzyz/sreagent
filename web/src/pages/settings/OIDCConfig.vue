<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { NButton, NIcon, NSwitch, NSelect, NInput, NFormItem, NSpin, NSpace } from 'naive-ui'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { PulseOutline, SaveOutline } from '@vicons/ionicons5'
import { oidcSettingsApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import { useConfigForm } from '@/composables'

const message = useMessage()
const { t } = useI18n()

const { form, loading, testing, load, saveAndTest } = useConfigForm({
  load: () => oidcSettingsApi.getConfig().then(r => r.data.data),
  save: (f) => oidcSettingsApi.updateConfig({ ...f }),
  test: async () => {
    if (!form.issuer_url) {
      message.warning(t('settings.oidcIssuerUrlRequired'))
      return
    }
    const fn = (oidcSettingsApi as Record<string, unknown>).testConnection as ((url: string) => Promise<unknown>) || (oidcSettingsApi as Record<string, unknown>).discover as ((url: string) => Promise<unknown>)
    if (typeof fn === 'function') {
      const res = await fn.call(oidcSettingsApi, form.issuer_url) as { data?: { data?: { success?: boolean; message?: string } } }
      const ok = !!res?.data?.data?.success
      const msg = res?.data?.data?.message || (ok ? t('settings.oidcDiscoveryFetched') : t('settings.oidcDiscoveryFailed'))
      if (ok) message.success(msg)
      else message.error(msg)
    } else {
      const url = form.issuer_url.replace(/\/$/, '') + '/.well-known/openid-configuration'
      const r = await fetch(url, { method: 'GET' })
      const msg = r.ok ? `${t('settings.oidcDiscoveryOk')} (${r.status})` : `${t('settings.oidcDiscoveryFailed')} (${r.status})`
      if (r.ok) message.success(msg)
      else message.error(msg)
    }
  },
})

// Per-section saving state
const savingProvider = ref(false)
const savingClaims = ref(false)
const savingBehavior = ref(false)

const defaultRoleOptions = computed(() => [
  { label: t('settings.admin'), value: 'admin' },
  { label: t('settings.teamLead'), value: 'team_lead' },
  { label: t('settings.member'), value: 'member' },
  { label: t('settings.viewerName'), value: 'viewer' },
])

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

async function saveProvider() {
  if (form.enabled) {
    if (!form.issuer_url?.trim()) { message.warning(t('settings.oidcIssuerUrlRequired')); return }
    if (issuerError.value) { message.warning(issuerError.value); return }
    if (!form.client_id?.trim()) { message.warning(t('settings.oidcClientIdRequired')); return }
    if (!form.client_secret?.trim()) { message.warning(t('settings.oidcClientSecretRequired')); return }
  }
  savingProvider.value = true
  try {
    await oidcSettingsApi.updateConfig({ ...form })
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { savingProvider.value = false }
}

async function saveClaims() {
  if (roleMappingError.value) { message.warning(roleMappingError.value); return }
  savingClaims.value = true
  try {
    await oidcSettingsApi.updateConfig({ ...form })
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { savingClaims.value = false }
}

async function saveBehavior() {
  savingBehavior.value = true
  try {
    await oidcSettingsApi.updateConfig({ ...form })
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { savingBehavior.value = false }
}

onMounted(() => load())
</script>

<template>
  <NSpin :show="loading">
    <div class="sre-config-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">{{ t('settings.oidcConfig') }}</h2>
          <p class="sre-config-header-sub">{{ t('settings.oidcSubtitle') }}</p>
        </div>
      </header>

      <div class="config-sections sre-stagger">
        <section class="sre-config-section">
          <div class="section-header-row">
            <div>
              <h3 class="sre-config-section-title" style="margin: 0">{{ t('settings.oidcProviderSection') }}</h3>
              <p class="sre-config-section-desc" style="margin-top: 4px">{{ t('settings.oidcProviderDesc') }}</p>
            </div>
            <NButton size="small" quaternary :loading="testing" @click="saveAndTest">
              <template #icon><NIcon :component="PulseOutline" /></template>
              {{ t('common.test') }}
            </NButton>
          </div>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.oidcEnabled')" class="full-row">
              <NSwitch v-model:value="form.enabled" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcIssuerUrl')" class="full-row" :required="form.enabled" :validation-status="issuerError ? 'error' : undefined" :feedback="issuerError || t('settings.oidcIssuerUrlHelp')">
              <NInput v-model:value="form.issuer_url" :placeholder="t('settings.oidcIssuerUrlPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcClientId')" :required="form.enabled">
              <NInput v-model:value="form.client_id" :placeholder="t('settings.oidcClientIdPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcClientSecret')" :required="form.enabled">
              <NInput v-model:value="form.client_secret" type="password" show-password-on="click" :placeholder="t('settings.oidcClientSecretPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcRedirectUrl')" class="full-row">
              <NInput v-model:value="form.redirect_url" :placeholder="t('settings.oidcRedirectUrlPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oidcScopes')" class="full-row">
              <NInput v-model:value="form.scopes" :placeholder="t('settings.oidcScopesPlaceholder')" />
            </NFormItem>
          </div>
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingProvider" @click="saveProvider">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
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
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingClaims" @click="saveClaims">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
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
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingBehavior" @click="saveBehavior">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
          </div>
        </section>
      </div>
    </div>
  </NSpin>
</template>

<style scoped>
.section-header-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}
.section-footer {
  display: flex; justify-content: flex-end;
  padding-top: 16px; margin-top: 16px;
  border-top: var(--sre-hairline);
}
</style>
