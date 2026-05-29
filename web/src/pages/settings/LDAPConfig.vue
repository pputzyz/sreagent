<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { NButton, NIcon, NSwitch, NSelect, NInput, NInputNumber, NFormItem, NSpin, NSpace } from 'naive-ui'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { PulseOutline, SaveOutline } from '@vicons/ionicons5'
import { ldapSettingsApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import { useConfigForm } from '@/composables'

const message = useMessage()
const { t } = useI18n()

const { form, loading, load, markSaved } = useConfigForm({
  load: () => ldapSettingsApi.getConfig().then(r => r.data.data),
  save: (f) => ldapSettingsApi.updateConfig({ ...f }),
})

// Per-section saving state
const savingProvider = ref(false)
const savingBehavior = ref(false)
const testing = ref(false)

const defaultRoleOptions = computed(() => [
  { label: t('settings.admin'), value: 'admin' },
  { label: t('settings.teamLead'), value: 'team_lead' },
  { label: t('settings.member'), value: 'member' },
])

async function saveProvider() {
  if (form.enabled) {
    if (!form.host?.trim()) { message.warning(t('settings.ldapHost')); return }
    if (!form.base_dn?.trim()) { message.warning(t('settings.ldapBaseDn')); return }
  }
  savingProvider.value = true
  try {
    await ldapSettingsApi.updateConfig({ ...form })
    markSaved()
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { savingProvider.value = false }
}

async function saveBehavior() {
  savingBehavior.value = true
  try {
    await ldapSettingsApi.updateConfig({ ...form })
    markSaved()
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { savingBehavior.value = false }
}

async function testConnection() {
  testing.value = true
  try {
    const res = await ldapSettingsApi.testConnection()
    const ok = res.data.data.success
    const msg = res.data.data.message
    if (ok) message.success(msg || t('settings.ldapTestSuccess'))
    else message.error(msg || t('settings.ldapTestFailed'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { testing.value = false }
}

onMounted(() => load())
</script>

<template>
  <NSpin :show="loading">
    <div class="sre-config-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">{{ t('settings.ldapTitle') }}</h2>
          <p class="sre-config-header-sub">{{ t('settings.ldapSubtitle') }}</p>
        </div>
      </header>

      <div class="config-sections sre-stagger">
        <!-- Server Configuration -->
        <section class="sre-config-section">
          <div class="section-header-row">
            <div>
              <h3 class="sre-config-section-title" style="margin: 0">{{ t('settings.ldapProviderSection') }}</h3>
              <p class="sre-config-section-desc" style="margin-top: 4px">{{ t('settings.ldapProviderDesc') }}</p>
            </div>
            <NButton size="small" quaternary :loading="testing" @click="testConnection">
              <template #icon><NIcon :component="PulseOutline" /></template>
              {{ t('settings.ldapTestConnection') }}
            </NButton>
          </div>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.ldapEnabled')" class="full-row">
              <NSwitch v-model:value="form.enabled" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapHost')" :required="form.enabled">
              <NInput v-model:value="form.host" :placeholder="t('settings.ldapHostPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapPort')">
              <NInputNumber v-model:value="form.port" :min="1" :max="65535" style="width: 100%" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapBaseDn')" class="full-row" :required="form.enabled">
              <NInput v-model:value="form.base_dn" :placeholder="t('settings.ldapBaseDnPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapBindDn')" class="full-row">
              <NInput v-model:value="form.bind_dn" :placeholder="t('settings.ldapBindDnPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapBindPassword')" class="full-row">
              <NInput v-model:value="form.bind_password" type="password" show-password-on="click" :placeholder="t('settings.ldapBindPasswordPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapUserFilter')" class="full-row">
              <NInput v-model:value="form.user_filter" :placeholder="t('settings.ldapUserFilterPlaceholder')" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapUserAttr')">
              <NInput v-model:value="form.user_attr" placeholder="uid" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapEmailAttr')">
              <NInput v-model:value="form.email_attr" placeholder="mail" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapDisplayNameAttr')">
              <NInput v-model:value="form.display_name_attr" placeholder="displayName" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapStartTLS')">
              <NSwitch v-model:value="form.start_tls" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapSkipVerify')">
              <NSwitch v-model:value="form.skip_verify" />
            </NFormItem>
          </div>
          <div class="section-footer">
            <NButton type="primary" size="small" :loading="savingProvider" @click="saveProvider">
              <template #icon><NIcon :component="SaveOutline" /></template>
              {{ t('common.save') }}
            </NButton>
          </div>
        </section>

        <!-- Behavior -->
        <section class="sre-config-section">
          <h3 class="sre-config-section-title">{{ t('settings.ldapBehavior') }}</h3>
          <p class="sre-config-section-desc">{{ t('settings.ldapBehaviorDesc') }}</p>
          <div class="sre-config-form-grid">
            <NFormItem :label="t('settings.ldapDefaultRole')">
              <NSelect v-model:value="form.default_role" :options="defaultRoleOptions" />
            </NFormItem>
            <NFormItem :label="t('settings.ldapAutoProvision')">
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
