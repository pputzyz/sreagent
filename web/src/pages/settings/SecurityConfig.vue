<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { NSpin, NForm, NFormItem, NSelect } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { securitySettingsApi } from '@/api'
import { useConfigForm } from '@/composables'

const { t } = useI18n()

const { form, loading, save } = useConfigForm({
  load: () => securitySettingsApi.getConfig().then(r => r.data.data),
  save: (f) => securitySettingsApi.updateConfig({ jwt_expire_seconds: f.jwt_expire_seconds }),
  autoSaveKeys: ['jwt_expire_seconds'],
})

const expireOptions = computed(() => [
  { label: t('settings.jwtExpire1h'), value: 3600 },
  { label: t('settings.jwtExpire4h'), value: 14400 },
  { label: t('settings.jwtExpire8h'), value: 28800 },
  { label: t('settings.jwtExpire24h'), value: 86400 },
  { label: t('settings.jwtExpire7d'), value: 604800 },
])

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
