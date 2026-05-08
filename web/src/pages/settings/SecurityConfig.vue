<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NButton, NSpin, NForm, NFormItem, NSelect, NText } from 'naive-ui'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { securitySettingsApi } from '@/api'

const message = useMessage()
const { t } = useI18n()
const loading = ref(false)
const saving = ref(false)

const jwtExpireSeconds = ref(86400)

const expireOptions = computed(() => [
  { label: t('settings.jwtExpire1h'), value: 3600 },
  { label: t('settings.jwtExpire4h'), value: 14400 },
  { label: t('settings.jwtExpire8h'), value: 28800 },
  { label: t('settings.jwtExpire24h'), value: 86400 },
  { label: t('settings.jwtExpire7d'), value: 604800 },
])

async function fetchConfig() {
  loading.value = true
  try {
    const { data } = await securitySettingsApi.getConfig()
    jwtExpireSeconds.value = data.data.jwt_expire_seconds
  } catch (err: any) {
    message.error(err.message)
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    await securitySettingsApi.updateConfig({ jwt_expire_seconds: jwtExpireSeconds.value })
    message.success(t('common.savedSuccess'))
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

onMounted(fetchConfig)
</script>

<template>
  <NSpin :show="loading">
    <div class="sre-config-page">
      <header class="sre-config-header">
        <div>
          <h2 class="sre-config-header-title">{{ t('settings.securityConfig') || 'Security' }}</h2>
          <p class="sre-config-header-sub">{{ t('settings.jwtExpireHint') }}</p>
        </div>
        <div class="sre-config-header-actions">
          <NButton type="primary" size="small" :loading="saving" @click="handleSave">
            {{ t('common.save') }}
          </NButton>
        </div>
      </header>

      <section class="sre-config-section">
        <h3 class="sre-config-section-title">{{ t('settings.jwtExpireTime') }}</h3>
        <p class="sre-config-section-desc">{{ t('settings.jwtExpireHint') }}</p>
        <div class="security-form-row">
          <NForm label-placement="top" class="security-form">
            <NFormItem :label="t('settings.jwtExpireTime')">
              <NSelect
                v-model:value="jwtExpireSeconds"
                :options="expireOptions"
                class="security-select"
              />
            </NFormItem>
          </NForm>
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
</style>
