<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { NButton, NIcon, NSpin, NForm, NFormItem, NSelect } from 'naive-ui'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { SaveOutline } from '@vicons/ionicons5'
import { securitySettingsApi } from '@/api'
import { getErrorMessage } from '@/utils/format'

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const form = { jwt_expire_seconds: 3600 }

const expireOptions = computed(() => [
  { label: t('settings.jwtExpire1h'), value: 3600 },
  { label: t('settings.jwtExpire4h'), value: 14400 },
  { label: t('settings.jwtExpire8h'), value: 28800 },
  { label: t('settings.jwtExpire24h'), value: 86400 },
  { label: t('settings.jwtExpire7d'), value: 604800 },
])

async function load() {
  loading.value = true
  try {
    const res = await securitySettingsApi.getConfig()
    const data = res.data.data
    if (data) form.jwt_expire_seconds = data.jwt_expire_seconds
  } catch { /* use default */ } finally { loading.value = false }
}

async function handleSave() {
  saving.value = true
  try {
    await securitySettingsApi.updateConfig({ jwt_expire_seconds: form.jwt_expire_seconds })
    message.success(t('common.savedSuccess'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { saving.value = false }
}

onMounted(() => load())
</script>

<template>
  <NSpin :show="loading">
    <div class="sre-config-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">{{ t('settings.securityConfig') }}</h2>
          <p class="sre-config-header-sub">{{ t('settings.jwtExpireHint') }}</p>
        </div>
      </header>

      <section class="sre-config-section">
        <h3 class="sre-config-section-title">{{ t('settings.jwtExpireTime') }}</h3>
        <div class="security-form-row">
          <NForm label-placement="top" class="security-form">
            <NFormItem :label="t('settings.jwtExpireTime')">
              <NSelect
                v-model:value="form.jwt_expire_seconds"
                :options="expireOptions"
                class="security-select"
              />
            </NFormItem>
          </NForm>
        </div>
        <div class="section-footer">
          <NButton type="primary" size="small" :loading="saving" @click="handleSave">
            <template #icon><NIcon :component="SaveOutline" /></template>
            {{ t('common.save') }}
          </NButton>
        </div>
      </section>
    </div>
  </NSpin>
</template>

<style scoped>
.security-form {
  max-width: 480px;
}
.security-select {
  width: 100%;
}
.security-form-row {
  margin-top: 8px;
}
.section-footer {
  display: flex; justify-content: flex-end;
  padding-top: 16px; margin-top: 16px;
  border-top: var(--sre-hairline);
}
</style>
