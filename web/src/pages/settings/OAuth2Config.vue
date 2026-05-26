<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { NButton, NIcon, NSwitch, NSelect, NInput, NFormItem, NSpin } from 'naive-ui'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { SaveOutline } from '@vicons/ionicons5'
import { oauth2SettingsApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import { useConfigForm } from '@/composables'

const message = useMessage()
const { t } = useI18n()

const { form, loading, load, markSaved } = useConfigForm({
  load: () => oauth2SettingsApi.getConfig().then(r => r.data.data),
  save: (f) => oauth2SettingsApi.updateConfig({ ...f }),
})

// Per-section saving state
const savingProvider = ref(false)
const savingMapping = ref(false)
const savingBehavior = ref(false)

const defaultRoleOptions = computed(() => [
  { label: t('settings.admin'), value: 'admin' },
  { label: t('settings.teamLead'), value: 'team_lead' },
  { label: t('settings.member'), value: 'member' },
  { label: t('settings.viewerName'), value: 'viewer' },
])

async function saveProvider() {
  if (form.enabled) {
    if (!form.client_id?.trim()) { message.warning(t('settings.oauth2ClientId')); return }
    if (!form.auth_url?.trim()) { message.warning(t('settings.oauth2AuthUrl')); return }
    if (!form.token_url?.trim()) { message.warning(t('settings.oauth2TokenUrl')); return }
  }
  savingProvider.value = true
  try {
    await oauth2SettingsApi.updateConfig({ ...form })
    markSaved()
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { savingProvider.value = false }
}

async function saveMapping() {
  savingMapping.value = true
  try {
    await oauth2SettingsApi.updateConfig({ ...form })
    markSaved()
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { savingMapping.value = false }
}

async function saveBehavior() {
  savingBehavior.value = true
  try {
    await oauth2SettingsApi.updateConfig({ ...form })
    markSaved()
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
          <h2 class="sre-config-header-title">{{ t('settings.oauth2Title') }}</h2>
          <p class="sre-config-header-sub">{{ t('settings.oauth2Subtitle') }}</p>
        </div>
      </header>

      <div class="config-sections sre-stagger">
        <!-- Provider Configuration -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.oauth2ProviderSection') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.oauth2ProviderDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.oauth2Enabled')" class="full-row">
              <NSwitch v-model:value="form.enabled" />
            </NFormItem>
            <NFormItem :label="t('settings.oauth2Name')">
              <NInput v-model:value="form.name" :placeholder="t('settings.oauth2NamePlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oauth2ClientId')" :required="form.enabled">
              <NInput v-model:value="form.client_id" :placeholder="t('settings.oauth2ClientIdPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oauth2ClientSecret')" :required="form.enabled">
              <NInput v-model:value="form.client_secret" type="password" show-password-on="click" :placeholder="t('settings.oauth2ClientSecretPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oauth2AuthUrl')" class="full-row" :required="form.enabled">
              <NInput v-model:value="form.auth_url" :placeholder="t('settings.oauth2AuthUrlPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oauth2TokenUrl')" class="full-row" :required="form.enabled">
              <NInput v-model:value="form.token_url" :placeholder="t('settings.oauth2TokenUrlPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oauth2UserInfoUrl')" class="full-row">
              <NInput v-model:value="form.user_info_url" :placeholder="t('settings.oauth2UserInfoUrlPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oauth2RedirectUrl')" class="full-row">
              <NInput v-model:value="form.redirect_url" :placeholder="t('settings.oauth2RedirectUrlPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.oauth2Scopes')" class="full-row">
              <NInput v-model:value="form.scopes" :placeholder="t('settings.oauth2ScopesPlaceholder')" />
            </NFormItem>
          </div>
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingProvider" @click="saveProvider">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
          </div>
        </section>

        <!-- Field Mapping -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.oauth2FieldMapping') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.oauth2FieldMappingDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.oauth2UserIdField')">
              <NInput v-model:value="form.user_id_field" placeholder="sub" />
            </NFormItem>
            <NFormItem :label="t('settings.oauth2EmailField')">
              <NInput v-model:value="form.email_field" placeholder="email" />
            </NFormItem>
            <NFormItem :label="t('settings.oauth2UsernameField')">
              <NInput v-model:value="form.username_field" placeholder="preferred_username" />
            </NFormItem>
          </div>
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingMapping" @click="saveMapping">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
          </div>
        </section>

        <!-- Behavior -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.oauth2Behavior') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.oauth2BehaviorDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.oauth2DefaultRole')">
              <NSelect v-model:value="form.default_role" :options="defaultRoleOptions" />
            </NFormItem>
            <NFormItem :label="t('settings.oauth2AutoProvision')">
              <NSwitch v-model:value="form.auto_provision" />
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
.section-footer {
  display: flex; justify-content: flex-end;
  padding-top: 16px; margin-top: 16px;
  border-top: var(--sre-hairline);
}
</style>
